/*
Copyright 2022 TriggerMesh Inc.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package azureeventgridsource

import (
	"context"
	"errors"
	"fmt"
	"hash/crc32"
	"net/http"
	"strconv"
	"strings"

	corev1 "k8s.io/api/core/v1"

	"knative.dev/pkg/controller"
	"knative.dev/pkg/reconciler"

	azureeventgrid "github.com/Azure/azure-sdk-for-go/profiles/latest/eventgrid/mgmt/eventgrid"
	"github.com/Azure/azure-sdk-for-go/profiles/latest/resources/mgmt/resources"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/to"
	"github.com/Azure/go-autorest/autorest"
	"github.com/Azure/go-autorest/autorest/azure"

	commonv1alpha1 "github.com/triggermesh/triggermesh/pkg/apis/common/v1alpha1"
	"github.com/triggermesh/triggermesh/pkg/apis/sources"
	"github.com/triggermesh/triggermesh/pkg/apis/sources/v1alpha1"
	"github.com/triggermesh/triggermesh/pkg/reconciler/event"
	"github.com/triggermesh/triggermesh/pkg/reconciler/skip"
	"github.com/triggermesh/triggermesh/pkg/sources/client/azure/eventgrid"
)

const (
	eventgridTagOwnerResource  = "io.triggermesh_owner-resource"
	eventgridTagOwnerNamespace = "io.triggermesh_owner-namespace"
	eventgridTagOwnerName      = "io.triggermesh_owner-name"
)

// Resource group where system topics for events pertaining to Azure
// subscriptions are created.
// We use the same name and default region as Azure does:
// https://docs.microsoft.com/en-us/azure/event-grid/system-topics
const (
	defaultResourceGroupName   = "DEFAULT-EVENTGRID"
	defaultResourceGroupRegion = "westus2"
)

// ensureSystemTopic ensures a system topic exists with the expected configuration.
// Required permissions:
//  - Microsoft.EventGrid/systemTopics/read
//  - Microsoft.EventGrid/systemTopics/write
//  To manage the default resource group, when the scope is an Azure subscription:
//   - Microsoft.Resources/subscriptions/resourceGroups/read
//   - Microsoft.Resources/subscriptions/resourceGroups/write
//  Additionally, for regional resources, "read" permission on the resource (scope).
func ensureSystemTopic(ctx context.Context, cli eventgrid.SystemTopicsClient,
	providersCli eventgrid.ProvidersClient,
	resGroupsCli eventgrid.ResourceGroupsClient) (*v1alpha1.AzureResourceID /*sysTopicResID*/, error) {

	if skip.Skip(ctx) {
		return nil, nil
	}

	src := commonv1alpha1.ReconcilableFromContext(ctx)
	typedSrc := src.(*v1alpha1.AzureEventGridSource)

	status := &typedSrc.Status

	scope := typedSrc.Spec.Scope.String()

	sysTopic, err := findSystemTopic(ctx, cli, typedSrc)
	switch {
	case isDenied(err):
		status.MarkNotSubscribed(v1alpha1.AzureReasonAPIError, "Access denied to system topic API: "+toErrMsg(err))
		return nil, controller.NewPermanentError(failFindSystemTopicEvent(scope, err))
	case err != nil:
		status.MarkNotSubscribed(v1alpha1.AzureReasonAPIError, "Cannot look up system topic: "+toErrMsg(err))
		// wrap any other error to fail the reconciliation
		return nil, fmt.Errorf("%w", failFindSystemTopicEvent(scope, err))
	}

	sysTopicExists := sysTopic != nil

	if sysTopicExists {
		sysTopicResID, err := parseResourceID(*sysTopic.ID)
		if err != nil {
			return nil, fmt.Errorf("converting resource ID string to structured resource ID: %w", err)
		}

		// re-own system topic if it is not tagged
		if sysTopic.Tags[eventgridTagOwnerResource] == nil ||
			sysTopic.Tags[eventgridTagOwnerNamespace] == nil ||
			sysTopic.Tags[eventgridTagOwnerName] == nil {

			if sysTopic.Tags == nil {
				sysTopic.Tags = make(map[string]*string, 3)
			}

			for t, v := range systemTopicResourceTags(typedSrc) {
				v := v
				sysTopic.Tags[t] = v
			}

			rgName := sysTopicResID.ResourceGroup
			sysTopicName := *sysTopic.Name

			restCtx, cancel := context.WithTimeout(ctx, crudTimeout)
			defer cancel()

			resultFuture, err := cli.CreateOrUpdate(restCtx, rgName, sysTopicName, *sysTopic)
			switch {
			case isDenied(err):
				status.MarkNotSubscribed(v1alpha1.AzureReasonAPIError, "Access denied to system topic API: "+toErrMsg(err))
				return nil, controller.NewPermanentError(failSystemTopicEvent(scope, sysTopicExists, err))
			case err != nil:
				status.MarkNotSubscribed(v1alpha1.AzureReasonAPIError, "Cannot update tags of system topic: "+toErrMsg(err))
				return nil, fmt.Errorf("%w", failSystemTopicEvent(scope, sysTopicExists, err))
			}

			if err := resultFuture.WaitForCompletionRef(ctx, cli.BaseClient()); err != nil {
				return nil, fmt.Errorf("waiting for update of system topic %q: %w", sysTopicResID, err)
			}

			event.Normal(ctx, ReasonSystemTopicSynced, "Re-owned orphan system topic %q", sysTopicResID)
		}

		return sysTopicResID, nil
	}

	desiredSysTopic, err := newSystemTopic(ctx, cli.BaseClient(), providersCli, typedSrc)
	switch {
	case isDenied(err):
		status.MarkNotSubscribed(v1alpha1.AzureReasonAPIError, "Access denied to Azure API: "+toErrMsg(err))
		return nil, controller.NewPermanentError(failSystemTopicEvent(scope, sysTopicExists, err))
	case err != nil:
		status.MarkNotSubscribed(v1alpha1.AzureReasonAPIError, "Cannot generate system topic state: "+toErrMsg(err))
		return nil, fmt.Errorf("%w", failSystemTopicEvent(scope, sysTopicExists, err))
	}

	rgName := typedSrc.Spec.Scope.ResourceGroup

	if isSubscriptionScope := typedSrc.Spec.Scope.ResourceProvider == "" && rgName == ""; isSubscriptionScope {
		rgName = defaultResourceGroupName

		err := ensureDefaultResourceGroup(ctx, resGroupsCli)
		switch {
		case isDenied(err):
			status.MarkNotSubscribed(v1alpha1.AzureReasonAPIError, "Access denied to resource groups API: "+toErrMsg(err))
			return nil, controller.NewPermanentError(failResourceGroupEvent(defaultResourceGroupName, err))
		case err != nil:
			status.MarkNotSubscribed(v1alpha1.AzureReasonAPIError, "Cannot ensure existence of default resource group: "+toErrMsg(err))
			return nil, fmt.Errorf("%w", failResourceGroupEvent(defaultResourceGroupName, err))
		}
	}

	sysTopicName := systemTopicResourceName(typedSrc)

	restCtx, cancel := context.WithTimeout(ctx, crudTimeout)
	defer cancel()

	resultFuture, err := cli.CreateOrUpdate(restCtx, rgName, sysTopicName, *desiredSysTopic)
	switch {
	case isDenied(err):
		status.MarkNotSubscribed(v1alpha1.AzureReasonAPIError, "Access denied to system topic API: "+toErrMsg(err))
		return nil, controller.NewPermanentError(failSystemTopicEvent(scope, sysTopicExists, err))
	case err != nil:
		status.MarkNotSubscribed(v1alpha1.AzureReasonAPIError, "Cannot create system topic: "+toErrMsg(err))
		return nil, fmt.Errorf("%w", failSystemTopicEvent(scope, sysTopicExists, err))
	}

	if err := resultFuture.WaitForCompletionRef(ctx, cli.BaseClient()); err != nil {
		return nil, fmt.Errorf("waiting for creation of system topic %q: %w", sysTopicName, err)
	}

	sysTopicResult, err := resultFuture.Result(cli.ConcreteClient())
	if err != nil {
		return nil, fmt.Errorf("reading created system topic %q: %w", sysTopicName, err)
	}

	sysTopicResID, err := parseResourceID(*sysTopicResult.ID)
	if err != nil {
		return nil, fmt.Errorf("converting resource ID string to structured resource ID: %w", err)
	}

	event.Normal(ctx, ReasonSystemTopicSynced, "Created system topic %q for resource %q", sysTopicResID, scope)

	return sysTopicResID, nil
}

// ensureNoSystemTopic ensures the system topic is removed.
// Required permissions:
//  - Microsoft.EventGrid/systemTopics/read
//  - Microsoft.EventGrid/systemTopics/delete
func ensureNoSystemTopic(ctx context.Context, cli eventgrid.SystemTopicsClient,
	eventSubsCli eventgrid.EventSubscriptionsClient, sysTopic *azureeventgrid.SystemTopic) reconciler.Event {

	if skip.Skip(ctx) {
		return nil
	}

	if sysTopic == nil {
		event.Warn(ctx, ReasonSystemTopicFinalized, "System topic not found, skipping finalization")
		return nil
	}

	sysTopicResID, err := parseResourceID(*sysTopic.ID)
	if err != nil {
		return fmt.Errorf("converting resource ID string to structured resource ID: %w", err)
	}

	src := commonv1alpha1.ReconcilableFromContext(ctx)
	typedSrc := src.(*v1alpha1.AzureEventGridSource)

	rgName := sysTopicResID.ResourceGroup
	sysTopicName := sysTopicResID.ResourceName

	restCtx, cancel := context.WithTimeout(ctx, crudTimeout)
	defer cancel()

	eventSubs, err := eventSubsCli.ListBySystemTopic(restCtx, rgName, sysTopicName, "", to.Ptr[int32](1))
	switch {
	case isNotFound(err):
		event.Warn(ctx, ReasonSystemTopicFinalized, "System topic not found, skipping finalization")
		return nil
	case isDenied(err):
		// it is unlikely that we recover from auth errors in the
		// finalizer, so we simply record a warning event and return
		event.Warn(ctx, ReasonFailedSystemTopic, "Access denied to event subscription API. Ignoring: %s", toErrMsg(err))
		return nil
	case err != nil:
		return failFinalizeSystemTopicEvent(*sysTopic.ID, fmt.Errorf("listing event subscriptions of system topic: %w", err))
	}

	if hasEventSubs := len(eventSubs.Values()) > 0; hasEventSubs {
		event.Warn(ctx, ReasonSystemTopicFinalized, "System topic has remaining event subscriptions, skipping deletion")

		if !assertSystemTopicOwnership(sysTopic, typedSrc) {
			return nil
		}

		delete(sysTopic.Tags, eventgridTagOwnerResource)
		delete(sysTopic.Tags, eventgridTagOwnerNamespace)
		delete(sysTopic.Tags, eventgridTagOwnerName)

		restCtx, cancel := context.WithTimeout(ctx, crudTimeout)
		defer cancel()

		resultFuture, err := cli.CreateOrUpdate(restCtx, rgName, sysTopicName, *sysTopic)
		switch {
		case isDenied(err):
			// it is unlikely that we recover from auth errors in the
			// finalizer, so we simply record a warning event and return
			event.Warn(ctx, ReasonFailedSystemTopic, "Access denied to system topic API. Ignoring: %s", toErrMsg(err))
			return nil
		case err != nil:
			return failFinalizeSystemTopicEvent(*sysTopic.ID, fmt.Errorf("updating tags of system topic: %w", err))
		}

		if err := resultFuture.WaitForCompletionRef(ctx, cli.BaseClient()); err != nil {
			return fmt.Errorf("waiting for update of system topic %q: %w", sysTopicResID, err)
		}

		event.Normal(ctx, ReasonSystemTopicFinalized, "Removed ownership tags on system topic %q", sysTopicResID)

		return nil
	}

	restCtx, cancel = context.WithTimeout(ctx, crudTimeout)
	defer cancel()

	resultFuture, err := cli.Delete(restCtx, rgName, sysTopicName)
	switch {
	case isDenied(err):
		// it is unlikely that we recover from auth errors in the
		// finalizer, so we simply record a warning event and return
		event.Warn(ctx, ReasonFailedSystemTopic, "Access denied to system topic API. Ignoring: %s", toErrMsg(err))
		return nil
	case err != nil:
		return failFinalizeSystemTopicEvent(*sysTopic.ID, fmt.Errorf("deleting system topic: %w", err))
	}

	if err := resultFuture.WaitForCompletionRef(ctx, cli.BaseClient()); err != nil {
		return fmt.Errorf("waiting for deletion of system topic %q: %w", sysTopicResID, err)
	}

	event.Normal(ctx, ReasonSystemTopicFinalized, "Deleted system topic %q", sysTopicResID)

	return nil
}

// findSystemTopic returns the system topic that matches the scope of the given
// source, if such system topic exists.
// If no system topic matches the description, nil is returned.
func findSystemTopic(ctx context.Context, cli eventgrid.SystemTopicsClient,
	src *v1alpha1.AzureEventGridSource) (*azureeventgrid.SystemTopic, error) {

	restCtx, cancel := context.WithTimeout(ctx, crudTimeout)
	defer cancel()

	// We list system topics at the scope of the entire subscription
	// instead of the resource group of the user-provided source (scope)
	// because Azure enforces that no more than one system topic may exist
	// in a subscription for a given scope.
	resIter, err := cli.ListBySubscriptionComplete(restCtx, "", nil)
	if err != nil {
		return nil, fmt.Errorf("listing system topics in subscription %q: %w", src.Spec.Scope.SubscriptionID, err)
	}

	scope := src.Spec.Scope.String()

	for st := resIter.Value(); resIter.NotDone(); err, st = resIter.NextWithContext(ctx), resIter.Value() {
		if err != nil {
			return nil, fmt.Errorf("failed to get next system topic: %w", err)
		}

		if strings.EqualFold(*st.Source, scope) {
			return &st, nil
		}
	}

	return nil, nil
}

// systemTopicResourceTags returns a set of Azure resource tags containing
// information from the given source instance to set on an Event Grid system topic.
func systemTopicResourceTags(src *v1alpha1.AzureEventGridSource) map[string]*string {
	return map[string]*string{
		eventgridTagOwnerResource:  to.Ptr(sources.AzureEventGridSourceResource.String()),
		eventgridTagOwnerNamespace: &src.Namespace,
		eventgridTagOwnerName:      &src.Name,
	}
}

// assertSystemTopicOwnership returns whether a system topic is owned by the
// given source.
func assertSystemTopicOwnership(st *azureeventgrid.SystemTopic, src *v1alpha1.AzureEventGridSource) bool {
	for k, v := range systemTopicResourceTags(src) {
		if t, exists := st.Tags[k]; !exists || t == nil || *t != *v {
			return false
		}
	}

	return true
}

// newSystemTopic returns the desired state of the system topic.
func newSystemTopic(ctx context.Context, autorestCli autorest.Client, providersCli eventgrid.ProvidersClient,
	src *v1alpha1.AzureEventGridSource) (*azureeventgrid.SystemTopic, error) {

	provider, resourceType := providerAndResourceType(src.Spec.Scope)

	topicType, err := topicType(provider, resourceType)
	if err != nil {
		return nil, fmt.Errorf("determining topic type for resource type %q: %w", provider+"/"+resourceType, err)
	}

	region := "global"

	if topicType.regionType == regionalResource {
		apiVersion, err := getAPIVersion(ctx, providersCli, provider, resourceType)
		if err != nil {
			return nil, fmt.Errorf("determining API version of resource type %q: %w", provider+"/"+resourceType, err)
		}

		// see azureeventgrid.UserAgent()
		autorestCli.UserAgent = "Azure-SDK-For-Go/" + azureeventgrid.Version()

		region, err = getResourceRegion(ctx, autorestCli, src.Spec.Scope, apiVersion)
		if err != nil {
			return nil, fmt.Errorf("getting region of resource %q: %w", &src.Spec.Scope, err)
		}
	}

	st := &azureeventgrid.SystemTopic{
		SystemTopicProperties: &azureeventgrid.SystemTopicProperties{
			Source:    to.Ptr(src.Spec.Scope.String()),
			TopicType: &topicType.typ,
		},
		Location: &region,
		Tags:     systemTopicResourceTags(src),
	}

	return st, nil
}

// getAPIVersion returns the API version of the provided resource type.
func getAPIVersion(ctx context.Context, providersCli eventgrid.ProvidersClient,
	provider, resourceType string) (string, error) {

	restCtx, cancel := context.WithTimeout(ctx, crudTimeout)
	defer cancel()

	providers, err := providersCli.Get(restCtx, provider, "")
	if err != nil {
		return "", fmt.Errorf("getting provider %q: %w", provider, err)
	}

	for _, rt := range *providers.ResourceTypes {
		if strings.ToLower(*rt.ResourceType) == resourceType {
			if defaultAPIVersion := rt.DefaultAPIVersion; defaultAPIVersion != nil {
				return *defaultAPIVersion, nil
			}
			return (*rt.APIVersions)[0], nil
		}
	}

	return "", controller.NewPermanentError(errors.New("could not determine API version for resource " +
		strconv.Quote(provider+"/"+resourceType)))
}

// getResourceRegion returns the region of the given resource (scope).
func getResourceRegion(ctx context.Context, cli autorest.Client,
	scope v1alpha1.AzureResourceID, apiVersion string) (string, error) {

	restCtx, cancel := context.WithTimeout(ctx, crudTimeout)
	defer cancel()

	req, err := getPreparer(restCtx, scope, apiVersion)
	if err != nil {
		return "", fmt.Errorf("preparing Get request for reading scope resource: %w", err)
	}

	resp, err := cli.Send(req, azure.DoRetryWithRegistration(cli))
	if err != nil {
		return "", fmt.Errorf("sending request for reading scope resource: %w", err)
	}

	result := struct {
		Location string `json:"location"`
	}{}

	err = autorest.Respond(
		resp,
		azure.WithErrorUnlessStatusCode(http.StatusOK),
		autorest.ByUnmarshallingJSON(&result),
		autorest.ByClosing(),
	)
	if err != nil {
		return "", fmt.Errorf("responding to request for reading scope resource: %w", err)
	}

	return result.Location, nil
}

// getPreparer prepares the Get request to read the resource described by the
// given scope.
func getPreparer(ctx context.Context, scope v1alpha1.AzureResourceID, apiVersion string) (*http.Request, error) {
	queryParameters := map[string]interface{}{
		"api-version": apiVersion,
	}

	preparer := autorest.CreatePreparer(
		autorest.AsGet(),
		autorest.WithBaseURL(azureeventgrid.DefaultBaseURI),
		autorest.WithPath(scope.String()),
		autorest.WithQueryParameters(queryParameters),
	)

	return preparer.Prepare((&http.Request{}).WithContext(ctx))
}

// ensureDefaultResourceGroup ensures that the default resource group exists.
func ensureDefaultResourceGroup(ctx context.Context, cli eventgrid.ResourceGroupsClient) error {
	restCtx, cancel := context.WithTimeout(ctx, crudTimeout)
	defer cancel()

	// CheckExistence returns a response for HTTP codes in [200, 204, 404], an error otherwise.
	resp, err := cli.CheckExistence(restCtx, defaultResourceGroupName)
	if err != nil {
		return fmt.Errorf("checking existence of default resource group %q: %w", defaultResourceGroupName, err)
	}

	if resp.StatusCode == http.StatusOK || resp.StatusCode == http.StatusNoContent {
		return nil
	}

	_, err = cli.CreateOrUpdate(ctx, defaultResourceGroupName, resources.Group{
		Location: to.Ptr(defaultResourceGroupRegion),
	})
	if err != nil {
		return fmt.Errorf("creating default resource group %q: %w", defaultResourceGroupName, err)
	}

	event.Normal(ctx, ReasonResourceGroupCreated, "Created resource group %q", defaultResourceGroupName)

	return nil
}

// systemTopicResourceName returns a deterministic name for an Event Grid
// system topic associated with the given source instance.
// System topic names can only contain A-Z, a-z, 0-9, and the '-' character.
func systemTopicResourceName(src *v1alpha1.AzureEventGridSource) string {
	scopeChecksum := crc32.ChecksumIEEE([]byte(strings.ToLower(src.Spec.Scope.String())))
	return "io-triggermesh-azureeventgridsources-" + strconv.FormatUint(uint64(scopeChecksum), 10)
}

// failFindSystemTopicEvent returns a reconciler event which indicates that a
// system topic for the given Azure resource (scope) could not be retrieved
// from the Azure API.
func failFindSystemTopicEvent(resource string, origErr error) reconciler.Event {
	return reconciler.NewEvent(corev1.EventTypeWarning, ReasonFailedSystemTopic,
		"Error finding system topic for resource %q: %s", resource, toErrMsg(origErr))
}

// failSystemTopicEvent returns a reconciler event which indicates that a
// system topic for the given Azure resource (scope) could not be created or
// updated via the Azure API.
func failSystemTopicEvent(resource string, isUpdate bool, origErr error) reconciler.Event {
	verb := "creating"
	if isUpdate {
		verb = "updating"
	}

	return reconciler.NewEvent(corev1.EventTypeWarning, ReasonFailedSystemTopic,
		"Error %s system topic for resource %q: %s", verb, resource, toErrMsg(origErr))
}

// failFinalizeSystemTopicEvent returns a reconciler event which indicates that
// a system topic could not be finalized.
func failFinalizeSystemTopicEvent(sysTopic string, origErr error) reconciler.Event {
	return reconciler.NewEvent(corev1.EventTypeWarning, ReasonFailedSystemTopic,
		"Error finalizing system topic %q: %s", sysTopic, toErrMsg(origErr))
}

// failResourceGroupEvent returns a reconciler event which indicates that a
// resource group could not be created via the Azure API.
func failResourceGroupEvent(name string, origErr error) reconciler.Event {
	return reconciler.NewEvent(corev1.EventTypeWarning, ReasonFailedResourceGroup,
		"Error ensuring resource group %q: %s", name, toErrMsg(origErr))
}

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

package azureactivitylogssource

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"sort"
	"time"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"knative.dev/pkg/controller"
	"knative.dev/pkg/logging"
	"knative.dev/pkg/reconciler"

	"github.com/Azure/azure-sdk-for-go/profiles/preview/preview/monitor/mgmt/insights"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/to"
	"github.com/Azure/go-autorest/autorest"
	"github.com/Azure/go-autorest/autorest/adal"
	"github.com/Azure/go-autorest/autorest/azure"

	commonv1alpha1 "github.com/triggermesh/triggermesh/pkg/apis/common/v1alpha1"
	"github.com/triggermesh/triggermesh/pkg/apis/sources/v1alpha1"
	"github.com/triggermesh/triggermesh/pkg/reconciler/event"
	"github.com/triggermesh/triggermesh/pkg/reconciler/skip"
	"github.com/triggermesh/triggermesh/pkg/sources/auth"
)

const (
	resourceTypeEventHubs  = "eventhubs"
	resourceTypeAuthzRules = "authorizationRules"
)

const defaultSASPolicyName = "RootManageSharedAccessKey"

const crudTimeout = time.Second * 15

// ensureDiagnosticSettings ensures diagnostic settings exist with the expected configuration.
// Required permissions:
//   - Microsoft.Insights/DiagnosticSettings/Read
//   - Microsoft.Insights/DiagnosticSettings/Write
//   - Microsoft.EventHub/namespaces/authorizationRules/listkeys/action
func (r *Reconciler) ensureDiagnosticSettings(ctx context.Context) error {
	if skip.Skip(ctx) {
		return nil
	}

	src := commonv1alpha1.ReconcilableFromContext(ctx).(*v1alpha1.AzureActivityLogsSource)
	status := &src.Status

	// initialize clients

	eventCatCli, diagSettingsCli, err := r.cg.Get(src)
	switch {
	case isNoCredentials(err):
		status.MarkNotSubscribed(v1alpha1.AzureReasonNoClient, "Azure credentials missing: "+toErrMsg(err))
		return controller.NewPermanentError(reconciler.NewEvent(corev1.EventTypeWarning, ReasonFailedSubscribe,
			"Azure credentials missing: %s", toErrMsg(err)))
	case err != nil:
		status.MarkNotSubscribed(v1alpha1.AzureReasonNoClient, "Error obtaining Azure clients: "+toErrMsg(err))
		// wrap any other error to fail the reconciliation
		return fmt.Errorf("%w", reconciler.NewEvent(corev1.EventTypeWarning, ReasonFailedSubscribe,
			"Error obtaining Azure clients: %s", err))
	}

	// read current diagnostics settings

	subsID := src.Spec.SubscriptionID
	subsResID := subscriptionResourceID(subsID)
	diagsName := diagnosticSettingsName(src)

	restCtx, cancel := context.WithTimeout(ctx, crudTimeout)
	defer cancel()

	currentDiagSettings, err := diagSettingsCli.Get(restCtx, subsResID, diagsName)
	switch {
	case isNotFound(err):
		// no-op
	case isDenied(err):
		status.MarkNotSubscribed(v1alpha1.AzureReasonAPIError,
			"Access denied to Diagnostic Settings API: "+toErrMsg(err))
		return controller.NewPermanentError(failGetDiagSettingsEvent(subsID, err))
	case err != nil:
		status.MarkNotSubscribed(v1alpha1.AzureReasonAPIError,
			"Cannot look up Diagnostic Settings: "+toErrMsg(err))
		return fmt.Errorf("%w", failGetDiagSettingsEvent(subsID, err))
	}

	hasDiagSettings := currentDiagSettings.ID != nil

	// read available logs categories

	// NOTE(antoineco): this step could potentially be removed in favour of
	// a hardcoded list, since the list of supported categories is
	// documented at https://docs.microsoft.com/en-us/rest/api/monitor/eventcategories/list
	availCatList, err := eventCatCli.List(ctx)
	if err != nil {
		switch {
		case isDenied(err):
			status.MarkNotSubscribed(v1alpha1.AzureReasonAPIError,
				"Access denied to Event Categories API: "+toErrMsg(err))
			return controller.NewPermanentError(failGetEventCategoriesEvent(err))
		case err != nil:
			status.MarkNotSubscribed(v1alpha1.AzureReasonAPIError,
				"Cannot list supported event categories: "+toErrMsg(err))
			return fmt.Errorf("%w", failGetEventCategoriesEvent(err))
		}
	}

	availCats := eventCategories(availCatList)

	logging.FromContext(ctx).Debugf("Supported Diagnostics Settings categories for subscription %s: %s",
		subsID, availCats)

	// generate desired diagnostic settings

	desiredCats, err := initDesiredCategories(ctx, src, availCats)
	if err != nil {
		status.MarkNotSubscribed(v1alpha1.ReasonFailedSync, "Invalid log categories: "+toErrMsg(err))
		return controller.NewPermanentError(reconciler.NewEvent(corev1.EventTypeWarning, ReasonFailedSubscribe,
			"Error initializing log categories from source spec: %s", toErrMsg(err)))
	}

	logSettings := initLogSettings(desiredCats, availCats)

	sasPolName := defaultSASPolicyName
	if pn := src.Spec.Destination.EventHubs.SASPolicy; pn != nil && *pn != "" {
		sasPolName = *pn
	}

	nsID := src.Spec.Destination.EventHubs.NamespaceID

	var eventHubName *string
	if hubName := src.Spec.Destination.EventHubs.HubName; hubName != nil && *hubName != "" {
		eventHubName = hubName
	}

	desiredDiagSettings := insights.DiagnosticSettingsResource{
		DiagnosticSettings: &insights.DiagnosticSettings{
			EventHubAuthorizationRuleID: to.Ptr(sasPolicyResourceID(&nsID, sasPolName)),
			EventHubName:                eventHubName,
			Logs:                        &logSettings,
		},
	}

	// compare and create/update diagnostic settings

	if r.equalDiagnosticSettings(ctx, desiredDiagSettings, currentDiagSettings) {
		status.MarkSubscribed()
		return nil
	}

	restCtx, cancel = context.WithTimeout(ctx, crudTimeout)
	defer cancel()

	_, err = diagSettingsCli.CreateOrUpdate(restCtx, subsResID, desiredDiagSettings, diagsName)
	switch {
	case isDenied(err):
		status.MarkNotSubscribed(v1alpha1.AzureReasonAPIError,
			"Access denied to Diagnostic Settings API: "+toErrMsg(err))
		return controller.NewPermanentError(failSubscribeEvent(subsID, hasDiagSettings, err))
	case err != nil:
		status.MarkNotSubscribed(v1alpha1.AzureReasonAPIError,
			"Cannot subscribe to Activity Logs: "+toErrMsg(err))
		return fmt.Errorf("%w", failSubscribeEvent(subsID, hasDiagSettings, err))
	}

	recordSubscribedEvent(ctx, diagsName, subsID, hasDiagSettings)

	status.MarkSubscribed()

	return nil
}

// sasPolicyResourceID returns the resource ID of a SAS policy identified by
// name, relatively to the given Event Hub's namepace.
func sasPolicyResourceID(namespaceID *v1alpha1.AzureResourceID, polName string) string {
	return namespaceID.String() + "/" + resourceTypeAuthzRules + "/" + polName
}

// ensureNoDiagnosticSettings ensures diagnostic settings are removed.
// Required permissions:
//   - Microsoft.Insights/DiagnosticSettings/Delete
func (r *Reconciler) ensureNoDiagnosticSettings(ctx context.Context) reconciler.Event {
	if skip.Skip(ctx) {
		return nil
	}

	src := commonv1alpha1.ReconcilableFromContext(ctx).(*v1alpha1.AzureActivityLogsSource)

	_, diagSettingsCli, err := r.cg.Get(src)
	switch {
	case isNoCredentials(err):
		// the finalizer is unlikely to recover from missing
		// credentials, so we simply record a warning event and return
		event.Warn(ctx, ReasonFailedUnsubscribe, "Azure credentials missing while finalizing event source. "+
			"Ignoring: %s", err)
		return nil
	case err != nil:
		return reconciler.NewEvent(corev1.EventTypeWarning, ReasonFailedUnsubscribe,
			"Error creating Azure clients: %s", err)
	}

	subsID := src.Spec.SubscriptionID
	subsResID := subscriptionResourceID(subsID)
	diagsName := diagnosticSettingsName(src)

	ctx, cancel := context.WithTimeout(ctx, crudTimeout)
	defer cancel()

	_, err = diagSettingsCli.Delete(ctx, subsResID, diagsName)
	switch {
	case isNotFound(err):
		event.Warn(ctx, ReasonUnsubscribed, "Diagnostic Settings not found, skipping deletion")
		return nil
	case isDenied(err):
		// it is unlikely that we recover from auth errors in the
		// finalizer, so we simply record a warning event and return
		event.Warn(ctx, ReasonFailedUnsubscribe,
			"Access denied to Diagnostic Settings API. Ignoring: %s", toErrMsg(err))
		return nil
	case err != nil:
		return failUnsubscribeEvent(subsID, err)
	}

	event.Normal(ctx, ReasonUnsubscribed, "Deleted Diagnostic Settings %q for subscription %q",
		diagsName, subsID)

	return nil
}

// subscriptionResourceID returns the resource ID of an Azure subscription,
// based on its ID.
func subscriptionResourceID(subsID string) string {
	subsResID := &v1alpha1.AzureResourceID{
		SubscriptionID: subsID,
	}
	return subsResID.String()
}

// equalDiagnosticSettings asserts the equality of two DiagnosticSettingsResource.
func (r *Reconciler) equalDiagnosticSettings(ctx context.Context,
	desired, current insights.DiagnosticSettingsResource) bool {

	cmpFn := cmp.Equal
	if logger := logging.FromContext(ctx); logger.Desugar().Core().Enabled(zapcore.DebugLevel) {
		cmpFn = diffLoggingCmp(logger)
	}
	return cmpFn(desired.DiagnosticSettings, current.DiagnosticSettings, cmpopts.SortSlices(lessLogSettings))
}

// diagnosticSettingsName returns a deterministic name for some Activity Logs
// diagnostic settings.
func diagnosticSettingsName(o *v1alpha1.AzureActivityLogsSource) string {
	return "io.triggermesh.azureactivitylogssource." + o.Namespace + "." + o.Name
}

// initDesiredCategories initializes a list of log categories to enable by cross
// comparing the values from the source's spec with a list of available values.
func initDesiredCategories(ctx context.Context,
	o *v1alpha1.AzureActivityLogsSource, availCategories stringSet) (stringSet, error) {

	if o.Spec.Categories == nil {
		return nil, nil
	}

	desiredCategories := make(stringSet, len(o.Spec.Categories))

	for _, cat := range o.Spec.Categories {
		if !availCategories.has(cat) {
			logging.FromContext(ctx).Warn("Unknown Activity Log category " + cat)
			continue
		}
		desiredCategories.set(cat)
	}

	if len(desiredCategories) == 0 {
		return nil, fmt.Errorf("object spec does not contain any valid log category. "+
			"Valid values include %s", availCategories)
	}
	return desiredCategories, nil
}

// initLogSettings returns a new instance of LogSettings with desired log
// categories enabled or disabled based on the source's spec.
func initLogSettings(desiredCategories, availCategories stringSet) []insights.LogSettings {
	logSettings := make([]insights.LogSettings, 0, len(availCategories))

	for cat := range availCategories {
		var enable bool
		if desiredCategories == nil || desiredCategories.has(cat) {
			enable = true
		}

		logSettings = append(logSettings,
			insights.LogSettings{
				Category: to.Ptr(cat),
				Enabled:  &enable,
			},
		)
	}

	return logSettings
}

// toErrMsg attempts to extract the message from the given error if it is an an
// Azure API error. Used to sanitize error logs before writing them to objects'
// status conditions.
func toErrMsg(err error) string {
	return recursErrMsg("", err)
}

// recursErrMsg concatenates the messages of deeply nested API errors recursively.
func recursErrMsg(errMsg string, err error) string {
	if errMsg != "" {
		errMsg += ": "
	}

	switch tErr := err.(type) {
	case autorest.DetailedError:
		return recursErrMsg(errMsg+tErr.Message, tErr.Original)
	case *azure.RequestError:
		if tErr.DetailedError.Original != nil {
			return recursErrMsg(errMsg+tErr.DetailedError.Message, tErr.DetailedError.Original)
		}
		if tErr.ServiceError != nil {
			return errMsg + tErr.ServiceError.Message
		}
	case adal.TokenRefreshError:
		// This type of error is returned when the OAuth authentication with Azure Active Directory fails, often
		// due to an invalid or expired secret.
		//
		// The associated message is typically opaque and contains elements that are unique to each request
		// (trace/correlation IDs, timestamps), which causes an infinite loop of reconciliation if propagated to
		// the object's status conditions.
		// Instead of resorting to over-engineered error parsing techniques to get around the verbosity of the
		// message, we simply return a short and generic error description.
		return errMsg + "failed to refresh token: the provided secret is either invalid or expired"
	}

	return errMsg + err.Error()
}

// isNotFound returns whether the given error indicates that some Azure
// resource was not found.
func isNotFound(err error) bool {
	if dErr := (autorest.DetailedError{}); errors.As(err, &dErr) {
		return dErr.StatusCode == http.StatusNotFound
	}
	return false
}

// isNoCredentials returns whether the given error indicates that some required
// Azure credentials could not be obtained.
func isNoCredentials(err error) bool {
	// consider that missing Secrets indicate missing credentials in this context
	if k8sErr := apierrors.APIStatus(nil); errors.As(err, &k8sErr) {
		return k8sErr.Status().Reason == metav1.StatusReasonNotFound
	}
	if permErr := (auth.PermanentCredentialsError)(nil); errors.As(err, &permErr) {
		return true
	}
	return false
}

// isDenied returns whether the given error indicates that a request to the
// Azure API could not be authorized.
// This category of issues is unrecoverable without user intervention.
func isDenied(err error) bool {
	if dErr := (autorest.DetailedError{}); errors.As(err, &dErr) {
		if code, ok := dErr.StatusCode.(int); ok {
			return code == http.StatusUnauthorized || code == http.StatusForbidden
		}
	}

	return false
}

// eventCategories returns an EventCategoryCollection as a set of category names.
func eventCategories(ecc insights.EventCategoryCollection) stringSet /*categories*/ {
	cats := stringSet{}
	for _, cat := range *ecc.Value {
		cats.set(*cat.Value)
	}
	return cats
}

// stringSet is a set of string elements.
type stringSet map[string]struct{}

var _ fmt.Stringer = (stringSet)(nil)

func (s stringSet) set(item string) {
	s[item] = struct{}{}
}

func (s stringSet) has(item string) bool {
	_, ok := s[item]
	return ok
}

// String implements the fmt.Stringer interface.
func (s stringSet) String() string {
	items := make([]string, 0, len(s))
	for item := range s {
		items = append(items, item)
	}

	sort.Strings(items)

	return fmt.Sprint(items)
}

// cmpFunc can compare the equality of two interfaces. The function signature
// is the same as cmp.Equal.
type cmpFunc func(x, y interface{}, opts ...cmp.Option) bool

// diffLoggingCmp compares the equality of two interfaces and logs the diff at
// the Debug level.
func diffLoggingCmp(logger *zap.SugaredLogger) cmpFunc {
	return func(desired, current interface{}, opts ...cmp.Option) bool {
		if diff := cmp.Diff(desired, current, opts...); diff != "" {
			logger.Debug("Diagnostic Settings differ (-desired, +current)\n" + diff)
			return false
		}
		return true
	}
}

// lessLogSettings reports whether the LogSettings element with index i should
// sort before the LogSettings element with index j.
func lessLogSettings(i, j insights.LogSettings) bool {
	return *i.Category < *j.Category
}

// recordSubscribedEvent records a Kubernetes API event which indicates that
// Diagnostic Settings were either created or updated.
func recordSubscribedEvent(ctx context.Context, diagsName, subsID string, isUpdate bool) {
	verb := "Created"
	if isUpdate {
		verb = "Updated"
	}

	event.Normal(ctx, ReasonSubscribed, "%s Diagnostic Settings %q for Azure subscription %q",
		verb, diagsName, subsID)
}

// failGetDiagSettingsEvent returns a reconciler event which indicates that
// Diagnostic Settings for the given Azure subscription could not be retrieved
// from the Azure API.
func failGetDiagSettingsEvent(subsID string, origErr error) reconciler.Event {
	return reconciler.NewEvent(corev1.EventTypeWarning, ReasonFailedSubscribe,
		"Error getting Diagnostic Settings for Azure subscription %q: %s", subsID, toErrMsg(origErr))
}

// failGetEventCategoriesEvent returns a reconciler event which indicates that
// available event categories for Diagnostic Settings could not be retrieved
// from the Azure API.
func failGetEventCategoriesEvent(origErr error) reconciler.Event {
	return reconciler.NewEvent(corev1.EventTypeWarning, ReasonFailedSubscribe,
		"Error getting list of supported event categories for Diagnostic Settings: %s", toErrMsg(origErr))
}

// failSubscribeEvent returns a reconciler event which indicates that
// Diagnostic Settings for the given Azure subscription could not be created or
// updated via the Azure API.
func failSubscribeEvent(subsID string, isUpdate bool, origErr error) reconciler.Event {
	verb := "creating"
	if isUpdate {
		verb = "updating"
	}

	return reconciler.NewEvent(corev1.EventTypeWarning, ReasonFailedSubscribe,
		"Error %s Diagnostic Settings for Azure subscription %q: %s", verb, subsID, toErrMsg(origErr))
}

// failUnsubscribeEvent returns a reconciler event which indicates that
// Diagnostic Settings for the given Azure subscription could not be deleted
// via the Azure API.
func failUnsubscribeEvent(subsID string, origErr error) reconciler.Event {
	return reconciler.NewEvent(corev1.EventTypeWarning, ReasonFailedSubscribe,
		"Error deleting Diagnostic Settings for Azure subscription %q: %s", subsID, toErrMsg(origErr))
}

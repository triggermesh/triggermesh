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

package googlecloudsourcerepositoriessource

import (
	"context"
	"fmt"

	gsourcerepo "google.golang.org/api/sourcerepo/v1"

	corev1 "k8s.io/api/core/v1"
	"knative.dev/pkg/controller"
	"knative.dev/pkg/reconciler"

	commonv1alpha1 "github.com/triggermesh/triggermesh/pkg/apis/common/v1alpha1"
	"github.com/triggermesh/triggermesh/pkg/apis/sources/v1alpha1"
	"github.com/triggermesh/triggermesh/pkg/reconciler/event"
	"github.com/triggermesh/triggermesh/pkg/reconciler/skip"
)

// Ensures that the Repo has the topic associated.
// Required permissions:
// - source.repos.updateRepoConfig
// - iam.serviceAccounts.actAs
func ensureTopicAssociated(ctx context.Context, cli *gsourcerepo.Service,
	topicResName *v1alpha1.GCloudResourceName, publishServiceAccount string) error {

	if skip.Skip(ctx) {
		return nil
	}

	src := commonv1alpha1.ReconcilableFromContext(ctx).(*v1alpha1.GoogleCloudSourceRepositoriesSource)
	status := &src.Status

	repoName := src.Spec.Repository.String()

	updateRepoRequest := &gsourcerepo.UpdateRepoRequest{
		Repo: &gsourcerepo.Repo{
			PubsubConfigs: map[string]gsourcerepo.PubsubConfig{
				topicResName.String(): {
					Topic:               topicResName.String(),
					MessageFormat:       "JSON",
					ServiceAccountEmail: publishServiceAccount,
				},
			},
		},
		UpdateMask: "pubsubConfigs",
	}

	patchRepo := cli.Projects.Repos.Patch(repoName, updateRepoRequest)

	_, err := patchRepo.Do()
	switch {
	case isDenied(err):
		status.MarkNotSubscribed(v1alpha1.GCloudReasonAPIError,
			"Access denied to Cloud Source Repositories API: "+toErrMsg(err))
		return controller.NewPermanentError(failEnableRepoNotifsEvent(repoName, err))
	case isNotFound(err):
		status.MarkNotSubscribed(v1alpha1.GCloudReasonAPIError,
			"Provided Source Repository was not found: "+toErrMsg(err))
		return controller.NewPermanentError(failEnableRepoNotifsEvent(repoName, err))
	case err != nil:
		status.MarkNotSubscribed(v1alpha1.GCloudReasonAPIError,
			"Cannot configure repository notifications: "+toErrMsg(err))
		return fmt.Errorf("%w", failEnableRepoNotifsEvent(repoName, err))
	}

	event.Normal(ctx, ReasonSubscribed, "Enabled notifications for Source Repository %q", repoName)
	status.MarkSubscribed()

	return err
}

// ensureNoTopicAssociated looks at status.Repositories and if non-empty will delete it
// Required permissions:
// - source.repos.updateRepoConfig
// - iam.serviceAccounts.actAs
func (r *Reconciler) ensureNoTopicAssociated(ctx context.Context, cli *gsourcerepo.Service) error {
	if skip.Skip(ctx) {
		return nil
	}

	src := commonv1alpha1.ReconcilableFromContext(ctx).(*v1alpha1.GoogleCloudSourceRepositoriesSource)

	repoName := src.Spec.Repository.String()

	updateRepoRequest := &gsourcerepo.UpdateRepoRequest{
		Repo: &gsourcerepo.Repo{
			PubsubConfigs: map[string]gsourcerepo.PubsubConfig{},
		},
		UpdateMask: "pubsubConfigs",
	}

	patchRepo := cli.Projects.Repos.Patch(repoName, updateRepoRequest)

	_, err := patchRepo.Do()
	switch {
	case isDenied(err):
		// it is unlikely that we recover from auth errors in the
		// finalizer, so we simply record a warning event and return
		event.Warn(ctx, ReasonFailedUnsubscribe,
			"Access denied to Cloud Source Repositories API. Ignoring: %s", toErrMsg(err))
		return nil
	case isNotFound(err):
		event.Warn(ctx, ReasonUnsubscribed,
			fmt.Sprintf("Source Repository %q not found, skipping deletion", repoName))
		return nil
	case err != nil:
		return failDisableRepoNotifsEvent(repoName, err)
	}

	event.Normal(ctx, ReasonUnsubscribed, "Disabled notifications for Source Repository %q", repoName)

	return err
}

// failEnableRepoNotifsEvent returns a reconciler event which indicates that
// notifications could not be enabled for a Source Repository.
func failEnableRepoNotifsEvent(repoName string, origErr error) reconciler.Event {
	return reconciler.NewEvent(corev1.EventTypeWarning, ReasonFailedSubscribe,
		"Error enabling notifications for Source Repository %q: %s", repoName, toErrMsg(origErr))
}

// failDisableRepoNotifsEvent returns a reconciler event which indicates that
// notifications could not be disabled for a Source Repository.
func failDisableRepoNotifsEvent(repoName string, origErr error) reconciler.Event {
	return reconciler.NewEvent(corev1.EventTypeWarning, ReasonFailedSubscribe,
		"Error disabling notifications for Source Repository %q: %s", repoName, toErrMsg(origErr))
}

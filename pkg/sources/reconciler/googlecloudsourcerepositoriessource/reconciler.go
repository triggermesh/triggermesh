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
	"errors"
	"fmt"
	"net/http"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"knative.dev/pkg/controller"
	"knative.dev/pkg/reconciler"

	"google.golang.org/api/googleapi"
	grpccodes "google.golang.org/grpc/codes"
	grpcstatus "google.golang.org/grpc/status"

	commonv1alpha1 "github.com/triggermesh/triggermesh/pkg/apis/common/v1alpha1"
	"github.com/triggermesh/triggermesh/pkg/apis/sources/v1alpha1"
	reconcilerv1alpha1 "github.com/triggermesh/triggermesh/pkg/client/generated/injection/reconciler/sources/v1alpha1/googlecloudsourcerepositoriessource"
	listersv1alpha1 "github.com/triggermesh/triggermesh/pkg/client/generated/listers/sources/v1alpha1"
	common "github.com/triggermesh/triggermesh/pkg/reconciler"
	"github.com/triggermesh/triggermesh/pkg/reconciler/event"
	"github.com/triggermesh/triggermesh/pkg/sources/client/gcloud/repositories"
)

// Reconciler implements controller.Reconciler for the event source type.
type Reconciler struct {
	// Getter than can obtain clients for interacting with Google Cloud APIs
	cg repositories.ClientGetter

	// Pub/Sub adapter
	base       common.GenericDeploymentReconciler[*v1alpha1.GoogleCloudSourceRepositoriesSource, listersv1alpha1.GoogleCloudSourceRepositoriesSourceNamespaceLister]
	adapterCfg *adapterConfig
}

// Check that our Reconciler implements Interface
var _ reconcilerv1alpha1.Interface = (*Reconciler)(nil)

// Check that our Reconciler implements Finalizer
var _ reconcilerv1alpha1.Finalizer = (*Reconciler)(nil)

// ReconcileKind implements Interface.ReconcileKind.
func (r *Reconciler) ReconcileKind(ctx context.Context, o *v1alpha1.GoogleCloudSourceRepositoriesSource) reconciler.Event {
	// inject source into context for usage in reconciliation logic
	ctx = commonv1alpha1.WithReconcilable(ctx, o)

	pubsubCli, repoCli, err := r.cg.Get(o)
	switch {
	case isNoCredentials(err):
		o.Status.MarkNotSubscribed(v1alpha1.GCloudReasonNoClient, "Google Cloud credentials missing: "+toErrMsg(err))
		return controller.NewPermanentError(reconciler.NewEvent(corev1.EventTypeWarning, ReasonFailedSubscribe,
			"Google Cloud credentials missing: %s", toErrMsg(err)))
	case err != nil:
		o.Status.MarkNotSubscribed(v1alpha1.GCloudReasonNoClient, "Error obtaining Google Cloud clients: "+toErrMsg(err))
		// wrap any other error to fail the reconciliation
		return fmt.Errorf("%w", reconciler.NewEvent(corev1.EventTypeWarning, ReasonFailedSubscribe,
			"Error obtaining Google Cloud clients: %s", err))
	}

	topic, err := EnsurePubSub(ctx, pubsubCli)
	if err != nil {
		return fmt.Errorf("failed to reconcile Pub/Sub resources: %w", err)
	}

	var publishServiceAccount string
	if sa := o.Spec.PublishServiceAccount; sa != nil {
		publishServiceAccount = *sa
	}

	if err = EnsureTopicAssociated(ctx, repoCli, topic, publishServiceAccount); err != nil {
		return fmt.Errorf("failed to reconcile Repo notification: %w", err)
	}

	return r.base.ReconcileAdapter(ctx, r)
}

// FinalizeKind is called when the resource is deleted.
func (r *Reconciler) FinalizeKind(ctx context.Context, o *v1alpha1.GoogleCloudSourceRepositoriesSource) reconciler.Event {
	// inject source into context for usage in finalization logic
	ctx = commonv1alpha1.WithReconcilable(ctx, o)

	pubsubCli, repoCli, err := r.cg.Get(o)
	switch {
	case isNoCredentials(err):
		// the finalizer is unlikely to recover from missing
		// credentials, so we simply record a warning event and return
		event.Warn(ctx, ReasonFailedUnsubscribe,
			"Google Cloud credentials missing while finalizing event source. Ignoring: %s", err)
		return nil
	case err != nil:
		return reconciler.NewEvent(corev1.EventTypeWarning, ReasonFailedUnsubscribe,
			"Error obtaining Google Cloud clients: %s", err)
	}

	// The finalizer blocks the deletion of the source object until
	// ensureNoTopicAssociated and ensureNoPubSub succeed to ensure that
	// we don't leave any dangling resources behind us.

	if err := EnsureNoTopicAssociated(ctx, repoCli); err != nil {
		return fmt.Errorf("failed to clean up Repo notification: %w", err)
	}

	if err := EnsureNoPubSub(ctx, pubsubCli); err != nil {
		return fmt.Errorf("failed to clean up Pub/Sub resources: %w", err)
	}

	return nil
}

// toErrMsg returns the given error as a string.
// If the error is a Google RPC status, the message contained in this status is returned.
func toErrMsg(err error) string {
	if s, ok := grpcstatus.FromError(err); ok {
		return s.Message()
	}

	if apiErr := (&googleapi.Error{}); errors.As(err, &apiErr) {
		return apiErr.Message
	}

	return err.Error()
}

// isNoCredentials returns whether the given error indicates that some required
// Google Cloud credentials could not be obtained.
func isNoCredentials(err error) bool {
	// consider that missing Secrets indicate missing credentials in this context
	if k8sErr := apierrors.APIStatus(nil); errors.As(err, &k8sErr) {
		return k8sErr.Status().Reason == metav1.StatusReasonNotFound
	}
	return false
}

// isNotFound returns whether the given error indicates that some Google Cloud
// resource was not found.
func isNotFound(err error) bool {
	if apiErr := (&googleapi.Error{}); errors.As(err, &apiErr) {
		return apiErr.Code == http.StatusNotFound
	}

	return grpcstatus.Code(err) == grpccodes.NotFound
}

// isDenied returns whether the given error indicates that a request to the
// Google Cloud API could not be authorized.
// This category of issues is unrecoverable without user intervention.
func isDenied(err error) bool {
	if apiErr := (&googleapi.Error{}); errors.As(err, &apiErr) {
		return apiErr.Code == http.StatusForbidden || apiErr.Code == http.StatusUnauthorized
	}

	grpcCode := grpcstatus.Code(err)
	return grpcCode == grpccodes.PermissionDenied || grpcCode == grpccodes.Unauthenticated
}

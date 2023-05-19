/*
Copyright 2023 TriggerMesh Inc.

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

package azureservicebussource

import (
	"context"
	"errors"
	"fmt"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"knative.dev/pkg/controller"
	"knative.dev/pkg/reconciler"

	commonv1alpha1 "github.com/triggermesh/triggermesh/pkg/apis/common/v1alpha1"
	"github.com/triggermesh/triggermesh/pkg/apis/sources/v1alpha1"
	reconcilerv1alpha1 "github.com/triggermesh/triggermesh/pkg/client/generated/injection/reconciler/sources/v1alpha1/azureservicebussource"
	listersv1alpha1 "github.com/triggermesh/triggermesh/pkg/client/generated/listers/sources/v1alpha1"
	common "github.com/triggermesh/triggermesh/pkg/reconciler"
	"github.com/triggermesh/triggermesh/pkg/reconciler/event"
	"github.com/triggermesh/triggermesh/pkg/sources/auth"
	"github.com/triggermesh/triggermesh/pkg/sources/client/azure/servicebus"
)

// Reconciler implements controller.Reconciler for the event source type.
type Reconciler struct {
	// Getter than can obtain clients for interacting with Azure APIs
	cg servicebus.ClientGetter

	// Event Hubs adapter
	base       common.GenericDeploymentReconciler[*v1alpha1.AzureServiceBusSource, listersv1alpha1.AzureServiceBusSourceNamespaceLister]
	adapterCfg *adapterConfig
}

// Check that our Reconciler implements Interface
var _ reconcilerv1alpha1.Interface = (*Reconciler)(nil)

// Check that our Reconciler implements Finalizer
var _ reconcilerv1alpha1.Finalizer = (*Reconciler)(nil)

// ReconcileKind implements Interface.ReconcileKind.
func (r *Reconciler) ReconcileKind(ctx context.Context, o *v1alpha1.AzureServiceBusSource) reconciler.Event {
	// inject source into context for usage in reconciliation logic
	ctx = commonv1alpha1.WithReconcilable(ctx, o)

	// In the case when TopicID is present, it is necessary to ensure the subscription.
	// when dealing with QueueID, there's no need.
	if o.Spec.TopicID != nil {
		subsCli, err := r.cg.Get(o)
		switch {
		case isNoCredentials(err):
			o.Status.MarkNotSubscribed(v1alpha1.AzureReasonNoClient, "Azure credentials missing: "+toErrMsg(err))
			return controller.NewPermanentError(reconciler.NewEvent(corev1.EventTypeWarning, ReasonFailedSubscribe,
				"Azure credentials missing: %s", toErrMsg(err)))
		case err != nil:
			o.Status.MarkNotSubscribed(v1alpha1.AzureReasonNoClient, "Error obtaining Azure clients: "+toErrMsg(err))
			// wrap any other error to fail the reconciliation
			return fmt.Errorf("%w", reconciler.NewEvent(corev1.EventTypeWarning, ReasonFailedSubscribe,
				"Error obtaining Azure clients: %s", err))
		}

		if err := EnsureSubscription(ctx, subsCli); err != nil {
			return fmt.Errorf("failed to reconcile Service Bus Subscription: %w", err)
		}
	}
	if o.Spec.QueueID != nil {
		o.Status.SubscriptionID = nil
		o.Status.MarkSubscribedWithReason("SUBSCRIPTIONNOTNEEDED", "QueueID is present, no subscription needed")
	}
	return r.base.ReconcileAdapter(ctx, r)
}

// FinalizeKind is called when the resource is deleted.
func (r *Reconciler) FinalizeKind(ctx context.Context, o *v1alpha1.AzureServiceBusSource) reconciler.Event {
	// In the case when QueueID is present, there's no need to finalize.
	if o.Spec.QueueID != nil {
		return nil
	}

	// inject source into context for usage in finalization logic
	ctx = commonv1alpha1.WithReconcilable(ctx, o)

	subsCli, err := r.cg.Get(o)
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

	// The finalizer blocks the deletion of the source object until the
	// deletion of the Subscription succeeds to ensure that we don't leave
	// any dangling resources behind us.
	return EnsureNoSubscription(ctx, subsCli)

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

/*
Copyright 2021 TriggerMesh Inc.

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

package common

import (
	"context"
	"fmt"
	"reflect"
	"strings"

	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime/schema"

	"knative.dev/pkg/apis"
	"knative.dev/pkg/controller"
	"knative.dev/pkg/kmeta"
	"knative.dev/pkg/reconciler"
	"knative.dev/serving/pkg/apis/serving"
	servingv1 "knative.dev/serving/pkg/apis/serving/v1"

	"github.com/triggermesh/triggermesh/pkg/apis/routing/v1alpha1"
	"github.com/triggermesh/triggermesh/pkg/routing/reconciler/common/event"
	"github.com/triggermesh/triggermesh/pkg/routing/reconciler/common/semantic"
)

// List of annotations set on Knative Serving objects by the Knative Serving
// admission webhook.
var knativeServingAnnotations = []string{
	serving.CreatorAnnotation,
	serving.UpdaterAnnotation,
}

// RBACOwnersLister returns a list of OwnerRefable to be set as a the
// OwnerReferences metadata attribute of a ServiceAccount.
type RBACOwnersLister interface {
	RBACOwners(rcl v1alpha1.Reconcilable) ([]kmeta.OwnerRefable, error)
}

// AdapterServiceBuilder provides all the necessary information for building
// objects related to a component's adapter backed by a Knative Service.
type AdapterServiceBuilder interface {
	RBACOwnersLister
	BuildAdapter(rcl v1alpha1.Reconcilable, sinkURI *apis.URL) *servingv1.Service
}

// ReconcileAdapter reconciles a receive adapter for a component instance.
func (r *GenericServiceReconciler) ReconcileAdapter(ctx context.Context, ab AdapterServiceBuilder) reconciler.Event {
	rcl := v1alpha1.ReconcilableFromContext(ctx)

	rcl.GetStatusManager().CloudEventAttributes = CreateCloudEventAttributes(
		rcl.AsEventSource(), rcl.GetEventTypes())

	sinkURI, err := r.resolveSinkURL(ctx)
	if err != nil {
		rcl.GetStatusManager().MarkNoSink()
		return controller.NewPermanentError(reconciler.NewEvent(corev1.EventTypeWarning,
			ReasonBadSinkURI, "Could not resolve sink URI: %s", err))
	}
	rcl.GetStatusManager().MarkSink(sinkURI)

	desiredAdapter := ab.BuildAdapter(rcl, sinkURI)

	saOwners, err := ab.RBACOwners(rcl)
	if err != nil {
		return fmt.Errorf("listing ServiceAccount owners: %w", err)
	}

	if err := r.reconcileAdapter(ctx, desiredAdapter, saOwners); err != nil {
		return fmt.Errorf("failed to reconcile adapter: %w", err)
	}
	return nil
}

// resolveSinkURL resolves the URL of a sink reference.
func (r *GenericServiceReconciler) resolveSinkURL(ctx context.Context) (*apis.URL, error) {
	rcl := v1alpha1.ReconcilableFromContext(ctx)
	sink := rcl.GetSink()

	if sinkRef := sink.Ref; sinkRef != nil && sinkRef.Namespace == "" {
		sinkRef.Namespace = rcl.GetNamespace()
	}

	return r.SinkResolver.URIFromDestinationV1(ctx, *sink, rcl)
}

// reconcileAdapter reconciles the state of the component's adapter.
func (r *GenericServiceReconciler) reconcileAdapter(ctx context.Context,
	desiredAdapter *servingv1.Service, rbacOwners []kmeta.OwnerRefable) error {

	rcl := v1alpha1.ReconcilableFromContext(ctx)

	isMultiTenant := v1alpha1.IsMultiTenant(rcl)

	sa, err := r.reconcileRBAC(ctx, rbacOwners)
	if err != nil {
		rcl.GetStatusManager().MarkRBACNotBound()
		return fmt.Errorf("reconciling RBAC objects: %w", err)
	}

	if isMultiTenant {
		// delegate ownership to the ServiceAccount in order to cause a
		// garbage collection once all instances of the given component
		// type have been deleted from the namespace
		OwnByServiceAccount(desiredAdapter, sa)
	}

	currentAdapter, err := r.getOrCreateAdapter(ctx, desiredAdapter)
	if err != nil {
		rcl.GetStatusManager().PropagateServiceAvailability(currentAdapter)
		return err
	}

	currentAdapter, err = r.syncAdapterService(ctx, currentAdapter, desiredAdapter)
	if err != nil {
		return fmt.Errorf("failed to synchronize adapter Service: %w", err)
	}

	rcl.GetStatusManager().PropagateServiceAvailability(currentAdapter)
	if isMultiTenant {
		rcl.GetStatusManager().SetRoute(URLPath(rcl))
	}

	return nil
}

// getOrCreateAdapter returns the existing adapter Service for a given
// component instance, or creates it if it is missing.
func (r *GenericServiceReconciler) getOrCreateAdapter(ctx context.Context, desiredAdapter *servingv1.Service) (*servingv1.Service, error) {
	rcl := v1alpha1.ReconcilableFromContext(ctx)

	adapter, err := findAdapter(r, rcl, metav1.GetControllerOfNoCopy(desiredAdapter))
	switch {
	case apierrors.IsNotFound(err):
		adapter, err = r.Client(rcl.GetNamespace()).Create(ctx, desiredAdapter, metav1.CreateOptions{})
		if err != nil {
			return nil, reconciler.NewEvent(corev1.EventTypeWarning, ReasonFailedAdapterCreate,
				"Failed to create adapter Service %q: %s", desiredAdapter.Name, err)
		}
		event.Normal(ctx, ReasonAdapterCreate, "Created adapter Service %q", adapter.GetName())

	case err != nil:
		return nil, fmt.Errorf("failed to get adapter Service from cache: %w", err)
	}

	return adapter.(*servingv1.Service), nil
}

// syncAdapterService synchronizes the desired state of an adapter Service
// against its current state in the running cluster.
func (r *GenericServiceReconciler) syncAdapterService(ctx context.Context,
	currentAdapter, desiredAdapter *servingv1.Service) (*servingv1.Service, error) {

	if semantic.Semantic.DeepEqual(desiredAdapter, currentAdapter) {
		return currentAdapter, nil
	}

	// resourceVersion must be returned to the API server unmodified for
	// optimistic concurrency, as per Kubernetes API conventions
	desiredAdapter.ResourceVersion = currentAdapter.ResourceVersion

	// immutable Knative annotations must be preserved
	for _, ann := range knativeServingAnnotations {
		if val, ok := currentAdapter.Annotations[ann]; ok {
			metav1.SetMetaDataAnnotation(&desiredAdapter.ObjectMeta, ann, val)
		}
	}

	// (fake Clientset) preserve status to avoid resetting conditions
	desiredAdapter.Status = currentAdapter.Status

	adapter, err := r.Client(currentAdapter.Namespace).Update(ctx, desiredAdapter, metav1.UpdateOptions{})
	if err != nil {
		return nil, reconciler.NewEvent(corev1.EventTypeWarning, ReasonFailedAdapterUpdate,
			"Failed to update adapter Service %q: %s", desiredAdapter.Name, err)
	}
	event.Normal(ctx, ReasonAdapterUpdate, "Updated adapter Service %q", adapter.Name)

	return adapter, nil
}

// findAdapter returns the adapter object for a given component instance if it exists.
func findAdapter(genericReconciler *GenericServiceReconciler,
	rcl v1alpha1.Reconcilable, owner *metav1.OwnerReference) (metav1.Object, error) {

	ls := CommonObjectLabels(rcl)

	if !v1alpha1.IsMultiTenant(rcl) {
		// the combination of standard labels {name,instance} is unique
		// and immutable for single-tenant components
		ls[appInstanceLabel] = rcl.GetName()
	}

	sel := labels.SelectorFromValidatedSet(ls)

	svcs, err := genericReconciler.Lister(rcl.GetNamespace()).List(sel)
	if err != nil {
		return nil, err
	}

	for _, obj := range svcs {
		objOwner := metav1.GetControllerOfNoCopy(obj)

		if objOwner.UID == owner.UID {
			return obj, nil
		}
	}

	gr := servingv1.Resource("service")

	return nil, newNotFoundForSelector(gr, sel)
}

// newNotFoundForSelector returns an error which indicates that no object of
// the type matching the given GroupResource was found for the given label
// selector.
func newNotFoundForSelector(gr schema.GroupResource, sel labels.Selector) *apierrors.StatusError {
	err := apierrors.NewNotFound(gr, "")
	err.ErrStatus.Message = fmt.Sprint(gr, " not found for selector ", sel)
	return err
}

// reconcileRBAC wraps the reconciliation logic for RBAC objects.
func (r *GenericRBACReconciler) reconcileRBAC(ctx context.Context,
	owners []kmeta.OwnerRefable) (*corev1.ServiceAccount, error) {

	// The ServiceAccount's ownership is shared between all instances of a
	// given component type. It gets garbage collected by Kubernetes as
	// soon as its last owner (component instance) is deleted, so we don't
	// need to clean things up explicitly once the last component instance
	// gets deleted.
	if len(owners) == 0 {
		return nil, nil
	}

	rcl := v1alpha1.ReconcilableFromContext(ctx)

	desiredSA := newServiceAccount(rcl, owners)
	currentSA, err := r.getOrCreateAdapterServiceAccount(ctx, desiredSA)
	if err != nil {
		return nil, err
	}

	if currentSA, err = r.syncAdapterServiceAccount(ctx, currentSA, desiredSA); err != nil {
		return nil, fmt.Errorf("synchronizing adapter ServiceAccount: %w", err)
	}

	if v1alpha1.IsMultiTenant(rcl) {
		desiredRB := newRoleBinding(rcl, currentSA)
		currentRB, err := r.getOrCreateAdapterRoleBinding(ctx, desiredRB)
		if err != nil {
			return nil, err
		}

		if _, err = r.syncAdapterRoleBinding(ctx, currentRB, desiredRB); err != nil {
			return nil, fmt.Errorf("synchronizing adapter RoleBinding: %w", err)
		}
	}

	return currentSA, nil
}

// getOrCreateAdapterServiceAccount returns the existing adapter ServiceAccount
// for a given component instance, or creates it if it is missing.
func (r *GenericRBACReconciler) getOrCreateAdapterServiceAccount(ctx context.Context,
	desiredSA *corev1.ServiceAccount) (*corev1.ServiceAccount, error) {

	rcl := v1alpha1.ReconcilableFromContext(ctx)

	sa, err := r.SALister(rcl.GetNamespace()).Get(desiredSA.Name)
	switch {
	case apierrors.IsNotFound(err):
		sa, err = r.SAClient(desiredSA.Namespace).Create(ctx, desiredSA, metav1.CreateOptions{})
		if err != nil {
			return nil, reconciler.NewEvent(corev1.EventTypeWarning, ReasonFailedRBACCreate,
				"Failed to create adapter ServiceAccount %q: %s", desiredSA.Name, err)
		}
		controller.GetEventRecorder(ctx).Eventf(newNamespace(desiredSA.Namespace), corev1.EventTypeNormal,
			ReasonRBACCreate, "Created ServiceAccount %q due to the creation of a %s object",
			sa.Name, rcl.GetGroupVersionKind().Kind)

	case err != nil:
		return nil, fmt.Errorf("getting adapter ServiceAccount from cache: %w", err)
	}

	return sa, nil
}

// syncAdapterServiceAccount synchronizes the desired state of an adapter
// ServiceAccount against its current state in the running cluster.
func (r *GenericRBACReconciler) syncAdapterServiceAccount(ctx context.Context,
	currentSA, desiredSA *corev1.ServiceAccount) (*corev1.ServiceAccount, error) {

	if reflect.DeepEqual(desiredSA.OwnerReferences, currentSA.OwnerReferences) {
		return currentSA, nil
	}

	// resourceVersion must be returned to the API server unmodified for
	// optimistic concurrency, as per Kubernetes API conventions
	desiredSA.ResourceVersion = currentSA.ResourceVersion

	// Kubernetes generates a secret named "<object_name>-token-<rand_id>"
	// upon creation of a ServiceAccount. We need to preserve the reference
	// to this secret during updates, otherwise Kubernetes generates a new
	// secret without garbage collecting the old one(s).
	for _, secr := range currentSA.Secrets {
		if strings.HasPrefix(secr.Name, desiredSA.Name+"-token-") {
			desiredSA.Secrets = append(desiredSA.Secrets, secr)
		}
	}

	sa, err := r.SAClient(desiredSA.Namespace).Update(ctx, desiredSA, metav1.UpdateOptions{})
	if err != nil {
		return nil, reconciler.NewEvent(corev1.EventTypeWarning, ReasonFailedRBACUpdate,
			"Failed to update adapter ServiceAccount %q: %s", desiredSA.Name, err)
	}

	controller.GetEventRecorder(ctx).Eventf(newNamespace(sa.Namespace), corev1.EventTypeNormal,
		ReasonRBACUpdate, "Updated ServiceAccount %q due to the creation/deletion of a %s object",
		sa.Name, v1alpha1.ReconcilableFromContext(ctx).GetGroupVersionKind().Kind)

	return sa, nil
}

// getOrCreateAdapterRoleBinding returns the existing adapter RoleBinding, or
// creates it if it is missing.
func (r *GenericRBACReconciler) getOrCreateAdapterRoleBinding(ctx context.Context,
	desiredRB *rbacv1.RoleBinding) (*rbacv1.RoleBinding, error) {

	rcl := v1alpha1.ReconcilableFromContext(ctx)

	rb, err := r.RBLister(desiredRB.Namespace).Get(desiredRB.Name)
	switch {
	case apierrors.IsNotFound(err):
		rb, err = r.RBClient(rcl.GetNamespace()).Create(ctx, desiredRB, metav1.CreateOptions{})
		if err != nil {
			return nil, reconciler.NewEvent(corev1.EventTypeWarning, ReasonFailedRBACCreate,
				"Failed to create adapter RoleBinding %q: %s", desiredRB.Name, err)
		}
		controller.GetEventRecorder(ctx).Eventf(newNamespace(rb.Namespace), corev1.EventTypeNormal,
			ReasonRBACCreate, "Created RoleBinding %q due to the creation of a %s object",
			rb.Name, rcl.GetGroupVersionKind().Kind)

	case err != nil:
		return nil, fmt.Errorf("getting adapter RoleBinding from cache: %w", err)
	}

	return rb, nil
}

// syncAdapterRoleBinding synchronizes the desired state of an adapter
// RoleBinding against its current state in the running cluster.
func (r *GenericRBACReconciler) syncAdapterRoleBinding(ctx context.Context,
	currentRB, desiredRB *rbacv1.RoleBinding) (*rbacv1.RoleBinding, error) {

	if reflect.DeepEqual(desiredRB.OwnerReferences, currentRB.OwnerReferences) &&
		reflect.DeepEqual(desiredRB.RoleRef, currentRB.RoleRef) &&
		reflect.DeepEqual(desiredRB.Subjects, currentRB.Subjects) {

		return currentRB, nil
	}

	// resourceVersion must be returned to the API server unmodified for
	// optimistic concurrency, as per Kubernetes API conventions
	desiredRB.ResourceVersion = currentRB.ResourceVersion

	rb, err := r.RBClient(desiredRB.Namespace).Update(ctx, desiredRB, metav1.UpdateOptions{})
	if err != nil {
		return nil, reconciler.NewEvent(corev1.EventTypeWarning, ReasonFailedRBACUpdate,
			"Failed to update adapter RoleBinding %q: %s", desiredRB.Name, err)
	}
	controller.GetEventRecorder(ctx).Eventf(newNamespace(rb.Namespace), corev1.EventTypeNormal,
		ReasonRBACUpdate, "Updated RoleBinding %q", rb.Name)

	return rb, nil
}

// newNamespace returns a Namespace object with the given name.
// It is used as a helper to record namespace-scoped API events.
func newNamespace(name string) *corev1.Namespace {
	return &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
	}
}

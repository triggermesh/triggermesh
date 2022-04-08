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

package reconciler

import (
	"context"
	"fmt"
	"reflect"
	"strings"

	appsv1 "k8s.io/api/apps/v1"
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

	"github.com/triggermesh/triggermesh/pkg/apis/common/v1alpha1"
	"github.com/triggermesh/triggermesh/pkg/mturl"
	"github.com/triggermesh/triggermesh/pkg/reconciler/event"
	"github.com/triggermesh/triggermesh/pkg/reconciler/resource"
	"github.com/triggermesh/triggermesh/pkg/reconciler/semantic"
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

// AdapterDeploymentBuilder provides all the necessary information for building
// objects related to a component's adapter backed by a Deployment.
type AdapterDeploymentBuilder interface {
	RBACOwnersLister
	BuildAdapter(rcl v1alpha1.Reconcilable, sinkURI *apis.URL) *appsv1.Deployment
}

// AdapterServiceBuilder provides all the necessary information for building
// objects related to a component's adapter backed by a Knative Service.
type AdapterServiceBuilder interface {
	RBACOwnersLister
	BuildAdapter(rcl v1alpha1.Reconcilable, sinkURI *apis.URL) *servingv1.Service
}

// ReconcileAdapter reconciles a receive adapter for a component instance.
func (r *GenericDeploymentReconciler) ReconcileAdapter(ctx context.Context, ab AdapterDeploymentBuilder) reconciler.Event {
	rcl := v1alpha1.ReconcilableFromContext(ctx)

	rcl.GetStatusManager().CloudEventAttributes = nil
	if src, isEventSource := rcl.(v1alpha1.EventSource); isEventSource {
		rcl.GetStatusManager().CloudEventAttributes = CreateCloudEventAttributes(
			src.AsEventSource(), src.GetEventTypes())
	}

	rcl.GetStatusManager().AcceptedEventTypes = nil
	if rcv, isEventReceiver := rcl.(v1alpha1.EventReceiver); isEventReceiver {
		rcl.GetStatusManager().AcceptedEventTypes = rcv.AcceptedEventTypes()
	}

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
func (r *GenericDeploymentReconciler) resolveSinkURL(ctx context.Context) (*apis.URL, error) {
	rcl := v1alpha1.ReconcilableFromContext(ctx)

	// If the current component type does not support sending events to a
	// sink, or if a particular instance does not define a sink Destination
	// as part of its spec, we return a nil apis.URL that is serializable
	// to an empty string. This effectively disables the sink feature and
	// clears any stale SinkProvided status condition (see MarkSink).
	sdr, isEventSender := rcl.(v1alpha1.EventSender)
	if !isEventSender {
		return nil, nil
	}

	sink := sdr.GetSink()
	if sink.Ref == nil && sink.URI == nil {
		return nil, nil
	}

	if sinkRef := sink.Ref; sinkRef != nil && sinkRef.Namespace == "" {
		sinkRef.Namespace = rcl.GetNamespace()
	}

	return r.SinkResolver.URIFromDestinationV1(ctx, *sink, rcl)
}

// reconcileAdapter reconciles the state of the component's adapter.
func (r *GenericDeploymentReconciler) reconcileAdapter(ctx context.Context,
	desiredAdapter *appsv1.Deployment, rbacOwners []kmeta.OwnerRefable) error {

	rcl := v1alpha1.ReconcilableFromContext(ctx)

	sa, err := r.reconcileRBAC(ctx, rbacOwners)
	if err != nil {
		rcl.GetStatusManager().MarkRBACNotBound()
		return fmt.Errorf("reconciling RBAC objects: %w", err)
	}

	if v1alpha1.IsMultiTenant(rcl) {
		// delegate ownership to the ServiceAccount in order to cause a
		// garbage collection once all instances of the given component
		// type have been deleted from the namespace
		OwnByServiceAccount(desiredAdapter, sa)
	}

	currentAdapter, err := r.getOrCreateAdapter(ctx, desiredAdapter)
	if err != nil {
		rcl.GetStatusManager().PropagateDeploymentAvailability(ctx, currentAdapter, r.PodClient(rcl.GetNamespace()))
		return err
	}

	currentAdapter, err = r.syncAdapterDeployment(ctx, currentAdapter, desiredAdapter)
	if err != nil {
		return fmt.Errorf("failed to synchronize adapter Deployment: %w", err)
	}
	rcl.GetStatusManager().PropagateDeploymentAvailability(ctx, currentAdapter, r.PodClient(rcl.GetNamespace()))

	return nil
}

// getOrCreateAdapter returns the existing adapter Deployment for a given
// component instance, or creates it if it is missing.
func (r *GenericDeploymentReconciler) getOrCreateAdapter(ctx context.Context, desiredAdapter *appsv1.Deployment) (*appsv1.Deployment, error) {
	rcl := v1alpha1.ReconcilableFromContext(ctx)

	adapter, err := findAdapter(r, rcl, metav1.GetControllerOfNoCopy(desiredAdapter))
	switch {
	case apierrors.IsNotFound(err):
		adapter, err = r.Client(rcl.GetNamespace()).Create(ctx, desiredAdapter, metav1.CreateOptions{})
		if err != nil {
			return nil, reconciler.NewEvent(corev1.EventTypeWarning, ReasonFailedAdapterCreate,
				"Failed to create adapter Deployment %q: %s", desiredAdapter.Name, err)
		}
		event.Normal(ctx, ReasonAdapterCreate, "Created adapter Deployment %q", adapter.GetName())

	case err != nil:
		return nil, fmt.Errorf("failed to get adapter Deployment from cache: %w", err)
	}

	return adapter.(*appsv1.Deployment), nil
}

// syncAdapterDeployment synchronizes the desired state of an adapter Deployment
// against its current state in the running cluster.
func (r *GenericDeploymentReconciler) syncAdapterDeployment(ctx context.Context,
	currentAdapter, desiredAdapter *appsv1.Deployment) (*appsv1.Deployment, error) {

	// We may have found an existing adapter object that is owned by the
	// component instance, but under a different name, e.g. created by an
	// older version of TriggerMesh.
	desiredAdapter.Name = currentAdapter.Name

	if semantic.Semantic.DeepEqual(desiredAdapter, currentAdapter) {
		return currentAdapter, nil
	}

	// resourceVersion must be returned to the API server unmodified for
	// optimistic concurrency, as per Kubernetes API conventions
	desiredAdapter.ResourceVersion = currentAdapter.ResourceVersion

	// (fake Clientset) preserve status to avoid resetting conditions
	desiredAdapter.Status = currentAdapter.Status

	adapter, err := r.Client(currentAdapter.Namespace).Update(ctx, desiredAdapter, metav1.UpdateOptions{})
	if err != nil {
		return nil, reconciler.NewEvent(corev1.EventTypeWarning, ReasonFailedAdapterUpdate,
			"Failed to update adapter Deployment %q: %s", desiredAdapter.Name, err)
	}
	event.Normal(ctx, ReasonAdapterUpdate, "Updated adapter Deployment %q", adapter.Name)

	return adapter, nil
}

// ReconcileAdapter reconciles a receive adapter for a component instance.
func (r *GenericServiceReconciler) ReconcileAdapter(ctx context.Context, ab AdapterServiceBuilder) reconciler.Event {
	rcl := v1alpha1.ReconcilableFromContext(ctx)

	rcl.GetStatusManager().CloudEventAttributes = nil
	if src, isEventSource := rcl.(v1alpha1.EventSource); isEventSource {
		rcl.GetStatusManager().CloudEventAttributes = CreateCloudEventAttributes(
			src.AsEventSource(), src.GetEventTypes())
	}

	rcl.GetStatusManager().AcceptedEventTypes = nil
	if rcv, isEventReceiver := rcl.(v1alpha1.EventReceiver); isEventReceiver {
		rcl.GetStatusManager().AcceptedEventTypes = rcv.AcceptedEventTypes()
	}

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

	// If the current component type does not support sending events to a
	// sink, or if a particular instance does not define a sink Destination
	// as part of its spec, we return a nil apis.URL that is serializable
	// to an empty string. This effectively disables the sink feature and
	// clears any stale SinkProvided status condition.
	sdr, isEventSender := rcl.(v1alpha1.EventSender)
	if !isEventSender {
		return nil, nil
	}

	sink := sdr.GetSink()
	if sink.Ref == nil && sink.URI == nil {
		return nil, nil
	}

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
		rcl.GetStatusManager().SetRoute(mturl.URLPath(rcl))
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

	// We may have found an existing adapter object that is owned by the
	// component instance, but under a different name, e.g. created by an
	// older version of TriggerMesh.
	desiredAdapter.Name = currentAdapter.Name

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
func findAdapter(genericReconciler interface{},
	rcl v1alpha1.Reconcilable, owner *metav1.OwnerReference) (metav1.Object, error) {

	ls := CommonObjectLabels(rcl)

	if !v1alpha1.IsMultiTenant(rcl) {
		// the combination of standard labels {name,instance} is unique
		// and immutable for single-tenant components
		ls[appInstanceLabel] = rcl.GetName()
	}

	sel := labels.SelectorFromValidatedSet(ls)

	var objs []metav1.Object
	var gr schema.GroupResource

	switch r := genericReconciler.(type) {
	case *GenericDeploymentReconciler:
		depls, err := r.Lister(rcl.GetNamespace()).List(sel)
		if err != nil {
			return nil, err
		}

		for _, d := range depls {
			objs = append(objs, d)
		}

		gr = appsv1.Resource("deployment")

	case *GenericServiceReconciler:
		svcs, err := r.Lister(rcl.GetNamespace()).List(sel)
		if err != nil {
			return nil, err
		}

		for _, s := range svcs {
			objs = append(objs, s)
		}

		gr = servingv1.Resource("service")
	}

	for _, obj := range objs {
		objOwner := metav1.GetControllerOfNoCopy(obj)

		if objOwner.UID == owner.UID {
			return obj, nil
		}
	}

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
	for _, m := range serviceAccountMutations(rcl) {
		m(desiredSA)
	}

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

	if semantic.Semantic.DeepEqual(desiredSA, currentSA) {
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

// serviceAccountMutations returns functional options for mutating the
// ServiceAccount associated with the given component instance.
func serviceAccountMutations(rcl v1alpha1.Reconcilable) []resource.ServiceAccountOption {
	if !v1alpha1.WantsOwnServiceAccount(rcl) {
		return nil
	}

	var saMutations []resource.ServiceAccountOption

	return append(saMutations, v1alpha1.ServiceAccountOptions(rcl)...)
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

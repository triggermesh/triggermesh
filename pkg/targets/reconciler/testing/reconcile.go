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

package testing

import (
	"context"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"

	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	clientgotesting "k8s.io/client-go/testing"

	"knative.dev/pkg/apis"
	"knative.dev/pkg/kmeta"
	"knative.dev/pkg/ptr"
	"knative.dev/pkg/reconciler"
	rt "knative.dev/pkg/reconciler/testing"
	servingv1 "knative.dev/serving/pkg/apis/serving/v1"

	"github.com/triggermesh/triggermesh/pkg/apis/targets"
	"github.com/triggermesh/triggermesh/pkg/apis/targets/v1alpha1"
	"github.com/triggermesh/triggermesh/pkg/targets/reconciler/common"
	"github.com/triggermesh/triggermesh/pkg/targets/reconciler/common/resource"
	"github.com/triggermesh/triggermesh/pkg/targets/routing"
	eventtesting "github.com/triggermesh/triggermesh/pkg/targets/testing/event"
)

const (
	tNs   = "testns"
	tName = "test"
	tKey  = tNs + "/" + tName
	tUID  = types.UID("00000000-0000-0000-0000-000000000000")
)

var (
	tAdapterURI = &apis.URL{
		Scheme: "http",
		Host:   "public.example.com",
		Path:   "/",
	}
)

// TestReconcileAdapter tests the Reconcile() method of the controller.Reconciler
// implemented by component Reconcilers, with focus on the generic ReconcileAdapter
// logic executed by the generic adapter reconciler embedded in every component Reconciler.
//
// The environment for each test case is set up as follows:
//  1. MakeFactory initializes fake clients with the objects declared in the test case
//  2. MakeFactory injects those clients into a context along with fake event recorders, etc.
//  3. A Reconciler is constructed via a Ctor function using the values injected above
//  4. The Reconciler returned by MakeFactory is used to run the test case
func TestReconcileAdapter(t *testing.T, ctor Ctor, rcl v1alpha1.Reconcilable, adapterBuilder interface{}) {
	assertPopulatedTarget(t, rcl)

	newEventTarget := targetCtor(rcl)
	newServiceAccount := NewServiceAccount(rcl)
	newRoleBinding := NewRoleBinding(newServiceAccount())
	newAdapter := adapterCtor(adapterBuilder, rcl)

	trg := newEventTarget()
	a := newAdapter()
	n, k, r := nameKindAndResource(a)

	testCases := rt.TableTest{
		// Creation/Deletion

		{
			Name: "Object creation",
			Key:  tKey,
			Objects: []runtime.Object{
				newEventTarget(noCEAttributes),
			},
			WantCreates: func() []runtime.Object {
				objs := []runtime.Object{
					newServiceAccount(NoToken),
					newAdapter(),
				}
				// only multi-tenant components expect a RoleBinding
				if v1alpha1.IsMultiTenant(trg) {
					return insertObject(objs, newRoleBinding(), 1)
				}
				return objs
			}(),
			WantStatusUpdates: []clientgotesting.UpdateActionImpl{{
				Object: newEventTarget(notDeployed(a)),
			}},
			WantEvents: func() []string {
				events := []string{
					createServiceAccountEvent(trg),
					createAdapterEvent(n, k),
				}
				// only multi-tenant components expect a RoleBinding
				if v1alpha1.IsMultiTenant(trg) {
					return insertString(events, createRoleBindingEvent(trg), 1)
				}
				return events
			}(),
		},
		{
			Name: "Object deletion",
			Key:  tKey,
			Objects: []runtime.Object{
				newEventTarget(deleted),
			},
		},

		// Lifecycle

		{
			Name: "Adapter becomes Ready",
			Key:  tKey,
			Objects: []runtime.Object{
				newEventTarget(notDeployed(a)),
				newServiceAccount(),
				newRoleBinding(),
				newAdapter(ready),
			},
			WantStatusUpdates: []clientgotesting.UpdateActionImpl{{
				Object: newEventTarget(deployed(a)),
			}},
		},
		{
			Name: "Adapter becomes NotReady",
			Key:  tKey,
			Objects: []runtime.Object{
				newEventTarget(deployed(a)),
				newServiceAccount(),
				newRoleBinding(),
				newAdapter(notReady),
			},
			WantStatusUpdates: []clientgotesting.UpdateActionImpl{{
				Object: newEventTarget(notDeployed(a)),
			}},
		},
		{
			Name: "Adapter is outdated",
			Key:  tKey,
			Objects: []runtime.Object{
				newEventTarget(deployed(a)),
				newServiceAccount(),
				newRoleBinding(),
				newAdapter(ready, bumpImage),
			},
			WantUpdates: []clientgotesting.UpdateActionImpl{{
				Object: newAdapter(ready),
			}},
			WantEvents: []string{
				updateAdapterEvent(n, k),
			},
		},
		{
			Name: "Service account owners are outdated",
			Key:  tKey,
			Objects: []runtime.Object{
				newEventTarget(deployed(a)),
				newServiceAccount(noOwner),
				newRoleBinding(),
				newAdapter(ready),
			},
			WantUpdates: []clientgotesting.UpdateActionImpl{{
				Object: newServiceAccount(),
			}},
			WantEvents: []string{
				updateServiceAccountEvent(trg),
			},
		},

		// Errors

		{
			Name: "Fail to create adapter",
			Key:  tKey,
			WithReactors: []clientgotesting.ReactionFunc{
				rt.InduceFailure("create", r),
			},
			Objects: []runtime.Object{
				newEventTarget(),
				newServiceAccount(),
				newRoleBinding(),
			},
			WantCreates: []runtime.Object{
				newAdapter(),
			},
			WantStatusUpdates: []clientgotesting.UpdateActionImpl{{
				Object: newEventTarget(unknownDeployedWithError(a)),
			}},
			WantEvents: []string{
				failCreateAdapterEvent(n, k, r),
			},
			WantErr: true,
		},
		{
			Name: "Fail to update adapter",
			Key:  tKey,
			WithReactors: []clientgotesting.ReactionFunc{
				rt.InduceFailure("update", r),
			},
			Objects: []runtime.Object{
				newEventTarget(deployed(a)),
				newServiceAccount(),
				newRoleBinding(),
				newAdapter(ready, bumpImage),
			},
			WantUpdates: []clientgotesting.UpdateActionImpl{{
				Object: newAdapter(ready),
			}},
			WantEvents: []string{
				failUpdateAdapterEvent(n, k, r),
			},
			WantErr: true,
		},

		// Edge cases

		{
			Name:    "Reconcile a non-existing object",
			Key:     tKey,
			Objects: nil,
			WantErr: false,
		},
	}

	testCases.Test(t, MakeFactory(ctor))
}

// assertPopulatedTarget asserts that all component attributes required in
// reconciliation tests are populated and valid.
func assertPopulatedTarget(t *testing.T, rcl v1alpha1.Reconcilable) {
	t.Helper()

	// used to generate the adapter's owner reference
	assert.NotEmpty(t, rcl.GetNamespace())
	assert.NotEmpty(t, rcl.GetName())
	assert.NotEmpty(t, rcl.GetUID())

	assert.NotEmpty(t, rcl.GetStatus().GetConditions(), "Provided component instance should have initialized conditions")

	if src, isEventSource := rcl.(targets.EventSource); isEventSource {
		assert.NotEmpty(t, src.GetEventTypes(), "Provided component instance should declare its event types")
	}
	if itrg, isIntegrationTarget := rcl.(targets.IntegrationTarget); isIntegrationTarget {
		assert.NotEmpty(t, itrg.AcceptedEventTypes(), "Provided component instance should declare its accepted event types")
	}
}

func nameKindAndResource(object runtime.Object) (string /*name*/, string /*kind*/, string /*resource*/) {
	metaObj, _ := meta.Accessor(object)
	name := metaObj.GetName()

	var kind, resource string

	switch object.(type) {
	case *servingv1.Service:
		kind = "Service"
		resource = "services"
	}

	return name, kind, resource
}

// insertObject inserts an runtime.Object into a slice at the given position.
// https://github.com/golang/go/wiki/SliceTricks#insert
func insertObject(objs []runtime.Object, obj runtime.Object, pos int) []runtime.Object {
	objs = append(objs, (runtime.Object)(nil))
	copy(objs[pos+1:], objs[pos:])

	objs[pos] = obj

	return objs
}

// insertString inserts an string into a slice at the given position.
// https://github.com/golang/go/wiki/SliceTricks#insert
func insertString(strs []string, str string, pos int) []string {
	strs = append(strs, "")
	copy(strs[pos+1:], strs[pos:])

	strs[pos] = str

	return strs
}

/* Event targets */

// Populate populates an component instance with generic attributes.
func Populate(rclCpy v1alpha1.Reconcilable) {
	rclCpy.SetNamespace(tNs)
	rclCpy.SetName(tName)
	rclCpy.SetUID(tUID)

	// *reconcilerImpl.Reconcile calls this method before any reconciliation loop. Calling it here ensures that the
	// object is initialized in the same manner, and prevents tests from wrongly reporting unexpected status updates.
	reconciler.PreProcessReconcile(context.Background(), rclCpy)

	if src, isEventSource := rclCpy.(targets.EventSource); isEventSource {
		rclCpy.GetStatusManager().ResponseAttributes = common.CreateCloudEventResponseAttributes(
			src.AsEventSource(), src.GetEventTypes())
	}
	if itrg, isIntegrationTarget := rclCpy.(targets.IntegrationTarget); isIntegrationTarget {
		rclCpy.GetStatusManager().AcceptedEventTypes = itrg.AcceptedEventTypes()
	}
}

// targetCtorWithOptions is a function that returns a component instance with
// modifications applied.
type targetCtorWithOptions func(...targetOption) v1alpha1.Reconcilable

// targetCtor creates a copy of the given component instance and returns a
// function that can be invoked to return that component instance, with the
// possibility to apply options to it.
func targetCtor(rcl v1alpha1.Reconcilable) targetCtorWithOptions {
	return func(opts ...targetOption) v1alpha1.Reconcilable {
		rclCpy := rcl.DeepCopyObject().(v1alpha1.Reconcilable)

		for _, opt := range opts {
			opt(rclCpy)
		}

		return rclCpy
	}
}

// targetOption is a functional option for a component instance.
type targetOption func(v1alpha1.Reconcilable)

// noCEAttributes sets empty CE attributes. Simulates the creation of a new
// component instance.
func noCEAttributes(rcl v1alpha1.Reconcilable) {
	rcl.GetStatusManager().AcceptedEventTypes = nil
	rcl.GetStatusManager().ResponseAttributes = nil
}

// Deployed: True
func deployed(adapter runtime.Object) targetOption {
	adapter = adapter.DeepCopyObject()
	ready(adapter)

	return propagateAdapterAvailabilityFunc(adapter)
}

// Deployed: False
func notDeployed(adapter runtime.Object) targetOption {
	adapter = adapter.DeepCopyObject()
	notReady(adapter)

	return propagateAdapterAvailabilityFunc(adapter)
}

// Deployed: Unknown with error
func unknownDeployedWithError(adapter runtime.Object) targetOption {
	var nilObj runtime.Object

	switch adapter.(type) {
	case *servingv1.Service:
		nilObj = (*servingv1.Service)(nil)
	}

	return propagateAdapterAvailabilityFunc(nilObj)
}

func propagateAdapterAvailabilityFunc(adapter runtime.Object) func(rcl v1alpha1.Reconcilable) {
	return func(rcl v1alpha1.Reconcilable) {
		switch a := adapter.(type) {
		case *servingv1.Service:
			rcl.GetStatusManager().PropagateServiceAvailability(a)

			if v1alpha1.IsMultiTenant(rcl) {
				rcl.GetStatusManager().SetRoute(routing.URLPath(rcl))
			}
		}
	}
}

// deleted marks the component instance as deleted.
func deleted(rcl v1alpha1.Reconcilable) {
	t := metav1.Unix(0, 0)
	rcl.SetDeletionTimestamp(&t)
	// ignore assertion of Finalizer in those tests because not all types
	// implement it
	rcl.SetFinalizers(nil)
}

/* Adapter */

// adapterCtorWithOptions is a function that returns a runtime object with
// modifications applied.
type adapterCtorWithOptions func(...adapterOption) runtime.Object

// adapterCtor creates a copy of the given adapter object and returns a
// function that can apply options to that object.
func adapterCtor(adapterBuilder interface{}, rcl v1alpha1.Reconcilable) adapterCtorWithOptions {
	return func(opts ...adapterOption) runtime.Object {
		var obj runtime.Object

		switch typedAdapterBuilder := adapterBuilder.(type) {
		case common.AdapterServiceBuilder:
			obj = typedAdapterBuilder.BuildAdapter(rcl)
		}

		// emulate the logic applied by the generic reconciler, which
		// automatically sets the ServiceAccount as owner of
		// multi-tenant adapters
		if v1alpha1.IsMultiTenant(rcl) {
			common.OwnByServiceAccount(obj.(metav1.Object), NewServiceAccount(rcl)())
		}

		for _, opt := range opts {
			opt(obj)
		}

		return obj
	}
}

// adapterOption is a functional option for an adapter object.
type adapterOption func(runtime.Object)

// Ready: True
func ready(object runtime.Object) {
	switch o := object.(type) {
	case *servingv1.Service:
		o.Status.SetConditions(apis.Conditions{{
			Type:   v1alpha1.ConditionReady,
			Status: corev1.ConditionTrue,
		}})
		o.Status.URL = tAdapterURI
	}
}

// Ready: False
func notReady(object runtime.Object) {
	switch o := object.(type) {
	case *servingv1.Service:
		o.Status.SetConditions(apis.Conditions{{
			Type:   v1alpha1.ConditionReady,
			Status: corev1.ConditionFalse,
		}})
	}
}

// bumpImage adds a static suffix to the Deployment's image.
func bumpImage(object runtime.Object) {
	switch o := object.(type) {
	case *servingv1.Service:
		o.Spec.Template.Spec.Containers[0].Image += "-test"
	}
}

/* RBAC */

// ServiceAccountCtorWithOptions returns a ServiceAccount constructor which accepts options.
type ServiceAccountCtorWithOptions func(...resource.ServiceAccountOption) *corev1.ServiceAccount

// NewServiceAccount returns a ServiceAccountCtorWithOptions for the given
// component instance.
func NewServiceAccount(rcl v1alpha1.Reconcilable) ServiceAccountCtorWithOptions {
	name := common.ServiceAccountName(rcl)
	labels := common.CommonObjectLabels(rcl)

	return func(opts ...resource.ServiceAccountOption) *corev1.ServiceAccount {
		sa := &corev1.ServiceAccount{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: tNs,
				Name:      name,
				Labels:    labels,
				OwnerReferences: []metav1.OwnerReference{
					*kmeta.NewControllerRef(rcl),
				},
			},
			Secrets: []corev1.ObjectReference{{
				// This is a sentinel list entry to ensure autogenerated tokens
				// are preserved during updates.
				Name: name + "-token-xxxxx",
			}},
		}
		sa.OwnerReferences[0].Controller = ptr.Bool(false)

		for _, opt := range opts {
			opt(sa)
		}

		return sa
	}
}

// NoToken ensures the ServiceAccount's secrets list doesn't contain any
// reference to auto-generated tokens.
// Useful in tests that expect the creation of a ServiceAccount, when this list
// is supposed to always be empty.
func NoToken(sa *corev1.ServiceAccount) {
	filteredSecr := sa.Secrets[:0]

	for _, secr := range sa.Secrets {
		if strings.HasPrefix(secr.Name, sa.Name+"-token-") {
			continue
		}
		filteredSecr = append(filteredSecr, secr)
	}

	sa.Secrets = filteredSecr
}

// noOwner ensures the ServiceAccount's OwnerReferences list is empty.
// Useful to cause a ServiceAccount update in tests that expect the update of a
// ServiceAccount.
func noOwner(sa *corev1.ServiceAccount) {
	sa.OwnerReferences = nil
}

// NewRoleBinding returns a RoleBinding constructor for the given ServiceAccount.
func NewRoleBinding(sa *corev1.ServiceAccount) func() *rbacv1.RoleBinding {
	return func() *rbacv1.RoleBinding {
		return &rbacv1.RoleBinding{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: tNs,
				Name:      sa.Name,
				Labels:    sa.Labels,
				OwnerReferences: []metav1.OwnerReference{
					{
						APIVersion:         "v1",
						Kind:               "ServiceAccount",
						Name:               sa.Name,
						UID:                sa.UID,
						Controller:         ptr.Bool(true),
						BlockOwnerDeletion: ptr.Bool(true),
					},
				},
			},
			RoleRef: rbacv1.RoleRef{
				APIGroup: "rbac.authorization.k8s.io",
				Kind:     "ClusterRole",
				Name:     sa.Name,
			},
			Subjects: []rbacv1.Subject{
				{
					APIGroup:  "",
					Kind:      "ServiceAccount",
					Namespace: tNs,
					Name:      sa.Name,
				},
			},
		}
	}
}

/* Events */

func createServiceAccountEvent(rcl v1alpha1.Reconcilable) string {
	return eventtesting.Eventf(corev1.EventTypeNormal, common.ReasonRBACCreate,
		"Created ServiceAccount %q due to the creation of a %s object",
		common.MTAdapterObjectName(rcl), rcl.GetGroupVersionKind().Kind)
}
func updateServiceAccountEvent(rcl v1alpha1.Reconcilable) string {
	return eventtesting.Eventf(corev1.EventTypeNormal, common.ReasonRBACUpdate,
		"Updated ServiceAccount %q due to the creation/deletion of a %s object",
		common.MTAdapterObjectName(rcl), rcl.GetGroupVersionKind().Kind)
}
func createRoleBindingEvent(rcl v1alpha1.Reconcilable) string {
	return eventtesting.Eventf(corev1.EventTypeNormal, common.ReasonRBACCreate,
		"Created RoleBinding %q due to the creation of a %s object",
		common.MTAdapterObjectName(rcl), rcl.GetGroupVersionKind().Kind)
}
func createAdapterEvent(name, kind string) string {
	return eventtesting.Eventf(corev1.EventTypeNormal, common.ReasonAdapterCreate, "Created adapter %s %q", kind, name)
}
func updateAdapterEvent(name, kind string) string {
	return eventtesting.Eventf(corev1.EventTypeNormal, common.ReasonAdapterUpdate, "Updated adapter %s %q", kind, name)
}
func failCreateAdapterEvent(name, kind, resource string) string {
	return eventtesting.Eventf(corev1.EventTypeWarning, common.ReasonFailedAdapterCreate, "Failed to create adapter %s %q: "+
		"inducing failure for create %s", kind, name, resource)
}
func failUpdateAdapterEvent(name, kind, resource string) string {
	return eventtesting.Eventf(corev1.EventTypeWarning, common.ReasonFailedAdapterUpdate, "Failed to update adapter %s %q: "+
		"inducing failure for update %s", kind, name, resource)
}

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

package testing

import (
	"context"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	clientgotesting "k8s.io/client-go/testing"

	"knative.dev/eventing/pkg/apis/eventing"
	eventingv1 "knative.dev/eventing/pkg/apis/eventing/v1"
	"knative.dev/pkg/apis"
	duckv1 "knative.dev/pkg/apis/duck/v1"
	"knative.dev/pkg/kmeta"
	"knative.dev/pkg/ptr"
	"knative.dev/pkg/reconciler"
	rt "knative.dev/pkg/reconciler/testing"
	servingv1 "knative.dev/serving/pkg/apis/serving/v1"

	"github.com/triggermesh/triggermesh/pkg/apis/sources/v1alpha1"
	"github.com/triggermesh/triggermesh/pkg/sources/reconciler/common"
	"github.com/triggermesh/triggermesh/pkg/sources/reconciler/common/skip"
	"github.com/triggermesh/triggermesh/pkg/sources/routing"
	eventtesting "github.com/triggermesh/triggermesh/pkg/sources/testing/event"
)

const (
	tNs   = "testns"
	tName = "test"
	tKey  = tNs + "/" + tName
	tUID  = types.UID("00000000-0000-0000-0000-000000000000")
)

var (
	tSinkURI = &apis.URL{
		Scheme: "http",
		Host:   "default.default.svc.example.com",
		Path:   "/",
	}
	tAdapterURI = &apis.URL{
		Scheme: "http",
		Host:   "public.example.com",
		Path:   "/",
	}
)

// Test the Reconcile() method of the controller.Reconciler implemented by
// source Reconcilers, with focus on the generic ReconcileSource logic executed
// by the generic adapter reconciler embedded in every source Reconciler.
//
// The environment for each test case is set up as follows:
//  1. MakeFactory initializes fake clients with the objects declared in the test case
//  2. MakeFactory injects those clients into a context along with fake event recorders, etc.
//  3. A Reconciler is constructed via a Ctor function using the values injected above
//  4. The Reconciler returned by MakeFactory is used to run the test case
func TestReconcileAdapter(t *testing.T, ctor Ctor, src v1alpha1.EventSource, adapterBuilder interface{}) {
	assertPopulatedSource(t, src)

	newEventSource := eventSourceCtor(src)
	newServiceAccount := NewServiceAccount(src)
	newRoleBinding := NewRoleBinding(newServiceAccount())
	newAdapter := adapterCtor(adapterBuilder, src)

	s := newEventSource()
	a := newAdapter()
	n, k, r := nameKindAndResource(a)

	// initialize a context that allows for skipping parts of the
	// reconciliation this test suite should not execute
	skipCtx := skip.EnableSkip(context.Background())

	testCases := rt.TableTest{
		// Creation/Deletion

		{
			Name: "Source object creation",
			Key:  tKey,
			Ctx:  skipCtx,
			Objects: []runtime.Object{
				newAdressable(),
				newEventSource(noCEAttributes),
			},
			WantCreates: []runtime.Object{
				newServiceAccount(noToken),
				newRoleBinding(),
				newAdapter(),
			},
			WantStatusUpdates: []clientgotesting.UpdateActionImpl{{
				Object: newEventSource(withSink, notDeployed(a)),
			}},
			WantEvents: []string{
				createServiceAccountEvent(s),
				createRoleBindingEvent(s),
				createAdapterEvent(n, k),
			},
		},
		{
			Name: "Source object deletion",
			Key:  tKey,
			Ctx:  skipCtx,
			Objects: []runtime.Object{
				newEventSource(deleted),
			},
		},

		// Lifecycle

		{
			Name: "Adapter becomes Ready",
			Key:  tKey,
			Ctx:  skipCtx,
			Objects: []runtime.Object{
				newAdressable(),
				newEventSource(withSink, notDeployed(a)),
				newServiceAccount(),
				newRoleBinding(),
				newAdapter(ready),
			},
			WantStatusUpdates: []clientgotesting.UpdateActionImpl{{
				Object: newEventSource(withSink, deployed(a)),
			}},
		},
		{
			Name: "Adapter becomes NotReady",
			Key:  tKey,
			Ctx:  skipCtx,
			Objects: []runtime.Object{
				newAdressable(),
				newEventSource(withSink, deployed(a)),
				newServiceAccount(),
				newRoleBinding(),
				newAdapter(notReady),
			},
			WantStatusUpdates: []clientgotesting.UpdateActionImpl{{
				Object: newEventSource(withSink, notDeployed(a)),
			}},
		},
		{
			Name: "Adapter is outdated",
			Key:  tKey,
			Ctx:  skipCtx,
			Objects: []runtime.Object{
				newAdressable(),
				newEventSource(withSink, deployed(a)),
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
			Ctx:  skipCtx,
			Objects: []runtime.Object{
				newAdressable(),
				newEventSource(withSink, deployed(a)),
				newServiceAccount(noOwner),
				newRoleBinding(),
				newAdapter(ready),
			},
			WantUpdates: []clientgotesting.UpdateActionImpl{{
				Object: newServiceAccount(),
			}},
			WantEvents: []string{
				updateServiceAccountEvent(s),
			},
		},

		// Errors

		{
			Name: "Sink goes missing",
			Key:  tKey,
			Ctx:  skipCtx,
			Objects: []runtime.Object{
				/* sink omitted */
				newEventSource(withSink, deployed(a)),
				newServiceAccount(),
				newRoleBinding(),
				newAdapter(ready),
			},
			WantStatusUpdates: []clientgotesting.UpdateActionImpl{{
				Object: newEventSource(withoutSink, deployed(a)),
			}},
			WantEvents: []string{
				badSinkEvent(),
			},
			WantErr: true,
		},
		{
			Name: "Fail to create adapter",
			Key:  tKey,
			Ctx:  skipCtx,
			WithReactors: []clientgotesting.ReactionFunc{
				rt.InduceFailure("create", r),
			},
			Objects: []runtime.Object{
				newAdressable(),
				newEventSource(withSink),
				newServiceAccount(),
				newRoleBinding(),
			},
			WantCreates: []runtime.Object{
				newAdapter(),
			},
			WantStatusUpdates: []clientgotesting.UpdateActionImpl{{
				Object: newEventSource(unknownDeployedWithError(a), withSink),
			}},
			WantEvents: []string{
				failCreateAdapterEvent(n, k, r),
			},
			WantErr: true,
		},
		{
			Name: "Fail to update adapter",
			Key:  tKey,
			Ctx:  skipCtx,
			WithReactors: []clientgotesting.ReactionFunc{
				rt.InduceFailure("update", r),
			},
			Objects: []runtime.Object{
				newAdressable(),
				newEventSource(withSink, deployed(a)),
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

// assertPopulatedSource asserts that all source attributes required in
// reconciliation tests are populated and valid.
func assertPopulatedSource(t *testing.T, src v1alpha1.EventSource) {
	t.Helper()

	// used to generate the adapter's owner reference
	assert.NotEmpty(t, src.GetNamespace())
	assert.NotEmpty(t, src.GetName())
	assert.NotEmpty(t, src.GetUID())

	assert.NotEmpty(t, src.GetSink().Ref, "Provided source should reference a sink")
	assert.NotEmpty(t, src.GetEventTypes(), "Provided source should declare its event types")
	assert.NotEmpty(t, src.GetStatus().GetConditions(), "Provided source should have initialized conditions")
}

func nameKindAndResource(object runtime.Object) (string /*name*/, string /*kind*/, string /*resource*/) {
	metaObj, _ := meta.Accessor(object)
	name := metaObj.GetName()

	var kind, resource string

	switch object.(type) {
	case *appsv1.Deployment:
		kind = "Deployment"
		resource = "deployments"
	case *servingv1.Service:
		kind = "Service"
		resource = "services"
	}

	return name, kind, resource
}

/* Event sources */

// Populate populates an event source with generic attributes.
func Populate(srcCpy v1alpha1.EventSource) {
	srcCpy.SetNamespace(tNs)
	srcCpy.SetName(tName)
	srcCpy.SetUID(tUID)

	addr := newAdressable()
	addrGVK := addr.GetGroupVersionKind()

	srcCpy.GetSink().Ref = &duckv1.KReference{
		APIVersion: addrGVK.GroupVersion().String(),
		Kind:       addrGVK.Kind,
		Name:       addr.GetName(),
	}

	// *reconcilerImpl.Reconcile calls this method before any reconciliation loop. Calling it here ensures that the
	// object is initialized in the same manner, and prevents tests from wrongly reporting unexpected status updates.
	reconciler.PreProcessReconcile(context.Background(), srcCpy)

	srcCpy.GetStatusManager().CloudEventAttributes = common.CreateCloudEventAttributes(
		srcCpy.AsEventSource(), srcCpy.GetEventTypes())
}

// sourceCtorWithOptions is a function that returns a source object with
// modifications applied.
type sourceCtorWithOptions func(...sourceOption) v1alpha1.EventSource

// eventSourceCtor creates a copy of the given source object and returns a
// function that can be invoked to return that source, with the possibility to
// apply options to it.
func eventSourceCtor(src v1alpha1.EventSource) sourceCtorWithOptions {
	return func(opts ...sourceOption) v1alpha1.EventSource {
		srcCpy := src.DeepCopyObject().(v1alpha1.EventSource)

		for _, opt := range opts {
			opt(srcCpy)
		}

		return srcCpy
	}
}

// sourceOption is a functional option for a source interface.
type sourceOption func(v1alpha1.EventSource)

// noCEAttributes sets empty CE attributes. Simulates the creation of a new source.
func noCEAttributes(src v1alpha1.EventSource) {
	src.GetStatusManager().CloudEventAttributes = nil
}

// Sink: True
func withSink(src v1alpha1.EventSource) {
	src.GetStatusManager().MarkSink(tSinkURI)
}

// Sink: False
func withoutSink(src v1alpha1.EventSource) {
	src.GetStatusManager().MarkNoSink()
}

// Deployed: True
func deployed(adapter runtime.Object) sourceOption {
	adapter = adapter.DeepCopyObject()
	ready(adapter)

	return propagateAdapterAvailabilityFunc(adapter)
}

// Deployed: False
func notDeployed(adapter runtime.Object) sourceOption {
	adapter = adapter.DeepCopyObject()
	notReady(adapter)

	return propagateAdapterAvailabilityFunc(adapter)
}

// Deployed: Unknown with error
func unknownDeployedWithError(adapter runtime.Object) sourceOption {
	var nilObj runtime.Object

	switch adapter.(type) {
	case *appsv1.Deployment:
		nilObj = (*appsv1.Deployment)(nil)
	case *servingv1.Service:
		nilObj = (*servingv1.Service)(nil)
	}

	return propagateAdapterAvailabilityFunc(nilObj)
}

func propagateAdapterAvailabilityFunc(adapter runtime.Object) func(src v1alpha1.EventSource) {
	return func(src v1alpha1.EventSource) {
		switch a := adapter.(type) {
		case *appsv1.Deployment:
			src.GetStatusManager().PropagateDeploymentAvailability(context.Background(), a, nil)
		case *servingv1.Service:
			src.GetStatusManager().PropagateServiceAvailability(a)

			if v1alpha1.IsMultiTenant(src) {
				src.GetStatusManager().SetRoute(routing.URLPath(src))
			}
		}
	}
}

// deleted marks the source as deleted.
func deleted(src v1alpha1.EventSource) {
	t := metav1.Unix(0, 0)
	src.SetDeletionTimestamp(&t)
	// ignore assertion of Finalizer in those tests because not all types
	// implement it
	src.SetFinalizers(nil)
}

/* Adapter */

// adapterCtorWithOptions is a function that returns a runtime object with
// modifications applied.
type adapterCtorWithOptions func(...adapterOption) runtime.Object

// adapterCtor creates a copy of the given adapter object and returns a
// function that can apply options to that object.
func adapterCtor(adapterBuilder interface{}, src v1alpha1.EventSource) adapterCtorWithOptions {
	return func(opts ...adapterOption) runtime.Object {
		var obj runtime.Object

		switch typedAdapterBuilder := adapterBuilder.(type) {
		case common.AdapterDeploymentBuilder:
			obj = typedAdapterBuilder.BuildAdapter(src, tSinkURI)
		case common.AdapterServiceBuilder:
			obj = typedAdapterBuilder.BuildAdapter(src, tSinkURI)
		}

		// emulate the logic applied by the generic reconciler, which
		// automatically sets the ServiceAccount as owner of
		// multi-tenant adapters
		if v1alpha1.IsMultiTenant(src) {
			common.OwnByServiceAccount(obj.(metav1.Object), NewServiceAccount(src)())
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
	case *appsv1.Deployment:
		o.Status = appsv1.DeploymentStatus{
			Conditions: []appsv1.DeploymentCondition{{
				Type:   appsv1.DeploymentAvailable,
				Status: corev1.ConditionTrue,
			}},
		}
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
	case *appsv1.Deployment:
		o.Status = appsv1.DeploymentStatus{
			Conditions: []appsv1.DeploymentCondition{{
				Type:   appsv1.DeploymentAvailable,
				Status: corev1.ConditionFalse,
			}},
		}
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
	case *appsv1.Deployment:
		o.Spec.Template.Spec.Containers[0].Image += "-test"
	case *servingv1.Service:
		o.Spec.Template.Spec.Containers[0].Image += "-test"
	}
}

/* Event sink */

// newAdressable returns a test Addressable to be used as a sink.
func newAdressable() *eventingv1.Broker {
	return &eventingv1.Broker{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: tNs,
			Name:      tName,
		},
		Status: eventingv1.BrokerStatus{
			Address: duckv1.Addressable{
				URL: tSinkURI,
			},
		},
	}
}

/* RBAC */

type serviceAccountCtorWithOptions func(...serviceAccountOption) *corev1.ServiceAccount

func NewServiceAccount(src kmeta.OwnerRefable) serviceAccountCtorWithOptions {
	name := common.ComponentName(src) + "-adapter"
	labels := common.CommonObjectLabels(src)

	return func(opts ...serviceAccountOption) *corev1.ServiceAccount {
		sa := &corev1.ServiceAccount{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: tNs,
				Name:      name,
				Labels:    labels,
				OwnerReferences: []metav1.OwnerReference{
					*kmeta.NewControllerRef(src),
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

// serviceAccountOption is a functional option for a ServiceAccount.
type serviceAccountOption func(*corev1.ServiceAccount)

// noToken ensures the ServiceAccount's secrets list doesn't contain any
// reference to auto-generated tokens.
// Useful in tests that expect the creation of a ServiceAccount, when this list
// is supposed to always be empty.
func noToken(sa *corev1.ServiceAccount) {
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

func createServiceAccountEvent(src v1alpha1.EventSource) string {
	return eventtesting.Eventf(corev1.EventTypeNormal, common.ReasonRBACCreate,
		"Created ServiceAccount %q due to the creation of a %s object",
		common.MTAdapterObjectName(src), src.GetGroupVersionKind().Kind)
}
func updateServiceAccountEvent(src v1alpha1.EventSource) string {
	return eventtesting.Eventf(corev1.EventTypeNormal, common.ReasonRBACUpdate,
		"Updated ServiceAccount %q due to the creation/deletion of a %s object",
		common.MTAdapterObjectName(src), src.GetGroupVersionKind().Kind)
}
func createRoleBindingEvent(src v1alpha1.EventSource) string {
	return eventtesting.Eventf(corev1.EventTypeNormal, common.ReasonRBACCreate,
		"Created RoleBinding %q due to the creation of a %s object",
		common.MTAdapterObjectName(src), src.GetGroupVersionKind().Kind)
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
func badSinkEvent() string {
	return eventtesting.Eventf(corev1.EventTypeWarning, common.ReasonBadSinkURI, "Could not resolve sink URI: "+
		"%s %q not found", eventing.BrokersResource, tName)
}

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
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	clientgotesting "k8s.io/client-go/testing"

	eventingv1 "knative.dev/eventing/pkg/apis/eventing/v1"
	"knative.dev/pkg/apis"
	duckv1 "knative.dev/pkg/apis/duck/v1"
	"knative.dev/pkg/kmeta"
	"knative.dev/pkg/ptr"
	"knative.dev/pkg/reconciler"
	rt "knative.dev/pkg/reconciler/testing"
	servingv1 "knative.dev/serving/pkg/apis/serving/v1"

	"github.com/triggermesh/triggermesh/pkg/apis/common/v1alpha1"
	"github.com/triggermesh/triggermesh/pkg/mturl"
	common "github.com/triggermesh/triggermesh/pkg/reconciler"
	"github.com/triggermesh/triggermesh/pkg/reconciler/resource"
	"github.com/triggermesh/triggermesh/pkg/reconciler/skip"
	eventtesting "github.com/triggermesh/triggermesh/pkg/testing/event"
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

// TestReconcileAdapter tests the Reconcile() method of the controller.Reconciler
// implemented by component Reconcilers, with focus on the generic ReconcileAdapter
// logic executed by the generic adapter reconciler embedded in every component Reconciler.
//
// The environment for each test case is set up as follows:
//  1. MakeFactory initializes fake clients with the objects declared in the test case
//  2. MakeFactory injects those clients into a context along with fake event recorders, etc.
//  3. A Reconciler is constructed via a Ctor function using the values injected above
//  4. The Reconciler returned by MakeFactory is used to run the test case
func TestReconcileAdapter[T kmeta.Accessor](t *testing.T,
	ctor Ctor, rcl v1alpha1.Reconcilable, ab common.AdapterBuilder[T]) {

	assertPopulatedComponentInstance(t, rcl)

	newComponentInstance := componentCtor(rcl)
	newServiceAccount := NewServiceAccount(rcl)
	newConfigWatchRoleBinding := NewConfigWatchRoleBinding(newServiceAccount())
	newMTAdapterRoleBinding := NewMTAdapterRoleBinding(newServiceAccount())
	newAdapter := mustAdapterCtor(t, ab, rcl)

	comp := newComponentInstance()
	a := newAdapter()
	n, k, r := nameKindAndResource(a)

	// initialize a context that allows for skipping parts of the
	// reconciliation that this test suite should not execute (e.g.
	// reconciliation of external resources).
	skipCtx := skip.EnableSkip(context.Background())

	testCases := rt.TableTest{
		// Creation/Deletion

		{
			Name: "Object creation",
			Key:  tKey,
			Ctx:  skipCtx,
			Objects: []runtime.Object{
				newAddressable(),
				newComponentInstance(noCEAttributes),
			},
			WantCreates: func() []runtime.Object {
				objs := []runtime.Object{
					newServiceAccount(NoToken),
					newConfigWatchRoleBinding(),
					newAdapter(),
				}
				// only multi-tenant components expect a RoleBinding
				if v1alpha1.IsMultiTenant(comp) {
					return insertObject(objs, newMTAdapterRoleBinding(), 2)
				}
				return objs
			}(),
			WantStatusUpdates: []clientgotesting.UpdateActionImpl{{
				Object: newComponentInstance(withSink, notDeployed(a)),
			}},
			WantEvents: func() []string {
				events := []string{
					createServiceAccountEvent(comp),
					createConfigWatchRoleBindingEvent(comp),
					createAdapterEvent(n, k),
				}
				// only multi-tenant components expect a RoleBinding
				if v1alpha1.IsMultiTenant(comp) {
					return insertString(events, createMTAdapterRoleBindingEvent(comp), 2)
				}
				return events
			}(),
		},
		{
			Name: "Object deletion",
			Key:  tKey,
			Ctx:  skipCtx,
			Objects: []runtime.Object{
				newComponentInstance(deleted),
			},
		},

		// Lifecycle

		{
			Name: "Adapter becomes Ready",
			Key:  tKey,
			Ctx:  skipCtx,
			Objects: []runtime.Object{
				newAddressable(),
				newComponentInstance(withSink, notDeployed(a)),
				newServiceAccount(),
				newConfigWatchRoleBinding(),
				newMTAdapterRoleBinding(),
				newAdapter(ready),
			},
			WantStatusUpdates: []clientgotesting.UpdateActionImpl{{
				Object: newComponentInstance(withSink, deployed(a)),
			}},
		},
		{
			Name: "Adapter becomes NotReady",
			Key:  tKey,
			Ctx:  skipCtx,
			Objects: []runtime.Object{
				newAddressable(),
				newComponentInstance(withSink, deployed(a)),
				newServiceAccount(),
				newConfigWatchRoleBinding(),
				newMTAdapterRoleBinding(),
				newAdapter(notReady),
			},
			WantStatusUpdates: []clientgotesting.UpdateActionImpl{{
				Object: newComponentInstance(withSink, notDeployed(a)),
			}},
		},
		{
			Name: "Adapter is outdated",
			Key:  tKey,
			Ctx:  skipCtx,
			Objects: []runtime.Object{
				newAddressable(),
				newComponentInstance(withSink, deployed(a)),
				newServiceAccount(),
				newConfigWatchRoleBinding(),
				newMTAdapterRoleBinding(),
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
				newAddressable(),
				newComponentInstance(withSink, deployed(a)),
				newServiceAccount(noOwner),
				newConfigWatchRoleBinding(),
				newMTAdapterRoleBinding(),
				newAdapter(ready),
			},
			WantUpdates: []clientgotesting.UpdateActionImpl{{
				Object: newServiceAccount(),
			}},
			WantEvents: []string{
				updateServiceAccountEvent(comp),
			},
		},
		{
			Name: "Everything is up-to-date",
			Key:  tKey,
			Ctx:  skipCtx,
			Objects: []runtime.Object{
				newAddressable(),
				newComponentInstance(withSink, deployed(a)),
				newServiceAccount(),
				newConfigWatchRoleBinding(),
				newMTAdapterRoleBinding(),
				newAdapter(ready),
			},
			WantUpdates: nil,
			WantEvents:  nil,
		},
		{
			Name: "Adapter exists under a different name",
			Key:  tKey,
			Ctx:  skipCtx,
			Objects: []runtime.Object{
				newAddressable(),
				newComponentInstance(withSink, deployed(a)),
				newServiceAccount(),
				newConfigWatchRoleBinding(),
				newMTAdapterRoleBinding(),
				newAdapter(ready, rename),
			},
			WantUpdates: nil,
			WantEvents:  nil,
		},
		{
			Name: "Switch from sink to replies",
			Key:  tKey,
			Ctx:  skipCtx,
			Objects: []runtime.Object{
				newAddressable(),
				newComponentInstance(
					deployed(a),
					withSink,        // 1. instance had the SinkProvided status condition set
					withoutSinkSpec, // 2. spec.sink got deleted by the user
				),
				newServiceAccount(),
				newConfigWatchRoleBinding(),
				newMTAdapterRoleBinding(),
				newAdapter(ready),
			},
			WantStatusUpdates: func() []clientgotesting.UpdateActionImpl {
				// only types that implement EventSender have
				// the SinkProvided status condition
				if _, isEventSender := rcl.(v1alpha1.EventSender); isEventSender {
					return []clientgotesting.UpdateActionImpl{{
						Object: newComponentInstance(
							withoutSinkSpec, // ensures conditionSet doesn't include SinkProvided
							withClearSink,   // SinkProvided status condition gets cleared
							deployed(a),     // /!\ keep last to force re-compute Ready condition
						),
					}}
				}

				return nil
			}(),
			WantUpdates: func() []clientgotesting.UpdateActionImpl {
				// only types that implement EventSender and that are
				// not multi-tenant have a populated K_SINK env var
				if _, isEventSender := rcl.(v1alpha1.EventSender); isEventSender && !v1alpha1.IsMultiTenant(rcl) {
					return []clientgotesting.UpdateActionImpl{{
						Object: newAdapter(ready, noSinkEnv),
					}}
				}

				return nil
			}(),
			WantEvents: func() []string {
				// see WantUpdates
				if _, isEventSender := rcl.(v1alpha1.EventSender); isEventSender && !v1alpha1.IsMultiTenant(rcl) {
					return []string{
						updateAdapterEvent(n, k),
					}
				}
				return nil
			}(),
		},

		// Errors

		{
			Name: "Sink goes missing",
			Key:  tKey,
			Ctx:  skipCtx,
			Objects: []runtime.Object{
				/* sink omitted */
				newComponentInstance(withSink, deployed(a)),
				newServiceAccount(),
				newConfigWatchRoleBinding(),
				newMTAdapterRoleBinding(),
				newAdapter(ready),
			},
			WantStatusUpdates: func() []clientgotesting.UpdateActionImpl {
				// only types that implement EventSender have
				// the SinkProvided status condition
				if _, isEventSender := rcl.(v1alpha1.EventSender); isEventSender {
					return []clientgotesting.UpdateActionImpl{{
						Object: newComponentInstance(withoutSink, deployed(a)),
					}}
				}
				return nil
			}(),
			WantEvents: func() []string {
				// only types that implement EventSender can
				// fail the sink resolution
				if _, isEventSender := rcl.(v1alpha1.EventSender); isEventSender {
					return []string{
						badSinkEvent(),
					}
				}
				return nil
			}(),
			WantErr: func() bool {
				// only types that implement EventSender can
				// fail the sink resolution
				_, isEventSender := rcl.(v1alpha1.EventSender)
				return isEventSender
			}(),
		},
		{
			Name: "Fail to create adapter",
			Key:  tKey,
			Ctx:  skipCtx,
			WithReactors: []clientgotesting.ReactionFunc{
				rt.InduceFailure("create", r),
			},
			Objects: []runtime.Object{
				newAddressable(),
				newComponentInstance(withSink),
				newServiceAccount(),
				newConfigWatchRoleBinding(),
				newMTAdapterRoleBinding(),
			},
			WantCreates: []runtime.Object{
				newAdapter(),
			},
			WantStatusUpdates: []clientgotesting.UpdateActionImpl{{
				Object: newComponentInstance(unknownDeployedWithError(a), withSink),
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
				newAddressable(),
				newComponentInstance(withSink, deployed(a)),
				newServiceAccount(),
				newConfigWatchRoleBinding(),
				newMTAdapterRoleBinding(),
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

// assertPopulatedComponentInstance asserts that all component attributes
// required in reconciliation tests are populated and valid.
func assertPopulatedComponentInstance(t *testing.T, rcl v1alpha1.Reconcilable) {
	t.Helper()

	// used to generate the adapter's owner reference
	assert.NotEmpty(t, rcl.GetNamespace())
	assert.NotEmpty(t, rcl.GetName())
	assert.NotEmpty(t, rcl.GetUID())

	assert.NotEmpty(t, rcl.GetStatus().GetConditions(), "Provided component instance should have initialized conditions")

	if sdr, isEventSender := rcl.(v1alpha1.EventSender); isEventSender {
		sinkIsSet := assert.Comparison(func() bool {
			return sdr.GetSink().URI != nil || sdr.GetSink().Ref != nil
		})
		assert.Condition(t, sinkIsSet, "Provided component instance should have its sink set")
	}

	if src, isEventSource := rcl.(v1alpha1.EventSource); isEventSource {
		assert.NotEmpty(t, src.GetEventTypes(), "Provided component instance should declare its event types")
	}
	if rcv, isEventReceiver := rcl.(v1alpha1.EventReceiver); isEventReceiver {
		assert.NotEmpty(t, rcv.AcceptedEventTypes(), "Provided component instance should declare its accepted event types")
	}
}

func nameKindAndResource(object metav1.Object) (string /*name*/, string /*kind*/, string /*resource*/) {
	name := object.GetName()

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

/* Component instances */

// Populate populates an component instance with generic attributes.
func Populate(rclCpy v1alpha1.Reconcilable) {
	rclCpy.SetNamespace(tNs)
	rclCpy.SetName(tName)
	rclCpy.SetUID(tUID)

	if sdr, isEventSender := rclCpy.(v1alpha1.EventSender); isEventSender {
		addr := newAddressable()
		addrGVK := addr.GetGroupVersionKind()

		sdr.GetSink().Ref = &duckv1.KReference{
			APIVersion: addrGVK.GroupVersion().String(),
			Kind:       addrGVK.Kind,
			Name:       addr.GetName(),
		}
	}

	if src, isEventSource := rclCpy.(v1alpha1.EventSource); isEventSource {
		rclCpy.GetStatusManager().CloudEventAttributes = common.CreateCloudEventAttributes(
			src.AsEventSource(), src.GetEventTypes())
	}
	if rcv, isEventReceiver := rclCpy.(v1alpha1.EventReceiver); isEventReceiver {
		rclCpy.GetStatusManager().AcceptedEventTypes = rcv.AcceptedEventTypes()
	}

	// *reconcilerImpl.Reconcile calls this method before any reconciliation loop. Calling it here ensures that the
	// object is initialized in the same manner, and prevents tests from wrongly reporting unexpected status updates.
	reconciler.PreProcessReconcile(context.Background(), rclCpy)
}

// componentCtorWithOptions is a function that returns a component instance
// with modifications applied.
type componentCtorWithOptions func(...componentOption) v1alpha1.Reconcilable

// componentCtor creates a copy of the given component instance and returns a
// function that can be invoked to return that component instance, with the
// possibility to apply options to it.
func componentCtor(rcl v1alpha1.Reconcilable) componentCtorWithOptions {
	return func(opts ...componentOption) v1alpha1.Reconcilable {
		rclCpy := rcl.DeepCopyObject().(v1alpha1.Reconcilable)

		for _, opt := range opts {
			opt(rclCpy)
		}

		return rclCpy
	}
}

// componentOption is a functional option for a component instance.
type componentOption func(v1alpha1.Reconcilable)

// noCEAttributes sets empty CE attributes. Simulates the creation of a new
// component instance.
func noCEAttributes(rcl v1alpha1.Reconcilable) {
	rcl.GetStatusManager().AcceptedEventTypes = nil
	rcl.GetStatusManager().CloudEventAttributes = nil
}

// Sink: True
func withSink(rcl v1alpha1.Reconcilable) {
	if _, isEventSender := rcl.(v1alpha1.EventSender); isEventSender {
		rcl.GetStatusManager().MarkSink(tSinkURI)
	}
}

// Sink: False
func withoutSink(rcl v1alpha1.Reconcilable) {
	if _, isEventSender := rcl.(v1alpha1.EventSender); isEventSender {
		rcl.GetStatusManager().MarkNoSink()
	}
}

// withoutSinkSpec clears the sink attributes from the existing spec.
func withoutSinkSpec(rcl v1alpha1.Reconcilable) {
	if sdr, isEventSender := rcl.(v1alpha1.EventSender); isEventSender {
		sink := sdr.GetSink()
		sink.Ref = nil
		sink.URI = nil
	}
}

// withClearSink clears the SinkProvided condition.
// In the production code, this occurs as a direct consequence of spec.sink
// being undefined.
func withClearSink(rcl v1alpha1.Reconcilable) {
	// NOTE(antoineco): This will implicitly behave differently based on whether spec.sink is optional for the given
	// component type (e.g. flow components) or required (e.g. sources).
	//  - in the former case, a nil sink causes a _removal_ of the SinkProvided condition.
	//  - in the latter case, a nil sink is an error that propagates to the Ready condition.
	//
	// We don't explicitly test for that nuance in the TableTest because, in production, OpenAPI schemas prevents
	// spec.sink from being undefined if the attribute is marked as required, which means the situation wouldn't
	// occur outside of unit tests.
	rcl.GetStatusManager().MarkSink(nil)
}

// Deployed: True
func deployed(adapter kmeta.Accessor) componentOption {
	adapter = adapter.DeepCopyObject().(kmeta.Accessor)
	ready(adapter)

	return propagateAdapterAvailabilityFunc(adapter)
}

// Deployed: False
func notDeployed(adapter kmeta.Accessor) componentOption {
	adapter = adapter.DeepCopyObject().(kmeta.Accessor)
	notReady(adapter)

	return propagateAdapterAvailabilityFunc(adapter)
}

// Deployed: Unknown with error
func unknownDeployedWithError(adapter kmeta.Accessor) componentOption {
	var nilObj kmeta.Accessor

	switch adapter.(type) {
	case *appsv1.Deployment:
		nilObj = (*appsv1.Deployment)(nil)
	case *servingv1.Service:
		nilObj = (*servingv1.Service)(nil)
	}

	return propagateAdapterAvailabilityFunc(nilObj)
}

func propagateAdapterAvailabilityFunc(adapter kmeta.Accessor) func(rcl v1alpha1.Reconcilable) {
	return func(rcl v1alpha1.Reconcilable) {
		switch a := adapter.(type) {
		case *appsv1.Deployment:
			rcl.GetStatusManager().PropagateDeploymentAvailability(context.Background(), a, nil)
		case *servingv1.Service:
			rcl.GetStatusManager().PropagateServiceAvailability(a)

			if v1alpha1.IsMultiTenant(rcl) {
				rcl.GetStatusManager().SetRoute(mturl.URLPath(rcl))
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

// adapterCtorWithOptions is a function that returns an adapter object with
// modifications applied.
type adapterCtorWithOptions func(opts ...adapterOption) (adapter kmeta.Accessor, err error)

// adapterCtor returns a function that can build an adapter object based on the
// given AdapterBuilder and Reconcilable, then apply options to that object.
func adapterCtor[T kmeta.Accessor](ab common.AdapterBuilder[T], rcl v1alpha1.Reconcilable) adapterCtorWithOptions {
	return func(opts ...adapterOption) (kmeta.Accessor, error) {
		var sinkURI *apis.URL
		if _, isEventSender := rcl.(v1alpha1.EventSender); isEventSender {
			sinkURI = tSinkURI
		}

		adapter, err := ab.BuildAdapter(rcl, sinkURI)
		if err != nil {
			return nil, fmt.Errorf("building adapter object using provided Reconcilable: %w", err)
		}

		// emulate the logic applied by the generic reconciler, which
		// automatically sets the ServiceAccount as owner of
		// multi-tenant adapters
		if v1alpha1.IsMultiTenant(rcl) {
			common.OwnByServiceAccount(adapter, NewServiceAccount(rcl)())
		}

		for _, opt := range opts {
			opt(adapter)
		}

		return adapter, nil
	}
}

// mustAdapterCtor is a wrapper around adapterCtor that fails the test in case
// of error.
func mustAdapterCtor[T kmeta.Accessor](t *testing.T,
	ab common.AdapterBuilder[T], rcl v1alpha1.Reconcilable) func(...adapterOption) kmeta.Accessor {

	return func(opts ...adapterOption) kmeta.Accessor {
		a, err := adapterCtor(ab, rcl)(opts...)
		require.NoError(t, err)
		return a
	}
}

// adapterOption is a functional option for an adapter object.
type adapterOption func(kmeta.Accessor)

// Ready: True
func ready(object kmeta.Accessor) {
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
func notReady(object kmeta.Accessor) {
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

// bumpImage adds a static suffix to the adapter's image.
func bumpImage(object kmeta.Accessor) {
	switch o := object.(type) {
	case *appsv1.Deployment:
		o.Spec.Template.Spec.Containers[0].Image += "-test"
	case *servingv1.Service:
		o.Spec.Template.Spec.Containers[0].Image += "-test"
	}
}

// rename changes the name of the adapter.
func rename(object kmeta.Accessor) {
	switch o := object.(type) {
	case *appsv1.Deployment:
		o.Name += "-oldname"
	case *servingv1.Service:
		o.Name += "-oldname"
	}
}

// noSinkEnv sets the K_SINK env var to an empty value.
// This mimics the behaviour of (pkg/reconciler).NewAdapter helpers when they
// receive an empty sink URI.
func noSinkEnv(object kmeta.Accessor) {
	var envs []corev1.EnvVar

	switch o := object.(type) {
	case *appsv1.Deployment:
		envs = o.Spec.Template.Spec.Containers[0].Env
	case *servingv1.Service:
		envs = o.Spec.Template.Spec.Containers[0].Env
	}

	for i := range envs {
		if envs[i].Name == "K_SINK" {
			envs[i].Value = ""
			return
		}
	}
}

/* Event sink */

// newAddressable returns a test Addressable to be used as a sink.
func newAddressable() *eventingv1.Broker {
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

// NewConfigWatchRoleBinding returns a config watcher RoleBinding constructor
// for the given ServiceAccount.
func NewConfigWatchRoleBinding(sa *corev1.ServiceAccount) func() *rbacv1.RoleBinding {
	return func() *rbacv1.RoleBinding {
		return &rbacv1.RoleBinding{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: tNs,
				Name:      sa.Name + "-config-watcher",
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
				Name:     "triggermesh-config-watcher",
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

// NewMTAdapterRoleBinding returns a (mt-)adapter RoleBinding constructor for
// the given ServiceAccount.
func NewMTAdapterRoleBinding(sa *corev1.ServiceAccount) func() *rbacv1.RoleBinding {
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
func createConfigWatchRoleBindingEvent(rcl v1alpha1.Reconcilable) string {
	return eventtesting.Eventf(corev1.EventTypeNormal, common.ReasonRBACCreate,
		"Created RoleBinding %q due to the creation of a %s object",
		common.MTAdapterObjectName(rcl)+"-config-watcher", rcl.GetGroupVersionKind().Kind)
}
func createMTAdapterRoleBindingEvent(rcl v1alpha1.Reconcilable) string {
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
func badSinkEvent() string {
	sinkObj := newAddressable()
	gvr, _ := meta.UnsafeGuessKindToResource(sinkObj.GetGroupVersionKind())

	return eventtesting.Eventf(corev1.EventTypeWarning, common.ReasonBadSinkURI, "Could not resolve sink URI: "+
		"failed to get object %s/%s: %s %q not found",
		sinkObj.Namespace, sinkObj.Name, gvr.GroupResource(), sinkObj.Name)
}

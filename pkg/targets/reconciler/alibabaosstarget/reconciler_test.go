/*
Copyright (c) 2021 TriggerMesh Inc.

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

package alibabaosstarget

import (
	"context"
	"testing"

	"github.com/triggermesh/triggermesh/pkg/targets/reconciler"
	resources2 "github.com/triggermesh/triggermesh/pkg/targets/reconciler/resources"
	. "github.com/triggermesh/triggermesh/pkg/targets/reconciler/testing"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	clientgotesting "k8s.io/client-go/testing"

	"knative.dev/eventing/pkg/reconciler/source"
	network "knative.dev/networking/pkg"
	"knative.dev/pkg/apis"
	duckv1 "knative.dev/pkg/apis/duck/v1"
	"knative.dev/pkg/controller"
	"knative.dev/pkg/kmeta"
	"knative.dev/pkg/logging"
	rt "knative.dev/pkg/reconciler/testing"
	"knative.dev/serving/pkg/apis/serving"
	servingv1 "knative.dev/serving/pkg/apis/serving/v1"
	fakeservinginjectionclient "knative.dev/serving/pkg/client/injection/client/fake"

	"github.com/triggermesh/triggermesh/pkg/apis/targets/v1alpha1"
	fakeinjectionclient "github.com/triggermesh/triggermesh/pkg/client/generated/injection/client/fake"
	reconcilerv1alpha1 "github.com/triggermesh/triggermesh/pkg/client/generated/injection/reconciler/targets/v1alpha1/alibabaosstarget"
)

const (
	tNs     = "testns"
	tName   = "test"
	tKey    = tNs + "/" + tName
	tUID    = types.UID("00000000-0000-0000-0000-000000000000")
	tBucket = "testbucket"

	tImg = "registry/image:tag"
)

var tGenName = kmeta.ChildName(adapterName+"-", tName)

var tAdapterURL = apis.URL{
	Scheme: "https",
	Host:   tGenName + "." + tNs + ".svc.cluster.local",
}

var (
	tEndpointURL = apis.URL{
		Scheme: "https",
		Host:   "example.com",
	}
)

var tSecretSelector = &corev1.SecretKeySelector{
	LocalObjectReference: corev1.LocalObjectReference{
		Name: "test-secret",
	},
	Key: "secret",
}

// Test the Reconcile method of the controller.Reconciler implemented by our controller.
//
// The environment for each test case is set up as follows:
//  1. MakeFactory initializes fake clients with the objects declared in the test case
//  2. MakeFactory injects those clients into a context along with fake event recorders, etc.
//  3. A Reconciler is constructed via a Ctor function using the values injected above
//  4. The Reconciler returned by MakeFactory is used to run the test case
func TestReconcile(t *testing.T) {
	testCases := rt.TableTest{
		// Creation

		{
			Name: "Target object creation",
			Key:  tKey,
			Objects: []runtime.Object{
				newSecret(),
				newEventTarget(),
			},
			WantCreates: []runtime.Object{
				newAdapterService(),
			},
			WantStatusUpdates: []clientgotesting.UpdateActionImpl{{
				Object: newEventTargetNotDeployed(),
			}},
			WantEvents: []string{
				createAdapterEvent(),
			},
		},

		// Lifecycle

		{
			Name: "Adapter becomes Ready",
			Key:  tKey,
			Objects: []runtime.Object{
				newEventTargetNotDeployed(),
				newAdapterServiceReady(),
			},
			WantStatusUpdates: []clientgotesting.UpdateActionImpl{{
				Object: newEventTargetDeployed(),
			}},
		},
		{
			Name: "Adapter becomes NotReady",
			Key:  tKey,
			Objects: []runtime.Object{
				newEventTargetDeployed(),
				newAdapterServiceNotReady(),
			},
			WantStatusUpdates: []clientgotesting.UpdateActionImpl{{
				Object: newEventTargetNotDeployed(),
			}},
		},
		{
			Name: "Adapter is outdated",
			Key:  tKey,
			Objects: []runtime.Object{
				newEventTargetDeployed(),
				setAdapterImage(
					newAdapterServiceReady(),
					tImg+":old",
				),
			},
			WantUpdates: []clientgotesting.UpdateActionImpl{{
				Object: newAdapterServiceReady(),
			}},
			WantEvents: []string{
				updateAdapterEvent(),
			},
		},

		// Errors

		{
			Name: "Fail to create adapter service",
			Key:  tKey,
			WithReactors: []clientgotesting.ReactionFunc{
				rt.InduceFailure("create", "services"),
			},
			Objects: []runtime.Object{
				newEventTarget(),
			},
			WantCreates: []runtime.Object{
				newAdapterService(),
			},
			WantStatusUpdates: []clientgotesting.UpdateActionImpl{{
				Object: newEventTargetUnknownDeployed(false),
			}},
			WantEvents: []string{
				failCreateAdapterEvent(),
			},
			WantErr: true,
		},

		{
			Name: "Fail to update adapter service",
			Key:  tKey,
			WithReactors: []clientgotesting.ReactionFunc{
				rt.InduceFailure("update", "services"),
			},
			Objects: []runtime.Object{
				newEventTargetDeployed(),
				setAdapterImage(
					newAdapterServiceReady(),
					tImg+":old",
				),
			},
			WantUpdates: []clientgotesting.UpdateActionImpl{{
				Object: newAdapterServiceReady(),
			}},
			WantStatusUpdates: []clientgotesting.UpdateActionImpl{{
				Object: newEventTargetUnknownDeployed(true),
			}},
			WantEvents: []string{
				failUpdateAdapterEvent(),
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

	testCases.Test(t, MakeFactory(reconcilerCtor))
}

// reconcilerCtor returns a Ctor for a AlibabaossTarget Reconciler.
var reconcilerCtor Ctor = func(t *testing.T, ctx context.Context, ls *Listers) controller.Reconciler {
	adapterCfg := &adapterConfig{
		Image:     tImg,
		obsConfig: &source.EmptyVarsGenerator{},
	}

	r := &Reconciler{
		adapterCfg: adapterCfg,
		ksvcr: reconciler.NewKServiceReconciler(
			fakeservinginjectionclient.Get(ctx),
			ls.GetServiceLister(),
		),
	}

	return reconcilerv1alpha1.NewReconciler(ctx, logging.FromContext(ctx),
		fakeinjectionclient.Get(ctx), ls.GetAlibabaOSSTargetLister(),
		controller.GetEventRecorder(ctx), r)
}

/* Event targets */

// newEventTarget returns a test AlibabaOSSTarget object with pre-filled attributes.
func newEventTarget() *v1alpha1.AlibabaOSSTarget {
	o := &v1alpha1.AlibabaOSSTarget{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: tNs,
			Name:      tName,
			UID:       tUID,
		},
		Spec: v1alpha1.AlibabaOSSTargetSpec{
			Endpoint: tEndpointURL.String(),
			Bucket:   tBucket,
			AccessKeyID: v1alpha1.SecretValueFromSource{
				SecretKeyRef: tSecretSelector,
			},
			AccessKeySecret: v1alpha1.SecretValueFromSource{
				SecretKeyRef: tSecretSelector,
			},
		},
	}

	o.Status.InitializeConditions()
	o.Status.AcceptedEventTypes = o.AcceptedEventTypes()
	o.Status.ResponseAttributes = reconciler.CeResponseAttributes(o)

	return o
}

// newSecret returns a test Secret object with pre-filled data.
func newSecret() *corev1.Secret {
	return &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: tNs,
			Name:      tSecretSelector.Name,
		},
		Data: map[string][]byte{
			tSecretSelector.Key: nil,
		},
	}
}

// Deployed: Unknown
func newEventTargetUnknownDeployed(adapterExists bool) *v1alpha1.AlibabaOSSTarget {
	o := newEventTarget()
	o.Status.PropagateKServiceAvailability(nil)

	// cover the case where the URL was already set because an adapter was successfully created at an earlier time,
	// but the new adapter status can't be propagated, e.g. due to an update error
	if adapterExists {
		o.Status.Address = &duckv1.Addressable{
			URL: &tAdapterURL,
		}
	}

	return o
}

// Deployed: True
func newEventTargetDeployed() *v1alpha1.AlibabaOSSTarget {
	o := newEventTarget()
	o.Status.PropagateKServiceAvailability(newAdapterServiceReady())
	return o
}

// Deployed: False
func newEventTargetNotDeployed() *v1alpha1.AlibabaOSSTarget {
	o := newEventTarget()
	o.Status.PropagateKServiceAvailability(newAdapterServiceNotReady())
	return o
}

/* Adapter service */

// newAdapterService returns a test Service object with pre-filled attributes.
func newAdapterService() *servingv1.Service {
	return &servingv1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: tNs,
			Name:      tGenName,
			Labels: labels.Set{
				resources2.AppNameLabel:      adapterName,
				resources2.AppInstanceLabel:  tName,
				resources2.AppComponentLabel: resources2.AdapterComponent,
				resources2.AppPartOfLabel:    resources2.PartOf,
				resources2.AppManagedByLabel: resources2.ManagedController,
				network.VisibilityLabelKey:   serving.VisibilityClusterLocal,
			},
			OwnerReferences: []metav1.OwnerReference{
				*kmeta.NewControllerRef(NewOwnerRefable(
					tName,
					(&v1alpha1.AlibabaOSSTarget{}).GetGroupVersionKind(),
					tUID,
				)),
			},
		},
		Spec: servingv1.ServiceSpec{
			ConfigurationSpec: servingv1.ConfigurationSpec{
				Template: servingv1.RevisionTemplateSpec{
					ObjectMeta: metav1.ObjectMeta{
						Labels: labels.Set{
							resources2.AppNameLabel:      adapterName,
							resources2.AppInstanceLabel:  tName,
							resources2.AppComponentLabel: resources2.AdapterComponent,
							resources2.AppPartOfLabel:    resources2.PartOf,
							resources2.AppManagedByLabel: resources2.ManagedController,
						},
					},
					Spec: servingv1.RevisionSpec{
						PodSpec: corev1.PodSpec{
							Containers: []corev1.Container{{
								Name:  resources2.AdapterComponent,
								Image: tImg,
								Env: []corev1.EnvVar{
									{
										Name:  resources2.EnvNamespace,
										Value: tNs,
									}, {
										Name:  resources2.EnvName,
										Value: "alibabaosstarget-" + tName,
									}, {
										Name:  "OSS_ENDPOINT",
										Value: tEndpointURL.String(),
									}, {
										Name:  "OSS_BUCKET",
										Value: tBucket,
									}, {
										Name: "OSS_ACCESS_KEY_ID",
										ValueFrom: &corev1.EnvVarSource{
											SecretKeyRef: tSecretSelector,
										},
									}, {
										Name: "OSS_ACCESS_KEY_SECRET",
										ValueFrom: &corev1.EnvVarSource{
											SecretKeyRef: tSecretSelector,
										},
									}, {
										Name: reconciler.EnvBridgeID,
									}, {
										Name: source.EnvLoggingCfg,
									}, {
										Name: source.EnvMetricsCfg,
									}, {
										Name: source.EnvTracingCfg,
									},
								},
							}},
						},
					},
				},
			},
		},
	}
}

// Ready: True
func newAdapterServiceReady() *servingv1.Service {
	svc := newAdapterService()
	svc.Status.SetConditions(apis.Conditions{{
		Type:   v1alpha1.ConditionReady,
		Status: corev1.ConditionTrue,
	}})
	svc.Status.URL = &tAdapterURL
	return svc
}

// Ready: False
func newAdapterServiceNotReady() *servingv1.Service {
	svc := newAdapterService()
	svc.Status.SetConditions(apis.Conditions{{
		Type:   v1alpha1.ConditionReady,
		Status: corev1.ConditionFalse,
	}})
	return svc
}

func setAdapterImage(o *servingv1.Service, img string) *servingv1.Service {
	o.Spec.Template.Spec.Containers[0].Image = img
	return o
}

/* Events */

// TODO(cab): make event generators public inside pkg/reconciler for
// easy reuse in tests

func createAdapterEvent() string {
	return Eventf(corev1.EventTypeNormal, "KServiceCreated", "created kservice: \"%s/%s\"",
		tNs, tGenName)
}

func updateAdapterEvent() string {
	return Eventf(corev1.EventTypeNormal, "KServiceUpdated", "updated kservice: \"%s/%s\"",
		tNs, tGenName)
}
func failCreateAdapterEvent() string {
	return Eventf(corev1.EventTypeWarning, "InternalError", "inducing failure for create services")
}

func failUpdateAdapterEvent() string {
	return Eventf(corev1.EventTypeWarning, "InternalError", "inducing failure for update services")
}

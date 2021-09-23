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

package googlesheettarget

import (
	"context"
	"fmt"
	"testing"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	clientgotesting "k8s.io/client-go/testing"

	"knative.dev/eventing/pkg/reconciler/source"
	"knative.dev/pkg/apis"
	fakek8sinjectionclient "knative.dev/pkg/client/injection/kube/client/fake"
	"knative.dev/pkg/controller"
	"knative.dev/pkg/kmeta"
	"knative.dev/pkg/logging"
	rt "knative.dev/pkg/reconciler/testing"
	servingv1 "knative.dev/serving/pkg/apis/serving/v1"
	fakeservinginjectionclient "knative.dev/serving/pkg/client/injection/client/fake"

	"github.com/triggermesh/triggermesh/pkg/apis/targets/v1alpha1"
	fakeinjectionclient "github.com/triggermesh/triggermesh/pkg/client/generated/injection/client/fake"
	reconcilerv1alpha1 "github.com/triggermesh/triggermesh/pkg/client/generated/injection/reconciler/targets/v1alpha1/googlesheettarget"
	libreconciler "github.com/triggermesh/triggermesh/pkg/targets/reconciler"
	"github.com/triggermesh/triggermesh/pkg/targets/reconciler/resources"
	. "github.com/triggermesh/triggermesh/pkg/targets/reconciler/testing"
)

const (
	tNs   = "testns"
	tName = "test"
	tKey  = tNs + "/" + tName
	tUID  = types.UID("00000000-0000-0000-0000-000000000000")

	tImg = "registry/image:tag"

	tSpreadsheetID = "0-abcd-EFG123"
	tDefaultPrefix = "prefix"
)

var tGenName = kmeta.ChildName(targetPrefix+"-", tName)

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
				Object: newEventTargetServiceNotReady(),
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
				newSecret(),
				newEventTargetServiceNotReady(),
				newAdapterServiceReady(),
			},
			WantStatusUpdates: []clientgotesting.UpdateActionImpl{{
				Object: newEventTargetServiceReady(),
			}},
		},
		{
			Name: "Adapter becomes NotReady",
			Key:  tKey,
			Objects: []runtime.Object{
				newSecret(),
				newEventTargetServiceReady(),
				newAdapterServiceNotReady(),
			},
			WantStatusUpdates: []clientgotesting.UpdateActionImpl{{
				Object: newEventTargetServiceNotReady(),
			}},
		},
		{
			Name: "Adapter is outdated",
			Key:  tKey,
			Objects: []runtime.Object{
				newSecret(),
				newEventTargetServiceReady(),
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
			Name: "Missing secret",
			Key:  tKey,
			Objects: []runtime.Object{
				newEventTarget(),
			},
			WantStatusUpdates: []clientgotesting.UpdateActionImpl{{
				Object: newEventTargetSecretsNotProvided(),
			}},
			WantEvents: []string{
				failReconcileSecretNotFound(),
			},
			WantErr: true,
		},
		{
			Name: "Fail to create adapter service",
			Key:  tKey,
			WithReactors: []clientgotesting.ReactionFunc{
				rt.InduceFailure("create", "services"),
			},
			Objects: []runtime.Object{
				newSecret(),
				newEventTarget(),
			},
			WantCreates: []runtime.Object{
				newAdapterService(),
			},
			WantStatusUpdates: []clientgotesting.UpdateActionImpl{{
				Object: newEventTargetServiceUnknown(),
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
				newSecret(),
				newEventTargetServiceReady(),
				setAdapterImage(
					newAdapterServiceReady(),
					tImg+":old",
				),
			},
			WantUpdates: []clientgotesting.UpdateActionImpl{{
				Object: newAdapterServiceReady(),
			}},
			WantStatusUpdates: []clientgotesting.UpdateActionImpl{{
				Object: newEventTargetServiceUnknown(),
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

// reconcilerCtor returns a Ctor for a GoogleSpreadsheetTarget Reconciler.
var reconcilerCtor Ctor = func(t *testing.T, ctx context.Context, ls *Listers) controller.Reconciler {
	r := &reconciler{
		TargetAdapterImage: tImg,
		ksvcr: libreconciler.NewKServiceReconciler(
			fakeservinginjectionclient.Get(ctx),
			ls.GetServiceLister(),
		),
		vg:      libreconciler.NewValueGetter(fakek8sinjectionclient.Get(ctx)),
		configs: &source.EmptyVarsGenerator{},
	}

	return reconcilerv1alpha1.NewReconciler(ctx, logging.FromContext(ctx),
		fakeinjectionclient.Get(ctx), ls.GetGoogleSheetTargetLister(),
		controller.GetEventRecorder(ctx), r)
}

/* Event targets */

// newEventTarget returns a test GoogleSpreadsheetTarget object with pre-filled attributes.
func newEventTarget() *v1alpha1.GoogleSheetTarget {
	o := &v1alpha1.GoogleSheetTarget{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: tNs,
			Name:      tName,
			UID:       tUID,
		},
		Spec: v1alpha1.GoogleSheetTargetSpec{
			GoogleServiceAccount: v1alpha1.SecretValueFromSource{
				SecretKeyRef: tSecretSelector,
			},
			ID:            tSpreadsheetID,
			DefaultPrefix: tDefaultPrefix,
		},
	}

	o.Status.InitializeConditions()
	o.Status.AcceptedEventTypes = o.AcceptedEventTypes()
	o.Status.ResponseAttributes = libreconciler.CeResponseAttributes(o)

	return o
}

// ServiceReady: Unknown, SecretsProvided: True
func newEventTargetSecretsNotProvided() *v1alpha1.GoogleSheetTarget {
	o := newEventTarget()
	o.Status.MarkNoSecrets(fmt.Errorf("secrets %q not found", tSecretSelector.Name))
	return o
}

// ServiceReady: Unknown with error, SecretsProvided: True
func newEventTargetServiceUnknown() *v1alpha1.GoogleSheetTarget {
	o := newEventTarget()
	o.Status.MarkSecrets()
	o.Status.PropagateAvailability(nil)
	return o
}

// ServiceReady: True, SecretsProvided: True
func newEventTargetServiceReady() *v1alpha1.GoogleSheetTarget {
	o := newEventTarget()
	o.Status.MarkSecrets()
	o.Status.PropagateAvailability(newAdapterServiceReady())
	return o
}

// ServiceReady: False, SecretsProvided: True
func newEventTargetServiceNotReady() *v1alpha1.GoogleSheetTarget {
	o := newEventTarget()
	o.Status.MarkSecrets()
	o.Status.PropagateAvailability(newAdapterServiceNotReady())
	return o
}

/* Adapter service */

// newAdapterService returns a test Service object with pre-filled attributes.
func newAdapterService() *servingv1.Service {
	return &servingv1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: tNs,
			Name:      tGenName,
			Labels: map[string]string{
				labelKnTargetController: targetPrefix + "-controller",
				labelKnTargetName:       tName,
			},
			OwnerReferences: []metav1.OwnerReference{
				*kmeta.NewControllerRef(NewOwnerRefable(
					tName,
					(&v1alpha1.GoogleSheetTarget{}).GetGroupVersionKind(),
					tUID,
				)),
			},
		},
		Spec: servingv1.ServiceSpec{
			ConfigurationSpec: servingv1.ConfigurationSpec{
				Template: servingv1.RevisionTemplateSpec{
					Spec: servingv1.RevisionSpec{
						PodSpec: corev1.PodSpec{
							Containers: []corev1.Container{{
								Image: tImg,
								Env: []corev1.EnvVar{
									{
										Name:  resources.EnvNamespace,
										Value: tNs,
									}, {
										Name:  resources.EnvName,
										Value: tName,
									}, {
										Name:  resources.EnvMetricsDomain,
										Value: resources.DefaultMetricsDomain,
									}, {
										Name: envCredentialsJSON,
										ValueFrom: &corev1.EnvVarSource{
											SecretKeyRef: tSecretSelector,
										},
									}, {
										Name:  envSheetID,
										Value: tSpreadsheetID,
									}, {
										Name:  envDefaultPrefix,
										Value: tDefaultPrefix,
									}, {
										Name: source.EnvLoggingCfg,
									}, {
										Name: source.EnvMetricsCfg,
									}, {
										Name: source.EnvTracingCfg,
									}, {
										// FIXME(antoineco): remove dupe. See target_adapter.go
										Name: source.EnvMetricsCfg,
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

/* Secrets */

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

/* Events */

// TODO(antoineco): make event generators public inside pkg/reconciler for
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

func failReconcileSecretNotFound() string {
	return Eventf(corev1.EventTypeWarning, "InternalError", "secrets %q not found", tSecretSelector.Name)
}

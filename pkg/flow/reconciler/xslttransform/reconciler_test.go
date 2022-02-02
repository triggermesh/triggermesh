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

package xslttransform

import (
	"context"
	"testing"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	clientgotesting "k8s.io/client-go/testing"

	"knative.dev/eventing/pkg/reconciler/source"
	network "knative.dev/networking/pkg"
	"knative.dev/pkg/apis"
	"knative.dev/pkg/controller"
	"knative.dev/pkg/kmeta"
	"knative.dev/pkg/logging"
	rt "knative.dev/pkg/reconciler/testing"
	"knative.dev/serving/pkg/apis/serving"
	servingv1 "knative.dev/serving/pkg/apis/serving/v1"
	fakeservinginjectionclient "knative.dev/serving/pkg/client/injection/client/fake"

	"github.com/triggermesh/triggermesh/pkg/apis/flow/v1alpha1"
	fakeinjectionclient "github.com/triggermesh/triggermesh/pkg/client/generated/injection/client/fake"
	reconcilerv1alpha1 "github.com/triggermesh/triggermesh/pkg/client/generated/injection/reconciler/flow/v1alpha1/xslttransform"
	libreconciler "github.com/triggermesh/triggermesh/pkg/flow/reconciler"
	"github.com/triggermesh/triggermesh/pkg/flow/reconciler/resources"
	. "github.com/triggermesh/triggermesh/pkg/flow/reconciler/testing"
)

const (
	tNs   = "testns"
	tName = "test"
	tKey  = tNs + "/" + tName
	tUID  = types.UID("00000000-0000-0000-0000-000000000000")

	tImg = "registry/image:tag"

	tBridge = "test-bridge"
)

var (
	tXSLT = `
<xsl:stylesheet version="1.0"	xmlns:xsl="http://www.w3.org/1999/XSL/Transform">
	<xsl:template match="tests">
		<output>
			<xsl:apply-templates select="test">
				<xsl:sort select="data/el1"/>
				<xsl:sort select="data/el2"/>
			</xsl:apply-templates>
		</output>
	</xsl:template>

	<xsl:template match="test">
		<item>
			<xsl:value-of select="data/el1"/>
			<xsl:value-of select="data/el2"/>
		</item>
	</xsl:template>
</xsl:stylesheet>
`
	tGenName = kmeta.ChildName(adapterName+"-", tName)
)

var tAdapterURL = apis.URL{
	Scheme: "https",
	Host:   tGenName + "." + tNs + ".svc.cluster.local",
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
			Name: "Component object creation",
			Key:  tKey,
			Objects: []runtime.Object{
				newComponent(),
			},
			WantCreates: []runtime.Object{
				newAdapterService(),
			},
			WantStatusUpdates: []clientgotesting.UpdateActionImpl{{
				Object: newComponentNotDeployed(),
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
				newComponentNotDeployed(),
				newAdapterServiceReady(),
			},
			WantStatusUpdates: []clientgotesting.UpdateActionImpl{{
				Object: newComponentDeployed(),
			}},
		},
		{
			Name: "Adapter becomes NotReady",
			Key:  tKey,
			Objects: []runtime.Object{
				newComponentDeployed(),
				newAdapterServiceNotReady(),
			},
			WantStatusUpdates: []clientgotesting.UpdateActionImpl{{
				Object: newComponentNotDeployed(),
			}},
		},
		{
			Name: "Adapter is outdated",
			Key:  tKey,
			Objects: []runtime.Object{
				newComponentDeployed(),
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

// reconcilerCtor returns a Ctor for a XSLTTransform Reconciler.
var reconcilerCtor Ctor = func(t *testing.T, ctx context.Context, ls *Listers) controller.Reconciler {
	adapterCfg := &adapterConfig{
		Image:   tImg,
		configs: &source.EmptyVarsGenerator{},
	}

	r := &reconciler{
		adapterCfg: adapterCfg,
		ksvcr: libreconciler.NewKServiceReconciler(
			fakeservinginjectionclient.Get(ctx),
			ls.GetServiceLister(),
		),
	}

	return reconcilerv1alpha1.NewReconciler(ctx, logging.FromContext(ctx),
		fakeinjectionclient.Get(ctx), ls.GetXSLTTransformLister(),
		controller.GetEventRecorder(ctx), r)
}

// newComponent returns a test XSLTTransform object with pre-filled attributes.
func newComponent() *v1alpha1.XSLTTransform {
	o := &v1alpha1.XSLTTransform{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: tNs,
			Name:      tName,
			UID:       tUID,
			Labels: labels.Set{
				libreconciler.LabelBridgeUsedByPrefix + tBridge: libreconciler.LabelValueBridgeDominant,
			},
		},
		Spec: v1alpha1.XSLTTransformSpec{
			XSLT: &v1alpha1.ValueFromField{
				Value: &tXSLT,
			},
		},
	}

	o.Status.InitializeConditions()
	return o
}

// Deployed: True
func newComponentDeployed() *v1alpha1.XSLTTransform {
	o := newComponent()
	o.Status.PropagateAvailability(newAdapterServiceReady())
	return o
}

// Deployed: False
func newComponentNotDeployed() *v1alpha1.XSLTTransform {
	o := newComponent()
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
			Labels: labels.Set{
				resources.AppNameLabel:      adapterName,
				resources.AppInstanceLabel:  tName,
				resources.AppComponentLabel: resources.AdapterComponent,
				resources.AppPartOfLabel:    resources.PartOf,
				resources.AppManagedByLabel: resources.ManagedController,
				network.VisibilityLabelKey:  serving.VisibilityClusterLocal,
			},
			OwnerReferences: []metav1.OwnerReference{
				*kmeta.NewControllerRef(NewOwnerRefable(
					tName,
					(&v1alpha1.XSLTTransform{}).GetGroupVersionKind(),
					tUID,
				)),
			},
		},
		Spec: servingv1.ServiceSpec{
			ConfigurationSpec: servingv1.ConfigurationSpec{
				Template: servingv1.RevisionTemplateSpec{
					ObjectMeta: metav1.ObjectMeta{
						Labels: labels.Set{
							resources.AppNameLabel:      adapterName,
							resources.AppInstanceLabel:  tName,
							resources.AppComponentLabel: resources.AdapterComponent,
							resources.AppPartOfLabel:    resources.PartOf,
							resources.AppManagedByLabel: resources.ManagedController,
						},
					},
					Spec: servingv1.RevisionSpec{
						PodSpec: corev1.PodSpec{
							Containers: []corev1.Container{{
								Name:  resources.AdapterComponent,
								Image: tImg,
								Env: []corev1.EnvVar{
									{
										Name:  resources.EnvNamespace,
										Value: tNs,
									}, {
										Name:  resources.EnvName,
										Value: tName,
									}, {
										Name:  envXSLT,
										Value: tXSLT,
									}, {
										Name:  libreconciler.EnvBridgeID,
										Value: tBridge,
									}, {
										Name:  envAllowXSLTOverride,
										Value: "false",
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
		Type:   v1alpha1.XSLTTransformConditionReady,
		Status: corev1.ConditionTrue,
	}})
	svc.Status.URL = &tAdapterURL
	return svc
}

// Ready: False
func newAdapterServiceNotReady() *servingv1.Service {
	svc := newAdapterService()
	svc.Status.SetConditions(apis.Conditions{{
		Type:   v1alpha1.XSLTTransformConditionReady,
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

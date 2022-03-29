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

package awssqssource

import (
	"context"
	"testing"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	clientgotesting "k8s.io/client-go/testing"

	"knative.dev/eventing/pkg/reconciler/source"
	"knative.dev/pkg/apis"
	"knative.dev/pkg/controller"
	"knative.dev/pkg/logging"
	rt "knative.dev/pkg/reconciler/testing"

	"github.com/aws/aws-sdk-go/service/iam"
	"github.com/aws/aws-sdk-go/service/sqs"

	"github.com/triggermesh/triggermesh/pkg/apis/sources/v1alpha1"
	fakeinjectionclient "github.com/triggermesh/triggermesh/pkg/client/generated/injection/client/fake"
	reconcilerv1alpha1 "github.com/triggermesh/triggermesh/pkg/client/generated/injection/reconciler/sources/v1alpha1/awssqssource"
	"github.com/triggermesh/triggermesh/pkg/sources/reconciler/common"
	"github.com/triggermesh/triggermesh/pkg/sources/reconciler/common/resource"
	. "github.com/triggermesh/triggermesh/pkg/sources/reconciler/testing"
	eventtesting "github.com/triggermesh/triggermesh/pkg/sources/testing/event"
)

// adapterCfg is used in every instance of Reconciler defined in reconciler tests.
var adapterCfg = &adapterConfig{
	Image:   "registry/image:tag",
	configs: &source.EmptyVarsGenerator{},
}

func TestReconcileSource(t *testing.T) {
	ctor := reconcilerCtor(adapterCfg)
	src := newEventSource()
	ab := adapterBuilder(adapterCfg)

	TestReconcileAdapter(t, ctor, src, ab)
}

// reconcilerCtor returns a Ctor for a AWSSQSSource Reconciler.
func reconcilerCtor(cfg *adapterConfig) Ctor {
	return func(t *testing.T, ctx context.Context, _ *rt.TableRow, ls *Listers) controller.Reconciler {
		r := &Reconciler{
			base:       NewTestDeploymentReconciler(ctx, ls),
			adapterCfg: cfg,
			srcLister:  ls.GetAWSSQSSourceLister().AWSSQSSources,
		}

		return reconcilerv1alpha1.NewReconciler(ctx, logging.FromContext(ctx),
			fakeinjectionclient.Get(ctx), ls.GetAWSSQSSourceLister(),
			controller.GetEventRecorder(ctx), r)
	}
}

// newEventSource returns a test source object with a minimal set of pre-filled attributes.
func newEventSource() *v1alpha1.AWSSQSSource {
	src := &v1alpha1.AWSSQSSource{
		Spec: v1alpha1.AWSSQSSourceSpec{
			ARN: NewARN(sqs.ServiceName, "triggermeshtest"),
		},
	}

	Populate(src)

	return src
}

// adapterBuilder returns a slim Reconciler containing only the fields accessed
// by r.BuildAdapter().
func adapterBuilder(cfg *adapterConfig) common.AdapterDeploymentBuilder {
	return &Reconciler{
		adapterCfg: cfg,
	}
}

// TestReconcileWithIAMRoleAuth contains tests specific to the SQS source.
func TestReconcileWithIAMRoleAuth(t *testing.T) {
	// We use a source which configuration is known to return
	// "WantsOwnServiceAccount() == true" in all the tests below.
	// This has an influence on the ServiceAccount that is going to be
	// synchronized for that source by the common reconciliation logic.
	s := newReconciledSource(iamRole)

	sa := common.ServiceAccountName(s)
	n := adapterBuilder(adapterCfg).BuildAdapter(s, tSinkURI).Name

	testCases := rt.TableTest{
		{
			Name: "Enable IAM role authentication",
			Key:  tKey,
			Objects: []runtime.Object{
				newReconciledSource(iamRole),
				newReconciledAdapter(),
			},
			WantCreates: []runtime.Object{
				newReconciledServiceAccount(NoToken, saName(sa), iamRoleAnnotation),
			},
			WantUpdates: []clientgotesting.UpdateActionImpl{{
				Object: newReconciledAdapter(podServiceAccount(sa)),
			}},
			WantEvents: []string{
				createServiceAccountEvent(s),
				updateAdapterEvent(n),
			},
		},
	}

	ctor := reconcilerCtor(adapterCfg)

	testCases.Test(t, MakeFactory(ctor))
}

// tNs/tName match the namespace/name set by (reconciler/testing).Populate.
const (
	tNs   = "testns"
	tName = "test"
	tKey  = tNs + "/" + tName
)

var (
	tSinkURI = &apis.URL{
		Scheme: "http",
		Host:   "default.default.svc.example.com",
		Path:   "/",
	}

	tIAMRoleARN = NewARN(iam.ServiceName, "role/test")
)

/* Source and receive adapter */

// newReconciledSource returns a test event source object that is identical to
// what ReconcileKind generates.
func newReconciledSource(opts ...sourceOption) *v1alpha1.AWSSQSSource {
	src := newEventSource()

	// assume the sink URI is resolved, so we don't have to include the
	// addressable object referenced by Sink.Ref in every test case
	src.Spec.Sink.Ref = nil
	src.Spec.Sink.URI = tSinkURI

	// assume status conditions are already set to True to ensure
	// ReconcileKind is a no-op
	status := src.GetStatusManager()
	status.MarkSink(tSinkURI)
	status.PropagateDeploymentAvailability(context.Background(), newReconciledAdapter(), nil)

	for _, opt := range opts {
		opt(src)
	}

	return src
}

// sourceOption is a functional option for an event source.
type sourceOption func(*v1alpha1.AWSSQSSource)

// iamRole enables IAM role authentication.
func iamRole(src *v1alpha1.AWSSQSSource) {
	src.Spec.Auth.EksIAMRole = &tIAMRoleARN
}

// newReconciledAdapter returns a test receive adapter object that is identical
// to what ReconcileKind generates.
func newReconciledAdapter(opts ...adapterOption) *appsv1.Deployment {
	adapter := adapterBuilder(adapterCfg).BuildAdapter(newEventSource(), tSinkURI)

	adapter.Status = appsv1.DeploymentStatus{
		Conditions: []appsv1.DeploymentCondition{{
			Type:   appsv1.DeploymentAvailable,
			Status: corev1.ConditionTrue,
		}},
	}

	for _, opt := range opts {
		opt(adapter)
	}

	return adapter
}

// adapterOption is a functional option for a Deployment object.
type adapterOption func(*appsv1.Deployment)

// podServiceAccount sets the name of the ServiceAccount used by the adapter.
func podServiceAccount(name string) adapterOption {
	return func(adapter *appsv1.Deployment) {
		adapter.Spec.Template.Spec.ServiceAccountName = name
	}
}

/* RBAC */

// newReconciledServiceAccount returns a test ServiceAccount object that is
// identical to what ReconcileKind generates.
func newReconciledServiceAccount(opts ...resource.ServiceAccountOption) *corev1.ServiceAccount {
	return NewServiceAccount(newEventSource())(opts...)
}

// iamRoleAnnotation sets the IAM Role annotation, which is expected to be
// present when IAM Role authentication is enabled.
func iamRoleAnnotation(sa *corev1.ServiceAccount) {
	metav1.SetMetaDataAnnotation(&sa.ObjectMeta, "eks.amazonaws.com/role-arn", tIAMRoleARN.String())
}

// saName overrides the object's Name.
func saName(name string) resource.ServiceAccountOption {
	return func(sa *corev1.ServiceAccount) {
		sa.Name = name
	}
}

/* Events */

func createServiceAccountEvent(src v1alpha1.Reconcilable) string {
	return eventtesting.Eventf(corev1.EventTypeNormal, common.ReasonRBACCreate,
		"Created ServiceAccount %q due to the creation of a AWSSQSSource object",
		common.ServiceAccountName(src))
}

func updateAdapterEvent(name string) string {
	return eventtesting.Eventf(corev1.EventTypeNormal, common.ReasonAdapterUpdate,
		"Updated adapter Deployment %q", name)
}

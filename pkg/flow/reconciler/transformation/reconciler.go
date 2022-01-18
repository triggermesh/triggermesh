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

package transformation

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"

	"go.uber.org/zap"
	corev1 "k8s.io/api/core/v1"
	apierrs "k8s.io/apimachinery/pkg/api/errors"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"knative.dev/pkg/apis"
	duckv1 "knative.dev/pkg/apis/duck/v1"
	"knative.dev/pkg/logging"
	"knative.dev/pkg/network"
	"knative.dev/pkg/reconciler"
	"knative.dev/pkg/resolver"
	"knative.dev/pkg/tracker"
	servingv1 "knative.dev/serving/pkg/apis/serving/v1"
	servingv1client "knative.dev/serving/pkg/client/clientset/versioned"
	servingv1listers "knative.dev/serving/pkg/client/listers/serving/v1"

	"github.com/triggermesh/triggermesh/pkg/apis/flow/v1alpha1"
	reconcilerv1alpha1 "github.com/triggermesh/triggermesh/pkg/client/generated/injection/reconciler/flow/v1alpha1/transformation"
	"github.com/triggermesh/triggermesh/pkg/flow/reconciler/transformation/resources"
)

const (
	envSink               = "K_SINK"
	envTransformationCtx  = "TRANSFORMATION_CONTEXT"
	envTransformationData = "TRANSFORMATION_DATA"
)

// newReconciledNormal makes a new reconciler event with event type Normal, and
// reason AddressableServiceReconciled.
func newReconciledNormal(namespace, name string) reconciler.Event {
	return reconciler.NewEvent(corev1.EventTypeNormal, "TransformationReconciled", "Transformation reconciled: \"%s/%s\"", namespace, name)
}

// Reconciler implements addressableservicereconciler.Interface for
// Transformation resources.
type Reconciler struct {
	// Tracker builds an index of what resources are watching other resources
	// so that we can immediately react to changes tracked resources.
	Tracker tracker.Interface

	// Listers index properties about resources
	knServiceLister  servingv1listers.ServiceLister
	servingClientSet servingv1client.Interface

	sinkResolver *resolver.URIResolver

	transformerImage string
}

// Check that our Reconciler implements Interface
var _ reconcilerv1alpha1.Interface = (*Reconciler)(nil)

// ReconcileKind implements Interface.ReconcileKind.
func (r *Reconciler) ReconcileKind(ctx context.Context, trn *v1alpha1.Transformation) reconciler.Event {
	logger := logging.FromContext(ctx)

	if err := r.Tracker.TrackReference(tracker.Reference{
		APIVersion: "serving.knative.dev/v1",
		Kind:       "Service",
		Name:       trn.Name,
		Namespace:  trn.Namespace,
	}, trn); err != nil {
		logger.Errorf("Error tracking service %s: %v", trn.Name, err)
		return err
	}

	// Reconcile Transformation Adapter
	ksvc, err := r.reconcileKnService(ctx, trn)
	if err != nil {
		logger.Error("Error reconciling Kn Service", zap.Error(err))
		trn.Status.MarkServiceUnavailable(trn.Name)
		return err
	}

	if ksvc.IsReady() {
		trn.Status.Address = &duckv1.Addressable{
			URL: &apis.URL{
				Scheme: "http",
				Host:   network.GetServiceHostname(trn.Name, trn.Namespace),
			},
		}
		trn.Status.MarkServiceAvailable()
	}
	trn.Status.CloudEventAttributes = r.createCloudEventAttributes(&trn.Spec)

	logger.Debug("Transformation reconciled")
	return newReconciledNormal(trn.Namespace, trn.Name)
}

func (r *Reconciler) reconcileKnService(ctx context.Context, trn *v1alpha1.Transformation) (*servingv1.Service, error) {
	logger := logging.FromContext(ctx)

	var sink string
	if trn.Spec.Sink != (duckv1.Destination{}) {
		uri, err := r.resolveDestination(ctx, trn)
		if err != nil {
			return nil, fmt.Errorf("cannot resolve Sink destination: %w", err)
		}
		trn.Status.SinkURI = uri
		sink = uri.String()
	}

	trnContext, err := json.Marshal(trn.Spec.Context)
	if err != nil {
		return nil, fmt.Errorf("cannot marshal context transformation spec: %w", err)
	}

	trnData, err := json.Marshal(trn.Spec.Data)
	if err != nil {
		return nil, fmt.Errorf("cannot marshal data transformation spec: %w", err)
	}

	expectedKsvc := resources.NewKnService(trn.Namespace, trn.Name,
		resources.Image(r.transformerImage),
		resources.EnvVar(envTransformationCtx, string(trnContext)),
		resources.EnvVar(envTransformationData, string(trnData)),
		resources.EnvVar(envSink, sink),
		resources.KsvcLabelVisibilityClusterLocal(),
		resources.Owner(trn),
	)

	ksvc, err := r.knServiceLister.Services(trn.Namespace).Get(trn.Name)
	if apierrs.IsNotFound(err) {
		logger.Infof("Creating Kn Service %q", trn.Name)
		return r.servingClientSet.ServingV1().Services(trn.Namespace).Create(ctx, expectedKsvc, v1.CreateOptions{})
	} else if err != nil {
		return nil, err
	}

	if !reflect.DeepEqual(ksvc.Spec.ConfigurationSpec.Template.Spec,
		expectedKsvc.Spec.ConfigurationSpec.Template.Spec) {
		ksvc.Spec = expectedKsvc.Spec
		return r.servingClientSet.ServingV1().Services(trn.Namespace).Update(ctx, ksvc, v1.UpdateOptions{})
	}
	return ksvc, nil
}

func (r *Reconciler) createCloudEventAttributes(ts *v1alpha1.TransformationSpec) []duckv1.CloudEventAttributes {
	ceAttributes := make([]duckv1.CloudEventAttributes, 0)
	for _, item := range ts.Context {
		if item.Operation == "add" {
			attribute := duckv1.CloudEventAttributes{}
			for _, path := range item.Paths {
				switch path.Key {
				case "type":
					attribute.Type = path.Value
				case "source":
					attribute.Source = path.Value
				}
			}
			if attribute.Source != "" || attribute.Type != "" {
				ceAttributes = append(ceAttributes, attribute)
			}
			break
		}
	}
	return ceAttributes
}

func (r *Reconciler) resolveDestination(ctx context.Context, trn *v1alpha1.Transformation) (*apis.URL, error) {
	dest := trn.Spec.Sink.DeepCopy()
	if dest.Ref != nil {
		if dest.Ref.Namespace == "" {
			dest.Ref.Namespace = trn.GetNamespace()
		}
	}
	return r.sinkResolver.URIFromDestinationV1(ctx, *dest, trn)
}

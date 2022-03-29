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

package function

import (
	"context"
	"fmt"
	"reflect"
	"sort"
	"strconv"
	"strings"

	"go.uber.org/zap"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/util/rand"
	"k8s.io/client-go/kubernetes"
	corev1listers "k8s.io/client-go/listers/core/v1"

	"knative.dev/pkg/apis"
	duckv1 "knative.dev/pkg/apis/duck/v1"
	"knative.dev/pkg/logging"
	"knative.dev/pkg/reconciler"
	"knative.dev/pkg/resolver"
	"knative.dev/pkg/tracker"
	servingv1 "knative.dev/serving/pkg/apis/serving/v1"
	servingv1client "knative.dev/serving/pkg/client/clientset/versioned"
	servingv1listers "knative.dev/serving/pkg/client/listers/serving/v1"

	"github.com/triggermesh/triggermesh/pkg/apis/extensions/v1alpha1"
	reconcilerv1alpha1 "github.com/triggermesh/triggermesh/pkg/client/generated/injection/reconciler/extensions/v1alpha1/function"
	"github.com/triggermesh/triggermesh/pkg/function/resources"
	"github.com/triggermesh/triggermesh/pkg/function/semantic"
)

const (
	adapterName         = "function"
	klrEntrypoint       = "/opt/aws-custom-runtime"
	functionNameLabel   = "extensions.triggermesh.io/function"
	ceDefaultTypePrefix = "io.triggermesh.function."

	metricsPrometheusPortKsvc uint16 = 9092
)

// Reconciler implements addressableservicereconciler.Interface for
// AddressableService resources.
type Reconciler struct {
	// Tracker builds an index of what resources are watching other resources
	// so that we can immediately react to changes tracked resources.
	Tracker tracker.Interface

	// Listers index properties about resources
	knServiceLister    servingv1listers.ServiceLister
	knServingClientSet servingv1client.Interface

	cmLister      corev1listers.ConfigMapLister
	coreClientSet kubernetes.Interface

	sinkResolver *resolver.URIResolver

	// runtime names and container URIs
	runtimes map[string]string
}

// Check that our Reconciler implements Interface
var _ reconcilerv1alpha1.Interface = (*Reconciler)(nil)

// ReconcileKind implements Interface.ReconcileKind.
func (r *Reconciler) ReconcileKind(ctx context.Context, o *v1alpha1.Function) reconciler.Event {
	logger := logging.FromContext(ctx)

	// Reconcile configmap
	cm, err := r.reconcileConfigmap(ctx, o)
	if err != nil {
		logger.Error("Error reconciling Configmap", zap.Error(err))
		o.Status.MarkConfigmapUnavailable(o.Name)
		return err
	}
	if err := r.Tracker.TrackReference(tracker.Reference{
		APIVersion: "v1",
		Kind:       "ConfigMap",
		Name:       cm.Name,
		Namespace:  cm.Namespace,
	}, o); err != nil {
		logger.Errorf("Error tracking configmap %s: %v", o.Name, err)
		return err
	}
	o.Status.MarkConfigmapAvailable()

	sink, err := r.resolveSink(ctx, o)
	if err != nil {
		o.Status.MarkSinkUnavailable()
		return fmt.Errorf("resolving sink URI: %w", err)
	}
	o.Status.SinkURI = sink
	o.Status.MarkSinkAvailable()

	// Reconcile adapter service
	ksvc, err := r.reconcileKnService(ctx, o, cm, sink)
	if err != nil {
		logger.Error("Error reconciling Kn Service", zap.Error(err))
		o.Status.MarkServiceUnavailable(o.Name)
		return err
	}

	if err := r.Tracker.TrackReference(tracker.Reference{
		APIVersion: "serving.knative.dev/v1",
		Kind:       "Service",
		Name:       ksvc.Name,
		Namespace:  ksvc.Namespace,
	}, o); err != nil {
		logger.Errorf("Error tracking Kn service %s: %v", o.Name, err)
		return err
	}

	if !ksvc.IsReady() {
		o.Status.MarkServiceUnavailable(o.Name)
		return nil
	}

	o.Status.Address = &duckv1.Addressable{
		URL: &apis.URL{
			Scheme: ksvc.Status.URL.Scheme,
			Host:   ksvc.Status.URL.Host,
		},
	}
	o.Status.MarkServiceAvailable()

	if o.Spec.CloudEventOverrides != nil {
		// in status we can set default attributes only;
		// there is no reliable way to get dynamic CE attributes from function source code
		o.Status.CloudEventAttributes = r.statusAttributes(o.Spec.CloudEventOverrides.Extensions)
	}

	logger.Debug("Function reconciled")
	return nil
}

func (r *Reconciler) resolveSink(ctx context.Context, f *v1alpha1.Function) (*apis.URL, error) {
	if f.Spec.Sink != nil {
		dest := f.Spec.Sink.DeepCopy()
		if dest.Ref != nil {
			if dest.Ref.Namespace == "" {
				dest.Ref.Namespace = f.GetNamespace()
			}
		}
		return r.sinkResolver.URIFromDestinationV1(ctx, *dest, f)
	}
	return &apis.URL{}, nil
}

func (r *Reconciler) reconcileConfigmap(ctx context.Context, f *v1alpha1.Function) (*corev1.ConfigMap, error) {
	logger := logging.FromContext(ctx)

	expectedCm := resources.NewConfigmap(f.Name+"-"+rand.String(6), f.Namespace,
		resources.CmOwner(f),
		resources.CmLabel(map[string]string{functionNameLabel: f.Name}),
		resources.CmData(f.Spec.Code),
	)

	cmList, err := r.coreClientSet.CoreV1().ConfigMaps(f.Namespace).List(ctx, v1.ListOptions{
		LabelSelector: fmt.Sprintf("%s=%s", functionNameLabel, f.Name),
	})
	if err != nil {
		return nil, err
	}
	if len(cmList.Items) == 0 {
		logger.Infof("Creating configmap %q", f.Name)
		return r.coreClientSet.CoreV1().ConfigMaps(f.Namespace).Create(ctx, expectedCm, v1.CreateOptions{})
	}
	actualCm := &cmList.Items[0]

	if !reflect.DeepEqual(actualCm.Data, expectedCm.Data) {
		actualCm.Data = expectedCm.Data
		return r.coreClientSet.CoreV1().ConfigMaps(f.Namespace).Update(ctx, actualCm, v1.UpdateOptions{})
	}

	return actualCm, nil
}

func (r *Reconciler) reconcileKnService(ctx context.Context, f *v1alpha1.Function, cm *corev1.ConfigMap, sink *apis.URL) (*servingv1.Service, error) {
	logger := logging.FromContext(ctx)

	image, err := r.lookupRuntimeImage(f.Spec.Runtime)
	if err != nil {
		return nil, err
	}

	filename := fmt.Sprintf("source.%s", fileExtension(f.Spec.Runtime))
	handler := fmt.Sprintf("source.%s", f.Spec.Entrypoint)

	overrides := map[string]string{
		// Default values for required attributes
		"type":   ceDefaultTypePrefix + f.Spec.Runtime,
		"source": filename,
	}

	if f.Spec.CloudEventOverrides != nil {
		for k, v := range f.Spec.CloudEventOverrides.Extensions {
			overrides[k] = v
		}
	}

	var responseMode string
	if f.Spec.ResponseIsEvent {
		responseMode = "event"
	}

	genericLabels := resources.MakeGenericLabels(adapterName, f.Name)
	ksvcLabels := resources.PropagateCommonLabels(f, genericLabels)
	podLabels := resources.PropagateCommonLabels(f, genericLabels)

	ksvcLabels[functionNameLabel] = f.Name

	expectedKsvc := resources.NewKnService(fmt.Sprintf("function-%s-%s", f.Name, rand.String(6)), f.Namespace,
		resources.KnSvcImage(image),
		resources.KnSvcMountCm(cm.Name, filename),
		resources.KnSvcEntrypoint(klrEntrypoint),
		resources.KnSvcEnvVar(resources.EnvName, f.Name),
		resources.KnSvcEnvVar(resources.EnvNamespace, f.Namespace),
		resources.KnSvcEnvVar(resources.EnvComponent, adapterName),
		resources.KnSvcEnvVar(resources.EnvMetricsPrometheusPort, strconv.FormatUint(uint64(metricsPrometheusPortKsvc), 10)),
		resources.KnSvcEnvVar(eventStoreEnv, f.Spec.EventStore.URI),
		resources.KnSvcEnvVar("K_SINK", sink.String()),
		resources.KnSvcEnvVar("_HANDLER", handler),
		resources.KnSvcEnvVar("RESPONSE_FORMAT", "CLOUDEVENTS"),
		resources.KnSvcEnvVar("CE_FUNCTION_RESPONSE_MODE", responseMode),
		resources.KnSvcEnvVars(sortedEnvVarsWithPrefix("CE_OVERRIDES_", overrides)...),
		resources.KnSvcAnnotation("extensions.triggermesh.io/codeVersion", cm.ResourceVersion),
		resources.KnSvcVisibility(f.Spec.Public),
		resources.KnSvcLabel(ksvcLabels),
		resources.KnSvcPodLabels(podLabels),
		resources.KnSvcOwner(f),
	)

	ksvcList, err := r.knServiceLister.Services(f.Namespace).List(labels.SelectorFromSet(labels.Set{functionNameLabel: f.Name}))
	if err != nil {
		return nil, err
	}
	if len(ksvcList) == 0 {
		logger.Infof("Creating Kn Service %q", f.Name)
		return r.knServingClientSet.ServingV1().Services(f.Namespace).Create(ctx, expectedKsvc, v1.CreateOptions{})
	}
	actualKsvc := ksvcList[0]
	expectedKsvc.Name = actualKsvc.Name

	if semantic.Semantic.DeepEqual(expectedKsvc, actualKsvc) {
		return actualKsvc, nil
	}
	actualKsvc.Spec = expectedKsvc.Spec
	actualKsvc.Labels = expectedKsvc.Labels
	return r.knServingClientSet.ServingV1().Services(f.Namespace).Update(ctx, actualKsvc, v1.UpdateOptions{})
}

func (r *Reconciler) statusAttributes(attributes map[string]string) []duckv1.CloudEventAttributes {
	res := duckv1.CloudEventAttributes{}

	if typ, ok := attributes["type"]; ok {
		res.Type = typ
	}

	if source, ok := attributes["source"]; ok {
		res.Source = source
	}

	return []duckv1.CloudEventAttributes{res}
}

func (r *Reconciler) lookupRuntimeImage(runtime string) (string, error) {
	rn := strings.ToLower(runtime)

	for name, uri := range r.runtimes {
		name = strings.ToLower(name)
		if strings.Contains(name, rn) {
			return uri, nil
		}
	}
	return "", fmt.Errorf("runtime %q not registered in the controller env", runtime)
}

// Lambda runtimes require file extensions to match the language,
// i.e. source file for Python runtime must have ".py" prefix, JavaScript - ".js", etc.
// It would be more correct to declare these extensions explicitly,
// along with the runtime container URIs, but since we manage the
// available runtimes list, this also works.
func fileExtension(runtime string) string {
	runtime = strings.ToLower(runtime)
	switch {
	case strings.Contains(runtime, "python"):
		return "py"
	case strings.Contains(runtime, "node") ||
		strings.Contains(runtime, "js"):
		return "js"
	case strings.Contains(runtime, "ruby"):
		return "rb"
	case strings.Contains(runtime, "sh"):
		return "sh"
	}
	return "txt"
}

// Env variables from extensions override map are sorted alphabetically before
// passing to container env to prevent reconciliation loop when map keys are randomized.
func sortedEnvVarsWithPrefix(prefix string, overrides map[string]string) []corev1.EnvVar {
	keys := make([]string, 0, len(overrides))
	for key := range overrides {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	res := make([]corev1.EnvVar, 0, len(keys))
	for _, key := range keys {
		res = append(res, corev1.EnvVar{
			Name:  strings.ToUpper(prefix + key),
			Value: overrides[key],
		})
	}
	return res
}

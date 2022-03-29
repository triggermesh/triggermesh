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

package dataweavetransformation

import (
	corev1 "k8s.io/api/core/v1"

	"knative.dev/eventing/pkg/reconciler/source"
	"knative.dev/pkg/apis"
	"knative.dev/pkg/kmeta"
	servingv1 "knative.dev/serving/pkg/apis/serving/v1"

	"github.com/triggermesh/triggermesh/pkg/apis/flow/v1alpha1"
	libreconciler "github.com/triggermesh/triggermesh/pkg/flow/reconciler"
	"github.com/triggermesh/triggermesh/pkg/flow/reconciler/resources"
)

const (
	adapterName = "dataweavetransformation"

	envDWSPELL             = "DATAWEAVETRANSFORMATION_DWSPELL"
	envIncomingContentType = "DATAWEAVETRANSFORMATION_INCOMING_CONTENT_TYPE"
	envOutputContentType   = "DATAWEAVETRANSFORMATION_OUTPUT_CONTENT_TYPE"
	envSink                = "K_SINK"
)

// adapterConfig contains properties used to configure the component's adapter.
// Public fields are automatically populated by envconfig.
type adapterConfig struct {
	// Configuration accessor for logging/metrics/tracing
	configs source.ConfigAccessor
	// Container image
	Image string `envconfig:"DATAWEAVETRANSFORMATION_IMAGE" default:"gcr.io/triggermesh/dataweavetransformation-adapter"`
}

// makeAdapterKService generates the adapter knative service structure.
func makeAdapterKService(o *v1alpha1.DataWeaveTransformation, cfg *adapterConfig, sink *apis.URL) (*servingv1.Service, error) {
	envApp, err := makeAppEnv(o, sink)
	if err != nil {
		return nil, err
	}

	genericLabels := libreconciler.MakeGenericLabels(adapterName, o.Name)
	ksvcLabels := libreconciler.PropagateCommonLabels(o, genericLabels)
	podLabels := libreconciler.PropagateCommonLabels(o, genericLabels)
	name := kmeta.ChildName(adapterName+"-", o.Name)
	envSvc := libreconciler.MakeServiceEnv(o.Name, o.Namespace)
	envObs := libreconciler.MakeObsEnv(cfg.configs)
	envs := append(envSvc, append(envApp, envObs...)...)

	return resources.MakeKService(o.Namespace, name, cfg.Image,
		resources.KsvcLabels(ksvcLabels),
		resources.KsvcLabelVisibilityClusterLocal,
		resources.KsvcPodLabels(podLabels),
		resources.KsvcOwner(o),
		resources.KsvcPodEnvVars(envs)), nil
}

func makeAppEnv(o *v1alpha1.DataWeaveTransformation, sink *apis.URL) ([]corev1.EnvVar, error) {
	env := []corev1.EnvVar{
		*o.Spec.DwSpell.ToEnvironmentVariable(envDWSPELL),
		{
			Name:  libreconciler.EnvBridgeID,
			Value: libreconciler.GetStatefulBridgeID(o),
		},
	}

	env = append(env, corev1.EnvVar{
		Name:  envIncomingContentType,
		Value: o.Spec.IncomingContentType,
	})

	env = append(env, corev1.EnvVar{
		Name:  envOutputContentType,
		Value: o.Spec.OutputContentType,
	})

	if sink != nil {
		env = append(env, corev1.EnvVar{
			Name:  envSink,
			Value: sink.String(),
		})
	}
	return env, nil
}

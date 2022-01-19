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

package synchronizer

import (
	"strconv"

	corev1 "k8s.io/api/core/v1"

	"knative.dev/eventing/pkg/reconciler/source"
	"knative.dev/pkg/kmeta"
	servingv1 "knative.dev/serving/pkg/apis/serving/v1"

	"github.com/triggermesh/triggermesh/pkg/apis/flow/v1alpha1"
	libreconciler "github.com/triggermesh/triggermesh/pkg/targets/reconciler"
	"github.com/triggermesh/triggermesh/pkg/targets/reconciler/resources"
)

const adapterName = "synchronizer"

// adapterConfig contains properties used to configure the synchronizer's adapter.
// Public fields are automatically populated by envconfig.
type adapterConfig struct {
	// Configuration accessor for logging/metrics/tracing
	obsConfig source.ConfigAccessor
	// Container image
	Image string `envconfig:"SYNCHRONIZER_ADAPTER_IMAGE" default:"gcr.io/triggermesh/synchronizer-adapter"`
}

// makeAdapterKService generates (but does not insert into K8s) the Synchronizer Adapter KService.
func makeAdapterKService(o *v1alpha1.Synchronizer, cfg *adapterConfig) *servingv1.Service {
	name := kmeta.ChildName(adapterName+"-", o.Name)
	lbl := libreconciler.MakeAdapterLabels(adapterName, o.Name)
	podLabels := libreconciler.MakeAdapterLabels(adapterName, o.Name)
	envSvc := libreconciler.MakeServiceEnv(name, o.Namespace)
	envApp := makeAppEnv(o)
	envObs := libreconciler.MakeObsEnv(cfg.obsConfig)
	envs := append(envSvc, envApp...)
	envs = append(envs, envObs...)

	return resources.MakeKService(o.Namespace, name, cfg.Image,
		resources.KsvcLabels(lbl),
		resources.KsvcLabelVisibilityClusterLocal,
		resources.KsvcOwner(o),
		resources.KsvcPodLabels(podLabels),
		resources.KsvcPodEnvVars(envs),
	)
}

func makeAppEnv(o *v1alpha1.Synchronizer) []corev1.EnvVar {
	env := []corev1.EnvVar{
		{
			Name:  libreconciler.EnvBridgeID,
			Value: libreconciler.GetStatefulBridgeID(o),
		},
		{
			Name:  "CORRELATION_KEY",
			Value: o.Spec.CorrelationKey.Attribute,
		},
		{
			Name:  "K_SINK",
			Value: o.Status.SinkURI.String(),
		},
	}

	if o.Spec.CorrelationKey.Length != 0 {
		env = append(env, corev1.EnvVar{
			Name:  "CORRELATION_KEY_LENGTH",
			Value: strconv.Itoa(o.Spec.CorrelationKey.Length),
		})
	}

	if o.Spec.Response.Timeout != nil {
		env = append(env, corev1.EnvVar{
			Name:  "RESPONSE_WAIT_TIMEOUT",
			Value: o.Spec.Response.Timeout.String(),
		})
	}

	return env
}

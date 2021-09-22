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

package tektontarget

import (
	libreconciler "github.com/triggermesh/triggermesh/pkg/targets/reconciler"
	"github.com/triggermesh/triggermesh/pkg/targets/reconciler/resources"
	corev1 "k8s.io/api/core/v1"

	"github.com/triggermesh/triggermesh/pkg/apis/targets/v1alpha1"
	"knative.dev/eventing/pkg/reconciler/source"
	"knative.dev/pkg/kmeta"
	servingv1 "knative.dev/serving/pkg/apis/serving/v1"
)

const adapterName = "tektontarget"

// adapterConfig contains properties used to configure the target's adapter.
// Public fields are automatically populated by envconfig.
type adapterConfig struct {
	// Configuration accessor for logging/metrics/tracing
	obsConfig source.ConfigAccessor
	// Container image
	Image           string `envconfig:"TEKTON_ADAPTER_IMAGE" default:"gcr.io/triggermesh/tekton-target-adapter"`
	ReapingInterval string `envconfig:"TEKTON_REAPING_INTERVAL" default:"5m"`
}

// TargetAdapterArgs are the arguments needed to create a Target Adapter.
// Every field is required.
type TargetAdapterArgs struct {
	Image  string
	Target *v1alpha1.TektonTarget
}

// makeTargetAdapterKService generates (but does not insert into K8s) the Target Adapter KService.
func makeTargetAdapterKService(target *v1alpha1.TektonTarget, cfg *adapterConfig) *servingv1.Service {
	name := kmeta.ChildName(adapterName+"-", target.Name)
	lbl := libreconciler.MakeAdapterLabels(adapterName, target.Name)
	podLabels := libreconciler.MakeAdapterLabels(adapterName, target.Name)
	envSvc := libreconciler.MakeServiceEnv(name, target.Namespace)
	envObs := libreconciler.MakeObsEnv(cfg.obsConfig)
	envApp := makeAppEnv(&target.Spec)
	envs := append(append(envSvc, envObs...), envApp...)

	return resources.MakeKService(target.Namespace, name, cfg.Image,
		resources.KsvcLabels(lbl),
		resources.KsvcLabelVisibilityClusterLocal,
		resources.KsvcServiceAccount(tektontargetServiceAccountName),
		resources.KsvcOwner(target),
		resources.KsvcPodLabels(podLabels),
		resources.KsvcPodEnvVars(envs),
	)
}

// makeAppEnv provide adapter specific configuration
func makeAppEnv(spec *v1alpha1.TektonTargetSpec) []corev1.EnvVar {
	envVar := make([]corev1.EnvVar, 0)

	if spec != nil && spec.ReapPolicy != nil {
		if spec.ReapPolicy.ReapSuccessAge != nil {
			envVar = append(envVar, corev1.EnvVar{
				Name:  "TEKTON_REAP_SUCCESS_AGE",
				Value: *spec.ReapPolicy.ReapSuccessAge,
			})
		}

		if spec.ReapPolicy.ReapSuccessAge != nil {
			envVar = append(envVar, corev1.EnvVar{
				Name:  "TEKTON_REAP_FAIL_AGE",
				Value: *spec.ReapPolicy.ReapFailAge,
			})
		}
	}

	return envVar
}

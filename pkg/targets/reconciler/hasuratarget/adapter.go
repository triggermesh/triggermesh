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

package hasuratarget

import (
	"encoding/json"

	corev1 "k8s.io/api/core/v1"
	"knative.dev/eventing/pkg/reconciler/source"
	"knative.dev/pkg/kmeta"
	servingv1 "knative.dev/serving/pkg/apis/serving/v1"

	"github.com/triggermesh/triggermesh/pkg/apis/targets/v1alpha1"
	libreconciler "github.com/triggermesh/triggermesh/pkg/targets/reconciler"
	"github.com/triggermesh/triggermesh/pkg/targets/reconciler/resources"
)

const adapterName = "hasuratarget"

// adapterConfig contains properties used to configure the target's adapter.
// Public fields are automatically populated by envconfig.
type adapterConfig struct {
	// Configuration accessor for logging/metrics/tracing
	configs source.ConfigAccessor
	// Container image
	Image string `default:"gcr.io/triggermesh/hasura-target-adapter"`
}

// makeAdapterKnService returns a Knative Service object for the target's adapter.
func makeAdapterKnService(o *v1alpha1.HasuraTarget, cfg *adapterConfig) (*servingv1.Service, error) {
	svcLabels := libreconciler.MakeAdapterLabels(adapterName, o.Name)
	name := kmeta.ChildName(adapterName+"-", o.Name)
	podLabels := libreconciler.MakeAdapterLabels(adapterName, o.Name)
	env := libreconciler.MakeObsEnv(cfg.configs)
	envSvc := libreconciler.MakeServiceEnv(o.Name, o.Namespace)
	envApp, err := makeAppEnv(&o.Spec)
	if err != nil {
		return nil, err
	}

	env = append(env, append(envApp, envSvc...)...)

	return resources.MakeKService(o.Namespace, name, cfg.Image,
		resources.KsvcLabels(svcLabels),
		resources.KsvcLabelVisibilityClusterLocal,
		resources.KsvcOwner(o),
		resources.KsvcPodLabels(podLabels),
		resources.KsvcPodEnvVars(env),
	), nil
}

// makeAppEnv return target-specific environment
func makeAppEnv(spec *v1alpha1.HasuraTargetSpec) ([]corev1.EnvVar, error) {
	envs := []corev1.EnvVar{{
		Name:  "HASURA_ENDPOINT",
		Value: spec.Endpoint,
	}}

	if spec.DefaultRole != nil {
		envs = append(envs, corev1.EnvVar{
			Name:  "HASURA_DEFAULT_ROLE",
			Value: *spec.DefaultRole,
		})
	}

	if spec.AdminToken != nil {
		envs = append(envs, corev1.EnvVar{
			Name: "HASURA_ADMIN_TOKEN",
			ValueFrom: &corev1.EnvVarSource{
				SecretKeyRef: spec.AdminToken.SecretKeyRef,
			},
		})
	}

	if spec.JwtToken != nil {
		envs = append(envs, corev1.EnvVar{
			Name: "HASURA_JWT_TOKEN",
			ValueFrom: &corev1.EnvVarSource{
				SecretKeyRef: spec.JwtToken.SecretKeyRef,
			},
		})
	}

	if spec.Queries != nil {
		encodedQuery, err := json.Marshal(*spec.Queries)
		if err != nil {
			return nil, err
		}
		envs = append(envs, corev1.EnvVar{
			Name:  "HASURA_QUERIES",
			Value: string(encodedQuery),
		})
	}

	return envs, nil
}

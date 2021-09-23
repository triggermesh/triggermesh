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

package awscomprehendtarget

import (
	corev1 "k8s.io/api/core/v1"

	"knative.dev/eventing/pkg/reconciler/source"
	"knative.dev/pkg/kmeta"
	servingv1 "knative.dev/serving/pkg/apis/serving/v1"

	"github.com/triggermesh/triggermesh/pkg/apis/targets/v1alpha1"
	awscomprehendv1alpha1 "github.com/triggermesh/triggermesh/pkg/apis/targets/v1alpha1"
	libreconciler "github.com/triggermesh/triggermesh/pkg/targets/reconciler"
	"github.com/triggermesh/triggermesh/pkg/targets/reconciler/resources"
)

const (
	adapterName = "awscomprehendtarget"

	envAWSAPIKey           = "AWS_ACCESS_KEY_ID"
	envAWSSecretKey        = "AWS_SECRET_ACCESS_KEY"
	envRegion              = "COMPREHEND_REGION"
	envLanguage            = "COMPREHEND_LANGUAGE"
	envEventsPayloadPolicy = "EVENTS_PAYLOAD_POLICY"
)

// adapterConfig contains properties used to configure the target's adapter.
// Public fields are automatically populated by envconfig.
type adapterConfig struct {
	// Configuration accessor for logging/metrics/tracing
	obsConfig source.ConfigAccessor
	// Container image
	Image string `envconfig:"AWS_COMPREHEND_ADAPTER_IMAGE"  default:"gcr.io/triggermesh/awscomprehend-target-adapter"`
}

// makeTargetAdapterKService generates (but does not insert into K8s) the Target Adapter KService.
func makeTargetAdapterKService(target *awscomprehendv1alpha1.AWSComprehendTarget, cfg *adapterConfig) *servingv1.Service {
	name := kmeta.ChildName(adapterName+"-", target.Name)
	lbl := libreconciler.MakeAdapterLabels(adapterName, target.Name)
	podLabels := libreconciler.MakeAdapterLabels(adapterName, target.Name)
	envSvc := libreconciler.MakeServiceEnv(name, target.Namespace)
	envApp := makeAppEnv(&target.Spec)
	envObs := libreconciler.MakeObsEnv(cfg.obsConfig)
	envs := append(envSvc, envApp...)
	envs = append(envs, envObs...)

	return resources.MakeKService(target.Namespace, name, cfg.Image,
		resources.KsvcLabels(lbl),
		resources.KsvcLabelVisibilityClusterLocal,
		resources.KsvcOwner(target),
		resources.KsvcPodLabels(podLabels),
		resources.KsvcPodEnvVars(envs),
	)
}

func makeAppEnv(spec *v1alpha1.AWSComprehendTargetSpec) []corev1.EnvVar {
	env := []corev1.EnvVar{
		{
			Name:  envRegion,
			Value: spec.Region,
		}, {
			Name:  envLanguage,
			Value: spec.Language,
		}, {
			Name: envAWSAPIKey,
			ValueFrom: &corev1.EnvVarSource{
				SecretKeyRef: spec.AWSApiKey.SecretKeyRef,
			},
		}, {
			Name: envAWSSecretKey,
			ValueFrom: &corev1.EnvVarSource{
				SecretKeyRef: spec.AWSApiSecret.SecretKeyRef,
			},
		},
	}

	if spec.EventOptions != nil && spec.EventOptions.PayloadPolicy != nil {
		env = append(env, corev1.EnvVar{
			Name:  envEventsPayloadPolicy,
			Value: string(*spec.EventOptions.PayloadPolicy),
		})
	}

	return env

}

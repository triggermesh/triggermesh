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

package alibabaosstarget

import (
	corev1 "k8s.io/api/core/v1"

	"knative.dev/eventing/pkg/reconciler/source"
	"knative.dev/pkg/kmeta"
	servingv1 "knative.dev/serving/pkg/apis/serving/v1"

	"github.com/triggermesh/triggermesh/pkg/apis/targets/v1alpha1"
	libreconciler "github.com/triggermesh/triggermesh/pkg/targets/reconciler"
	"github.com/triggermesh/triggermesh/pkg/targets/reconciler/resources"
)

const (
	adapterName = "alibabaosstarget"

	envEndpoint            = "OSS_ENDPOINT"
	envBucket              = "OSS_BUCKET"
	envAccessKeyID         = "OSS_ACCESS_KEY_ID"
	envKeySecret           = "OSS_ACCESS_KEY_SECRET"
	envEventsPayloadPolicy = "EVENTS_PAYLOAD_POLICY"
)

// adapterConfig contains properties used to configure the target's adapter.
// Public fields are automatically populated by envconfig.
type adapterConfig struct {
	// Configuration accessor for logging/metrics/tracing
	obsConfig source.ConfigAccessor
	// Container image
	Image string `envconfig:"ALIBABAOSS_ADAPTER_IMAGE" default:"gcr.io/triggermesh-private/alibabaosstarget-adapter"`
}

// makeTargetAdapterKService generates (but does not insert into K8s) the Target Adapter KService.
func makeTargetAdapterKService(target *v1alpha1.AlibabaOSSTarget, cfg *adapterConfig) *servingv1.Service {
	name := kmeta.ChildName(adapterName+"-", target.Name)
	ksvcLabels := libreconciler.MakeAdapterLabels(adapterName, target)
	podLabels := libreconciler.MakeAdapterLabels(adapterName, target)
	envSvc := libreconciler.MakeServiceEnv(name, target.Namespace)
	envApp := makeAppEnv(target)
	envObs := libreconciler.MakeObsEnv(cfg.obsConfig)
	envs := append(envSvc, envApp...)
	envs = append(envs, envObs...)

	return resources.MakeKService(target.Namespace, name, cfg.Image,
		resources.KsvcLabels(ksvcLabels),
		resources.KsvcLabelVisibilityClusterLocal,
		resources.KsvcOwner(target),
		resources.KsvcPodLabels(podLabels),
		resources.KsvcPodEnvVars(envs),
	)
}

func makeAppEnv(o *v1alpha1.AlibabaOSSTarget) []corev1.EnvVar {
	env := []corev1.EnvVar{
		{
			Name:  envEndpoint,
			Value: o.Spec.Endpoint,
		}, {
			Name:  envBucket,
			Value: o.Spec.Bucket,
		}, {
			Name: envAccessKeyID,
			ValueFrom: &corev1.EnvVarSource{
				SecretKeyRef: o.Spec.AccessKeyID.SecretKeyRef,
			},
		}, {
			Name: envKeySecret,
			ValueFrom: &corev1.EnvVarSource{
				SecretKeyRef: o.Spec.AccessKeySecret.SecretKeyRef,
			},
		},
		{
			Name:  libreconciler.EnvBridgeID,
			Value: libreconciler.GetStatefulBridgeID(o),
		},
	}

	if o.Spec.EventOptions != nil && o.Spec.EventOptions.PayloadPolicy != nil {
		env = append(env, corev1.EnvVar{
			Name:  envEventsPayloadPolicy,
			Value: string(*o.Spec.EventOptions.PayloadPolicy),
		})
	}

	return env

}

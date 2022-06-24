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

package azuresentineltarget

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
	adapterName = "azuresentineltarget"

	envEventsPayloadPolicy = "EVENTS_PAYLOAD_POLICY"

	envSubscriptionID = "AZURE_SUBSCRIPTION_ID"
	envResourceGroup  = "AZURE_RESOURCE_GROUP"
	envWorkspace      = "AZURE_WORKSPACE"
	envClientID       = "AZURE_CLIENT_ID"
	envClientSecret   = "AZURE_CLIENT_SECRET"
	envTenantID       = "AZURE_TENANT_ID"
)

// adapterConfig contains properties used to configure the target's adapter.
// Public fields are automatically populated by envconfig.
type adapterConfig struct {
	// Configuration accessor for logging/metrics/tracing
	obsConfig source.ConfigAccessor
	// Container image
	Image string `envconfig:"AZURESENTINELTARGET_ADAPTER_IMAGE" default:"gcr.io/triggermesh/azuresentineltarget-adapter"`
}

// makeTargetAdapterKService generates (but does not insert into K8s) the Target Adapter KService.
func makeTargetAdapterKService(target *v1alpha1.AzureSentinelTarget, cfg *adapterConfig) *servingv1.Service {
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

func makeAppEnv(o *v1alpha1.AzureSentinelTarget) []corev1.EnvVar {
	env := []corev1.EnvVar{
		{
			Name:  libreconciler.EnvBridgeID,
			Value: libreconciler.GetStatefulBridgeID(o),
		},
		{
			Name:  envSubscriptionID,
			Value: o.Spec.SubscriptionID,
		},
		{
			Name:  envResourceGroup,
			Value: o.Spec.ResourceGroup,
		},
		{
			Name:  envWorkspace,
			Value: o.Spec.Workspace,
		},
		{
			Name: envClientSecret,
			ValueFrom: &corev1.EnvVarSource{
				SecretKeyRef: o.Spec.ClientSecret.SecretKeyRef,
			},
		},
		{
			Name:  envClientID,
			Value: o.Spec.ClientID,
		},
		{
			Name:  envTenantID,
			Value: o.Spec.TenantID,
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

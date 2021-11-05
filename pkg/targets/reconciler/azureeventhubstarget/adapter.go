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

package azureeventhubstarget

import (
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"

	"knative.dev/eventing/pkg/reconciler/source"
	"knative.dev/pkg/apis"

	"github.com/triggermesh/triggermesh/pkg/apis/targets/v1alpha1"
	"github.com/triggermesh/triggermesh/pkg/sources/reconciler/common"
	"github.com/triggermesh/triggermesh/pkg/sources/reconciler/common/resource"
)

const (
	adapterName = "azureeventhubstarget"

	envHubKeyName      = "EVENTHUB_KEY_NAME"
	envHubNamespace    = "EVENTHUB_NAMESPACE"
	envHubName         = "EVENTHUB_NAME"
	envHubKeyValue     = "EVENTHUB_KEY_VALUE"
	envHubConnStr      = "EVENTHUB_CONNECTION_STRING"
	envHubResourceID   = "EVENTHUB_RESOURCE_ID"
	envAADTenantID     = "AZURE_TENANT_ID"
	envAADClientID     = "AZURE_CLIENT_ID"
	envAADClientSecret = "AZURE_CLIENT_SECRET"

	envEventsPayloadPolicy = "EVENTS_PAYLOAD_POLICY"
)

// BuildAdapter implements common.AdapterDeploymentBuilder.
func (r *Reconciler) BuildAdapter(src v1alpha1.AzureEventHubsTarget, sinkURI *apis.URL) *appsv1.Deployment {

	var hubEnvs []corev1.EnvVar

	if sasAuth := src.Spec.Auth.SASToken; sasAuth != nil {
		hubEnvs = common.MaybeAppendValueFromEnvVar(hubEnvs, envHubKeyName, sasAuth.KeyName)
		hubEnvs = common.MaybeAppendValueFromEnvVar(hubEnvs, envHubKeyValue, sasAuth.KeyValue)
		hubEnvs = common.MaybeAppendValueFromEnvVar(hubEnvs, envHubConnStr, sasAuth.ConnectionString)
	}

	if spAuth := src.Spec.Auth.ServicePrincipal; spAuth != nil {
		hubEnvs = common.MaybeAppendValueFromEnvVar(hubEnvs, envAADTenantID, spAuth.TenantID)
		hubEnvs = common.MaybeAppendValueFromEnvVar(hubEnvs, envAADClientID, spAuth.ClientID)
		hubEnvs = common.MaybeAppendValueFromEnvVar(hubEnvs, envAADClientSecret, spAuth.ClientSecret)
	}

	return common.NewAdapterDeployment(src, sinkURI,
		resource.Image(r.adapterCfg.Image),

		resource.EnvVar(common.EnvHubResourceID, src.Spec.EventHubID.String()),
		resource.EnvVar(common.EnvHubNamespace, src.Spec.EventHubID.Namespace),
		resource.EnvVar(common.EnvHubName, src.Spec.EventHubID.EventHub),
		resource.EnvVars(hubEnvs...),
		resource.EnvVars(r.adapterCfg.configs.ToEnvVars()...),
	)
}

// adapterConfig contains properties used to configure the target's adapter.
// Public fields are automatically populated by envconfig.
type adapterConfig struct {
	// Configuration accessor for logging/metrics/tracing
	obsConfig source.ConfigAccessor
	// Container image
	Image string `envconfig:"AZURE_EVENTHUBS_ADAPTER_IMAGE" default:"gcr.io/triggermesh-private/azureeventhubstarget-adapter"`
}

// // makeTargetAdapterKService generates (but does not insert into K8s) the Target Adapter KService.
// func makeTargetAdapterKService(target *v1alpha1.AzureEventHubsTarget, cfg *adapterConfig) *servingv1.Service {
// 	name := kmeta.ChildName(adapterName+"-", target.Name)
// 	lbl := libreconciler.MakeAdapterLabels(adapterName, target.Name)
// 	podLabels := libreconciler.MakeAdapterLabels(adapterName, target.Name)
// 	envSvc := libreconciler.MakeServiceEnv(name, target.Namespace)
// 	envApp := makeAppEnv(target)
// 	envObs := libreconciler.MakeObsEnv(cfg.obsConfig)
// 	envs := append(envSvc, envApp...)
// 	envs = append(envs, envObs...)

// 	return resources.MakeKService(target.Namespace, name, cfg.Image,
// 		resources.KsvcLabels(lbl),
// 		resources.KsvcLabelVisibilityClusterLocal,
// 		resources.KsvcOwner(target),
// 		resources.KsvcPodLabels(podLabels),
// 		resources.KsvcPodEnvVars(envs),
// 	)
// }

// func makeAppEnv(o *v1alpha1.AzureEventHubsTarget) []corev1.EnvVar {
// 	env := []corev1.EnvVar{
// 		{
// 			Name:  libreconciler.EnvBridgeID,
// 			Value: libreconciler.GetStatefulBridgeID(o),
// 		},
// 	}

// 	if o.Spec.EventOptions != nil && o.Spec.EventOptions.PayloadPolicy != nil {
// 		env = append(env, corev1.EnvVar{
// 			Name:  envEventsPayloadPolicy,
// 			Value: string(*o.Spec.EventOptions.PayloadPolicy),
// 		})
// 	}

// 	if sasAuth := o.Spec.Auth.SASToken; sasAuth != nil {
// 		env = append(env, corev1.EnvVar{
// 			Name:  envHubKeyName,
// 			Value:  o.Spec.Auth.SASToken.s ,
// 		})

// 		env = append(env, corev1.EnvVar{
// 			Name:  envHubKeyValue,
// 			Value: sasAuth.KeyValue,
// 		})

// 		env = append(env, corev1.EnvVar{
// 			Name:  envHubConnStr,
// 			Value: sasAuth.ConnectionString,
// 		})
// 	}

// 	return env

// }

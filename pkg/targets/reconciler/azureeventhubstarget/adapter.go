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
	corev1 "k8s.io/api/core/v1"
	"knative.dev/eventing/pkg/reconciler/source"
	"knative.dev/pkg/kmeta"
	servingv1 "knative.dev/serving/pkg/apis/serving/v1"

	"github.com/triggermesh/triggermesh/pkg/apis/targets/v1alpha1"
	libreconciler "github.com/triggermesh/triggermesh/pkg/targets/reconciler"
	"github.com/triggermesh/triggermesh/pkg/targets/reconciler/common"
	"github.com/triggermesh/triggermesh/pkg/targets/reconciler/resources"
)

const (
	adapterName = "azureeventhubstarget"

	envDiscardCECtx    = "DISCARD_CE_CONTEXT"
	envHubKeyName      = "EVENTHUB_KEY_NAME"
	envHubNamespace    = "EVENTHUB_NAMESPACE"
	envHubName         = "EVENTHUB_NAME"
	envHubKeyValue     = "EVENTHUB_KEY_VALUE"
	envHubConnStr      = "EVENTHUB_CONNECTION_STRING"
	envAADTenantID     = "AZURE_TENANT_ID"
	envAADClientID     = "AZURE_CLIENT_ID"
	envAADClientSecret = "AZURE_CLIENT_SECRET"

	envEventsPayloadPolicy = "EVENTS_PAYLOAD_POLICY"
)

// adapterConfig contains properties used to configure the target's adapter.
// Public fields are automatically populated by envconfig.
type adapterConfig struct {
	// Configuration accessor for logging/metrics/tracing
	obsConfig source.ConfigAccessor
	// Container image
	Image string `envconfig:"AZURE_EVENTHUBS_ADAPTER_IMAGE" default:"gcr.io/triggermesh/azureeventhubstarget-adapter"`
}

// makeTargetAdapterKService generates (but does not insert into K8s) the Target Adapter KService.
func makeTargetAdapterKService(target *v1alpha1.AzureEventHubsTarget, cfg *adapterConfig) *servingv1.Service {
	name := kmeta.ChildName(adapterName+"-", target.Name)
	lbl := libreconciler.MakeAdapterLabels(adapterName, target.Name)
	podLabels := libreconciler.MakeAdapterLabels(adapterName, target.Name)
	envSvc := libreconciler.MakeServiceEnv(name, target.Namespace)
	envObs := libreconciler.MakeObsEnv(cfg.obsConfig)
	envs := []corev1.EnvVar{}
	envs = append(envs, envSvc...)
	envs = append(envs, envObs...)

	if sasAuth := target.Spec.Auth.SASToken; sasAuth != nil {
		envs = common.MaybeAppendValueFromEnvVar(envs, envHubKeyName, sasAuth.KeyName)
		envs = common.MaybeAppendValueFromEnvVar(envs, envHubKeyValue, sasAuth.KeyValue)
		envs = common.MaybeAppendValueFromEnvVar(envs, envHubConnStr, sasAuth.ConnectionString)
	}

	if spAuth := target.Spec.Auth.ServicePrincipal; spAuth != nil {
		envs = common.MaybeAppendValueFromEnvVar(envs, envAADTenantID, spAuth.TenantID)
		envs = common.MaybeAppendValueFromEnvVar(envs, envAADClientID, spAuth.ClientID)
		envs = common.MaybeAppendValueFromEnvVar(envs, envAADClientSecret, spAuth.ClientSecret)
	}

	if target.Spec.EventOptions != nil && target.Spec.EventOptions.PayloadPolicy != nil {
		envs = append(envs, corev1.EnvVar{
			Name:  envEventsPayloadPolicy,
			Value: string(*target.Spec.EventOptions.PayloadPolicy),
		})
	}

	if target.Spec.DiscardCEContext {
		envs = append(envs, corev1.EnvVar{Name: envDiscardCECtx, Value: "true"})
	}

	return resources.MakeKService(target.Namespace, name, cfg.Image,
		resources.KsvcLabels(lbl),
		resources.KsvcLabelVisibilityClusterLocal,
		resources.KsvcOwner(target),
		resources.KsvcPodLabels(podLabels),
		resources.KsvcPodEnvVars(envs),
		resources.EnvVar(envHubNamespace, target.Spec.EventHubID.Namespace),
		resources.EnvVar(envHubName, target.Spec.EventHubID.EventHub),
	)
}

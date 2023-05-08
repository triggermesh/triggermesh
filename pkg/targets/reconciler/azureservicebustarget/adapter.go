/*
Copyright 2023 TriggerMesh Inc.

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

package azureservicebustarget

import (
	"strconv"

	corev1 "k8s.io/api/core/v1"

	"knative.dev/eventing/pkg/reconciler/source"
	"knative.dev/pkg/apis"
	servingv1 "knative.dev/serving/pkg/apis/serving/v1"

	commonv1alpha1 "github.com/triggermesh/triggermesh/pkg/apis/common/v1alpha1"
	"github.com/triggermesh/triggermesh/pkg/apis/targets/v1alpha1"
	common "github.com/triggermesh/triggermesh/pkg/reconciler"
	"github.com/triggermesh/triggermesh/pkg/reconciler/resource"
)

const (
	envDiscardCECtx        = "DISCARD_CE_CONTEXT"
	envEventsPayloadPolicy = "EVENTS_PAYLOAD_POLICY"
)

// adapterConfig contains properties used to configure the target's adapter.
// Public fields are automatically populated by envconfig.
type adapterConfig struct {
	// Configuration accessor for logging/metrics/tracing
	obsConfig source.ConfigAccessor
	// Container image
	Image string `default:"gcr.io/triggermesh/azureeservicebustarget-adapter"`
}

// Verify that Reconciler implements common.AdapterBuilder.
var _ common.AdapterBuilder[*servingv1.Service] = (*Reconciler)(nil)

// BuildAdapter implements common.AdapterBuilder.
func (r *Reconciler) BuildAdapter(trg commonv1alpha1.Reconcilable, _ *apis.URL) (*servingv1.Service, error) {
	typedTrg := trg.(*v1alpha1.AzureServiceBusTarget)

	return common.NewAdapterKnService(trg, nil,
		resource.Image(r.adapterCfg.Image),

		resource.EnvVars(MakeAppEnv(typedTrg)...),
		resource.EnvVars(r.adapterCfg.obsConfig.ToEnvVars()...),
	), nil
}

// MakeAppEnv extracts environment variables from the object.
// Exported to be used in external tools for local test environments.
func MakeAppEnv(o *v1alpha1.AzureServiceBusTarget) []corev1.EnvVar {
	envs := []corev1.EnvVar{
		{
			Name:  envDiscardCECtx,
			Value: strconv.FormatBool(o.Spec.DiscardCEContext),
		},
	}

	var resourceID string
	if o.Spec.TopicID != nil {
		resourceID = o.Spec.TopicID.String()
	} else if o.Spec.QueueID != nil {
		resourceID = o.Spec.QueueID.String()
	}

	var webSocketsEnable bool
	if wss := o.Spec.WebSocketsEnable; wss != nil {
		webSocketsEnable = *wss
	}

	if sasAuth := o.Spec.Auth.SASToken; sasAuth != nil {
		envs = common.MaybeAppendValueFromEnvVar(envs, common.EnvServiceBusKeyName, sasAuth.KeyName)
		envs = common.MaybeAppendValueFromEnvVar(envs, common.EnvServiceBusKeyValue, sasAuth.KeyValue)
		envs = common.MaybeAppendValueFromEnvVar(envs, common.EnvServiceBusConnStr, sasAuth.ConnectionString)
	}

	if spAuth := o.Spec.Auth.ServicePrincipal; spAuth != nil {
		envs = common.MaybeAppendValueFromEnvVar(envs, common.EnvAADTenantID, spAuth.TenantID)
		envs = common.MaybeAppendValueFromEnvVar(envs, common.EnvAADClientID, spAuth.ClientID)
		envs = common.MaybeAppendValueFromEnvVar(envs, common.EnvAADClientSecret, spAuth.ClientSecret)
	}

	if o.Spec.EventOptions != nil && o.Spec.EventOptions.PayloadPolicy != nil {
		envs = append(envs, corev1.EnvVar{
			Name:  envEventsPayloadPolicy,
			Value: string(*o.Spec.EventOptions.PayloadPolicy),
		})
	}

	return append(envs, corev1.EnvVar{
		Name:  common.EnvServiceBusEntityResourceID,
		Value: resourceID,
	}, corev1.EnvVar{
		Name:  common.EnvServiceBusWebSocketsEnable,
		Value: strconv.FormatBool(webSocketsEnable),
	})
}

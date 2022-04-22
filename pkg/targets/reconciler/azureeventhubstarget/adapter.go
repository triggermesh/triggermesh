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

package azureeventhubstarget

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
	Image string `default:"gcr.io/triggermesh/azureeventhubstarget-adapter"`
}

// Verify that Reconciler implements common.AdapterServiceBuilder.
var _ common.AdapterBuilder[*servingv1.Service] = (*Reconciler)(nil)

// BuildAdapter implements common.AdapterServiceBuilder.
func (r *Reconciler) BuildAdapter(trg commonv1alpha1.Reconcilable, _ *apis.URL) *servingv1.Service {
	typedTrg := trg.(*v1alpha1.AzureEventHubsTarget)

	var envs []corev1.EnvVar

	if sasAuth := typedTrg.Spec.Auth.SASToken; sasAuth != nil {
		envs = common.MaybeAppendValueFromEnvVar(envs, common.EnvHubKeyName, sasAuth.KeyName)
		envs = common.MaybeAppendValueFromEnvVar(envs, common.EnvHubKeyValue, sasAuth.KeyValue)
		envs = common.MaybeAppendValueFromEnvVar(envs, common.EnvHubConnStr, sasAuth.ConnectionString)
	}

	if spAuth := typedTrg.Spec.Auth.ServicePrincipal; spAuth != nil {
		envs = common.MaybeAppendValueFromEnvVar(envs, common.EnvAADTenantID, spAuth.TenantID)
		envs = common.MaybeAppendValueFromEnvVar(envs, common.EnvAADClientID, spAuth.ClientID)
		envs = common.MaybeAppendValueFromEnvVar(envs, common.EnvAADClientSecret, spAuth.ClientSecret)
	}

	if typedTrg.Spec.EventOptions != nil && typedTrg.Spec.EventOptions.PayloadPolicy != nil {
		envs = append(envs, corev1.EnvVar{
			Name:  envEventsPayloadPolicy,
			Value: string(*typedTrg.Spec.EventOptions.PayloadPolicy),
		})
	}

	return common.NewAdapterKnService(trg, nil,
		resource.Image(r.adapterCfg.Image),
		resource.EnvVar(common.EnvHubNamespace, typedTrg.Spec.EventHubID.Namespace),
		resource.EnvVar(common.EnvHubName, typedTrg.Spec.EventHubID.EventHub),
		resource.EnvVar(envDiscardCECtx, strconv.FormatBool(typedTrg.Spec.DiscardCEContext)),
		resource.EnvVars(envs...),
		resource.EnvVars(r.adapterCfg.obsConfig.ToEnvVars()...),
	)
}

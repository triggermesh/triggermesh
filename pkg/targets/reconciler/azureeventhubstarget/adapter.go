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
	"fmt"
	"strconv"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/labels"

	"knative.dev/eventing/pkg/reconciler/source"
	"knative.dev/pkg/kmeta"
	servingv1 "knative.dev/serving/pkg/apis/serving/v1"

	"github.com/triggermesh/triggermesh/pkg/apis/targets/v1alpha1"
	"github.com/triggermesh/triggermesh/pkg/targets/reconciler/common"
	"github.com/triggermesh/triggermesh/pkg/targets/reconciler/common/resource"
)

const (
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
	Image string `default:"gcr.io/triggermesh/azureeventhubstarget-adapter"`
}

// Verify that Reconciler implements common.AdapterServiceBuilder.
var _ common.AdapterServiceBuilder = (*Reconciler)(nil)

// BuildAdapter implements common.AdapterServiceBuilder.
func (r *Reconciler) BuildAdapter(trg v1alpha1.Reconcilable) *servingv1.Service {
	typedTrg := trg.(*v1alpha1.AzureEventHubsTarget)

	var envs []corev1.EnvVar

	if sasAuth := typedTrg.Spec.Auth.SASToken; sasAuth != nil {
		envs = common.MaybeAppendValueFromEnvVar(envs, envHubKeyName, sasAuth.KeyName)
		envs = common.MaybeAppendValueFromEnvVar(envs, envHubKeyValue, sasAuth.KeyValue)
		envs = common.MaybeAppendValueFromEnvVar(envs, envHubConnStr, sasAuth.ConnectionString)
	}

	if spAuth := typedTrg.Spec.Auth.ServicePrincipal; spAuth != nil {
		envs = common.MaybeAppendValueFromEnvVar(envs, envAADTenantID, spAuth.TenantID)
		envs = common.MaybeAppendValueFromEnvVar(envs, envAADClientID, spAuth.ClientID)
		envs = common.MaybeAppendValueFromEnvVar(envs, envAADClientSecret, spAuth.ClientSecret)
	}

	if typedTrg.Spec.EventOptions != nil && typedTrg.Spec.EventOptions.PayloadPolicy != nil {
		envs = append(envs, corev1.EnvVar{
			Name:  envEventsPayloadPolicy,
			Value: string(*typedTrg.Spec.EventOptions.PayloadPolicy),
		})
	}

	return common.NewAdapterKnService(trg,
		resource.Image(r.adapterCfg.Image),
		resource.EnvVar(envHubNamespace, typedTrg.Spec.EventHubID.Namespace),
		resource.EnvVar(envHubName, typedTrg.Spec.EventHubID.EventHub),
		resource.EnvVar(envDiscardCECtx, strconv.FormatBool(typedTrg.Spec.DiscardCEContext)),
		resource.EnvVars(envs...),
		resource.EnvVars(r.adapterCfg.obsConfig.ToEnvVars()...),
	)
}

// RBACOwners implements common.AdapterServiceBuilder.
func (r *Reconciler) RBACOwners(trg v1alpha1.Reconcilable) ([]kmeta.OwnerRefable, error) {
	trgs, err := r.trgLister(trg.GetNamespace()).List(labels.Everything())
	if err != nil {
		return nil, fmt.Errorf("listing objects from cache: %w", err)
	}

	ownerRefables := make([]kmeta.OwnerRefable, len(trgs))
	for i := range trgs {
		ownerRefables[i] = trgs[i]
	}

	return ownerRefables, nil
}

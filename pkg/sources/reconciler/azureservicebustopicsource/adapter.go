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

package azureservicebustopicsource

import (
	"strconv"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"

	"knative.dev/eventing/pkg/reconciler/source"
	"knative.dev/pkg/apis"

	commonv1alpha1 "github.com/triggermesh/triggermesh/pkg/apis/common/v1alpha1"
	"github.com/triggermesh/triggermesh/pkg/apis/sources/v1alpha1"
	common "github.com/triggermesh/triggermesh/pkg/reconciler"
	"github.com/triggermesh/triggermesh/pkg/reconciler/resource"
)

// adapterConfig contains properties used to configure the source's adapter.
// These are automatically populated by envconfig.
type adapterConfig struct {
	// Container image
	// Uses a common adapter for both Azure Service Bus sources instead of a source-specific image.
	Image string `envconfig:"AZURESERVICEBUSSOURCE_IMAGE" default:"gcr.io/triggermesh/azureservicebussource-adapter"`
	// Configuration accessor for logging/metrics/tracing
	configs source.ConfigAccessor
}

// Verify that Reconciler implements common.AdapterBuilder.
var _ common.AdapterBuilder[*appsv1.Deployment] = (*Reconciler)(nil)

// BuildAdapter implements common.AdapterBuilder.
func (r *Reconciler) BuildAdapter(src commonv1alpha1.Reconcilable, sinkURI *apis.URL) (*appsv1.Deployment, error) {
	typedSrc := src.(*v1alpha1.AzureServiceBusTopicSource)

	return common.NewAdapterDeployment(src, sinkURI,
		resource.Image(r.adapterCfg.Image),

		resource.EnvVars(MakeAppEnv(typedSrc)...),
		resource.EnvVars(r.adapterCfg.configs.ToEnvVars()...),
	), nil
}

// MakeAppEnv extracts environment variables from the object.
// Exported to be used in external tools for local test environments.
func MakeAppEnv(o *v1alpha1.AzureServiceBusTopicSource) []corev1.EnvVar {
	var subsID string
	if sID := o.Status.SubscriptionID; sID != nil {
		subsID = sID.String()
	}

	var webSocketsEnable bool
	if wss := o.Spec.WebSocketsEnable; wss != nil {
		webSocketsEnable = *wss
	}

	var authEnvs []corev1.EnvVar
	if spAuth := o.Spec.Auth.ServicePrincipal; spAuth != nil {
		authEnvs = common.MaybeAppendValueFromEnvVar(authEnvs, common.EnvAADTenantID, spAuth.TenantID)
		authEnvs = common.MaybeAppendValueFromEnvVar(authEnvs, common.EnvAADClientID, spAuth.ClientID)
		authEnvs = common.MaybeAppendValueFromEnvVar(authEnvs, common.EnvAADClientSecret, spAuth.ClientSecret)
	}
	return append(authEnvs, []corev1.EnvVar{{
		Name:  common.EnvServiceBusEntityResourceID,
		Value: subsID,
	}, {
		Name:  common.EnvServiceBusWebSocketsEnable,
		Value: strconv.FormatBool(webSocketsEnable),
	}}...)
}

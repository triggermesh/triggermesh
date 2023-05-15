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

package azureeventhubssource

import (
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"

	"knative.dev/eventing/pkg/reconciler/source"
	"knative.dev/pkg/apis"

	commonv1alpha1 "github.com/triggermesh/triggermesh/pkg/apis/common/v1alpha1"
	"github.com/triggermesh/triggermesh/pkg/apis/sources/v1alpha1"
	common "github.com/triggermesh/triggermesh/pkg/reconciler"
	"github.com/triggermesh/triggermesh/pkg/reconciler/resource"
)

const healthPortName = "health"

// adapterConfig contains properties used to configure the source's adapter.
// These are automatically populated by envconfig.
type adapterConfig struct {
	// Container image
	Image string `default:"gcr.io/triggermesh/azureeventhubssource-adapter"`
	// Configuration accessor for logging/metrics/tracing
	configs source.ConfigAccessor
}

// Verify that Reconciler implements common.AdapterBuilder.
var _ common.AdapterBuilder[*appsv1.Deployment] = (*Reconciler)(nil)

// BuildAdapter implements common.AdapterBuilder.
func (r *Reconciler) BuildAdapter(src commonv1alpha1.Reconcilable, sinkURI *apis.URL) (*appsv1.Deployment, error) {
	typedSrc := src.(*v1alpha1.AzureEventHubsSource)

	return common.NewAdapterDeployment(src, sinkURI,
		resource.Image(r.adapterCfg.Image),

		resource.EnvVars(MakeAppEnv(typedSrc)...),
		resource.EnvVars(r.adapterCfg.configs.ToEnvVars()...),

		resource.Port(healthPortName, 8080),
		resource.StartupProbe("/health", healthPortName),
	), nil
}

// MakeAppEnv extracts environment variables from the object.
// Exported to be used in external tools for local test environments.
func MakeAppEnv(o *v1alpha1.AzureEventHubsSource) []corev1.EnvVar {
	var hubEnvs []corev1.EnvVar
	if sasAuth := o.Spec.Auth.SASToken; sasAuth != nil {
		hubEnvs = common.MaybeAppendValueFromEnvVar(hubEnvs, common.EnvHubKeyName, sasAuth.KeyName)
		hubEnvs = common.MaybeAppendValueFromEnvVar(hubEnvs, common.EnvHubKeyValue, sasAuth.KeyValue)
		hubEnvs = common.MaybeAppendValueFromEnvVar(hubEnvs, common.EnvHubConnStr, sasAuth.ConnectionString)
	}
	if spAuth := o.Spec.Auth.ServicePrincipal; spAuth != nil {
		hubEnvs = common.MaybeAppendValueFromEnvVar(hubEnvs, common.EnvAADTenantID, spAuth.TenantID)
		hubEnvs = common.MaybeAppendValueFromEnvVar(hubEnvs, common.EnvAADClientID, spAuth.ClientID)
		hubEnvs = common.MaybeAppendValueFromEnvVar(hubEnvs, common.EnvAADClientSecret, spAuth.ClientSecret)
	}

	if o.Spec.ConsumerGroup != nil {
		hubEnvs = append(hubEnvs, corev1.EnvVar{
			Name:  common.EnvHubConsumerGroup,
			Value: *o.Spec.ConsumerGroup,
		})
	}
	if o.Spec.MessageCountSize != nil {
		hubEnvs = append(hubEnvs, corev1.EnvVar{
			Name:  common.EnvHubMessageCountSize,
			Value: *o.Spec.MessageCountSize,
		})
	}
	if o.Spec.MessageTimeout != nil {
		hubEnvs = append(hubEnvs, corev1.EnvVar{
			Name:  common.EnvHubMessageTimeout,
			Value: *o.Spec.MessageTimeout,
		})
	}

	return append(hubEnvs,
		[]corev1.EnvVar{
			{
				Name:  common.EnvHubResourceID,
				Value: o.Spec.EventHubID.String(),
			}, {
				Name:  common.EnvHubNamespace,
				Value: o.Spec.EventHubID.Namespace,
			}, {
				Name:  common.EnvHubName,
				Value: o.Spec.EventHubID.ResourceName,
			},
		}...,
	)
}

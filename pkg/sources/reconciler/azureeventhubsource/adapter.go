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

package azureeventhubsource

import (
	"fmt"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/labels"

	"knative.dev/eventing/pkg/reconciler/source"
	"knative.dev/pkg/apis"
	"knative.dev/pkg/kmeta"

	"github.com/triggermesh/triggermesh/pkg/apis/sources/v1alpha1"
	"github.com/triggermesh/triggermesh/pkg/sources/reconciler/common"
	"github.com/triggermesh/triggermesh/pkg/sources/reconciler/common/resource"
)

const healthPortName = "health"

// adapterConfig contains properties used to configure the source's adapter.
// These are automatically populated by envconfig.
type adapterConfig struct {
	// Container image
	Image string `default:"gcr.io/triggermesh/azureeventhubsource-adapter"`
	// Configuration accessor for logging/metrics/tracing
	configs source.ConfigAccessor
}

// Verify that Reconciler implements common.AdapterDeploymentBuilder.
var _ common.AdapterDeploymentBuilder = (*Reconciler)(nil)

// BuildAdapter implements common.AdapterDeploymentBuilder.
func (r *Reconciler) BuildAdapter(src v1alpha1.EventSource, sinkURI *apis.URL) *appsv1.Deployment {
	typedSrc := src.(*v1alpha1.AzureEventHubSource)

	var hubEnvs []corev1.EnvVar
	if sasAuth := typedSrc.Spec.Auth.SASToken; sasAuth != nil {
		hubEnvs = common.MaybeAppendValueFromEnvVar(hubEnvs, common.EnvHubKeyName, sasAuth.KeyName)
		hubEnvs = common.MaybeAppendValueFromEnvVar(hubEnvs, common.EnvHubKeyValue, sasAuth.KeyValue)
		hubEnvs = common.MaybeAppendValueFromEnvVar(hubEnvs, common.EnvHubConnStr, sasAuth.ConnectionString)
	}
	if spAuth := typedSrc.Spec.Auth.ServicePrincipal; spAuth != nil {
		hubEnvs = common.MaybeAppendValueFromEnvVar(hubEnvs, common.EnvAADTenantID, spAuth.TenantID)
		hubEnvs = common.MaybeAppendValueFromEnvVar(hubEnvs, common.EnvAADClientID, spAuth.ClientID)
		hubEnvs = common.MaybeAppendValueFromEnvVar(hubEnvs, common.EnvAADClientSecret, spAuth.ClientSecret)
	}

	return common.NewAdapterDeployment(src, sinkURI,
		resource.Image(r.adapterCfg.Image),

		resource.EnvVar(common.EnvHubResourceID, typedSrc.Spec.EventHubID.String()),
		resource.EnvVar(common.EnvHubNamespace, typedSrc.Spec.EventHubID.Namespace),
		resource.EnvVar(common.EnvHubName, typedSrc.Spec.EventHubID.ResourceName),
		resource.EnvVars(hubEnvs...),
		resource.EnvVars(r.adapterCfg.configs.ToEnvVars()...),

		resource.Port(healthPortName, 8080),
		resource.StartupProbe("/health", healthPortName),
	)
}

// RBACOwners implements common.AdapterDeploymentBuilder.
func (r *Reconciler) RBACOwners(src v1alpha1.EventSource) ([]kmeta.OwnerRefable, error) {
	srcs, err := r.srcLister(src.GetNamespace()).List(labels.Everything())
	if err != nil {
		return nil, fmt.Errorf("listing objects from cache: %w", err)
	}

	ownerRefables := make([]kmeta.OwnerRefable, len(srcs))
	for i := range srcs {
		ownerRefables[i] = srcs[i]
	}

	return ownerRefables, nil
}

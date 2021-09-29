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

	var authEnvs []corev1.EnvVar
	authEnvs = common.MaybeAppendValueFromEnvVar(authEnvs, common.EnvConnStr, typedSrc.Spec.Auth.SASToken.ConnectionString)
	authEnvs = common.MaybeAppendValueFromEnvVar(authEnvs, common.EnvHubKeyName, typedSrc.Spec.Auth.SASToken.KeyName)
	authEnvs = common.MaybeAppendValueFromEnvVar(authEnvs, common.EnvHubKeyValue, typedSrc.Spec.Auth.SASToken.KeyValue)

	if typedSrc.Spec.Auth.ServicePrincipal != nil {
		authEnvs = append(authEnvs, []corev1.EnvVar{
			{
				Name:  common.EnvHubName,
				Value: typedSrc.Spec.HubName,
			}, {
				Name:  common.EnvHubNamespace,
				Value: typedSrc.Spec.HubNamespace,
			}, {
				Name: common.EnvTenantID,
				ValueFrom: &corev1.EnvVarSource{
					SecretKeyRef: typedSrc.Spec.Auth.ServicePrincipal.TenantID.ValueFromSecret,
				},
			}, {
				Name: common.EnvClientID,
				ValueFrom: &corev1.EnvVarSource{
					SecretKeyRef: typedSrc.Spec.Auth.ServicePrincipal.ClientID.ValueFromSecret,
				},
			}, {
				Name: common.EnvClientSecret,
				ValueFrom: &corev1.EnvVarSource{
					SecretKeyRef: typedSrc.Spec.Auth.ServicePrincipal.ClientSecret.ValueFromSecret,
				},
			},
		}...)
	}

	return common.NewAdapterDeployment(src, sinkURI,
		resource.Image(r.adapterCfg.Image),

		resource.EnvVars(authEnvs...),
		resource.EnvVars(r.adapterCfg.configs.ToEnvVars()...),
	)
}

// RBACOwners implements common.AdapterDeploymentBuilder.
func (r *Reconciler) RBACOwners(namespace string) ([]kmeta.OwnerRefable, error) {
	srcs, err := r.srcLister(namespace).List(labels.Everything())
	if err != nil {
		return nil, fmt.Errorf("listing objects from cache: %w", err)
	}

	ownerRefables := make([]kmeta.OwnerRefable, len(srcs))
	for i := range srcs {
		ownerRefables[i] = srcs[i]
	}

	return ownerRefables, nil
}

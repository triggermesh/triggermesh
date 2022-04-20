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

package salesforcesource

import (
	"strconv"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"

	"knative.dev/eventing/pkg/reconciler/source"
	"knative.dev/pkg/apis"
	"knative.dev/pkg/kmeta"

	commonv1alpha1 "github.com/triggermesh/triggermesh/pkg/apis/common/v1alpha1"
	"github.com/triggermesh/triggermesh/pkg/apis/sources/v1alpha1"
	common "github.com/triggermesh/triggermesh/pkg/reconciler"
	"github.com/triggermesh/triggermesh/pkg/reconciler/resource"
)

const (
	envSalesforceAuthClientID = "SALESFORCE_AUTH_CLIENT_ID"
	envSalesforceAuthServer   = "SALESFORCE_AUTH_SERVER"
	envSalesforceAuthUser     = "SALESFORCE_AUTH_USER"
	envSalesforceAuthCertKey  = "SALESFORCE_AUTH_CERT_KEY"
	envSalesforceAPIVersion   = "SALESFORCE_API_VERSION"
	envSalesforceChannel      = "SALESFORCE_SUBCRIPTION_CHANNEL"
	envSalesforceReplayID     = "SALESFORCE_SUBCRIPTION_REPLAY_ID"
)

// adapterConfig contains properties used to configure the source's adapter.
// These are automatically populated by envconfig.
type adapterConfig struct {
	// Container image
	Image string `default:"gcr.io/triggermesh/salesforcesource-adapter"`
	// Configuration accessor for logging/metrics/tracing
	configs source.ConfigAccessor
}

// Verify that Reconciler implements common.AdapterDeploymentBuilder.
var _ common.AdapterDeploymentBuilder = (*Reconciler)(nil)

// BuildAdapter implements common.AdapterDeploymentBuilder.
func (r *Reconciler) BuildAdapter(src commonv1alpha1.Reconcilable, sinkURI *apis.URL) *appsv1.Deployment {
	typedSrc := src.(*v1alpha1.SalesforceSource)

	appEnv := []corev1.EnvVar{
		{
			Name:  envSalesforceAuthClientID,
			Value: typedSrc.Spec.Auth.ClientID,
		},
		{
			Name:  envSalesforceAuthServer,
			Value: typedSrc.Spec.Auth.Server,
		},
		{
			Name:  envSalesforceAuthUser,
			Value: typedSrc.Spec.Auth.User,
		},
		{
			Name:  envSalesforceChannel,
			Value: typedSrc.Spec.Subscription.Channel,
		},
	}

	appEnv = common.MaybeAppendValueFromEnvVar(appEnv,
		envSalesforceAuthCertKey, typedSrc.Spec.Auth.CertKey,
	)

	if typedSrc.Spec.Subscription.ReplayID != nil {
		appEnv = append(appEnv, corev1.EnvVar{
			Name:  envSalesforceReplayID,
			Value: strconv.Itoa(*typedSrc.Spec.Subscription.ReplayID),
		})
	}

	if typedSrc.Spec.APIVersion != nil {
		appEnv = append(appEnv, corev1.EnvVar{
			Name:  envSalesforceAPIVersion,
			Value: *typedSrc.Spec.APIVersion,
		})
	}

	return common.NewAdapterDeployment(src, sinkURI,
		resource.Image(r.adapterCfg.Image),

		resource.EnvVars(appEnv...),
		resource.EnvVars(r.adapterCfg.configs.ToEnvVars()...),
	)
}

// RBACOwners implements common.AdapterDeploymentBuilder.
func (r *Reconciler) RBACOwners(src commonv1alpha1.Reconcilable) ([]kmeta.OwnerRefable, error) {
	return common.RBACOwners[*v1alpha1.SalesforceSource](r.srcLister(src.GetNamespace()))
}

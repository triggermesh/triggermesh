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

// Verify that Reconciler implements common.AdapterBuilder.
var _ common.AdapterBuilder[*appsv1.Deployment] = (*Reconciler)(nil)

// BuildAdapter implements common.AdapterBuilder.
func (r *Reconciler) BuildAdapter(src commonv1alpha1.Reconcilable, sinkURI *apis.URL) (*appsv1.Deployment, error) {
	typedSrc := src.(*v1alpha1.SalesforceSource)

	return common.NewAdapterDeployment(src, sinkURI,
		resource.Image(r.adapterCfg.Image),

		resource.EnvVars(MakeAppEnv(typedSrc)...),
		resource.EnvVars(r.adapterCfg.configs.ToEnvVars()...),
	), nil
}

// MakeAppEnv extracts environment variables from the object.
// Exported to be used in external tools for local test environments.
func MakeAppEnv(o *v1alpha1.SalesforceSource) []corev1.EnvVar {
	appEnv := []corev1.EnvVar{
		{
			Name:  envSalesforceAuthClientID,
			Value: o.Spec.Auth.ClientID,
		},
		{
			Name:  envSalesforceAuthServer,
			Value: o.Spec.Auth.Server,
		},
		{
			Name:  envSalesforceAuthUser,
			Value: o.Spec.Auth.User,
		},
		{
			Name:  envSalesforceChannel,
			Value: o.Spec.Subscription.Channel,
		},
	}

	appEnv = common.MaybeAppendValueFromEnvVar(appEnv,
		envSalesforceAuthCertKey, o.Spec.Auth.CertKey,
	)

	if o.Spec.Subscription.ReplayID != nil {
		appEnv = append(appEnv, corev1.EnvVar{
			Name:  envSalesforceReplayID,
			Value: strconv.Itoa(*o.Spec.Subscription.ReplayID),
		})
	}

	if o.Spec.APIVersion != nil {
		appEnv = append(appEnv, corev1.EnvVar{
			Name:  envSalesforceAPIVersion,
			Value: *o.Spec.APIVersion,
		})
	}

	return appEnv
}

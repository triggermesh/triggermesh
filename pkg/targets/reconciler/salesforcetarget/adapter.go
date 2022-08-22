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

package salesforcetarget

import (
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
	envSalesforceAuthClientID = "SALESFORCE_AUTH_CLIENT_ID"
	envSalesforceAuthServer   = "SALESFORCE_AUTH_SERVER"
	envSalesforceAuthUser     = "SALESFORCE_AUTH_USER"
	envSalesforceAuthCertKey  = "SALESFORCE_AUTH_CERT_KEY"
	envSalesforceAPIVersion   = "SALESFORCE_API_VERSION"
	envEventsPayloadPolicy    = "EVENTS_PAYLOAD_POLICY"
)

// adapterConfig contains properties used to configure the target's adapter.
// Public fields are automatically populated by envconfig.
type adapterConfig struct {
	// Configuration accessor for logging/metrics/tracing
	obsConfig source.ConfigAccessor
	// Container image
	Image string `default:"gcr.io/triggermesh/salesforcetarget-adapter"`
}

// Verify that Reconciler implements common.AdapterBuilder.
var _ common.AdapterBuilder[*servingv1.Service] = (*Reconciler)(nil)

// BuildAdapter implements common.AdapterBuilder.
func (r *Reconciler) BuildAdapter(trg commonv1alpha1.Reconcilable, _ *apis.URL) (*servingv1.Service, error) {
	typedTrg := trg.(*v1alpha1.SalesforceTarget)

	return common.NewAdapterKnService(trg, nil,
		resource.Image(r.adapterCfg.Image),
		resource.EnvVars(MakeAppEnv(typedTrg)...),
		resource.EnvVars(r.adapterCfg.obsConfig.ToEnvVars()...),
	), nil
}

// MakeAppEnv extracts environment variables from the object.
// Exported to be used in external tools for local test environments.
func MakeAppEnv(o *v1alpha1.SalesforceTarget) []corev1.EnvVar {
	env := []corev1.EnvVar{
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
			Name: envSalesforceAuthCertKey,
			ValueFrom: &corev1.EnvVarSource{
				SecretKeyRef: o.Spec.Auth.CertKey.SecretKeyRef,
			},
		},
		{
			Name:  common.EnvBridgeID,
			Value: common.GetStatefulBridgeID(o),
		},
	}

	if o.Spec.APIVersion != nil {
		env = append(env, corev1.EnvVar{
			Name:  envSalesforceAPIVersion,
			Value: *o.Spec.APIVersion,
		})
	}

	if o.Spec.EventOptions != nil && o.Spec.EventOptions.PayloadPolicy != nil {
		env = append(env, corev1.EnvVar{
			Name:  envEventsPayloadPolicy,
			Value: string(*o.Spec.EventOptions.PayloadPolicy),
		})
	}

	return env
}

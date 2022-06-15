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

package oracletarget

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

// adapterConfig contains properties used to configure the target's adapter.
// Public fields are automatically populated by envconfig.
type adapterConfig struct {
	// Configuration accessor for logging/metrics/tracing
	obsConfig source.ConfigAccessor
	// Container image
	Image string `default:"gcr.io/triggermesh/oracletarget-adapter"`
}

// Verify that Reconciler implements common.AdapterBuilder.
var _ common.AdapterBuilder[*servingv1.Service] = (*Reconciler)(nil)

// BuildAdapter implements common.AdapterBuilder.
func (r *Reconciler) BuildAdapter(trg commonv1alpha1.Reconcilable, _ *apis.URL) (*servingv1.Service, error) {
	typedTrg := trg.(*v1alpha1.OracleTarget)

	return common.NewAdapterKnService(trg, nil,
		resource.Image(r.adapterCfg.Image),
		resource.EnvVars(MakeAppEnv(typedTrg)...),
		resource.EnvVars(r.adapterCfg.obsConfig.ToEnvVars()...),
	), nil
}

// MakeAppEnv extracts environment variables from the object.
// Exported to be used in external tools for local test environments.
func MakeAppEnv(o *v1alpha1.OracleTarget) []corev1.EnvVar {
	env := []corev1.EnvVar{
		{
			Name: "ORACLE_API_PRIVATE_KEY",
			ValueFrom: &corev1.EnvVarSource{
				SecretKeyRef: o.Spec.OracleAPIPrivateKey.SecretKeyRef,
			},
		}, {
			Name: "ORACLE_API_PRIVATE_KEY_PASSPHRASE",
			ValueFrom: &corev1.EnvVarSource{
				SecretKeyRef: o.Spec.OracleAPIPrivateKeyPassphrase.SecretKeyRef,
			},
		}, {
			Name: "ORACLE_API_PRIVATE_KEY_FINGERPRINT",
			ValueFrom: &corev1.EnvVarSource{
				SecretKeyRef: o.Spec.OracleAPIPrivateKeyFingerprint.SecretKeyRef,
			},
		}, {
			Name:  "TENANT_OCID",
			Value: o.Spec.Tenancy,
		}, {
			Name:  "ORACLE_REGION",
			Value: o.Spec.Region,
		}, {
			Name:  "USER_OCID",
			Value: o.Spec.User,
		},
	}

	if o.Spec.OracleFunctionSpec != nil {
		env = append(env, corev1.EnvVar{
			Name:  "ORACLE_FUNCTION",
			Value: o.Spec.OracleFunctionSpec.Function,
		})
	}

	return env
}

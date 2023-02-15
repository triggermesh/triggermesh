/*
Copyright 2023 TriggerMesh Inc.

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

package solacesource

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
	envURL        = "URL"
	envQueueName  = "QUEUE_NAME"
	envUsername   = "USERNAME"
	envPassword   = "PASSWORD"
	envCA         = "CA"
	envClientCert = "CLIENT_CERT"
	envClientKey  = "CLIENT_KEY"
	envSkipVerify = "SKIP_VERIFY"

	envSaslEnable = "SASL_ENABLE"
	envTLSEnable  = "TLS_ENABLE"
)

// adapterConfig contains properties used to configure the target's adapter.
// Public fields are automatically populated by envconfig.
type adapterConfig struct {
	// Configuration accessor for logging/metrics/tracing
	configs source.ConfigAccessor
	// Container image
	Image string `default:"gcr.io/triggermesh/solacesource-adapter"`
}

// Verify that Reconciler implements common.AdapterBuilder.
var _ common.AdapterBuilder[*appsv1.Deployment] = (*Reconciler)(nil)

// BuildAdapter implements common.AdapterBuilder.
func (r *Reconciler) BuildAdapter(src commonv1alpha1.Reconcilable, sinkURI *apis.URL) (*appsv1.Deployment, error) {
	typedSrc := src.(*v1alpha1.SolaceSource)

	var secretVolumes []corev1.Volume
	var secretVolMounts []corev1.VolumeMount

	return common.NewAdapterDeployment(src, sinkURI,
		resource.Image(r.adapterCfg.Image),

		resource.EnvVars(MakeAppEnv(typedSrc)...),
		resource.EnvVars(r.adapterCfg.configs.ToEnvVars()...),

		resource.Volumes(secretVolumes...),
		resource.VolumeMounts(secretVolMounts...),
	), nil
}

// MakeAppEnv extracts environment variables from the object.
// Exported to be used in external tools for local test environments.
func MakeAppEnv(o *v1alpha1.SolaceSource) []corev1.EnvVar {
	envs := []corev1.EnvVar{
		{
			Name:  envURL,
			Value: o.Spec.URL,
		},
		{
			Name:  envQueueName,
			Value: o.Spec.QueueName,
		},
	}

	if o.Spec.Auth != nil {

		if o.Spec.Auth.SASLEnable != nil {
			envs = append(envs, corev1.EnvVar{
				Name:  envSaslEnable,
				Value: strconv.FormatBool(*o.Spec.Auth.SASLEnable),
			})
		}

		if o.Spec.Auth.TLSEnable != nil {
			envs = append(envs, corev1.EnvVar{
				Name:  envTLSEnable,
				Value: strconv.FormatBool(*o.Spec.Auth.TLSEnable),
			})
		}

		if o.Spec.Auth.Username != nil {
			envs = append(envs, corev1.EnvVar{
				Name:  envUsername,
				Value: *o.Spec.Auth.Username,
			})
		}

		if o.Spec.Auth.Password != nil {
			envs = common.MaybeAppendValueFromEnvVar(
				envs, envPassword, *o.Spec.Auth.Password,
			)
		}

		if o.Spec.Auth.TLS != nil {
			if o.Spec.Auth.TLS.CA != nil {
				envs = common.MaybeAppendValueFromEnvVar(
					envs, envCA, *o.Spec.Auth.TLS.CA,
				)
			}

			if o.Spec.Auth.TLS.ClientCert != nil {
				envs = common.MaybeAppendValueFromEnvVar(
					envs, envClientCert, *o.Spec.Auth.TLS.ClientCert,
				)
			}

			if o.Spec.Auth.TLS.ClientKey != nil {
				envs = common.MaybeAppendValueFromEnvVar(
					envs, envClientKey, *o.Spec.Auth.TLS.ClientKey,
				)
			}

			if o.Spec.Auth.TLS.SkipVerify != nil {
				envs = append(envs, corev1.EnvVar{
					Name:  envSkipVerify,
					Value: strconv.FormatBool(*o.Spec.Auth.TLS.SkipVerify),
				})
			}
		}
	}

	return envs
}

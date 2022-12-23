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

package kafkatarget

import (
	"path/filepath"
	"strconv"
	"strings"

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
	envBootstrapServers   = "BOOTSTRAP_SERVERS"
	envTopic              = "TOPIC"
	envUsername           = "USERNAME"
	envPassword           = "PASSWORD"
	envSecurityMechanisms = "SECURITY_MECHANISMS"
	envCA                 = "CA"
	envClientCert         = "CLIENT_CERT"
	envClientKey          = "CLIENT_KEY"
	envSkipVerify         = "SKIP_VERIFY"

	envSaslEnable = "SASL_ENABLE"
	envTLSEnable  = "TLS_ENABLE"

	envKerberosConfigPath  = "KERBEROS_CONFIG_PATH"
	envKerberosKeytabPath  = "KERBEROS_KEYTAB_PATH"
	envKerberosServiceName = "KERBEROS_SERVICE_NAME"
	envKerberosRealm       = "KERBEROS_REALM"
	envKerberosUsername    = "KERBEROS_USERNAME"
	envKerberosPassword    = "KERBEROS_PASSWORD"

	krb5ConfPath   = "/etc/krb5.conf"
	krb5KeytabPath = "/etc/krb5.keytab"
)

// adapterConfig contains properties used to configure the target's adapter.
// Public fields are automatically populated by envconfig.
type adapterConfig struct {
	// Configuration accessor for logging/metrics/tracing
	obsConfig source.ConfigAccessor
	// Container image
	Image string `default:"gcr.io/triggermesh/kafkatarget-adapter"`
}

// Verify that Reconciler implements common.AdapterBuilder.
var _ common.AdapterBuilder[*servingv1.Service] = (*Reconciler)(nil)

// BuildAdapter implements common.AdapterBuilder.
func (r *Reconciler) BuildAdapter(trg commonv1alpha1.Reconcilable, _ *apis.URL) (*servingv1.Service, error) {
	typedTrg := trg.(*v1alpha1.KafkaTarget)

	var secretVolumes []corev1.Volume
	var secretVolMounts []corev1.VolumeMount

	if typedTrg.Spec.Auth != nil {
		if typedTrg.Spec.Auth.Kerberos != nil {
			if typedTrg.Spec.Auth.Kerberos.Config != nil {
				configVol, configVolMount := secretVolumeAndMountAtPath(
					"krb5-config",
					krb5ConfPath,
					typedTrg.Spec.Auth.Kerberos.Config.ValueFromSecret.Name,
					typedTrg.Spec.Auth.Kerberos.Config.ValueFromSecret.Key,
				)
				secretVolumes = append(secretVolumes, configVol)
				secretVolMounts = append(secretVolMounts, configVolMount)
			}

			if typedTrg.Spec.Auth.Kerberos.Keytab != nil {
				keytabVol, keytabVolMount := secretVolumeAndMountAtPath(
					"krb5-keytab",
					krb5KeytabPath,
					typedTrg.Spec.Auth.Kerberos.Keytab.ValueFromSecret.Name,
					typedTrg.Spec.Auth.Kerberos.Keytab.ValueFromSecret.Key,
				)
				secretVolumes = append(secretVolumes, keytabVol)
				secretVolMounts = append(secretVolMounts, keytabVolMount)
			}
		}
	}

	return common.NewAdapterKnService(trg, nil,
		resource.Image(r.adapterCfg.Image),
		resource.EnvVars(MakeAppEnv(typedTrg)...),
		resource.EnvVars(r.adapterCfg.obsConfig.ToEnvVars()...),
		resource.Volumes(secretVolumes...),
		resource.VolumeMounts(secretVolMounts...),
	), nil
}

// MakeAppEnv extracts environment variables from the object.
// Exported to be used in external tools for local test environments.
func MakeAppEnv(o *v1alpha1.KafkaTarget) []corev1.EnvVar {
	env := []corev1.EnvVar{
		{
			Name:  envBootstrapServers,
			Value: strings.Join(o.Spec.BootstrapServers, ","),
		},
		{
			Name:  envTopic,
			Value: o.Spec.Topic,
		},
		{
			Name:  "DISCARD_CE_CONTEXT",
			Value: strconv.FormatBool(o.Spec.DiscardCEContext),
		},
	}

	if o.Spec.Auth != nil {

		if o.Spec.Auth.SASLEnable {
			env = append(env, corev1.EnvVar{
				Name:  envSaslEnable,
				Value: strconv.FormatBool(o.Spec.Auth.SASLEnable),
			})
		}

		if o.Spec.Auth.TLSEnable != nil {
			env = append(env, corev1.EnvVar{
				Name:  envTLSEnable,
				Value: strconv.FormatBool(*o.Spec.Auth.TLSEnable),
			})
		}

		if o.Spec.Auth.SecurityMechanisms != nil {
			env = append(env, corev1.EnvVar{
				Name:  envSecurityMechanisms,
				Value: *o.Spec.Auth.SecurityMechanisms,
			})
		}

		if o.Spec.Auth.Username != nil {
			env = append(env, corev1.EnvVar{
				Name:  envUsername,
				Value: *o.Spec.Auth.Username,
			})
		}

		if o.Spec.Auth.Password != nil {
			env = common.MaybeAppendValueFromEnvVar(
				env, envPassword, *o.Spec.Auth.Password,
			)
		}

		if o.Spec.Auth.TLS != nil {
			if o.Spec.Auth.TLS.CA != nil {
				env = common.MaybeAppendValueFromEnvVar(
					env, envCA, *o.Spec.Auth.TLS.CA,
				)
			}

			if o.Spec.Auth.TLS.ClientCert != nil {
				env = common.MaybeAppendValueFromEnvVar(
					env, envClientCert, *o.Spec.Auth.TLS.ClientCert,
				)
			}

			if o.Spec.Auth.TLS.ClientKey != nil {
				env = common.MaybeAppendValueFromEnvVar(
					env, envClientKey, *o.Spec.Auth.TLS.ClientKey,
				)
			}

			if o.Spec.Auth.TLS.SkipVerify != nil {
				env = append(env, corev1.EnvVar{
					Name:  envSkipVerify,
					Value: strconv.FormatBool(*o.Spec.Auth.TLS.SkipVerify),
				})
			}
		}

		if o.Spec.Auth.Kerberos != nil {
			if o.Spec.Auth.Kerberos.Config != nil {
				env = append(env, corev1.EnvVar{
					Name:  envKerberosConfigPath,
					Value: krb5ConfPath,
				})
			}

			if o.Spec.Auth.Kerberos.Keytab != nil {
				env = append(env, corev1.EnvVar{
					Name:  envKerberosKeytabPath,
					Value: krb5KeytabPath,
				})
			}

			if o.Spec.Auth.Kerberos.ServiceName != nil {
				env = append(env, corev1.EnvVar{
					Name:  envKerberosServiceName,
					Value: *o.Spec.Auth.Kerberos.ServiceName,
				})
			}

			if o.Spec.Auth.Kerberos.Realm != nil {
				env = append(env, corev1.EnvVar{
					Name:  envKerberosRealm,
					Value: *o.Spec.Auth.Kerberos.Realm,
				})
			}

			if o.Spec.Auth.Kerberos.Username != nil {
				env = append(env, corev1.EnvVar{
					Name:  envKerberosUsername,
					Value: *o.Spec.Auth.Kerberos.Username,
				})
			}

			if o.Spec.Auth.Kerberos.Password != nil {
				env = common.MaybeAppendValueFromEnvVar(
					env, envKerberosPassword, *o.Spec.Auth.Kerberos.Password,
				)
			}
		}
	}

	if o.Spec.TopicReplicationFactor != nil {
		env = append(env, corev1.EnvVar{
			Name:  "TOPIC_REPLICATION_FACTOR",
			Value: strconv.Itoa(int(*o.Spec.TopicReplicationFactor)),
		})
	}

	if o.Spec.TopicPartitions != nil {
		env = append(env, corev1.EnvVar{
			Name:  "TOPIC_PARTITIONS",
			Value: strconv.Itoa(int(*o.Spec.TopicPartitions)),
		})
	}

	return env
}

// secretVolumeAndMountAtPath returns a Secret-based volume and corresponding
// mount at the given path.
func secretVolumeAndMountAtPath(name, mountPath, secretName, secretKey string) (corev1.Volume, corev1.VolumeMount) {
	v := corev1.Volume{
		Name: name,
		VolumeSource: corev1.VolumeSource{
			Secret: &corev1.SecretVolumeSource{
				SecretName: secretName,
				Items: []corev1.KeyToPath{
					{
						Key:  secretKey,
						Path: filepath.Base(mountPath),
					},
				},
			},
		},
	}

	vm := corev1.VolumeMount{
		Name:      name,
		ReadOnly:  true,
		MountPath: mountPath,
		SubPath:   filepath.Base(mountPath),
	}

	return v, vm
}

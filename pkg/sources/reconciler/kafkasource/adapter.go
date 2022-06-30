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

package kafkasource

import (
	"path/filepath"
	"strconv"
	"strings"

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
	envBootstrapServers      = "BOOTSTRAP_SERVERS"
	envTopics                = "TOPICS"
	envGroupID               = "GROUP_ID"
	envUsername              = "USERNAME"
	envPassword              = "PASSWORD"
	envSecurityMechanisms    = "SECURITY_MECHANISMS"
	envSSLCA                 = "SSL_CA"
	envSSLClientCert         = "SSL_CLIENT_CERT"
	envSSLClientKey          = "SSL_CLIENT_KEY"
	envSSLInsecureSkipVerify = "SSL_INSECURE_SKIP_VERIFY"

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
	configs source.ConfigAccessor
	// Container image
	Image string `default:"gcr.io/triggermesh/kafkasource-adapter"`
}

// Verify that Reconciler implements common.AdapterBuilder.
var _ common.AdapterBuilder[*appsv1.Deployment] = (*Reconciler)(nil)

// BuildAdapter implements common.AdapterBuilder.
func (r *Reconciler) BuildAdapter(src commonv1alpha1.Reconcilable, sinkURI *apis.URL) (*appsv1.Deployment, error) {
	typedSrc := src.(*v1alpha1.KafkaSource)

	var secretVolumes []corev1.Volume
	var secretVolMounts []corev1.VolumeMount

	if typedSrc.Spec.Auth.KerberosAuth != nil {
		if typedSrc.Spec.Auth.KerberosAuth.KerberosConfig != nil {
			configVol, configVolMount := secretVolumeAndMountAtPath(
				"krb5-config",
				krb5ConfPath,
				typedSrc.Spec.Auth.KerberosAuth.KerberosConfig.ValueFromSecret.Name,
				typedSrc.Spec.Auth.KerberosAuth.KerberosConfig.ValueFromSecret.Key,
			)
			secretVolumes = append(secretVolumes, configVol)
			secretVolMounts = append(secretVolMounts, configVolMount)
		}

		if typedSrc.Spec.Auth.KerberosAuth.KerberosKeytab != nil {
			keytabVol, keytabVolMount := secretVolumeAndMountAtPath(
				"krb5-keytab",
				krb5KeytabPath,
				typedSrc.Spec.Auth.KerberosAuth.KerberosKeytab.ValueFromSecret.Name,
				typedSrc.Spec.Auth.KerberosAuth.KerberosKeytab.ValueFromSecret.Key,
			)
			secretVolumes = append(secretVolumes, keytabVol)
			secretVolMounts = append(secretVolMounts, keytabVolMount)
		}
	}

	return common.NewAdapterDeployment(src, sinkURI,
		resource.Image(r.adapterCfg.Image),

		resource.EnvVars(makeAppEnv(typedSrc)...),
		resource.EnvVars(r.adapterCfg.configs.ToEnvVars()...),

		resource.Volumes(secretVolumes...),
		resource.VolumeMounts(secretVolMounts...),
	), nil
}

func makeAppEnv(o *v1alpha1.KafkaSource) []corev1.EnvVar {
	envs := []corev1.EnvVar{
		{
			Name:  envBootstrapServers,
			Value: strings.Join(o.Spec.BootstrapServers, ","),
		},
		{
			Name:  envTopics,
			Value: strings.Join(o.Spec.Topics, ","),
		},
		{
			Name:  envSaslEnable,
			Value: strconv.FormatBool(o.Spec.Auth.SASLEnable),
		},
		{
			Name:  envTLSEnable,
			Value: strconv.FormatBool(o.Spec.Auth.TLSEnable),
		},
		{
			Name:  envGroupID,
			Value: o.Spec.GroupID,
		},
	}

	if o.Spec.Auth.SecurityMechanisms != nil {
		envs = append(envs, corev1.EnvVar{
			Name:  envSecurityMechanisms,
			Value: *o.Spec.Auth.SecurityMechanisms,
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

	if o.Spec.Auth.SSLAuth != nil {
		if o.Spec.Auth.SSLAuth.SSLCA != nil {
			envs = common.MaybeAppendValueFromEnvVar(
				envs, envSSLCA, *o.Spec.Auth.SSLAuth.SSLCA,
			)
		}

		if o.Spec.Auth.SSLAuth.SSLClientCert != nil {
			envs = common.MaybeAppendValueFromEnvVar(
				envs, envSSLClientCert, *o.Spec.Auth.SSLAuth.SSLClientCert,
			)
		}

		if o.Spec.Auth.SSLAuth.SSLClientKey != nil {
			envs = common.MaybeAppendValueFromEnvVar(
				envs, envSSLClientKey, *o.Spec.Auth.SSLAuth.SSLClientKey,
			)
		}

		if o.Spec.Auth.SSLAuth.InsecureSkipVerify != nil {
			envs = append(envs, corev1.EnvVar{
				Name:  envSSLInsecureSkipVerify,
				Value: strconv.FormatBool(*o.Spec.Auth.SSLAuth.InsecureSkipVerify),
			})
		}
	}

	if o.Spec.Auth.KerberosAuth != nil {
		if o.Spec.Auth.KerberosAuth.KerberosConfig != nil {
			envs = append(envs, corev1.EnvVar{
				Name:  envKerberosConfigPath,
				Value: krb5ConfPath,
			})
		}

		if o.Spec.Auth.KerberosAuth.KerberosKeytab != nil {
			envs = append(envs, corev1.EnvVar{
				Name:  envKerberosKeytabPath,
				Value: krb5KeytabPath,
			})
		}

		if o.Spec.Auth.KerberosAuth.KerberosServiceName != nil {
			envs = append(envs, corev1.EnvVar{
				Name:  envKerberosServiceName,
				Value: *o.Spec.Auth.KerberosAuth.KerberosServiceName,
			})
		}

		if o.Spec.Auth.KerberosAuth.KerberosRealm != nil {
			envs = append(envs, corev1.EnvVar{
				Name:  envKerberosRealm,
				Value: *o.Spec.Auth.KerberosAuth.KerberosRealm,
			})
		}

		if o.Spec.Auth.KerberosAuth.Username != nil {
			envs = append(envs, corev1.EnvVar{
				Name:  envKerberosUsername,
				Value: *o.Spec.Auth.KerberosAuth.Username,
			})
		}

		if o.Spec.Auth.KerberosAuth.Password != nil {
			envs = common.MaybeAppendValueFromEnvVar(
				envs, envKerberosPassword, *o.Spec.Auth.KerberosAuth.Password,
			)
		}
	}

	return envs
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

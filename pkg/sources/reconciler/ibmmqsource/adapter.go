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

package ibmmqsource

import (
	"path/filepath"
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
	envQueueManager       = "QUEUE_MANAGER"
	envChannelName        = "CHANNEL_NAME"
	envConnectionName     = "CONNECTION_NAME"
	envUser               = "USER"
	envPassword           = "PASSWORD"
	envQueueName          = "QUEUE_NAME"
	envDeadLetterQManager = "DEAD_LETTER_QUEUE_MANAGER"
	envDeadLetterQueue    = "DEAD_LETTER_QUEUE"
	envBackoffDelay       = "BACKOFF_DELAY"
	envRetry              = "DELIVERY_RETRY"
	envTLSCipher          = "TLS_CIPHER"
	envTLSClientAuth      = "TLS_CLIENT_AUTH"
	envTLSCertLabel       = "TLS_CERT_LABEL"

	KeystoreMountPath    = "/opt/mqm-keystore/key.kdb"
	PasswdStashMountPath = "/opt/mqm-keystore/key.sth"
)

// adapterConfig contains properties used to configure the target's adapter.
// Public fields are automatically populated by envconfig.
type adapterConfig struct {
	// Configuration accessor for logging/metrics/tracing
	configs source.ConfigAccessor
	// Container image
	Image string `default:"gcr.io/triggermesh/ibmmqsource-adapter"`
}

// Verify that Reconciler implements common.AdapterBuilder.
var _ common.AdapterBuilder[*appsv1.Deployment] = (*Reconciler)(nil)

// BuildAdapter implements common.AdapterBuilder.
func (r *Reconciler) BuildAdapter(src commonv1alpha1.Reconcilable, sinkURI *apis.URL) (*appsv1.Deployment, error) {
	typedSrc := src.(*v1alpha1.IBMMQSource)

	var secretVolumes []corev1.Volume
	var secretVolMounts []corev1.VolumeMount

	if typedSrc.Spec.Auth.TLS != nil {
		keyDBVol, keyDBVolMount := secretVolumeAndMountAtPath(
			"key-database",
			KeystoreMountPath,
			typedSrc.Spec.Auth.TLS.KeyRepository.KeyDatabase.ValueFromSecret.Name,
			typedSrc.Spec.Auth.TLS.KeyRepository.KeyDatabase.ValueFromSecret.Key,
		)

		pwStashVol, pwStashVolMount := secretVolumeAndMountAtPath(
			"db-password",
			PasswdStashMountPath,
			typedSrc.Spec.Auth.TLS.KeyRepository.PasswordStash.ValueFromSecret.Name,
			typedSrc.Spec.Auth.TLS.KeyRepository.PasswordStash.ValueFromSecret.Key,
		)

		secretVolumes = append(secretVolumes, keyDBVol, pwStashVol)
		secretVolMounts = append(secretVolMounts, keyDBVolMount, pwStashVolMount)
	}

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
func MakeAppEnv(o *v1alpha1.IBMMQSource) []corev1.EnvVar {
	env := []corev1.EnvVar{
		{
			Name:  envConnectionName,
			Value: o.Spec.ConnectionName,
		},
		{
			Name:  envQueueManager,
			Value: o.Spec.QueueManager,
		},
		{
			Name:  envQueueName,
			Value: o.Spec.QueueName,
		},
		{
			Name:  envChannelName,
			Value: o.Spec.ChannelName,
		},
		{
			Name:  envDeadLetterQManager,
			Value: o.Spec.Delivery.DeadLetterQueueManager,
		},
		{
			Name:  envDeadLetterQueue,
			Value: o.Spec.Delivery.DeadLetterQueue,
		},
		{
			Name:  envBackoffDelay,
			Value: strconv.Itoa(o.Spec.Delivery.BackoffDelay),
		},
		{
			Name:  envRetry,
			Value: strconv.Itoa(o.Spec.Delivery.Retry),
		},
	}

	env = common.MaybeAppendValueFromEnvVar(env, envUser, o.Spec.Auth.User)
	env = common.MaybeAppendValueFromEnvVar(env, envPassword, o.Spec.Auth.Password)

	if o.Spec.Auth.TLS != nil {
		env = append(env, []corev1.EnvVar{
			{
				Name:  envTLSCipher,
				Value: o.Spec.Auth.TLS.Cipher,
			},
			{
				Name:  envTLSClientAuth,
				Value: strconv.FormatBool(o.Spec.Auth.TLS.ClientAuthRequired),
			},
		}...)

		if o.Spec.Auth.TLS.CertLabel != nil {
			env = append(env, []corev1.EnvVar{
				{
					Name:  envTLSCertLabel,
					Value: *o.Spec.Auth.TLS.CertLabel,
				},
			}...)
		}
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

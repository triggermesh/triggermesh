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

package ibmmqtarget

import (
	"path/filepath"
	"strconv"

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
	envQueueManager   = "QUEUE_MANAGER"
	envChannelName    = "CHANNEL_NAME"
	envConnectionName = "CONNECTION_NAME"
	envUser           = "USER"
	envPassword       = "PASSWORD"
	envQueueName      = "QUEUE_NAME"
	envReplyToManager = "REPLY_TO_MANAGER"
	envReplyToQueue   = "REPLY_TO_QUEUE"
	envTLSCipher      = "TLS_CIPHER"
	envTLSClientAuth  = "TLS_CLIENT_AUTH"
	envTLSCertLabel   = "TLS_CERT_LABEL"

	envDiscardCEContext    = "DISCARD_CE_CONTEXT"
	envEventsPayloadPolicy = "EVENTS_PAYLOAD_POLICY"

	KeystoreMountPath    = "/opt/mqm-keystore/key.kdb"
	PasswdStashMountPath = "/opt/mqm-keystore/key.sth"
)

// adapterConfig contains properties used to configure the target's adapter.
// Public fields are automatically populated by envconfig.
type adapterConfig struct {
	// Configuration accessor for logging/metrics/tracing
	obsConfig source.ConfigAccessor
	// Container image
	Image string `default:"gcr.io/triggermesh/ibmmqtarget-adapter"`
}

// Verify that Reconciler implements common.AdapterBuilder.
var _ common.AdapterBuilder[*servingv1.Service] = (*Reconciler)(nil)

// BuildAdapter implements common.AdapterBuilder.
func (r *Reconciler) BuildAdapter(trg commonv1alpha1.Reconcilable, _ *apis.URL) (*servingv1.Service, error) {
	typedTrg := trg.(*v1alpha1.IBMMQTarget)

	var secretVolumes []corev1.Volume
	var secretVolMounts []corev1.VolumeMount

	if typedTrg.Spec.Auth.TLS != nil {
		keyDBVol, keyDBVolMount := secretVolumeAndMountAtPath(
			"key-database",
			KeystoreMountPath,
			typedTrg.Spec.Auth.TLS.KeyRepository.KeyDatabase.ValueFromSecret.Name,
			typedTrg.Spec.Auth.TLS.KeyRepository.KeyDatabase.ValueFromSecret.Key,
		)

		pwStashVol, pwStashVolMount := secretVolumeAndMountAtPath(
			"db-password",
			PasswdStashMountPath,
			typedTrg.Spec.Auth.TLS.KeyRepository.PasswordStash.ValueFromSecret.Name,
			typedTrg.Spec.Auth.TLS.KeyRepository.PasswordStash.ValueFromSecret.Key,
		)

		secretVolumes = append(secretVolumes, keyDBVol, pwStashVol)
		secretVolMounts = append(secretVolMounts, keyDBVolMount, pwStashVolMount)
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
func MakeAppEnv(o *v1alpha1.IBMMQTarget) []corev1.EnvVar {
	env := []corev1.EnvVar{
		{
			Name:  common.EnvBridgeID,
			Value: common.GetStatefulBridgeID(o),
		},
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
			Name:  envDiscardCEContext,
			Value: strconv.FormatBool(o.Spec.DiscardCEContext),
		},
	}

	if o.Spec.ReplyTo != nil {
		env = append(env, []corev1.EnvVar{
			{
				Name:  envReplyToManager,
				Value: o.Spec.ReplyTo.QueueManager,
			},
			{
				Name:  envReplyToQueue,
				Value: o.Spec.ReplyTo.QueueName,
			},
		}...)
	}

	env = common.MaybeAppendValueFromEnvVar(env, envUser, o.Spec.Auth.User)
	env = common.MaybeAppendValueFromEnvVar(env, envPassword, o.Spec.Auth.Password)

	if o.Spec.EventOptions != nil && o.Spec.EventOptions.PayloadPolicy != nil {
		env = append(env, corev1.EnvVar{
			Name:  envEventsPayloadPolicy,
			Value: string(*o.Spec.EventOptions.PayloadPolicy),
		})
	}

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

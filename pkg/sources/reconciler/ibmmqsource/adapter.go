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

package ibmmqsource

import (
	"fmt"
	"strconv"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/labels"

	"knative.dev/eventing/pkg/reconciler/source"
	"knative.dev/pkg/apis"
	"knative.dev/pkg/kmeta"

	"github.com/triggermesh/triggermesh/pkg/apis/sources/v1alpha1"
	"github.com/triggermesh/triggermesh/pkg/sources/reconciler/common"
	"github.com/triggermesh/triggermesh/pkg/sources/reconciler/common/resource"
	libreconciler "github.com/triggermesh/triggermesh/pkg/targets/reconciler"
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
	Image string `envconfig:"IBMMQSOURCE_ADAPTER_IMAGE" default:"gcr.io/triggermesh/ibmmqsource-adapter"`
}

// Verify that Reconciler implements common.AdapterDeploymentBuilder.
var _ common.AdapterDeploymentBuilder = (*Reconciler)(nil)

// BuildAdapter implements common.AdapterDeploymentBuilder.
func (r *Reconciler) BuildAdapter(src v1alpha1.EventSource, sinkURI *apis.URL) *appsv1.Deployment {
	typedSrc := src.(*v1alpha1.IBMMQSource)
	appEnv := makeAppEnv(typedSrc)
	appEnv = common.MaybeAppendValueFromEnvVar(appEnv, envUser, typedSrc.Spec.Auth.User)
	appEnv = common.MaybeAppendValueFromEnvVar(appEnv, envPassword, typedSrc.Spec.Auth.Password)

	keystoreMount := func(interface{}) {}
	passwdStashMount := func(interface{}) {}

	if typedSrc.Spec.Auth.TLS != nil {
		appEnv = append(appEnv, []corev1.EnvVar{
			{
				Name:  envTLSCipher,
				Value: typedSrc.Spec.Auth.TLS.Cipher,
			},
			{
				Name:  envTLSClientAuth,
				Value: fmt.Sprintf("%t", typedSrc.Spec.Auth.TLS.ClientAuthRequired),
			},
		}...)

		if typedSrc.Spec.Auth.TLS.CertLabel != nil {
			appEnv = append(appEnv, []corev1.EnvVar{
				{
					Name:  envTLSCertLabel,
					Value: *typedSrc.Spec.Auth.TLS.CertLabel,
				},
			}...)
		}

		keystoreMount = resource.SecretMount("key-database", KeystoreMountPath, typedSrc.Spec.Auth.TLS.KeyRepository.KeyDatabase.ValueFromSecret)
		passwdStashMount = resource.SecretMount("db-password", PasswdStashMountPath, typedSrc.Spec.Auth.TLS.KeyRepository.PasswordStash.ValueFromSecret)
	}

	return common.NewAdapterDeployment(src, sinkURI,
		resource.Image(r.adapterCfg.Image),
		resource.EnvVars(appEnv...),
		resource.EnvVar(common.EnvNamespace, src.GetNamespace()),
		resource.EnvVar(common.EnvName, src.GetName()),
		resource.EnvVars(r.adapterCfg.configs.ToEnvVars()...),
		keystoreMount,
		passwdStashMount,
	)
}

func makeAppEnv(o *v1alpha1.IBMMQSource) []corev1.EnvVar {
	env := []corev1.EnvVar{
		{
			Name:  libreconciler.EnvBridgeID,
			Value: libreconciler.GetStatefulBridgeID(o),
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

	return env
}

// RBACOwners implements common.AdapterDeploymentBuilder.
func (r *Reconciler) RBACOwners(src v1alpha1.EventSource) ([]kmeta.OwnerRefable, error) {
	srcs, err := r.srcLister(src.GetNamespace()).List(labels.Everything())
	if err != nil {
		return nil, fmt.Errorf("listing objects from cache: %w", err)
	}

	ownerRefables := make([]kmeta.OwnerRefable, len(srcs))
	for i := range srcs {
		ownerRefables[i] = srcs[i]
	}

	return ownerRefables, nil
}

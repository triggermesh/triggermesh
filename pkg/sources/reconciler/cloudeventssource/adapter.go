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

package cloudeventssource

import (
	"encoding/json"
	"fmt"
	"path"
	"path/filepath"
	"strconv"

	corev1 "k8s.io/api/core/v1"

	"knative.dev/eventing/pkg/adapter/v2"
	"knative.dev/eventing/pkg/reconciler/source"
	"knative.dev/pkg/apis"
	servingv1 "knative.dev/serving/pkg/apis/serving/v1"

	commonv1alpha1 "github.com/triggermesh/triggermesh/pkg/apis/common/v1alpha1"
	"github.com/triggermesh/triggermesh/pkg/apis/sources/v1alpha1"
	common "github.com/triggermesh/triggermesh/pkg/reconciler"
	"github.com/triggermesh/triggermesh/pkg/reconciler/resource"
	"github.com/triggermesh/triggermesh/pkg/sources/cloudevents"
)

const (
	envCloudEventsPath                 = "CLOUDEVENTS_PATH"
	envCloudEventsBasicAuthCredentials = "CLOUDEVENTS_BASICAUTH_CREDENTIALS"
	envCloudEventsRateLimiterRPS       = "CLOUDEVENTS_RATELIMITER_RPS"
)

// adapterConfig contains properties used to configure the source's adapter.
// These are automatically populated by envconfig.
type adapterConfig struct {
	// Container image
	Image string `default:"gcr.io/triggermesh/cloudeventssource-adapter"`
	// Configuration accessor for logging/metrics/tracing
	configs source.ConfigAccessor
}

// Verify that Reconciler implements common.AdapterDeploymentBuilder.
var _ common.AdapterServiceBuilder = (*Reconciler)(nil)

// BuildAdapter implements common.AdapterDeploymentBuilder.
func (r *Reconciler) BuildAdapter(src commonv1alpha1.Reconcilable, sinkURI *apis.URL) *servingv1.Service {
	typedSrc := src.(*v1alpha1.CloudEventsSource)

	ceOverridesStr := cloudevents.OverridesJSON(typedSrc.Spec.CloudEventOverrides)

	options := []resource.ObjectOption{
		resource.Image(r.adapterCfg.Image),

		resource.VisibilityPublic,

		resource.EnvVar(adapter.EnvConfigCEOverrides, ceOverridesStr),
		resource.EnvVars(r.adapterCfg.configs.ToEnvVars()...),
		resource.EnvVars(makeAppEnv(typedSrc)...),
	}

	if typedSrc.Spec.Credentials != nil {
		// For each BasicAuth credentials a secret is mounted and a tuple
		// key/mounted-file pair is added to the environment variable.
		kvs := []KeyMountedValue{}

		secretArrayNamePrefix := "basicauths"
		secretBasePath := "/opt"
		secretFileName := "cesource"

		for i, ba := range typedSrc.Spec.Credentials.BasicAuths {
			if ba.Password.ValueFromSecret != nil {
				secretName := fmt.Sprintf("%s%d", secretArrayNamePrefix, i)
				secretPath := filepath.Join(secretBasePath, secretName)

				options = append(options, secretMountAtPath(
					secretName,
					secretPath,
					secretFileName,
					ba.Password.ValueFromSecret.Name,
					ba.Password.ValueFromSecret.Key))

				kvs = append(kvs, KeyMountedValue{
					Key:              ba.Username,
					MountedValueFile: path.Join(secretPath, secretFileName),
				})
			}
		}

		if len(kvs) != 0 {
			s, _ := json.Marshal(kvs)
			options = append(options, resource.EnvVar(envCloudEventsBasicAuthCredentials, string(s)))
		}
	}

	return common.NewAdapterKnService(src, sinkURI, options...)
}

type KeyMountedValue struct {
	Key              string
	MountedValueFile string
}

func (kmv *KeyMountedValue) Decode(value string) error {
	if err := json.Unmarshal([]byte(value), kmv); err != nil {
		return err
	}
	return nil
}

// makeAppEnv creates the environment variables specific to this adapter component.
func makeAppEnv(o *v1alpha1.CloudEventsSource) []corev1.EnvVar {
	envs := []corev1.EnvVar{
		{
			Name:  common.EnvBridgeID,
			Value: common.GetStatefulBridgeID(o),
		},
	}

	if o.Spec.Path != nil {
		envs = append(envs, corev1.EnvVar{
			Name:  envCloudEventsPath,
			Value: *o.Spec.Path,
		})
	}

	if o.Spec.RateLimiter != nil {
		envs = append(envs, corev1.EnvVar{
			Name:  envCloudEventsRateLimiterRPS,
			Value: strconv.Itoa(o.Spec.RateLimiter.RequestsPerSecond),
		})
	}

	return envs
}

// secretMountAtPath returns a build option for a service that adds a
// secret based volume and mount a key at a path.
func secretMountAtPath(name, mountPath, mountFile, secretName, secretKey string) resource.ObjectOption {
	return func(object interface{}) {
		ksvc, ok := object.(*servingv1.Service)
		if !ok {
			return
		}

		ksvc.Spec.Template.Spec.Volumes = append(
			ksvc.Spec.Template.Spec.Volumes,
			corev1.Volume{
				Name: name,
				VolumeSource: corev1.VolumeSource{
					Secret: &corev1.SecretVolumeSource{
						SecretName: secretName,
						Items: []corev1.KeyToPath{{
							Key:  secretKey,
							Path: mountFile,
						}},
					},
				},
			})

		if len(ksvc.Spec.Template.Spec.Containers) == 0 {
			ksvc.Spec.Template.Spec.Containers = make([]corev1.Container, 1)
		}

		ksvc.Spec.Template.Spec.Containers[0].VolumeMounts = append(
			ksvc.Spec.Template.Spec.Containers[0].VolumeMounts,
			corev1.VolumeMount{
				Name:      name,
				ReadOnly:  true,
				MountPath: mountPath,
			},
		)
	}
}

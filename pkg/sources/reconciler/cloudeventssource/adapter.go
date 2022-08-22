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

// Verify that Reconciler implements common.AdapterBuilder.
var _ common.AdapterBuilder[*servingv1.Service] = (*Reconciler)(nil)

// BuildAdapter implements common.AdapterBuilder.
func (r *Reconciler) BuildAdapter(src commonv1alpha1.Reconcilable, sinkURI *apis.URL) (*servingv1.Service, error) {
	typedSrc := src.(*v1alpha1.CloudEventsSource)

	var authVolumes []corev1.Volume
	var authVolumeMounts []corev1.VolumeMount
	var authEnvs []corev1.EnvVar

	if typedSrc.Spec.Credentials != nil {
		// For each BasicAuth credentials a secret is mounted and a tuple
		// key/mounted-file pair is added to the environment variable.
		kvs := []KeyMountedValue{}

		const (
			secretArrayNamePrefix = "basicauths"
			secretBasePath        = "/opt"
			secretFileName        = "cesource"
		)

		for i, ba := range typedSrc.Spec.Credentials.BasicAuths {
			if ba.Password.ValueFromSecret != nil {
				secretName := fmt.Sprintf("%s%d", secretArrayNamePrefix, i)
				secretPath := filepath.Join(secretBasePath, secretName)

				v, vm := secretVolumeAndMountAtPath(
					secretName,
					secretPath,
					secretFileName,
					ba.Password.ValueFromSecret.Name,
					ba.Password.ValueFromSecret.Key,
				)
				authVolumes = append(authVolumes, v)
				authVolumeMounts = append(authVolumeMounts, vm)

				kvs = append(kvs, KeyMountedValue{
					Key:              ba.Username,
					MountedValueFile: path.Join(secretPath, secretFileName),
				})
			}
		}

		if len(kvs) > 0 {
			s, err := json.Marshal(kvs)
			if err != nil {
				return nil, fmt.Errorf("serializing keyMountedValues to JSON: %w", err)
			}

			authEnvs = append(authEnvs, corev1.EnvVar{
				Name:  envCloudEventsBasicAuthCredentials,
				Value: string(s),
			})
		}
	}

	ceOverridesStr := cloudevents.OverridesJSON(typedSrc.Spec.CloudEventOverrides)

	// Common reconciler internals set the visibility to non public by default. That does
	// not play well with sources which should default to being public if no visibility
	// configuration is provided.
	switch {
	case typedSrc.Spec.AdapterOverrides == nil:
		t := true
		typedSrc.Spec.AdapterOverrides = &commonv1alpha1.AdapterOverrides{
			Public: &t,
		}
	case typedSrc.Spec.AdapterOverrides.Public == nil:
		t := true
		typedSrc.Spec.AdapterOverrides.Public = &t
	}

	return common.NewAdapterKnService(src, sinkURI,
		resource.Image(r.adapterCfg.Image),

		resource.Volumes(authVolumes...),
		resource.VolumeMounts(authVolumeMounts...),
		resource.EnvVars(authEnvs...),

		resource.EnvVars(MakeAppEnv(typedSrc)...),
		resource.EnvVar(adapter.EnvConfigCEOverrides, ceOverridesStr),
		resource.EnvVars(r.adapterCfg.configs.ToEnvVars()...),
	), nil
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

// MakeAppEnv extracts environment variables from the object.
// Exported to be used in external tools for local test environments.
func MakeAppEnv(o *v1alpha1.CloudEventsSource) []corev1.EnvVar {
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

// secretVolumeAndMountAtPath returns a Secret-based volume and corresponding
// mount at the given path.
func secretVolumeAndMountAtPath(name, mountPath, mountFile, secretName, secretKey string) (corev1.Volume, corev1.VolumeMount) {
	v := corev1.Volume{
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
	}

	vm := corev1.VolumeMount{
		Name:      name,
		ReadOnly:  true,
		MountPath: mountPath,
	}

	return v, vm
}

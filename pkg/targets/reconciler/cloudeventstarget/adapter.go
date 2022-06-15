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

package cloudeventstarget

import (
	"path"

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
	envCloudEventsPath                  = "CLOUDEVENTS_PATH"
	envCloudEventsURL                   = "CLOUDEVENTS_URL"
	envCloudEventsBasicAuthUsername     = "CLOUDEVENTS_BASICAUTH_USERNAME"
	envCloudEventsBasicAuthPasswordPath = "CLOUDEVENTS_BASICAUTH_PASSWORD_PATH"
)

// adapterConfig contains properties used to configure the target's adapter.
// Public fields are automatically populated by envconfig.
type adapterConfig struct {
	// Configuration accessor for logging/metrics/tracing
	obsConfig source.ConfigAccessor
	// Container image
	Image string `default:"gcr.io/triggermesh/cloudeventstarget-adapter"`
}

// Verify that Reconciler implements common.AdapterBuilder.
var _ common.AdapterBuilder[*servingv1.Service] = (*Reconciler)(nil)

// BuildAdapter implements common.AdapterBuilder.
func (r *Reconciler) BuildAdapter(trg commonv1alpha1.Reconcilable, _ *apis.URL) (*servingv1.Service, error) {
	typedTrg := trg.(*v1alpha1.CloudEventsTarget)

	options := []resource.ObjectOption{}

	if typedTrg.Spec.Credentials != nil {
		secretName := "basicauths"
		secretPath := "/opt/basicauths"
		secretFileName := "cesource"

		options = append(options, resource.EnvVar(envCloudEventsBasicAuthUsername, typedTrg.Spec.Credentials.BasicAuth.Username))

		if typedTrg.Spec.Credentials.BasicAuth.Password.ValueFromSecret != nil {
			v, vm := secretVolumeAndMountAtPath(
				secretName,
				secretPath,
				secretFileName,
				typedTrg.Spec.Credentials.BasicAuth.Password.ValueFromSecret.Name,
				typedTrg.Spec.Credentials.BasicAuth.Password.ValueFromSecret.Key,
			)

			options = append(options,
				resource.Volumes(v),
				resource.VolumeMounts(vm),
				resource.EnvVar(envCloudEventsBasicAuthPasswordPath, path.Join(secretPath, secretFileName)),
			)
		}
	}

	options = append(options,
		resource.EnvVars(MakeAppEnv(typedTrg)...),
		// make sure that a non optional parameter is located as the last element
		// to avoid derivative comparison issues when the environment variables
		// tail element is removed.
		resource.Image(r.adapterCfg.Image),
		resource.EnvVars(r.adapterCfg.obsConfig.ToEnvVars()...),
	)

	return common.NewAdapterKnService(trg, nil, options...), nil
}

// MakeAppEnv extracts environment variables from the object.
// Exported to be used in external tools for local test environments.
func MakeAppEnv(o *v1alpha1.CloudEventsTarget) []corev1.EnvVar {
	env := []corev1.EnvVar{
		{
			Name:  envCloudEventsURL,
			Value: o.Spec.Endpoint.String(),
		},
	}

	if o.Spec.Path != nil {
		env = append(env, corev1.EnvVar{
			Name:  envCloudEventsPath,
			Value: *o.Spec.Path,
		})
	}

	return env
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

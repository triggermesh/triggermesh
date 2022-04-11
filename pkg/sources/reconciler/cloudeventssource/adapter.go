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

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/labels"

	"knative.dev/eventing/pkg/adapter/v2"
	"knative.dev/eventing/pkg/reconciler/source"
	"knative.dev/pkg/apis"
	"knative.dev/pkg/kmeta"
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
	envCloudEventsTokenCredentials     = "CLOUDEVENTS_TOKEN_CREDENTIALS"
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

	// Add mount secrets for each BasicAuth and Token element.
	options := []resource.ObjectOption{
		resource.Image(r.adapterCfg.Image),

		resource.VisibilityPublic,
		resource.EnvVar(common.EnvNamespace, src.GetNamespace()),
		resource.EnvVar(common.EnvName, src.GetName()),
		resource.EnvVar(adapter.EnvConfigCEOverrides, ceOverridesStr),
		resource.EnvVars(r.adapterCfg.configs.ToEnvVars()...),
		resource.EnvVars(makeAppEnv(typedSrc)...),
	}

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

			options = append(options, resource.SecretMount(
				secretName,
				secretPath,
				ba.Password.ValueFromSecret.Name,
				resource.WithVolumeSecretItem(ba.Password.ValueFromSecret.Key, secretFileName)))

			kvs = append(kvs, KeyMountedValue{
				Key:              ba.Username,
				MountedValueFile: path.Join(secretPath, secretFileName),
			})
		}
	}

	if len(kvs) != 0 {
		s, _ := json.Marshal(kvs)
		options = append(options, resource.EnvVar(envCloudEventsBasicAuthCredentials, string(s)))

		// empty kvs for re-using at tokens
		kvs = kvs[:0]
	}

	secretArrayNamePrefix = "tokens"

	for i, t := range typedSrc.Spec.Credentials.Tokens {
		if t.Value.ValueFromSecret != nil {
			secretName := fmt.Sprintf("%s%d", secretArrayNamePrefix, i)
			secretPath := filepath.Join(secretBasePath, secretName)

			options = append(options, resource.SecretMount(
				secretName,
				secretPath,
				t.Value.ValueFromSecret.Name,
				resource.WithVolumeSecretItem(t.Value.ValueFromSecret.Key, secretFileName)))

			kvs = append(kvs, KeyMountedValue{
				Key:              t.Header,
				MountedValueFile: path.Join(secretPath, secretFileName),
			})
		}
	}

	if len(kvs) != 0 {
		s, _ := json.Marshal(kvs)
		options = append(options, resource.EnvVar(envCloudEventsTokenCredentials, string(s)))
	}

	return common.NewAdapterKnService(src, sinkURI, options...)
}

// RBACOwners implements common.AdapterDeploymentBuilder.
func (r *Reconciler) RBACOwners(src commonv1alpha1.Reconcilable) ([]kmeta.OwnerRefable, error) {
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

	if o.Spec.Credentials == nil {
		return envs
	}

	return envs
}

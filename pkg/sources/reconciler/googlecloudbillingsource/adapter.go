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

package googlecloudbillingsource

import (
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"

	"knative.dev/eventing/pkg/adapter/v2"
	"knative.dev/eventing/pkg/reconciler/source"
	"knative.dev/pkg/apis"

	commonv1alpha1 "github.com/triggermesh/triggermesh/pkg/apis/common/v1alpha1"
	"github.com/triggermesh/triggermesh/pkg/apis/sources/v1alpha1"
	common "github.com/triggermesh/triggermesh/pkg/reconciler"
	"github.com/triggermesh/triggermesh/pkg/reconciler/resource"
	"github.com/triggermesh/triggermesh/pkg/sources/cloudevents"
)

// adapterConfig contains properties used to configure the source's adapter.
// These are automatically populated by envconfig.
type adapterConfig struct {
	// Container image
	// Uses the adapter for Google Cloud Pub/Sub instead of a source-specific image.
	Image string `envconfig:"GOOGLECLOUDPUBSUBSOURCE_IMAGE" default:"gcr.io/triggermesh/googlecloudpubsubsource-adapter"`
	// Configuration accessor for logging/metrics/tracing
	configs source.ConfigAccessor
}

// Verify that Reconciler implements common.AdapterBuilder.
var _ common.AdapterBuilder[*appsv1.Deployment] = (*Reconciler)(nil)

// BuildAdapter implements common.AdapterBuilder.
func (r *Reconciler) BuildAdapter(src commonv1alpha1.Reconcilable, sinkURI *apis.URL) (*appsv1.Deployment, error) {
	typedSrc := src.(*v1alpha1.GoogleCloudBillingSource)

	return common.NewAdapterDeployment(src, sinkURI,
		resource.Image(r.adapterCfg.Image),

		resource.EnvVars(MakeAppEnv(typedSrc)...),
		resource.EnvVars(r.adapterCfg.configs.ToEnvVars()...),
	), nil
}

// MakeAppEnv extracts environment variables from the object.
// Exported to be used in external tools for local test environments.
func MakeAppEnv(o *v1alpha1.GoogleCloudBillingSource) []corev1.EnvVar {
	// we rely on the source's status to persist the ID of the Pub/Sub subscription
	var subsName string
	if sn := o.Status.Subscription; sn != nil {
		subsName = sn.String()
	}

	envVar := []corev1.EnvVar{
		{
			Name:  common.EnvGCloudPubSubSubscription,
			Value: subsName,
		}, {
			Name:  common.EnvCESource,
			Value: o.AsEventSource(),
		}, {
			Name:  common.EnvCEType,
			Value: v1alpha1.GoogleCloudBillingGenericEventType,
		}, {
			Name:  adapter.EnvConfigCEOverrides,
			Value: cloudevents.OverridesJSON(o.Spec.CloudEventOverrides),
		},
	}

	if o.Spec.Auth.ServiceAccountKey != nil {
		envVar = append(envVar, common.MaybeAppendValueFromEnvVar([]corev1.EnvVar{}, common.EnvGCloudSAKey, *o.Spec.Auth.ServiceAccountKey)...)
	}

	return envVar
}

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

package googlecloudrepositoriessource

import (
	"fmt"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/labels"

	"knative.dev/eventing/pkg/adapter/v2"
	"knative.dev/eventing/pkg/reconciler/source"
	"knative.dev/pkg/apis"
	"knative.dev/pkg/kmeta"

	"github.com/triggermesh/triggermesh/pkg/apis/sources/v1alpha1"
	"github.com/triggermesh/triggermesh/pkg/sources/cloudevents"
	"github.com/triggermesh/triggermesh/pkg/sources/reconciler/common"
	"github.com/triggermesh/triggermesh/pkg/sources/reconciler/common/resource"
)

// adapterConfig contains properties used to configure the source's adapter.
// These are automatically populated by envconfig.
type adapterConfig struct {
	// Container image
	// Uses the adapter for Google Cloud Pub/Sub instead of a source-specific image.
	Image string `envconfig:"GOOGLECLOUDPUBSUBSOURCE_IMAGE" default:"gcr.io/triggermesh-private/googlecloudpubsubsource-adapter"`
	// Configuration accessor for logging/metrics/tracing
	configs source.ConfigAccessor
}

// Verify that Reconciler implements common.AdapterDeploymentBuilder.
var _ common.AdapterDeploymentBuilder = (*Reconciler)(nil)

// BuildAdapter implements common.AdapterDeploymentBuilder.
func (r *Reconciler) BuildAdapter(src v1alpha1.EventSource, sinkURI *apis.URL) *appsv1.Deployment {
	typedSrc := src.(*v1alpha1.GoogleCloudRepositoriesSource)

	// we rely on the source's status to persist the ID of the Pub/Sub subscription
	var subsName string
	if sn := typedSrc.Status.Subscription; sn != nil {
		subsName = sn.String()
	}

	var authEnvs []corev1.EnvVar
	authEnvs = common.MaybeAppendValueFromEnvVar(authEnvs, common.EnvGCloudSAKey, typedSrc.Spec.ServiceAccountKey)

	ceOverridesStr := cloudevents.OverridesJSON(typedSrc.Spec.CloudEventOverrides,
		cloudevents.SetExtension(cloudevents.AttributeSource, src.AsEventSource()),
		cloudevents.SetExtension(cloudevents.AttributeType, v1alpha1.GoogleCloudRepositoriesGenericEventType),
	)

	return common.NewAdapterDeployment(src, sinkURI,
		resource.Image(r.adapterCfg.Image),

		resource.EnvVar(common.EnvGCloudPubSubSubscription, subsName),
		resource.EnvVars(authEnvs...),
		resource.EnvVar(adapter.EnvConfigCEOverrides, ceOverridesStr),
		resource.EnvVars(r.adapterCfg.configs.ToEnvVars()...),
	)
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

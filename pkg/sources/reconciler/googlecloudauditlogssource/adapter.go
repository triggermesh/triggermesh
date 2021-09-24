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

package googlecloudauditlogssource

import (
	"encoding/json"
	"fmt"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/labels"

	"knative.dev/eventing/pkg/adapter/v2"
	"knative.dev/eventing/pkg/reconciler/source"
	"knative.dev/pkg/apis"
	duckv1 "knative.dev/pkg/apis/duck/v1"
	"knative.dev/pkg/kmeta"

	"github.com/triggermesh/triggermesh/pkg/apis/sources/v1alpha1"
	"github.com/triggermesh/triggermesh/pkg/sources/reconciler/common"
	"github.com/triggermesh/triggermesh/pkg/sources/reconciler/common/resource"
)

const (
	envAuditLogsSubscription = "GCLOUD_PUBSUB_SUBSCRIPTION"
	envCloudEventSource      = "CE_SOURCE"
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
	typedSrc := src.(*v1alpha1.GoogleCloudAuditLogsSource)

	// we rely on the source's status to persist the ID of the Pub/Sub subscription
	var pubsubProject string
	var subsID string
	if sn := typedSrc.Status.Subscription; sn != nil {
		pubsubProject = sn.Project
		subsID = sn.Resource
	}

	var authEnvs []corev1.EnvVar
	authEnvs = common.MaybeAppendValueFromEnvVar(authEnvs, common.EnvGCloudSAKey, typedSrc.Spec.ServiceAccountKey)

	ceOverridesStr := ceOverridesJSON(typedSrc.Spec.CloudEventOverrides)

	return common.NewAdapterDeployment(src, sinkURI,
		resource.Image(r.adapterCfg.Image),

		// TODO(antoineco): remove usage of CE_SOURCE / CE overrides
		// and configure a message handler for Cloud Audit Logs events in
		// the adapter instead.
		resource.EnvVar(envCloudEventSource, typedSrc.AsEventSource()),
		resource.EnvVar(adapter.EnvConfigCEOverrides, ceOverridesStr),

		resource.EnvVar(common.EnvGCloudProject, pubsubProject),
		resource.EnvVar(envAuditLogsSubscription, subsID),
		resource.EnvVars(authEnvs...),
		resource.EnvVars(r.adapterCfg.configs.ToEnvVars()...),
	)
}

// RBACOwners implements common.AdapterDeploymentBuilder.
func (r *Reconciler) RBACOwners(namespace string) ([]kmeta.OwnerRefable, error) {
	srcs, err := r.srcLister(namespace).List(labels.Everything())
	if err != nil {
		return nil, fmt.Errorf("listing objects from cache: %w", err)
	}

	ownerRefables := make([]kmeta.OwnerRefable, len(srcs))
	for i := range srcs {
		ownerRefables[i] = srcs[i]
	}

	return ownerRefables, nil
}

// ceOverridesJSON returns the source's CloudEvent overrides as a JSON object.
func ceOverridesJSON(ceo *duckv1.CloudEventOverrides) string {
	ceo = setCETypeOverride(ceo)

	var ceoStr string
	if b, err := json.Marshal(ceo); err == nil {
		ceoStr = string(b)
	}

	return ceoStr
}

// setCETypeOverride sets an override on the CloudEvent "type" attribute that
// matches event payloads sent by Google Cloud Audit Logs.
func setCETypeOverride(ceo *duckv1.CloudEventOverrides) *duckv1.CloudEventOverrides {
	if ceo == nil {
		ceo = &duckv1.CloudEventOverrides{}
	}

	ext := &ceo.Extensions
	if *ext == nil {
		*ext = make(map[string]string, 1)
	}

	if _, isSet := (*ext)["type"]; !isSet {
		(*ext)["type"] = v1alpha1.GoogleCloudAuditLogsGenericEventType
	}

	return ceo
}

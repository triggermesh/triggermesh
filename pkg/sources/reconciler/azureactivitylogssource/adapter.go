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

package azureactivitylogssource

import (
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"

	"knative.dev/eventing/pkg/adapter/v2"
	"knative.dev/eventing/pkg/reconciler/source"
	"knative.dev/pkg/apis"

	commonv1alpha1 "github.com/triggermesh/triggermesh/pkg/apis/common/v1alpha1"
	"github.com/triggermesh/triggermesh/pkg/apis/sources"
	"github.com/triggermesh/triggermesh/pkg/apis/sources/v1alpha1"
	common "github.com/triggermesh/triggermesh/pkg/reconciler"
	"github.com/triggermesh/triggermesh/pkg/reconciler/resource"
	"github.com/triggermesh/triggermesh/pkg/sources/cloudevents"
)

const healthPortName = "health"

const defaultActivityLogsEventHubName = "insights-activity-logs"

// adapterConfig contains properties used to configure the source's adapter.
// These are automatically populated by envconfig.
type adapterConfig struct {
	// Container image
	// Uses the adapter for Azure Event Hubs instead of a source-specific image.
	Image string `envconfig:"AZUREEVENTHUBSOURCE_IMAGE" default:"gcr.io/triggermesh/azureeventhubsource-adapter"`
	// Configuration accessor for logging/metrics/tracing
	configs source.ConfigAccessor
}

// Verify that Reconciler implements common.AdapterDeploymentBuilder.
var _ common.AdapterBuilder[*appsv1.Deployment] = (*Reconciler)(nil)

// BuildAdapter implements common.AdapterDeploymentBuilder.
func (r *Reconciler) BuildAdapter(src commonv1alpha1.Reconcilable, sinkURI *apis.URL) *appsv1.Deployment {
	typedSrc := src.(*v1alpha1.AzureActivityLogsSource)

	hubNamespaceID := typedSrc.Spec.Destination.EventHubs.NamespaceID

	eventHubName := defaultActivityLogsEventHubName
	if hubName := typedSrc.Spec.Destination.EventHubs.HubName; hubName != nil && *hubName != "" {
		eventHubName = *hubName
	}

	var hubEnvs []corev1.EnvVar
	if spAuth := typedSrc.Spec.Auth.ServicePrincipal; spAuth != nil {
		hubEnvs = common.MaybeAppendValueFromEnvVar(hubEnvs, common.EnvAADTenantID, spAuth.TenantID)
		hubEnvs = common.MaybeAppendValueFromEnvVar(hubEnvs, common.EnvAADClientID, spAuth.ClientID)
		hubEnvs = common.MaybeAppendValueFromEnvVar(hubEnvs, common.EnvAADClientSecret, spAuth.ClientSecret)
	}

	ceType := v1alpha1.AzureEventType(sources.AzureServiceMonitor, v1alpha1.AzureActivityLogsActivityLogEventType)

	ceOverridesStr := cloudevents.OverridesJSON(typedSrc.Spec.CloudEventOverrides)

	return common.NewAdapterDeployment(src, sinkURI,
		resource.Image(r.adapterCfg.Image),

		resource.EnvVar(common.EnvHubResourceID, makeEventHubID(&hubNamespaceID, eventHubName)),
		resource.EnvVar(common.EnvHubNamespace, hubNamespaceID.ResourceName),
		resource.EnvVar(common.EnvHubName, eventHubName),
		resource.EnvVars(hubEnvs...),
		resource.EnvVar(common.EnvCESource, src.(commonv1alpha1.EventSource).AsEventSource()),
		resource.EnvVar(common.EnvCEType, ceType),
		resource.EnvVar(adapter.EnvConfigCEOverrides, ceOverridesStr),
		resource.EnvVars(r.adapterCfg.configs.ToEnvVars()...),

		resource.Port(healthPortName, 8080),
		resource.StartupProbe("/health", healthPortName),
	)
}

// makeEventHubID returns the Resource ID of an Event Hubs instance based on
// the given Event Hubs namespace and Hub name.
func makeEventHubID(namespaceID *v1alpha1.AzureResourceID, hubName string) string {
	hubID := *namespaceID
	hubID.Namespace = namespaceID.ResourceName
	hubID.ResourceType = resourceTypeEventHubs
	hubID.ResourceName = hubName
	return hubID.String()
}

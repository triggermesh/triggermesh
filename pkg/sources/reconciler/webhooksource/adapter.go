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

package webhooksource

import (
	"strings"

	corev1 "k8s.io/api/core/v1"

	"knative.dev/eventing/pkg/reconciler/source"
	"knative.dev/pkg/apis"
	servingv1 "knative.dev/serving/pkg/apis/serving/v1"

	commonv1alpha1 "github.com/triggermesh/triggermesh/pkg/apis/common/v1alpha1"
	"github.com/triggermesh/triggermesh/pkg/apis/sources/v1alpha1"
	common "github.com/triggermesh/triggermesh/pkg/reconciler"
	"github.com/triggermesh/triggermesh/pkg/reconciler/resource"
)

const (
	envWebhookEventType                    = "WEBHOOK_EVENT_TYPE"
	envWebhookEventSource                  = "WEBHOOK_EVENT_SOURCE"
	envWebhookEventExtensionAttributesFrom = "WEBHOOK_EVENT_EXTENSION_ATTRIBUTES_FROM"
	envWebhookBasicAuthUsername            = "WEBHOOK_BASICAUTH_USERNAME"
	envWebhookBasicAuthPassword            = "WEBHOOK_BASICAUTH_PASSWORD"
	envCorsAllowOrigin                     = "WEBHOOK_CORS_ALLOW_ORIGIN"
)

// adapterConfig contains properties used to configure the adapter.
// These are automatically populated by envconfig.
type adapterConfig struct {
	// Container image
	Image string `default:"gcr.io/triggermesh/webhook-adapter"`

	// Configuration accessor for logging/metrics/tracing
	configs source.ConfigAccessor
}

// Verify that Reconciler implements common.AdapterBuilder.
var _ common.AdapterBuilder[*servingv1.Service] = (*Reconciler)(nil)

// BuildAdapter implements common.AdapterBuilder.
func (r *Reconciler) BuildAdapter(src commonv1alpha1.Reconcilable, sinkURI *apis.URL) (*servingv1.Service, error) {
	typedSrc := src.(*v1alpha1.WebhookSource)

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

		resource.EnvVars(MakeAppEnv(typedSrc)...),
		resource.EnvVars(r.adapterCfg.configs.ToEnvVars()...),
	), nil
}

// MakeAppEnv extracts environment variables from the object.
// Exported to be used in external tools for local test environments.
func MakeAppEnv(src *v1alpha1.WebhookSource) []corev1.EnvVar {
	envs := []corev1.EnvVar{{
		Name:  envWebhookEventType,
		Value: src.Spec.EventType,
	}, {
		Name:  envWebhookEventSource,
		Value: src.AsEventSource(),
	}}

	if extAttributes := src.Spec.EventExtensionAttributes; extAttributes != nil {
		if len(extAttributes.From) != 0 {
			envs = append(envs, corev1.EnvVar{
				Name:  envWebhookEventExtensionAttributesFrom,
				Value: strings.Join(extAttributes.From, ","),
			})
		}
	}

	if origin := src.Spec.CORSAllowOrigin; origin != nil {
		envs = append(envs, corev1.EnvVar{
			Name:  envCorsAllowOrigin,
			Value: *origin,
		})
	}

	if user := src.Spec.BasicAuthUsername; user != nil {
		envs = append(envs, corev1.EnvVar{
			Name:  envWebhookBasicAuthUsername,
			Value: *user,
		})
	}

	if passw := src.Spec.BasicAuthPassword; passw != nil {
		envs = common.MaybeAppendValueFromEnvVar(envs,
			envWebhookBasicAuthPassword, *passw,
		)
	}

	return envs
}

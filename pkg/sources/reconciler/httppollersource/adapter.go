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

package httppollersource

import (
	"sort"
	"strconv"
	"strings"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"

	"knative.dev/eventing/pkg/reconciler/source"
	"knative.dev/pkg/apis"

	commonv1alpha1 "github.com/triggermesh/triggermesh/pkg/apis/common/v1alpha1"
	"github.com/triggermesh/triggermesh/pkg/apis/sources/v1alpha1"
	common "github.com/triggermesh/triggermesh/pkg/reconciler"
	"github.com/triggermesh/triggermesh/pkg/reconciler/resource"
)

const (
	envHTTPPollerEventType         = "HTTPPOLLER_EVENT_TYPE"
	envHTTPPollerEventSource       = "HTTPPOLLER_EVENT_SOURCE"
	envHTTPPollerEndpoint          = "HTTPPOLLER_ENDPOINT"
	envHTTPPollerMethod            = "HTTPPOLLER_METHOD"
	envHTTPPollerSkipVerify        = "HTTPPOLLER_SKIP_VERIFY"
	envHTTPPollerCACertificate     = "HTTPPOLLER_CA_CERTIFICATE"
	envHTTPPollerBasicAuthUsername = "HTTPPOLLER_BASICAUTH_USERNAME"
	envHTTPPollerBasicAuthPassword = "HTTPPOLLER_BASICAUTH_PASSWORD"
	envHTTPPollerHeaders           = "HTTPPOLLER_HEADERS"
	envHTTPPollerInterval          = "HTTPPOLLER_INTERVAL"
)

// adapterConfig contains properties used to configure the source's adapter.
// These are automatically populated by envconfig.
type adapterConfig struct {
	// Container image
	Image string `default:"gcr.io/triggermesh/httppollersource-adapter"`

	// Configuration accessor for logging/metrics/tracing
	configs source.ConfigAccessor
}

// BuildAdapter implements common.AdapterBuilder.
func (r *Reconciler) BuildAdapter(src commonv1alpha1.Reconcilable, sinkURI *apis.URL) (*appsv1.Deployment, error) {
	typedSrc := src.(*v1alpha1.HTTPPollerSource)

	return common.NewAdapterDeployment(src, sinkURI,
		resource.Image(r.adapterCfg.Image),

		resource.EnvVars(MakeAppEnv(typedSrc)...),
		resource.EnvVars(r.adapterCfg.configs.ToEnvVars()...),
	), nil
}

// MakeAppEnv extracts environment variables from the object.
// Exported to be used in external tools for local test environments.
func MakeAppEnv(src *v1alpha1.HTTPPollerSource) []corev1.EnvVar {
	skipVerify := false
	if src.Spec.SkipVerify != nil {
		skipVerify = *src.Spec.SkipVerify
	}

	envs := []corev1.EnvVar{{
		Name:  envHTTPPollerEventType,
		Value: src.Spec.EventType,
	}, {
		Name:  envHTTPPollerEventSource,
		Value: src.AsEventSource(),
	}, {
		Name:  envHTTPPollerEndpoint,
		Value: src.Spec.Endpoint.String(),
	}, {
		Name:  envHTTPPollerMethod,
		Value: src.Spec.Method,
	}, {
		Name:  envHTTPPollerSkipVerify,
		Value: strconv.FormatBool(skipVerify),
	}, {
		Name:  envHTTPPollerInterval,
		Value: src.Spec.Interval.String(),
	}}

	if src.Spec.Headers != nil {
		headers := make([]string, 0, len(src.Spec.Headers))
		for k, v := range src.Spec.Headers {
			headers = append(headers, k+":"+v)
		}
		sort.Strings(headers)

		envs = append(envs, corev1.EnvVar{
			Name:  envHTTPPollerHeaders,
			Value: strings.Join(headers, ","),
		})
	}

	if user := src.Spec.BasicAuthUsername; user != nil {
		envs = append(envs, corev1.EnvVar{
			Name:  envHTTPPollerBasicAuthUsername,
			Value: *user,
		})
	}

	if passw := src.Spec.BasicAuthPassword; passw != nil {
		envs = common.MaybeAppendValueFromEnvVar(envs,
			envHTTPPollerBasicAuthPassword, *passw,
		)
	}

	if src.Spec.CACertificate != nil {
		envs = append(envs, corev1.EnvVar{
			Name:  envHTTPPollerCACertificate,
			Value: *src.Spec.CACertificate,
		})
	}

	return envs
}

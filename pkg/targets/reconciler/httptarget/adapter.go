/*
Copyright (c) 2021 TriggerMesh Inc.

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

package httptarget

import (
	"sort"
	"strconv"
	"strings"

	corev1 "k8s.io/api/core/v1"

	"knative.dev/eventing/pkg/reconciler/source"
	"knative.dev/pkg/kmeta"
	servingv1 "knative.dev/serving/pkg/apis/serving/v1"

	"github.com/triggermesh/triggermesh/pkg/apis/targets/v1alpha1"
	pkgreconciler "github.com/triggermesh/triggermesh/pkg/targets/reconciler"
	"github.com/triggermesh/triggermesh/pkg/targets/reconciler/resources"
)

const (
	adapterName = "httptarget"

	envHTTPEventType         = "HTTP_EVENT_TYPE"
	envHTTPEventSource       = "HTTP_EVENT_SOURCE"
	envHTTPURL               = "HTTP_URL"
	envHTTPMethod            = "HTTP_METHOD"
	envHTTPSkipVerify        = "HTTP_SKIP_VERIFY"
	envHTTPCACertificate     = "HTTP_CA_CERTIFICATE"
	envHTTPHeaders           = "HTTP_HEADERS"
	envHTTPBasicAuthUsername = "HTTP_BASICAUTH_USERNAME"
	envHTTPBasicAuthPassword = "HTTP_BASICAUTH_PASSWORD"
	envHTTPOAuthClientID     = "HTTP_OAUTH_CLIENT_ID"
	envHTTPOAuthClientSecret = "HTTP_OAUTH_CLIENT_SECRET"
	envHTTPOAuthTokenURL     = "HTTP_OAUTH_TOKEN_URL"
	envHTTPOAuthScopes       = "HTTP_OAUTH_SCOPE"
)

// adapterConfig contains properties used to configure the target's adapter.
// Public fields are automatically populated by envconfig.
type adapterConfig struct {
	// Configuration accessor for logging/metrics/tracing
	configs source.ConfigAccessor
	// Container image
	Image string `default:"gcr.io/triggermesh/http-target-adapter"`
}

// makeAdapterKnService returns a Knative Service object for the target's adapter.
func makeAdapterKnService(o *v1alpha1.HTTPTarget, cfg *adapterConfig) *servingv1.Service {
	envApp := makeCommonAppEnv(o)

	ksvcLabels := pkgreconciler.MakeAdapterLabels(adapterName, o.Name)
	podLabels := pkgreconciler.MakeAdapterLabels(adapterName, o.Name)
	name := kmeta.ChildName(adapterName+"-", o.Name)
	envSvc := pkgreconciler.MakeServiceEnv(o.Name, o.Namespace)
	envObs := pkgreconciler.MakeObsEnv(cfg.configs)
	envs := append(envSvc, envApp...)
	envs = append(envs, envObs...)

	return resources.MakeKService(o.Namespace, name, cfg.Image,
		resources.KsvcLabels(ksvcLabels),
		resources.KsvcLabelVisibilityClusterLocal,
		resources.KsvcPodLabels(podLabels),
		resources.KsvcOwner(o),
		resources.KsvcPodEnvVars(envs))
}

func makeCommonAppEnv(o *v1alpha1.HTTPTarget) []corev1.EnvVar {
	skipVerify := false
	if o.Spec.SkipVerify != nil {
		skipVerify = *o.Spec.SkipVerify
	}

	eventSource := o.Spec.Response.EventSource
	if eventSource == "" {
		eventSource = o.Namespace + "." + o.Name
	}

	env := []corev1.EnvVar{
		{
			Name:  envHTTPEventType,
			Value: o.Spec.Response.EventType,
		}, {
			Name:  envHTTPEventSource,
			Value: eventSource,
		}, {
			Name:  envHTTPURL,
			Value: o.Spec.Endpoint.String(),
		}, {
			Name:  envHTTPMethod,
			Value: o.Spec.Method,
		}, {
			Name:  envHTTPSkipVerify,
			Value: strconv.FormatBool(skipVerify),
		},
	}

	// Headers environment format is dictated by https://github.com/kelseyhightower/envconfig
	// Each key and value separated by colon, elements by commas.
	// To avoid map comparison issues when reconciling, header keys are ordered first, then
	// serialized to the environment variable.

	if o.Spec.Headers != nil {
		headers := make([]string, 0, len(o.Spec.Headers))
		for k := range o.Spec.Headers {
			headers = append(headers, k)
		}
		sort.Strings(headers)

		for i, k := range headers {
			headers[i] = headers[i] + ":" + o.Spec.Headers[k]
		}
		env = append(env, corev1.EnvVar{
			Name:  envHTTPHeaders,
			Value: strings.Join(headers, ","),
		})
	}

	if o.Spec.CACertificate != nil {
		env = append(env, corev1.EnvVar{
			Name:  envHTTPCACertificate,
			Value: *o.Spec.CACertificate,
		})
	}

	if o.Spec.BasicAuthUsername != nil {
		env = append(env, corev1.EnvVar{
			Name:  envHTTPBasicAuthUsername,
			Value: *o.Spec.BasicAuthUsername,
		})
	}

	if o.Spec.BasicAuthPassword.SecretKeyRef != nil {
		env = append(env, corev1.EnvVar{
			Name: envHTTPBasicAuthPassword,
			ValueFrom: &corev1.EnvVarSource{
				SecretKeyRef: o.Spec.BasicAuthPassword.SecretKeyRef,
			},
		})
	}

	if o.Spec.OAuthClientID != nil {
		env = append(env, corev1.EnvVar{
			Name:  envHTTPOAuthClientID,
			Value: *o.Spec.OAuthClientID,
		})
	}

	if o.Spec.OAuthClientSecret.SecretKeyRef != nil {
		env = append(env, corev1.EnvVar{
			Name: envHTTPOAuthClientSecret,
			ValueFrom: &corev1.EnvVarSource{
				SecretKeyRef: o.Spec.OAuthClientSecret.SecretKeyRef,
			},
		})
	}

	if o.Spec.OAuthTokenURL != nil {
		env = append(env, corev1.EnvVar{
			Name:  envHTTPOAuthTokenURL,
			Value: *o.Spec.OAuthTokenURL,
		})
	}

	if o.Spec.OAuthScopes != nil {
		env = append(env, corev1.EnvVar{
			Name:  envHTTPOAuthScopes,
			Value: strings.Join(*o.Spec.OAuthScopes, ","),
		})
	}

	return env
}

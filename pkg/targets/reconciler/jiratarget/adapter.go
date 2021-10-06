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

package jiratarget

import (
	corev1 "k8s.io/api/core/v1"

	"knative.dev/eventing/pkg/reconciler/source"
	"knative.dev/pkg/kmeta"
	servingv1 "knative.dev/serving/pkg/apis/serving/v1"

	"github.com/triggermesh/triggermesh/pkg/apis/targets/v1alpha1"
	pkgreconciler "github.com/triggermesh/triggermesh/pkg/targets/reconciler"
	"github.com/triggermesh/triggermesh/pkg/targets/reconciler/resources"
)

const (
	adapterName = "jiratarget"

	envJiraAuthUser  = "JIRA_AUTH_USER"
	envJiraAuthToken = "JIRA_AUTH_TOKEN"
	envJiraURL       = "JIRA_URL"
)

// adapterConfig contains properties used to configure the target's adapter.
// Public fields are automatically populated by envconfig.
type adapterConfig struct {
	// Configuration accessor for logging/metrics/tracing
	configs source.ConfigAccessor
	// Container image
	Image string `default:"gcr.io/triggermesh/jiratarget-adapter"`
}

// makeAdapterKnService returns a Knative Service object for the target's adapter.
func makeAdapterKnService(o *v1alpha1.JiraTarget, cfg *adapterConfig) *servingv1.Service {
	envApp := makeCommonAppEnv(o)

	ksvcLabels := pkgreconciler.MakeAdapterLabels(adapterName, o.Name)
	podLabels := pkgreconciler.MakeAdapterLabels(adapterName, o.Name)
	name := kmeta.ChildName(adapterName+"-", o.Name)
	envSvc := pkgreconciler.MakeServiceEnv(o.Name, o.Namespace)
	envObs := pkgreconciler.MakeObsEnv(cfg.configs)
	envs := append(envSvc, append(envApp, envObs...)...)

	return resources.MakeKService(o.Namespace, name, cfg.Image,
		resources.KsvcLabels(ksvcLabels),
		resources.KsvcLabelVisibilityClusterLocal,
		resources.KsvcPodLabels(podLabels),
		resources.KsvcOwner(o),
		resources.KsvcPodEnvVars(envs))
}

func makeCommonAppEnv(o *v1alpha1.JiraTarget) []corev1.EnvVar {
	return []corev1.EnvVar{
		{
			Name:  envJiraAuthUser,
			Value: o.Spec.Auth.User,
		},
		{
			Name: envJiraAuthToken,
			ValueFrom: &corev1.EnvVarSource{
				SecretKeyRef: o.Spec.Auth.Token.SecretKeyRef,
			},
		},
		{
			Name:  envJiraURL,
			Value: o.Spec.URL,
		},
		{
			Name:  pkgreconciler.EnvBridgeID,
			Value: pkgreconciler.GetStatefulBridgeID(o),
		},
	}
}

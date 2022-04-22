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

package awscomprehendtarget

import (
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
	envRegion              = "COMPREHEND_REGION"
	envLanguage            = "COMPREHEND_LANGUAGE"
	envEventsPayloadPolicy = "EVENTS_PAYLOAD_POLICY"
)

// adapterConfig contains properties used to configure the target's adapter.
// Public fields are automatically populated by envconfig.
type adapterConfig struct {
	// Configuration accessor for logging/metrics/tracing
	obsConfig source.ConfigAccessor
	// Container image
	Image string `default:"gcr.io/triggermesh/awscomprehendtarget-adapter"`
}

// Verify that Reconciler implements common.AdapterServiceBuilder.
var _ common.AdapterBuilder[*servingv1.Service] = (*Reconciler)(nil)

// BuildAdapter implements common.AdapterServiceBuilder.
func (r *Reconciler) BuildAdapter(trg commonv1alpha1.Reconcilable, _ *apis.URL) *servingv1.Service {
	typedTrg := trg.(*v1alpha1.AWSComprehendTarget)

	return common.NewAdapterKnService(trg, nil,
		resource.Image(r.adapterCfg.Image),
		resource.EnvVars(makeAppEnv(typedTrg)...),
		resource.EnvVars(r.adapterCfg.obsConfig.ToEnvVars()...),
	)
}

func makeAppEnv(o *v1alpha1.AWSComprehendTarget) []corev1.EnvVar {
	env := []corev1.EnvVar{
		{
			Name:  envRegion,
			Value: o.Spec.Region,
		}, {
			Name:  envLanguage,
			Value: o.Spec.Language,
		}, {
			Name: common.EnvAccessKeyID,
			ValueFrom: &corev1.EnvVarSource{
				SecretKeyRef: o.Spec.AWSApiKey.SecretKeyRef,
			},
		}, {
			Name: common.EnvSecretAccessKey,
			ValueFrom: &corev1.EnvVarSource{
				SecretKeyRef: o.Spec.AWSApiSecret.SecretKeyRef,
			},
		},
	}

	if o.Spec.EventOptions != nil && o.Spec.EventOptions.PayloadPolicy != nil {
		env = append(env, corev1.EnvVar{
			Name:  envEventsPayloadPolicy,
			Value: string(*o.Spec.EventOptions.PayloadPolicy),
		})
	}

	return env
}

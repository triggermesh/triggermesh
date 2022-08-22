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

package infratarget

import (
	"strconv"

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
	envInfraScriptCode         = "INFRA_SCRIPT_CODE"
	envInfraScriptTimeout      = "INFRA_SCRIPT_TIMEOUT"
	envInfraStateHeadersPolicy = "INFRA_STATE_HEADERS_POLICY"
	envInfraStateBridge        = "INFRA_STATE_BRIDGE"
	envInfraTypeLoopProtection = "INFRA_TYPE_LOOP_PROTECTION"
)

// adapterConfig contains properties used to configure the target's adapter.
// Public fields are automatically populated by envconfig.
type adapterConfig struct {
	// Configuration accessor for logging/metrics/tracing
	obsConfig source.ConfigAccessor
	// Container image
	Image string `default:"gcr.io/triggermesh/infratarget-adapter"`
}

// Verify that Reconciler implements common.AdapterBuilder.
var _ common.AdapterBuilder[*servingv1.Service] = (*Reconciler)(nil)

// BuildAdapter implements common.AdapterBuilder.
func (r *Reconciler) BuildAdapter(trg commonv1alpha1.Reconcilable, _ *apis.URL) (*servingv1.Service, error) {
	typedTrg := trg.(*v1alpha1.InfraTarget)

	return common.NewAdapterKnService(trg, nil,
		resource.Image(r.adapterCfg.Image),
		resource.EnvVars(MakeAppEnv(typedTrg)...),
		resource.EnvVars(r.adapterCfg.obsConfig.ToEnvVars()...),
	), nil
}

// MakeAppEnv extracts environment variables from the object.
// Exported to be used in external tools for local test environments.
func MakeAppEnv(o *v1alpha1.InfraTarget) []corev1.EnvVar {
	var env []corev1.EnvVar

	if o.Spec.Script != nil {
		env = append(env, corev1.EnvVar{
			Name:  envInfraScriptCode,
			Value: o.Spec.Script.Code,
		})

		if o.Spec.Script.Timeout != nil && *o.Spec.Script.Timeout > 0 {
			env = append(env, corev1.EnvVar{
				Name:  envInfraScriptTimeout,
				Value: strconv.Itoa(*o.Spec.Script.Timeout),
			})
		}
	}

	if o.Spec.State != nil {

		if o.Spec.State.HeadersPolicy != nil {
			env = append(env, corev1.EnvVar{
				Name:  envInfraStateHeadersPolicy,
				Value: string(*o.Spec.State.HeadersPolicy),
			})
		}

		if o.Spec.State.Bridge != nil {
			env = append(env, corev1.EnvVar{
				Name:  envInfraStateBridge,
				Value: *o.Spec.State.Bridge,
			})
		}
	}

	if o.Spec.TypeLoopProtection != nil {
		env = append(env, corev1.EnvVar{
			Name:  envInfraTypeLoopProtection,
			Value: strconv.FormatBool(*o.Spec.TypeLoopProtection),
		})
	}

	return env
}

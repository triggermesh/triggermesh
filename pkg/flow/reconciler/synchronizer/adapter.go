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

package synchronizer

import (
	"strconv"

	corev1 "k8s.io/api/core/v1"

	"knative.dev/eventing/pkg/reconciler/source"
	"knative.dev/pkg/apis"
	servingv1 "knative.dev/serving/pkg/apis/serving/v1"

	commonv1alpha1 "github.com/triggermesh/triggermesh/pkg/apis/common/v1alpha1"
	"github.com/triggermesh/triggermesh/pkg/apis/flow/v1alpha1"
	common "github.com/triggermesh/triggermesh/pkg/reconciler"
	"github.com/triggermesh/triggermesh/pkg/reconciler/resource"
)

// adapterConfig contains properties used to configure the target's adapter.
// Public fields are automatically populated by envconfig.
type adapterConfig struct {
	// Configuration accessor for logging/metrics/tracing
	obsConfig source.ConfigAccessor
	// Container image
	Image string `default:"gcr.io/triggermesh/synchronizer-adapter"`
}

// Verify that Reconciler implements common.AdapterBuilder.
var _ common.AdapterBuilder[*servingv1.Service] = (*Reconciler)(nil)

// BuildAdapter implements common.AdapterBuilder.
func (r *Reconciler) BuildAdapter(trg commonv1alpha1.Reconcilable, sinkURI *apis.URL) (*servingv1.Service, error) {
	typedTrg := trg.(*v1alpha1.Synchronizer)

	return common.NewAdapterKnService(trg, sinkURI,
		resource.Image(r.adapterCfg.Image),
		resource.EnvVars(MakeAppEnv(typedTrg)...),
		resource.EnvVars(r.adapterCfg.obsConfig.ToEnvVars()...),
	), nil
}

// MakeAppEnv extracts environment variables from the object.
// Exported to be used in external tools for local test environments.
func MakeAppEnv(o *v1alpha1.Synchronizer) []corev1.EnvVar {
	env := []corev1.EnvVar{
		{
			Name:  common.EnvBridgeID,
			Value: common.GetStatefulBridgeID(o),
		},
		{
			Name:  "CORRELATION_KEY",
			Value: o.Spec.CorrelationKey.Attribute,
		},
		{
			Name:  "RESPONSE_WAIT_TIMEOUT",
			Value: o.Spec.Response.Timeout.String(),
		},
	}

	if o.Spec.CorrelationKey.Length != 0 {
		env = append(env, corev1.EnvVar{
			Name:  "CORRELATION_KEY_LENGTH",
			Value: strconv.Itoa(o.Spec.CorrelationKey.Length),
		})
	}

	return env
}

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

package awseventbridgetarget

import (
	"strconv"

	corev1 "k8s.io/api/core/v1"

	"knative.dev/eventing/pkg/reconciler/source"
	"knative.dev/pkg/apis"
	"knative.dev/pkg/kmeta"
	servingv1 "knative.dev/serving/pkg/apis/serving/v1"

	commonv1alpha1 "github.com/triggermesh/triggermesh/pkg/apis/common/v1alpha1"
	"github.com/triggermesh/triggermesh/pkg/apis/targets/v1alpha1"
	common "github.com/triggermesh/triggermesh/pkg/reconciler"
	"github.com/triggermesh/triggermesh/pkg/reconciler/resource"
)

// adapterConfig contains properties used to configure the target's adapter.
// Public fields are automatically populated by envconfig.
type adapterConfig struct {
	// Configuration accessor for logging/metrics/tracing
	obsConfig source.ConfigAccessor
	// Container image
	Image string `default:"gcr.io/triggermesh/awseventbridgetarget-adapter"`
}

// Verify that Reconciler implements common.AdapterServiceBuilder.
var _ common.AdapterServiceBuilder = (*Reconciler)(nil)

// BuildAdapter implements common.AdapterServiceBuilder.
func (r *Reconciler) BuildAdapter(trg commonv1alpha1.Reconcilable, _ *apis.URL) *servingv1.Service {
	typedTrg := trg.(*v1alpha1.AWSEventBridgeTarget)

	return common.NewAdapterKnService(trg, nil,
		resource.Image(r.adapterCfg.Image),
		resource.EnvVars(makeAppEnv(typedTrg)...),
		resource.EnvVars(r.adapterCfg.obsConfig.ToEnvVars()...),
	)
}

func makeAppEnv(o *v1alpha1.AWSEventBridgeTarget) []corev1.EnvVar {
	envs := []corev1.EnvVar{
		{
			Name: common.EnvAccessKeyID,
			ValueFrom: &corev1.EnvVarSource{
				SecretKeyRef: o.Spec.AWSApiKey.SecretKeyRef,
			},
		}, {
			Name: common.EnvSecretAccessKey,
			ValueFrom: &corev1.EnvVarSource{
				SecretKeyRef: o.Spec.AWSApiSecret.SecretKeyRef,
			},
		}, {
			Name:  common.EnvARN,
			Value: o.Spec.ARN,
		}, {
			Name:  "AWS_DISCARD_CE_CONTEXT",
			Value: strconv.FormatBool(o.Spec.DiscardCEContext),
		},
	}

	return envs
}

// RBACOwners implements common.AdapterServiceBuilder.
func (r *Reconciler) RBACOwners(trg commonv1alpha1.Reconcilable) ([]kmeta.OwnerRefable, error) {
	return common.RBACOwners[*v1alpha1.AWSEventBridgeTarget](r.trgLister(trg.GetNamespace()))
}

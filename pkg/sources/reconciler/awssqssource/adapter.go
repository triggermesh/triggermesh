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

package awssqssource

import (
	appsv1 "k8s.io/api/apps/v1"
	kr "k8s.io/apimachinery/pkg/api/resource"

	"knative.dev/eventing/pkg/reconciler/source"
	"knative.dev/pkg/apis"

	commonv1alpha1 "github.com/triggermesh/triggermesh/pkg/apis/common/v1alpha1"
	"github.com/triggermesh/triggermesh/pkg/apis/sources/v1alpha1"
	common "github.com/triggermesh/triggermesh/pkg/reconciler"
	"github.com/triggermesh/triggermesh/pkg/reconciler/resource"
	"github.com/triggermesh/triggermesh/pkg/sources/reconciler"
)

const healthPortName = "health"

// adapterConfig contains properties used to configure the source's adapter.
// These are automatically populated by envconfig.
type adapterConfig struct {
	// Container image
	Image string `default:"gcr.io/triggermesh/awssqssource"`
	// Configuration accessor for logging/metrics/tracing
	configs source.ConfigAccessor
}

// Verify that Reconciler implements common.AdapterDeploymentBuilder.
var _ common.AdapterDeploymentBuilder = (*Reconciler)(nil)

// BuildAdapter implements common.AdapterDeploymentBuilder.
func (r *Reconciler) BuildAdapter(src commonv1alpha1.Reconcilable, sinkURI *apis.URL) *appsv1.Deployment {
	typedSrc := src.(*v1alpha1.AWSSQSSource)

	return common.NewAdapterDeployment(src, sinkURI,
		resource.Image(r.adapterCfg.Image),

		resource.EnvVar(common.EnvARN, typedSrc.Spec.ARN.String()),
		resource.EnvVars(reconciler.MakeAWSAuthEnvVars(typedSrc.Spec.Auth)...),
		resource.EnvVars(reconciler.MakeAWSEndpointEnvVars(typedSrc.Spec.Endpoint)...),
		resource.EnvVar(common.EnvNamespace, src.GetNamespace()),
		resource.EnvVar(common.EnvName, src.GetName()),
		resource.EnvVars(r.adapterCfg.configs.ToEnvVars()...),

		resource.Port(healthPortName, 8080),

		resource.StartupProbe("/health", healthPortName),

		// CPU throttling can be observed below a limit of 1,
		// although the CPU usage under load remains below 400m.
		resource.Requests(
			kr.NewMilliQuantity(90, kr.DecimalSI),     // 90m
			kr.NewQuantity(1024*1024*30, kr.BinarySI), // 30Mi
		),
		resource.Limits(
			kr.NewMilliQuantity(1000, kr.DecimalSI),   // 1
			kr.NewQuantity(1024*1024*45, kr.BinarySI), // 45Mi
		),
	)
}

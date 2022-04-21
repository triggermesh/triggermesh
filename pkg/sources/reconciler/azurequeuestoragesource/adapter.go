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

package azurequeuestoragesource

import (
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"

	"knative.dev/eventing/pkg/reconciler/source"
	"knative.dev/pkg/apis"

	commonv1alpha1 "github.com/triggermesh/triggermesh/pkg/apis/common/v1alpha1"
	"github.com/triggermesh/triggermesh/pkg/apis/sources/v1alpha1"
	common "github.com/triggermesh/triggermesh/pkg/reconciler"
	"github.com/triggermesh/triggermesh/pkg/reconciler/resource"
)

// adapterConfig contains properties used to configure the adapter.
// These are automatically populated by envconfig.
type adapterConfig struct {
	// Container image
	Image string `default:"gcr.io/triggermesh/azurequeuestoragesource-adapter"`
	// Configuration accessor for logging/metrics/tracing
	configs source.ConfigAccessor
}

// Verify that Reconciler implements common.AdapterDeploymentBuilder.
var _ common.AdapterDeploymentBuilder = (*Reconciler)(nil)

// BuildAdapter implements common.AdapterDeploymentBuilder.
func (r *Reconciler) BuildAdapter(src commonv1alpha1.Reconcilable, sinkURI *apis.URL) *appsv1.Deployment {
	typedSrc := src.(*v1alpha1.AzureQueueStorageSource)

	storageQueueEnvs := []corev1.EnvVar{}

	storageQueueEnvs = common.MaybeAppendValueFromEnvVar(storageQueueEnvs, "AZURE_ACCOUNT_KEY", typedSrc.Spec.AccountKey)

	return common.NewAdapterDeployment(src, sinkURI,
		resource.Image(r.adapterCfg.Image),
		resource.EnvVar("AZURE_ACCOUNT_NAME", typedSrc.Spec.AccountName),
		resource.EnvVar("AZURE_QUEUE_NAME", typedSrc.Spec.QueueName),
		resource.EnvVars(storageQueueEnvs...),
		resource.EnvVar(common.EnvNamespace, src.GetNamespace()),
		resource.EnvVar(common.EnvName, src.GetName()),
		resource.EnvVars(r.adapterCfg.configs.ToEnvVars()...),
	)
}

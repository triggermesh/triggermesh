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

package ocimetricssource

import (
	"encoding/json"
	"fmt"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/labels"

	"knative.dev/eventing/pkg/reconciler/source"
	"knative.dev/pkg/apis"
	"knative.dev/pkg/kmeta"

	"github.com/triggermesh/triggermesh/pkg/apis/sources/v1alpha1"
	"github.com/triggermesh/triggermesh/pkg/sources/reconciler/common"
	"github.com/triggermesh/triggermesh/pkg/sources/reconciler/common/resource"
)

const (
	oracleAPIKey            = "ORACLE_API_PRIVATE_KEY"
	oracleAPIKeyPassphrase  = "ORACLE_API_PRIVATE_KEY_PASSPHRASE"
	oracleAPIKeyFingerprint = "ORACLE_API_PRIVATE_KEY_FINGERPRINT"
	userOCID                = "ORACLE_USER_OCID"
	tenantOCID              = "ORACLE_TENANT_OCID"
	oracleRegion            = "ORACLE_REGION"
	pollingFrequency        = "ORACLE_METRICS_POLLING_FREQUENCY"
	metrics                 = "ORACLE_METRICS"
)

const defaultPollingFrequency = "5m"

// adapterConfig contains properties used to configure the source's adapter.
// These are automatically populated by envconfig.
type adapterConfig struct {
	// Container image
	Image string `default:"gcr.io/triggermesh/ocimetricssource-adapter"`
	// Configuration accessor for logging/metrics/tracing
	configs source.ConfigAccessor
}

// Verify that Reconciler implements common.AdapterDeploymentBuilder.
var _ common.AdapterDeploymentBuilder = (*Reconciler)(nil)

// BuildAdapter implements common.AdapterDeploymentBuilder.
func (r *Reconciler) BuildAdapter(src v1alpha1.EventSource, sinkURI *apis.URL) *appsv1.Deployment {
	typedSrc := src.(*v1alpha1.OCIMetricsSource)

	return common.NewAdapterDeployment(src, sinkURI,
		resource.Image(r.adapterCfg.Image),

		resource.EnvVars(makeOCIMetricsEnvs(typedSrc)...),
		resource.EnvVars(r.adapterCfg.configs.ToEnvVars()...),
	)
}

// RBACOwners implements common.AdapterDeploymentBuilder.
func (r *Reconciler) RBACOwners(namespace string) ([]kmeta.OwnerRefable, error) {
	srcs, err := r.srcLister(namespace).List(labels.Everything())
	if err != nil {
		return nil, fmt.Errorf("listing objects from cache: %w", err)
	}

	ownerRefables := make([]kmeta.OwnerRefable, len(srcs))
	for i := range srcs {
		ownerRefables[i] = srcs[i]
	}

	return ownerRefables, nil
}

func makeOCIMetricsEnvs(src *v1alpha1.OCIMetricsSource) []corev1.EnvVar {
	m, _ := json.Marshal(src.Spec.Metrics)

	frequency := defaultPollingFrequency
	if src.Spec.PollingFrequency != nil {
		frequency = *src.Spec.PollingFrequency
	}

	ociEnvs := []corev1.EnvVar{
		{
			Name:  common.EnvNamespace,
			Value: src.GetNamespace(),
		},
		{
			Name:  common.EnvName,
			Value: src.GetName(),
		},
		{
			Name:  userOCID,
			Value: src.Spec.User,
		},
		{
			Name:  tenantOCID,
			Value: src.Spec.Tenancy,
		},
		{
			Name:  oracleRegion,
			Value: src.Spec.Region,
		},
		{
			Name:  pollingFrequency,
			Value: frequency,
		},
		{
			Name:  metrics,
			Value: string(m),
		}}

	ociEnvs = common.MaybeAppendValueFromEnvVar(ociEnvs, oracleAPIKey, src.Spec.OracleAPIPrivateKey)
	ociEnvs = common.MaybeAppendValueFromEnvVar(ociEnvs, oracleAPIKeyPassphrase, src.Spec.OracleAPIPrivateKeyPassphrase)
	ociEnvs = common.MaybeAppendValueFromEnvVar(ociEnvs, oracleAPIKeyFingerprint, src.Spec.OracleAPIPrivateKeyFingerprint)

	return ociEnvs
}

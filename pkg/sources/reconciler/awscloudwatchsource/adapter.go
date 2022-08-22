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

package awscloudwatchsource

import (
	"encoding/json"
	"fmt"
	"time"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"

	"knative.dev/eventing/pkg/reconciler/source"
	"knative.dev/pkg/apis"

	commonv1alpha1 "github.com/triggermesh/triggermesh/pkg/apis/common/v1alpha1"
	"github.com/triggermesh/triggermesh/pkg/apis/sources/v1alpha1"
	common "github.com/triggermesh/triggermesh/pkg/reconciler"
	"github.com/triggermesh/triggermesh/pkg/reconciler/resource"
	"github.com/triggermesh/triggermesh/pkg/sources/reconciler"
)

const (
	envRegion          = "AWS_REGION"
	envQueries         = "QUERIES"
	envPollingInterval = "POLLING_INTERVAL"
)

const defaultPollingInterval = 5 * time.Minute

const healthPortName = "health"

// adapterConfig contains properties used to configure the source's adapter.
// These are automatically populated by envconfig.
type adapterConfig struct {
	// Container image
	Image string `default:"gcr.io/triggermesh/awscloudwatchsource"`
	// Configuration accessor for logging/metrics/tracing
	configs source.ConfigAccessor
}

// Verify that Reconciler implements common.AdapterBuilder.
var _ common.AdapterBuilder[*appsv1.Deployment] = (*Reconciler)(nil)

// BuildAdapter implements common.AdapterBuilder.
func (r *Reconciler) BuildAdapter(src commonv1alpha1.Reconcilable, sinkURI *apis.URL) (*appsv1.Deployment, error) {
	typedSrc := src.(*v1alpha1.AWSCloudWatchSource)

	env, err := MakeAppEnv(typedSrc)
	if err != nil {
		return nil, fmt.Errorf("building adapter environment: %w", err)
	}

	return common.NewAdapterDeployment(src, sinkURI,
		resource.Image(r.adapterCfg.Image),

		resource.EnvVars(env...),
		resource.EnvVars(r.adapterCfg.configs.ToEnvVars()...),

		resource.Port(healthPortName, 8080),
		resource.StartupProbe("/health", healthPortName),
	), nil
}

// MakeAppEnv extracts environment variables from the object.
// Exported to be used in external tools for local test environments.
func MakeAppEnv(o *v1alpha1.AWSCloudWatchSource) ([]corev1.EnvVar, error) {
	var queries string
	if qs := o.Spec.MetricQueries; len(qs) > 0 {
		q, err := json.Marshal(qs)
		if err != nil {
			return nil, fmt.Errorf("serializing spec.metricQueries to JSON: %w", err)
		}
		queries = string(q)
	}

	pollingInterval := defaultPollingInterval
	if f := o.Spec.PollingInterval; f != nil && time.Duration(*f).Nanoseconds() > 0 {
		pollingInterval = time.Duration(*f)
	}

	return append(reconciler.MakeAWSAuthEnvVars(o.Spec.Auth),
		[]corev1.EnvVar{
			{
				Name:  envRegion,
				Value: o.Spec.Region,
			}, {
				Name:  envQueries,
				Value: queries,
			}, {
				Name:  envPollingInterval,
				Value: pollingInterval.String(),
			},
		}...,
	), nil
}

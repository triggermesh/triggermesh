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
	corev1 "k8s.io/api/core/v1"

	"knative.dev/eventing/pkg/reconciler/source"
	"knative.dev/pkg/apis"

	commonv1alpha1 "github.com/triggermesh/triggermesh/pkg/apis/common/v1alpha1"
	"github.com/triggermesh/triggermesh/pkg/apis/sources/v1alpha1"
	common "github.com/triggermesh/triggermesh/pkg/reconciler"
	"github.com/triggermesh/triggermesh/pkg/reconciler/resource"
	"github.com/triggermesh/triggermesh/pkg/sources/reconciler"
)

const envMessageProcessor = "SQS_MESSAGE_PROCESSOR"

const healthPortName = "health"

// adapterConfig contains properties used to configure the source's adapter.
// These are automatically populated by envconfig.
type adapterConfig struct {
	// Container image
	Image string `default:"gcr.io/triggermesh/awssqssource"`
	// Configuration accessor for logging/metrics/tracing
	configs source.ConfigAccessor
}

// Verify that Reconciler implements common.AdapterBuilder.
var _ common.AdapterBuilder[*appsv1.Deployment] = (*Reconciler)(nil)

// BuildAdapter implements common.AdapterBuilder.
func (r *Reconciler) BuildAdapter(src commonv1alpha1.Reconcilable, sinkURI *apis.URL) (*appsv1.Deployment, error) {
	typedSrc := src.(*v1alpha1.AWSSQSSource)

	return common.NewAdapterDeployment(src, sinkURI,
		resource.Image(r.adapterCfg.Image),

		resource.EnvVars(MakeAppEnv(typedSrc)...),
		resource.EnvVars(r.adapterCfg.configs.ToEnvVars()...),

		resource.Port(healthPortName, 8080),

		resource.StartupProbe("/health", healthPortName),
	), nil
}

// maybeSetMessageProcessor conditionally sets the envMessageProcessor
// environment variable.
func maybeSetMessageProcessor(envs []corev1.EnvVar, src *v1alpha1.AWSSQSSource) []corev1.EnvVar {
	if mp := src.Spec.MessageProcessor; mp != nil && *mp == "s3" {
		envs = append(envs, corev1.EnvVar{
			Name:  envMessageProcessor,
			Value: *mp,
		})
	}

	return envs
}

// MakeAppEnv extracts environment variables from the object.
// Exported to be used in external tools for local test environments.
func MakeAppEnv(o *v1alpha1.AWSSQSSource) []corev1.EnvVar {
	awsEnvs := append(reconciler.MakeAWSAuthEnvVars(o.Spec.Auth),
		reconciler.MakeAWSEndpointEnvVars(o.Spec.Endpoint)...)
	awsEnvs = maybeSetMessageProcessor(awsEnvs, o)

	return append(awsEnvs, corev1.EnvVar{
		Name:  common.EnvARN,
		Value: o.Spec.ARN.String(),
	})
}

func customizeDeployment(d *appsv1.Deployment, typedSrc *v1alpha1.AWSSQSSource) *appsv1.Deployment {
	d.Spec.Replicas = typedSrc.Spec.Replicas

	nodeSelectorMap := map[string]string{}
	for k, v := range typedSrc.Spec.NodeSelector {
		nodeSelectorMap[k] = v
	}

	envVars := d.Spec.Template.Spec.Containers[0].Env

	annotations := map[string]string{"sidecar.istio.io/inject": "true"}
	for k, v := range typedSrc.Spec.Annotations {
		annotations[k] = v
	}

	d.Spec.Template.Spec.NodeSelector = nodeSelectorMap
	d.Spec.Template.Spec.Affinity = &typedSrc.Spec.Affinity
	d.Spec.Template.Annotations = annotations

	maxBatchSizeProvided := typedSrc.Spec.MaxBatchSize
	if len(maxBatchSizeProvided) != 0 {
		envVars = append(envVars, corev1.EnvVar{Name: "AWS_SQS_MAX_BATCH_SIZE", Value: maxBatchSizeProvided})
	}

	sendBatchedResponse := typedSrc.Spec.SendBatchedResponse
	if len(sendBatchedResponse) != 0 {
		envVars = append(envVars, corev1.EnvVar{Name: "AWS_SQS_SEND_BATCH_RESPONSE", Value: sendBatchedResponse})
	}

	onFailedPollWaitSecs := typedSrc.Spec.OnFailedPollWaitSecs
	if len(onFailedPollWaitSecs) != 0 {
		envVars = append(envVars, corev1.EnvVar{Name: "AWS_SQS_POLL_FAILED_WAIT_TIME", Value: onFailedPollWaitSecs})
	}

	waitTimeSeconds := typedSrc.Spec.WaitTimeSeconds
	if len(waitTimeSeconds) != 0 {
		envVars = append(envVars, corev1.EnvVar{Name: "AWS_SQS_WAIT_TIME_SECONDS", Value: waitTimeSeconds})
	}
	d.Spec.Template.Spec.Containers[0].Env = envVars

	return d
}

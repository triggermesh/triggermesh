/*
Copyright (c) 2020-2021 TriggerMesh Inc.

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

package awstarget

import (
	"strconv"

	libreconciler "github.com/triggermesh/triggermesh/pkg/targets/reconciler"
	"github.com/triggermesh/triggermesh/pkg/targets/reconciler/resources"
	corev1 "k8s.io/api/core/v1"

	"knative.dev/eventing/pkg/reconciler/source"
	"knative.dev/pkg/kmeta"
	servingv1 "knative.dev/serving/pkg/apis/serving/v1"

	"github.com/aws/aws-sdk-go/aws/arn"

	"github.com/triggermesh/triggermesh/pkg/apis/targets/v1alpha1"
)

const adapterName = "awstarget"

// adapterConfig contains properties used to configure the target's adapter.
// Public fields are automatically populated by envconfig.
type adapterConfig struct {
	// Configuration accessor for logging/metrics/tracing
	obsConfig source.ConfigAccessor
	// Container image
	Image string `envconfig:"AWS_ADAPTER_IMAGE" default:"gcr.io/triggermesh/aws-target-adapter"`
}

// makeTargetDynamoDBAdapterKService generates (but does not insert into K8s) the Target Adapter KService.
func makeTargetDynamoDBAdapterKService(target *v1alpha1.AWSDynamoDBTarget, cfg *adapterConfig) *servingv1.Service {
	labels := libreconciler.MakeAdapterLabels(adapterName, target.Name)
	name := kmeta.ChildName(adapterName+"-", target.Name)
	podLabels := libreconciler.MakeAdapterLabels(adapterName, target.Name)
	envSvc := libreconciler.MakeServiceEnv(name, target.Namespace)
	envApp := makeCommonAppEnv(&target.Spec.AWSApiKey, &target.Spec.AWSApiSecret, target.Spec.ARN, false, nil)
	envObs := libreconciler.MakeObsEnv(cfg.obsConfig)
	envs := append(envSvc, envApp...)
	envs = append(envs, envObs...)

	return resources.MakeKService(target.Namespace, name, cfg.Image,
		resources.KsvcLabels(labels),
		resources.KsvcLabelVisibilityClusterLocal,
		resources.KsvcOwner(target),
		resources.KsvcPodLabels(podLabels),
		resources.KsvcPodEnvVars(envs),
	)
}

// MakeTargetAdapterKService generates (but does not insert into K8s) the Target Adapter KService.
func makeTargetLambdaAdapterKService(target *v1alpha1.AWSLambdaTarget, cfg *adapterConfig) *servingv1.Service {
	labels := libreconciler.MakeAdapterLabels(adapterName, target.Name)
	name := kmeta.ChildName(adapterName+"-", target.Name)
	podLabels := libreconciler.MakeAdapterLabels(adapterName, target.Name)
	envSvc := libreconciler.MakeServiceEnv(name, target.Namespace)
	envApp := makeCommonAppEnv(&target.Spec.AWSApiKey, &target.Spec.AWSApiSecret, target.Spec.ARN, target.Spec.DiscardCEContext, nil)
	envObs := libreconciler.MakeObsEnv(cfg.obsConfig)
	envs := append(envSvc, envApp...)
	envs = append(envs, envObs...)

	return resources.MakeKService(target.Namespace, name, cfg.Image,
		resources.KsvcLabels(labels),
		resources.KsvcLabelVisibilityClusterLocal,
		resources.KsvcOwner(target),
		resources.KsvcPodLabels(podLabels),
		resources.KsvcPodEnvVars(envs),
	)
}

func makeTargetS3AdapterKService(target *v1alpha1.AWSS3Target, cfg *adapterConfig) *servingv1.Service {
	labels := libreconciler.MakeAdapterLabels(adapterName, target.Name)
	name := kmeta.ChildName(adapterName+"-", target.Name)
	podLabels := libreconciler.MakeAdapterLabels(adapterName, target.Name)
	envSvc := libreconciler.MakeServiceEnv(name, target.Namespace)
	envApp := makeCommonAppEnv(&target.Spec.AWSApiKey, &target.Spec.AWSApiSecret, target.Spec.ARN, target.Spec.DiscardCEContext, nil)
	envObs := libreconciler.MakeObsEnv(cfg.obsConfig)
	envs := append(envSvc, envApp...)
	envs = append(envs, envObs...)

	return resources.MakeKService(target.Namespace, name, cfg.Image,
		resources.KsvcLabels(labels),
		resources.KsvcLabelVisibilityClusterLocal,
		resources.KsvcOwner(target),
		resources.KsvcPodLabels(podLabels),
		resources.KsvcPodEnvVars(envs),
	)
}

func makeTargetSNSAdapterKService(target *v1alpha1.AWSSNSTarget, cfg *adapterConfig) *servingv1.Service {
	labels := libreconciler.MakeAdapterLabels(adapterName, target.Name)
	name := kmeta.ChildName(adapterName+"-", target.Name)
	podLabels := libreconciler.MakeAdapterLabels(adapterName, target.Name)
	envSvc := libreconciler.MakeServiceEnv(name, target.Namespace)
	envApp := makeCommonAppEnv(&target.Spec.AWSApiKey, &target.Spec.AWSApiSecret, target.Spec.ARN, target.Spec.DiscardCEContext, nil)
	envObs := libreconciler.MakeObsEnv(cfg.obsConfig)
	envs := append(envSvc, envApp...)
	envs = append(envs, envObs...)

	return resources.MakeKService(target.Namespace, name, cfg.Image,
		resources.KsvcLabels(labels),
		resources.KsvcLabelVisibilityClusterLocal,
		resources.KsvcOwner(target),
		resources.KsvcPodLabels(podLabels),
		resources.KsvcPodEnvVars(envs),
	)
}

func makeTargetSQSAdapterKService(target *v1alpha1.AWSSQSTarget, cfg *adapterConfig) *servingv1.Service {
	labels := libreconciler.MakeAdapterLabels(adapterName, target.Name)
	name := kmeta.ChildName(adapterName+"-", target.Name)
	podLabels := libreconciler.MakeAdapterLabels(adapterName, target.Name)
	envSvc := libreconciler.MakeServiceEnv(name, target.Namespace)
	envApp := makeCommonAppEnv(&target.Spec.AWSApiKey, &target.Spec.AWSApiSecret, target.Spec.ARN, target.Spec.DiscardCEContext, nil)
	envObs := libreconciler.MakeObsEnv(cfg.obsConfig)
	envs := append(envSvc, envApp...)
	envs = append(envs, envObs...)

	return resources.MakeKService(target.Namespace, name, cfg.Image,
		resources.KsvcLabels(labels),
		resources.KsvcLabelVisibilityClusterLocal,
		resources.KsvcOwner(target),
		resources.KsvcPodLabels(podLabels),
		resources.KsvcPodEnvVars(envs),
	)
}

func makeTargetKinesisAdapterKService(target *v1alpha1.AWSKinesisTarget, cfg *adapterConfig) *servingv1.Service {
	labels := libreconciler.MakeAdapterLabels(adapterName, target.Name)
	name := kmeta.ChildName(adapterName+"-", target.Name)
	podLabels := libreconciler.MakeAdapterLabels(adapterName, target.Name)
	envSvc := libreconciler.MakeServiceEnv(name, target.Namespace)
	envApp := makeCommonAppEnv(&target.Spec.AWSApiKey, &target.Spec.AWSApiSecret, target.Spec.ARN, target.Spec.DiscardCEContext, &target.Spec.Partition)
	envObs := libreconciler.MakeObsEnv(cfg.obsConfig)
	envs := append(envSvc, envApp...)
	envs = append(envs, envObs...)

	return resources.MakeKService(target.Namespace, name, cfg.Image,
		resources.KsvcLabels(labels),
		resources.KsvcLabelVisibilityClusterLocal,
		resources.KsvcOwner(target),
		resources.KsvcPodLabels(podLabels),
		resources.KsvcPodEnvVars(envs),
	)
}

func makeCommonAppEnv(key, secret *v1alpha1.SecretValueFromSource, arnStr string,
	discardCEContext bool, kinesisPartition *string) []corev1.EnvVar {

	var targetType string
	if arn, err := arn.Parse(arnStr); err == nil {
		// An invalid ARN would cause targetType to remain empty, but
		// this kind of error can be safely discarded because a valid
		// ARN is required for the target adapter to operate.
		targetType = arn.Service
	}

	envs := []corev1.EnvVar{{
		Name: "AWS_ACCESS_KEY_ID",
		ValueFrom: &corev1.EnvVarSource{
			SecretKeyRef: key.SecretKeyRef,
		},
	}, {
		Name: "AWS_SECRET_ACCESS_KEY",
		ValueFrom: &corev1.EnvVarSource{
			SecretKeyRef: secret.SecretKeyRef,
		},
	}, {
		Name:  "AWS_TARGET_TYPE",
		Value: targetType,
	}, {
		Name:  "AWS_TARGET_ARN",
		Value: arnStr,
	}, {
		Name:  "AWS_DISCARD_CE_CONTEXT",
		Value: strconv.FormatBool(discardCEContext),
	}}

	if kinesisPartition != nil && *kinesisPartition != "" {
		envs = append(envs, corev1.EnvVar{
			Name:  "AWS_KINESIS_PARTITION",
			Value: *kinesisPartition,
		})
	}

	return envs
}

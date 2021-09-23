/*
Copyright (c) 2021 TriggerMesh Inc.

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

package confluenttarget

import (
	"strconv"
	"strings"

	corev1 "k8s.io/api/core/v1"
	"knative.dev/eventing/pkg/reconciler/source"
	"knative.dev/pkg/kmeta"
	servingv1 "knative.dev/serving/pkg/apis/serving/v1"

	"github.com/triggermesh/triggermesh/pkg/apis/targets/v1alpha1"
	confluentv1alpha1 "github.com/triggermesh/triggermesh/pkg/apis/targets/v1alpha1"
	libreconciler "github.com/triggermesh/triggermesh/pkg/targets/reconciler"
	"github.com/triggermesh/triggermesh/pkg/targets/reconciler/resources"
)

const adapterName = "confluenttarget"

// adapterConfig contains properties used to configure the target's adapter.
// Public fields are automatically populated by envconfig.
type adapterConfig struct {
	// Configuration accessor for logging/metrics/tracing
	obsConfig source.ConfigAccessor
	// Container image
	Image string `envconfig:"CONFLUENT_ADAPTER_IMAGE" default:"gcr.io/triggermesh/confluent-target-adapter"`
}

// TargetAdapterArgs are the arguments needed to create a Target Adapter.
// Every field is required.
type TargetAdapterArgs struct {
	Image  string
	Target *confluentv1alpha1.ConfluentTarget
}

// makeTargetAdapterKService generates (but does not insert into K8s) the Target Adapter KService.
func makeTargetAdapterKService(target *confluentv1alpha1.ConfluentTarget, cfg *adapterConfig) *servingv1.Service {
	name := kmeta.ChildName(adapterName+"-", target.Name)
	lbl := libreconciler.MakeAdapterLabels(adapterName, target.Name)
	podLabels := libreconciler.MakeAdapterLabels(adapterName, target.Name)
	envSvc := libreconciler.MakeServiceEnv(name, target.Namespace)
	envApp := makeAppEnv(&target.Spec)
	envObs := libreconciler.MakeObsEnv(cfg.obsConfig)
	envs := append(envSvc, envApp...)
	envs = append(envs, envObs...)

	return resources.MakeKService(target.Namespace, name, cfg.Image,
		resources.KsvcLabels(lbl),
		resources.KsvcLabelVisibilityClusterLocal,
		resources.KsvcOwner(target),
		resources.KsvcPodLabels(podLabels),
		resources.KsvcPodEnvVars(envs),
	)
}

func makeAppEnv(spec *v1alpha1.ConfluentTargetSpec) []corev1.EnvVar {
	env := []corev1.EnvVar{
		{
			Name:  "CONFLUENT_SASL_USERNAME",
			Value: spec.SASLUsername,
		}, {
			Name: "CONFLUENT_SASL_PASSWORD",
			ValueFrom: &corev1.EnvVarSource{
				SecretKeyRef: spec.SASLPassword.SecretKeyRef,
			},
		}, {
			Name:  "CONFLUENT_TOPIC",
			Value: spec.Topic,
		}, {
			Name:  "CONFLUENT_BOOTSTRAP_SERVERS",
			Value: strings.Join(spec.BootstrapServers, ","),
		}, {
			Name:  "CONFLUENT_DISCARD_CE_CONTEXT",
			Value: strconv.FormatBool(spec.DiscardCEContext),
		},
		{
			Name:  "CONFLUENT_SASL_MECHANISMS",
			Value: spec.SASLMechanisms,
		},
		{
			Name:  "CONFLUENT_SECURITY_PROTOCOL",
			Value: spec.SecurityProtocol,
		},
	}

	if spec.TopicReplicationFactor != nil {
		env = append(env, corev1.EnvVar{
			Name:  "CONFLUENT_TOPIC_REPLICATION_FACTOR",
			Value: strconv.Itoa(*spec.TopicReplicationFactor),
		})
	}

	if spec.TopicPartitions != nil {
		env = append(env, corev1.EnvVar{
			Name:  "CONFLUENT_TOPIC_PARTITIONS",
			Value: strconv.Itoa(*spec.TopicPartitions),
		})
	}

	return env
}

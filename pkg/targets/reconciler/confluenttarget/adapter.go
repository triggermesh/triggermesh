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

package confluenttarget

import (
	"strconv"
	"strings"

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
	Image string `default:"gcr.io/triggermesh/confluenttarget-adapter"`
}

// Verify that Reconciler implements common.AdapterServiceBuilder.
var _ common.AdapterServiceBuilder = (*Reconciler)(nil)

// BuildAdapter implements common.AdapterServiceBuilder.
func (r *Reconciler) BuildAdapter(trg commonv1alpha1.Reconcilable, _ *apis.URL) *servingv1.Service {
	typedTrg := trg.(*v1alpha1.ConfluentTarget)

	return common.NewAdapterKnService(trg, nil,
		resource.Image(r.adapterCfg.Image),
		resource.EnvVars(makeAppEnv(typedTrg)...),
		resource.EnvVars(r.adapterCfg.obsConfig.ToEnvVars()...),
	)
}

func makeAppEnv(o *v1alpha1.ConfluentTarget) []corev1.EnvVar {
	env := []corev1.EnvVar{
		{
			Name:  "CONFLUENT_SASL_USERNAME",
			Value: o.Spec.SASLUsername,
		}, {
			Name: "CONFLUENT_SASL_PASSWORD",
			ValueFrom: &corev1.EnvVarSource{
				SecretKeyRef: o.Spec.SASLPassword.SecretKeyRef,
			},
		}, {
			Name:  "CONFLUENT_TOPIC",
			Value: o.Spec.Topic,
		}, {
			Name:  "CONFLUENT_BOOTSTRAP_SERVERS",
			Value: strings.Join(o.Spec.BootstrapServers, ","),
		}, {
			Name:  "CONFLUENT_DISCARD_CE_CONTEXT",
			Value: strconv.FormatBool(o.Spec.DiscardCEContext),
		},
		{
			Name:  "CONFLUENT_SASL_MECHANISMS",
			Value: o.Spec.SASLMechanisms,
		},
		{
			Name:  "CONFLUENT_SECURITY_PROTOCOL",
			Value: o.Spec.SecurityProtocol,
		},
	}

	if o.Spec.TopicReplicationFactor != nil {
		env = append(env, corev1.EnvVar{
			Name:  "CONFLUENT_TOPIC_REPLICATION_FACTOR",
			Value: strconv.Itoa(*o.Spec.TopicReplicationFactor),
		})
	}

	if o.Spec.TopicPartitions != nil {
		env = append(env, corev1.EnvVar{
			Name:  "CONFLUENT_TOPIC_PARTITIONS",
			Value: strconv.Itoa(*o.Spec.TopicPartitions),
		})
	}

	return env
}

// RBACOwners implements common.AdapterServiceBuilder.
func (r *Reconciler) RBACOwners(trg commonv1alpha1.Reconcilable) ([]kmeta.OwnerRefable, error) {
	return common.RBACOwners[*v1alpha1.ConfluentTarget](r.trgLister(trg.GetNamespace()))
}

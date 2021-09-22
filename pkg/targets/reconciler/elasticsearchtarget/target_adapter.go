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

package elasticsearchtarget

import (
	"strconv"
	"strings"

	"github.com/triggermesh/triggermesh/pkg/apis/targets/v1alpha1"
	pkgreconciler "github.com/triggermesh/triggermesh/pkg/targets/reconciler"
	"github.com/triggermesh/triggermesh/pkg/targets/reconciler/resources"
	"knative.dev/eventing/pkg/reconciler/source"

	corev1 "k8s.io/api/core/v1"
	"knative.dev/pkg/kmeta"
	servingv1 "knative.dev/serving/pkg/apis/serving/v1"
)

const adapterName = "elasticsearchtarget"

const envEventsPayloadPolicy = "EVENTS_PAYLOAD_POLICY"

// adapterConfig contains properties used to configure the target's adapter.
// Public fields are automatically populated by envconfig.
type adapterConfig struct {
	// Configuration accessor for logging/metrics/tracing
	obsConfig source.ConfigAccessor
	// Container image
	Image string `envconfig:"ELASTICSEARCH_ADAPTER_IMAGE" default:"gcr.io/triggermesh/elasticsearch-target-adapter"`
}

// TargetAdapterArgs are the arguments needed to create a Target Adapter.
// Every field is required.
type TargetAdapterArgs struct {
	Image  string
	Target *v1alpha1.ElasticsearchTarget
}

// MakeTargetAdapterKService generates the target adapter KService.
func makeTargetAdapterKService(target *v1alpha1.ElasticsearchTarget, cfg *adapterConfig) *servingv1.Service {
	labels := pkgreconciler.MakeAdapterLabels(adapterName, target.Name)
	name := kmeta.ChildName(adapterName+"-", target.Name)
	podLabels := pkgreconciler.MakeAdapterLabels(adapterName, target.Name)
	envSvc := pkgreconciler.MakeServiceEnv(name, target.Namespace)
	envApp := makeAppEnv(target)
	envObs := pkgreconciler.MakeObsEnv(cfg.obsConfig)
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

func makeAppEnv(o *v1alpha1.ElasticsearchTarget) []corev1.EnvVar {
	env := []corev1.EnvVar{
		{
			Name:  "ELASTICSEARCH_INDEX",
			Value: o.Spec.IndexName,
		}, {
			Name:  "ELASTICSEARCH_DISCARD_CE_CONTEXT",
			Value: strconv.FormatBool(o.Spec.DiscardCEContext),
		}, {
			Name:  pkgreconciler.EnvBridgeID,
			Value: pkgreconciler.GetStatefulBridgeID(o),
		},
	}

	if o.Spec.Connection.SkipVerify != nil {
		env = append(env, corev1.EnvVar{
			Name:  "ELASTICSEARCH_SKIPVERIFY",
			Value: strconv.FormatBool(*o.Spec.Connection.SkipVerify)})
	}

	if len(o.Spec.Connection.Addresses) != 0 {
		env = append(env, corev1.EnvVar{
			Name:  "ELASTICSEARCH_ADDRESSES",
			Value: strings.Join(o.Spec.Connection.Addresses, " ")})
	}

	if o.Spec.Connection.Username != nil {
		env = append(env, corev1.EnvVar{
			Name:  "ELASTICSEARCH_USER",
			Value: *o.Spec.Connection.Username})
	}

	if o.Spec.Connection.Password != nil && o.Spec.Connection.Password.SecretKeyRef != nil {
		env = append(env, corev1.EnvVar{
			Name:      "ELASTICSEARCH_PASSWORD",
			ValueFrom: &corev1.EnvVarSource{SecretKeyRef: o.Spec.Connection.Password.SecretKeyRef}})
	}

	if o.Spec.Connection.APIKey != nil && o.Spec.Connection.APIKey.SecretKeyRef != nil {
		env = append(env, corev1.EnvVar{
			Name:      "ELASTICSEARCH_APIKEY",
			ValueFrom: &corev1.EnvVarSource{SecretKeyRef: o.Spec.Connection.APIKey.SecretKeyRef}})
	}

	if o.Spec.Connection.CACert != nil {
		env = append(env, corev1.EnvVar{
			Name:  "ELASTICSEARCH_CACERT",
			Value: *o.Spec.Connection.CACert})
	}

	if o.Spec.EventOptions != nil && o.Spec.EventOptions.PayloadPolicy != nil {
		env = append(env, corev1.EnvVar{
			Name:  envEventsPayloadPolicy,
			Value: string(*o.Spec.EventOptions.PayloadPolicy),
		})
	}

	return env
}

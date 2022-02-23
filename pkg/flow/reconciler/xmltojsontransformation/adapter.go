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

package xmltojsontransformation

import (
	corev1 "k8s.io/api/core/v1"

	"knative.dev/eventing/pkg/reconciler/source"
	"knative.dev/pkg/apis"
	"knative.dev/pkg/kmeta"
	servingv1 "knative.dev/serving/pkg/apis/serving/v1"

	"github.com/triggermesh/triggermesh/pkg/apis/flow/v1alpha1"
	libreconciler "github.com/triggermesh/triggermesh/pkg/flow/reconciler"
	"github.com/triggermesh/triggermesh/pkg/flow/reconciler/resources"
)

const (
	adapterName            = "xmltojsontransformation"
	envEventsPayloadPolicy = "EVENTS_PAYLOAD_POLICY"
)

// adapterConfig contains properties used to configure the target's adapter.
// Public fields are automatically populated by envconfig.
type adapterConfig struct {
	// Configuration accessor for logging/metrics/tracing
	obsConfig source.ConfigAccessor
	// Container image
	Image string `envconfig:"XMLTOJSONTRANSFORMATION_IMAGE" default:"gcr.io/triggermesh/xmltojsontransformation-adapter"`
}

// makeAdapterKService generates (but does not insert into K8s) the Synchronizer Adapter KService.
func makeAdapterKService(o *v1alpha1.XMLToJSONTransformation, cfg *adapterConfig, sink *apis.URL) *servingv1.Service {
	name := kmeta.ChildName(adapterName+"-", o.Name)
	lbl := libreconciler.MakeAdapterLabels(adapterName, o)
	envSvc := libreconciler.MakeServiceEnv(name, o.Namespace)
	envApp := makeAppEnv(o, sink)
	envObs := libreconciler.MakeObsEnv(cfg.obsConfig)
	envs := append(envSvc, envApp...)
	envs = append(envs, envObs...)

	return resources.MakeKService(o.Namespace, name, cfg.Image,
		resources.KsvcLabels(lbl),
		resources.KsvcLabelVisibilityClusterLocal,
		resources.KsvcOwner(o),
		resources.KsvcPodLabels(lbl),
		resources.KsvcPodEnvVars(envs),
	)
}

func makeAppEnv(o *v1alpha1.XMLToJSONTransformation, sink *apis.URL) []corev1.EnvVar {
	env := []corev1.EnvVar{
		{
			Name:  libreconciler.EnvBridgeID,
			Value: libreconciler.GetStatefulBridgeID(o),
		},
	}

	if o.Spec.EventOptions != nil && o.Spec.EventOptions.PayloadPolicy != nil {
		env = append(env, corev1.EnvVar{
			Name:  envEventsPayloadPolicy,
			Value: string(*o.Spec.EventOptions.PayloadPolicy),
		})
	}

	if sink != nil {
		env = append(env, corev1.EnvVar{
			Name:  "K_SINK",
			Value: sink.String(),
		})
	}

	return env

}

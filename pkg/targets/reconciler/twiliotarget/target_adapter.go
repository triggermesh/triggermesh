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

package twiliotarget

import (
	"github.com/triggermesh/triggermesh/pkg/apis/targets/v1alpha1"
	libreconciler "github.com/triggermesh/triggermesh/pkg/targets/reconciler"
	"github.com/triggermesh/triggermesh/pkg/targets/reconciler/resources"

	corev1 "k8s.io/api/core/v1"

	"knative.dev/eventing/pkg/reconciler/source"
	"knative.dev/pkg/kmeta"
	servingv1 "knative.dev/serving/pkg/apis/serving/v1"
)

const (
	adapterName = "twiliotarget"

	envTwilioSID           = "TWILIO_SID"
	envTwilioToken         = "TWILIO_TOKEN"
	envTwilioDefaultFrom   = "TWILIO_DEFAULT_FROM"
	envTwilioDefaultTo     = "TWILIO_DEFAULT_TO"
	envEventsPayloadPolicy = "EVENTS_PAYLOAD_POLICY"
)

// adapterConfig contains properties used to configure the target's adapter.
// Public fields are automatically populated by envconfig.
type adapterConfig struct {
	// Configuration accessor for logging/metrics/tracing
	obsConfig source.ConfigAccessor
	// Container image
	Image string `envconfig:"TWILIO_ADAPTER_IMAGE" default:"gcr.io/triggermesh/twilio-target-adapter"`
}

// TargetAdapterArgs are the arguments needed to create a Target Adapter.
// Every field is required.
type TargetAdapterArgs struct {
	Image  string
	Target *v1alpha1.TwilioTarget
}

// makeTargetAdapterKService generates (but does not insert into K8s) the Target Adapter KService.
func makeTargetAdapterKService(target *v1alpha1.TwilioTarget, cfg *adapterConfig) *servingv1.Service {
	name := kmeta.ChildName(adapterName+"-", target.Name)
	lbl := libreconciler.MakeAdapterLabels(adapterName, target.Name)
	podLabels := libreconciler.MakeAdapterLabels(adapterName, target.Name)
	envSvc := libreconciler.MakeServiceEnv(name, target.Namespace)
	envApp := makeAppEnv(target)
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

func makeAppEnv(o *v1alpha1.TwilioTarget) []corev1.EnvVar {
	env := []corev1.EnvVar{
		{
			Name: envTwilioSID,
			ValueFrom: &corev1.EnvVarSource{
				SecretKeyRef: o.Spec.AccountSID.SecretKeyRef,
			},
		}, {
			Name:  libreconciler.EnvBridgeID,
			Value: libreconciler.GetStatefulBridgeID(o),
		}, {
			Name: envTwilioToken,
			ValueFrom: &corev1.EnvVarSource{
				SecretKeyRef: o.Spec.Token.SecretKeyRef,
			},
		},
	}

	if o.Spec.DefaultPhoneFrom != nil {
		env = append(env, corev1.EnvVar{
			Name:  envTwilioDefaultFrom,
			Value: *o.Spec.DefaultPhoneFrom,
		})
	}

	if o.Spec.DefaultPhoneTo != nil {
		env = append(env, corev1.EnvVar{
			Name:  envTwilioDefaultTo,
			Value: *o.Spec.DefaultPhoneTo,
		})
	}

	if o.Spec.EventOptions != nil && o.Spec.EventOptions.PayloadPolicy != nil {
		env = append(env, corev1.EnvVar{
			Name:  envEventsPayloadPolicy,
			Value: string(*o.Spec.EventOptions.PayloadPolicy),
		})
	}

	return env
}

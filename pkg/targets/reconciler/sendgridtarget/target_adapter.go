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

package sendgridtarget

import (
	corev1 "k8s.io/api/core/v1"
	"knative.dev/eventing/pkg/reconciler/source"
	"knative.dev/pkg/kmeta"
	servingv1 "knative.dev/serving/pkg/apis/serving/v1"

	"github.com/triggermesh/triggermesh/pkg/apis/targets/v1alpha1"
	libreconciler "github.com/triggermesh/triggermesh/pkg/targets/reconciler"
	pkgreconciler "github.com/triggermesh/triggermesh/pkg/targets/reconciler"
	"github.com/triggermesh/triggermesh/pkg/targets/reconciler/resources"
)

const adapterName = "sendgrid"

// adapterConfig contains properties used to configure the target's adapter.
// Public fields are automatically populated by envconfig.
type adapterConfig struct {
	// Configuration accessor for logging/metrics/tracing
	obsConfig source.ConfigAccessor
	// Container image
	Image string `envconfig:"SENDGRID_ADAPTER_IMAGE" default:"gcr.io/triggermesh/sendgrid-target-adapter"`
}

// TargetAdapterArgs are the arguments needed to create a Target Adapter.
// Every field is required.
type TargetAdapterArgs struct {
	Image  string
	Target *v1alpha1.SendGridTarget
}

// MakeTargetAdapterKService generates (but does not insert into K8s) the Target Adapter KService.
func makeTargetAdapterKService(target *v1alpha1.SendGridTarget, cfg *adapterConfig) *servingv1.Service {
	labels := libreconciler.MakeAdapterLabels(adapterName, target.Name)
	name := kmeta.ChildName(adapterName+"-", target.Name)
	podLabels := libreconciler.MakeAdapterLabels(adapterName, target.Name)
	envSvc := libreconciler.MakeServiceEnv(name, target.Namespace)
	envApp := makeAppEnv(target)
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

func makeAppEnv(o *v1alpha1.SendGridTarget) []corev1.EnvVar {
	env := []corev1.EnvVar{
		{
			Name: "SENDGRID_API_KEY",
			ValueFrom: &corev1.EnvVarSource{
				SecretKeyRef: o.Spec.APIKey.SecretKeyRef,
			},
		},
	}

	if o.Spec.DefaultFromEmail != nil {
		env = append(env, corev1.EnvVar{
			Name:  "SENDGRID_DEFAULT_FROM_EMAIL",
			Value: *o.Spec.DefaultFromEmail,
		})
	}

	if o.Spec.DefaultToEmail != nil {
		env = append(env, corev1.EnvVar{
			Name:  "SENDGRID_DEFAULT_TO_EMAIL",
			Value: *o.Spec.DefaultToEmail,
		})
	}

	if o.Spec.DefaultToName != nil {
		env = append(env, corev1.EnvVar{
			Name:  "SENDGRID_DEFAULT_TO_NAME",
			Value: *o.Spec.DefaultToName,
		})
	}
	if o.Spec.DefaultFromName != nil {
		env = append(env, corev1.EnvVar{
			Name:  "SENDGRID_DEFAULT_FROM_NAME",
			Value: *o.Spec.DefaultFromName,
		})
	}

	if o.Spec.DefaultSubject != nil {
		env = append(env, corev1.EnvVar{
			Name:  "SENDGRID_DEFAULT_SUBJECT",
			Value: *o.Spec.DefaultSubject,
		})
	}

	if o.Spec.DefaultMessage != nil {
		env = append(env, corev1.EnvVar{
			Name:  "SENDGRID_DEFAULT_MESSAGE",
			Value: *o.Spec.DefaultMessage,
		})
	}

	if o.Spec.EventOptions != nil && o.Spec.EventOptions.PayloadPolicy != nil {
		env = append(env, corev1.EnvVar{
			Name:  "EVENTS_PAYLOAD_POLICY",
			Value: string(*o.Spec.EventOptions.PayloadPolicy),
		})
	}

	env = append(env, corev1.EnvVar{
		Name:  pkgreconciler.EnvBridgeID,
		Value: pkgreconciler.GetStatefulBridgeID(o),
	})

	return env
}

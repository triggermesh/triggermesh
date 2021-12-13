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

package ibmmqtarget

import (
	"strconv"

	corev1 "k8s.io/api/core/v1"

	"knative.dev/eventing/pkg/reconciler/source"
	"knative.dev/pkg/kmeta"
	servingv1 "knative.dev/serving/pkg/apis/serving/v1"

	libreconciler "github.com/triggermesh/triggermesh/pkg/targets/reconciler"
	tmcommon "github.com/triggermesh/triggermesh/pkg/targets/reconciler/common"
	"github.com/triggermesh/triggermesh/pkg/targets/reconciler/resources"

	"github.com/triggermesh/triggermesh/pkg/apis/targets/v1alpha1"
)

const (
	adapterName = "ibmmqtarget"

	envQueueManager   = "QUEUE_MANAGER"
	envChannelName    = "CHANNEL_NAME"
	envConnectionName = "CONNECTION_NAME"
	envUser           = "USER"
	envPassword       = "PASSWORD"
	envQueueName      = "QUEUE_NAME"
	envReplyToManager = "REPLY_TO_MANAGER"
	envReplyToQueue   = "REPLY_TO_QUEUE"

	envDiscardCEContext    = "DISCARD_CE_CONTEXT"
	envEventsPayloadPolicy = "EVENTS_PAYLOAD_POLICY"
)

// adapterConfig contains properties used to configure the target's adapter.
// Public fields are automatically populated by envconfig.
type adapterConfig struct {
	// Configuration accessor for logging/metrics/tracing
	obsConfig source.ConfigAccessor
	// Container image
	Image string `envconfig:"IBMMQTARGET_ADAPTER_IMAGE" default:"gcr.io/triggermesh/ibmmqtarget-adapter"`
}

// makeTargetAdapterKService generates (but does not insert into K8s) the Target Adapter KService.
func makeTargetAdapterKService(target *v1alpha1.IBMMQTarget, cfg *adapterConfig) *servingv1.Service {
	name := kmeta.ChildName(adapterName+"-", target.Name)
	lbl := libreconciler.MakeAdapterLabels(adapterName, target.Name)
	podLabels := libreconciler.MakeAdapterLabels(adapterName, target.Name)
	envSvc := libreconciler.MakeServiceEnv(name, target.Namespace)
	envApp := makeAppEnv(target)
	envApp = tmcommon.MaybeAppendValueFromEnvVar(envApp, envUser, target.Spec.Auth.User)
	envApp = tmcommon.MaybeAppendValueFromEnvVar(envApp, envPassword, target.Spec.Auth.Password)
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

func makeAppEnv(o *v1alpha1.IBMMQTarget) []corev1.EnvVar {
	env := []corev1.EnvVar{
		{
			Name:  libreconciler.EnvBridgeID,
			Value: libreconciler.GetStatefulBridgeID(o),
		},
		{
			Name:  envConnectionName,
			Value: o.Spec.ConnectionName,
		},
		{
			Name:  envQueueManager,
			Value: o.Spec.QueueManager,
		},
		{
			Name:  envQueueName,
			Value: o.Spec.QueueName,
		},
		{
			Name:  envChannelName,
			Value: o.Spec.ChannelName,
		},
		{
			Name:  envDiscardCEContext,
			Value: strconv.FormatBool(o.Spec.DiscardCEContext),
		},
	}

	if o.Spec.ReplyTo != nil {
		env = append(env, []corev1.EnvVar{
			{
				Name:  envReplyToManager,
				Value: o.Spec.ReplyTo.QueueManager,
			},
			{
				Name:  envReplyToQueue,
				Value: o.Spec.ReplyTo.QueueName,
			},
		}...)
	}

	if o.Spec.EventOptions != nil && o.Spec.EventOptions.PayloadPolicy != nil {
		env = append(env, corev1.EnvVar{
			Name:  envEventsPayloadPolicy,
			Value: string(*o.Spec.EventOptions.PayloadPolicy),
		})
	}

	return env

}

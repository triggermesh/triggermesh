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

package uipathtarget

import (
	libreconciler "github.com/triggermesh/triggermesh/pkg/targets/reconciler"
	"github.com/triggermesh/triggermesh/pkg/targets/reconciler/resources"
	corev1 "k8s.io/api/core/v1"
	"knative.dev/eventing/pkg/reconciler/source"
	"knative.dev/pkg/kmeta"
	servingv1 "knative.dev/serving/pkg/apis/serving/v1"

	"github.com/triggermesh/triggermesh/pkg/apis/targets/v1alpha1"
	uipathv1alpha1 "github.com/triggermesh/triggermesh/pkg/apis/targets/v1alpha1"
)

const adapterName = "uipathtarget"

// adapterConfig contains properties used to configure the target's adapter.
// Public fields are automatically populated by envconfig.
type adapterConfig struct {
	// Configuration accessor for logging/metrics/tracing
	obsConfig source.ConfigAccessor
	// Container image
	Image string `envconfig:"UIPATH_ADAPTER_IMAGE" default:"gcr.io/triggermesh/uipath-target-adapter"`
}

// TargetAdapterArgs are the arguments needed to create a Target Adapter.
// Every field is required.
type TargetAdapterArgs struct {
	Image  string
	Target *uipathv1alpha1.UiPathTarget
}

// makeTargetAdapterKService generates (but does not insert into K8s) the Target Adapter KService.
func makeTargetAdapterKService(target *uipathv1alpha1.UiPathTarget, cfg *adapterConfig) *servingv1.Service {
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

func makeAppEnv(spec *v1alpha1.UiPathTargetSpec) []corev1.EnvVar {
	env := []corev1.EnvVar{
		{
			Name:  "UIPATH_ROBOT_NAME",
			Value: spec.RobotName,
		}, {
			Name: "UIPATH_USER_KEY",
			ValueFrom: &corev1.EnvVarSource{
				SecretKeyRef: spec.UserKey.SecretKeyRef,
			},
		}, {
			Name:  "UIPATH_PROCESS_NAME",
			Value: spec.ProcessName,
		}, {
			Name:  "UIPATH_TENANT_NAME",
			Value: spec.TenantName,
		}, {
			Name:  "UIPATH_ACCOUNT_LOGICAL_NAME",
			Value: spec.AccountLogicalName,
		},
		{
			Name:  "UIPATH_CLIENT_ID",
			Value: spec.ClientID,
		},
		{
			Name:  "UIPATH_ORGANIZATION_UNIT_ID",
			Value: spec.OrganizationUnitID,
		},
	}

	return env
}

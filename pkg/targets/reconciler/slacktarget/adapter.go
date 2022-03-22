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

package slacktarget

import (
	"fmt"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"knative.dev/eventing/pkg/reconciler/source"
	"knative.dev/pkg/kmeta"
	servingv1 "knative.dev/serving/pkg/apis/serving/v1"

	"github.com/triggermesh/triggermesh/pkg/apis/targets/v1alpha1"
	libreconciler "github.com/triggermesh/triggermesh/pkg/targets/reconciler"
	"github.com/triggermesh/triggermesh/pkg/targets/reconciler/resources"
)

const targetPrefix = "slacktarget"

// TargetAdapterArgs are the arguments needed to create a Target Adapter.
// Every field is required.
type TargetAdapterArgs struct {
	Image   string
	Configs source.ConfigAccessor
	Target  *v1alpha1.SlackTarget
}

// MakeTargetAdapterKService generates (but does not insert into K8s) the Target Adapter KService.
func MakeTargetAdapterKService(args *TargetAdapterArgs) *servingv1.Service {
	genericLabels := libreconciler.MakeGenericLabels(targetPrefix, args.Target.Name)
	ksvcLabels := libreconciler.PropagateCommonLabels(args.Target, genericLabels)
	podLabels := libreconciler.PropagateCommonLabels(args.Target, genericLabels)
	name := kmeta.ChildName(fmt.Sprintf("%s-%s", targetPrefix, args.Target.Name), "")
	return &servingv1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: args.Target.Namespace,
			Name:      name,
			Labels:    ksvcLabels,
			OwnerReferences: []metav1.OwnerReference{
				*kmeta.NewControllerRef(args.Target),
			},
		},
		Spec: servingv1.ServiceSpec{
			ConfigurationSpec: servingv1.ConfigurationSpec{
				Template: servingv1.RevisionTemplateSpec{
					ObjectMeta: metav1.ObjectMeta{
						Labels: podLabels,
					},
					Spec: servingv1.RevisionSpec{
						PodSpec: corev1.PodSpec{
							Containers: []corev1.Container{{
								Image: args.Image,
								Env:   makeEnv(args),
							}},
						},
					},
				},
			},
		},
	}
}

// TODO ugly params, review this!
func makeEnv(args *TargetAdapterArgs) []corev1.EnvVar {
	env := []corev1.EnvVar{
		{
			Name:  resources.EnvNamespace,
			Value: args.Target.Namespace,
		}, {
			Name:  resources.EnvName,
			Value: args.Target.Name,
		}, {
			Name:  resources.EnvMetricsDomain,
			Value: resources.DefaultMetricsDomain,
		}, {
			Name: "SLACK_TOKEN",
			ValueFrom: &corev1.EnvVarSource{
				SecretKeyRef: args.Target.Spec.Token.SecretKeyRef,
			},
		},
	}

	return append(env, libreconciler.MakeObsEnv(args.Configs)...)
}

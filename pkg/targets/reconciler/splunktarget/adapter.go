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

package splunktarget

import (
	"strconv"

	resources2 "github.com/triggermesh/triggermesh/pkg/targets/reconciler/resources"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"

	"knative.dev/eventing/pkg/reconciler/source"
	network "knative.dev/networking/pkg"
	"knative.dev/pkg/apis"
	"knative.dev/pkg/kmeta"
	"knative.dev/serving/pkg/apis/serving"
	servingv1 "knative.dev/serving/pkg/apis/serving/v1"

	"github.com/triggermesh/triggermesh/pkg/apis/targets/v1alpha1"
)

const adapterName = "splunktarget"

const (
	envHECEndpoint   = "SPLUNK_HEC_ENDPOINT"
	envHECToken      = "SPLUNK_HEC_TOKEN"
	envIndex         = "SPLUNK_INDEX"
	envSkipTLSVerify = "SPLUNK_SKIP_TLS_VERIFY"
)

// adapterConfig contains properties used to configure the target's adapter.
// Public fields are automatically populated by envconfig.
type adapterConfig struct {
	// Configuration accessor for logging/metrics/tracing
	configs source.ConfigAccessor
	// Container image
	Image string `default:"gcr.io/triggermesh/splunk-target-adapter"`
}

// makeAdapterKnService returns a Knative Service object for the target's adapter.
func makeAdapterKnService(o *v1alpha1.SplunkTarget, cfg *adapterConfig) *servingv1.Service {
	svcLabels := labels.Set{
		resources2.AppNameLabel:      adapterName,
		resources2.AppInstanceLabel:  o.Name,
		resources2.AppComponentLabel: resources2.AdapterComponent,
		resources2.AppPartOfLabel:    partOf,
		resources2.AppManagedByLabel: resources2.ManagedController,
		// do not expose the service publicly
		network.VisibilityLabelKey: serving.VisibilityClusterLocal,
	}

	podLabels := labels.Set{
		resources2.AppNameLabel:      adapterName,
		resources2.AppInstanceLabel:  o.Name,
		resources2.AppComponentLabel: resources2.AdapterComponent,
		resources2.AppPartOfLabel:    partOf,
		resources2.AppManagedByLabel: resources2.ManagedController,
	}

	hecURL := apis.URL{
		Scheme: o.Spec.Endpoint.Scheme,
		Host:   o.Spec.Endpoint.Host,
	}

	env := []corev1.EnvVar{
		{
			Name:  resources2.EnvName,
			Value: o.Name,
		}, {
			Name:  resources2.EnvNamespace,
			Value: o.Namespace,
		}, {
			Name:  resources2.EnvMetricsDomain,
			Value: resources2.DefaultMetricsDomain,
		}, {
			Name:  envHECEndpoint,
			Value: hecURL.String(),
		},
	}

	tokenEnvVar := corev1.EnvVar{
		Name:  envHECToken,
		Value: o.Spec.Token.Value,
	}
	if tokenFromSecret := o.Spec.Token.ValueFromSecret; tokenFromSecret != nil {
		tokenEnvVar.Value = ""
		tokenEnvVar.ValueFrom = &corev1.EnvVarSource{
			SecretKeyRef: &corev1.SecretKeySelector{
				LocalObjectReference: corev1.LocalObjectReference{
					Name: tokenFromSecret.Name,
				},
				Key: tokenFromSecret.Key,
			},
		}
	}
	env = append(env, tokenEnvVar)

	if idx := o.Spec.Index; idx != nil && *idx != "" {
		env = append(env, corev1.EnvVar{
			Name:  envIndex,
			Value: *idx,
		})
	}

	if o.Spec.SkipTLSVerify != nil {
		env = append(env, corev1.EnvVar{
			Name:  envSkipTLSVerify,
			Value: strconv.FormatBool(*o.Spec.SkipTLSVerify),
		})
	}

	env = append(env, cfg.configs.ToEnvVars()...)

	// FIXME(antoineco): default metrics port 9090 overlaps with queue-proxy
	// Requires fix from https://github.com/knative/pkg/pull/1411:
	// {
	//	Name: "METRICS_PROMETHEUS_PORT",
	//	Value: "9092",
	// }
	env = append(env, corev1.EnvVar{
		Name:  source.EnvMetricsCfg,
		Value: "",
	})

	svc := &servingv1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: o.Namespace,
			Name:      kmeta.ChildName(adapterName+"-", o.Name),
			Labels:    svcLabels,
			OwnerReferences: []metav1.OwnerReference{
				*kmeta.NewControllerRef(o),
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
								Image: cfg.Image,
								Env:   env,
							}},
						},
					},
				},
			},
		},
	}

	return svc
}

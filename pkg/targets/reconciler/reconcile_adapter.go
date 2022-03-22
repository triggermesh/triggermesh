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

package reconciler

import (
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/labels"

	"knative.dev/eventing/pkg/reconciler/source"
	"knative.dev/pkg/kmeta"

	"github.com/triggermesh/triggermesh/pkg/targets/reconciler/resources"
)

// MakeServiceEnv Adds default environment variables
func MakeServiceEnv(name, namespace string) []corev1.EnvVar {
	return []corev1.EnvVar{
		{
			Name:  "NAMESPACE",
			Value: namespace,
		}, {
			Name:  "NAME",
			Value: name,
		},
	}
}

// MakeObsEnv adds support for observability configs
func MakeObsEnv(cfg source.ConfigAccessor) []corev1.EnvVar {
	return append(cfg.ToEnvVars(), corev1.EnvVar{
		Name:  "METRICS_PROMETHEUS_PORT",
		Value: "9092",
	})
}

// MakeGenericLabels returns generic labels set.
func MakeGenericLabels(adapterName string, componentName string) labels.Set {
	lbls := labels.Set{
		resources.AppNameLabel:      adapterName,
		resources.AppComponentLabel: resources.AdapterComponent,
		resources.AppPartOfLabel:    resources.PartOf,
		resources.AppManagedByLabel: resources.ManagedController,
	}

	if componentName != "" {
		lbls[resources.AppInstanceLabel] = componentName
	}

	return lbls
}

// PropagateCommonLabels adds common labels to the existing label set.
func PropagateCommonLabels(object kmeta.OwnerRefable, genericLabels labels.Set) labels.Set {
	parentLabels := object.GetObjectMeta().GetLabels()
	adapterLabels := make(labels.Set)

	for _, key := range resources.LabelsPropagationList {
		if value, exists := parentLabels[key]; exists {
			adapterLabels[key] = value
		}
	}

	for k, v := range genericLabels {
		adapterLabels[k] = v
	}

	return adapterLabels
}

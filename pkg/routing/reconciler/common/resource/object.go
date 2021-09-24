/*
Copyright (c) 2020 TriggerMesh Inc.

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

package resource

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"

	"knative.dev/pkg/kmeta"
)

// ObjectOption is a functional option for building Kubernetes API objects.
type ObjectOption func(interface{})

// Controller sets the given object as the controller (owner) of an API object.
func Controller(obj kmeta.OwnerRefable) ObjectOption {
	return func(object interface{}) {
		meta := object.(metav1.Object)

		meta.SetOwnerReferences([]metav1.OwnerReference{
			*kmeta.NewControllerRef(obj),
		})
	}
}

// Label sets the value of an API object's label.
func Label(key, val string) ObjectOption {
	return func(object interface{}) {
		meta := object.(metav1.Object)

		lbls := meta.GetLabels()

		if lbls == nil {
			lbls = make(labels.Set, 1)
			meta.SetLabels(lbls)
		}
		lbls[key] = val
	}
}

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

package resource

import (
	"strconv"
	"testing"

	"github.com/google/go-cmp/cmp"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"

	"knative.dev/pkg/kmeta"
	"knative.dev/pkg/ptr"
)

func makeEnvVars(count int, name, val string) []corev1.EnvVar {
	envVars := make([]corev1.EnvVar, count)
	for i := 1; i <= count; i++ {
		iStr := strconv.Itoa(i)
		envVars[i-1] = corev1.EnvVar{
			Name:  name + iStr,
			Value: val + iStr,
		}
	}
	return envVars
}

func TestMetaObjectOptions(t *testing.T) {
	owner1 := makeOwnerRefable("fake1")
	owner2 := makeOwnerRefable("fake2")
	owner3 := makeOwnerRefable("fake3")

	objMeta := NewKnService(tNs, tName,
		Label("test.label/2", "val2"),
		Controller(owner1),
		Annotation("test.annot/2", "val2"),
		Labels(labels.Set{
			"test.label/3": "val3",
			"test.label/4": "val4",
		}),
		Label("test.label/1", "val1"),
		Annotation("test.annot/1", "val1"),
		Owners(owner2, owner3),
	).ObjectMeta

	expectObjMeta := metav1.ObjectMeta{
		Namespace: tNs,
		Name:      tName,
		OwnerReferences: []metav1.OwnerReference{
			*kmeta.NewControllerRef(owner1),
			newRegularOwnerRef(owner2),
			newRegularOwnerRef(owner3),
		},
		Labels: map[string]string{
			"test.label/1": "val1",
			"test.label/2": "val2",
			"test.label/3": "val3",
			"test.label/4": "val4",
		},
		Annotations: map[string]string{
			"test.annot/1": "val1",
			"test.annot/2": "val2",
		},
	}

	if d := cmp.Diff(expectObjMeta, objMeta); d != "" {
		t.Errorf("Unexpected diff: (-:expect, +:got) %s", d)
	}
}

// newRegularOwnerRef returns a regular owner reference to the given object.
func newRegularOwnerRef(o kmeta.OwnerRefable) metav1.OwnerReference {
	owner := kmeta.NewControllerRef(o)
	owner.Controller = ptr.Bool(false)
	return *owner
}

// makeOwnerRefable returns a OwnerRefable with fake attributes values.
func makeOwnerRefable(name string) *fakeOwnerRefable {
	return &fakeOwnerRefable{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
			UID:  types.UID("00000000-0000-0000-0000-000000000000"),
		},
		GroupVersionKind: schema.GroupVersionKind{
			Group:   "fakegroup.fakeapi",
			Version: "v0",
			Kind:    "FakeKind",
		},
	}
}

var _ kmeta.OwnerRefable = (*fakeOwnerRefable)(nil)

// fakeOwnerRefable implements OwnerRefable.
type fakeOwnerRefable struct {
	metav1.ObjectMeta
	schema.GroupVersionKind
}

// GetGroupVersionKind returns the GroupVersionKind from the object.
func (o *fakeOwnerRefable) GetGroupVersionKind() schema.GroupVersionKind {
	return o.GroupVersionKind
}

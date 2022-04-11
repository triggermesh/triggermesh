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
	"testing"

	"github.com/google/go-cmp/cmp"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestNewConfigmap(t *testing.T) {
	cmap := NewConfigMap(tNs, tName,
		Data("test.txt", "Lorem ipsum dolor sit amet"),
	)

	expectCmap := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: tNs,
			Name:      tName,
		},
		Data: map[string]string{
			"test.txt": "Lorem ipsum dolor sit amet",
		},
	}

	if d := cmp.Diff(expectCmap, cmap); d != "" {
		t.Errorf("Unexpected diff: (-:expect, +:got) %s", d)
	}
}

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

package status

import (
	"context"
	"encoding/json"
	"fmt"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"knative.dev/pkg/apis/duck"

	"github.com/triggermesh/triggermesh/pkg/apis/sources/v1alpha1"
	clientv1alpha1 "github.com/triggermesh/triggermesh/pkg/client/generated/clientset/internalclientset/typed/sources/v1alpha1"
)

// NewPatcher returns a named Patcher scoped at the provided namespace and
// initialized from the client interface carried by ctx.
func NewPatcher(component string, cli clientv1alpha1.AWSSNSSourceInterface) *Patcher {
	return &Patcher{
		component: component,
		cli:       cli,
	}
}

// Patcher can apply patches to the status of source objects.
type Patcher struct {
	component string
	cli       clientv1alpha1.AWSSNSSourceInterface
}

// Patch applies the given JSON patch to the status of the source object
// referenced by name.
func (p *Patcher) Patch(ctx context.Context, name string, patch duck.JSONPatch) (*v1alpha1.AWSSNSSource, error) {
	jsonPatch, err := json.Marshal(patch)
	if err != nil {
		return nil, fmt.Errorf("applying JSON patch: %w", err)
	}

	opts := metav1.PatchOptions{
		FieldManager: p.component,
	}

	return p.cli.Patch(ctx, name, types.JSONPatchType, jsonPatch, opts, "status")
}

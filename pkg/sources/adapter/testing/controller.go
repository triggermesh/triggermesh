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

package testing

import (
	"testing"

	"knative.dev/eventing/pkg/adapter/v2"
	rt "knative.dev/pkg/reconciler/testing"

	"github.com/triggermesh/triggermesh/pkg/testing/structs"
)

// TestControllerConstructor tests that a controller constructor meets our requirements.
func TestControllerConstructor(t *testing.T, ctor adapter.ControllerConstructor, a adapter.Adapter) {
	t.Helper()

	defer func() {
		if r := recover(); r != nil {
			t.Errorf("Unexpected panic: %v", r)
		}
	}()

	ctx, informers := rt.SetupFakeContext(t)

	// expected informers: Source
	if expect, got := 1, len(informers); got != expect {
		t.Errorf("Expected %d injected informers, got %d", expect, got)
	}

	ctrler := ctor(ctx, a)

	// catch unitialized fields in Reconciler struct
	structs.EnsureNoNilField(t, ctrler)
}

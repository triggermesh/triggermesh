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

package transformation

import (
	"testing"

	. "github.com/triggermesh/triggermesh/pkg/reconciler/testing"

	// Link fake informers accessed by our controller
	_ "github.com/triggermesh/triggermesh/pkg/client/generated/injection/informers/flow/v1alpha1/transformation/fake"
	_ "knative.dev/pkg/client/injection/ducks/duck/v1/addressable/fake"
	_ "knative.dev/pkg/client/injection/kube/informers/core/v1/serviceaccount/fake"
	_ "knative.dev/pkg/client/injection/kube/informers/rbac/v1/rolebinding/fake"
	_ "knative.dev/serving/pkg/client/injection/informers/serving/v1/service/fake"
)

func TestNewController(t *testing.T) {
	t.Run("No failure", func(t *testing.T) {
		TestControllerConstructor(t, NewController)
	})

	t.Run("Failure cases", func(t *testing.T) {
		TestControllerConstructorFailures(t, NewController)
	})
}

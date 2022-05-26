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
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"

	"knative.dev/pkg/configmap"
	"knative.dev/pkg/injection"
	"knative.dev/pkg/logging"
	"knative.dev/pkg/metrics"
	rt "knative.dev/pkg/reconciler/testing"
	tracing "knative.dev/pkg/tracing/config"

	"github.com/triggermesh/triggermesh/pkg/testing/structs"
)

// TestControllerConstructor tests that a controller constructor meets our requirements.
func TestControllerConstructor(t *testing.T, ctor injection.ControllerConstructor, opts ...ctrlerCtorTestDecorator) {
	t.Helper()

	defer func() {
		if r := recover(); r != nil {
			t.Errorf("Unexpected panic: %v", r)
		}
	}()

	ctx, informers := rt.SetupFakeContext(t)

	// Informers that are always expected:
	// - TriggerMesh component kind (Source/Target/...)
	// - Adapter kind (Deployment/Kn Service)
	// - ServiceAccount
	// - RoleBinding
	expectInformers := 4

	for _, opt := range opts {
		switch t := reflect.TypeOf(opt); {
		case t == reflect.TypeOf(ExpectExtraInformers(0)):
			expectInformers += int(opt.(ExpectExtraInformers))
		}
	}

	if got := len(informers); got != expectInformers {
		t.Errorf("Expected %d injected informers, got %d", expectInformers, got)
	}

	// updateAdapterMetricsConfig panics when METRICS_DOMAIN is unset
	t.Setenv(metrics.DomainEnv, "testing")

	cmw := configmap.NewStaticWatcher(
		NewConfigMap(metrics.ConfigMapName(), nil),
		NewConfigMap(logging.ConfigMapName(), nil),
		NewConfigMap(tracing.ConfigName, nil),
	)

	ctrler := ctor(ctx, cmw)

	// catch unitialized fields in Reconciler struct
	structs.EnsureNoNilField(t, ctrler)
}

// TestControllerConstructorFailures tests that a controller constructor fails
// when various requirements are not met.
func TestControllerConstructorFailures(t *testing.T, ctor injection.ControllerConstructor) {
	t.Helper()

	testCases := map[string]struct {
		initFn   func(**configmap.StaticWatcher) (undo func())
		assertFn func(*testing.T, assert.PanicTestFunc)
	}{
		"Fails when watching missing ConfigMaps": {
			initFn: func(cmw **configmap.StaticWatcher) func() {
				*cmw = configmap.NewStaticWatcher()
				return nil
			},
			assertFn: func(t *testing.T, testFn assert.PanicTestFunc) {
				assert.PanicsWithValue(t, `Tried to watch unknown config with name "config-logging"`, testFn)
			},
		},
		"Fails when mandatory env var is missing": {
			initFn: func(cmw **configmap.StaticWatcher) func() {
				*cmw = configmap.NewStaticWatcher(
					NewConfigMap(metrics.ConfigMapName(), nil),
					NewConfigMap(logging.ConfigMapName(), nil),
				)
				return nil
			},
			assertFn: func(t *testing.T, testFn assert.PanicTestFunc) {
				assert.PanicsWithValue(t, "The environment variable \"METRICS_DOMAIN\" is not set\n\nIf this is a process running on Kubernetes, then it should be specifying\nthis via:\n\n  env:\n  - name: METRICS_DOMAIN\n    value: knative.dev/some-repository\n\nIf this is a Go unit test consuming metric.Domain() then it should add the\nfollowing import:\n\nimport (\n\t_ \"knative.dev/pkg/metrics/testing\"\n)", testFn)
			},
		},
	}

	for n, tc := range testCases {
		//nolint:scopelint
		t.Run(n, func(t *testing.T) {
			ctx, _ := rt.SetupFakeContext(t)
			cmw := &configmap.StaticWatcher{}

			undo := tc.initFn(&cmw)
			if undo != nil {
				defer undo()
			}

			tc.assertFn(t, func() {
				_ = ctor(ctx, cmw)
			})
		})
	}
}

type ctrlerCtorTestDecorator interface{}

type ExpectExtraInformers int

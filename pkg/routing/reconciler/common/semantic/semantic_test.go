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

package semantic

import (
	"encoding/json"
	"io/ioutil"
	"testing"

	"github.com/stretchr/testify/assert"

	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/runtime"

	servingv1 "knative.dev/serving/pkg/apis/serving/v1"
)

const (
	fixtureDeploymentPath = "../../../../test/fixtures/deployment.json"
	fixtureKnServicePath  = "../../../../test/fixtures/knService.json"
)

func TestDeploymentEqual(t *testing.T) {
	current := &appsv1.Deployment{}
	loadFixture(t, fixtureDeploymentPath, current)

	assert.GreaterOrEqual(t, len(current.Labels), 2,
		"Test suite requires a reference object with at least 2 labels to run properly")

	assert.True(t, deploymentEqual(nil, nil), "Two nil elements should be equal")

	testCases := map[string]struct {
		prep   func() *appsv1.Deployment
		expect bool
	}{
		"not equal when one element is nil": {
			func() *appsv1.Deployment {
				return nil
			},
			false,
		},
		// counter intuitive but expected result for deep derivative comparisons
		"equal when all desired attributes are empty": {
			func() *appsv1.Deployment {
				return &appsv1.Deployment{}
			},
			true,
		},
		"not equal when some existing attribute differs": {
			func() *appsv1.Deployment {
				desired := current.DeepCopy()
				for k := range desired.Labels {
					desired.Labels[k] += "test"
					break // changing one is enough
				}
				return desired
			},
			false,
		},
		"equal when current has more attributes than desired": {
			func() *appsv1.Deployment {
				desired := current.DeepCopy()
				for k := range desired.Labels {
					delete(desired.Labels, k)
					break // deleting one is enough
				}
				return desired
			},
			true,
		},
		"not equal when desired has more attributes than current": {
			func() *appsv1.Deployment {
				desired := current.DeepCopy()
				for k := range desired.Labels {
					desired.Labels[k+"test"] = "test"
					break // adding one is enough
				}
				return desired
			},
			false,
		},
	}

	for name, tc := range testCases {
		//nolint:scopelint
		t.Run(name, func(t *testing.T) {
			desired := tc.prep()
			switch tc.expect {
			case true:
				assert.True(t, deploymentEqual(desired, current))
			case false:
				assert.False(t, deploymentEqual(desired, current))
			}
		})
	}
}

func TestKnServiceEqual(t *testing.T) {
	current := &servingv1.Service{}
	loadFixture(t, fixtureKnServicePath, current)

	assert.GreaterOrEqual(t, len(current.Labels), 2,
		"Test suite requires a reference object with at least 2 labels to run properly")

	assert.True(t, knServiceEqual(nil, nil), "Two nil elements should be equal")

	testCases := map[string]struct {
		prep   func() *servingv1.Service
		expect bool
	}{
		"not equal when one element is nil": {
			func() *servingv1.Service {
				return nil
			},
			false,
		},
		// counter intuitive but expected result for deep derivative comparisons
		"equal when all desired attributes are empty": {
			func() *servingv1.Service {
				return &servingv1.Service{}
			},
			true,
		},
		"not equal when some existing attribute differs": {
			func() *servingv1.Service {
				desired := current.DeepCopy()
				for k := range desired.Labels {
					desired.Labels[k] += "test"
					break // changing one is enough
				}
				return desired
			},
			false,
		},
		"equal when current has more attributes than desired": {
			func() *servingv1.Service {
				desired := current.DeepCopy()
				for k := range desired.Labels {
					delete(desired.Labels, k)
					break // deleting one is enough
				}
				return desired
			},
			true,
		},
		"not equal when desired has more attributes than current": {
			func() *servingv1.Service {
				desired := current.DeepCopy()
				for k := range desired.Labels {
					desired.Labels[k+"test"] = "test"
					break // adding one is enough
				}
				return desired
			},
			false,
		},
	}

	for name, tc := range testCases {
		// //nolint:scopelint
		t.Run(name, func(t *testing.T) {
			desired := tc.prep()
			switch tc.expect {
			case true:
				assert.True(t, knServiceEqual(desired, current))
			case false:
				assert.False(t, knServiceEqual(desired, current))
			}
		})
	}
}

func loadFixture(t *testing.T, file string, obj runtime.Object) {
	t.Helper()

	data, err := ioutil.ReadFile(file)
	if err != nil {
		t.Fatalf("Error reading fixture file: %s", err)
	}

	if err := json.Unmarshal(data, obj); err != nil {
		t.Fatalf("Error deserializing fixture object: %s", err)
	}
}

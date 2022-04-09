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

package secret

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/fake"

	commonv1alpha1 "github.com/triggermesh/triggermesh/pkg/apis/common/v1alpha1"
)

func TestGetter(t *testing.T) {
	const ns = "fake-namespace"

	testCases := []struct {
		name        string
		initSecrets []*corev1.Secret
		input       []commonv1alpha1.ValueFromField
		expect      Secrets
		getRequests int
	}{
		{
			name:        "No input parameter",
			input:       []commonv1alpha1.ValueFromField{},
			expect:      nil,
			getRequests: 0,
		},
		{
			name: "All from value",
			input: []commonv1alpha1.ValueFromField{
				{
					Value: "value1",
				},
				{
					Value: "value2",
				},
			},
			expect: Secrets{
				"value1",
				"value2",
			},
			getRequests: 0,
		},
		{
			name: "One from value, another from secret",
			initSecrets: []*corev1.Secret{
				newSecret(ns, "secret", map[string]string{
					"key": "value from secret",
				}),
			},
			input: []commonv1alpha1.ValueFromField{
				{
					Value: "direct value",
				},
				{
					ValueFromSecret: &corev1.SecretKeySelector{
						LocalObjectReference: corev1.LocalObjectReference{
							Name: "secret",
						},
						Key: "key",
					},
				},
			},
			expect: Secrets{
				"direct value",
				"value from secret",
			},
			getRequests: 1,
		},
		{
			name: "Multiple from same secret",
			initSecrets: []*corev1.Secret{
				newSecret(ns, "secret", map[string]string{
					"key1": "value1",
					"key2": "value2",
				}),
			},
			input: []commonv1alpha1.ValueFromField{
				{
					ValueFromSecret: &corev1.SecretKeySelector{
						LocalObjectReference: corev1.LocalObjectReference{
							Name: "secret",
						},
						Key: "key1",
					},
				},
				{
					ValueFromSecret: &corev1.SecretKeySelector{
						LocalObjectReference: corev1.LocalObjectReference{
							Name: "secret",
						},
						Key: "key2",
					},
				},
			},
			expect: Secrets{
				"value1",
				"value2",
			},
			getRequests: 1,
		},
		{
			name: "Multiple from different secrets",
			initSecrets: []*corev1.Secret{
				newSecret(ns, "secret1", map[string]string{
					"key1": "value1",
				}),
				newSecret(ns, "secret2", map[string]string{
					"key2": "value2",
				}),
			},
			input: []commonv1alpha1.ValueFromField{
				{
					ValueFromSecret: &corev1.SecretKeySelector{
						LocalObjectReference: corev1.LocalObjectReference{
							Name: "secret1",
						},
						Key: "key1",
					},
				},
				{
					ValueFromSecret: &corev1.SecretKeySelector{
						LocalObjectReference: corev1.LocalObjectReference{
							Name: "secret2",
						},
						Key: "key2",
					},
				},
			},
			expect: Secrets{
				"value1",
				"value2",
			},
			getRequests: 2,
		},
	}

	for _, tc := range testCases {
		//nolint:scopelint
		t.Run(tc.name, func(t *testing.T) {
			secrets := make([]runtime.Object, len(tc.initSecrets))
			for i, secret := range tc.initSecrets {
				secrets[i] = secret
			}

			cli := fake.NewSimpleClientset(secrets...)

			sg := NewGetter(cli.CoreV1().Secrets(ns))
			output, err := sg.Get(tc.input...)

			require.NoError(t, err)

			assert.Equal(t, tc.expect, output)
			assert.Equal(t, tc.getRequests, len(cli.Actions()), "Unexpected number of API requests")
		})
	}
}

func newSecret(ns, name string, data map[string]string) *corev1.Secret {
	secr := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: ns,
			Name:      name,
		},
		Data: make(map[string][]byte, len(data)),
	}

	for k, v := range data {
		secr.Data[k] = []byte(v)
	}

	return secr
}

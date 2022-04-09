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

// Package secret contains utilities for consuming secret values from various
// data sources.
package secret

import (
	"context"
	"fmt"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	coreclientv1 "k8s.io/client-go/kubernetes/typed/core/v1"

	"github.com/triggermesh/triggermesh/pkg/apis/common/v1alpha1"
)

// Secrets is list of secret values.
type Secrets []string

// Getter can obtain secrets.
type Getter interface {
	// Get returns exactly one secret value per input.
	Get(...v1alpha1.ValueFromField) (Secrets, error)
}

// NewGetter returns a Getter for the given namespaced Secret client interface.
func NewGetter(cli coreclientv1.SecretInterface) *GetterWithClientset {
	return &GetterWithClientset{
		cli: cli,
	}
}

// GetterWithClientset gets Kubernetes secrets using a namespaced Secret client
// interface.
type GetterWithClientset struct {
	cli coreclientv1.SecretInterface
}

// GetterWithClientset implements Getter.
var _ Getter = (*GetterWithClientset)(nil)

// Get implements Getter.
func (g *GetterWithClientset) Get(refs ...v1alpha1.ValueFromField) (Secrets, error) {
	var s Secrets

	// cache Secret objects by name between iterations to avoid multiple
	// round trips to the Kubernetes API for the same Secret object.
	secretCache := make(map[string]*corev1.Secret)

	for _, ref := range refs {
		val := ref.Value

		if vfs := ref.ValueFromSecret; vfs != nil {
			var secr *corev1.Secret
			var err error

			if secretCache != nil && secretCache[vfs.Name] != nil {
				secr = secretCache[vfs.Name]
			} else {
				secr, err = g.cli.Get(context.Background(), vfs.Name, metav1.GetOptions{})
				if err != nil {
					return nil, fmt.Errorf("getting Secret from cluster: %w", err)
				}

				secretCache[vfs.Name] = secr
			}

			val = string(secr.Data[vfs.Key])
		}

		s = append(s, val)
	}

	return s, nil
}

// GetterFunc allows the use of ordinary functions as Getter.
type GetterFunc func(...v1alpha1.ValueFromField) (Secrets, error)

// GetterFunc implements Getter.
var _ Getter = (GetterFunc)(nil)

// Get implements Getter.
func (f GetterFunc) Get(refs ...v1alpha1.ValueFromField) (Secrets, error) {
	return f(refs...)
}

/*
Copyright 2021 TriggerMesh Inc.

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

package reconciler

import (
	"context"
	"fmt"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

// ValueGetter groups kubernetes value retrieval functions
type ValueGetter interface {
	FromSecret(context.Context, string, *corev1.SecretKeySelector) (string, error)
	FromConfig(context.Context, string, *corev1.ConfigMapKeySelector) (string, error)
}

// NewValueGetter returns the default implementation of ValueGetter
func NewValueGetter(kubeClientSet kubernetes.Interface) ValueGetter {
	return &valueGetter{
		kubeClientSet: kubeClientSet,
	}
}

// ValueGetter retrieves secrets and configmaps
type valueGetter struct {
	kubeClientSet kubernetes.Interface
}

// FromSecret retrieves value from secret
func (g *valueGetter) FromSecret(ctx context.Context, namespace string, secretKeySelector *corev1.SecretKeySelector) (string, error) {
	secret, err := g.kubeClientSet.CoreV1().Secrets(namespace).Get(ctx, secretKeySelector.Name, metav1.GetOptions{})
	if err != nil {
		return "", err
	}
	secretVal, ok := secret.Data[secretKeySelector.Key]
	if !ok {
		return "", fmt.Errorf(`key "%s" not found in secret "%s"`, secretKeySelector.Key, secretKeySelector.Name)
	}
	return string(secretVal), nil
}

// FromConfig retrieves value from config map
func (g *valueGetter) FromConfig(ctx context.Context, namespace string, configKeySelector *corev1.ConfigMapKeySelector) (string, error) {
	cm, err := g.kubeClientSet.CoreV1().ConfigMaps(namespace).Get(ctx, configKeySelector.Name, metav1.GetOptions{})
	if err != nil {
		return "", err
	}
	val, ok := cm.Data[configKeySelector.Key]
	if !ok {
		return "", fmt.Errorf(`key "%s" not found in configmap "%s"`, configKeySelector.Key, configKeySelector.Name)
	}
	return string(val), nil
}

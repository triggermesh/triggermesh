/*
Copyright 2020 TriggerMesh Inc.

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

package framework

import (
	"context"
	"time"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/apimachinery/pkg/watch"
	clientset "k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
	watchtools "k8s.io/client-go/tools/watch"
)

const namespacePrefix = "e2e-"

// CreateNamespace creates a namespace and marks it for automatic cleanup at
// the end of the test.
func (f *Framework) CreateNamespace(baseName string, l ...labels.Set) (*corev1.Namespace, error) {
	var lbls labels.Set
	if len(l) > 0 {
		lbls = l[0]
	}

	ns, err := createNamespace(f.KubeClient, baseName, lbls)
	f.AddNamespacesToDelete(ns)
	return ns, err
}

func createNamespace(c clientset.Interface, baseName string, labels labels.Set) (*corev1.Namespace, error) {
	// Generate a random name instead of setting ObjectMeta.GenerateName so
	// we can ensure the namespace is always deleted, even if the API call
	// to create it failed.
	var genName = func() string {
		return namespacePrefix + baseName + "-" + RandomSuffix()
	}
	name := genName()

	ns := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name:   name,
			Labels: labels,
		},
	}

	var createdNs *corev1.Namespace

	var createNsFunc wait.ConditionFunc = func() (bool, error) {
		var err error
		if createdNs, err = c.CoreV1().Namespaces().Create(context.Background(), ns, metav1.CreateOptions{}); err != nil {
			if apierrors.IsAlreadyExists(err) {
				Logf("Namespace %q already exists, will retry with a new name", ns.Name)
				ns.Name = genName()
			} else {
				Logf("Error while creating namespace %q, will retry: %v", ns.Name, err)
			}
			return false, nil
		}
		return true, nil
	}

	if err := wait.PollImmediate(200*time.Millisecond, 10*time.Second, createNsFunc); err != nil {
		// return ns despite the failure, to allow the caller to ensure
		// that the namespace is really absent.
		return ns, err
	}

	if err := WaitForDefaultServiceAccountInNamespace(c, createdNs.Name); err != nil {
		// Even if the serviceAccount creation failed, we still return
		// the successfully created namespace.
		Logf("Error waiting for the creation of the default ServiceAccount, ignoring: %v", err)
	}

	return createdNs, nil
}

func deleteNamespace(c clientset.Interface, name string) error {
	var deleteNsFunc wait.ConditionFunc = func() (bool, error) {
		err := c.CoreV1().Namespaces().Delete(context.Background(), name, metav1.DeleteOptions{})
		switch {
		case apierrors.IsNotFound(err):
			Logf("Namespace %q does not exist or was already deleted", name)
			return true, nil
		case err != nil:
			Logf("Error while deleting namespace %q, will retry: %v", name, err)
			return false, nil
		}

		return true, nil
	}

	return wait.PollImmediate(200*time.Millisecond, 10*time.Second, deleteNsFunc)
}

// WaitForDefaultServiceAccountInNamespace waits for the default ServiceAccount
// to be provisioned in the given namespace.
func WaitForDefaultServiceAccountInNamespace(c clientset.Interface, namespace string) error {
	return waitForServiceAccountInNamespace(c, namespace, "default")
}

// waitForServiceAccountInNamespace waits until the given ServiceAccount is
// provisioned in the given namespace.
func waitForServiceAccountInNamespace(c clientset.Interface, namespace, saName string) error {
	fieldSelector := fields.OneTermEqualSelector("metadata.name", saName).String()

	lw := &cache.ListWatch{
		ListFunc: func(options metav1.ListOptions) (runtime.Object, error) {
			options.FieldSelector = fieldSelector
			return c.CoreV1().ServiceAccounts(namespace).List(context.Background(), options)
		},
		WatchFunc: func(options metav1.ListOptions) (watch.Interface, error) {
			options.FieldSelector = fieldSelector
			return c.CoreV1().ServiceAccounts(namespace).Watch(context.Background(), options)
		},
	}

	// checks whether the ServiceAccount referenced in the given
	// watch.Event has at least one secret.
	var serviceAccountHasSecrets watchtools.ConditionFunc = func(e watch.Event) (bool, error) {
		if e.Type == watch.Deleted {
			return false, apierrors.NewNotFound(schema.GroupResource{Resource: "serviceaccounts"}, saName)
		}

		if sa, ok := e.Object.(*corev1.ServiceAccount); ok {
			return len(sa.Secrets) > 0, nil
		}

		return false, nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	_, err := watchtools.UntilWithSync(ctx, lw, &corev1.ServiceAccount{}, nil, serviceAccountHasSecrets)
	return err
}

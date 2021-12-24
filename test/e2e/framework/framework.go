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
	"fmt"

	"github.com/onsi/ginkgo"
	"github.com/onsi/gomega"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/labels"
	errorsutil "k8s.io/apimachinery/pkg/util/errors"
	"k8s.io/client-go/dynamic"
	clientset "k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

// Framework wraps the context and common operations for tests.
type Framework struct {
	// Test identifiers
	baseName   string // e.g. "mytest"
	UniqueName string // e.g. "mytest-1234"

	// Test context
	namespacesToDelete []*corev1.Namespace

	// API clients
	clientCfg     *rest.Config
	KubeClient    clientset.Interface
	DynamicClient dynamic.Interface
}

// New creates a test Framework.
func New(baseName string) *Framework {
	f := &Framework{
		baseName: baseName,
	}

	ginkgo.BeforeEach(f.BeforeEach)
	ginkgo.AfterEach(f.AfterEach)

	return f
}

// BeforeEach performs common initialization tasks.
func (f *Framework) BeforeEach() {
	// ensure the list of namespaces to delete is re-initialized in every
	// BeforeEach, otherwise test namespaces are appended to the list from
	// the previous test, etc.
	f.namespacesToDelete = nil

	ginkgo.By("creating test REST clients", func() {
		restCfg := getRESTConfig()
		f.clientCfg = restCfg
		f.KubeClient = getKubeClient(restCfg)
		f.DynamicClient = getDynamicClient(restCfg)
	})

	ginkgo.By("creating test namespace with base name "+f.baseName, func() {
		f.UniqueName = createTestNamespace(f).Name
	})
}

// AfterEach performs common cleanup tasks.
func (f *Framework) AfterEach() {
	// defer deletion of namespaces to avoid any intermediate failure from
	// short-circuiting this action.
	defer func() {
		deleteNsFuncs := make([]func() error, len(f.namespacesToDelete))

		for i, ns := range f.namespacesToDelete {
			ginkgo.By("marking test namespace "+ns.Name+" for deletion", func() {
				deleteNsFuncs[i] = func() error {
					if err := deleteNamespace(f.KubeClient, ns.Name); err != nil {
						return fmt.Errorf("failed to delete namespace %q: %v", ns.Name, err)
					}
					return nil
				}
			})
		}

		// offset level above caller
		// e.g. "AfterEach -> f -> ExpectWithOffset(2, ...)" will be logged for "AfterEach"
		const offset = 2
		if err := errorsutil.AggregateGoroutines(deleteNsFuncs...); err != nil {
			ginkgo.Fail(err.Error(), offset)
		}
	}()
}

// ClientConfig returns a copy of the Framework's rest.Config. Can be used to
// generate new API clients.
func (f *Framework) ClientConfig() *rest.Config {
	return rest.CopyConfig(f.clientCfg)
}

// AddNamespacesToDelete marks one or more namespaces for deletion when the
// test completes.
func (f *Framework) AddNamespacesToDelete(namespaces ...*corev1.Namespace) {
	for _, ns := range namespaces {
		if ns == nil {
			continue
		}
		f.namespacesToDelete = append(f.namespacesToDelete, ns)
	}
}

func getRESTConfig() *rest.Config {
	if kubeconfig := Config.Kubeconfig; kubeconfig != "" {
		cfg, err := clientcmd.BuildConfigFromFlags("", kubeconfig)
		expectNoError(err, "Unable to build client config from kubeconfig %s", kubeconfig)

		return cfg
	}

	// fall back to in-cluster config if neither the KUBECONFIG env var not
	// the kubeconfig flag was set
	cfg, err := rest.InClusterConfig()
	expectNoError(err, "Unable to build client config from in-cluster config")

	return cfg
}

func getKubeClient(cfg *rest.Config) clientset.Interface {
	cli, err := clientset.NewForConfig(cfg)
	expectNoError(err, "Unable to obtain Kubernetes ClientSet using the provided REST config")

	return cli
}

func getDynamicClient(cfg *rest.Config) dynamic.Interface {
	cli, err := dynamic.NewForConfig(cfg)
	expectNoError(err, "Unable to obtain dynamic ClientSet using the provided REST config")

	return cli
}

func createTestNamespace(f *Framework) *corev1.Namespace {
	ns, err := f.CreateNamespace(f.baseName, labels.Set{
		"e2e-framework": f.baseName,
	})
	expectNoError(err, "Failed to create test namespace: %v", err)

	return ns
}

// expectNoError is a convenience wrapper for failing functions called from
// whithin this package at the expected offset level.
func expectNoError(err error, context ...interface{}) {
	// offset level above caller
	// e.g. "BeforeEach -> f -> ExpectWithOffset(2, ...)" will be logged for "BeforeEach"
	const offset = 2
	gomega.ExpectWithOffset(offset, err).NotTo(gomega.HaveOccurred(), context...)
}

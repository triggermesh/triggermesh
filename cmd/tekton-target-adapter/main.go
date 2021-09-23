/*
Copyright (c) 2021 TriggerMesh Inc.

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

package main

import (
	"fmt"
	"os"
	"os/user"
	"path/filepath"

	"github.com/triggermesh/triggermesh/pkg/targets/adapter/tektontarget"

	pkgadapter "knative.dev/eventing/pkg/adapter/v2"

	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"

	"knative.dev/pkg/injection"
	"knative.dev/pkg/signals"
)

func main() {
	ctx := signals.NewContext()

	// Will need to load up the cluster configuration and inject the Tekton client
	config, err := configPath()
	if err != nil {
		fmt.Println("Unable to load configuration file: ", err)
		os.Exit(1)
	}

	ctx, _ = injection.Default.SetupInformers(ctx, config)

	pkgadapter.MainWithContext(ctx, "tekton-target-adapter", tektontarget.EnvAccessorCtor, tektontarget.NewTarget)
}

// Locate the cluster configuration for the adapter to properly instantiate the Tekton injector
func configPath() (*rest.Config, error) {
	kubeconfig := os.Getenv("KUBECONFIG")

	// If we have an explicit indication of where the kubernetes config lives, read that.
	if kubeconfig != "" {
		return clientcmd.BuildConfigFromFlags("", kubeconfig)
	}
	// If not, try the in-cluster config.
	if c, err := rest.InClusterConfig(); err == nil {
		return c, nil
	}
	// If no in-cluster config, try the default location in the user's home directory.
	if usr, err := user.Current(); err == nil {
		if c, err := clientcmd.BuildConfigFromFlags("", filepath.Join(usr.HomeDir, ".kube", "config")); err == nil {
			return c, nil
		}
	}

	return nil, fmt.Errorf("cannot obtain valid kubeconfig")
}

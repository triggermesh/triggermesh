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

package framework

import (
	"flag"
	"os"

	"k8s.io/client-go/tools/clientcmd"
)

// registerFlags registers command-line flags
// NOTE: `go test` calls flag.Parse() during normal operation.
func registerFlags() {
	stringFlag(&Config.Kubeconfig, clientcmd.RecommendedConfigPathFlag, os.Getenv(clientcmd.RecommendedConfigPathEnvVar),
		"Path to a kubeconfig file containing credentials to interact with a Kubernetes cluster.")
}

// Prefix prepended to all command-line flags declared by this test suite.
const flagPrefix = "e2e"

// stringFlag is a wrapper around flag.StringVar.
func stringFlag(varPtr *string, name, value, usage string) {
	flag.StringVar(varPtr, flagName(name), value, usage)
}

// flagName prepends the given flag name with flagPrefix.
func flagName(name string) string {
	return flagPrefix + "." + name
}

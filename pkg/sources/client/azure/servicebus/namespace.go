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

package servicebus

import (
	servicebus "github.com/Azure/azure-service-bus-go"
	"github.com/Azure/go-autorest/autorest/azure"
)

// Namespace is an alias for the servicebus.Namespace type.
type Namespace = servicebus.Namespace

// withName returns a NamespaceOption which sets the name of the Namespace.
func withName(n string) servicebus.NamespaceOption {
	return func(ns *servicebus.Namespace) error {
		ns.Name = n
		return nil
	}
}

// withAzureEnvironment returns a NamespaceOption which sets the environment of
// the Namespace.
func withAzureEnvironment(env *azure.Environment) servicebus.NamespaceOption {
	return func(ns *servicebus.Namespace) error {
		ns.Environment = *env
		ns.Suffix = env.ServiceBusEndpointSuffix
		return nil
	}
}

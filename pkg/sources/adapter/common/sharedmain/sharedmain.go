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

package sharedmain

import (
	"knative.dev/eventing/pkg/adapter/v2"
	"knative.dev/pkg/injection"
	"knative.dev/pkg/signals"

	"github.com/triggermesh/triggermesh/pkg/sources/adapter/common/env"
)

type namedControllerConstructor func(component string) adapter.ControllerConstructor
type namedAdapterConstructor func(component string) adapter.AdapterConstructor

// MainWithController is a shared main tailored to multi-tenant receive-adapters.
// It performs the following initializations:
//  * process environment variables
//  * enable leader election / HA
//  * set the scope to a single namespace
//  * inject the given controller constructor
func MainWithController(envCtor env.ConfigConstructor,
	cCtor namedControllerConstructor, aCtor namedAdapterConstructor) {

	envAcc := env.MustProcessConfig(envCtor)
	ns := envAcc.GetNamespace()
	component := envAcc.GetComponent()

	ctx := signals.NewContext()
	ctx = adapter.WithHAEnabled(ctx)
	ctx = injection.WithNamespaceScope(ctx, ns)
	ctx = adapter.WithController(ctx, cCtor(component))

	adapter.MainWithEnv(ctx, component, envAcc, aCtor(component))
}

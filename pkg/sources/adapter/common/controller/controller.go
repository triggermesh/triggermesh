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

// Package controller contains helpers shared between controllers embedded in
// source adapters.
package controller

import "knative.dev/pkg/controller"

// Opts returns a callback function that sets the controller's agent name and
// configures the reconciler to skip status updates.
func Opts(component string) controller.OptionsFn {
	return func(impl *controller.Impl) controller.Options {
		return controller.Options{
			AgentName:         component,
			SkipStatusUpdates: true,
		}
	}
}

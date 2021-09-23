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

// Package event contains functions for generating Kubernetes API events.
package event

import (
	"context"

	"k8s.io/apimachinery/pkg/runtime"
	"knative.dev/pkg/controller"
)

// Record event at the kubernetes API
func Record(ctx context.Context, owner runtime.Object, eventtype, reason, message string, args ...interface{}) {
	controller.GetEventRecorder(ctx).Eventf(owner, eventtype, reason, message, args...)
}

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

// Package event contains functions for generating Kubernetes API events.
package event

import (
	"context"

	corev1 "k8s.io/api/core/v1"
	"knative.dev/pkg/controller"

	"github.com/triggermesh/triggermesh/pkg/apis/common/v1alpha1"
)

// Normal records a normal event for an API object.
func Normal(ctx context.Context, reason, msgFmt string, args ...interface{}) {
	recordEvent(ctx, corev1.EventTypeNormal, reason, msgFmt, args...)
}

// Warn records a warning event for an API object.
func Warn(ctx context.Context, reason, msgFmt string, args ...interface{}) {
	recordEvent(ctx, corev1.EventTypeWarning, reason, msgFmt, args...)
}

func recordEvent(ctx context.Context, typ, reason, msgFmt string, args ...interface{}) {
	recorder := controller.GetEventRecorder(ctx)
	if recorder != nil {
		recorder.Eventf(v1alpha1.ReconcilableFromContext(ctx), typ, reason, msgFmt, args...)
	}
}

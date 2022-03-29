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

package filter

import (
	"context"

	"knative.dev/pkg/reconciler"

	"github.com/triggermesh/triggermesh/pkg/apis/routing/v1alpha1"
	reconcilerv1alpha1 "github.com/triggermesh/triggermesh/pkg/client/generated/injection/reconciler/routing/v1alpha1/filter"
)

// Reconciler implements controller.Reconciler for the event source type.
type Reconciler struct{}

// Check the interfaces Reconciler should implement.
var _ reconcilerv1alpha1.Interface = (*Reconciler)(nil)

// ReconcileKind implements reconcilerv1alpha1.Interface.
func (r *Reconciler) ReconcileKind(ctx context.Context, s *v1alpha1.Filter) reconciler.Event {
	return nil
}

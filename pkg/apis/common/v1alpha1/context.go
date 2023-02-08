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

package v1alpha1

import "context"

type reconcilableInstanceKey struct{}

// WithReconcilable returns a copy of the parent context in which the value
// associated with the reconcilableInstanceKey is the given component instance.
func WithReconcilable(ctx context.Context, r Reconcilable) context.Context {
	return context.WithValue(ctx, reconcilableInstanceKey{}, r)
}

// ReconcilableFromContext returns the component instance stored in the context.
func ReconcilableFromContext(ctx context.Context) Reconcilable {
	if r, ok := ctx.Value(reconcilableInstanceKey{}).(Reconcilable); ok {
		return r
	}
	return nil
}

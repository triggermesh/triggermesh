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

// Package skip allows a Context to carry the intention to skip parts of the
// code execution. Mainly used to avoid variances while testing certain
// functions.
package skip

import "context"

type skipKey struct{}

// EnableSkip returns a copy of a parent context with skipping enabled.
func EnableSkip(ctx context.Context) context.Context {
	return context.WithValue(ctx, skipKey{}, struct{}{})
}

// Skip returns whether the given context has skipping enabled.
func Skip(ctx context.Context) bool {
	return ctx.Value(skipKey{}) != nil
}

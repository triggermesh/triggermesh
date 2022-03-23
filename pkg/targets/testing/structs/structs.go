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

// Package structs provides helpers to test Go structs.
package structs

import (
	"reflect"
	"testing"

	"knative.dev/pkg/controller"
)

// EnsureNoNilField fails the test if the provided Impl's reconciler contains
// nil pointers or interfaces.
func EnsureNoNilField(t *testing.T, impl *controller.Impl) {
	t.Helper()

	recVal := reflect.ValueOf(impl.Reconciler).
		Elem().                    // injection/reconciler/sources/v1alpha1/<type>.reconcilerImpl
		FieldByName("reconciler"). // injection/reconciler/sources/v1alpha1/<type>.Interface
		Elem().                    // *pkg/reconciler/<type>.Reconciler (ptr)
		Elem()                     //  pkg/reconciler/<type>.Reconciler (val)

	for i := 0; i < recVal.NumField(); i++ {
		f := recVal.Field(i)
		switch f.Kind() {
		case reflect.Interface, reflect.Ptr, reflect.Func:
			if f.IsNil() {
				t.Errorf("struct field %q is nil", recVal.Type().Field(i).Name)
			}
		}
	}
}

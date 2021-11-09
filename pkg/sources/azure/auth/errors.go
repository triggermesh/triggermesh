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

package auth

// FatalCredentialsError allows callers to assert an error for behaviour.
//
// Examples of assertion:
//
//   _, ok := err.(FatalCredentialsError)
//
//   fatalErr := (FatalCredentialsError)(nil)
//   ok := errors.As(err, &fatalErr)
//
type FatalCredentialsError interface {
	error
	IsFatal()
}

// NewFatalCredentialsError marks an auth-related error as fatal (non retryable).
func NewFatalCredentialsError(err error) error {
	return unobtainableCredentialsError{e: err}
}

// unobtainableCredentialsError is an opaque error type that wraps another
// error to indicate that required Azure credentials could not be obtained.
//
// This allows callers to handle that special case if required, especially when
// the original error can not be asserted any other way because it is untyped.
// For example, Kubernetes finalizers are unlikely to be able to proceed when
// credentials can not be determined.
type unobtainableCredentialsError struct {
	e error
}

// IsFatal implements FatalCredentialsError.
func (unobtainableCredentialsError) IsFatal() {}

// Error implements the error interface.
func (e unobtainableCredentialsError) Error() string {
	if e.e == nil {
		return ""
	}
	return e.e.Error()
}

// Unwrap implements errors.Unwrap.
func (e unobtainableCredentialsError) Unwrap() error {
	return e.e
}

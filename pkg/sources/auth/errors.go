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

package auth

// PermanentCredentialsError is an error behaviour which signals that the
// interaction with an external service shouldn't be retried, due to
// credentials which are either invalid, expired, or missing permissions.
//
// This allows callers to handle that special case if required, especially when
// the original error can not be asserted any other way because it is untyped.
// For example, Kubernetes finalizers are unlikely to be able to proceed when
// credentials can not be determined.
//
// Examples of assertion:
//
//	_, ok := err.(PermanentCredentialsError)
//
//	permErr := (PermanentCredentialsError)(nil)
//	ok := errors.As(err, &permErr)
type PermanentCredentialsError interface {
	error
	IsPermanent()
}

// NewPermanentCredentialsError marks an auth-related error as permanent (non retryable).
func NewPermanentCredentialsError(err error) error {
	return permanentCredentialsError{e: err}
}

var _ PermanentCredentialsError = (*permanentCredentialsError)(nil)

// permanentCredentialsError is an opaque error type that wraps another error
// and implements the PermanentCredentialsError error behaviour.
type permanentCredentialsError struct {
	e error
}

// IsFatal implements FatalCredentialsError.
func (permanentCredentialsError) IsPermanent() {}

// Error implements the error interface.
func (e permanentCredentialsError) Error() string {
	if e.e == nil {
		return ""
	}
	return e.e.Error()
}

// Unwrap implements errors.Unwrap.
func (e permanentCredentialsError) Unwrap() error {
	return e.e
}

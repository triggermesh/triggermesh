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

package v1alpha1

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"knative.dev/pkg/apis"
)

var (
	errs                      = &apis.FieldError{}
	errXsltAndOrAllowOverride = errs.Also(apis.ErrGeneric("when XSLT is empty, per event XSLT must be allowed", "allowPerEventXslt", "xslt").ViaField("spec"))
	errXsltTooMany            = errs.Also(apis.ErrMultipleOneOf("value", "valueFromSecret", "valueFromConfigMap").ViaField("XSLT").ViaField("spec"))
)

func TestXsltTransformValidate(t *testing.T) {
	testCases := map[string]struct {
		xslt        *XsltTransform
		expectError *apis.FieldError
	}{
		"XSLT informed": {
			xslt:        xsltTransform(xsltWithXSLT(valueFromField(vffWithValue(tValue)))),
			expectError: nil,
		},
		"AllowOverride true": {
			xslt:        xsltTransform(xsltWithAllowEventXslt(true)),
			expectError: nil,
		},
		"XSLT and AllowOverride true": {
			xslt: xsltTransform(
				xsltWithXSLT(valueFromField(vffWithValue(tValue))),
				xsltWithAllowEventXslt(true)),
			expectError: nil,
		},
		"XSL nil and AllowOverride false": {
			xslt:        xsltTransform(xsltWithAllowEventXslt(false)),
			expectError: errXsltAndOrAllowOverride,
		},
		"XSLT empty and AllowOverride false": {
			xslt: xsltTransform(
				xsltWithXSLT(valueFromField()),
				xsltWithAllowEventXslt(false),
			),
			expectError: errXsltAndOrAllowOverride,
		},
		"XSLT nil and missing AllowOverride": {
			xslt:        xsltTransform(xsltWithXSLT(valueFromField())),
			expectError: errXsltAndOrAllowOverride,
		},

		"XSLT informed wrong": {
			xslt: xsltTransform(xsltWithXSLT(
				valueFromField(
					vffWithValue(tValue),
					vffWithSecret(tName, tKey),
				))),
			expectError: errXsltTooMany,
		},
	}

	for name, tc := range testCases {
		//nolint:scopelint
		t.Run(name, func(t *testing.T) {
			assert.Equal(t, tc.expectError, tc.xslt.Validate(context.Background()))
		})
	}
}

type xsltTransformOption func(*XsltTransform)

func xsltTransform(opts ...xsltTransformOption) *XsltTransform {
	xslt := &XsltTransform{}

	for _, o := range opts {
		o(xslt)
	}

	return xslt
}

func xsltWithXSLT(vff *ValueFromField) xsltTransformOption {
	return func(xslt *XsltTransform) {
		xslt.Spec.XSLT = vff
	}
}

func xsltWithAllowEventXslt(allowEventXslt bool) xsltTransformOption {
	return func(xslt *XsltTransform) {
		xslt.Spec.AllowPerEventXSLT = &allowEventXslt
	}
}

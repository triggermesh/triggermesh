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

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestXSLTTransformationSetDefaults(t *testing.T) {
	testCases := map[string]struct {
		xslt      *XSLTTransformation
		defaulted *XSLTTransformation
	}{
		"XSLT with allow event xslt value set to false, needs no defaulting": {
			xslt: xsltTransform(
				xsltWithXSLT(valueFromField(vffWithValue(tValue))),
				xsltWithAllowEventXSLT(false),
			),
			defaulted: xsltTransform(
				xsltWithXSLT(valueFromField(vffWithValue(tValue))),
				xsltWithAllowEventXSLT(false),
			),
		},
		"XSLT with allow event xslt value set to true, needs no defaulting": {
			xslt: xsltTransform(
				xsltWithXSLT(valueFromField(vffWithValue(tValue))),
				xsltWithAllowEventXSLT(true),
			),
			defaulted: xsltTransform(
				xsltWithXSLT(valueFromField(vffWithValue(tValue))),
				xsltWithAllowEventXSLT(true),
			),
		},
		"XSLT without allow event xslt value, needs defaulting": {
			xslt: xsltTransform(xsltWithXSLT(valueFromField(vffWithValue(tValue)))),
			defaulted: xsltTransform(
				xsltWithXSLT(valueFromField(vffWithValue(tValue))),
				xsltWithAllowEventXSLT(false),
			),
		},
		"XSLT nil does not defaulting": {
			xslt:      nil,
			defaulted: nil,
		},
	}

	for name, tc := range testCases {
		//nolint:scopelint
		t.Run(name, func(t *testing.T) {
			tc.xslt.SetDefaults(context.Background())
			assert.Equal(t, tc.defaulted, tc.xslt)
		})
	}
}

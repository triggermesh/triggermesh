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

package probe_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"

	"knative.dev/pkg/apis"

	"github.com/triggermesh/triggermesh/pkg/apis/sources/v1alpha1"
	"github.com/triggermesh/triggermesh/pkg/sources/adapter/awssnssource/probe"
	"github.com/triggermesh/triggermesh/pkg/sources/adapter/common/router"
	adaptesting "github.com/triggermesh/triggermesh/pkg/sources/adapter/testing"
)

func TestAdapterReadyChecker(t *testing.T) {
	testCases := map[string]struct {
		cachedSrcs         []runtime.Object
		registeredHandlers []string /*url path*/
		expectReady        bool
	}{
		"Ready when as many source handlers as sources with sink": {
			cachedSrcs: []runtime.Object{
				newSourceWithSink("src1"),
				newSourceWithSink("src2"),
				newSourceWithoutSink("src3"),
			},
			registeredHandlers: []string{
				"/test/src1",
				"/test/src2",
			},
			expectReady: true,
		},
		"Ready when more source handlers than sources with sink": {
			cachedSrcs: []runtime.Object{
				newSourceWithSink("src1"),
				newSourceWithoutSink("src2"),
				newSourceWithoutSink("src3"),
			},
			registeredHandlers: []string{
				"/test/src1",
				"/test/src4",
			},
			expectReady: true,
		},
		"Not ready when less source handlers than sources with sink": {
			cachedSrcs: []runtime.Object{
				newSourceWithSink("src1"),
				newSourceWithSink("src2"),
				newSourceWithoutSink("src3"),
			},
			registeredHandlers: []string{
				"/test/src1",
				"/health", // expected to be ignored
			},
			expectReady: false,
		},
		"Not ready when no source handler is registered": {
			cachedSrcs: []runtime.Object{
				newSourceWithSink("src1"),
			},
			registeredHandlers: []string{
				"/health", // expected to be ignored
			},
			expectReady: false,
		},
		"Not ready when informer cache is empty": {
			cachedSrcs: []runtime.Object{},
			registeredHandlers: []string{
				"/test/src1",
			},
			expectReady: false,
		},
	}

	for n, tc := range testCases {
		t.Run(n, func(t *testing.T) {
			ls := adaptesting.NewListers(adaptesting.NewScheme(), tc.cachedSrcs)
			l := ls.GetAWSSNSSourceLister()

			r := &router.Router{}
			for _, h := range tc.registeredHandlers {
				r.RegisterPath(h, nil)
			}

			c := probe.NewAdapterReadyChecker(l, r)

			isReady, err := c.IsReady()
			require.NoError(t, err, "IsReady shouldn't fail")

			if tc.expectReady {
				assert.True(t, isReady, "ReadinessChecker should be ready")
			} else {
				assert.False(t, isReady, "ReadinessChecker shouldn't be ready")
			}
		})
	}
}

func newSourceWithSink(name string) *v1alpha1.AWSSNSSource {
	src := newSourceWithoutSink(name)
	src.Status.SinkURI = apis.HTTP("example.com")
	return src
}

func newSourceWithoutSink(name string) *v1alpha1.AWSSNSSource {
	return &v1alpha1.AWSSNSSource{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "test", // irrelevant in the context of these tests
			Name:      name,
		},
	}
}

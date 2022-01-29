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

// Package probe contains facilities for asserting the readiness of a
// multi-tenant receive adapter.
package probe

import (
	"fmt"
	"sync"

	"k8s.io/apimachinery/pkg/labels"

	"github.com/triggermesh/triggermesh/pkg/apis/sources/v1alpha1"
	listersv1alpha1 "github.com/triggermesh/triggermesh/pkg/client/generated/listers/sources/v1alpha1"
	"github.com/triggermesh/triggermesh/pkg/sources/adapter/common/router"
)

// ReadinessChecker can assert the readiness of a component.
type ReadinessChecker interface {
	IsReady() (bool, error)
}

// AdapterReadyChecker asserts the readiness of a receive adapter.
type AdapterReadyChecker struct {
	srcLister listersv1alpha1.AWSSNSSourceLister

	srcRouter *router.Router

	sync.RWMutex
	isReady bool
}

var _ ReadinessChecker = (*AdapterReadyChecker)(nil)

// NewAdapterReadyChecker returns an AdapterReadyChecker initialized with the
// given Lister and Router.
func NewAdapterReadyChecker(ls listersv1alpha1.AWSSNSSourceLister, r *router.Router) *AdapterReadyChecker {
	return &AdapterReadyChecker{
		srcLister: ls,
		srcRouter: r,
	}
}

// IsReady implements ReadinessChecker.
// It checks whether the adapter has registered a handler for all observed sources.
//
// NOTE(antoineco): we might want to revisit this in the future, because the
// following implementation details are currently leaking into this
// ReadinessChecker implementation:
//
// - The fact that the adapter exposes a health endpoint via the HTTP router.
// - The exact URL path of this health endpoint.
// - The fact that a handler isn't registered for sources which don't have a valid sink.
//
// We consider this tight coupling acceptable for the time being, because this
// is a sub-package of the main adapter package for the SNS source
// specifically.
func (c *AdapterReadyChecker) IsReady() (bool, error) {
	// readiness already observed at an earlier point in time,
	// short-circuit the check
	if c.readLockedIsReady() {
		return true, nil
	}

	c.Lock()
	defer c.Unlock()

	// double-checked lock to ensure we don't write the value of "isReady"
	// twice if multiple goroutines called IsReady() simultaneously.
	if c.isReady {
		return true, nil
	}

	sources, err := c.srcLister.List(labels.Everything())
	if err != nil {
		return false, fmt.Errorf("listing cached sources: %w", err)
	}
	// informer cache wasn't populated yet
	if len(sources) == 0 {
		return false, nil
	}

	numCachedSrcsWithSink := len(filterSourcesWithSink(sources))
	numSrcHandlers := c.srcRouter.HandlersCount(ignoreHealthEndpoint)

	c.isReady = numSrcHandlers >= numCachedSrcsWithSink

	return c.isReady, nil
}

// readLockedIsReady returns the value of c.isReady while guaranteeing that
// this value isn't currently being written by another goroutine.
func (c *AdapterReadyChecker) readLockedIsReady() bool {
	c.RLock()
	defer c.RUnlock()

	return c.isReady
}

// filterSourcesWithSink filters all source objects which don't report a valid
// sink URL.
func filterSourcesWithSink(srcs []*v1alpha1.AWSSNSSource) []*v1alpha1.AWSSNSSource {
	srcsWithSink := make([]*v1alpha1.AWSSNSSource, 0, len(srcs))

	for _, src := range srcs {
		if src.Status.SinkURI != nil {
			srcsWithSink = append(srcsWithSink, src)
		}
	}

	return srcsWithSink
}

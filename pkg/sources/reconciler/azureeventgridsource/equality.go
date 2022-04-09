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

package azureeventgridsource

import (
	"context"
	"reflect"
	"sort"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"

	"knative.dev/pkg/logging"

	azureeventgrid "github.com/Azure/azure-sdk-for-go/profiles/latest/eventgrid/mgmt/eventgrid"
)

// equalEventSubscription asserts the equality of two EventSubscriptions.
func equalEventSubscription(ctx context.Context, x, y azureeventgrid.EventSubscription) bool {
	cmpFn := cmp.Equal
	if logger := logging.FromContext(ctx); logger.Desugar().Core().Enabled(zapcore.DebugLevel) {
		cmpFn = diffLoggingCmp(logger)
	}

	return cmpFn(x.EventSubscriptionProperties, y.EventSubscriptionProperties,
		// Exclude read-only fields from comparison
		cmpopts.IgnoreFields(azureeventgrid.EventSubscriptionProperties{},
			"Topic",
			"ProvisioningState",
		),

		// When users don't specify a list of event types to subscribe to,
		// Azure automatically selects _all_ available event types for the
		// provided scope. We need to handle this special case separately.
		cmp.FilterPath(matchFilterEventTypes, cmp.Comparer(sortedEqualEmptyStrSlice)),
	)
}

// matchFilterEventTypes is a path filter function which returns true if the
// visited cmp.Path is the "IncludedEventTypes" field of an
// azureeventgrid.EventSubscriptionFilter struct.
func matchFilterEventTypes(p cmp.Path) bool {
	t := reflect.TypeOf(azureeventgrid.EventSubscriptionFilter{})
	evenTypesFieldName := "IncludedEventTypes"

	// For p == "Filter.IncludedEventTypes", we expect the following PathSteps:
	//
	// i   p[i]                                       p[i].Type()
	// -   ----------------------------------------   -----------
	// 0   {*eventgrid.EventSubscriptionProperties}   *eventgrid.EventSubscriptionProperties
	// 1   *                                          eventgrid.EventSubscriptionProperties
	// 2   .Filter                                    *eventgrid.EventSubscriptionFilter
	// 3   *                                          eventgrid.EventSubscriptionFilter
	// 4   .IncludedEventTypes                        *[]string
	// 5   *                                          []string

	for i, ps := range p {
		if ps.Type().AssignableTo(t) {
			// isolate the cmp.Path of the next struct field
			nextPath := p[i+1:]

			for _, ps := range nextPath {
				if ps, ok := ps.(cmp.StructField); ok {
					if ps.Name() == evenTypesFieldName {
						return true
					}
				}
			}
		}
	}

	return false
}

// sortedEqualEmptyStrSlice is an order-agnostic comparer function for []string
// types which considers x and y to be equal if either x or y is empty.
func sortedEqualEmptyStrSlice(x, y []string) bool {
	if len(x) == 0 || len(y) == 0 {
		return true
	}
	if len(x) != len(y) {
		return false
	}

	xCpy := make([]string, len(x))
	yCpy := make([]string, len(y))
	copy(xCpy, x)
	copy(yCpy, y)

	sort.Strings(xCpy)
	sort.Strings(yCpy)

	for i := 0; i < len(xCpy); i++ {
		if xCpy[i] != yCpy[i] {
			return false
		}
	}

	return true
}

// cmpFunc can compare the equality of two interfaces. The function signature
// is the same as cmp.Equal.
type cmpFunc func(x, y interface{}, opts ...cmp.Option) bool

// diffLoggingCmp compares the equality of two interfaces and logs the diff at
// the Debug level.
func diffLoggingCmp(logger *zap.SugaredLogger) cmpFunc {
	return func(x, y interface{}, opts ...cmp.Option) bool {
		if diff := cmp.Diff(x, y, opts...); diff != "" {
			logger.Debug("Event subscriptions differ (-want, +got)\n" + diff)
			return false
		}
		return true
	}
}

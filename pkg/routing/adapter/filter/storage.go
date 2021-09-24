/*
Copyright 2021 Triggermesh Inc.

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

package filter

import (
	"sync"

	"github.com/triggermesh/triggermesh/pkg/routing/eventfilter/cel"
	"k8s.io/apimachinery/pkg/types"
)

type filterGenerations map[int64]cel.ConditionalFilter
type filterUIDs map[types.UID]filterGenerations

type expressionStorage struct {
	*sync.RWMutex
	filterUIDs
}

func newExpressionStorage() *expressionStorage {
	return &expressionStorage{
		RWMutex:    &sync.RWMutex{},
		filterUIDs: make(filterUIDs),
	}
}

func (f *expressionStorage) get(uid types.UID, generation int64) (cel.ConditionalFilter, bool) {
	f.RLock()
	defer f.RUnlock()

	filterGens, exist := f.filterUIDs[uid]
	if !exist {
		return cel.ConditionalFilter{}, false
	}

	filter, exist := filterGens[generation]
	return filter, exist
}

// set method overrides previous generations of compiled expressions
func (f *expressionStorage) set(uid types.UID, generation int64, condition cel.ConditionalFilter) {
	f.Lock()
	defer f.Unlock()

	f.filterUIDs[uid] = filterGenerations{
		generation: condition,
	}
}

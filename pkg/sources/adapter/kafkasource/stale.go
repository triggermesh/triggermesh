/*
Copyright 2023 TriggerMesh Inc.

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

package kafkasource

import (
	"sync"
	"time"
)

type item struct {
	object interface{}
	added  time.Time
}

// StaleList is a list of items that timeout lazily,
// only checking for item expiration when a new one is added.
//
// This should not be used for storing a big number of items.
type StaleList struct {
	items   []item
	timeout time.Duration
	m       sync.Mutex
}

func NewStaleList(timeout time.Duration) *StaleList {
	return &StaleList{
		items:   []item{},
		timeout: timeout,
	}
}

func (sl *StaleList) count() int {
	index := -1
	for i := range sl.items {
		if time.Since(sl.items[i].added) > sl.timeout {
			index = i
			continue
		}
		break
	}

	if index != -1 {
		sl.items = sl.items[index+1:]
	}

	return len(sl.items)
}

// AddAndCount adds a new element to the list and updates the count, removing
// any stale items from it.
func (sl *StaleList) AddAndCount(object interface{}) int {
	sl.m.Lock()
	defer sl.m.Unlock()

	sl.items = append(sl.items, item{
		added:  time.Now(),
		object: object,
	})

	return sl.count()
}

// Count updates the count removing any stale items from it.
func (sl *StaleList) Count() int {
	sl.m.Lock()
	defer sl.m.Unlock()

	return sl.count()
}

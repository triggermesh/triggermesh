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

package storage

import "sync"

// Storage is a simple object that provides thread safe
// methods to read and write into a map.
type Storage struct {
	data map[string]interface{}
	mux  sync.RWMutex
}

// New returns an instance of Storage.
func New() *Storage {
	return &Storage{
		data: make(map[string]interface{}),
		mux:  sync.RWMutex{},
	}
}

// Set writes a value interface to a string key.
func (s *Storage) Set(k string, v interface{}) {
	s.mux.Lock()
	s.data[k] = v
	s.mux.Unlock()
}

// Get reads value by a key.
func (s *Storage) Get(k string) interface{} {
	s.mux.RLock()
	defer s.mux.RUnlock()
	return s.data[k]
}

// ListKeys returns the slice of var keys stored in memory.
func (s *Storage) ListKeys() []string {
	s.mux.RLock()
	defer s.mux.RUnlock()
	list := []string{}
	for k := range s.data {
		list = append(list, k)
	}
	return list
}

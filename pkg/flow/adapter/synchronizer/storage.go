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

package synchronizer

import (
	"fmt"
	"sync"

	cloudevents "github.com/cloudevents/sdk-go/v2"
)

// storage holds the map of open connections and corresponding channels.
type storage struct {
	sync.Mutex
	sessions map[string]chan *cloudevents.Event
}

// newStorage returns an instance of the sessions storage.
func newStorage() *storage {
	return &storage{
		sessions: make(map[string]chan *cloudevents.Event),
	}
}

// add creates the new communication channel and adds it to the session storage.
func (s *storage) add(id string) (<-chan *cloudevents.Event, error) {
	s.Lock()
	defer s.Unlock()

	if _, exists := s.sessions[id]; exists {
		return nil, fmt.Errorf("session already exists")
	}

	c := make(chan *cloudevents.Event)
	s.sessions[id] = c
	return c, nil
}

// delete closes the communication channel and removes it from the storage.
func (s *storage) delete(id string) {
	s.Lock()
	defer s.Unlock()

	close(s.sessions[id])
	delete(s.sessions, id)
}

// open returns the communication channel for the session id.
func (s *storage) get(id string) (chan<- *cloudevents.Event, bool) {
	s.Lock()
	defer s.Unlock()

	session, exists := s.sessions[id]
	return session, exists
}

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

package fs

import (
	"context"
	"fmt"
	"sync"

	"github.com/triggermesh/triggermesh/pkg/adapter/fs"
)

type FakeFileWatcher interface {
	fs.FileWatcher
	DoCallback(path string) error
}

type fakeFileWatcher struct {
	watchedFiles map[string][]fs.WatchCallback

	m sync.RWMutex
}

func NewFileWatcher() FakeFileWatcher {
	return &fakeFileWatcher{
		watchedFiles: make(map[string][]fs.WatchCallback),
	}
}

func (cw *fakeFileWatcher) Start(_ context.Context) {}

func (cw *fakeFileWatcher) Add(path string, cb fs.WatchCallback) error {
	cw.m.Lock()
	defer cw.m.Unlock()

	if _, ok := cw.watchedFiles[path]; !ok {
		cw.watchedFiles[path] = []fs.WatchCallback{}
	}
	cw.watchedFiles[path] = append(cw.watchedFiles[path], cb)

	return nil
}

func (ccw *fakeFileWatcher) DoCallback(path string) error {
	ccw.m.RLock()
	defer ccw.m.RUnlock()

	cbs, ok := ccw.watchedFiles[path]
	if !ok {
		return fmt.Errorf("path %q is not being watched", path)
	}

	for _, cb := range cbs {
		cb()
	}
	return nil
}

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

type FakeCachedFileWatcher interface {
	fs.CachedFileWatcher
	SetContent(path string, content []byte) error
}

type fakeCachedFileWatcher struct {
	watchedFiles map[string][]byte

	m sync.RWMutex
}

func NewCachedFileWatcher() FakeCachedFileWatcher {
	return &fakeCachedFileWatcher{
		watchedFiles: make(map[string][]byte),
	}
}

func (ccw *fakeCachedFileWatcher) Start(_ context.Context) {}

func (ccw *fakeCachedFileWatcher) Add(path string) error {
	ccw.m.Lock()
	defer ccw.m.Unlock()

	if _, ok := ccw.watchedFiles[path]; !ok {
		ccw.watchedFiles[path] = nil
	}
	return nil
}

func (ccw *fakeCachedFileWatcher) GetContent(path string) ([]byte, error) {
	ccw.m.RLock()
	defer ccw.m.RUnlock()

	content, ok := ccw.watchedFiles[path]
	if !ok {
		return nil, fmt.Errorf("file %q is not being watched", path)
	}

	return content, nil
}

func (ccw *fakeCachedFileWatcher) SetContent(path string, content []byte) error {
	ccw.m.Lock()
	defer ccw.m.Unlock()

	if _, ok := ccw.watchedFiles[path]; !ok {
		return fmt.Errorf("file %q is not being watched", path)
	}

	ccw.watchedFiles[path] = content
	return nil
}

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
	"os"
	"sync"

	"go.uber.org/zap"
)

// CachedFileWatcher is a FileWatcher that caches and tracks the contents
// of watched files.
type CachedFileWatcher interface {
	// Start the FileWatcher process.
	Start(ctx context.Context)
	// Add a file path to be watched.
	Add(path string) error
	// GetContent of watched file.
	GetContent(path string) ([]byte, error)
}

type cachedFileWatcher struct {
	cw           FileWatcher
	watchedFiles map[string][]byte

	m      sync.RWMutex
	logger *zap.SugaredLogger
}

// NewCachedFileWatcher creates a new FileWatcher object that register files
// and calls back when they change.
func NewCachedFileWatcher(logger *zap.SugaredLogger) (CachedFileWatcher, error) {
	cw, err := NewWatcher(logger)
	if err != nil {
		return nil, err
	}

	return &cachedFileWatcher{
		watchedFiles: make(map[string][]byte),
		cw:           cw,
		logger:       logger,
	}, nil
}

// Start the FileWatcher process.
func (ccw *cachedFileWatcher) Start(ctx context.Context) {
	ccw.cw.Start(ctx)
}

// updateContentFromFile does not locks the watchedFiles map, it is up
// to the caller to do so.
func (ccw *cachedFileWatcher) updateContentFromFile(path string) error {
	content, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	ccw.watchedFiles[path] = content
	return nil
}

func (ccw *cachedFileWatcher) callback(path string) WatchCallback {
	return func() {
		ccw.m.Lock()
		defer ccw.m.Unlock()
		if err := ccw.updateContentFromFile(path); err != nil {
			ccw.logger.Errorw("Could not read watched file", zap.Error(err))
		}
	}
}

// Add a file path to be watched.
func (ccw *cachedFileWatcher) Add(path string) error {
	if err := ccw.cw.Add(path, ccw.callback(path)); err != nil {
		return err
	}

	ccw.m.Lock()
	defer ccw.m.Unlock()
	if _, ok := ccw.watchedFiles[path]; !ok {
		if err := ccw.updateContentFromFile(path); err != nil {
			ccw.logger.Errorw("Could not get content from file", zap.Error(err))
			// initialize to be able to distinguish paths not being watched
			// and those being watched but not available.
			ccw.watchedFiles[path] = nil
		}
	}

	return nil
}

// GetContent of watched file.
func (ccw *cachedFileWatcher) GetContent(path string) ([]byte, error) {
	ccw.m.RLock()
	defer ccw.m.RUnlock()

	content, ok := ccw.watchedFiles[path]
	if !ok {
		return nil, fmt.Errorf("file %q is not being watched", path)
	}

	return content, nil
}

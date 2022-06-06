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
	"sync"

	"github.com/fsnotify/fsnotify"
	"go.uber.org/zap"
)

// WatchCallback is called when a watched file
// is updated.
type WatchCallback func()

// FileWatcher object tracks changes to files.
type FileWatcher interface {
	Add(path string, cb WatchCallback) error
	Start(ctx context.Context)
}

type fileWatcher struct {
	watcher      *fsnotify.Watcher
	watchedFiles map[string][]WatchCallback

	m      sync.RWMutex
	start  sync.Once
	logger *zap.SugaredLogger
}

// NewWatcher creates a new FileWatcher object that register files
// and calls back when they change.
func NewWatcher(logger *zap.SugaredLogger) (FileWatcher, error) {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, err
	}

	return &fileWatcher{
		watcher:      watcher,
		watchedFiles: make(map[string][]WatchCallback),
		logger:       logger,
	}, nil
}

// Add path/callback tuple to the  FileWatcher.
func (cw *fileWatcher) Add(path string, cb WatchCallback) error {
	cw.m.Lock()
	defer cw.m.Unlock()

	if _, ok := cw.watchedFiles[path]; !ok {
		if err := cw.watcher.Add(path); err != nil {
			return err
		}
		cw.watchedFiles[path] = []WatchCallback{cb}
		return nil
	}

	cw.watchedFiles[path] = append(cw.watchedFiles[path], cb)
	return nil
}

// Start the FileWatcher process.
func (cw *fileWatcher) Start(ctx context.Context) {
	cw.start.Do(func() {
		// Do not block, exit on context done.
		go func() {
			defer cw.watcher.Close()
			for {
				select {
				case e, ok := <-cw.watcher.Events:
					if !ok {
						// watcher event channel finished
						return
					}

					cw.m.RLock()
					cbs, ok := cw.watchedFiles[e.Name]
					if !ok {
						cw.logger.Warn("Received a notification for a non watched file")
					}

					for _, cb := range cbs {
						cb()
					}
					cw.m.RUnlock()

				case err, ok := <-cw.watcher.Errors:
					if !ok {
						// watcher error channel finished
						return
					}
					cw.logger.Errorw("Error watching files", zap.Error(err))

				case <-ctx.Done():
					cw.logger.Debug("Exiting file watcher process")
					return
				}
			}
		}()
	})
}

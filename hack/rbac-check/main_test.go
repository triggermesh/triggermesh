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

package main

import (
	"bytes"
	"io/fs"
	"sort"
	"testing"
)

func TestComputeDiffs(t *testing.T) {
	filesystemOrig := filesystem
	filesystem = populatedTestFS()
	t.Cleanup(func() { filesystem = filesystemOrig })

	diffs, err := computeDiffs("") // the in-memory fs implementation has no notion of directory
	if err != nil {
		t.Fatal("Error computing diffs:", err)
	}

	expectDiffs := []struct {
		roleName string
		diffLine int
	}{{
		// node with too many resources
		roleName: "test-role-1",
		diffLine: 15,
	}, {
		// node with misspelled subresources
		roleName: "test-role-1",
		diffLine: 41,
	}, {
		// node with missing resources
		roleName: "test-role-2",
		diffLine: 67,
	}}

	if expectNumDiffs, nDiffs := len(expectDiffs), len(diffs); nDiffs != expectNumDiffs {
		t.Fatalf("Expected %d diffs, got %d:\n%s", expectNumDiffs, nDiffs, diffsText(diffs))
	}

	for i, expectDiff := range expectDiffs {
		if expectRoleName, roleName := expectDiff.roleName, diffs[i].objectName; roleName != expectRoleName {
			t.Errorf("Expected diff %d to be in ClusterRole %s, got %s:\n%s",
				i, expectRoleName, roleName, diffs[i])
		}
		if expectDiffLine, diffLine := expectDiff.diffLine, diffs[i].yamlDocLine; diffLine != expectDiffLine {
			t.Errorf("Expected diff %d to have occured at line %d, got %d:\n%s",
				i, expectDiffLine, diffLine, diffs[i])
		}
	}
}

/*
   Test data
*/

// populatedTestFS returns a filesystem populated with test data.
func populatedTestFS() fs.ReadDirFS {
	emptyFile := make([]byte, 0)

	return memFS{
		"200-clusterroles.yaml": []byte(testClusterRolesFile),
		"300-aaasource.yaml":    emptyFile,
		"300-bbbsource.yaml":    emptyFile,
		"301-aaatarget.yaml":    emptyFile,
		"301-bbbtarget.yaml":    emptyFile,
		"302-aaarouter.yaml":    emptyFile,
		"302-bbbrouter.yaml":    emptyFile,
		"303-aaaextension.yaml": emptyFile,
		"303-bbbextension.yaml": emptyFile,
		"304-aaaflow.yaml":      emptyFile,
		"304-bbbflow.yaml":      emptyFile,
	}
}

const testClusterRolesFile = `
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRole
metadata:
  name: test-role-1
rules:

# This node contains too many resources.
# +rbac-check
- apiGroups:
  - sources.triggermesh.io
  - targets.triggermesh.io
  - routing.triggermesh.io
  resources:
  - aaasources
  - bbbsources
  - zzzsources
  - aaatargets
  - bbbtargets
  - aaarouters
  - zzzrouters
  - bbbrouters
  verbs:
  - get

# This node is not flagged and is expected to be ignored.
- apiGroups:
  - sources.triggermesh.io
  resources:
  - xyzsources
  verbs:
  - get

# This node contains misspelled subresources.
# +rbac-check:subresource=status
- apiGroups:
  - routing.triggermesh.io
  - extensions.triggermesh.io
  - flow.triggermesh.io
  resources:
  - aaarouters/status
  - bbbrouters/status
  - aaaextensions/status
  - bbbextensions/zzzzzzzzzzzzzzzzz
  - aaaflows/status
  - bbbflows/zzzzzzzzzzzzzzzzz
  verbs:
  - get

---

apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRole
metadata:
  name: test-role-2
rules:

# This node has missing components.
# +rbac-check
- apiGroups:
  - sources.triggermesh.io
  - targets.triggermesh.io
  - routing.triggermesh.io
  - extensions.triggermesh.io
  - flow.triggermesh.io
  resources:
  - aaasources
  - aaatargets
  - bbbtargets
  - aaarouters
  - aaaextensions
  - aaaflows
  - bbbflows
  verbs:
  - get
`

/*
   In-memory filesystem implementation for tests
*/

// memFS is an in-memory fs.FS implementation backed by a map of files indexed
// by name (path).
type memFS map[string][]byte

var _ fs.ReadDirFS = (memFS)(nil)

// Open implements fs.ReadDirFS.
func (mfs memFS) Open(name string) (fs.File, error) {
	f, exists := mfs[name]
	if !exists {
		return nil, &fs.PathError{Op: "read", Path: name, Err: fs.ErrNotExist}
	}

	return &memFD{b: *bytes.NewBuffer(f)}, nil
}

// ReadDirFS implements fs.ReadDirFS.
func (mfs memFS) ReadDir(name string) ([]fs.DirEntry, error) {
	dirEntries := make([]fs.DirEntry, 0, len(mfs))

	fileNames := make([]string, 0, len(mfs))
	for name := range mfs {
		fileNames = append(fileNames, name)
	}
	sort.Strings(fileNames)

	for _, name := range fileNames {
		dirEntries = append(dirEntries, &memDirEntry{
			name: name,
		})
	}

	return dirEntries, nil
}

// memFD is an in-memory representation of a file descriptor backed by a
// bytes.Buffer.
type memFD struct {
	b bytes.Buffer
}

var _ fs.File = (*memFD)(nil)

// Read implements fs.File and satisfies io.Reader.
func (fd *memFD) Read(b []byte) (int, error) {
	return fd.b.Read(b)
}

// Close implements fs.File.
func (fd *memFD) Close() error {
	fd.b.Reset()
	return nil
}

// Stat implements fs.File.
func (*memFD) Stat() (fs.FileInfo, error) {
	panic("not implemented")
}

// memDirEntry is an in-memory representation of a
type memDirEntry struct {
	name string
}

var _ fs.DirEntry = (*memDirEntry)(nil)

// Name implements fs.DirEntry.
func (d *memDirEntry) Name() string {
	return d.name
}

// IsDir implements fs.DirEntry.
func (d *memDirEntry) IsDir() bool {
	panic("not implemented")
}

// Type implements fs.DirEntry.
func (d *memDirEntry) Type() fs.FileMode {
	return 0 // assume only regular files
}

// Info implements fs.DirEntry.
func (*memDirEntry) Info() (fs.FileInfo, error) {
	panic("not implemented")
}

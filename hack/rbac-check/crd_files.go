/*
Copyright 2021 TriggerMesh Inc.

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
	"path"
)

// Prefixes for CRD manifest files.
const (
	filePrefixLen = 4

	crdPrefixSources    = "300-"
	crdPrefixTargets    = "301-"
	crdPrefixRouting    = "302-"
	crdPrefixExtensions = "303-"
	crdPrefixFlow       = "304-"
)

// crdFilenameToResource extracts the singular resource name from the given
// file name.
// The given file name is expected to be valid (for example using isCRDFile).
func crdFilenameToResource(name string) string {
	return name[filePrefixLen : len(name)-len(path.Ext(name))]
}

// isCRDFile asserts that the given file name corresponds to
// a manifest file for a CRD.
func isCRDFile(name string) bool {
	const expectPattern = "3xx-*.yaml"
	if len(name) < len(expectPattern) {
		return false
	}

	// starts with /3[0-9]{2}/
	if name[0] != '3' || !isDigit(name[1]) || !isDigit(name[2]) {
		return false
	}

	// ends with '.yaml'
	return path.Ext(name) == ".yaml"
}

// isDigit reports whether the given char is a digit in the
// latin unicode character range.
func isDigit(c byte) bool {
	return '0' <= c && c <= '9'
}

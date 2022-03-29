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

import "fmt"

// TriggerMesh's API groups.
const (
	apiGroupSources    = "sources.triggermesh.io"
	apiGroupTargets    = "targets.triggermesh.io"
	apiGroupRouting    = "routing.triggermesh.io"
	apiGroupExtensions = "extensions.triggermesh.io"
	apiGroupFlow       = "flow.triggermesh.io"
)

// componentsDictionary is a dictionary of component names indexed by API group.
type componentsDictionary map[ /*API group*/ string][]string

// AddSource adds a resource to the dictionary's "sources" apiGroup.
func (d componentsDictionary) AddSource(resource string) {
	d[apiGroupSources] = append(d[apiGroupSources], resource)
}

// AddTarget adds a resource to the dictionary's "targets" apiGroup.
func (d componentsDictionary) AddTarget(resource string) {
	d[apiGroupTargets] = append(d[apiGroupTargets], resource)
}

// AddRouter adds a resource to the dictionary's "routers" apiGroup.
func (d componentsDictionary) AddRouter(resource string) {
	d[apiGroupRouting] = append(d[apiGroupRouting], resource)
}

// AddExtension adds a resource to the dictionary's "extensions" apiGroup.
func (d componentsDictionary) AddExtension(resource string) {
	d[apiGroupExtensions] = append(d[apiGroupExtensions], resource)
}

// AddFlow adds a resource to the dictionary's "flows" apiGroup.
func (d componentsDictionary) AddFlow(resource string) {
	d[apiGroupFlow] = append(d[apiGroupFlow], resource)
}

// readComponents returns a componentsDictionary populated with the resources
// discovered in the given config directory.
func readComponents(dir string) (componentsDictionary, error) {
	dirEntries, err := filesystem.ReadDir(dir)
	if err != nil {
		return nil, err
	}

	components := make(componentsDictionary)

	for _, e := range dirEntries {
		fileName := e.Name()

		// ignore non-regular files (e.g. symlinks, directories)
		// ignore non-CRD manifests (3xx-*.yaml)
		if !e.Type().IsRegular() || !isCRDFile(fileName) {
			continue
		}

		resource := crdFilenameToResource(fileName)

		switch prefix := fileName[:filePrefixLen]; prefix {
		case crdPrefixSources:
			components.AddSource(resource)
		case crdPrefixTargets:
			components.AddTarget(resource)
		case crdPrefixRouting:
			components.AddRouter(resource)
		case crdPrefixExtensions:
			components.AddExtension(resource)
		case crdPrefixFlow:
			components.AddFlow(resource)
		default:
			// This shouldn't happen.
			// Fail loudly if we find a file with an unknown prefix.
			panic(fmt.Errorf("undefined file prefix %q in file name %s", prefix, fileName))
		}
	}

	return components, nil
}

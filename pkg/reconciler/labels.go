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

package reconciler

// Kubernetes recommended labels
// https://kubernetes.io/docs/concepts/overview/working-with-objects/common-labels/
const (
	// appNameLabel is the name of the application.
	appNameLabel = "app.kubernetes.io/name"
	// appInstanceLabel is a unique name identifying the instance of an application.
	appInstanceLabel = "app.kubernetes.io/instance"
	// appComponentLabel is the component within the architecture.
	appComponentLabel = "app.kubernetes.io/component"
	// appPartOfLabel is the name of a higher level application this one is part of.
	appPartOfLabel = "app.kubernetes.io/part-of"
	// appManagedByLabel is the tool being used to manage the operation of an application.
	appManagedByLabel = "app.kubernetes.io/managed-by"
)

// Common label values
const (
	partOf           = "triggermesh"
	managedBy        = "triggermesh-controller"
	componentAdapter = "adapter"
)

// labelsPropagationList is a list of labels that, if present on the parent
// object, should be propagated to the adapters.
var labelsPropagationList = []string{
	"bridges.triggermesh.io/id",
	"flow.triggermesh.io/created-by",
}

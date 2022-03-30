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

// Reasons for API Events
const (
	// ReasonRBACCreate indicates that an RBAC object was successfully created.
	ReasonRBACCreate = "CreateRBAC"
	// ReasonRBACUpdate indicates that an RBAC object was successfully updated.
	ReasonRBACUpdate = "UpdateRBAC"
	// ReasonFailedRBACCreate indicates that the creation of an RBAC object failed.
	ReasonFailedRBACCreate = "FailedRBACCreate"
	// ReasonFailedRBACUpdate indicates that the update of an RBAC object failed.
	ReasonFailedRBACUpdate = "FailedRBACUpdate"

	// ReasonAdapterCreate indicates that an adapter object was successfully created.
	ReasonAdapterCreate = "CreateAdapter"
	// ReasonAdapterUpdate indicates that an adapter object was successfully updated.
	ReasonAdapterUpdate = "UpdateAdapter"
	// ReasonFailedAdapterCreate indicates that the creation of an adapter object failed.
	ReasonFailedAdapterCreate = "FailedAdapterCreate"
	// ReasonFailedAdapterUpdate indicates that the update of an adapter object failed.
	ReasonFailedAdapterUpdate = "FailedAdapterUpdate"

	// ReasonBadSinkURI indicates that the URI of a sink can't be determined.
	ReasonBadSinkURI = "BadSinkURI"

	// ReasonInvalidSpec indicates that spec of a reconciled object is invalid.
	ReasonInvalidSpec = "InvalidSpec"
)

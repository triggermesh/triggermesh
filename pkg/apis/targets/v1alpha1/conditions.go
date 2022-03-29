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

package v1alpha1

import "knative.dev/pkg/apis"

// status conditions
const (
	// ConditionReady has status True when the target is ready to receive events.
	ConditionReady = apis.ConditionReady
	// ConditionDeployed has status True when the target's adapter is up and running.
	ConditionDeployed apis.ConditionType = "Deployed"
)

// reasons for conditions
const (
	// ReasonRBACNotBound is set on a Deployed condition when an adapter's
	// ServiceAccount cannot be bound.
	ReasonRBACNotBound = "RBACNotBound"
	// ReasonUnavailable is set on a Deployed condition when an adapter is unavailable.
	ReasonUnavailable = "AdapterUnavailable"
)

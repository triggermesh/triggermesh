/*
Copyright (c) 2020-2021 TriggerMesh Inc.

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

// Status conditions
const (
	// ConditionReady has status True when the router is ready to send events.
	ConditionReady = apis.ConditionReady
	// ConditionSinkProvided has status True when the router has been configured with a sink target.
	ConditionSinkProvided apis.ConditionType = "SinkProvided"
	// ConditionDeployed has status True when the router's adapter is up and running.
	ConditionDeployed apis.ConditionType = "Deployed"
)

// Reasons for status conditions
const (
	// ReasonSinkNotFound is set on a SinkProvided condition when a sink does not exist.
	ReasonSinkNotFound = "SinkNotFound"
	// ReasonSinkEmpty is set on a SinkProvided condition when a sink URI is empty.
	ReasonSinkEmpty = "EmptySinkURI"

	// ReasonRBACNotBound is set on a Deployed condition when an adapter's
	// ServiceAccount cannot be bound.
	ReasonRBACNotBound = "RBACNotBound"
	// ReasonUnavailable is set on a Deployed condition when an adapter in unavailable.
	ReasonUnavailable = "AdapterUnavailable"
)

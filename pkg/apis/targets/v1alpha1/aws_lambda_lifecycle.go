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

package v1alpha1

import (
	"k8s.io/apimachinery/pkg/runtime/schema"
	"knative.dev/pkg/apis"
	duckv1 "knative.dev/pkg/apis/duck/v1"
	servingv1 "knative.dev/serving/pkg/apis/serving/v1"
)

// AwsLambdaCondSet is the group of possible conditions
var AwsLambdaCondSet = apis.NewLivingConditionSet(
	ConditionServiceReady,
	ConditionSecretsProvided,
)

// GetCondition returns the condition currently associated with the given type, or nil.
func (a *AWSLambdaTargetStatus) GetCondition(t apis.ConditionType) *apis.Condition {
	return AwsLambdaCondSet.Manage(a).GetCondition(t)
}

// InitializeConditions sets relevant unset conditions to Unknown state.
func (a *AWSLambdaTargetStatus) InitializeConditions() {
	AwsLambdaCondSet.Manage(a).InitializeConditions()
	a.Address = &duckv1.Addressable{}
}

// GetGroupVersionKind returns the GroupVersionKind.
func (a *AWSLambdaTarget) GetGroupVersionKind() schema.GroupVersionKind {
	return SchemeGroupVersion.WithKind("AWSLambdaTarget")
}

// PropagateKServiceAvailability uses the availability of the provided KService to determine if
// ConditionServiceReady should be marked as true or false.
func (a *AWSLambdaTargetStatus) PropagateKServiceAvailability(ksvc *servingv1.Service) {
	if ksvc != nil && ksvc.IsReady() {
		a.Address.URL = ksvc.Status.Address.URL
		AwsLambdaCondSet.Manage(a).MarkTrue(ConditionServiceReady)
		return
	} else if ksvc == nil {
		a.MarkNoKService(ReasonUnavailable, "Adapter service unknown: ksvc is not available")
	} else {
		a.MarkNoKService(ReasonUnavailable, "Adapter service \"%s/%s\" is unavailable", ksvc.Namespace, ksvc.Name)
	}
	a.Address.URL = nil
}

// MarkNoKService sets the condition that the service is not ready
func (a *AWSLambdaTargetStatus) MarkNoKService(reason, messageFormat string, messageA ...interface{}) {
	AwsLambdaCondSet.Manage(a).MarkFalse(ConditionServiceReady, reason, messageFormat, messageA...)
}

// IsReady returns true if the resource is ready overall.
func (a *AWSLambdaTargetStatus) IsReady() bool {
	return AwsLambdaCondSet.Manage(a).IsHappy()
}

// MarkSecrets sets the condition that the resource is valid when the associated secrets are provided
func (a *AWSLambdaTargetStatus) MarkSecrets() {
	AwsLambdaCondSet.Manage(a).MarkTrue(ConditionSecretsProvided)
}

// MarkNoSecrets sets the condition that the resource is not valid  when the associated secrets are missing
func (a *AWSLambdaTargetStatus) MarkNoSecrets(reason, messageFormat string, messageA ...interface{}) {
	AwsLambdaCondSet.Manage(a).MarkFalse(ConditionSecretsProvided, reason, messageFormat, messageA...)
}

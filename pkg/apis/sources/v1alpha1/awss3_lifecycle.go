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

import (
	"context"
	"sort"
	"strings"

	"k8s.io/apimachinery/pkg/runtime/schema"

	"knative.dev/pkg/apis"
	duckv1 "knative.dev/pkg/apis/duck/v1"

	"github.com/triggermesh/triggermesh/pkg/apis/common/v1alpha1"
	"github.com/triggermesh/triggermesh/pkg/reconciler/resource"
	"github.com/triggermesh/triggermesh/pkg/sources/aws/s3"
)

// GetGroupVersionKind implements kmeta.OwnerRefable.
func (s *AWSS3Source) GetGroupVersionKind() schema.GroupVersionKind {
	return SchemeGroupVersion.WithKind("AWSS3Source")
}

// GetConditionSet implements duckv1.KRShaped.
func (s *AWSS3Source) GetConditionSet() apis.ConditionSet {
	return awsS3SourceConditionSet
}

// GetStatus implements duckv1.KRShaped.
func (s *AWSS3Source) GetStatus() *duckv1.Status {
	return &s.Status.Status.Status
}

// GetSink implements EventSender.
func (s *AWSS3Source) GetSink() *duckv1.Destination {
	return &s.Spec.Sink
}

// GetStatusManager implements Reconcilable.
func (s *AWSS3Source) GetStatusManager() *v1alpha1.StatusManager {
	return &v1alpha1.StatusManager{
		ConditionSet: s.GetConditionSet(),
		Status:       &s.Status.Status,
	}
}

// SetDefaults implements apis.Defaultable
func (s *AWSS3Source) SetDefaults(ctx context.Context) {
}

// Validate implements apis.Validatable
func (s *AWSS3Source) Validate(ctx context.Context) *apis.FieldError {
	// Do not validate authentication object in case of resource deletion
	if s.DeletionTimestamp != nil {
		return nil
	}
	return s.Spec.Auth.Validate(ctx)
}

// Supported event types (see AWSS3SourceSpec)
const (
	AWSS3ObjCreatedEventType               = "objectcreated"
	AWSS3ObjRemovedEventType               = "objectremoved"
	AWSS3ObjRestoreEventType               = "objectrestore"
	AWSS3ReducedRedundancyLostObjEventType = "reducedredundancylostobject"
	AWSS3ReplicationEventType              = "replication"
	AWSS3TestEventType                     = "testevent"
)

// GetEventTypes implements EventSource.
func (s *AWSS3Source) GetEventTypes() []string {
	selectedTypes := make(map[string]struct{})
	for _, t := range s.Spec.EventTypes {
		if _, alreadySet := selectedTypes[s3EventTypeFromSpecEventType(t)]; !alreadySet {
			selectedTypes[s3EventTypeFromSpecEventType(t)] = struct{}{}
		}
	}

	eventTypes := make([]string, 0, len(selectedTypes)+1)

	// s3:TestEvent is always sent when event notifications are enabled/updated
	eventTypes = append(eventTypes, AWSEventType(s.Spec.ARN.Service, AWSS3TestEventType))

	for t := range selectedTypes {
		eventTypes = append(eventTypes, AWSEventType(s.Spec.ARN.Service, t))
	}

	sort.Strings(eventTypes)

	return eventTypes
}

// s3EventTypeFromSpecEventType returns the type element of an event type
// formatted as "s3:<type>:<other>", which is the format expected in the
// object's spec.
// The returned element should match the value of one of the "*EventType"
// constants declared in this file.
func s3EventTypeFromSpecEventType(specEventType string) string {
	// Example: "s3:ObjectCreated:*" -> "objectcreated"
	return strings.ToLower(strings.SplitN(strings.TrimPrefix(specEventType, "s3:"), ":", 2)[0])
}

// AsEventSource implements EventSource.
func (s *AWSS3Source) AsEventSource() string {
	return s3.RealBucketARN(s.Spec.ARN)
}

// GetAdapterOverrides implements AdapterConfigurable.
func (s *AWSS3Source) GetAdapterOverrides() *v1alpha1.AdapterOverrides {
	return s.Spec.AdapterOverrides
}

// WantsOwnServiceAccount implements ServiceAccountProvider.
func (s *AWSS3Source) WantsOwnServiceAccount() bool {
	return s.Spec.Auth.WantsOwnServiceAccount()
}

// ServiceAccountOptions implements ServiceAccountProvider.
func (s *AWSS3Source) ServiceAccountOptions() []resource.ServiceAccountOption {
	return s.Spec.Auth.ServiceAccountOptions()
}

// Status conditions
const (
	// AWSS3ConditionSubscribed has status True when event notifications
	// have been successfully enabled on a S3 bucket.
	AWSS3ConditionSubscribed apis.ConditionType = "Subscribed"
)

// Reasons for status conditions
const (
	// AWSS3ReasonNoClient is set on a Subscribed condition when a S3/SQS API client cannot be obtained.
	AWSS3ReasonNoClient = "NoClient"
	// AWSS3ReasonNoBucket is set on a Subscribed condition when the S3 bucket does not exist.
	AWSS3ReasonNoBucket = "BucketNotFound"
	// AWSS3ReasonAPIError is set on a Subscribed condition when the S3/SQS API returns any other error.
	AWSS3ReasonAPIError = "APIError"
)

// awsS3SourceConditionSet is a set of conditions for AWSS3Source objects.
var awsS3SourceConditionSet = v1alpha1.NewConditionSet(
	AWSS3ConditionSubscribed,
)

// MarkSubscribed sets the Subscribed condition to True.
func (s *AWSS3SourceStatus) MarkSubscribed() {
	awsS3SourceConditionSet.Manage(s).MarkTrue(AWSS3ConditionSubscribed)
}

// MarkNotSubscribed sets the Subscribed condition to False with the given
// reason and associated message.
func (s *AWSS3SourceStatus) MarkNotSubscribed(reason, msg string) {
	awsS3SourceConditionSet.Manage(s).MarkFalse(AWSS3ConditionSubscribed,
		reason, msg)
}

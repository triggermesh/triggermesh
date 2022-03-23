/*
Copyright 2020 TriggerMesh Inc.

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
	"strings"

	"k8s.io/apimachinery/pkg/runtime/schema"

	"knative.dev/pkg/apis"
	duckv1 "knative.dev/pkg/apis/duck/v1"
)

// GetGroupVersionKind implements kmeta.OwnerRefable.
func (*AWSCloudWatchSource) GetGroupVersionKind() schema.GroupVersionKind {
	return SchemeGroupVersion.WithKind("AWSCloudWatchSource")
}

// GetConditionSet implements duckv1.KRShaped.
func (*AWSCloudWatchSource) GetConditionSet() apis.ConditionSet {
	return eventSourceConditionSet
}

// GetStatus implements duckv1.KRShaped.
func (s *AWSCloudWatchSource) GetStatus() *duckv1.Status {
	return &s.Status.Status
}

// GetSink implements Reconcilable.
func (s *AWSCloudWatchSource) GetSink() *duckv1.Destination {
	return &s.Spec.Sink
}

// GetStatusManager implements Reconcilable.
func (s *AWSCloudWatchSource) GetStatusManager() *StatusManager {
	return &StatusManager{
		ConditionSet:      s.GetConditionSet(),
		EventSourceStatus: &s.Status,
	}
}

// Supported event types
const (
	AWSCloudWatchMetricEventType  = "metrics.metric"
	AWSCloudWatchMessageEventType = "metrics.message"
)

// ServiceCloudWatch is the name of the CloudWatch service, as exposed in ARNs.
// https://docs.aws.amazon.com/service-authorization/latest/reference/list_amazoncloudwatch.html#amazoncloudwatch-resources-for-iam-policies
const ServiceCloudWatch = "cloudwatch"

// GetEventTypes implements Reconcilable.
func (*AWSCloudWatchSource) GetEventTypes() []string {
	return []string{
		AWSEventType(ServiceCloudWatch, AWSCloudWatchMetricEventType),
		AWSEventType(ServiceCloudWatch, AWSCloudWatchMessageEventType),
	}
}

// AsEventSource implements Reconcilable.
func (s *AWSCloudWatchSource) AsEventSource() string {
	return AWSCloudWatchSourceName(s.Namespace, s.Name)
}

// AWSCloudWatchSourceName returns a unique reference to the source suitable
// for use as as a CloudEvent source.
func AWSCloudWatchSourceName(ns, name string) string {
	kind := strings.ToLower((*AWSCloudWatchSource)(nil).GetGroupVersionKind().Kind)
	return "io.triggermesh." + kind + "." + ns + "." + name
}

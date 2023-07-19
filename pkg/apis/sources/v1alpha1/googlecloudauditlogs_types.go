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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	duckv1 "knative.dev/pkg/apis/duck/v1"

	"github.com/triggermesh/triggermesh/pkg/apis/common/v1alpha1"
)

// +genclient
// +genreconciler
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// GoogleCloudAuditLogsSource is the Schema for the event source.
type GoogleCloudAuditLogsSource struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   GoogleCloudAuditLogsSourceSpec   `json:"spec,omitempty"`
	Status GoogleCloudAuditLogsSourceStatus `json:"status,omitempty"`
}

// Check the interfaces the event source should be implementing.
var (
	_ v1alpha1.Reconcilable           = (*GoogleCloudAuditLogsSource)(nil)
	_ v1alpha1.AdapterConfigurable    = (*GoogleCloudAuditLogsSource)(nil)
	_ v1alpha1.EventSource            = (*GoogleCloudAuditLogsSource)(nil)
	_ v1alpha1.EventSender            = (*GoogleCloudAuditLogsSource)(nil)
	_ v1alpha1.ServiceAccountProvider = (*GoogleCloudAuditLogsSource)(nil)
)

// GoogleCloudAuditLogsSourceSpec defines the desired state of the event source.
type GoogleCloudAuditLogsSourceSpec struct {
	duckv1.SourceSpec `json:",inline"`

	// The CloudAuditLogsSource will pull events matching the following
	// parameters:
	// https://cloud.google.com/logging/docs/reference/audit/auditlog/rest/Shared.Types/AuditLog

	// The GCP service this instance should source audit logs from. Required.
	// example: compute.googleapis.com
	ServiceName string `json:"serviceName"`

	// The name of the service method or operation. For API calls,
	// this should be the name of the API method. Required.
	// beta.compute.instances.insert
	MethodName string `json:"methodName"`

	// The resource or collection that is the target of the
	// operation. The name is a scheme-less URI, not including the
	// API service name.
	// example: "projects/PROJECT_ID/zones/us-central1-a/instances"
	ResourceName *string `json:"resourceName,omitempty"`

	// Settings related to the Pub/Sub resources associated with the Audit Logs event sink.
	PubSub GoogleCloudSourcePubSubSpec `json:"pubsub"`

	// Different authentication methods available in sources on GCP.
	Auth v1alpha1.GoogleCloudAuth `json:"auth"`

	// Adapter spec overrides parameters.
	// +optional
	AdapterOverrides *v1alpha1.AdapterOverrides `json:"adapterOverrides,omitempty"`
}

// GoogleCloudAuditLogsSourceStatus defines the observed state of the event source.
type GoogleCloudAuditLogsSourceStatus struct {
	v1alpha1.Status `json:",inline"`

	// ID of the AuditLogSink used to publish audit log messages.
	AuditLogsSink *string `json:"auditLogsSink,omitempty"`

	// Resource name of the target Pub/Sub topic.
	Topic *GCloudResourceName `json:"topic,omitempty"`

	// Resource name of the managed Pub/Sub subscription associated with
	// the managed topic.
	Subscription *GCloudResourceName `json:"subscription,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// GoogleCloudAuditLogsSourceList contains a list of event sources.
type GoogleCloudAuditLogsSourceList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []GoogleCloudAuditLogsSource `json:"items"`
}

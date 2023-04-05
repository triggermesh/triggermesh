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
	"strings"

	"k8s.io/apimachinery/pkg/runtime/schema"

	"knative.dev/pkg/apis"
	duckv1 "knative.dev/pkg/apis/duck/v1"

	"github.com/triggermesh/triggermesh/pkg/apis/common/v1alpha1"
)

// GetGroupVersionKind implements kmeta.OwnerRefable.
func (*Function) GetGroupVersionKind() schema.GroupVersionKind {
	return SchemeGroupVersion.WithKind("Function")
}

// GetConditionSet implements duckv1.KRShaped.
func (f *Function) GetConditionSet() apis.ConditionSet {
	if f.Spec.Sink.Ref != nil || f.Spec.Sink.URI != nil {
		return funcSenderConditionSet
	}
	return functionConditionSet
}

// GetStatus implements duckv1.KRShaped.
func (f *Function) GetStatus() *duckv1.Status {
	return &f.Status.Status.Status
}

// GetStatusManager implements Reconcilable.
func (f *Function) GetStatusManager() *v1alpha1.StatusManager {
	return &v1alpha1.StatusManager{
		ConditionSet: f.GetConditionSet(),
		Status:       &f.Status.Status,
	}
}

// GetEventTypes implements EventSource.
func (f *Function) GetEventTypes() []string {
	if f.Spec.CloudEventOverrides == nil || len(f.Spec.CloudEventOverrides.Extensions) == 0 {
		return []string{defaultCEType(f)}
	}

	if typ := f.Spec.CloudEventOverrides.Extensions["type"]; typ != "" {
		return []string{typ}
	}

	return []string{defaultCEType(f)}
}

func defaultCEType(f *Function) string {
	const ceDefaultTypePrefix = "io.triggermesh.function."
	return ceDefaultTypePrefix + f.Spec.Runtime
}

// AsEventSource implements EventSource.
func (f *Function) AsEventSource() string {
	if f.Spec.CloudEventOverrides == nil || len(f.Spec.CloudEventOverrides.Extensions) == 0 {
		return defaultCESource(f)
	}

	if source := f.Spec.CloudEventOverrides.Extensions["source"]; source != "" {
		return source
	}

	return defaultCESource(f)
}

func defaultCESource(f *Function) string {
	kind := strings.ToLower(f.GetGroupVersionKind().Kind)
	return "io.triggermesh." + kind + "." + f.Namespace + "." + f.Name
}

// GetSink implements EventSender.
func (f *Function) GetSink() *duckv1.Destination {
	return &f.Spec.Sink
}

// GetAdapterOverrides implements AdapterConfigurable.
func (f *Function) GetAdapterOverrides() *v1alpha1.AdapterOverrides {
	return f.Spec.AdapterOverrides
}

// Status conditions
const (
	// FunctionConditionConfigMapReady has status True when the ConfigMap
	// containing the code of the Function was successfully reconciled.
	FunctionConditionConfigMapReady apis.ConditionType = "ConfigMapReady"
)

// Reasons for status conditions
const (
	// FunctionReasonFailedSync encompasses any type of error occuring while synchronizing a Kubernetes API object.
	// It is meant to be set on the ConfigMapReady condition.
	FunctionReasonFailedSync = "FailedSync"
)

// ConditionSets
var (
	// functionConditionSet is used when the component instance is
	// configured without a sink (reply mode).
	functionConditionSet = v1alpha1.NewConditionSet(
		FunctionConditionConfigMapReady,
	)

	// funcSenderConditionSet is used when the component instance is
	// configured with a sink (send mode).
	funcSenderConditionSet = v1alpha1.NewConditionSet(
		FunctionConditionConfigMapReady,
		v1alpha1.ConditionSinkProvided,
	)
)

// MarkConfigMapAvailable sets the ConfigMapReady condition to True and reports
// the name and resource version of the code ConfigMap.
func (s *FunctionStatus) MarkConfigMapAvailable(cmapName, resourceVersion string) {
	s.ConfigMap = &FunctionConfigMapIdentity{
		Name:            cmapName,
		ResourceVersion: resourceVersion,
	}

	functionConditionSet.Manage(s).MarkTrue(FunctionConditionConfigMapReady)
}

// MarkConfigMapUnavailable sets the ConfigMapReady condition to False with the
// given reason and associated message.
func (s *FunctionStatus) MarkConfigMapUnavailable(reason, msg string) {
	functionConditionSet.Manage(s).MarkFalse(FunctionConditionConfigMapReady,
		reason, msg)
}

// SetDefaults implements apis.Defaultable
func (f *Function) SetDefaults(ctx context.Context) {
}

// Validate implements apis.Validatable
func (f *Function) Validate(ctx context.Context) *apis.FieldError {
	return nil
}

//go:build !ignore_autogenerated
// +build !ignore_autogenerated

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

// Code generated by deepcopy-gen. DO NOT EDIT.

package v1alpha1

import (
	cloudevents "github.com/triggermesh/triggermesh/pkg/targets/adapter/cloudevents"
	v1 "k8s.io/api/core/v1"
	runtime "k8s.io/apimachinery/pkg/runtime"
)

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *CloudEventStatus) DeepCopyInto(out *CloudEventStatus) {
	*out = *in
	if in.AcceptedEventTypes != nil {
		in, out := &in.AcceptedEventTypes, &out.AcceptedEventTypes
		*out = make([]string, len(*in))
		copy(*out, *in)
	}
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new CloudEventStatus.
func (in *CloudEventStatus) DeepCopy() *CloudEventStatus {
	if in == nil {
		return nil
	}
	out := new(CloudEventStatus)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *Correlation) DeepCopyInto(out *Correlation) {
	*out = *in
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new Correlation.
func (in *Correlation) DeepCopy() *Correlation {
	if in == nil {
		return nil
	}
	out := new(Correlation)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *DataWeaveTransformation) DeepCopyInto(out *DataWeaveTransformation) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ObjectMeta.DeepCopyInto(&out.ObjectMeta)
	in.Spec.DeepCopyInto(&out.Spec)
	in.Status.DeepCopyInto(&out.Status)
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new DataWeaveTransformation.
func (in *DataWeaveTransformation) DeepCopy() *DataWeaveTransformation {
	if in == nil {
		return nil
	}
	out := new(DataWeaveTransformation)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *DataWeaveTransformation) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *DataWeaveTransformationList) DeepCopyInto(out *DataWeaveTransformationList) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ListMeta.DeepCopyInto(&out.ListMeta)
	if in.Items != nil {
		in, out := &in.Items, &out.Items
		*out = make([]DataWeaveTransformation, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new DataWeaveTransformationList.
func (in *DataWeaveTransformationList) DeepCopy() *DataWeaveTransformationList {
	if in == nil {
		return nil
	}
	out := new(DataWeaveTransformationList)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *DataWeaveTransformationList) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *DataWeaveTransformationSpec) DeepCopyInto(out *DataWeaveTransformationSpec) {
	*out = *in
	in.DwSpell.DeepCopyInto(&out.DwSpell)
	in.SourceSpec.DeepCopyInto(&out.SourceSpec)
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new DataWeaveTransformationSpec.
func (in *DataWeaveTransformationSpec) DeepCopy() *DataWeaveTransformationSpec {
	if in == nil {
		return nil
	}
	out := new(DataWeaveTransformationSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *EventOptions) DeepCopyInto(out *EventOptions) {
	*out = *in
	if in.PayloadPolicy != nil {
		in, out := &in.PayloadPolicy, &out.PayloadPolicy
		*out = new(cloudevents.PayloadPolicy)
		**out = **in
	}
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new EventOptions.
func (in *EventOptions) DeepCopy() *EventOptions {
	if in == nil {
		return nil
	}
	out := new(EventOptions)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *JQTransformation) DeepCopyInto(out *JQTransformation) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ObjectMeta.DeepCopyInto(&out.ObjectMeta)
	in.Spec.DeepCopyInto(&out.Spec)
	in.Status.DeepCopyInto(&out.Status)
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new JQTransformation.
func (in *JQTransformation) DeepCopy() *JQTransformation {
	if in == nil {
		return nil
	}
	out := new(JQTransformation)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *JQTransformation) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *JQTransformationList) DeepCopyInto(out *JQTransformationList) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ListMeta.DeepCopyInto(&out.ListMeta)
	if in.Items != nil {
		in, out := &in.Items, &out.Items
		*out = make([]JQTransformation, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new JQTransformationList.
func (in *JQTransformationList) DeepCopy() *JQTransformationList {
	if in == nil {
		return nil
	}
	out := new(JQTransformationList)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *JQTransformationList) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *JQTransformationSpec) DeepCopyInto(out *JQTransformationSpec) {
	*out = *in
	if in.EventOptions != nil {
		in, out := &in.EventOptions, &out.EventOptions
		*out = new(EventOptions)
		(*in).DeepCopyInto(*out)
	}
	in.SourceSpec.DeepCopyInto(&out.SourceSpec)
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new JQTransformationSpec.
func (in *JQTransformationSpec) DeepCopy() *JQTransformationSpec {
	if in == nil {
		return nil
	}
	out := new(JQTransformationSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *JQTransformationStatus) DeepCopyInto(out *JQTransformationStatus) {
	*out = *in
	in.SourceStatus.DeepCopyInto(&out.SourceStatus)
	in.AddressStatus.DeepCopyInto(&out.AddressStatus)
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new JQTransformationStatus.
func (in *JQTransformationStatus) DeepCopy() *JQTransformationStatus {
	if in == nil {
		return nil
	}
	out := new(JQTransformationStatus)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *Path) DeepCopyInto(out *Path) {
	*out = *in
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new Path.
func (in *Path) DeepCopy() *Path {
	if in == nil {
		return nil
	}
	out := new(Path)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *Response) DeepCopyInto(out *Response) {
	*out = *in
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new Response.
func (in *Response) DeepCopy() *Response {
	if in == nil {
		return nil
	}
	out := new(Response)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *Synchronizer) DeepCopyInto(out *Synchronizer) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ObjectMeta.DeepCopyInto(&out.ObjectMeta)
	in.Spec.DeepCopyInto(&out.Spec)
	in.Status.DeepCopyInto(&out.Status)
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new Synchronizer.
func (in *Synchronizer) DeepCopy() *Synchronizer {
	if in == nil {
		return nil
	}
	out := new(Synchronizer)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *Synchronizer) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *SynchronizerList) DeepCopyInto(out *SynchronizerList) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ListMeta.DeepCopyInto(&out.ListMeta)
	if in.Items != nil {
		in, out := &in.Items, &out.Items
		*out = make([]Synchronizer, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new SynchronizerList.
func (in *SynchronizerList) DeepCopy() *SynchronizerList {
	if in == nil {
		return nil
	}
	out := new(SynchronizerList)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *SynchronizerList) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *SynchronizerSpec) DeepCopyInto(out *SynchronizerSpec) {
	*out = *in
	out.CorrelationKey = in.CorrelationKey
	out.Response = in.Response
	in.SourceSpec.DeepCopyInto(&out.SourceSpec)
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new SynchronizerSpec.
func (in *SynchronizerSpec) DeepCopy() *SynchronizerSpec {
	if in == nil {
		return nil
	}
	out := new(SynchronizerSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *TargetStatus) DeepCopyInto(out *TargetStatus) {
	*out = *in
	in.SourceStatus.DeepCopyInto(&out.SourceStatus)
	in.AddressStatus.DeepCopyInto(&out.AddressStatus)
	in.CloudEventStatus.DeepCopyInto(&out.CloudEventStatus)
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new TargetStatus.
func (in *TargetStatus) DeepCopy() *TargetStatus {
	if in == nil {
		return nil
	}
	out := new(TargetStatus)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *Transform) DeepCopyInto(out *Transform) {
	*out = *in
	if in.Paths != nil {
		in, out := &in.Paths, &out.Paths
		*out = make([]Path, len(*in))
		copy(*out, *in)
	}
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new Transform.
func (in *Transform) DeepCopy() *Transform {
	if in == nil {
		return nil
	}
	out := new(Transform)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *Transformation) DeepCopyInto(out *Transformation) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ObjectMeta.DeepCopyInto(&out.ObjectMeta)
	in.Spec.DeepCopyInto(&out.Spec)
	in.Status.DeepCopyInto(&out.Status)
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new Transformation.
func (in *Transformation) DeepCopy() *Transformation {
	if in == nil {
		return nil
	}
	out := new(Transformation)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *Transformation) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *TransformationList) DeepCopyInto(out *TransformationList) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ListMeta.DeepCopyInto(&out.ListMeta)
	if in.Items != nil {
		in, out := &in.Items, &out.Items
		*out = make([]Transformation, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new TransformationList.
func (in *TransformationList) DeepCopy() *TransformationList {
	if in == nil {
		return nil
	}
	out := new(TransformationList)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *TransformationList) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *TransformationSpec) DeepCopyInto(out *TransformationSpec) {
	*out = *in
	if in.Context != nil {
		in, out := &in.Context, &out.Context
		*out = make([]Transform, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
	if in.Data != nil {
		in, out := &in.Data, &out.Data
		*out = make([]Transform, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
	in.SourceSpec.DeepCopyInto(&out.SourceSpec)
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new TransformationSpec.
func (in *TransformationSpec) DeepCopy() *TransformationSpec {
	if in == nil {
		return nil
	}
	out := new(TransformationSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ValueFromField) DeepCopyInto(out *ValueFromField) {
	*out = *in
	if in.ValueFromSecret != nil {
		in, out := &in.ValueFromSecret, &out.ValueFromSecret
		*out = new(v1.SecretKeySelector)
		(*in).DeepCopyInto(*out)
	}
	if in.ValueFromConfigMap != nil {
		in, out := &in.ValueFromConfigMap, &out.ValueFromConfigMap
		*out = new(v1.ConfigMapKeySelector)
		(*in).DeepCopyInto(*out)
	}
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ValueFromField.
func (in *ValueFromField) DeepCopy() *ValueFromField {
	if in == nil {
		return nil
	}
	out := new(ValueFromField)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *XMLToJSONTransformation) DeepCopyInto(out *XMLToJSONTransformation) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ObjectMeta.DeepCopyInto(&out.ObjectMeta)
	in.Spec.DeepCopyInto(&out.Spec)
	in.Status.DeepCopyInto(&out.Status)
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new XMLToJSONTransformation.
func (in *XMLToJSONTransformation) DeepCopy() *XMLToJSONTransformation {
	if in == nil {
		return nil
	}
	out := new(XMLToJSONTransformation)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *XMLToJSONTransformation) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *XMLToJSONTransformationList) DeepCopyInto(out *XMLToJSONTransformationList) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ListMeta.DeepCopyInto(&out.ListMeta)
	if in.Items != nil {
		in, out := &in.Items, &out.Items
		*out = make([]XMLToJSONTransformation, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new XMLToJSONTransformationList.
func (in *XMLToJSONTransformationList) DeepCopy() *XMLToJSONTransformationList {
	if in == nil {
		return nil
	}
	out := new(XMLToJSONTransformationList)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *XMLToJSONTransformationList) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *XMLToJSONTransformationSpec) DeepCopyInto(out *XMLToJSONTransformationSpec) {
	*out = *in
	if in.EventOptions != nil {
		in, out := &in.EventOptions, &out.EventOptions
		*out = new(EventOptions)
		(*in).DeepCopyInto(*out)
	}
	in.SourceSpec.DeepCopyInto(&out.SourceSpec)
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new XMLToJSONTransformationSpec.
func (in *XMLToJSONTransformationSpec) DeepCopy() *XMLToJSONTransformationSpec {
	if in == nil {
		return nil
	}
	out := new(XMLToJSONTransformationSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *XSLTTransformation) DeepCopyInto(out *XSLTTransformation) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ObjectMeta.DeepCopyInto(&out.ObjectMeta)
	in.Spec.DeepCopyInto(&out.Spec)
	in.Status.DeepCopyInto(&out.Status)
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new XSLTTransformation.
func (in *XSLTTransformation) DeepCopy() *XSLTTransformation {
	if in == nil {
		return nil
	}
	out := new(XSLTTransformation)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *XSLTTransformation) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *XSLTTransformationList) DeepCopyInto(out *XSLTTransformationList) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ListMeta.DeepCopyInto(&out.ListMeta)
	if in.Items != nil {
		in, out := &in.Items, &out.Items
		*out = make([]XSLTTransformation, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new XSLTTransformationList.
func (in *XSLTTransformationList) DeepCopy() *XSLTTransformationList {
	if in == nil {
		return nil
	}
	out := new(XSLTTransformationList)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *XSLTTransformationList) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *XSLTTransformationSpec) DeepCopyInto(out *XSLTTransformationSpec) {
	*out = *in
	if in.XSLT != nil {
		in, out := &in.XSLT, &out.XSLT
		*out = new(ValueFromField)
		(*in).DeepCopyInto(*out)
	}
	if in.AllowPerEventXSLT != nil {
		in, out := &in.AllowPerEventXSLT, &out.AllowPerEventXSLT
		*out = new(bool)
		**out = **in
	}
	in.SourceSpec.DeepCopyInto(&out.SourceSpec)
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new XSLTTransformationSpec.
func (in *XSLTTransformationSpec) DeepCopy() *XSLTTransformationSpec {
	if in == nil {
		return nil
	}
	out := new(XSLTTransformationSpec)
	in.DeepCopyInto(out)
	return out
}

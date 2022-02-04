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

// Code generated by informer-gen. DO NOT EDIT.

package v1alpha1

import (
	internalinterfaces "github.com/triggermesh/triggermesh/pkg/client/generated/informers/externalversions/internalinterfaces"
)

// Interface provides access to all the informers in this group version.
type Interface interface {
	// Synchronizers returns a SynchronizerInformer.
	Synchronizers() SynchronizerInformer
	// Transformations returns a TransformationInformer.
	Transformations() TransformationInformer
	// XMLToJSONTransformations returns a XMLToJSONTransformationInformer.
	XMLToJSONTransformations() XMLToJSONTransformationInformer
	// XSLTTransformations returns a XSLTTransformationInformer.
	XSLTTransformations() XSLTTransformationInformer
}

type version struct {
	factory          internalinterfaces.SharedInformerFactory
	namespace        string
	tweakListOptions internalinterfaces.TweakListOptionsFunc
}

// New returns a new Interface.
func New(f internalinterfaces.SharedInformerFactory, namespace string, tweakListOptions internalinterfaces.TweakListOptionsFunc) Interface {
	return &version{factory: f, namespace: namespace, tweakListOptions: tweakListOptions}
}

// Synchronizers returns a SynchronizerInformer.
func (v *version) Synchronizers() SynchronizerInformer {
	return &synchronizerInformer{factory: v.factory, namespace: v.namespace, tweakListOptions: v.tweakListOptions}
}

// Transformations returns a TransformationInformer.
func (v *version) Transformations() TransformationInformer {
	return &transformationInformer{factory: v.factory, namespace: v.namespace, tweakListOptions: v.tweakListOptions}
}

// XMLToJSONTransformations returns a XMLToJSONTransformationInformer.
func (v *version) XMLToJSONTransformations() XMLToJSONTransformationInformer {
	return &xMLToJSONTransformationInformer{factory: v.factory, namespace: v.namespace, tweakListOptions: v.tweakListOptions}
}

// XSLTTransformations returns a XSLTTransformationInformer.
func (v *version) XSLTTransformations() XSLTTransformationInformer {
	return &xSLTTransformInformer{factory: v.factory, namespace: v.namespace, tweakListOptions: v.tweakListOptions}
}

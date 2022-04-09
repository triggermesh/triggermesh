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

// Code generated by client-gen. DO NOT EDIT.

package v1alpha1

import (
	"context"
	"time"

	v1alpha1 "github.com/triggermesh/triggermesh/pkg/apis/targets/v1alpha1"
	scheme "github.com/triggermesh/triggermesh/pkg/client/generated/clientset/internalclientset/scheme"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	types "k8s.io/apimachinery/pkg/types"
	watch "k8s.io/apimachinery/pkg/watch"
	rest "k8s.io/client-go/rest"
)

// HasuraTargetsGetter has a method to return a HasuraTargetInterface.
// A group's client should implement this interface.
type HasuraTargetsGetter interface {
	HasuraTargets(namespace string) HasuraTargetInterface
}

// HasuraTargetInterface has methods to work with HasuraTarget resources.
type HasuraTargetInterface interface {
	Create(ctx context.Context, hasuraTarget *v1alpha1.HasuraTarget, opts v1.CreateOptions) (*v1alpha1.HasuraTarget, error)
	Update(ctx context.Context, hasuraTarget *v1alpha1.HasuraTarget, opts v1.UpdateOptions) (*v1alpha1.HasuraTarget, error)
	UpdateStatus(ctx context.Context, hasuraTarget *v1alpha1.HasuraTarget, opts v1.UpdateOptions) (*v1alpha1.HasuraTarget, error)
	Delete(ctx context.Context, name string, opts v1.DeleteOptions) error
	DeleteCollection(ctx context.Context, opts v1.DeleteOptions, listOpts v1.ListOptions) error
	Get(ctx context.Context, name string, opts v1.GetOptions) (*v1alpha1.HasuraTarget, error)
	List(ctx context.Context, opts v1.ListOptions) (*v1alpha1.HasuraTargetList, error)
	Watch(ctx context.Context, opts v1.ListOptions) (watch.Interface, error)
	Patch(ctx context.Context, name string, pt types.PatchType, data []byte, opts v1.PatchOptions, subresources ...string) (result *v1alpha1.HasuraTarget, err error)
	HasuraTargetExpansion
}

// hasuraTargets implements HasuraTargetInterface
type hasuraTargets struct {
	client rest.Interface
	ns     string
}

// newHasuraTargets returns a HasuraTargets
func newHasuraTargets(c *TargetsV1alpha1Client, namespace string) *hasuraTargets {
	return &hasuraTargets{
		client: c.RESTClient(),
		ns:     namespace,
	}
}

// Get takes name of the hasuraTarget, and returns the corresponding hasuraTarget object, and an error if there is any.
func (c *hasuraTargets) Get(ctx context.Context, name string, options v1.GetOptions) (result *v1alpha1.HasuraTarget, err error) {
	result = &v1alpha1.HasuraTarget{}
	err = c.client.Get().
		Namespace(c.ns).
		Resource("hasuratargets").
		Name(name).
		VersionedParams(&options, scheme.ParameterCodec).
		Do(ctx).
		Into(result)
	return
}

// List takes label and field selectors, and returns the list of HasuraTargets that match those selectors.
func (c *hasuraTargets) List(ctx context.Context, opts v1.ListOptions) (result *v1alpha1.HasuraTargetList, err error) {
	var timeout time.Duration
	if opts.TimeoutSeconds != nil {
		timeout = time.Duration(*opts.TimeoutSeconds) * time.Second
	}
	result = &v1alpha1.HasuraTargetList{}
	err = c.client.Get().
		Namespace(c.ns).
		Resource("hasuratargets").
		VersionedParams(&opts, scheme.ParameterCodec).
		Timeout(timeout).
		Do(ctx).
		Into(result)
	return
}

// Watch returns a watch.Interface that watches the requested hasuraTargets.
func (c *hasuraTargets) Watch(ctx context.Context, opts v1.ListOptions) (watch.Interface, error) {
	var timeout time.Duration
	if opts.TimeoutSeconds != nil {
		timeout = time.Duration(*opts.TimeoutSeconds) * time.Second
	}
	opts.Watch = true
	return c.client.Get().
		Namespace(c.ns).
		Resource("hasuratargets").
		VersionedParams(&opts, scheme.ParameterCodec).
		Timeout(timeout).
		Watch(ctx)
}

// Create takes the representation of a hasuraTarget and creates it.  Returns the server's representation of the hasuraTarget, and an error, if there is any.
func (c *hasuraTargets) Create(ctx context.Context, hasuraTarget *v1alpha1.HasuraTarget, opts v1.CreateOptions) (result *v1alpha1.HasuraTarget, err error) {
	result = &v1alpha1.HasuraTarget{}
	err = c.client.Post().
		Namespace(c.ns).
		Resource("hasuratargets").
		VersionedParams(&opts, scheme.ParameterCodec).
		Body(hasuraTarget).
		Do(ctx).
		Into(result)
	return
}

// Update takes the representation of a hasuraTarget and updates it. Returns the server's representation of the hasuraTarget, and an error, if there is any.
func (c *hasuraTargets) Update(ctx context.Context, hasuraTarget *v1alpha1.HasuraTarget, opts v1.UpdateOptions) (result *v1alpha1.HasuraTarget, err error) {
	result = &v1alpha1.HasuraTarget{}
	err = c.client.Put().
		Namespace(c.ns).
		Resource("hasuratargets").
		Name(hasuraTarget.Name).
		VersionedParams(&opts, scheme.ParameterCodec).
		Body(hasuraTarget).
		Do(ctx).
		Into(result)
	return
}

// UpdateStatus was generated because the type contains a Status member.
// Add a +genclient:noStatus comment above the type to avoid generating UpdateStatus().
func (c *hasuraTargets) UpdateStatus(ctx context.Context, hasuraTarget *v1alpha1.HasuraTarget, opts v1.UpdateOptions) (result *v1alpha1.HasuraTarget, err error) {
	result = &v1alpha1.HasuraTarget{}
	err = c.client.Put().
		Namespace(c.ns).
		Resource("hasuratargets").
		Name(hasuraTarget.Name).
		SubResource("status").
		VersionedParams(&opts, scheme.ParameterCodec).
		Body(hasuraTarget).
		Do(ctx).
		Into(result)
	return
}

// Delete takes name of the hasuraTarget and deletes it. Returns an error if one occurs.
func (c *hasuraTargets) Delete(ctx context.Context, name string, opts v1.DeleteOptions) error {
	return c.client.Delete().
		Namespace(c.ns).
		Resource("hasuratargets").
		Name(name).
		Body(&opts).
		Do(ctx).
		Error()
}

// DeleteCollection deletes a collection of objects.
func (c *hasuraTargets) DeleteCollection(ctx context.Context, opts v1.DeleteOptions, listOpts v1.ListOptions) error {
	var timeout time.Duration
	if listOpts.TimeoutSeconds != nil {
		timeout = time.Duration(*listOpts.TimeoutSeconds) * time.Second
	}
	return c.client.Delete().
		Namespace(c.ns).
		Resource("hasuratargets").
		VersionedParams(&listOpts, scheme.ParameterCodec).
		Timeout(timeout).
		Body(&opts).
		Do(ctx).
		Error()
}

// Patch applies the patch and returns the patched hasuraTarget.
func (c *hasuraTargets) Patch(ctx context.Context, name string, pt types.PatchType, data []byte, opts v1.PatchOptions, subresources ...string) (result *v1alpha1.HasuraTarget, err error) {
	result = &v1alpha1.HasuraTarget{}
	err = c.client.Patch(pt).
		Namespace(c.ns).
		Resource("hasuratargets").
		Name(name).
		SubResource(subresources...).
		VersionedParams(&opts, scheme.ParameterCodec).
		Body(data).
		Do(ctx).
		Into(result)
	return
}

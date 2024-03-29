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

// MongoDBTargetsGetter has a method to return a MongoDBTargetInterface.
// A group's client should implement this interface.
type MongoDBTargetsGetter interface {
	MongoDBTargets(namespace string) MongoDBTargetInterface
}

// MongoDBTargetInterface has methods to work with MongoDBTarget resources.
type MongoDBTargetInterface interface {
	Create(ctx context.Context, mongoDBTarget *v1alpha1.MongoDBTarget, opts v1.CreateOptions) (*v1alpha1.MongoDBTarget, error)
	Update(ctx context.Context, mongoDBTarget *v1alpha1.MongoDBTarget, opts v1.UpdateOptions) (*v1alpha1.MongoDBTarget, error)
	UpdateStatus(ctx context.Context, mongoDBTarget *v1alpha1.MongoDBTarget, opts v1.UpdateOptions) (*v1alpha1.MongoDBTarget, error)
	Delete(ctx context.Context, name string, opts v1.DeleteOptions) error
	DeleteCollection(ctx context.Context, opts v1.DeleteOptions, listOpts v1.ListOptions) error
	Get(ctx context.Context, name string, opts v1.GetOptions) (*v1alpha1.MongoDBTarget, error)
	List(ctx context.Context, opts v1.ListOptions) (*v1alpha1.MongoDBTargetList, error)
	Watch(ctx context.Context, opts v1.ListOptions) (watch.Interface, error)
	Patch(ctx context.Context, name string, pt types.PatchType, data []byte, opts v1.PatchOptions, subresources ...string) (result *v1alpha1.MongoDBTarget, err error)
	MongoDBTargetExpansion
}

// mongoDBTargets implements MongoDBTargetInterface
type mongoDBTargets struct {
	client rest.Interface
	ns     string
}

// newMongoDBTargets returns a MongoDBTargets
func newMongoDBTargets(c *TargetsV1alpha1Client, namespace string) *mongoDBTargets {
	return &mongoDBTargets{
		client: c.RESTClient(),
		ns:     namespace,
	}
}

// Get takes name of the mongoDBTarget, and returns the corresponding mongoDBTarget object, and an error if there is any.
func (c *mongoDBTargets) Get(ctx context.Context, name string, options v1.GetOptions) (result *v1alpha1.MongoDBTarget, err error) {
	result = &v1alpha1.MongoDBTarget{}
	err = c.client.Get().
		Namespace(c.ns).
		Resource("mongodbtargets").
		Name(name).
		VersionedParams(&options, scheme.ParameterCodec).
		Do(ctx).
		Into(result)
	return
}

// List takes label and field selectors, and returns the list of MongoDBTargets that match those selectors.
func (c *mongoDBTargets) List(ctx context.Context, opts v1.ListOptions) (result *v1alpha1.MongoDBTargetList, err error) {
	var timeout time.Duration
	if opts.TimeoutSeconds != nil {
		timeout = time.Duration(*opts.TimeoutSeconds) * time.Second
	}
	result = &v1alpha1.MongoDBTargetList{}
	err = c.client.Get().
		Namespace(c.ns).
		Resource("mongodbtargets").
		VersionedParams(&opts, scheme.ParameterCodec).
		Timeout(timeout).
		Do(ctx).
		Into(result)
	return
}

// Watch returns a watch.Interface that watches the requested mongoDBTargets.
func (c *mongoDBTargets) Watch(ctx context.Context, opts v1.ListOptions) (watch.Interface, error) {
	var timeout time.Duration
	if opts.TimeoutSeconds != nil {
		timeout = time.Duration(*opts.TimeoutSeconds) * time.Second
	}
	opts.Watch = true
	return c.client.Get().
		Namespace(c.ns).
		Resource("mongodbtargets").
		VersionedParams(&opts, scheme.ParameterCodec).
		Timeout(timeout).
		Watch(ctx)
}

// Create takes the representation of a mongoDBTarget and creates it.  Returns the server's representation of the mongoDBTarget, and an error, if there is any.
func (c *mongoDBTargets) Create(ctx context.Context, mongoDBTarget *v1alpha1.MongoDBTarget, opts v1.CreateOptions) (result *v1alpha1.MongoDBTarget, err error) {
	result = &v1alpha1.MongoDBTarget{}
	err = c.client.Post().
		Namespace(c.ns).
		Resource("mongodbtargets").
		VersionedParams(&opts, scheme.ParameterCodec).
		Body(mongoDBTarget).
		Do(ctx).
		Into(result)
	return
}

// Update takes the representation of a mongoDBTarget and updates it. Returns the server's representation of the mongoDBTarget, and an error, if there is any.
func (c *mongoDBTargets) Update(ctx context.Context, mongoDBTarget *v1alpha1.MongoDBTarget, opts v1.UpdateOptions) (result *v1alpha1.MongoDBTarget, err error) {
	result = &v1alpha1.MongoDBTarget{}
	err = c.client.Put().
		Namespace(c.ns).
		Resource("mongodbtargets").
		Name(mongoDBTarget.Name).
		VersionedParams(&opts, scheme.ParameterCodec).
		Body(mongoDBTarget).
		Do(ctx).
		Into(result)
	return
}

// UpdateStatus was generated because the type contains a Status member.
// Add a +genclient:noStatus comment above the type to avoid generating UpdateStatus().
func (c *mongoDBTargets) UpdateStatus(ctx context.Context, mongoDBTarget *v1alpha1.MongoDBTarget, opts v1.UpdateOptions) (result *v1alpha1.MongoDBTarget, err error) {
	result = &v1alpha1.MongoDBTarget{}
	err = c.client.Put().
		Namespace(c.ns).
		Resource("mongodbtargets").
		Name(mongoDBTarget.Name).
		SubResource("status").
		VersionedParams(&opts, scheme.ParameterCodec).
		Body(mongoDBTarget).
		Do(ctx).
		Into(result)
	return
}

// Delete takes name of the mongoDBTarget and deletes it. Returns an error if one occurs.
func (c *mongoDBTargets) Delete(ctx context.Context, name string, opts v1.DeleteOptions) error {
	return c.client.Delete().
		Namespace(c.ns).
		Resource("mongodbtargets").
		Name(name).
		Body(&opts).
		Do(ctx).
		Error()
}

// DeleteCollection deletes a collection of objects.
func (c *mongoDBTargets) DeleteCollection(ctx context.Context, opts v1.DeleteOptions, listOpts v1.ListOptions) error {
	var timeout time.Duration
	if listOpts.TimeoutSeconds != nil {
		timeout = time.Duration(*listOpts.TimeoutSeconds) * time.Second
	}
	return c.client.Delete().
		Namespace(c.ns).
		Resource("mongodbtargets").
		VersionedParams(&listOpts, scheme.ParameterCodec).
		Timeout(timeout).
		Body(&opts).
		Do(ctx).
		Error()
}

// Patch applies the patch and returns the patched mongoDBTarget.
func (c *mongoDBTargets) Patch(ctx context.Context, name string, pt types.PatchType, data []byte, opts v1.PatchOptions, subresources ...string) (result *v1alpha1.MongoDBTarget, err error) {
	result = &v1alpha1.MongoDBTarget{}
	err = c.client.Patch(pt).
		Namespace(c.ns).
		Resource("mongodbtargets").
		Name(name).
		SubResource(subresources...).
		VersionedParams(&opts, scheme.ParameterCodec).
		Body(data).
		Do(ctx).
		Into(result)
	return
}

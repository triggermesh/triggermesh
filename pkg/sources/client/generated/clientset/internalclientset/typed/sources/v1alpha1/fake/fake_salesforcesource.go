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

// Code generated by client-gen. DO NOT EDIT.

package fake

import (
	"context"

	v1alpha1 "github.com/triggermesh/triggermesh/pkg/apis/sources/v1alpha1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	labels "k8s.io/apimachinery/pkg/labels"
	schema "k8s.io/apimachinery/pkg/runtime/schema"
	types "k8s.io/apimachinery/pkg/types"
	watch "k8s.io/apimachinery/pkg/watch"
	testing "k8s.io/client-go/testing"
)

// FakeSalesforceSources implements SalesforceSourceInterface
type FakeSalesforceSources struct {
	Fake *FakeSourcesV1alpha1
	ns   string
}

var salesforcesourcesResource = schema.GroupVersionResource{Group: "sources.triggermesh.io", Version: "v1alpha1", Resource: "salesforcesources"}

var salesforcesourcesKind = schema.GroupVersionKind{Group: "sources.triggermesh.io", Version: "v1alpha1", Kind: "SalesforceSource"}

// Get takes name of the salesforceSource, and returns the corresponding salesforceSource object, and an error if there is any.
func (c *FakeSalesforceSources) Get(ctx context.Context, name string, options v1.GetOptions) (result *v1alpha1.SalesforceSource, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewGetAction(salesforcesourcesResource, c.ns, name), &v1alpha1.SalesforceSource{})

	if obj == nil {
		return nil, err
	}
	return obj.(*v1alpha1.SalesforceSource), err
}

// List takes label and field selectors, and returns the list of SalesforceSources that match those selectors.
func (c *FakeSalesforceSources) List(ctx context.Context, opts v1.ListOptions) (result *v1alpha1.SalesforceSourceList, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewListAction(salesforcesourcesResource, salesforcesourcesKind, c.ns, opts), &v1alpha1.SalesforceSourceList{})

	if obj == nil {
		return nil, err
	}

	label, _, _ := testing.ExtractFromListOptions(opts)
	if label == nil {
		label = labels.Everything()
	}
	list := &v1alpha1.SalesforceSourceList{ListMeta: obj.(*v1alpha1.SalesforceSourceList).ListMeta}
	for _, item := range obj.(*v1alpha1.SalesforceSourceList).Items {
		if label.Matches(labels.Set(item.Labels)) {
			list.Items = append(list.Items, item)
		}
	}
	return list, err
}

// Watch returns a watch.Interface that watches the requested salesforceSources.
func (c *FakeSalesforceSources) Watch(ctx context.Context, opts v1.ListOptions) (watch.Interface, error) {
	return c.Fake.
		InvokesWatch(testing.NewWatchAction(salesforcesourcesResource, c.ns, opts))

}

// Create takes the representation of a salesforceSource and creates it.  Returns the server's representation of the salesforceSource, and an error, if there is any.
func (c *FakeSalesforceSources) Create(ctx context.Context, salesforceSource *v1alpha1.SalesforceSource, opts v1.CreateOptions) (result *v1alpha1.SalesforceSource, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewCreateAction(salesforcesourcesResource, c.ns, salesforceSource), &v1alpha1.SalesforceSource{})

	if obj == nil {
		return nil, err
	}
	return obj.(*v1alpha1.SalesforceSource), err
}

// Update takes the representation of a salesforceSource and updates it. Returns the server's representation of the salesforceSource, and an error, if there is any.
func (c *FakeSalesforceSources) Update(ctx context.Context, salesforceSource *v1alpha1.SalesforceSource, opts v1.UpdateOptions) (result *v1alpha1.SalesforceSource, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewUpdateAction(salesforcesourcesResource, c.ns, salesforceSource), &v1alpha1.SalesforceSource{})

	if obj == nil {
		return nil, err
	}
	return obj.(*v1alpha1.SalesforceSource), err
}

// UpdateStatus was generated because the type contains a Status member.
// Add a +genclient:noStatus comment above the type to avoid generating UpdateStatus().
func (c *FakeSalesforceSources) UpdateStatus(ctx context.Context, salesforceSource *v1alpha1.SalesforceSource, opts v1.UpdateOptions) (*v1alpha1.SalesforceSource, error) {
	obj, err := c.Fake.
		Invokes(testing.NewUpdateSubresourceAction(salesforcesourcesResource, "status", c.ns, salesforceSource), &v1alpha1.SalesforceSource{})

	if obj == nil {
		return nil, err
	}
	return obj.(*v1alpha1.SalesforceSource), err
}

// Delete takes name of the salesforceSource and deletes it. Returns an error if one occurs.
func (c *FakeSalesforceSources) Delete(ctx context.Context, name string, opts v1.DeleteOptions) error {
	_, err := c.Fake.
		Invokes(testing.NewDeleteAction(salesforcesourcesResource, c.ns, name), &v1alpha1.SalesforceSource{})

	return err
}

// DeleteCollection deletes a collection of objects.
func (c *FakeSalesforceSources) DeleteCollection(ctx context.Context, opts v1.DeleteOptions, listOpts v1.ListOptions) error {
	action := testing.NewDeleteCollectionAction(salesforcesourcesResource, c.ns, listOpts)

	_, err := c.Fake.Invokes(action, &v1alpha1.SalesforceSourceList{})
	return err
}

// Patch applies the patch and returns the patched salesforceSource.
func (c *FakeSalesforceSources) Patch(ctx context.Context, name string, pt types.PatchType, data []byte, opts v1.PatchOptions, subresources ...string) (result *v1alpha1.SalesforceSource, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewPatchSubresourceAction(salesforcesourcesResource, c.ns, name, pt, data, subresources...), &v1alpha1.SalesforceSource{})

	if obj == nil {
		return nil, err
	}
	return obj.(*v1alpha1.SalesforceSource), err
}

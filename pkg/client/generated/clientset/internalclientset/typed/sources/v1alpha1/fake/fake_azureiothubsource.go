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

// FakeAzureIOTHubSources implements AzureIOTHubSourceInterface
type FakeAzureIOTHubSources struct {
	Fake *FakeSourcesV1alpha1
	ns   string
}

var azureiothubsourcesResource = schema.GroupVersionResource{Group: "sources.triggermesh.io", Version: "v1alpha1", Resource: "azureiothubsources"}

var azureiothubsourcesKind = schema.GroupVersionKind{Group: "sources.triggermesh.io", Version: "v1alpha1", Kind: "AzureIOTHubSource"}

// Get takes name of the azureIOTHubSource, and returns the corresponding azureIOTHubSource object, and an error if there is any.
func (c *FakeAzureIOTHubSources) Get(ctx context.Context, name string, options v1.GetOptions) (result *v1alpha1.AzureIOTHubSource, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewGetAction(azureiothubsourcesResource, c.ns, name), &v1alpha1.AzureIOTHubSource{})

	if obj == nil {
		return nil, err
	}
	return obj.(*v1alpha1.AzureIOTHubSource), err
}

// List takes label and field selectors, and returns the list of AzureIOTHubSources that match those selectors.
func (c *FakeAzureIOTHubSources) List(ctx context.Context, opts v1.ListOptions) (result *v1alpha1.AzureIOTHubSourceList, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewListAction(azureiothubsourcesResource, azureiothubsourcesKind, c.ns, opts), &v1alpha1.AzureIOTHubSourceList{})

	if obj == nil {
		return nil, err
	}

	label, _, _ := testing.ExtractFromListOptions(opts)
	if label == nil {
		label = labels.Everything()
	}
	list := &v1alpha1.AzureIOTHubSourceList{ListMeta: obj.(*v1alpha1.AzureIOTHubSourceList).ListMeta}
	for _, item := range obj.(*v1alpha1.AzureIOTHubSourceList).Items {
		if label.Matches(labels.Set(item.Labels)) {
			list.Items = append(list.Items, item)
		}
	}
	return list, err
}

// Watch returns a watch.Interface that watches the requested azureIOTHubSources.
func (c *FakeAzureIOTHubSources) Watch(ctx context.Context, opts v1.ListOptions) (watch.Interface, error) {
	return c.Fake.
		InvokesWatch(testing.NewWatchAction(azureiothubsourcesResource, c.ns, opts))

}

// Create takes the representation of a azureIOTHubSource and creates it.  Returns the server's representation of the azureIOTHubSource, and an error, if there is any.
func (c *FakeAzureIOTHubSources) Create(ctx context.Context, azureIOTHubSource *v1alpha1.AzureIOTHubSource, opts v1.CreateOptions) (result *v1alpha1.AzureIOTHubSource, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewCreateAction(azureiothubsourcesResource, c.ns, azureIOTHubSource), &v1alpha1.AzureIOTHubSource{})

	if obj == nil {
		return nil, err
	}
	return obj.(*v1alpha1.AzureIOTHubSource), err
}

// Update takes the representation of a azureIOTHubSource and updates it. Returns the server's representation of the azureIOTHubSource, and an error, if there is any.
func (c *FakeAzureIOTHubSources) Update(ctx context.Context, azureIOTHubSource *v1alpha1.AzureIOTHubSource, opts v1.UpdateOptions) (result *v1alpha1.AzureIOTHubSource, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewUpdateAction(azureiothubsourcesResource, c.ns, azureIOTHubSource), &v1alpha1.AzureIOTHubSource{})

	if obj == nil {
		return nil, err
	}
	return obj.(*v1alpha1.AzureIOTHubSource), err
}

// UpdateStatus was generated because the type contains a Status member.
// Add a +genclient:noStatus comment above the type to avoid generating UpdateStatus().
func (c *FakeAzureIOTHubSources) UpdateStatus(ctx context.Context, azureIOTHubSource *v1alpha1.AzureIOTHubSource, opts v1.UpdateOptions) (*v1alpha1.AzureIOTHubSource, error) {
	obj, err := c.Fake.
		Invokes(testing.NewUpdateSubresourceAction(azureiothubsourcesResource, "status", c.ns, azureIOTHubSource), &v1alpha1.AzureIOTHubSource{})

	if obj == nil {
		return nil, err
	}
	return obj.(*v1alpha1.AzureIOTHubSource), err
}

// Delete takes name of the azureIOTHubSource and deletes it. Returns an error if one occurs.
func (c *FakeAzureIOTHubSources) Delete(ctx context.Context, name string, opts v1.DeleteOptions) error {
	_, err := c.Fake.
		Invokes(testing.NewDeleteAction(azureiothubsourcesResource, c.ns, name), &v1alpha1.AzureIOTHubSource{})

	return err
}

// DeleteCollection deletes a collection of objects.
func (c *FakeAzureIOTHubSources) DeleteCollection(ctx context.Context, opts v1.DeleteOptions, listOpts v1.ListOptions) error {
	action := testing.NewDeleteCollectionAction(azureiothubsourcesResource, c.ns, listOpts)

	_, err := c.Fake.Invokes(action, &v1alpha1.AzureIOTHubSourceList{})
	return err
}

// Patch applies the patch and returns the patched azureIOTHubSource.
func (c *FakeAzureIOTHubSources) Patch(ctx context.Context, name string, pt types.PatchType, data []byte, opts v1.PatchOptions, subresources ...string) (result *v1alpha1.AzureIOTHubSource, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewPatchSubresourceAction(azureiothubsourcesResource, c.ns, name, pt, data, subresources...), &v1alpha1.AzureIOTHubSource{})

	if obj == nil {
		return nil, err
	}
	return obj.(*v1alpha1.AzureIOTHubSource), err
}

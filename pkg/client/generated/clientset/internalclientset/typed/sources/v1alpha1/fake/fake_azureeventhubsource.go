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

// FakeAzureEventHubSources implements AzureEventHubSourceInterface
type FakeAzureEventHubSources struct {
	Fake *FakeSourcesV1alpha1
	ns   string
}

var azureeventhubsourcesResource = schema.GroupVersionResource{Group: "sources.triggermesh.io", Version: "v1alpha1", Resource: "azureeventhubsources"}

var azureeventhubsourcesKind = schema.GroupVersionKind{Group: "sources.triggermesh.io", Version: "v1alpha1", Kind: "AzureEventHubSource"}

// Get takes name of the azureEventHubSource, and returns the corresponding azureEventHubSource object, and an error if there is any.
func (c *FakeAzureEventHubSources) Get(ctx context.Context, name string, options v1.GetOptions) (result *v1alpha1.AzureEventHubSource, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewGetAction(azureeventhubsourcesResource, c.ns, name), &v1alpha1.AzureEventHubSource{})

	if obj == nil {
		return nil, err
	}
	return obj.(*v1alpha1.AzureEventHubSource), err
}

// List takes label and field selectors, and returns the list of AzureEventHubSources that match those selectors.
func (c *FakeAzureEventHubSources) List(ctx context.Context, opts v1.ListOptions) (result *v1alpha1.AzureEventHubSourceList, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewListAction(azureeventhubsourcesResource, azureeventhubsourcesKind, c.ns, opts), &v1alpha1.AzureEventHubSourceList{})

	if obj == nil {
		return nil, err
	}

	label, _, _ := testing.ExtractFromListOptions(opts)
	if label == nil {
		label = labels.Everything()
	}
	list := &v1alpha1.AzureEventHubSourceList{ListMeta: obj.(*v1alpha1.AzureEventHubSourceList).ListMeta}
	for _, item := range obj.(*v1alpha1.AzureEventHubSourceList).Items {
		if label.Matches(labels.Set(item.Labels)) {
			list.Items = append(list.Items, item)
		}
	}
	return list, err
}

// Watch returns a watch.Interface that watches the requested azureEventHubSources.
func (c *FakeAzureEventHubSources) Watch(ctx context.Context, opts v1.ListOptions) (watch.Interface, error) {
	return c.Fake.
		InvokesWatch(testing.NewWatchAction(azureeventhubsourcesResource, c.ns, opts))

}

// Create takes the representation of a azureEventHubSource and creates it.  Returns the server's representation of the azureEventHubSource, and an error, if there is any.
func (c *FakeAzureEventHubSources) Create(ctx context.Context, azureEventHubSource *v1alpha1.AzureEventHubSource, opts v1.CreateOptions) (result *v1alpha1.AzureEventHubSource, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewCreateAction(azureeventhubsourcesResource, c.ns, azureEventHubSource), &v1alpha1.AzureEventHubSource{})

	if obj == nil {
		return nil, err
	}
	return obj.(*v1alpha1.AzureEventHubSource), err
}

// Update takes the representation of a azureEventHubSource and updates it. Returns the server's representation of the azureEventHubSource, and an error, if there is any.
func (c *FakeAzureEventHubSources) Update(ctx context.Context, azureEventHubSource *v1alpha1.AzureEventHubSource, opts v1.UpdateOptions) (result *v1alpha1.AzureEventHubSource, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewUpdateAction(azureeventhubsourcesResource, c.ns, azureEventHubSource), &v1alpha1.AzureEventHubSource{})

	if obj == nil {
		return nil, err
	}
	return obj.(*v1alpha1.AzureEventHubSource), err
}

// UpdateStatus was generated because the type contains a Status member.
// Add a +genclient:noStatus comment above the type to avoid generating UpdateStatus().
func (c *FakeAzureEventHubSources) UpdateStatus(ctx context.Context, azureEventHubSource *v1alpha1.AzureEventHubSource, opts v1.UpdateOptions) (*v1alpha1.AzureEventHubSource, error) {
	obj, err := c.Fake.
		Invokes(testing.NewUpdateSubresourceAction(azureeventhubsourcesResource, "status", c.ns, azureEventHubSource), &v1alpha1.AzureEventHubSource{})

	if obj == nil {
		return nil, err
	}
	return obj.(*v1alpha1.AzureEventHubSource), err
}

// Delete takes name of the azureEventHubSource and deletes it. Returns an error if one occurs.
func (c *FakeAzureEventHubSources) Delete(ctx context.Context, name string, opts v1.DeleteOptions) error {
	_, err := c.Fake.
		Invokes(testing.NewDeleteAction(azureeventhubsourcesResource, c.ns, name), &v1alpha1.AzureEventHubSource{})

	return err
}

// DeleteCollection deletes a collection of objects.
func (c *FakeAzureEventHubSources) DeleteCollection(ctx context.Context, opts v1.DeleteOptions, listOpts v1.ListOptions) error {
	action := testing.NewDeleteCollectionAction(azureeventhubsourcesResource, c.ns, listOpts)

	_, err := c.Fake.Invokes(action, &v1alpha1.AzureEventHubSourceList{})
	return err
}

// Patch applies the patch and returns the patched azureEventHubSource.
func (c *FakeAzureEventHubSources) Patch(ctx context.Context, name string, pt types.PatchType, data []byte, opts v1.PatchOptions, subresources ...string) (result *v1alpha1.AzureEventHubSource, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewPatchSubresourceAction(azureeventhubsourcesResource, c.ns, name, pt, data, subresources...), &v1alpha1.AzureEventHubSource{})

	if obj == nil {
		return nil, err
	}
	return obj.(*v1alpha1.AzureEventHubSource), err
}

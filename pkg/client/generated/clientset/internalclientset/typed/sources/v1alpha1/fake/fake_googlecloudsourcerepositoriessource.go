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

// FakeGoogleCloudSourceRepositoriesSources implements GoogleCloudSourceRepositoriesSourceInterface
type FakeGoogleCloudSourceRepositoriesSources struct {
	Fake *FakeSourcesV1alpha1
	ns   string
}

var googlecloudsourcerepositoriessourcesResource = schema.GroupVersionResource{Group: "sources.triggermesh.io", Version: "v1alpha1", Resource: "googlecloudsourcerepositoriessources"}

var googlecloudsourcerepositoriessourcesKind = schema.GroupVersionKind{Group: "sources.triggermesh.io", Version: "v1alpha1", Kind: "GoogleCloudSourceRepositoriesSource"}

// Get takes name of the googleCloudSourceRepositoriesSource, and returns the corresponding googleCloudSourceRepositoriesSource object, and an error if there is any.
func (c *FakeGoogleCloudSourceRepositoriesSources) Get(ctx context.Context, name string, options v1.GetOptions) (result *v1alpha1.GoogleCloudSourceRepositoriesSource, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewGetAction(googlecloudsourcerepositoriessourcesResource, c.ns, name), &v1alpha1.GoogleCloudSourceRepositoriesSource{})

	if obj == nil {
		return nil, err
	}
	return obj.(*v1alpha1.GoogleCloudSourceRepositoriesSource), err
}

// List takes label and field selectors, and returns the list of GoogleCloudSourceRepositoriesSources that match those selectors.
func (c *FakeGoogleCloudSourceRepositoriesSources) List(ctx context.Context, opts v1.ListOptions) (result *v1alpha1.GoogleCloudSourceRepositoriesSourceList, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewListAction(googlecloudsourcerepositoriessourcesResource, googlecloudsourcerepositoriessourcesKind, c.ns, opts), &v1alpha1.GoogleCloudSourceRepositoriesSourceList{})

	if obj == nil {
		return nil, err
	}

	label, _, _ := testing.ExtractFromListOptions(opts)
	if label == nil {
		label = labels.Everything()
	}
	list := &v1alpha1.GoogleCloudSourceRepositoriesSourceList{ListMeta: obj.(*v1alpha1.GoogleCloudSourceRepositoriesSourceList).ListMeta}
	for _, item := range obj.(*v1alpha1.GoogleCloudSourceRepositoriesSourceList).Items {
		if label.Matches(labels.Set(item.Labels)) {
			list.Items = append(list.Items, item)
		}
	}
	return list, err
}

// Watch returns a watch.Interface that watches the requested googleCloudSourceRepositoriesSources.
func (c *FakeGoogleCloudSourceRepositoriesSources) Watch(ctx context.Context, opts v1.ListOptions) (watch.Interface, error) {
	return c.Fake.
		InvokesWatch(testing.NewWatchAction(googlecloudsourcerepositoriessourcesResource, c.ns, opts))

}

// Create takes the representation of a googleCloudSourceRepositoriesSource and creates it.  Returns the server's representation of the googleCloudSourceRepositoriesSource, and an error, if there is any.
func (c *FakeGoogleCloudSourceRepositoriesSources) Create(ctx context.Context, googleCloudSourceRepositoriesSource *v1alpha1.GoogleCloudSourceRepositoriesSource, opts v1.CreateOptions) (result *v1alpha1.GoogleCloudSourceRepositoriesSource, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewCreateAction(googlecloudsourcerepositoriessourcesResource, c.ns, googleCloudSourceRepositoriesSource), &v1alpha1.GoogleCloudSourceRepositoriesSource{})

	if obj == nil {
		return nil, err
	}
	return obj.(*v1alpha1.GoogleCloudSourceRepositoriesSource), err
}

// Update takes the representation of a googleCloudSourceRepositoriesSource and updates it. Returns the server's representation of the googleCloudSourceRepositoriesSource, and an error, if there is any.
func (c *FakeGoogleCloudSourceRepositoriesSources) Update(ctx context.Context, googleCloudSourceRepositoriesSource *v1alpha1.GoogleCloudSourceRepositoriesSource, opts v1.UpdateOptions) (result *v1alpha1.GoogleCloudSourceRepositoriesSource, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewUpdateAction(googlecloudsourcerepositoriessourcesResource, c.ns, googleCloudSourceRepositoriesSource), &v1alpha1.GoogleCloudSourceRepositoriesSource{})

	if obj == nil {
		return nil, err
	}
	return obj.(*v1alpha1.GoogleCloudSourceRepositoriesSource), err
}

// UpdateStatus was generated because the type contains a Status member.
// Add a +genclient:noStatus comment above the type to avoid generating UpdateStatus().
func (c *FakeGoogleCloudSourceRepositoriesSources) UpdateStatus(ctx context.Context, googleCloudSourceRepositoriesSource *v1alpha1.GoogleCloudSourceRepositoriesSource, opts v1.UpdateOptions) (*v1alpha1.GoogleCloudSourceRepositoriesSource, error) {
	obj, err := c.Fake.
		Invokes(testing.NewUpdateSubresourceAction(googlecloudsourcerepositoriessourcesResource, "status", c.ns, googleCloudSourceRepositoriesSource), &v1alpha1.GoogleCloudSourceRepositoriesSource{})

	if obj == nil {
		return nil, err
	}
	return obj.(*v1alpha1.GoogleCloudSourceRepositoriesSource), err
}

// Delete takes name of the googleCloudSourceRepositoriesSource and deletes it. Returns an error if one occurs.
func (c *FakeGoogleCloudSourceRepositoriesSources) Delete(ctx context.Context, name string, opts v1.DeleteOptions) error {
	_, err := c.Fake.
		Invokes(testing.NewDeleteAction(googlecloudsourcerepositoriessourcesResource, c.ns, name), &v1alpha1.GoogleCloudSourceRepositoriesSource{})

	return err
}

// DeleteCollection deletes a collection of objects.
func (c *FakeGoogleCloudSourceRepositoriesSources) DeleteCollection(ctx context.Context, opts v1.DeleteOptions, listOpts v1.ListOptions) error {
	action := testing.NewDeleteCollectionAction(googlecloudsourcerepositoriessourcesResource, c.ns, listOpts)

	_, err := c.Fake.Invokes(action, &v1alpha1.GoogleCloudSourceRepositoriesSourceList{})
	return err
}

// Patch applies the patch and returns the patched googleCloudSourceRepositoriesSource.
func (c *FakeGoogleCloudSourceRepositoriesSources) Patch(ctx context.Context, name string, pt types.PatchType, data []byte, opts v1.PatchOptions, subresources ...string) (result *v1alpha1.GoogleCloudSourceRepositoriesSource, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewPatchSubresourceAction(googlecloudsourcerepositoriessourcesResource, c.ns, name, pt, data, subresources...), &v1alpha1.GoogleCloudSourceRepositoriesSource{})

	if obj == nil {
		return nil, err
	}
	return obj.(*v1alpha1.GoogleCloudSourceRepositoriesSource), err
}

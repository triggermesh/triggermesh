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

// FakeTwilioSources implements TwilioSourceInterface
type FakeTwilioSources struct {
	Fake *FakeSourcesV1alpha1
	ns   string
}

var twiliosourcesResource = schema.GroupVersionResource{Group: "sources.triggermesh.io", Version: "v1alpha1", Resource: "twiliosources"}

var twiliosourcesKind = schema.GroupVersionKind{Group: "sources.triggermesh.io", Version: "v1alpha1", Kind: "TwilioSource"}

// Get takes name of the twilioSource, and returns the corresponding twilioSource object, and an error if there is any.
func (c *FakeTwilioSources) Get(ctx context.Context, name string, options v1.GetOptions) (result *v1alpha1.TwilioSource, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewGetAction(twiliosourcesResource, c.ns, name), &v1alpha1.TwilioSource{})

	if obj == nil {
		return nil, err
	}
	return obj.(*v1alpha1.TwilioSource), err
}

// List takes label and field selectors, and returns the list of TwilioSources that match those selectors.
func (c *FakeTwilioSources) List(ctx context.Context, opts v1.ListOptions) (result *v1alpha1.TwilioSourceList, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewListAction(twiliosourcesResource, twiliosourcesKind, c.ns, opts), &v1alpha1.TwilioSourceList{})

	if obj == nil {
		return nil, err
	}

	label, _, _ := testing.ExtractFromListOptions(opts)
	if label == nil {
		label = labels.Everything()
	}
	list := &v1alpha1.TwilioSourceList{ListMeta: obj.(*v1alpha1.TwilioSourceList).ListMeta}
	for _, item := range obj.(*v1alpha1.TwilioSourceList).Items {
		if label.Matches(labels.Set(item.Labels)) {
			list.Items = append(list.Items, item)
		}
	}
	return list, err
}

// Watch returns a watch.Interface that watches the requested twilioSources.
func (c *FakeTwilioSources) Watch(ctx context.Context, opts v1.ListOptions) (watch.Interface, error) {
	return c.Fake.
		InvokesWatch(testing.NewWatchAction(twiliosourcesResource, c.ns, opts))

}

// Create takes the representation of a twilioSource and creates it.  Returns the server's representation of the twilioSource, and an error, if there is any.
func (c *FakeTwilioSources) Create(ctx context.Context, twilioSource *v1alpha1.TwilioSource, opts v1.CreateOptions) (result *v1alpha1.TwilioSource, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewCreateAction(twiliosourcesResource, c.ns, twilioSource), &v1alpha1.TwilioSource{})

	if obj == nil {
		return nil, err
	}
	return obj.(*v1alpha1.TwilioSource), err
}

// Update takes the representation of a twilioSource and updates it. Returns the server's representation of the twilioSource, and an error, if there is any.
func (c *FakeTwilioSources) Update(ctx context.Context, twilioSource *v1alpha1.TwilioSource, opts v1.UpdateOptions) (result *v1alpha1.TwilioSource, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewUpdateAction(twiliosourcesResource, c.ns, twilioSource), &v1alpha1.TwilioSource{})

	if obj == nil {
		return nil, err
	}
	return obj.(*v1alpha1.TwilioSource), err
}

// UpdateStatus was generated because the type contains a Status member.
// Add a +genclient:noStatus comment above the type to avoid generating UpdateStatus().
func (c *FakeTwilioSources) UpdateStatus(ctx context.Context, twilioSource *v1alpha1.TwilioSource, opts v1.UpdateOptions) (*v1alpha1.TwilioSource, error) {
	obj, err := c.Fake.
		Invokes(testing.NewUpdateSubresourceAction(twiliosourcesResource, "status", c.ns, twilioSource), &v1alpha1.TwilioSource{})

	if obj == nil {
		return nil, err
	}
	return obj.(*v1alpha1.TwilioSource), err
}

// Delete takes name of the twilioSource and deletes it. Returns an error if one occurs.
func (c *FakeTwilioSources) Delete(ctx context.Context, name string, opts v1.DeleteOptions) error {
	_, err := c.Fake.
		Invokes(testing.NewDeleteAction(twiliosourcesResource, c.ns, name), &v1alpha1.TwilioSource{})

	return err
}

// DeleteCollection deletes a collection of objects.
func (c *FakeTwilioSources) DeleteCollection(ctx context.Context, opts v1.DeleteOptions, listOpts v1.ListOptions) error {
	action := testing.NewDeleteCollectionAction(twiliosourcesResource, c.ns, listOpts)

	_, err := c.Fake.Invokes(action, &v1alpha1.TwilioSourceList{})
	return err
}

// Patch applies the patch and returns the patched twilioSource.
func (c *FakeTwilioSources) Patch(ctx context.Context, name string, pt types.PatchType, data []byte, opts v1.PatchOptions, subresources ...string) (result *v1alpha1.TwilioSource, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewPatchSubresourceAction(twiliosourcesResource, c.ns, name, pt, data, subresources...), &v1alpha1.TwilioSource{})

	if obj == nil {
		return nil, err
	}
	return obj.(*v1alpha1.TwilioSource), err
}

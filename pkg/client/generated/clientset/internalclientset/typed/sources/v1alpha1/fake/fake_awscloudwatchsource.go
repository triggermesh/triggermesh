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

// FakeAWSCloudWatchSources implements AWSCloudWatchSourceInterface
type FakeAWSCloudWatchSources struct {
	Fake *FakeSourcesV1alpha1
	ns   string
}

var awscloudwatchsourcesResource = schema.GroupVersionResource{Group: "sources.triggermesh.io", Version: "v1alpha1", Resource: "awscloudwatchsources"}

var awscloudwatchsourcesKind = schema.GroupVersionKind{Group: "sources.triggermesh.io", Version: "v1alpha1", Kind: "AWSCloudWatchSource"}

// Get takes name of the aWSCloudWatchSource, and returns the corresponding aWSCloudWatchSource object, and an error if there is any.
func (c *FakeAWSCloudWatchSources) Get(ctx context.Context, name string, options v1.GetOptions) (result *v1alpha1.AWSCloudWatchSource, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewGetAction(awscloudwatchsourcesResource, c.ns, name), &v1alpha1.AWSCloudWatchSource{})

	if obj == nil {
		return nil, err
	}
	return obj.(*v1alpha1.AWSCloudWatchSource), err
}

// List takes label and field selectors, and returns the list of AWSCloudWatchSources that match those selectors.
func (c *FakeAWSCloudWatchSources) List(ctx context.Context, opts v1.ListOptions) (result *v1alpha1.AWSCloudWatchSourceList, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewListAction(awscloudwatchsourcesResource, awscloudwatchsourcesKind, c.ns, opts), &v1alpha1.AWSCloudWatchSourceList{})

	if obj == nil {
		return nil, err
	}

	label, _, _ := testing.ExtractFromListOptions(opts)
	if label == nil {
		label = labels.Everything()
	}
	list := &v1alpha1.AWSCloudWatchSourceList{ListMeta: obj.(*v1alpha1.AWSCloudWatchSourceList).ListMeta}
	for _, item := range obj.(*v1alpha1.AWSCloudWatchSourceList).Items {
		if label.Matches(labels.Set(item.Labels)) {
			list.Items = append(list.Items, item)
		}
	}
	return list, err
}

// Watch returns a watch.Interface that watches the requested aWSCloudWatchSources.
func (c *FakeAWSCloudWatchSources) Watch(ctx context.Context, opts v1.ListOptions) (watch.Interface, error) {
	return c.Fake.
		InvokesWatch(testing.NewWatchAction(awscloudwatchsourcesResource, c.ns, opts))

}

// Create takes the representation of a aWSCloudWatchSource and creates it.  Returns the server's representation of the aWSCloudWatchSource, and an error, if there is any.
func (c *FakeAWSCloudWatchSources) Create(ctx context.Context, aWSCloudWatchSource *v1alpha1.AWSCloudWatchSource, opts v1.CreateOptions) (result *v1alpha1.AWSCloudWatchSource, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewCreateAction(awscloudwatchsourcesResource, c.ns, aWSCloudWatchSource), &v1alpha1.AWSCloudWatchSource{})

	if obj == nil {
		return nil, err
	}
	return obj.(*v1alpha1.AWSCloudWatchSource), err
}

// Update takes the representation of a aWSCloudWatchSource and updates it. Returns the server's representation of the aWSCloudWatchSource, and an error, if there is any.
func (c *FakeAWSCloudWatchSources) Update(ctx context.Context, aWSCloudWatchSource *v1alpha1.AWSCloudWatchSource, opts v1.UpdateOptions) (result *v1alpha1.AWSCloudWatchSource, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewUpdateAction(awscloudwatchsourcesResource, c.ns, aWSCloudWatchSource), &v1alpha1.AWSCloudWatchSource{})

	if obj == nil {
		return nil, err
	}
	return obj.(*v1alpha1.AWSCloudWatchSource), err
}

// UpdateStatus was generated because the type contains a Status member.
// Add a +genclient:noStatus comment above the type to avoid generating UpdateStatus().
func (c *FakeAWSCloudWatchSources) UpdateStatus(ctx context.Context, aWSCloudWatchSource *v1alpha1.AWSCloudWatchSource, opts v1.UpdateOptions) (*v1alpha1.AWSCloudWatchSource, error) {
	obj, err := c.Fake.
		Invokes(testing.NewUpdateSubresourceAction(awscloudwatchsourcesResource, "status", c.ns, aWSCloudWatchSource), &v1alpha1.AWSCloudWatchSource{})

	if obj == nil {
		return nil, err
	}
	return obj.(*v1alpha1.AWSCloudWatchSource), err
}

// Delete takes name of the aWSCloudWatchSource and deletes it. Returns an error if one occurs.
func (c *FakeAWSCloudWatchSources) Delete(ctx context.Context, name string, opts v1.DeleteOptions) error {
	_, err := c.Fake.
		Invokes(testing.NewDeleteAction(awscloudwatchsourcesResource, c.ns, name), &v1alpha1.AWSCloudWatchSource{})

	return err
}

// DeleteCollection deletes a collection of objects.
func (c *FakeAWSCloudWatchSources) DeleteCollection(ctx context.Context, opts v1.DeleteOptions, listOpts v1.ListOptions) error {
	action := testing.NewDeleteCollectionAction(awscloudwatchsourcesResource, c.ns, listOpts)

	_, err := c.Fake.Invokes(action, &v1alpha1.AWSCloudWatchSourceList{})
	return err
}

// Patch applies the patch and returns the patched aWSCloudWatchSource.
func (c *FakeAWSCloudWatchSources) Patch(ctx context.Context, name string, pt types.PatchType, data []byte, opts v1.PatchOptions, subresources ...string) (result *v1alpha1.AWSCloudWatchSource, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewPatchSubresourceAction(awscloudwatchsourcesResource, c.ns, name, pt, data, subresources...), &v1alpha1.AWSCloudWatchSource{})

	if obj == nil {
		return nil, err
	}
	return obj.(*v1alpha1.AWSCloudWatchSource), err
}

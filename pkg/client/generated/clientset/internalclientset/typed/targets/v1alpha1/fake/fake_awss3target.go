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

	v1alpha1 "github.com/triggermesh/triggermesh/pkg/apis/targets/v1alpha1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	labels "k8s.io/apimachinery/pkg/labels"
	schema "k8s.io/apimachinery/pkg/runtime/schema"
	types "k8s.io/apimachinery/pkg/types"
	watch "k8s.io/apimachinery/pkg/watch"
	testing "k8s.io/client-go/testing"
)

// FakeAWSS3Targets implements AWSS3TargetInterface
type FakeAWSS3Targets struct {
	Fake *FakeTargetsV1alpha1
	ns   string
}

var awss3targetsResource = schema.GroupVersionResource{Group: "targets.triggermesh.io", Version: "v1alpha1", Resource: "awss3targets"}

var awss3targetsKind = schema.GroupVersionKind{Group: "targets.triggermesh.io", Version: "v1alpha1", Kind: "AWSS3Target"}

// Get takes name of the aWSS3Target, and returns the corresponding aWSS3Target object, and an error if there is any.
func (c *FakeAWSS3Targets) Get(ctx context.Context, name string, options v1.GetOptions) (result *v1alpha1.AWSS3Target, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewGetAction(awss3targetsResource, c.ns, name), &v1alpha1.AWSS3Target{})

	if obj == nil {
		return nil, err
	}
	return obj.(*v1alpha1.AWSS3Target), err
}

// List takes label and field selectors, and returns the list of AWSS3Targets that match those selectors.
func (c *FakeAWSS3Targets) List(ctx context.Context, opts v1.ListOptions) (result *v1alpha1.AWSS3TargetList, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewListAction(awss3targetsResource, awss3targetsKind, c.ns, opts), &v1alpha1.AWSS3TargetList{})

	if obj == nil {
		return nil, err
	}

	label, _, _ := testing.ExtractFromListOptions(opts)
	if label == nil {
		label = labels.Everything()
	}
	list := &v1alpha1.AWSS3TargetList{ListMeta: obj.(*v1alpha1.AWSS3TargetList).ListMeta}
	for _, item := range obj.(*v1alpha1.AWSS3TargetList).Items {
		if label.Matches(labels.Set(item.Labels)) {
			list.Items = append(list.Items, item)
		}
	}
	return list, err
}

// Watch returns a watch.Interface that watches the requested aWSS3Targets.
func (c *FakeAWSS3Targets) Watch(ctx context.Context, opts v1.ListOptions) (watch.Interface, error) {
	return c.Fake.
		InvokesWatch(testing.NewWatchAction(awss3targetsResource, c.ns, opts))

}

// Create takes the representation of a aWSS3Target and creates it.  Returns the server's representation of the aWSS3Target, and an error, if there is any.
func (c *FakeAWSS3Targets) Create(ctx context.Context, aWSS3Target *v1alpha1.AWSS3Target, opts v1.CreateOptions) (result *v1alpha1.AWSS3Target, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewCreateAction(awss3targetsResource, c.ns, aWSS3Target), &v1alpha1.AWSS3Target{})

	if obj == nil {
		return nil, err
	}
	return obj.(*v1alpha1.AWSS3Target), err
}

// Update takes the representation of a aWSS3Target and updates it. Returns the server's representation of the aWSS3Target, and an error, if there is any.
func (c *FakeAWSS3Targets) Update(ctx context.Context, aWSS3Target *v1alpha1.AWSS3Target, opts v1.UpdateOptions) (result *v1alpha1.AWSS3Target, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewUpdateAction(awss3targetsResource, c.ns, aWSS3Target), &v1alpha1.AWSS3Target{})

	if obj == nil {
		return nil, err
	}
	return obj.(*v1alpha1.AWSS3Target), err
}

// UpdateStatus was generated because the type contains a Status member.
// Add a +genclient:noStatus comment above the type to avoid generating UpdateStatus().
func (c *FakeAWSS3Targets) UpdateStatus(ctx context.Context, aWSS3Target *v1alpha1.AWSS3Target, opts v1.UpdateOptions) (*v1alpha1.AWSS3Target, error) {
	obj, err := c.Fake.
		Invokes(testing.NewUpdateSubresourceAction(awss3targetsResource, "status", c.ns, aWSS3Target), &v1alpha1.AWSS3Target{})

	if obj == nil {
		return nil, err
	}
	return obj.(*v1alpha1.AWSS3Target), err
}

// Delete takes name of the aWSS3Target and deletes it. Returns an error if one occurs.
func (c *FakeAWSS3Targets) Delete(ctx context.Context, name string, opts v1.DeleteOptions) error {
	_, err := c.Fake.
		Invokes(testing.NewDeleteAction(awss3targetsResource, c.ns, name), &v1alpha1.AWSS3Target{})

	return err
}

// DeleteCollection deletes a collection of objects.
func (c *FakeAWSS3Targets) DeleteCollection(ctx context.Context, opts v1.DeleteOptions, listOpts v1.ListOptions) error {
	action := testing.NewDeleteCollectionAction(awss3targetsResource, c.ns, listOpts)

	_, err := c.Fake.Invokes(action, &v1alpha1.AWSS3TargetList{})
	return err
}

// Patch applies the patch and returns the patched aWSS3Target.
func (c *FakeAWSS3Targets) Patch(ctx context.Context, name string, pt types.PatchType, data []byte, opts v1.PatchOptions, subresources ...string) (result *v1alpha1.AWSS3Target, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewPatchSubresourceAction(awss3targetsResource, c.ns, name, pt, data, subresources...), &v1alpha1.AWSS3Target{})

	if obj == nil {
		return nil, err
	}
	return obj.(*v1alpha1.AWSS3Target), err
}

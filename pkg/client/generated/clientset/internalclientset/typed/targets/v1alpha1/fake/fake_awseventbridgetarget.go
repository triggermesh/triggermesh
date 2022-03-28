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

	v1alpha1 "github.com/triggermesh/triggermesh/pkg/apis/targets/v1alpha1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	labels "k8s.io/apimachinery/pkg/labels"
	schema "k8s.io/apimachinery/pkg/runtime/schema"
	types "k8s.io/apimachinery/pkg/types"
	watch "k8s.io/apimachinery/pkg/watch"
	testing "k8s.io/client-go/testing"
)

// FakeAWSEventBridgeTargets implements AWSEventBridgeTargetInterface
type FakeAWSEventBridgeTargets struct {
	Fake *FakeTargetsV1alpha1
	ns   string
}

var awseventbridgetargetsResource = schema.GroupVersionResource{Group: "targets.triggermesh.io", Version: "v1alpha1", Resource: "awseventbridgetargets"}

var awseventbridgetargetsKind = schema.GroupVersionKind{Group: "targets.triggermesh.io", Version: "v1alpha1", Kind: "AWSEventBridgeTarget"}

// Get takes name of the aWSEventBridgeTarget, and returns the corresponding aWSEventBridgeTarget object, and an error if there is any.
func (c *FakeAWSEventBridgeTargets) Get(ctx context.Context, name string, options v1.GetOptions) (result *v1alpha1.AWSEventBridgeTarget, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewGetAction(awseventbridgetargetsResource, c.ns, name), &v1alpha1.AWSEventBridgeTarget{})

	if obj == nil {
		return nil, err
	}
	return obj.(*v1alpha1.AWSEventBridgeTarget), err
}

// List takes label and field selectors, and returns the list of AWSEventBridgeTargets that match those selectors.
func (c *FakeAWSEventBridgeTargets) List(ctx context.Context, opts v1.ListOptions) (result *v1alpha1.AWSEventBridgeTargetList, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewListAction(awseventbridgetargetsResource, awseventbridgetargetsKind, c.ns, opts), &v1alpha1.AWSEventBridgeTargetList{})

	if obj == nil {
		return nil, err
	}

	label, _, _ := testing.ExtractFromListOptions(opts)
	if label == nil {
		label = labels.Everything()
	}
	list := &v1alpha1.AWSEventBridgeTargetList{ListMeta: obj.(*v1alpha1.AWSEventBridgeTargetList).ListMeta}
	for _, item := range obj.(*v1alpha1.AWSEventBridgeTargetList).Items {
		if label.Matches(labels.Set(item.Labels)) {
			list.Items = append(list.Items, item)
		}
	}
	return list, err
}

// Watch returns a watch.Interface that watches the requested aWSEventBridgeTargets.
func (c *FakeAWSEventBridgeTargets) Watch(ctx context.Context, opts v1.ListOptions) (watch.Interface, error) {
	return c.Fake.
		InvokesWatch(testing.NewWatchAction(awseventbridgetargetsResource, c.ns, opts))

}

// Create takes the representation of a aWSEventBridgeTarget and creates it.  Returns the server's representation of the aWSEventBridgeTarget, and an error, if there is any.
func (c *FakeAWSEventBridgeTargets) Create(ctx context.Context, aWSEventBridgeTarget *v1alpha1.AWSEventBridgeTarget, opts v1.CreateOptions) (result *v1alpha1.AWSEventBridgeTarget, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewCreateAction(awseventbridgetargetsResource, c.ns, aWSEventBridgeTarget), &v1alpha1.AWSEventBridgeTarget{})

	if obj == nil {
		return nil, err
	}
	return obj.(*v1alpha1.AWSEventBridgeTarget), err
}

// Update takes the representation of a aWSEventBridgeTarget and updates it. Returns the server's representation of the aWSEventBridgeTarget, and an error, if there is any.
func (c *FakeAWSEventBridgeTargets) Update(ctx context.Context, aWSEventBridgeTarget *v1alpha1.AWSEventBridgeTarget, opts v1.UpdateOptions) (result *v1alpha1.AWSEventBridgeTarget, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewUpdateAction(awseventbridgetargetsResource, c.ns, aWSEventBridgeTarget), &v1alpha1.AWSEventBridgeTarget{})

	if obj == nil {
		return nil, err
	}
	return obj.(*v1alpha1.AWSEventBridgeTarget), err
}

// UpdateStatus was generated because the type contains a Status member.
// Add a +genclient:noStatus comment above the type to avoid generating UpdateStatus().
func (c *FakeAWSEventBridgeTargets) UpdateStatus(ctx context.Context, aWSEventBridgeTarget *v1alpha1.AWSEventBridgeTarget, opts v1.UpdateOptions) (*v1alpha1.AWSEventBridgeTarget, error) {
	obj, err := c.Fake.
		Invokes(testing.NewUpdateSubresourceAction(awseventbridgetargetsResource, "status", c.ns, aWSEventBridgeTarget), &v1alpha1.AWSEventBridgeTarget{})

	if obj == nil {
		return nil, err
	}
	return obj.(*v1alpha1.AWSEventBridgeTarget), err
}

// Delete takes name of the aWSEventBridgeTarget and deletes it. Returns an error if one occurs.
func (c *FakeAWSEventBridgeTargets) Delete(ctx context.Context, name string, opts v1.DeleteOptions) error {
	_, err := c.Fake.
		Invokes(testing.NewDeleteAction(awseventbridgetargetsResource, c.ns, name), &v1alpha1.AWSEventBridgeTarget{})

	return err
}

// DeleteCollection deletes a collection of objects.
func (c *FakeAWSEventBridgeTargets) DeleteCollection(ctx context.Context, opts v1.DeleteOptions, listOpts v1.ListOptions) error {
	action := testing.NewDeleteCollectionAction(awseventbridgetargetsResource, c.ns, listOpts)

	_, err := c.Fake.Invokes(action, &v1alpha1.AWSEventBridgeTargetList{})
	return err
}

// Patch applies the patch and returns the patched aWSEventBridgeTarget.
func (c *FakeAWSEventBridgeTargets) Patch(ctx context.Context, name string, pt types.PatchType, data []byte, opts v1.PatchOptions, subresources ...string) (result *v1alpha1.AWSEventBridgeTarget, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewPatchSubresourceAction(awseventbridgetargetsResource, c.ns, name, pt, data, subresources...), &v1alpha1.AWSEventBridgeTarget{})

	if obj == nil {
		return nil, err
	}
	return obj.(*v1alpha1.AWSEventBridgeTarget), err
}

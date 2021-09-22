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

// FakeLogzTargets implements LogzTargetInterface
type FakeLogzTargets struct {
	Fake *FakeTargetsV1alpha1
	ns   string
}

var logztargetsResource = schema.GroupVersionResource{Group: "targets.triggermesh.io", Version: "v1alpha1", Resource: "logztargets"}

var logztargetsKind = schema.GroupVersionKind{Group: "targets.triggermesh.io", Version: "v1alpha1", Kind: "LogzTarget"}

// Get takes name of the logzTarget, and returns the corresponding logzTarget object, and an error if there is any.
func (c *FakeLogzTargets) Get(ctx context.Context, name string, options v1.GetOptions) (result *v1alpha1.LogzTarget, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewGetAction(logztargetsResource, c.ns, name), &v1alpha1.LogzTarget{})

	if obj == nil {
		return nil, err
	}
	return obj.(*v1alpha1.LogzTarget), err
}

// List takes label and field selectors, and returns the list of LogzTargets that match those selectors.
func (c *FakeLogzTargets) List(ctx context.Context, opts v1.ListOptions) (result *v1alpha1.LogzTargetList, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewListAction(logztargetsResource, logztargetsKind, c.ns, opts), &v1alpha1.LogzTargetList{})

	if obj == nil {
		return nil, err
	}

	label, _, _ := testing.ExtractFromListOptions(opts)
	if label == nil {
		label = labels.Everything()
	}
	list := &v1alpha1.LogzTargetList{ListMeta: obj.(*v1alpha1.LogzTargetList).ListMeta}
	for _, item := range obj.(*v1alpha1.LogzTargetList).Items {
		if label.Matches(labels.Set(item.Labels)) {
			list.Items = append(list.Items, item)
		}
	}
	return list, err
}

// Watch returns a watch.Interface that watches the requested logzTargets.
func (c *FakeLogzTargets) Watch(ctx context.Context, opts v1.ListOptions) (watch.Interface, error) {
	return c.Fake.
		InvokesWatch(testing.NewWatchAction(logztargetsResource, c.ns, opts))

}

// Create takes the representation of a logzTarget and creates it.  Returns the server's representation of the logzTarget, and an error, if there is any.
func (c *FakeLogzTargets) Create(ctx context.Context, logzTarget *v1alpha1.LogzTarget, opts v1.CreateOptions) (result *v1alpha1.LogzTarget, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewCreateAction(logztargetsResource, c.ns, logzTarget), &v1alpha1.LogzTarget{})

	if obj == nil {
		return nil, err
	}
	return obj.(*v1alpha1.LogzTarget), err
}

// Update takes the representation of a logzTarget and updates it. Returns the server's representation of the logzTarget, and an error, if there is any.
func (c *FakeLogzTargets) Update(ctx context.Context, logzTarget *v1alpha1.LogzTarget, opts v1.UpdateOptions) (result *v1alpha1.LogzTarget, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewUpdateAction(logztargetsResource, c.ns, logzTarget), &v1alpha1.LogzTarget{})

	if obj == nil {
		return nil, err
	}
	return obj.(*v1alpha1.LogzTarget), err
}

// UpdateStatus was generated because the type contains a Status member.
// Add a +genclient:noStatus comment above the type to avoid generating UpdateStatus().
func (c *FakeLogzTargets) UpdateStatus(ctx context.Context, logzTarget *v1alpha1.LogzTarget, opts v1.UpdateOptions) (*v1alpha1.LogzTarget, error) {
	obj, err := c.Fake.
		Invokes(testing.NewUpdateSubresourceAction(logztargetsResource, "status", c.ns, logzTarget), &v1alpha1.LogzTarget{})

	if obj == nil {
		return nil, err
	}
	return obj.(*v1alpha1.LogzTarget), err
}

// Delete takes name of the logzTarget and deletes it. Returns an error if one occurs.
func (c *FakeLogzTargets) Delete(ctx context.Context, name string, opts v1.DeleteOptions) error {
	_, err := c.Fake.
		Invokes(testing.NewDeleteAction(logztargetsResource, c.ns, name), &v1alpha1.LogzTarget{})

	return err
}

// DeleteCollection deletes a collection of objects.
func (c *FakeLogzTargets) DeleteCollection(ctx context.Context, opts v1.DeleteOptions, listOpts v1.ListOptions) error {
	action := testing.NewDeleteCollectionAction(logztargetsResource, c.ns, listOpts)

	_, err := c.Fake.Invokes(action, &v1alpha1.LogzTargetList{})
	return err
}

// Patch applies the patch and returns the patched logzTarget.
func (c *FakeLogzTargets) Patch(ctx context.Context, name string, pt types.PatchType, data []byte, opts v1.PatchOptions, subresources ...string) (result *v1alpha1.LogzTarget, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewPatchSubresourceAction(logztargetsResource, c.ns, name, pt, data, subresources...), &v1alpha1.LogzTarget{})

	if obj == nil {
		return nil, err
	}
	return obj.(*v1alpha1.LogzTarget), err
}

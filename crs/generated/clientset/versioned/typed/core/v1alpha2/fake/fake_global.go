//
// Copyright 2019 HAProxy Technologies LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//    http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Code generated by client-gen. DO NOT EDIT.

package fake

import (
	"context"

	v1alpha2 "github.com/haproxytech/kubernetes-ingress/crs/api/core/v1alpha2"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	labels "k8s.io/apimachinery/pkg/labels"
	schema "k8s.io/apimachinery/pkg/runtime/schema"
	types "k8s.io/apimachinery/pkg/types"
	watch "k8s.io/apimachinery/pkg/watch"
	testing "k8s.io/client-go/testing"
)

// FakeGlobals implements GlobalInterface
type FakeGlobals struct {
	Fake *FakeCoreV1alpha2
	ns   string
}

var globalsResource = schema.GroupVersionResource{Group: "core.haproxy.org", Version: "v1alpha2", Resource: "globals"}

var globalsKind = schema.GroupVersionKind{Group: "core.haproxy.org", Version: "v1alpha2", Kind: "Global"}

// Get takes name of the global, and returns the corresponding global object, and an error if there is any.
func (c *FakeGlobals) Get(ctx context.Context, name string, options v1.GetOptions) (result *v1alpha2.Global, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewGetAction(globalsResource, c.ns, name), &v1alpha2.Global{})

	if obj == nil {
		return nil, err
	}
	return obj.(*v1alpha2.Global), err
}

// List takes label and field selectors, and returns the list of Globals that match those selectors.
func (c *FakeGlobals) List(ctx context.Context, opts v1.ListOptions) (result *v1alpha2.GlobalList, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewListAction(globalsResource, globalsKind, c.ns, opts), &v1alpha2.GlobalList{})

	if obj == nil {
		return nil, err
	}

	label, _, _ := testing.ExtractFromListOptions(opts)
	if label == nil {
		label = labels.Everything()
	}
	list := &v1alpha2.GlobalList{ListMeta: obj.(*v1alpha2.GlobalList).ListMeta}
	for _, item := range obj.(*v1alpha2.GlobalList).Items {
		if label.Matches(labels.Set(item.Labels)) {
			list.Items = append(list.Items, item)
		}
	}
	return list, err
}

// Watch returns a watch.Interface that watches the requested globals.
func (c *FakeGlobals) Watch(ctx context.Context, opts v1.ListOptions) (watch.Interface, error) {
	return c.Fake.
		InvokesWatch(testing.NewWatchAction(globalsResource, c.ns, opts))

}

// Create takes the representation of a global and creates it.  Returns the server's representation of the global, and an error, if there is any.
func (c *FakeGlobals) Create(ctx context.Context, global *v1alpha2.Global, opts v1.CreateOptions) (result *v1alpha2.Global, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewCreateAction(globalsResource, c.ns, global), &v1alpha2.Global{})

	if obj == nil {
		return nil, err
	}
	return obj.(*v1alpha2.Global), err
}

// Update takes the representation of a global and updates it. Returns the server's representation of the global, and an error, if there is any.
func (c *FakeGlobals) Update(ctx context.Context, global *v1alpha2.Global, opts v1.UpdateOptions) (result *v1alpha2.Global, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewUpdateAction(globalsResource, c.ns, global), &v1alpha2.Global{})

	if obj == nil {
		return nil, err
	}
	return obj.(*v1alpha2.Global), err
}

// Delete takes name of the global and deletes it. Returns an error if one occurs.
func (c *FakeGlobals) Delete(ctx context.Context, name string, opts v1.DeleteOptions) error {
	_, err := c.Fake.
		Invokes(testing.NewDeleteAction(globalsResource, c.ns, name), &v1alpha2.Global{})

	return err
}

// DeleteCollection deletes a collection of objects.
func (c *FakeGlobals) DeleteCollection(ctx context.Context, opts v1.DeleteOptions, listOpts v1.ListOptions) error {
	action := testing.NewDeleteCollectionAction(globalsResource, c.ns, listOpts)

	_, err := c.Fake.Invokes(action, &v1alpha2.GlobalList{})
	return err
}

// Patch applies the patch and returns the patched global.
func (c *FakeGlobals) Patch(ctx context.Context, name string, pt types.PatchType, data []byte, opts v1.PatchOptions, subresources ...string) (result *v1alpha2.Global, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewPatchSubresourceAction(globalsResource, c.ns, name, pt, data, subresources...), &v1alpha2.Global{})

	if obj == nil {
		return nil, err
	}
	return obj.(*v1alpha2.Global), err
}

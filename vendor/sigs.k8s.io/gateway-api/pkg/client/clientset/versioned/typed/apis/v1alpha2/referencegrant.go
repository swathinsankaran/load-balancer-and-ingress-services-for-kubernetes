/*
Copyright The Kubernetes Authors.

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

package v1alpha2

import (
	"context"
	"time"

	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	types "k8s.io/apimachinery/pkg/types"
	watch "k8s.io/apimachinery/pkg/watch"
	rest "k8s.io/client-go/rest"
	v1alpha2 "sigs.k8s.io/gateway-api/apis/v1alpha2"
	scheme "sigs.k8s.io/gateway-api/pkg/client/clientset/versioned/scheme"
)

// ReferenceGrantsGetter has a method to return a ReferenceGrantInterface.
// A group's client should implement this interface.
type ReferenceGrantsGetter interface {
	ReferenceGrants(namespace string) ReferenceGrantInterface
}

// ReferenceGrantInterface has methods to work with ReferenceGrant resources.
type ReferenceGrantInterface interface {
	Create(ctx context.Context, referenceGrant *v1alpha2.ReferenceGrant, opts v1.CreateOptions) (*v1alpha2.ReferenceGrant, error)
	Update(ctx context.Context, referenceGrant *v1alpha2.ReferenceGrant, opts v1.UpdateOptions) (*v1alpha2.ReferenceGrant, error)
	Delete(ctx context.Context, name string, opts v1.DeleteOptions) error
	DeleteCollection(ctx context.Context, opts v1.DeleteOptions, listOpts v1.ListOptions) error
	Get(ctx context.Context, name string, opts v1.GetOptions) (*v1alpha2.ReferenceGrant, error)
	List(ctx context.Context, opts v1.ListOptions) (*v1alpha2.ReferenceGrantList, error)
	Watch(ctx context.Context, opts v1.ListOptions) (watch.Interface, error)
	Patch(ctx context.Context, name string, pt types.PatchType, data []byte, opts v1.PatchOptions, subresources ...string) (result *v1alpha2.ReferenceGrant, err error)
	ReferenceGrantExpansion
}

// referenceGrants implements ReferenceGrantInterface
type referenceGrants struct {
	client rest.Interface
	ns     string
}

// newReferenceGrants returns a ReferenceGrants
func newReferenceGrants(c *GatewayV1alpha2Client, namespace string) *referenceGrants {
	return &referenceGrants{
		client: c.RESTClient(),
		ns:     namespace,
	}
}

// Get takes name of the referenceGrant, and returns the corresponding referenceGrant object, and an error if there is any.
func (c *referenceGrants) Get(ctx context.Context, name string, options v1.GetOptions) (result *v1alpha2.ReferenceGrant, err error) {
	result = &v1alpha2.ReferenceGrant{}
	err = c.client.Get().
		Namespace(c.ns).
		Resource("referencegrants").
		Name(name).
		VersionedParams(&options, scheme.ParameterCodec).
		Do(ctx).
		Into(result)
	return
}

// List takes label and field selectors, and returns the list of ReferenceGrants that match those selectors.
func (c *referenceGrants) List(ctx context.Context, opts v1.ListOptions) (result *v1alpha2.ReferenceGrantList, err error) {
	var timeout time.Duration
	if opts.TimeoutSeconds != nil {
		timeout = time.Duration(*opts.TimeoutSeconds) * time.Second
	}
	result = &v1alpha2.ReferenceGrantList{}
	err = c.client.Get().
		Namespace(c.ns).
		Resource("referencegrants").
		VersionedParams(&opts, scheme.ParameterCodec).
		Timeout(timeout).
		Do(ctx).
		Into(result)
	return
}

// Watch returns a watch.Interface that watches the requested referenceGrants.
func (c *referenceGrants) Watch(ctx context.Context, opts v1.ListOptions) (watch.Interface, error) {
	var timeout time.Duration
	if opts.TimeoutSeconds != nil {
		timeout = time.Duration(*opts.TimeoutSeconds) * time.Second
	}
	opts.Watch = true
	return c.client.Get().
		Namespace(c.ns).
		Resource("referencegrants").
		VersionedParams(&opts, scheme.ParameterCodec).
		Timeout(timeout).
		Watch(ctx)
}

// Create takes the representation of a referenceGrant and creates it.  Returns the server's representation of the referenceGrant, and an error, if there is any.
func (c *referenceGrants) Create(ctx context.Context, referenceGrant *v1alpha2.ReferenceGrant, opts v1.CreateOptions) (result *v1alpha2.ReferenceGrant, err error) {
	result = &v1alpha2.ReferenceGrant{}
	err = c.client.Post().
		Namespace(c.ns).
		Resource("referencegrants").
		VersionedParams(&opts, scheme.ParameterCodec).
		Body(referenceGrant).
		Do(ctx).
		Into(result)
	return
}

// Update takes the representation of a referenceGrant and updates it. Returns the server's representation of the referenceGrant, and an error, if there is any.
func (c *referenceGrants) Update(ctx context.Context, referenceGrant *v1alpha2.ReferenceGrant, opts v1.UpdateOptions) (result *v1alpha2.ReferenceGrant, err error) {
	result = &v1alpha2.ReferenceGrant{}
	err = c.client.Put().
		Namespace(c.ns).
		Resource("referencegrants").
		Name(referenceGrant.Name).
		VersionedParams(&opts, scheme.ParameterCodec).
		Body(referenceGrant).
		Do(ctx).
		Into(result)
	return
}

// Delete takes name of the referenceGrant and deletes it. Returns an error if one occurs.
func (c *referenceGrants) Delete(ctx context.Context, name string, opts v1.DeleteOptions) error {
	return c.client.Delete().
		Namespace(c.ns).
		Resource("referencegrants").
		Name(name).
		Body(&opts).
		Do(ctx).
		Error()
}

// DeleteCollection deletes a collection of objects.
func (c *referenceGrants) DeleteCollection(ctx context.Context, opts v1.DeleteOptions, listOpts v1.ListOptions) error {
	var timeout time.Duration
	if listOpts.TimeoutSeconds != nil {
		timeout = time.Duration(*listOpts.TimeoutSeconds) * time.Second
	}
	return c.client.Delete().
		Namespace(c.ns).
		Resource("referencegrants").
		VersionedParams(&listOpts, scheme.ParameterCodec).
		Timeout(timeout).
		Body(&opts).
		Do(ctx).
		Error()
}

// Patch applies the patch and returns the patched referenceGrant.
func (c *referenceGrants) Patch(ctx context.Context, name string, pt types.PatchType, data []byte, opts v1.PatchOptions, subresources ...string) (result *v1alpha2.ReferenceGrant, err error) {
	result = &v1alpha2.ReferenceGrant{}
	err = c.client.Patch(pt).
		Namespace(c.ns).
		Resource("referencegrants").
		Name(name).
		SubResource(subresources...).
		VersionedParams(&opts, scheme.ParameterCodec).
		Body(data).
		Do(ctx).
		Into(result)
	return
}

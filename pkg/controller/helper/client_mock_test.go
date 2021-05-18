package helper

import (
	"context"

	"github.com/epam/edp-keycloak-operator/pkg/apis/v1/v1alpha1"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/runtime"

	"github.com/stretchr/testify/mock"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type Client struct {
	client.Reader
	client.Writer
	client.StatusClient
	mock.Mock
}

func (c *Client) Scheme() *runtime.Scheme {
	panic("not implemented yet")
}

func (c *Client) RESTMapper() meta.RESTMapper {
	panic("not implemented yet")
}

func (c *Client) Get(ctx context.Context, key client.ObjectKey, obj client.Object) error {
	err := c.Called(key, obj).Error(0)

	if kc, ok := obj.(*v1alpha1.Keycloak); ok {
		kc.Status.Connected = true
	}

	return err
}

func (c *Client) List(ctx context.Context, list client.ObjectList, opts ...client.ListOption) error {
	return c.Called(opts, list).Error(0)
}

func (c *Client) Create(ctx context.Context, obj client.Object, options ...client.CreateOption) error {
	return c.Called(obj).Error(0)
}

func (c *Client) Delete(ctx context.Context, obj client.Object, options ...client.DeleteOption) error {
	return c.Called(obj, options).Error(0)
}

func (c *Client) Update(ctx context.Context, obj client.Object, options ...client.UpdateOption) error {
	return c.Called(obj).Error(0)
}

func (c *Client) Status() client.StatusWriter {
	return c.Called().Get(0).(client.StatusWriter)
}

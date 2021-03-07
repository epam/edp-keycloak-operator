package helper

import (
	"context"

	"github.com/epmd-edp/keycloak-operator/pkg/apis/v1/v1alpha1"

	"github.com/epmd-edp/keycloak-operator/pkg/client/keycloak"
	"github.com/epmd-edp/keycloak-operator/pkg/client/keycloak/dto"
	"github.com/stretchr/testify/mock"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type Client struct {
	mock.Mock
}

func (c *Client) Get(ctx context.Context, key client.ObjectKey, obj runtime.Object) error {
	err := c.Called(key, obj).Error(0)

	if kc, ok := obj.(*v1alpha1.Keycloak); ok {
		kc.Status.Connected = true
	}

	return err
}

func (c *Client) List(ctx context.Context, opts *client.ListOptions, list runtime.Object) error {
	return c.Called(opts, list).Error(0)
}

func (c *Client) Create(ctx context.Context, obj runtime.Object) error {
	return c.Called(obj).Error(0)
}

func (c *Client) Delete(ctx context.Context, obj runtime.Object, opts ...client.DeleteOptionFunc) error {
	return c.Called(obj, opts).Error(0)
}

func (c *Client) Update(ctx context.Context, obj runtime.Object) error {
	return c.Called(obj).Error(0)
}

func (c *Client) Status() client.StatusWriter {
	return c.Called().Get(0).(client.StatusWriter)
}

type ClientFactory struct {
	mock.Mock
}

func (c *ClientFactory) New(kc dto.Keycloak) (keycloak.Client, error) {
	args := c.Called(kc)
	return args.Get(0).(keycloak.Client), args.Error(1)
}

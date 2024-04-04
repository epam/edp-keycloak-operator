package helper

import (
	"context"

	"github.com/stretchr/testify/mock"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	commonmock "github.com/epam/edp-common/pkg/mock/controller-runtime/client"
)

// Deprecated: use fake.NewClientBuilder instead.
type K8SClientMock struct {
	mock.Mock
	commonmock.Client
	restMapper meta.RESTMapper
}

func (m *K8SClientMock) Get(ctx context.Context, key client.ObjectKey, obj client.Object, opts ...client.GetOption) error {
	called := m.Called(key, obj)

	parent, ok := called.Get(0).(client.Client)
	if ok {
		return parent.Get(ctx, key, obj)
	}

	return called.Error(0)
}

func (m *K8SClientMock) List(ctx context.Context, list client.ObjectList, opts ...client.ListOption) error {
	called := m.Called(list, opts)

	parent, ok := called.Get(0).(client.Client)
	if ok {
		return parent.List(ctx, list, opts...)
	}

	return called.Error(0)
}

func (m *K8SClientMock) Create(ctx context.Context, obj client.Object, options ...client.CreateOption) error {
	called := m.Called(obj, options)

	parent, ok := called.Get(0).(client.Client)
	if ok {
		return parent.Create(ctx, obj, options...)
	}

	return called.Error(0)
}

func (m *K8SClientMock) Delete(ctx context.Context, obj client.Object, options ...client.DeleteOption) error {
	called := m.Called(obj, options)

	parent, ok := called.Get(0).(client.Client)
	if ok {
		return parent.Delete(ctx, obj, options...)
	}

	return called.Error(0)
}

func (m *K8SClientMock) Update(ctx context.Context, obj client.Object, options ...client.UpdateOption) error {
	called := m.Called(obj, options)

	parent, ok := called.Get(0).(client.Client)
	if ok {
		return parent.Update(ctx, obj, options...)
	}

	return called.Error(0)
}

func (m *K8SClientMock) Patch(_ context.Context, obj client.Object, patch client.Patch, opts ...client.PatchOption) error {
	return m.Called(obj, patch, opts).Error(0)
}

func (m *K8SClientMock) DeleteAllOf(_ context.Context, obj client.Object, opts ...client.DeleteAllOfOption) error {
	return m.Called(obj, opts).Error(0)
}

func (m *K8SClientMock) Scheme() *runtime.Scheme {
	called := m.Called()

	parent, ok := called.Get(0).(client.Client)
	if ok {
		return parent.Scheme()
	}

	return called.Get(0).(*runtime.Scheme)
}

func (m *K8SClientMock) Status() client.StatusWriter {
	called := m.Called()

	parent, ok := called.Get(0).(client.Client)
	if ok {
		return parent.Status()
	}

	return called.Get(0).(client.StatusWriter)
}

func (m *K8SClientMock) RESTMapper() meta.RESTMapper {
	return m.restMapper
}

type K8SStatusMock struct {
	mock.Mock
}

func (m *K8SStatusMock) Create(_ context.Context, obj client.Object, subResource client.Object, opts ...client.SubResourceCreateOption) error {
	return m.Called(obj, subResource, opts).Error(0)
}

func (m *K8SStatusMock) Update(_ context.Context, obj client.Object, opts ...client.SubResourceUpdateOption) error {
	return m.Called(obj, opts).Error(0)
}

func (m *K8SStatusMock) Patch(_ context.Context, obj client.Object, patch client.Patch, opts ...client.SubResourcePatchOption) error {
	return m.Called(obj, patch, opts).Error(0)
}

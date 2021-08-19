package helper

import (
	"context"
	"encoding/json"

	"github.com/stretchr/testify/mock"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type K8SClientMock struct {
	mock.Mock
	scheme     *runtime.Scheme
	status     client.StatusWriter
	restMapper meta.RESTMapper
}

func (m *K8SClientMock) SetScheme(s *runtime.Scheme) {
	m.scheme = s
}

func (m *K8SClientMock) SetStatus(s client.StatusWriter) {
	m.status = s
}

func (m *K8SClientMock) Get(_ context.Context, key client.ObjectKey, obj client.Object) error {
	called := m.Called(key, obj)

	if len(called) > 1 {
		if o := called.Get(1); o != nil {
			object, ok := o.(client.Object)

			if ok {
				bts, _ := json.Marshal(object)
				_ = json.Unmarshal(bts, obj)
			}
		}
	}

	return called.Error(0)
}

func (m *K8SClientMock) List(_ context.Context, list client.ObjectList, opts ...client.ListOption) error {
	return m.Called(list, opts).Error(0)
}

func (m *K8SClientMock) Create(_ context.Context, obj client.Object, opts ...client.CreateOption) error {
	return m.Called(obj, opts).Error(0)
}

func (m *K8SClientMock) Delete(_ context.Context, obj client.Object, opts ...client.DeleteOption) error {
	return m.Called(obj, opts).Error(0)
}

func (m *K8SClientMock) Update(_ context.Context, obj client.Object, opts ...client.UpdateOption) error {
	return m.Called(obj, opts).Error(0)
}

func (m *K8SClientMock) Patch(_ context.Context, obj client.Object, patch client.Patch, opts ...client.PatchOption) error {
	return m.Called(obj, patch, opts).Error(0)
}

func (m *K8SClientMock) DeleteAllOf(_ context.Context, obj client.Object, opts ...client.DeleteAllOfOption) error {
	return m.Called(obj, opts).Error(0)
}

func (m *K8SClientMock) Scheme() *runtime.Scheme {
	return m.scheme
}

func (m *K8SClientMock) Status() client.StatusWriter {
	return m.status
}

func (m *K8SClientMock) RESTMapper() meta.RESTMapper {
	return m.restMapper
}

type K8SStatusMock struct {
	mock.Mock
}

func (m *K8SStatusMock) Update(_ context.Context, obj client.Object, opts ...client.UpdateOption) error {
	return m.Called(obj, opts).Error(0)
}

func (m *K8SStatusMock) Patch(_ context.Context, obj client.Object, patch client.Patch, opts ...client.PatchOption) error {
	return m.Called(obj, patch, opts).Error(0)
}

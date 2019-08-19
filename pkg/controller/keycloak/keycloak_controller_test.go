package keycloak

import (
	"context"
	"errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/scheme"
	"keycloak-operator/pkg/adapter/keycloak"
	"keycloak-operator/pkg/apis/v1/v1alpha1"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"testing"
)

type MockGoCloakAdapter struct {
	mock.Mock
}

func (m *MockGoCloakAdapter) GetConnection(cr v1alpha1.Keycloak) (*keycloak.GoCloakConnection, error) {
	args := m.Called(cr)
	return args.Get(0).(*keycloak.GoCloakConnection), args.Error(1)
}

func TestNewKeycloakCRShouldUpdateStatusConnectedTrue(t *testing.T) {
	//prepare
	//client & scheme

	cr := &v1alpha1.Keycloak{
		ObjectMeta: metav1.ObjectMeta{
			Name: "TestKeycloak",
		},
	}
	objs := []runtime.Object{
		cr,
	}
	s := scheme.Scheme
	s.AddKnownTypes(v1.SchemeGroupVersion, cr, &v1alpha1.KeycloakRealm{})
	client := fake.NewFakeClient(objs...)

	//adapter
	adapter := new(MockGoCloakAdapter)
	adapter.On("GetConnection", mock.Anything).
		Return(&keycloak.GoCloakConnection{}, nil)

	//request
	req := reconcile.Request{
		NamespacedName: types.NamespacedName{
			Name: "TestKeycloak",
		},
	}

	//reconcile
	r := ReconcileKeycloak{
		client:  client,
		scheme:  s,
		adapter: adapter,
	}

	//test
	res, err := r.Reconcile(req)

	//verify
	assert.NoError(t, err)
	assert.False(t, res.Requeue)

	persisted := &v1alpha1.Keycloak{}
	err = client.Get(context.TODO(), req.NamespacedName, persisted)
	assert.True(t, persisted.Status.Connected)
}

func TestNewKeycloakCRShouldUpdateStatusConnectedFalse(t *testing.T) {
	//prepare
	//client & scheme

	cr := &v1alpha1.Keycloak{
		ObjectMeta: metav1.ObjectMeta{
			Name: "TestKeycloak",
		},
	}
	objs := []runtime.Object{
		cr,
	}
	s := scheme.Scheme
	s.AddKnownTypes(v1.SchemeGroupVersion, cr, &v1alpha1.KeycloakRealm{})
	client := fake.NewFakeClient(objs...)

	//adapter
	adapter := new(MockGoCloakAdapter)
	adapter.On("GetConnection", mock.Anything).
		Return(&keycloak.GoCloakConnection{}, errors.New("some test error"))

	//request
	req := reconcile.Request{
		NamespacedName: types.NamespacedName{
			Name: "TestKeycloak",
		},
	}

	//reconcile
	r := ReconcileKeycloak{
		client:  client,
		scheme:  s,
		adapter: adapter,
	}

	//test
	res, err := r.Reconcile(req)

	//verify
	assert.NoError(t, err)
	assert.False(t, res.Requeue)

	persisted := &v1alpha1.Keycloak{}
	err = client.Get(context.TODO(), req.NamespacedName, persisted)
	assert.False(t, persisted.Status.Connected)
}

func TestReconcileCreatesKeycloakRealmWihtOwnerReference(t *testing.T) {
	//prepare
	//client & scheme

	cr := &v1alpha1.Keycloak{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "TestKeycloak",
			Namespace: "test-namespace",
		},
	}
	objs := []runtime.Object{
		cr,
	}
	s := scheme.Scheme
	s.AddKnownTypes(v1.SchemeGroupVersion, cr, &v1alpha1.KeycloakRealm{})
	client := fake.NewFakeClient(objs...)

	//adapter
	adapter := new(MockGoCloakAdapter)
	adapter.On("GetConnection", mock.Anything).
		Return(&keycloak.GoCloakConnection{}, nil)

	//request
	req := reconcile.Request{
		NamespacedName: types.NamespacedName{
			Name:      "TestKeycloak",
			Namespace: "test-namespace",
		},
	}

	//reconcile
	r := ReconcileKeycloak{
		client:  client,
		scheme:  s,
		adapter: adapter,
	}

	//test
	res, err := r.Reconcile(req)

	//verify
	assert.NoError(t, err)
	assert.False(t, res.Requeue)

	persisted := &v1alpha1.KeycloakRealm{}
	nsnRealm := types.NamespacedName{
		Name:      "main",
		Namespace: "test-namespace",
	}
	err = client.Get(context.TODO(), nsnRealm, persisted)

	assert.NotNil(t, persisted)
	assert.Equal(t, "test-namespace.main", persisted.Spec.RealmName)

	or := persisted.GetOwnerReferences()
	assert.True(t, len(or) > 0)
	owner := or[0]
	assert.Equal(t, owner.Name, "TestKeycloak")
}

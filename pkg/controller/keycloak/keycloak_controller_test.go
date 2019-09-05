package keycloak

import (
	"context"
	"errors"
	"github.com/stretchr/testify/assert"
	"k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/scheme"
	"keycloak-operator/pkg/apis/v1/v1alpha1"
	"keycloak-operator/pkg/client/keycloak/mock"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"testing"

	k8sErrors "k8s.io/apimachinery/pkg/api/errors"
)

func TestReconcileKeycloak_ReconcileNewKeycloakCR(t *testing.T) {
	//prepare
	//client & scheme

	cr := &v1alpha1.Keycloak{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "NewKeycloak",
			Namespace: "namespace",
		},
		Spec: v1alpha1.KeycloakSpec{
			Url:  "https://some",
			User: "user",
			Pwd:  "pass",
		},
	}
	objs := []runtime.Object{
		cr,
	}
	s := scheme.Scheme
	s.AddKnownTypes(v1.SchemeGroupVersion, cr, &v1alpha1.KeycloakRealm{})
	client := fake.NewFakeClient(objs...)

	//factory
	factory := new(mock.MockGoCloakFactory)
	factory.On("New", cr.Spec).
		Return(nil, nil)

	//request
	req := reconcile.Request{
		NamespacedName: types.NamespacedName{
			Name:      "NewKeycloak",
			Namespace: "namespace",
		},
	}

	//reconcile
	r := ReconcileKeycloak{
		client:  client,
		scheme:  s,
		factory: factory,
	}

	//test
	res, err := r.Reconcile(req)

	//verify
	assert.NoError(t, err)
	assert.False(t, res.Requeue)

	persisted := &v1alpha1.Keycloak{}
	err = client.Get(context.TODO(), req.NamespacedName, persisted)
	assert.True(t, persisted.Status.Connected)

	realm := &v1alpha1.KeycloakRealm{}
	nsnRealm := types.NamespacedName{
		Name:      "main",
		Namespace: "namespace",
	}
	err = client.Get(context.TODO(), nsnRealm, realm)

	assert.NotNil(t, realm)
	assert.Equal(t, "namespace.main", realm.Spec.RealmName)

	or := realm.GetOwnerReferences()
	assert.True(t, len(or) > 0)
	owner := or[0]
	assert.Equal(t, owner.Name, "NewKeycloak")
}

func TestReconcileKeycloak_ReconcileInvalidSpec(t *testing.T) {
	//prepare
	//client & scheme

	cr := &v1alpha1.Keycloak{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "NewKeycloak",
			Namespace: "namespace",
		},
		Spec: v1alpha1.KeycloakSpec{
			Url:  "https://some",
			User: "user",
			Pwd:  "pass",
		},
	}
	objs := []runtime.Object{
		cr,
	}
	s := scheme.Scheme
	s.AddKnownTypes(v1.SchemeGroupVersion, cr, &v1alpha1.KeycloakRealm{})
	client := fake.NewFakeClient(objs...)

	//factory
	factory := new(mock.MockGoCloakFactory)
	factory.On("New", cr.Spec).
		Return(nil, errors.New("some error"))

	//request
	req := reconcile.Request{
		NamespacedName: types.NamespacedName{
			Name:      "NewKeycloak",
			Namespace: "namespace",
		},
	}

	//reconcile
	r := ReconcileKeycloak{
		client:  client,
		scheme:  s,
		factory: factory,
	}

	//test
	res, err := r.Reconcile(req)

	//verify
	assert.NoError(t, err)
	assert.False(t, res.Requeue)

	persisted := &v1alpha1.Keycloak{}
	err = client.Get(context.TODO(), req.NamespacedName, persisted)
	assert.False(t, persisted.Status.Connected)

	realm := &v1alpha1.KeycloakRealm{}
	nsnRealm := types.NamespacedName{
		Name:      "main",
		Namespace: "namespace",
	}
	err = client.Get(context.TODO(), nsnRealm, realm)

	assert.Error(t, err)

	assert.True(t, k8sErrors.IsNotFound(err))
}

func TestReconcileKeycloak_ReconcileExistingCRRealm(t *testing.T) {
	//prepare
	//client & scheme

	cr := &v1alpha1.Keycloak{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "NewKeycloak",
			Namespace: "namespace",
		},
		Spec: v1alpha1.KeycloakSpec{
			Url:  "https://some",
			User: "user",
			Pwd:  "pass",
		},
	}
	realm := &v1alpha1.KeycloakRealm{
		ObjectMeta: metav1.ObjectMeta{
			Name: "main",
		},
		Spec: v1alpha1.KeycloakRealmSpec{
			RealmName: "namespace.main",
		},
	}
	objs := []runtime.Object{
		cr, realm,
	}
	s := scheme.Scheme
	s.AddKnownTypes(v1.SchemeGroupVersion, cr, realm)
	client := fake.NewFakeClient(objs...)

	//factory
	factory := new(mock.MockGoCloakFactory)
	factory.On("New", cr.Spec).
		Return(nil, nil)

	//request
	req := reconcile.Request{
		NamespacedName: types.NamespacedName{
			Name:      "NewKeycloak",
			Namespace: "namespace",
		},
	}

	//reconcile
	r := ReconcileKeycloak{
		client:  client,
		scheme:  s,
		factory: factory,
	}

	//test
	res, err := r.Reconcile(req)

	//verify
	assert.NoError(t, err)
	assert.False(t, res.Requeue)

	persisted := &v1alpha1.Keycloak{}
	err = client.Get(context.TODO(), req.NamespacedName, persisted)
	assert.True(t, persisted.Status.Connected)

	persistedRealm := &v1alpha1.KeycloakRealm{}
	nsnRealm := types.NamespacedName{
		Name:      "main",
		Namespace: "namespace",
	}
	err = client.Get(context.TODO(), nsnRealm, persistedRealm)

	assert.NotNil(t, realm)
	assert.Equal(t, "namespace.main", persistedRealm.Spec.RealmName)
}

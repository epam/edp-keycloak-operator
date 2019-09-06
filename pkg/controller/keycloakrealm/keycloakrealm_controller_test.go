package keycloakrealm

import (
	"context"
	"errors"
	"github.com/stretchr/testify/assert"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/scheme"
	"keycloak-operator/pkg/apis/v1/v1alpha1"
	"keycloak-operator/pkg/client/keycloak/dto"
	"keycloak-operator/pkg/client/keycloak/mock"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"testing"

	"k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	k8sErrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestReconcileKeycloakRealm_ReconcileNewCr(t *testing.T) {
	//prepare
	//client & scheme
	k := &v1alpha1.Keycloak{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-keycloak",
			Namespace: "namespace",
		},
		Spec: v1alpha1.KeycloakSpec{
			Url:    "https://some",
			Secret: "keycloak-secret",
		},
		Status: v1alpha1.KeycloakStatus{
			Connected: true,
		},
	}
	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "keycloak-secret",
			Namespace: "namespace",
		},
		Data: map[string][]byte{
			"username": []byte("user"),
			"password": []byte("pass"),
		},
	}
	kr := &v1alpha1.KeycloakRealm{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "main",
			Namespace: "namespace",
		},
		Spec: v1alpha1.KeycloakRealmSpec{
			RealmName: "namespace.main",
		},
	}
	kr.SetOwnerReferences([]metav1.OwnerReference{
		{
			Kind: "Keycloak",
			Name: k.Name,
		},
	})
	objs := []runtime.Object{
		k, kr, secret,
	}
	s := scheme.Scheme
	s.AddKnownTypes(v1.SchemeGroupVersion, k, kr, &v1alpha1.KeycloakClient{})
	client := fake.NewFakeClient(objs...)

	//keycloak client and factory

	kclient := new(mock.MockKeycloakClient)
	rDto := dto.ConvertSpecToRealm(kr.Spec)
	kclient.On("ExistRealm", rDto).Return(
		false, nil)
	kclient.On("CreateRealmWithDefaultConfig", rDto).Return(
		nil)
	kclient.On("ExistCentralIdentityProvider", rDto).Return(true, nil)

	keycloakDto := dto.Keycloak{
		Url:  "https://some",
		User: "user",
		Pwd:  "pass",
	}
	factory := new(mock.MockGoCloakFactory)
	factory.On("New", keycloakDto).
		Return(kclient, nil)

	//request
	req := reconcile.Request{
		NamespacedName: types.NamespacedName{
			Name:      "main",
			Namespace: "namespace",
		},
	}

	//reconcile
	r := ReconcileKeycloakRealm{
		client:  client,
		scheme:  s,
		factory: factory,
	}

	//test
	res, err := r.Reconcile(req)

	//verify
	assert.NoError(t, err)
	assert.False(t, res.Requeue)

	persKr := &v1alpha1.KeycloakRealm{}
	err = client.Get(context.TODO(), req.NamespacedName, persKr)
	assert.True(t, persKr.Status.Available)

	persCl := &v1alpha1.KeycloakClient{}
	nsnClient := types.NamespacedName{
		Name:      kr.Spec.RealmName,
		Namespace: "namespace",
	}
	err = client.Get(context.TODO(), nsnClient, persCl)

	assert.NoError(t, err)
	assert.Equal(t, "namespace.main", persCl.Spec.ClientId)
	assert.Equal(t, "openshift", persCl.Spec.TargetRealm)
}

func TestReconcileKeycloakRealm_ReconcileWithoutOwners(t *testing.T) {
	//prepare
	//client & scheme
	kr := &v1alpha1.KeycloakRealm{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "main",
			Namespace: "namespace",
		},
		Spec: v1alpha1.KeycloakRealmSpec{
			RealmName: "namespace.main",
		},
	}
	objs := []runtime.Object{
		kr,
	}
	s := scheme.Scheme
	s.AddKnownTypes(v1.SchemeGroupVersion, kr)
	client := fake.NewFakeClient(objs...)

	//request
	req := reconcile.Request{
		NamespacedName: types.NamespacedName{
			Name:      "main",
			Namespace: "namespace",
		},
	}

	//reconcile
	r := ReconcileKeycloakRealm{
		client: client,
		scheme: s,
	}

	//test
	_, err := r.Reconcile(req)

	//verify
	assert.Error(t, err)

	persKr := &v1alpha1.KeycloakRealm{}
	err = client.Get(context.TODO(), req.NamespacedName, persKr)
	assert.False(t, persKr.Status.Available)

	persCl := &v1alpha1.KeycloakClient{}
	nsnClient := types.NamespacedName{
		Name:      kr.Spec.RealmName,
		Namespace: "namespace",
	}
	err = client.Get(context.TODO(), nsnClient, persCl)

	assert.Error(t, err)
	assert.True(t, k8sErrors.IsNotFound(err))
}

func TestReconcileKeycloakRealm_ReconcileWithoutKeycloakOwner(t *testing.T) {
	//prepare
	//client & scheme
	kr := &v1alpha1.KeycloakRealm{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "main",
			Namespace: "namespace",
		},
		Spec: v1alpha1.KeycloakRealmSpec{
			RealmName: "namespace.main",
		},
	}
	kr.SetOwnerReferences([]metav1.OwnerReference{
		{
			Kind: "anotherType",
			Name: "another",
		},
	})
	objs := []runtime.Object{
		kr,
	}
	s := scheme.Scheme
	s.AddKnownTypes(v1.SchemeGroupVersion, kr, &v1alpha1.KeycloakClient{})
	client := fake.NewFakeClient(objs...)

	//request
	req := reconcile.Request{
		NamespacedName: types.NamespacedName{
			Name:      "main",
			Namespace: "namespace",
		},
	}

	//reconcile
	r := ReconcileKeycloakRealm{
		client: client,
		scheme: s,
	}

	//test
	_, err := r.Reconcile(req)

	//verify
	assert.Error(t, err)

	persKr := &v1alpha1.KeycloakRealm{}
	err = client.Get(context.TODO(), req.NamespacedName, persKr)
	assert.False(t, persKr.Status.Available)

	persCl := &v1alpha1.KeycloakClient{}
	nsnClient := types.NamespacedName{
		Name:      kr.Spec.RealmName,
		Namespace: "namespace",
	}
	err = client.Get(context.TODO(), nsnClient, persCl)

	assert.Error(t, err)
	assert.True(t, k8sErrors.IsNotFound(err))
}

func TestReconcileKeycloakRealm_ReconcileNotConnectedOwner(t *testing.T) {
	//prepare
	//client & scheme
	k := &v1alpha1.Keycloak{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-keycloak",
			Namespace: "namespace",
		},
		Spec: v1alpha1.KeycloakSpec{
			Url:    "https://some",
			Secret: "keycloak-secret",
		},
		Status: v1alpha1.KeycloakStatus{
			Connected: false,
		},
	}
	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "keycloak-secret",
			Namespace: "namespace",
		},
		Data: map[string][]byte{
			"username": []byte("user"),
			"password": []byte("pass"),
		},
	}
	kr := &v1alpha1.KeycloakRealm{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "main",
			Namespace: "namespace",
		},
		Spec: v1alpha1.KeycloakRealmSpec{
			RealmName: "namespace.main",
		},
	}
	kr.SetOwnerReferences([]metav1.OwnerReference{
		{
			Kind: "Keycloak",
			Name: k.Name,
		},
	})
	objs := []runtime.Object{
		k, kr, secret,
	}
	s := scheme.Scheme
	s.AddKnownTypes(v1.SchemeGroupVersion, k, kr, &v1alpha1.KeycloakClient{})
	client := fake.NewFakeClient(objs...)

	//request
	req := reconcile.Request{
		NamespacedName: types.NamespacedName{
			Name:      "main",
			Namespace: "namespace",
		},
	}

	//reconcile
	r := ReconcileKeycloakRealm{
		client: client,
		scheme: s,
	}

	//test
	_, err := r.Reconcile(req)

	//verify
	assert.Error(t, err)

	persKr := &v1alpha1.KeycloakRealm{}
	err = client.Get(context.TODO(), req.NamespacedName, persKr)
	assert.False(t, persKr.Status.Available)

	persCl := &v1alpha1.KeycloakClient{}
	nsnClient := types.NamespacedName{
		Name:      kr.Spec.RealmName,
		Namespace: "namespace",
	}
	err = client.Get(context.TODO(), nsnClient, persCl)

	assert.Error(t, err)
	assert.True(t, k8sErrors.IsNotFound(err))
}

func TestReconcileKeycloakRealm_ReconcileInvalidOwnerCredentials(t *testing.T) {
	//prepare
	//client & scheme
	k := &v1alpha1.Keycloak{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-keycloak",
			Namespace: "namespace",
		},
		Spec: v1alpha1.KeycloakSpec{
			Url:    "https://some",
			Secret: "keycloak-secret",
		},
		Status: v1alpha1.KeycloakStatus{
			Connected: true,
		},
	}
	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "keycloak-secret",
			Namespace: "namespace",
		},
		Data: map[string][]byte{
			"username": []byte("user"),
			"password": []byte("pass"),
		},
	}
	kr := &v1alpha1.KeycloakRealm{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "main",
			Namespace: "namespace",
		},
		Spec: v1alpha1.KeycloakRealmSpec{
			RealmName: "namespace.main",
		},
	}
	kr.SetOwnerReferences([]metav1.OwnerReference{
		{
			Kind: "Keycloak",
			Name: k.Name,
		},
	})
	objs := []runtime.Object{
		k, kr, secret,
	}
	s := scheme.Scheme
	s.AddKnownTypes(v1.SchemeGroupVersion, k, kr, &v1alpha1.KeycloakClient{})
	client := fake.NewFakeClient(objs...)

	//keycloak factory
	kDto := dto.Keycloak{
		Url:  "https://some",
		User: "user",
		Pwd:  "pass",
	}
	factory := new(mock.MockGoCloakFactory)
	factory.On("New", kDto).
		Return(nil, errors.New("error in login"))

	//request
	req := reconcile.Request{
		NamespacedName: types.NamespacedName{
			Name:      "main",
			Namespace: "namespace",
		},
	}

	//reconcile
	r := ReconcileKeycloakRealm{
		client:  client,
		scheme:  s,
		factory: factory,
	}

	//test
	_, err := r.Reconcile(req)

	//verify
	assert.Error(t, err)

	persKr := &v1alpha1.KeycloakRealm{}
	err = client.Get(context.TODO(), req.NamespacedName, persKr)
	assert.False(t, persKr.Status.Available)

	persCl := &v1alpha1.KeycloakClient{}
	nsnClient := types.NamespacedName{
		Name:      kr.Spec.RealmName,
		Namespace: "namespace",
	}
	err = client.Get(context.TODO(), nsnClient, persCl)

	assert.Error(t, err)
	assert.True(t, k8sErrors.IsNotFound(err))
}

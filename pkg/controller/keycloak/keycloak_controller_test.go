package keycloak

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/Nerzal/gocloak/v8"
	edpCompApi "github.com/epam/edp-component-operator/pkg/apis/v1/v1alpha1"
	"github.com/epam/edp-keycloak-operator/pkg/apis/v1/v1alpha1"
	"github.com/epam/edp-keycloak-operator/pkg/client/keycloak/dto"
	"github.com/epam/edp-keycloak-operator/pkg/client/keycloak/mock"
	"github.com/stretchr/testify/assert"
	v1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	k8sErrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

func TestReconcileKeycloak_ReconcileInvalidSpec(t *testing.T) {
	//prepare
	//client & scheme

	cr := &v1alpha1.Keycloak{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "NewKeycloak",
			Namespace: "namespace",
		},
		Spec: v1alpha1.KeycloakSpec{
			Url:    "https://some",
			Secret: "keycloak-secret",
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
	objs := []runtime.Object{
		cr, secret,
	}
	s := scheme.Scheme
	s.AddKnownTypes(v1.SchemeGroupVersion, cr, &v1alpha1.KeycloakRealm{})
	client := fake.NewFakeClient(objs...)

	//factory
	keycloakDto := dto.Keycloak{
		Url:  "https://some",
		User: "user",
		Pwd:  "pass",
	}
	factory := new(mock.GoCloakFactory)
	factory.On("New", keycloakDto).
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
		Client:  client,
		Scheme:  s,
		Factory: factory,
		Log:     &mock.Logger{},
	}

	//test
	res, err := r.Reconcile(context.TODO(), req)

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

func TestReconcileKeycloak_ReconcileCreateMainRealm(t *testing.T) {
	cr := &v1alpha1.Keycloak{ObjectMeta: metav1.ObjectMeta{Name: "NewKeycloak", Namespace: "namespace"},
		Spec: v1alpha1.KeycloakSpec{Url: "https://some", Secret: "keycloak-secret"}}
	secret := &corev1.Secret{ObjectMeta: metav1.ObjectMeta{Name: "keycloak-secret", Namespace: "namespace"},
		Data: map[string][]byte{"username": []byte("user"), "password": []byte("pass")},
	}
	comp := &edpCompApi.EDPComponent{ObjectMeta: metav1.ObjectMeta{Name: fmt.Sprintf("%v-keycloak", cr.Name),
		Namespace: cr.Namespace}}
	s := scheme.Scheme
	s.AddKnownTypes(v1.SchemeGroupVersion, cr, &v1alpha1.KeycloakRealm{}, comp)
	client := fake.NewFakeClient(cr, secret, comp)
	keycloakDto := dto.Keycloak{Url: "https://some", User: "user", Pwd: "pass"}

	kClient := new(mock.KeycloakClient)
	factory := new(mock.GoCloakFactory)
	factory.On("New", keycloakDto).Return(kClient, nil)

	req := reconcile.Request{NamespacedName: types.NamespacedName{Name: cr.Name, Namespace: cr.Namespace}}
	r := ReconcileKeycloak{
		Client:  client,
		Scheme:  s,
		Factory: factory,
		Log:     &mock.Logger{},
	}

	_, err := r.Reconcile(context.TODO(), req)
	if err != nil {
		t.Fatalf("%+v", err)
	}

	if err := client.Get(context.Background(),
		types.NamespacedName{Namespace: cr.Namespace, Name: "main"}, &v1alpha1.KeycloakRealm{}); err != nil {
		t.Fatal(err)
	}
}

func TestReconcileKeycloak_ReconcileDontCreateMainRealm(t *testing.T) {
	cr := &v1alpha1.Keycloak{ObjectMeta: metav1.ObjectMeta{Name: "NewKeycloak", Namespace: "namespace"},
		Spec: v1alpha1.KeycloakSpec{Url: "https://some", Secret: "keycloak-secret",
			InstallMainRealm: gocloak.BoolP(false)}}
	secret := &corev1.Secret{ObjectMeta: metav1.ObjectMeta{Name: "keycloak-secret", Namespace: "namespace"},
		Data: map[string][]byte{"username": []byte("user"), "password": []byte("pass")},
	}
	comp := &edpCompApi.EDPComponent{ObjectMeta: metav1.ObjectMeta{Name: fmt.Sprintf("%v-keycloak", cr.Name),
		Namespace: cr.Namespace}}
	s := scheme.Scheme
	s.AddKnownTypes(v1.SchemeGroupVersion, cr, &v1alpha1.KeycloakRealm{}, comp)
	client := fake.NewFakeClient(cr, secret, comp)
	keycloakDto := dto.Keycloak{Url: "https://some", User: "user", Pwd: "pass"}

	kClient := new(mock.KeycloakClient)
	factory := new(mock.GoCloakFactory)
	factory.On("New", keycloakDto).Return(kClient, nil)

	req := reconcile.Request{NamespacedName: types.NamespacedName{Name: cr.Name, Namespace: cr.Namespace}}
	r := ReconcileKeycloak{
		Client:  client,
		Scheme:  s,
		Factory: factory,
		Log:     &mock.Logger{},
	}

	_, err := r.Reconcile(context.TODO(), req)
	if err != nil {
		t.Fatalf("%+v", err)
	}

	if err := client.Get(context.Background(),
		types.NamespacedName{Namespace: cr.Namespace, Name: "main"}, &v1alpha1.KeycloakRealm{}); err == nil {
		t.Fatal("main realm has been created")
	}
}

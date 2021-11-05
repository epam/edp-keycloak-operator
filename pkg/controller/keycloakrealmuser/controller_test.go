package keycloakrealmuser

import (
	"context"
	"testing"

	"github.com/epam/edp-keycloak-operator/pkg/apis/v1/v1alpha1"
	"github.com/epam/edp-keycloak-operator/pkg/client/keycloak/adapter"
	"github.com/epam/edp-keycloak-operator/pkg/client/keycloak/mock"
	"github.com/epam/edp-keycloak-operator/pkg/controller/helper"
	k8sErrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

func TestNewReconcile_Init(t *testing.T) {
	c := NewReconcile(nil, nil, &mock.Logger{})
	if c.client != nil {
		t.Fatal("something went wrong")
	}
}

func TestNewReconcile(t *testing.T) {
	ns := "namespace1"
	scheme := runtime.NewScheme()
	utilruntime.Must(v1alpha1.AddToScheme(scheme))
	realmName := "realm1"

	usr := v1alpha1.KeycloakRealmUser{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "user321",
			Namespace: ns,
		},
		TypeMeta: metav1.TypeMeta{
			Kind:       "KeycloakRealmUser",
			APIVersion: "v1.edp.epam.com/v1alpha1",
		},
		Spec: v1alpha1.KeycloakRealmUserSpec{
			Email:    "usr@gmail.com",
			Username: "user.g1",
			Realm:    realmName,
		},
	}

	client := fake.NewClientBuilder().WithScheme(scheme).WithRuntimeObjects(&usr).Build()
	h := helper.Mock{}
	log := mock.Logger{}
	realm := v1alpha1.KeycloakRealm{
		Spec: v1alpha1.KeycloakRealmSpec{
			RealmName: realmName,
		},
	}
	kClient := adapter.Mock{}

	h.On("GetOrCreateRealmOwnerRef", &usr, usr.ObjectMeta).Return(&realm, nil)
	h.On("CreateKeycloakClientForRealm", &realm).Return(&kClient, nil)

	r := Reconcile{
		helper: &h,
		log:    &log,
		client: client,
		scheme: scheme,
	}

	adapterUser := adapter.KeycloakUser{
		Username:            usr.Spec.Username,
		Groups:              usr.Spec.Groups,
		Roles:               usr.Spec.Roles,
		RequiredUserActions: usr.Spec.RequiredUserActions,
		LastName:            usr.Spec.LastName,
		FirstName:           usr.Spec.FirstName,
		EmailVerified:       usr.Spec.EmailVerified,
		Enabled:             usr.Spec.Enabled,
		Email:               usr.Spec.Email,
	}

	kClient.On("SyncRealmUser", realmName, &adapterUser, false).Return(nil)

	if _, err := r.Reconcile(context.Background(), reconcile.Request{NamespacedName: types.NamespacedName{
		Namespace: ns,
		Name:      usr.Name,
	}}); err != nil {
		t.Fatal(err)
	}

	var checkUser v1alpha1.KeycloakRealmUser
	err := client.Get(context.Background(), types.NamespacedName{Name: usr.Name, Namespace: usr.Namespace}, &checkUser)
	if err == nil {
		t.Fatal("user is not deleted")
	}

	if !k8sErrors.IsNotFound(err) {
		t.Log(err)
		t.Fatal("wrong error returned")
	}
}

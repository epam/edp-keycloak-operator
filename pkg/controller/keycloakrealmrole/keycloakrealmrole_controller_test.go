package keycloakrealmrole

import (
	"testing"
	"time"

	"github.com/epmd-edp/keycloak-operator/pkg/apis"
	"github.com/epmd-edp/keycloak-operator/pkg/apis/v1/v1alpha1"
	"github.com/epmd-edp/keycloak-operator/pkg/client/keycloak/dto"
	"github.com/epmd-edp/keycloak-operator/pkg/client/keycloak/mock"
	"github.com/epmd-edp/keycloak-operator/pkg/controller/helper"
	v1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

func TestReconcileKeycloakRealmRole_Reconcile(t *testing.T) {
	scheme := runtime.NewScheme()
	if err := apis.AddToScheme(scheme); err != nil {
		t.Fatal(err)
	}
	scheme.AddKnownTypes(v1.SchemeGroupVersion, &corev1.Secret{})

	ns := "security"
	keycloak := v1alpha1.Keycloak{
		ObjectMeta: metav1.ObjectMeta{Name: "test", Namespace: ns}, Status: v1alpha1.KeycloakStatus{Connected: true}}
	realm := v1alpha1.KeycloakRealm{ObjectMeta: metav1.ObjectMeta{Name: "test", Namespace: ns,
		OwnerReferences: []metav1.OwnerReference{{Name: "test", Kind: "Keycloak"}}},
		Spec: v1alpha1.KeycloakRealmSpec{RealmName: "test"}}
	now := metav1.Time{Time: time.Now()}
	role := v1alpha1.KeycloakRealmRole{ObjectMeta: metav1.ObjectMeta{DeletionTimestamp: &now, Name: "test", Namespace: ns,
		OwnerReferences: []metav1.OwnerReference{{Name: "test", Kind: "KeycloakRealm"}}},
		Spec:   v1alpha1.KeycloakRealmRoleSpec{Name: "test"},
		Status: v1alpha1.KeycloakRealmRoleStatus{Value: ""},
	}
	secret := corev1.Secret{ObjectMeta: metav1.ObjectMeta{Name: "keycloak-secret", Namespace: ns},
		Data: map[string][]byte{"username": []byte("user"), "password": []byte("pass")}}

	client := fake.NewFakeClientWithScheme(scheme, &role, &realm, &keycloak, &secret)

	kClient := new(mock.KeycloakClient)
	kClient.On("SyncRealmRole", &dto.Realm{Name: "test", SsoRealmEnabled: true},
		&dto.RealmRole{Name: "test", Composites: []string{}}).Return(nil)
	kClient.On("DeleteRealmRole", "test", "test").Return(nil)
	factory := new(mock.GoCloakFactory)
	factory.On("New", dto.Keycloak{User: "user", Pwd: "pass"}).
		Return(kClient, nil)

	rkr := ReconcileKeycloakRealmRole{
		scheme:  scheme,
		client:  client,
		helper:  helper.MakeHelper(client, scheme),
		factory: factory,
	}

	if _, err := rkr.Reconcile(reconcile.Request{
		NamespacedName: types.NamespacedName{
			Name:      "test",
			Namespace: ns,
		},
	}); err != nil {
		t.Fatalf("%+v", err)
	}
}

package keycloakrealmrole

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/epam/edp-keycloak-operator/pkg/apis"
	"github.com/epam/edp-keycloak-operator/pkg/apis/v1/v1alpha1"
	"github.com/epam/edp-keycloak-operator/pkg/client/keycloak/dto"
	"github.com/epam/edp-keycloak-operator/pkg/client/keycloak/mock"
	"github.com/epam/edp-keycloak-operator/pkg/controller/helper"
	"github.com/pkg/errors"
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
		Spec: v1alpha1.KeycloakRealmSpec{RealmName: "ns.test"}}
	now := metav1.Time{Time: time.Now()}
	role := v1alpha1.KeycloakRealmRole{ObjectMeta: metav1.ObjectMeta{DeletionTimestamp: &now, Name: "test-role", Namespace: ns,
		OwnerReferences: []metav1.OwnerReference{{Name: "test", Kind: "KeycloakRealm"}}},
		Spec:   v1alpha1.KeycloakRealmRoleSpec{Name: "role-test"},
		Status: v1alpha1.KeycloakRealmRoleStatus{Value: ""},
	}
	secret := corev1.Secret{ObjectMeta: metav1.ObjectMeta{Name: "keycloak-secret", Namespace: ns},
		Data: map[string][]byte{"username": []byte("user"), "password": []byte("pass")}}

	client := fake.NewFakeClientWithScheme(scheme, &role, &realm, &keycloak, &secret)

	kClient := new(mock.KeycloakClient)
	kClient.On("SyncRealmRole", &dto.Realm{Name: "ns.test", SsoRealmEnabled: true, SsoAutoRedirectEnabled: true},
		&dto.PrimaryRealmRole{Name: "role-test", Composites: []string{}}).Return(nil)
	kClient.On("DeleteRealmRole", "ns.test", "role-test").Return(nil)
	factory := new(mock.GoCloakFactory)
	factory.On("New", dto.Keycloak{User: "user", Pwd: "pass"}).
		Return(kClient, nil)

	logger := mock.Logger{}

	rkr := ReconcileKeycloakRealmRole{
		scheme:  scheme,
		client:  client,
		helper:  helper.MakeHelper(client, scheme),
		factory: factory,
		logger:  &logger,
	}

	if _, err := rkr.Reconcile(reconcile.Request{
		NamespacedName: types.NamespacedName{
			Name:      "test-role",
			Namespace: ns,
		},
	}); err != nil {
		t.Fatalf("%+v", err)
	}

	var checkRole v1alpha1.KeycloakRealmRole
	if err := client.Get(context.Background(), types.NamespacedName{
		Namespace: ns,
		Name:      "test-role",
	}, &checkRole); err != nil {
		t.Fatal(err)
	}

	if checkRole.Status.Value != helper.StatusOK {
		t.Fatal("role status not updated on success")
	}
}

func TestReconcileKeycloakRealmRole_ReconcileFailure(t *testing.T) {
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

	mockErr := errors.New("test mock fatal")

	kClient := new(mock.KeycloakClient)
	kClient.On("SyncRealmRole", &dto.Realm{Name: "test", SsoRealmEnabled: true, SsoAutoRedirectEnabled: true},
		&dto.PrimaryRealmRole{Name: "test", Composites: []string{}}).Return(mockErr)

	factory := new(mock.GoCloakFactory)
	factory.On("New", dto.Keycloak{User: "user", Pwd: "pass"}).
		Return(kClient, nil)

	logger := mock.Logger{}

	rkr := ReconcileKeycloakRealmRole{scheme: scheme, client: client, helper: helper.MakeHelper(client, scheme),
		factory: factory, logger: &logger}

	_, err := rkr.Reconcile(reconcile.Request{
		NamespacedName: types.NamespacedName{
			Name:      "test",
			Namespace: ns,
		},
	})

	if err != nil {
		t.Fatal(err)
	}

	loggerErr := logger.LastError()
	if loggerErr == nil {
		t.Fatal("no error on mock fatal")
	}

	if errors.Cause(loggerErr) != mockErr {
		t.Fatal("wrong error returned")
	}

	var checkRole v1alpha1.KeycloakRealmRole
	if err := client.Get(context.Background(), types.NamespacedName{
		Namespace: ns,
		Name:      "test",
	}, &checkRole); err != nil {
		t.Fatal(err)
	}

	if !strings.Contains(checkRole.Status.Value, mockErr.Error()) {
		t.Fatal("role status not updated on failure")
	}
}

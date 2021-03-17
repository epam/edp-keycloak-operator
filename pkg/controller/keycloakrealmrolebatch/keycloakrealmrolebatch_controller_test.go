package keycloakrealmrolebatch

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/epmd-edp/keycloak-operator/pkg/apis"
	"github.com/epmd-edp/keycloak-operator/pkg/apis/v1/v1alpha1"
	"github.com/epmd-edp/keycloak-operator/pkg/controller/helper"
	v1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

func TestReconcileKeycloakRealmRoleBatch_Reconcile(t *testing.T) {
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
	secret := corev1.Secret{ObjectMeta: metav1.ObjectMeta{Name: "keycloak-secret", Namespace: ns},
		Data: map[string][]byte{"username": []byte("user"), "password": []byte("pass")}}
	now := metav1.Time{Time: time.Now()}
	batch := v1alpha1.KeycloakRealmRoleBatch{ObjectMeta: metav1.ObjectMeta{Name: "test", Namespace: ns,
		DeletionTimestamp: &now, OwnerReferences: []metav1.OwnerReference{{Name: "test", Kind: "KeycloakRealm"}}},
		Spec: v1alpha1.KeycloakRealmRoleBatchSpec{Realm: "test", Roles: []v1alpha1.BatchRole{
			{Name: "test1"},
			{Name: "test2"},
		}},
		Status: v1alpha1.KeycloakRealmRoleBatchStatus{}}

	role := v1alpha1.KeycloakRealmRole{ObjectMeta: metav1.ObjectMeta{Name: "test2", Namespace: ns,
		OwnerReferences: []metav1.OwnerReference{
			{Name: "test", Kind: "KeycloakRealm"},
			{Name: "test", Kind: batch.Kind},
		}},
		Spec:   v1alpha1.KeycloakRealmRoleSpec{Name: "test"},
		Status: v1alpha1.KeycloakRealmRoleStatus{Value: ""},
	}

	client := fake.NewFakeClientWithScheme(scheme, &batch, &realm, &keycloak, &secret, &role)

	rkr := ReconcileKeycloakRealmRoleBatch{scheme: scheme, client: client, helper: helper.MakeHelper(client, scheme)}

	if _, err := rkr.Reconcile(reconcile.Request{
		NamespacedName: types.NamespacedName{
			Name:      "test",
			Namespace: ns,
		},
	}); err != nil {
		t.Fatal(err)
	}

	var checkBatch v1alpha1.KeycloakRealmRoleBatch
	if err := client.Get(context.Background(), types.NamespacedName{
		Namespace: ns,
		Name:      "test",
	}, &checkBatch); err != nil {
		t.Fatal(err)
	}

	if checkBatch.Status.Value != helper.StatusOK {
		t.Log(checkBatch.Status.Value)
		t.Fatal("batch status not updated on success")
	}
}

func TestReconcileKeycloakRealmRoleBatch_ReconcileFailure(t *testing.T) {
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
	secret := corev1.Secret{ObjectMeta: metav1.ObjectMeta{Name: "keycloak-secret", Namespace: ns},
		Data: map[string][]byte{"username": []byte("user"), "password": []byte("pass")}}
	now := metav1.Time{Time: time.Now()}
	batch := v1alpha1.KeycloakRealmRoleBatch{ObjectMeta: metav1.ObjectMeta{Name: "test", Namespace: ns,
		DeletionTimestamp: &now, OwnerReferences: []metav1.OwnerReference{{Name: "test", Kind: "KeycloakRealm"}}},
		Spec: v1alpha1.KeycloakRealmRoleBatchSpec{Realm: "test", Roles: []v1alpha1.BatchRole{
			{Name: "test1"},
			{Name: "test2"},
		}},
		Status: v1alpha1.KeycloakRealmRoleBatchStatus{}}

	role := v1alpha1.KeycloakRealmRole{ObjectMeta: metav1.ObjectMeta{Name: "test1", Namespace: ns,
		OwnerReferences: []metav1.OwnerReference{{Name: "test", Kind: "KeycloakRealm"}}},
		Spec:   v1alpha1.KeycloakRealmRoleSpec{Name: "test"},
		Status: v1alpha1.KeycloakRealmRoleStatus{Value: ""},
	}

	client := fake.NewFakeClientWithScheme(scheme, &batch, &realm, &keycloak, &secret, &role)

	rkr := ReconcileKeycloakRealmRoleBatch{scheme: scheme, client: client, helper: helper.MakeHelper(client, scheme)}

	_, err := rkr.Reconcile(reconcile.Request{
		NamespacedName: types.NamespacedName{
			Name:      "test",
			Namespace: ns,
		},
	})

	if err == nil {
		t.Fatal("no error on fatal")
	}

	var checkBatch v1alpha1.KeycloakRealmRoleBatch
	if err := client.Get(context.Background(), types.NamespacedName{
		Namespace: ns,
		Name:      "test",
	}, &checkBatch); err != nil {
		t.Fatal(err)
	}

	if !strings.Contains(checkBatch.Status.Value, "one of batch role already exists") {
		t.Log(checkBatch.Status.Value)
		t.Fatal("batch status not updated on failure")
	}
}

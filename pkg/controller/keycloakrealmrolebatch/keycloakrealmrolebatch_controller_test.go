package keycloakrealmrolebatch

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	k8sCLient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	keycloakApi "github.com/epam/edp-keycloak-operator/pkg/apis/v1/v1"
	"github.com/epam/edp-keycloak-operator/pkg/client/keycloak/mock"
	"github.com/epam/edp-keycloak-operator/pkg/controller/helper"
)

func TestReconcileKeycloakRealmRoleBatch_ReconcileDelete(t *testing.T) {
	scheme := runtime.NewScheme()
	utilruntime.Must(keycloakApi.AddToScheme(scheme))
	utilruntime.Must(corev1.AddToScheme(scheme))

	ns := "security"
	keycloak := keycloakApi.Keycloak{
		ObjectMeta: metav1.ObjectMeta{Name: "test", Namespace: ns}, Status: keycloakApi.KeycloakStatus{Connected: true}}
	realm := keycloakApi.KeycloakRealm{ObjectMeta: metav1.ObjectMeta{Name: "test", Namespace: ns,
		OwnerReferences: []metav1.OwnerReference{{Name: "test", Kind: "Keycloak"}}},
		Spec: keycloakApi.KeycloakRealmSpec{RealmName: "test"}}
	secret := corev1.Secret{ObjectMeta: metav1.ObjectMeta{Name: "keycloak-secret", Namespace: ns},
		Data: map[string][]byte{"username": []byte("user"), "password": []byte("pass")}}
	now := metav1.Time{Time: time.Now()}
	batch := keycloakApi.KeycloakRealmRoleBatch{ObjectMeta: metav1.ObjectMeta{Name: "test", Namespace: ns,
		DeletionTimestamp: &now,
		OwnerReferences:   []metav1.OwnerReference{{Name: "test", Kind: "KeycloakRealm"}}},
		Spec: keycloakApi.KeycloakRealmRoleBatchSpec{Realm: "test", Roles: []keycloakApi.BatchRole{
			{Name: "sub-role1"},
			{Name: "sub-role2", IsDefault: true},
		}},
		Status: keycloakApi.KeycloakRealmRoleBatchStatus{}}

	client := fake.NewClientBuilder().WithScheme(scheme).WithRuntimeObjects(&batch, &realm, &keycloak, &secret).Build()

	rkr := ReconcileKeycloakRealmRoleBatch{client: client, helper: helper.MakeHelper(client, scheme, nil),
		log: &mock.Logger{}}

	if _, err := rkr.Reconcile(context.Background(), reconcile.Request{
		NamespacedName: types.NamespacedName{
			Name:      "test",
			Namespace: ns,
		},
	}); err != nil {
		t.Fatal(err)
	}

	var checkList keycloakApi.KeycloakRealmRoleList
	if err := client.List(context.Background(), &checkList, &k8sCLient.ListOptions{}); err != nil {
		t.Fatal(err)
	}

	if len(checkList.Items) > 0 {
		t.Fatal("batch roles is not deleted")
	}

}

func TestReconcileKeycloakRealmRoleBatch_Reconcile(t *testing.T) {
	sch := runtime.NewScheme()
	utilruntime.Must(keycloakApi.AddToScheme(sch))
	utilruntime.Must(corev1.AddToScheme(sch))

	ns := "security"
	keycloak := keycloakApi.Keycloak{
		ObjectMeta: metav1.ObjectMeta{Name: "test", Namespace: ns}, Status: keycloakApi.KeycloakStatus{Connected: true}}
	realm := keycloakApi.KeycloakRealm{ObjectMeta: metav1.ObjectMeta{Name: "test", Namespace: ns,
		OwnerReferences: []metav1.OwnerReference{{Name: "test", Kind: "Keycloak"}}},
		Spec: keycloakApi.KeycloakRealmSpec{RealmName: "test"}}
	secret := corev1.Secret{ObjectMeta: metav1.ObjectMeta{Name: "keycloak-secret", Namespace: ns},
		Data: map[string][]byte{"username": []byte("user"), "password": []byte("pass")}}
	batch := keycloakApi.KeycloakRealmRoleBatch{ObjectMeta: metav1.ObjectMeta{Name: "test", Namespace: ns,
		OwnerReferences: []metav1.OwnerReference{{Name: "test", Kind: "KeycloakRealm"}}},
		Spec: keycloakApi.KeycloakRealmRoleBatchSpec{Realm: "test", Roles: []keycloakApi.BatchRole{
			{Name: "sub-role1"},
			{Name: "sub-role2", IsDefault: true},
		}},
		Status: keycloakApi.KeycloakRealmRoleBatchStatus{}}

	role := keycloakApi.KeycloakRealmRole{ObjectMeta: metav1.ObjectMeta{Name: "test2", Namespace: ns,
		OwnerReferences: []metav1.OwnerReference{
			{Name: "test", Kind: "KeycloakRealm"},
			{Name: "test", Kind: batch.Kind},
		}},
		Spec:   keycloakApi.KeycloakRealmRoleSpec{Name: "test"},
		Status: keycloakApi.KeycloakRealmRoleStatus{Value: ""},
	}

	client := fake.NewClientBuilder().WithScheme(sch).
		WithRuntimeObjects(&batch, &realm, &keycloak, &secret, &role).Build()

	logger := mock.Logger{}

	rkr := ReconcileKeycloakRealmRoleBatch{
		client:                  client,
		helper:                  helper.MakeHelper(client, sch, nil),
		log:                     &logger,
		successReconcileTimeout: time.Hour,
	}

	res, err := rkr.Reconcile(context.TODO(), reconcile.Request{
		NamespacedName: types.NamespacedName{
			Name:      "test",
			Namespace: ns,
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	if err := logger.LastError(); err != nil {
		t.Fatalf("%+v", err)
	}

	if res.RequeueAfter != rkr.successReconcileTimeout {
		t.Fatal("success reconcile timeout is not set")
	}

	var checkBatch keycloakApi.KeycloakRealmRoleBatch
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

	var roles keycloakApi.KeycloakRealmRoleList
	if err := client.List(context.Background(), &roles, &k8sCLient.ListOptions{}); err != nil {
		t.Fatal(err)
	}

	var checkRole keycloakApi.KeycloakRealmRole
	if err := client.Get(context.Background(), types.NamespacedName{Namespace: ns,
		Name: fmt.Sprintf("%s-sub-role2", batch.Name)}, &checkRole); err != nil {
		t.Fatal(err)
	}

	if !checkRole.Spec.IsDefault {
		t.Fatal("sub-role2 is not default")
	}

	checkBatch.Spec.Roles = checkBatch.Spec.Roles[1:]
	if err := client.Update(context.Background(), &checkBatch); err != nil {
		t.Fatal(err)
	}

	if _, err := rkr.Reconcile(context.TODO(), reconcile.Request{
		NamespacedName: types.NamespacedName{
			Name:      "test",
			Namespace: ns,
		},
	}); err != nil {
		t.Fatal(err)
	}

	if err := client.Get(context.Background(), types.NamespacedName{Namespace: ns,
		Name: fmt.Sprintf("%s-sub-role1", batch.Name)}, &checkRole); err == nil {
		t.Fatal("sub role is not marked for deletion")
	}
}

func TestReconcileKeycloakRealmRoleBatch_ReconcileFailure(t *testing.T) {
	scheme := runtime.NewScheme()
	utilruntime.Must(keycloakApi.AddToScheme(scheme))
	utilruntime.Must(corev1.AddToScheme(scheme))

	ns := "security"
	keycloak := keycloakApi.Keycloak{
		ObjectMeta: metav1.ObjectMeta{Name: "keycloak1", Namespace: ns}, Status: keycloakApi.KeycloakStatus{Connected: true}}
	realm := keycloakApi.KeycloakRealm{ObjectMeta: metav1.ObjectMeta{Name: "realm1", Namespace: ns,
		OwnerReferences: []metav1.OwnerReference{{Name: "keycloak1", Kind: "Keycloak"}}},
		Spec: keycloakApi.KeycloakRealmSpec{RealmName: "ns-realm1"}}
	secret := corev1.Secret{ObjectMeta: metav1.ObjectMeta{Name: "keycloak-secret", Namespace: ns},
		Data: map[string][]byte{"username": []byte("user"), "password": []byte("pass")}}
	now := metav1.Time{Time: time.Now()}
	batch := keycloakApi.KeycloakRealmRoleBatch{ObjectMeta: metav1.ObjectMeta{Name: "batch1", Namespace: ns,
		DeletionTimestamp: &now},
		Spec: keycloakApi.KeycloakRealmRoleBatchSpec{Realm: "realm1", Roles: []keycloakApi.BatchRole{
			{Name: "role1"},
			{Name: "role2"},
		}},
		Status: keycloakApi.KeycloakRealmRoleBatchStatus{}}

	role := keycloakApi.KeycloakRealmRole{ObjectMeta: metav1.ObjectMeta{Name: "batch1-role2", Namespace: ns},
		Spec:   keycloakApi.KeycloakRealmRoleSpec{Name: "batch1-role2", Realm: "realm1"},
		Status: keycloakApi.KeycloakRealmRoleStatus{Value: ""},
	}

	client := fake.NewClientBuilder().WithScheme(scheme).
		WithRuntimeObjects(&batch, &realm, &keycloak, &secret, &role).Build()

	logger := mock.Logger{}
	rkr := ReconcileKeycloakRealmRoleBatch{
		client: client,
		helper: helper.MakeHelper(client, scheme, &logger),
		log:    &logger,
	}

	_, err := rkr.Reconcile(context.TODO(), reconcile.Request{
		NamespacedName: types.NamespacedName{
			Name:      "batch1",
			Namespace: ns,
		},
	})

	if err != nil {
		t.Fatal(err)
	}

	logErr := logger.LastError()

	if logErr == nil {
		t.Fatal("no error on fatal")
	}

	if errors.Cause(logErr).Error() != "one of batch role already exists" {
		t.Fatal("wrong error returned")
	}

	var checkBatch keycloakApi.KeycloakRealmRoleBatch
	if err := client.Get(context.Background(), types.NamespacedName{
		Namespace: ns,
		Name:      "batch1",
	}, &checkBatch); err != nil {
		t.Fatal(err)
	}

	if !strings.Contains(checkBatch.Status.Value, "one of batch role already exists") {
		t.Log(checkBatch.Status.Value)
		t.Fatal("batch status not updated on failure")
	}
}

package keycloakrealmrolebatch

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	k8sCLient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	"github.com/epam/edp-keycloak-operator/api/common"
	keycloakApi "github.com/epam/edp-keycloak-operator/api/v1"
	"github.com/epam/edp-keycloak-operator/internal/controller/helper"
	"github.com/epam/edp-keycloak-operator/pkg/client/keycloak/mock"
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
	batch := keycloakApi.KeycloakRealmRoleBatch{
		ObjectMeta: metav1.ObjectMeta{
			Name:              "test",
			Namespace:         ns,
			DeletionTimestamp: &now,
			Finalizers:        []string{keyCloakRealmRoleBatchOperatorFinalizerName},
			OwnerReferences:   []metav1.OwnerReference{{Name: "test", Kind: "KeycloakRealm"}},
		},
		Spec: keycloakApi.KeycloakRealmRoleBatchSpec{
			RealmRef: common.RealmRef{
				Kind: keycloakApi.KeycloakRealmKind,
				Name: realm.Name,
			},
			Roles: []keycloakApi.BatchRole{
				{Name: "sub-role1"},
				{Name: "sub-role2", IsDefault: true},
			},
		},
		Status: keycloakApi.KeycloakRealmRoleBatchStatus{},
	}

	client := fake.NewClientBuilder().WithScheme(scheme).WithRuntimeObjects(&batch, &realm, &keycloak, &secret).Build()
	log := mock.NewLogr()
	rkr := ReconcileKeycloakRealmRoleBatch{
		client: client,
		helper: helper.MakeHelper(client, scheme, "default"),
	}

	_, err := rkr.Reconcile(context.Background(), reconcile.Request{
		NamespacedName: types.NamespacedName{
			Name:      "test",
			Namespace: ns,
		},
	})
	require.NoError(t, err)

	var checkList keycloakApi.KeycloakRealmRoleList
	err = client.List(ctrl.LoggerInto(context.Background(), log), &checkList, &k8sCLient.ListOptions{})
	require.NoError(t, err)

	assert.Empty(t, checkList.Items, "batch roles is not deleted")
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
		Spec: keycloakApi.KeycloakRealmRoleBatchSpec{
			Roles: []keycloakApi.BatchRole{
				{Name: "sub-role1"},
				{Name: "sub-role2", IsDefault: true},
			},
			RealmRef: common.RealmRef{
				Kind: keycloakApi.KeycloakRealmKind,
				Name: realm.Name,
			},
		},
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
		WithRuntimeObjects(&batch, &realm, &keycloak, &secret, &role).WithStatusSubresource(&batch).Build()

	logger := mock.NewLogr()

	rkr := ReconcileKeycloakRealmRoleBatch{
		client:                  client,
		helper:                  helper.MakeHelper(client, sch, "default"),
		successReconcileTimeout: time.Hour,
	}

	res, err := rkr.Reconcile(ctrl.LoggerInto(context.Background(), logger), reconcile.Request{
		NamespacedName: types.NamespacedName{
			Name:      "test",
			Namespace: ns,
		},
	})
	require.NoError(t, err)

	loggerSink, ok := logger.GetSink().(*mock.Logger)
	require.True(t, ok, "wrong logger type")
	require.NoError(t, loggerSink.LastError())

	if res.RequeueAfter != rkr.successReconcileTimeout {
		t.Fatal("success reconcile timeout is not set")
	}

	var checkBatch keycloakApi.KeycloakRealmRoleBatch
	err = client.Get(context.Background(), types.NamespacedName{
		Namespace: ns,
		Name:      "test",
	}, &checkBatch)
	require.NoError(t, err)

	if checkBatch.Status.Value != common.StatusOK {
		t.Log(checkBatch.Status.Value)
		t.Fatal("batch status not updated on success")
	}

	var roles keycloakApi.KeycloakRealmRoleList
	err = client.List(context.Background(), &roles, &k8sCLient.ListOptions{})
	require.NoError(t, err)

	var checkRole keycloakApi.KeycloakRealmRole
	err = client.Get(context.Background(), types.NamespacedName{Namespace: ns,
		Name: fmt.Sprintf("%s-sub-role2", batch.Name)}, &checkRole)
	require.NoError(t, err)

	if !checkRole.Spec.IsDefault {
		t.Fatal("sub-role2 is not default")
	}

	checkBatch.Spec.Roles = checkBatch.Spec.Roles[1:]
	err = client.Update(context.Background(), &checkBatch)
	require.NoError(t, err)

	_, err = rkr.Reconcile(context.TODO(), reconcile.Request{
		NamespacedName: types.NamespacedName{
			Name:      "test",
			Namespace: ns,
		},
	})
	require.NoError(t, err)

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
	batch := keycloakApi.KeycloakRealmRoleBatch{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "batch1",
			Namespace: ns,
		},
		Spec: keycloakApi.KeycloakRealmRoleBatchSpec{
			Roles: []keycloakApi.BatchRole{
				{Name: "role1"},
				{Name: "role2"},
			},
			RealmRef: common.RealmRef{
				Kind: keycloakApi.KeycloakRealmKind,
				Name: realm.Name,
			},
		},
		Status: keycloakApi.KeycloakRealmRoleBatchStatus{}}

	role := keycloakApi.KeycloakRealmRole{ObjectMeta: metav1.ObjectMeta{Name: "batch1-role2", Namespace: ns},
		Spec:   keycloakApi.KeycloakRealmRoleSpec{Name: "batch1-role2", RealmRef: common.RealmRef{Name: "realm1"}},
		Status: keycloakApi.KeycloakRealmRoleStatus{Value: ""},
	}

	client := fake.NewClientBuilder().WithScheme(scheme).
		WithRuntimeObjects(&batch, &realm, &keycloak, &secret, &role).WithStatusSubresource(&batch).Build()

	logger := mock.NewLogr()
	rkr := ReconcileKeycloakRealmRoleBatch{
		client: client,
		helper: helper.MakeHelper(client, scheme, "default"),
	}

	_, err := rkr.Reconcile(ctrl.LoggerInto(context.Background(), logger), reconcile.Request{
		NamespacedName: types.NamespacedName{
			Name:      "batch1",
			Namespace: ns,
		},
	})

	require.NoError(t, err)

	loggerSink, ok := logger.GetSink().(*mock.Logger)
	require.True(t, ok, "wrong logger type")

	require.Error(t, loggerSink.LastError())
	assert.Contains(t, loggerSink.LastError().Error(), "one of batch role already exists")

	var checkBatch keycloakApi.KeycloakRealmRoleBatch
	err = client.Get(context.Background(), types.NamespacedName{
		Namespace: ns,
		Name:      "batch1",
	}, &checkBatch)
	require.NoError(t, err)

	if !strings.Contains(checkBatch.Status.Value, "one of batch role already exists") {
		t.Log(checkBatch.Status.Value)
		t.Fatal("batch status not updated on failure")
	}
}

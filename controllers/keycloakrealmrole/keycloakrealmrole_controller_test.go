package keycloakrealmrole

import (
	"context"
	"testing"
	"time"

	"github.com/Nerzal/gocloak/v12"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	testifymock "github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	"github.com/epam/edp-keycloak-operator/api/common"
	keycloakApi "github.com/epam/edp-keycloak-operator/api/v1"
	"github.com/epam/edp-keycloak-operator/controllers/helper"
	helpermock "github.com/epam/edp-keycloak-operator/controllers/helper/mocks"
	"github.com/epam/edp-keycloak-operator/pkg/client/keycloak/adapter"
	"github.com/epam/edp-keycloak-operator/pkg/client/keycloak/dto"
	"github.com/epam/edp-keycloak-operator/pkg/client/keycloak/mock"
)

func TestReconcileKeycloakRealmRole_Reconcile(t *testing.T) {
	sch := runtime.NewScheme()
	utilruntime.Must(keycloakApi.AddToScheme(sch))
	utilruntime.Must(corev1.AddToScheme(sch))

	ns := "security"
	keycloak := keycloakApi.Keycloak{
		ObjectMeta: metav1.ObjectMeta{Name: "test", Namespace: ns},
		Spec: keycloakApi.KeycloakSpec{
			Secret: "keycloak-secret",
		},
		Status: keycloakApi.KeycloakStatus{Connected: true}}
	realm := keycloakApi.KeycloakRealm{ObjectMeta: metav1.ObjectMeta{Name: "test", Namespace: ns,
		OwnerReferences: []metav1.OwnerReference{{Name: "test", Kind: "Keycloak"}}},
		Spec: keycloakApi.KeycloakRealmSpec{RealmName: "ns.test"}}
	role := keycloakApi.KeycloakRealmRole{TypeMeta: metav1.TypeMeta{
		APIVersion: "v1.edp.epam.com/v1", Kind: "KeycloakRealmRole",
	}, ObjectMeta: metav1.ObjectMeta{ /*DeletionTimestamp: &now,*/ Name: "test-role", Namespace: ns,
		OwnerReferences: []metav1.OwnerReference{{Name: "test", Kind: "KeycloakRealm"}}},
		Spec: keycloakApi.KeycloakRealmRoleSpec{
			Name: "role-test",
			RealmRef: common.RealmRef{
				Kind: keycloakApi.KeycloakRealmKind,
				Name: "realm",
			},
		},
		Status: keycloakApi.KeycloakRealmRoleStatus{Value: helper.StatusOK},
	}
	secret := corev1.Secret{ObjectMeta: metav1.ObjectMeta{Name: "keycloak-secret", Namespace: ns},
		Data: map[string][]byte{"username": []byte("user"), "password": []byte("pass")}}

	client := fake.NewClientBuilder().WithScheme(sch).WithRuntimeObjects(&role, &realm, &keycloak, &secret).Build()

	kClient := new(adapter.Mock)
	kClient.On("SyncRealmRole", "ns.test",
		&dto.PrimaryRealmRole{Name: "role-test", Composites: []string{}}).Return(nil)
	kClient.On("DeleteRealmRole", "ns.test", "role-test").Return(nil)

	logger := mock.NewLogr()
	h := helpermock.NewControllerHelper(t)
	h.On("SetRealmOwnerRef", testifymock.Anything, testifymock.Anything).Return(nil)
	h.On("CreateKeycloakClientFromRealmRef", testifymock.Anything, testifymock.Anything).Return(kClient, nil)
	h.On("GetKeycloakRealmFromRef", testifymock.Anything, testifymock.Anything, testifymock.Anything).
		Return(&gocloak.RealmRepresentation{
			Realm: gocloak.StringP(realm.Spec.RealmName),
		}, nil)
	h.On("TryToDelete", testifymock.Anything, testifymock.Anything, testifymock.Anything, testifymock.Anything).
		Return(false, nil)

	rkr := ReconcileKeycloakRealmRole{
		client: client,
		helper: h,
	}

	res, err := rkr.Reconcile(ctrl.LoggerInto(context.Background(), logger), reconcile.Request{
		NamespacedName: types.NamespacedName{
			Name:      "test-role",
			Namespace: ns,
		},
	})
	if err != nil {
		t.Fatalf("%+v", err)
	}

	loggerSink, ok := logger.GetSink().(*mock.Logger)
	require.True(t, ok, "wrong logger type")
	require.NoError(t, loggerSink.LastError())

	if res.RequeueAfter != rkr.successReconcileTimeout {
		t.Fatal("success reconcile timeout is not set")
	}
}

func TestReconcileDuplicatedRoleIgnore(t *testing.T) {
	ns := "namespace"
	role := keycloakApi.KeycloakRealmRole{ObjectMeta: metav1.ObjectMeta{Name: "test", Namespace: ns,
		OwnerReferences: []metav1.OwnerReference{{Name: "test", Kind: "KeycloakRealm"}}},
		TypeMeta: metav1.TypeMeta{Kind: "KeycloakRealmRole", APIVersion: "v1.edp.epam.com/v1"},
		Spec: keycloakApi.KeycloakRealmRoleSpec{
			Name: "test",
			RealmRef: common.RealmRef{
				Kind: keycloakApi.KeycloakRealmKind,
				Name: "realm",
			},
		},
		Status: keycloakApi.KeycloakRealmRoleStatus{Value: keycloakApi.StatusDuplicated},
	}

	scheme := runtime.NewScheme()
	utilruntime.Must(keycloakApi.AddToScheme(scheme))

	client := fake.NewClientBuilder().WithScheme(scheme).WithRuntimeObjects(&role).Build()
	logger := mock.NewLogr()
	rkr := ReconcileKeycloakRealmRole{
		client: client,
	}

	_, err := rkr.Reconcile(ctrl.LoggerInto(context.Background(), logger), reconcile.Request{
		NamespacedName: types.NamespacedName{
			Name:      role.Name,
			Namespace: role.Namespace,
		}})
	require.NoError(t, err)

	if _, ok := logger.GetSink().(*mock.Logger).InfoMessages()["Role is duplicated, exit."]; !ok {
		t.Fatal("duplicated message is not printed to log")
	}

	var checkRole keycloakApi.KeycloakRealmRole
	err = client.Get(context.Background(), types.NamespacedName{
		Name:      role.Name,
		Namespace: role.Namespace,
	}, &checkRole)
	require.NoError(t, err)

	if checkRole.Status.Value != keycloakApi.StatusDuplicated {
		t.Fatal("wrong status in duplicated role")
	}
}

func TestReconcileRoleMarkDuplicated(t *testing.T) {
	ns := "namespace"
	role := keycloakApi.KeycloakRealmRole{ObjectMeta: metav1.ObjectMeta{Name: "test", Namespace: ns,
		OwnerReferences: []metav1.OwnerReference{{Name: "test", Kind: "KeycloakRealm"}}},
		TypeMeta: metav1.TypeMeta{Kind: "KeycloakRealmRole", APIVersion: "v1.edp.epam.com/v1"},
		Spec: keycloakApi.KeycloakRealmRoleSpec{
			Name: "test",
			RealmRef: common.RealmRef{
				Kind: keycloakApi.KeycloakRealmKind,
				Name: "realm",
			},
		},
		Status: keycloakApi.KeycloakRealmRoleStatus{},
	}

	realm := keycloakApi.KeycloakRealm{ObjectMeta: metav1.ObjectMeta{Name: "test", Namespace: ns,
		OwnerReferences: []metav1.OwnerReference{{Name: "test", Kind: "Keycloak"}}},
		TypeMeta: metav1.TypeMeta{Kind: "KeycloakRealm", APIVersion: "v1.edp.epam.com/v1"},
		Spec:     keycloakApi.KeycloakRealmSpec{RealmName: "test"}}

	scheme := runtime.NewScheme()
	utilruntime.Must(keycloakApi.AddToScheme(scheme))

	client := fake.NewClientBuilder().WithScheme(scheme).WithRuntimeObjects(&role).Build()
	logger := mock.NewLogr()

	prr := dto.ConvertSpecToRole(&role)
	kClient := new(adapter.Mock)
	kClient.On("SyncRealmRole", "test", prr).
		Return(errors.Wrap(adapter.DuplicatedError("dup"), "test unwrap"))

	h := helpermock.NewControllerHelper(t)
	h.On("CreateKeycloakClientFromRealmRef", testifymock.Anything, testifymock.Anything).Return(kClient, nil)
	h.On("SetRealmOwnerRef", testifymock.Anything, testifymock.Anything).Return(nil)
	h.On("GetKeycloakRealmFromRef", testifymock.Anything, testifymock.Anything, testifymock.Anything).
		Return(&gocloak.RealmRepresentation{
			Realm: gocloak.StringP(realm.Spec.RealmName),
		}, nil)

	rkr := ReconcileKeycloakRealmRole{
		client: client,
		helper: h,
	}

	_, err := rkr.Reconcile(ctrl.LoggerInto(context.Background(), logger), reconcile.Request{
		NamespacedName: types.NamespacedName{
			Name:      role.Name,
			Namespace: role.Namespace,
		}})
	require.NoError(t, err)

	if _, ok := logger.GetSink().(*mock.Logger).InfoMessages()["Role is duplicated"]; !ok {
		t.Fatal("duplicated message is not printed to log")
	}
}

func TestReconcileKeycloakRealmRole_ReconcileFailure(t *testing.T) {
	scheme := runtime.NewScheme()
	utilruntime.Must(keycloakApi.AddToScheme(scheme))
	utilruntime.Must(corev1.AddToScheme(scheme))

	ns := "security"
	keycloak := keycloakApi.Keycloak{
		ObjectMeta: metav1.ObjectMeta{Name: "test", Namespace: ns},
		Spec: keycloakApi.KeycloakSpec{
			Secret: "keycloak-secret",
		},
		Status: keycloakApi.KeycloakStatus{Connected: true}}
	realm := keycloakApi.KeycloakRealm{ObjectMeta: metav1.ObjectMeta{Name: "test", Namespace: ns,
		OwnerReferences: []metav1.OwnerReference{{Name: "test", Kind: "Keycloak"}}},
		TypeMeta: metav1.TypeMeta{Kind: "KeycloakRealm", APIVersion: "v1.edp.epam.com/v1"},
		Spec:     keycloakApi.KeycloakRealmSpec{RealmName: "test"}}
	role := keycloakApi.KeycloakRealmRole{ObjectMeta: metav1.ObjectMeta{Name: "test", Namespace: ns,
		OwnerReferences: []metav1.OwnerReference{{Name: "test", Kind: "KeycloakRealm"}}},
		TypeMeta: metav1.TypeMeta{Kind: "KeycloakRealmRole", APIVersion: "v1.edp.epam.com/v1"},
		Spec: keycloakApi.KeycloakRealmRoleSpec{
			Name: "test",
			RealmRef: common.RealmRef{
				Kind: keycloakApi.KeycloakRealmKind,
				Name: "realm",
			},
		},
		Status: keycloakApi.KeycloakRealmRoleStatus{Value: "unable to put role: unable to sync realm role CR: test mock fatal"},
	}
	secret := corev1.Secret{ObjectMeta: metav1.ObjectMeta{Name: "keycloak-secret", Namespace: ns},
		Data: map[string][]byte{"username": []byte("user"), "password": []byte("pass")}}

	client := fake.NewClientBuilder().WithScheme(scheme).WithRuntimeObjects(&role, &realm, &keycloak, &secret).Build()

	mockErr := errors.New("test mock fatal")

	kClient := new(adapter.Mock)
	kClient.On("SyncRealmRole", "test",
		&dto.PrimaryRealmRole{Name: "test", Composites: []string{}}).Return(mockErr)

	h := helpermock.NewControllerHelper(t)
	logger := mock.NewLogr()

	h.On("CreateKeycloakClientFromRealmRef", testifymock.Anything, testifymock.Anything).Return(kClient, nil)
	h.On("SetRealmOwnerRef", testifymock.Anything, testifymock.Anything).Return(nil)
	h.On("SetFailureCount", &role).Return(time.Second)
	h.On("GetKeycloakRealmFromRef", testifymock.Anything, testifymock.Anything, testifymock.Anything).
		Return(&gocloak.RealmRepresentation{
			Realm: gocloak.StringP(realm.Spec.RealmName),
		}, nil)

	rkr := ReconcileKeycloakRealmRole{
		client: client,
		helper: h,
	}

	_, err := rkr.Reconcile(ctrl.LoggerInto(context.Background(), logger), reconcile.Request{
		NamespacedName: types.NamespacedName{
			Name:      "test",
			Namespace: ns,
		},
	})

	require.NoError(t, err)

	loggerSink, ok := logger.GetSink().(*mock.Logger)
	require.True(t, ok, "wrong logger type")
	assert.ErrorIs(t, loggerSink.LastError(), mockErr)
}

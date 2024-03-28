package keycloakauthflow

import (
	"context"
	"testing"
	"time"

	"github.com/Nerzal/gocloak/v12"
	"github.com/pkg/errors"
	testifymock "github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	"github.com/epam/edp-keycloak-operator/api/common"
	keycloakApi "github.com/epam/edp-keycloak-operator/api/v1"
	"github.com/epam/edp-keycloak-operator/controllers/helper"
	helpermock "github.com/epam/edp-keycloak-operator/controllers/helper/mocks"
	"github.com/epam/edp-keycloak-operator/pkg/client/keycloak/adapter"
	"github.com/epam/edp-keycloak-operator/pkg/client/keycloak/mocks"
)

func TestNewReconcile_Init(t *testing.T) {
	c := NewReconcile(nil, helpermock.NewControllerHelper(t))
	if c.client != nil {
		t.Fatal("something went wrong")
	}
}

func TestNewReconcile(t *testing.T) {
	ns := "namespace1"
	scheme := runtime.NewScheme()
	utilruntime.Must(keycloakApi.AddToScheme(scheme))

	flow := keycloakApi.KeycloakAuthFlow{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "flow123",
			Namespace: ns,
		},
		TypeMeta: metav1.TypeMeta{
			Kind:       "KeycloakAuthFlow",
			APIVersion: "v1.edp.epam.com/v1",
		},
		Spec: keycloakApi.KeycloakAuthFlowSpec{
			Alias: "flow123",
			RealmRef: common.RealmRef{
				Kind: keycloakApi.KeycloakRealmKind,
				Name: "realm",
			},
		},
		Status: keycloakApi.KeycloakAuthFlowStatus{
			Value: helper.StatusOK,
		},
	}

	client := fake.NewClientBuilder().WithScheme(scheme).WithRuntimeObjects(&flow).Build()
	h := helpermock.NewControllerHelper(t)
	realm := &keycloakApi.KeycloakRealm{
		Spec: keycloakApi.KeycloakRealmSpec{
			RealmName: "realm11",
		},
	}
	kClient := mocks.NewMockClient(t)

	h.On("SetRealmOwnerRef", testifymock.Anything, testifymock.Anything).Return(nil)
	h.On("CreateKeycloakClientFromRealmRef", testifymock.Anything, testifymock.Anything).Return(kClient, nil)
	h.On("GetKeycloakRealmFromRef", testifymock.Anything, testifymock.Anything, testifymock.Anything).
		Return(&gocloak.RealmRepresentation{
			Realm: gocloak.StringP("realm11"),
		}, nil)

	kClient.On("SyncAuthFlow", realm.Spec.RealmName, &adapter.KeycloakAuthFlow{
		Alias:                    flow.Spec.Alias,
		AuthenticationExecutions: []adapter.AuthenticationExecution{},
	}).Return(nil)

	h.On("TryToDelete", testifymock.Anything, testifymock.Anything, testifymock.Anything, testifymock.Anything).
		Return(false, nil)

	r := Reconcile{
		helper:                  h,
		client:                  client,
		successReconcileTimeout: time.Hour,
	}

	res, err := r.Reconcile(context.Background(), reconcile.Request{NamespacedName: types.NamespacedName{
		Namespace: ns,
		Name:      flow.Name,
	}})
	require.NoError(t, err)

	if res.RequeueAfter != r.successReconcileTimeout {
		t.Fatal("result RequeueAfter is not set")
	}
}

func TestReconcile_Reconcile_Failure(t *testing.T) {
	ns := "namespace1"
	scheme := runtime.NewScheme()
	utilruntime.Must(keycloakApi.AddToScheme(scheme))

	flow := keycloakApi.KeycloakAuthFlow{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "flow123",
			Namespace: ns,
		},
		TypeMeta: metav1.TypeMeta{
			Kind:       "KeycloakAuthFlow",
			APIVersion: "v1.edp.epam.com/v1",
		},
		Spec: keycloakApi.KeycloakAuthFlowSpec{
			Alias: "flow123",
			RealmRef: common.RealmRef{
				Kind: keycloakApi.KeycloakRealmKind,
				Name: "realm",
			},
		},
		Status: keycloakApi.KeycloakAuthFlowStatus{
			Value: "unable to sync auth flow: fatal",
		},
	}

	client := fake.NewClientBuilder().WithScheme(scheme).WithRuntimeObjects(&flow).Build()
	h := helpermock.NewControllerHelper(t)
	realm := keycloakApi.KeycloakRealm{
		Spec: keycloakApi.KeycloakRealmSpec{
			RealmName: "realm11",
		},
	}
	kClient := mocks.NewMockClient(t)

	h.On("SetRealmOwnerRef", testifymock.Anything, testifymock.Anything).Return(nil)
	h.On("CreateKeycloakClientFromRealmRef", testifymock.Anything, testifymock.Anything).Return(kClient, nil)
	h.On("GetKeycloakRealmFromRef", testifymock.Anything, testifymock.Anything, testifymock.Anything).
		Return(&gocloak.RealmRepresentation{
			Realm: gocloak.StringP("realm11"),
		}, nil)
	h.On("SetFailureCount", &flow).Return(time.Second)
	h.On("TryToDelete", testifymock.Anything, testifymock.Anything, testifymock.Anything, testifymock.Anything).
		Return(false, nil)

	mockErr := errors.New("fatal")
	kClient.On("SyncAuthFlow", realm.Spec.RealmName, &adapter.KeycloakAuthFlow{
		Alias:                    flow.Spec.Alias,
		AuthenticationExecutions: []adapter.AuthenticationExecution{},
	}).Return(mockErr)

	r := Reconcile{
		helper: h,
		client: client,
	}

	result, err := r.Reconcile(context.Background(), reconcile.Request{NamespacedName: types.NamespacedName{
		Namespace: ns,
		Name:      flow.Name,
	}})
	require.NoError(t, err)

	if result.RequeueAfter != time.Second {
		t.Fatal("RequeueAfter is not set")
	}
}

func TestReconcile_Reconcile_FailureToGetParentRealm(t *testing.T) {
	ns := "namespace1"
	scheme := runtime.NewScheme()
	utilruntime.Must(keycloakApi.AddToScheme(scheme))

	flow := keycloakApi.KeycloakAuthFlow{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "flow123",
			Namespace: ns,
		},
		TypeMeta: metav1.TypeMeta{
			Kind:       "KeycloakAuthFlow",
			APIVersion: "v1.edp.epam.com/v1",
		},
		Spec: keycloakApi.KeycloakAuthFlowSpec{
			Alias: "flow123",
			RealmRef: common.RealmRef{
				Kind: keycloakApi.KeycloakRealmKind,
				Name: "realm",
			},
		},
		Status: keycloakApi.KeycloakAuthFlowStatus{
			Value: "unable to sync auth flow: fatal",
		},
	}

	client := fake.NewClientBuilder().WithScheme(scheme).WithRuntimeObjects(&flow).Build()
	h := helpermock.NewControllerHelper(t)
	realm := keycloakApi.KeycloakRealm{
		Spec: keycloakApi.KeycloakRealmSpec{
			RealmName: "realm11",
		},
	}
	kClient := mocks.NewMockClient(t)

	h.On("SetRealmOwnerRef", testifymock.Anything, testifymock.Anything).Return(nil)
	h.On("CreateKeycloakClientFromRealmRef", testifymock.Anything, testifymock.Anything).Return(kClient, nil)
	h.On("GetKeycloakRealmFromRef", testifymock.Anything, testifymock.Anything, testifymock.Anything).
		Return(&gocloak.RealmRepresentation{
			Realm: gocloak.StringP("realm11"),
		}, nil)
	h.On("SetFailureCount", &flow).Return(time.Second)
	h.On("TryToDelete", testifymock.Anything, testifymock.Anything, testifymock.Anything, testifymock.Anything).
		Return(false, nil)

	mockErr := errors.New("fatal")
	kClient.On("SyncAuthFlow", realm.Spec.RealmName, &adapter.KeycloakAuthFlow{
		Alias:                    flow.Spec.Alias,
		AuthenticationExecutions: []adapter.AuthenticationExecution{},
	}).Return(mockErr)

	r := Reconcile{
		helper: h,
		client: client,
	}

	result, err := r.Reconcile(context.Background(), reconcile.Request{NamespacedName: types.NamespacedName{
		Namespace: ns,
		Name:      flow.Name,
	}})
	require.NoError(t, err)

	if result.RequeueAfter != time.Second {
		t.Fatal("RequeueAfter is not set")
	}
}

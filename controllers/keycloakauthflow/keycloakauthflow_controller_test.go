package keycloakauthflow

import (
	"context"
	"testing"
	"time"

	"github.com/pkg/errors"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	keycloakApi "github.com/epam/edp-keycloak-operator/api/v1/v1"
	"github.com/epam/edp-keycloak-operator/controllers/helper"
	"github.com/epam/edp-keycloak-operator/pkg/client/keycloak/adapter"
	"github.com/epam/edp-keycloak-operator/pkg/client/keycloak/mock"
)

func TestNewReconcile_Init(t *testing.T) {
	c := NewReconcile(nil, mock.NewLogr(), &helper.Mock{})
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
		},
		Status: keycloakApi.KeycloakAuthFlowStatus{
			Value: helper.StatusOK,
		},
	}

	client := fake.NewClientBuilder().WithScheme(scheme).WithRuntimeObjects(&flow).Build()
	h := helper.Mock{}
	log := mock.NewLogr()
	realm := keycloakApi.KeycloakRealm{
		Spec: keycloakApi.KeycloakRealmSpec{
			RealmName: "realm11",
		},
	}
	kClient := adapter.Mock{}

	h.On("GetOrCreateRealmOwnerRef", &flow, &flow.ObjectMeta).Return(&realm, nil)
	h.On("CreateKeycloakClientForRealm", &realm).Return(&kClient, nil)

	kClient.On("SyncAuthFlow", realm.Spec.RealmName, &adapter.KeycloakAuthFlow{
		Alias:                    flow.Spec.Alias,
		AuthenticationExecutions: []adapter.AuthenticationExecution{},
	}).Return(nil)

	keycloakAuthFlow := authFlowSpecToAdapterAuthFlow(&flow.Spec)

	h.On("TryToDelete", &flow,
		makeTerminator(&realm, keycloakAuthFlow, client, &kClient, log), finalizerName).Return(false, nil)
	h.On("UpdateStatus", &flow).Return(nil)

	r := Reconcile{
		helper:                  &h,
		log:                     log,
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
		},
		Status: keycloakApi.KeycloakAuthFlowStatus{
			Value: "unable to sync auth flow: fatal",
		},
	}

	client := fake.NewClientBuilder().WithScheme(scheme).WithRuntimeObjects(&flow).Build()
	h := helper.Mock{}
	log := mock.NewLogr()
	realm := keycloakApi.KeycloakRealm{
		Spec: keycloakApi.KeycloakRealmSpec{
			RealmName: "realm11",
		},
	}
	kClient := adapter.Mock{}

	h.On("GetOrCreateRealmOwnerRef", &flow, &flow.ObjectMeta).Return(&realm, nil)
	h.On("CreateKeycloakClientForRealm", &realm).Return(&kClient, nil)
	h.On("SetFailureCount", &flow).Return(time.Second)
	h.On("UpdateStatus", &flow).Return(nil)
	h.On("TryToDelete", &flow,
		makeTerminator(&realm, authFlowSpecToAdapterAuthFlow(&flow.Spec), client, &kClient, log),
		finalizerName).Return(false, nil)

	mockErr := errors.New("fatal")
	kClient.On("SyncAuthFlow", realm.Spec.RealmName, &adapter.KeycloakAuthFlow{
		Alias:                    flow.Spec.Alias,
		AuthenticationExecutions: []adapter.AuthenticationExecution{},
	}).Return(mockErr)

	r := Reconcile{
		helper: &h,
		log:    log,
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
		},
		Status: keycloakApi.KeycloakAuthFlowStatus{
			Value: "unable to sync auth flow: fatal",
		},
	}

	client := fake.NewClientBuilder().WithScheme(scheme).WithRuntimeObjects(&flow).Build()
	h := helper.Mock{}
	log := mock.NewLogr()
	realm := keycloakApi.KeycloakRealm{
		Spec: keycloakApi.KeycloakRealmSpec{
			RealmName: "realm11",
		},
	}
	kClient := adapter.Mock{}

	h.On("GetOrCreateRealmOwnerRef", &flow, &flow.ObjectMeta).Return(&realm, nil)
	h.On("CreateKeycloakClientForRealm", &realm).Return(&kClient, nil)
	h.On("SetFailureCount", &flow).Return(time.Second)
	h.On("UpdateStatus", &flow).Return(nil)
	h.On("TryToDelete", &flow,
		makeTerminator(&realm, authFlowSpecToAdapterAuthFlow(&flow.Spec), client, &kClient, log),
		finalizerName).Return(false, nil)

	mockErr := errors.New("fatal")
	kClient.On("SyncAuthFlow", realm.Spec.RealmName, &adapter.KeycloakAuthFlow{
		Alias:                    flow.Spec.Alias,
		AuthenticationExecutions: []adapter.AuthenticationExecution{},
	}).Return(mockErr)

	r := Reconcile{
		helper: &h,
		log:    log,
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

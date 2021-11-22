package keycloakauthflow

import (
	"context"
	"testing"
	"time"

	"github.com/epam/edp-keycloak-operator/pkg/apis/v1/v1alpha1"
	"github.com/epam/edp-keycloak-operator/pkg/client/keycloak/adapter"
	"github.com/epam/edp-keycloak-operator/pkg/client/keycloak/mock"
	"github.com/epam/edp-keycloak-operator/pkg/controller/helper"
	"github.com/pkg/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

func TestNewReconcile_Init(t *testing.T) {
	c := NewReconcile(nil, &mock.Logger{}, &helper.Mock{})
	if c.client != nil {
		t.Fatal("something went wrong")
	}
}

func TestNewReconcile(t *testing.T) {
	ns := "namespace1"
	scheme := runtime.NewScheme()
	utilruntime.Must(v1alpha1.AddToScheme(scheme))
	flow := v1alpha1.KeycloakAuthFlow{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "flow123",
			Namespace: ns,
		},
		TypeMeta: metav1.TypeMeta{
			Kind:       "KeycloakAuthFlow",
			APIVersion: "v1.edp.epam.com/v1alpha1",
		},
		Spec: v1alpha1.KeycloakAuthFlowSpec{
			Alias: "flow123",
		},
		Status: v1alpha1.KeycloakAuthFlowStatus{
			Value: helper.StatusOK,
		},
	}

	client := fake.NewClientBuilder().WithScheme(scheme).WithRuntimeObjects(&flow).Build()
	h := helper.Mock{}
	log := mock.Logger{}
	realm := v1alpha1.KeycloakRealm{
		Spec: v1alpha1.KeycloakRealmSpec{
			RealmName: "realm11",
		},
	}
	kClient := adapter.Mock{}

	h.On("GetOrCreateRealmOwnerRef", &flow, flow.ObjectMeta).Return(&realm, nil)
	h.On("CreateKeycloakClientForRealm", &realm).Return(&kClient, nil)

	kClient.On("SyncAuthFlow", realm.Spec.RealmName, &adapter.KeycloakAuthFlow{
		Alias:                    flow.Spec.Alias,
		AuthenticationExecutions: []adapter.AuthenticationExecution{},
	}).Return(nil)

	h.On("TryToDelete", &flow,
		makeTerminator(realm.Spec.RealmName, flow.Spec.Alias, &kClient, &log), finalizerName).Return(false, nil)
	h.On("UpdateStatus", &flow).Return(nil)

	r := Reconcile{
		helper:                  &h,
		log:                     &log,
		client:                  client,
		successReconcileTimeout: time.Hour,
	}

	res, err := r.Reconcile(context.Background(), reconcile.Request{NamespacedName: types.NamespacedName{
		Namespace: ns,
		Name:      flow.Name,
	}})
	if err != nil {
		t.Fatal(err)
	}

	if res.RequeueAfter != r.successReconcileTimeout {
		t.Fatal("result RequeueAfter is not set")
	}
}

func TestReconcile_Reconcile_Failure(t *testing.T) {
	ns := "namespace1"
	scheme := runtime.NewScheme()
	utilruntime.Must(v1alpha1.AddToScheme(scheme))
	flow := v1alpha1.KeycloakAuthFlow{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "flow123",
			Namespace: ns,
		},
		TypeMeta: metav1.TypeMeta{
			Kind:       "KeycloakAuthFlow",
			APIVersion: "v1.edp.epam.com/v1alpha1",
		},
		Spec: v1alpha1.KeycloakAuthFlowSpec{
			Alias: "flow123",
		},
		Status: v1alpha1.KeycloakAuthFlowStatus{
			Value: "unable to sync auth flow: fatal",
		},
	}

	client := fake.NewClientBuilder().WithScheme(scheme).WithRuntimeObjects(&flow).Build()
	h := helper.Mock{}
	log := mock.Logger{}
	realm := v1alpha1.KeycloakRealm{
		Spec: v1alpha1.KeycloakRealmSpec{
			RealmName: "realm11",
		},
	}
	kClient := adapter.Mock{}

	h.On("GetOrCreateRealmOwnerRef", &flow, flow.ObjectMeta).Return(&realm, nil)
	h.On("CreateKeycloakClientForRealm", &realm).Return(&kClient, nil)
	h.On("SetFailureCount", &flow).Return(time.Second)
	h.On("UpdateStatus", &flow).Return(nil)

	mockErr := errors.New("fatal")
	kClient.On("SyncAuthFlow", realm.Spec.RealmName, &adapter.KeycloakAuthFlow{
		Alias:                    flow.Spec.Alias,
		AuthenticationExecutions: []adapter.AuthenticationExecution{},
	}).Return(mockErr)

	r := Reconcile{
		helper: &h,
		log:    &log,
		client: client,
	}

	result, err := r.Reconcile(context.Background(), reconcile.Request{NamespacedName: types.NamespacedName{
		Namespace: ns,
		Name:      flow.Name,
	}})
	if err != nil {
		t.Fatal(err)
	}

	if result.RequeueAfter != time.Second {
		t.Fatal("RequeueAfter is not set")
	}
}

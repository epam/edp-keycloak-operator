package keycloakclient

import (
	"context"
	"testing"
	"time"

	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	v1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	keycloakApi "github.com/epam/edp-keycloak-operator/pkg/apis/v1/v1"
	"github.com/epam/edp-keycloak-operator/pkg/client/keycloak/adapter"
	"github.com/epam/edp-keycloak-operator/pkg/client/keycloak/mock"
	"github.com/epam/edp-keycloak-operator/pkg/controller/helper"
	"github.com/epam/edp-keycloak-operator/pkg/controller/keycloakclient/chain"
)

func TestReconcileKeycloakClient_WithoutOwnerReference(t *testing.T) {
	kc := &keycloakApi.KeycloakClient{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "main",
			Namespace: "namespace",
		},
		TypeMeta: metav1.TypeMeta{
			Kind:       "KeycloakClient",
			APIVersion: "apps/v1",
		},
		Spec: keycloakApi.KeycloakClientSpec{
			TargetRealm: "main",
			Secret:      "keycloak-secret",
			RealmRoles: &[]keycloakApi.RealmRole{
				{
					Name:      "fake-client-administrators",
					Composite: "administrator",
				},
				{
					Name:      "fake-client-users",
					Composite: "developer",
				},
			},
			Public:                  true,
			ClientId:                "fake-client",
			WebUrl:                  "fake-url",
			DirectAccess:            false,
			AdvancedProtocolMappers: true,
			ClientRoles:             nil,
		},
		Status: keycloakApi.KeycloakClientStatus{
			Value: "error during kc chain: fatal",
		},
	}
	s := scheme.Scheme
	s.AddKnownTypes(v1.SchemeGroupVersion, kc)
	client := fake.NewClientBuilder().WithRuntimeObjects(kc).Build()

	//request
	req := reconcile.Request{
		NamespacedName: types.NamespacedName{
			Name:      "main",
			Namespace: "namespace",
		},
	}

	logger := mock.Logger{}
	h := helper.Mock{}
	chainMock := chain.Mock{}
	realm := keycloakApi.KeycloakRealm{}
	kClient := adapter.Mock{}

	chainMock.On("Serve", kc).Return(errors.New("fatal"))

	h.On("SetFailureCount", kc).Return(time.Second)
	h.On("UpdateStatus", kc).Return(nil)
	h.On("GetOrCreateRealmOwnerRef", &clientRealmFinder{parent: kc,
		client: client}, kc.ObjectMeta).Return(&realm, nil)
	h.On("CreateKeycloakClientForRealm", &realm).Return(&kClient, nil)

	//reconcile
	r := ReconcileKeycloakClient{
		client: client,
		helper: &h,
		log:    &logger,
		chain:  &chainMock,
	}

	//test
	res, err := r.Reconcile(context.TODO(), req)

	//verify
	assert.Nil(t, err)
	if res.RequeueAfter <= 0 {
		t.Fatal("requeue duration is not changed")
	}
	assert.False(t, res.Requeue)

	persKc := &keycloakApi.KeycloakClient{}
	assert.Nil(t, client.Get(context.TODO(), req.NamespacedName, persKc))
	assert.Equal(t, "error during kc chain: fatal", persKc.Status.Value)
	assert.Empty(t, persKc.Status.ClientID)
}

func TestReconcileKeycloakClient_ReconcileWithMappers(t *testing.T) {
	kc := keycloakApi.KeycloakClient{ObjectMeta: metav1.ObjectMeta{Name: "main", Namespace: "namespace"},
		TypeMeta: metav1.TypeMeta{
			Kind:       "KeycloakClient",
			APIVersion: "apps/v1",
		},
		Spec: keycloakApi.KeycloakClientSpec{TargetRealm: "namespace.main", Secret: "keycloak-secret",
			RealmRoles: &[]keycloakApi.RealmRole{{Name: "fake-client-administrators", Composite: "administrator"},
				{Name: "fake-client-users", Composite: "developer"},
			}, Public: true, ClientId: "fake-client", WebUrl: "fake-url", DirectAccess: false,
			AdvancedProtocolMappers: true, ClientRoles: nil, ProtocolMappers: &[]keycloakApi.ProtocolMapper{
				{Name: "bar", Config: map[string]string{"bar": "1"}},
				{Name: "foo", Config: map[string]string{"foo": "2"}},
			},
		},
		Status: keycloakApi.KeycloakClientStatus{
			Value: helper.StatusOK,
		},
	}

	s := scheme.Scheme
	s.AddKnownTypes(v1.SchemeGroupVersion, &kc)
	client := fake.NewClientBuilder().WithRuntimeObjects(&kc).Build()
	kclient := new(adapter.Mock)
	logger := mock.Logger{}
	h := helper.Mock{}
	chainMock := chain.Mock{}
	chainMock.On("Serve", &kc).Return(nil)
	realm := keycloakApi.KeycloakRealm{}
	h.On("GetOrCreateRealmOwnerRef", &clientRealmFinder{parent: &kc,
		client: client}, kc.ObjectMeta).Return(&realm, nil)
	h.On("CreateKeycloakClientForRealm", &realm).Return(kclient, nil)
	h.On("TryToDelete", &kc,
		makeTerminator(kc.Status.ClientID, kc.Spec.TargetRealm, kclient, &logger),
		keyCloakClientOperatorFinalizerName).Return(true, nil)
	h.On("UpdateStatus", &kc).Return(nil)
	r := ReconcileKeycloakClient{
		client:                  client,
		helper:                  &h,
		log:                     &logger,
		chain:                   &chainMock,
		successReconcileTimeout: time.Hour,
	}

	req := reconcile.Request{NamespacedName: types.NamespacedName{Name: "main", Namespace: "namespace"}}
	res, err := r.Reconcile(context.TODO(), req)
	if err != nil {
		t.Fatal(err)
	}

	if res.RequeueAfter != r.successReconcileTimeout {
		t.Fatal("success reconcile timeout is not set")
	}
}

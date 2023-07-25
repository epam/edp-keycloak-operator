package chain

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	testifyMock "github.com/stretchr/testify/mock"
	v1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	keycloakApi "github.com/epam/edp-keycloak-operator/api/v1"
	"github.com/epam/edp-keycloak-operator/pkg/client/keycloak/adapter"
	"github.com/epam/edp-keycloak-operator/pkg/client/keycloak/mock"
)

func TestPutClient_Serve(t *testing.T) {
	logger := mock.NewLogr()

	kc := keycloakApi.KeycloakClient{ObjectMeta: metav1.ObjectMeta{Name: "main", Namespace: "namespace"},
		Spec: keycloakApi.KeycloakClientSpec{TargetRealm: "namespace.main",
			RealmRoles: &[]keycloakApi.RealmRole{{Name: "fake-client-administrators", Composite: "administrator"},
				{Name: "fake-client-users", Composite: "developer"},
			}, Public: false, ClientId: "fake-client", WebUrl: "fake-url", DirectAccess: false,
			AdvancedProtocolMappers: true, ClientRoles: nil, ProtocolMappers: &[]keycloakApi.ProtocolMapper{
				{Name: "bar", Config: map[string]string{"bar": "1"}},
				{Name: "foo", Config: map[string]string{"foo": "2"}},
			},
		},
	}

	s := scheme.Scheme
	s.AddKnownTypes(v1.SchemeGroupVersion,
		&kc)

	client := fake.NewClientBuilder().WithScheme(s).WithRuntimeObjects(&kc).Build()

	pc := PutClient{
		BaseElement: BaseElement{
			Logger: logger,
			Client: client,
			scheme: s,
		},
	}
	kClient := new(adapter.Mock)

	realmName := fmt.Sprintf("%s.%s", kc.Namespace, kc.Name)

	kClient.On("UpdateClient", testifyMock.Anything).Return(nil).Once()
	kClient.On("GetClientID", kc.Spec.ClientId, realmName).Return("id1", nil).Once()

	err := pc.Serve(context.Background(), &kc, kClient, realmName)

	assert.NoError(t, err)

	kClient.On("GetClientID", kc.Spec.ClientId, realmName).Return("", adapter.NotFoundError("1")).Once()
	kClient.On("GetClientID", kc.Spec.ClientId, realmName).Return("1", nil).Once()
	kClient.On("CreateClient", testifyMock.Anything).Return(nil)

	err = pc.Serve(context.Background(), &kc, kClient, realmName)
	assert.NoError(t, err)

	kClient.AssertExpectations(t)
}

func TestPutClient_Serve_FailureToUpdateClient(t *testing.T) {
	logger := mock.NewLogr()

	kc := keycloakApi.KeycloakClient{ObjectMeta: metav1.ObjectMeta{Name: "main", Namespace: "namespace"},
		Spec: keycloakApi.KeycloakClientSpec{TargetRealm: "namespace.main",
			RealmRoles: &[]keycloakApi.RealmRole{{Name: "fake-client-administrators", Composite: "administrator"},
				{Name: "fake-client-users", Composite: "developer"},
			}, Public: false, ClientId: "fake-client", WebUrl: "fake-url", DirectAccess: false,
			AdvancedProtocolMappers: true, ClientRoles: nil, ProtocolMappers: &[]keycloakApi.ProtocolMapper{
				{Name: "bar", Config: map[string]string{"bar": "1"}},
				{Name: "foo", Config: map[string]string{"foo": "2"}},
			},
		},
	}

	s := scheme.Scheme
	s.AddKnownTypes(v1.SchemeGroupVersion,
		&kc)

	client := fake.NewClientBuilder().WithScheme(s).WithRuntimeObjects(&kc).Build()

	pc := PutClient{
		BaseElement: BaseElement{
			Logger: logger,
			Client: client,
			scheme: s,
		},
	}
	kClient := new(adapter.Mock)

	realmName := fmt.Sprintf("%s.%s", kc.Namespace, kc.Name)

	updateErr := errors.New("update-err")

	kClient.On("GetClientID", kc.Spec.ClientId, realmName).Return("id1", nil).Once()
	kClient.On("UpdateClient", testifyMock.Anything).Return(updateErr).Once()

	err := pc.Serve(context.Background(), &kc, kClient, realmName)
	assert.ErrorIs(t, err, updateErr)

	kClient.AssertExpectations(t)

	kc.Spec.Secret = "sec"
	kc.Spec.Public = false

	pc.BaseElement.Client = fake.NewClientBuilder().WithScheme(s).WithRuntimeObjects(&kc).Build()
	err = pc.Serve(context.Background(), &kc, kClient, realmName)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "secrets \"sec\" not found")
}

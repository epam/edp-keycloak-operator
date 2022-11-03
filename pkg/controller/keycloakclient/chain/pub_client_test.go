package chain

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/epam/edp-keycloak-operator/pkg/apis/v1/v1alpha1"
	"github.com/epam/edp-keycloak-operator/pkg/client/keycloak/adapter"
	"github.com/epam/edp-keycloak-operator/pkg/client/keycloak/mock"
	"github.com/stretchr/testify/assert"
	testifyMock "github.com/stretchr/testify/mock"
	v1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func TestPutClient_Serve(t *testing.T) {
	logger := mock.Logger{}

	kc := v1alpha1.KeycloakClient{ObjectMeta: metav1.ObjectMeta{Name: "main", Namespace: "namespace"},
		Spec: v1alpha1.KeycloakClientSpec{TargetRealm: "namespace.main",
			RealmRoles: &[]v1alpha1.RealmRole{{Name: "fake-client-administrators", Composite: "administrator"},
				{Name: "fake-client-users", Composite: "developer"},
			}, Public: false, ClientId: "fake-client", WebUrl: "fake-url", DirectAccess: false,
			AdvancedProtocolMappers: true, ClientRoles: nil, ProtocolMappers: &[]v1alpha1.ProtocolMapper{
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
			Logger: &logger,
			Client: client,
			scheme: s,
		},
	}
	kClient := new(adapter.Mock)

	realmName := fmt.Sprintf("%s.%s", kc.Namespace, kc.Name)

	kClient.On("UpdateClient", testifyMock.Anything).Return(nil).Once()
	kClient.On("GetClientID", kc.Spec.ClientId, realmName).Return("id1", nil).Once()

	err := pc.Serve(context.Background(), &kc, kClient)
	assert.NoError(t, err)

	kClient.On("GetClientID", kc.Spec.ClientId, realmName).Return("", adapter.ErrNotFound("1")).Once()
	kClient.On("GetClientID", kc.Spec.ClientId, realmName).Return("1", nil).Once()
	kClient.On("CreateClient", testifyMock.Anything).Return(nil)

	err = pc.Serve(context.Background(), &kc, kClient)
	assert.NoError(t, err)

	kClient.AssertExpectations(t)
}

func TestPutClient_Serve_Failure(t *testing.T) {
	logger := mock.Logger{}

	kc := v1alpha1.KeycloakClient{ObjectMeta: metav1.ObjectMeta{Name: "main", Namespace: "namespace"},
		Spec: v1alpha1.KeycloakClientSpec{TargetRealm: "namespace.main",
			RealmRoles: &[]v1alpha1.RealmRole{{Name: "fake-client-administrators", Composite: "administrator"},
				{Name: "fake-client-users", Composite: "developer"},
			}, Public: false, ClientId: "fake-client", WebUrl: "fake-url", DirectAccess: false,
			AdvancedProtocolMappers: true, ClientRoles: nil, ProtocolMappers: &[]v1alpha1.ProtocolMapper{
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
			Logger: &logger,
			Client: client,
			scheme: s,
		},
	}
	kClient := new(adapter.Mock)

	realmName := fmt.Sprintf("%s.%s", kc.Namespace, kc.Name)

	updateErr := errors.New("update-err")
	kClient.On("GetClientID", kc.Spec.ClientId, realmName).Return("id1", nil).Once()
	kClient.On("UpdateClient", testifyMock.Anything).Return(updateErr).Once()

	err := pc.Serve(context.Background(), &kc, kClient)
	assert.True(t, errors.Is(err, updateErr))

	kClient.AssertExpectations(t)

	kc.Spec.Secret = "sec"
	kc.Spec.Public = false

	pc.BaseElement.Client = fake.NewClientBuilder().WithScheme(s).WithRuntimeObjects(&kc).Build()
	err = pc.Serve(context.Background(), &kc, kClient)
	assert.Contains(t, err.Error(), "secrets \"sec\" not found")
}

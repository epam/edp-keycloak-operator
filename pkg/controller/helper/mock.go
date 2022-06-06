package helper

import (
	"context"
	"sync"
	"time"

	"github.com/stretchr/testify/mock"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	keycloakApi "github.com/epam/edp-keycloak-operator/pkg/apis/v1/v1"
	"github.com/epam/edp-keycloak-operator/pkg/client/keycloak"
)

type Mock struct {
	mock.Mock
	tokenSecretLock sync.Mutex
}

func (m *Mock) TryToDelete(_ context.Context, obj Deletable, terminator Terminator, finalizer string) (isDeleted bool, resultErr error) {
	called := m.Called(obj, terminator, finalizer)
	if err := called.Error(1); err != nil {
		return false, err
	}

	return called.Bool(0), nil
}

func (m *Mock) SetFailureCount(fc FailureCountable) time.Duration {
	return m.Called(fc).Get(0).(time.Duration)
}

func (m *Mock) UpdateStatus(obj client.Object) error {
	return m.Called(obj).Error(0)
}

func (m *Mock) CreateKeycloakClientForRealm(ctx context.Context, realm *keycloakApi.KeycloakRealm) (keycloak.Client, error) {
	called := m.Called(realm)
	if err := called.Error(1); err != nil {
		return nil, err
	}

	return called.Get(0).(keycloak.Client), nil
}

func (m *Mock) CreateKeycloakClient(ctx context.Context, url, user, password string) (keycloak.Client, error) {
	called := m.Called(url, user, password)
	if err := called.Error(1); err != nil {
		return nil, err
	}

	return called.Get(0).(keycloak.Client), nil
}

func (m *Mock) GetOrCreateRealmOwnerRef(object RealmChild, objectMeta v1.ObjectMeta) (*keycloakApi.KeycloakRealm, error) {
	called := m.Called(object, objectMeta)
	if err := called.Error(1); err != nil {
		return nil, err
	}

	return called.Get(0).(*keycloakApi.KeycloakRealm), nil
}

func (m *Mock) GetScheme() *runtime.Scheme {
	return m.Called().Get(0).(*runtime.Scheme)
}

func (m *Mock) IsOwner(slave client.Object, master client.Object) bool {
	return m.Called(slave, master).Bool(0)
}

func (m *Mock) CreateKeycloakClientFromTokenSecret(ctx context.Context, kc *keycloakApi.Keycloak) (keycloak.Client, error) {
	called := m.Called(kc)
	if err := called.Error(1); err != nil {
		return nil, err
	}

	return called.Get(0).(keycloak.Client), nil
}

func (m *Mock) CreateKeycloakClientFromLoginPassword(ctx context.Context, kc *keycloakApi.Keycloak) (keycloak.Client, error) {
	called := m.Called(kc)
	if err := called.Error(1); err != nil {
		return nil, err
	}

	return called.Get(0).(keycloak.Client), nil
}

func (m *Mock) InvalidateKeycloakClientTokenSecret(ctx context.Context, namespace, rootKeycloakName string) error {
	return m.Called(namespace, rootKeycloakName).Error(0)
}

func (m *Mock) TokenSecretLock() *sync.Mutex {
	return &m.tokenSecretLock
}

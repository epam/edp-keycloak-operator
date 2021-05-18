package helper

import (
	"time"

	"github.com/epam/edp-keycloak-operator/pkg/client/keycloak"

	"github.com/epam/edp-keycloak-operator/pkg/apis/v1/v1alpha1"
	"github.com/go-logr/logr"
	"github.com/stretchr/testify/mock"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type Mock struct {
	mock.Mock
}

func (m *Mock) TryToDelete(obj Deletable, terminator Terminator, finalizer string) (isDeleted bool, resultErr error) {
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

func (m *Mock) CreateKeycloakClientForRealm(realm *v1alpha1.KeycloakRealm, log logr.Logger) (keycloak.Client, error) {
	called := m.Called(realm, log)
	if err := called.Error(1); err != nil {
		return nil, err
	}

	return called.Get(0).(keycloak.Client), nil
}

func (m *Mock) CreateKeycloakClient(url, user, password string, log logr.Logger) (keycloak.Client, error) {
	called := m.Called(url, user, password, log)
	if err := called.Error(1); err != nil {
		return nil, err
	}

	return called.Get(0).(keycloak.Client), nil
}

func (m *Mock) GetOrCreateRealmOwnerRef(object RealmChild, objectMeta v1.ObjectMeta) (*v1alpha1.KeycloakRealm, error) {
	called := m.Called(object, objectMeta)
	if err := called.Error(1); err != nil {
		return nil, err
	}

	return called.Get(0).(*v1alpha1.KeycloakRealm), nil
}

func (m *Mock) GetScheme() *runtime.Scheme {
	return m.Called().Get(0).(*runtime.Scheme)
}

func (m *Mock) IsOwner(slave client.Object, master client.Object) bool {
	return m.Called(slave, master).Bool(0)
}

package helper

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/scheme"

	keycloakApi "github.com/epam/edp-keycloak-operator/api/v1"
	"github.com/epam/edp-keycloak-operator/pkg/client/keycloak/adapter"
)

func TestMock_CreateKeycloakClient(t *testing.T) {
	m := Mock{}
	m.On("CreateKeycloakClient", "", "", "").Return(&adapter.Mock{}, nil).Once()
	_, err := m.CreateKeycloakClient(context.Background(), "", "", "")
	require.NoError(t, err)

	m.On("CreateKeycloakClient", "", "", "").Return(nil, errors.New("fatal")).Once()
	if _, err := m.CreateKeycloakClient(context.Background(), "", "", ""); err == nil {
		t.Fatal("no error returned")
	}
}

func TestMock_CreateKeycloakClientForRealm(t *testing.T) {
	m := Mock{}
	r := keycloakApi.KeycloakRealm{}
	m.On("CreateKeycloakClientForRealm", &r).Return(&adapter.Mock{}, nil).Once()
	_, err := m.CreateKeycloakClientForRealm(context.Background(), &r)
	require.NoError(t, err)

	m.On("CreateKeycloakClientForRealm", &r).Return(nil, errors.New("fatal"))
	_, err = m.CreateKeycloakClientForRealm(context.Background(), &r)
	require.Error(t, err)
}

func TestMock_CreateKeycloakClientFromLoginPassword(t *testing.T) {
	m := Mock{}
	kc := keycloakApi.Keycloak{}
	m.On("CreateKeycloakClientFromLoginPassword", &kc).Return(&adapter.Mock{}, nil).Once()
	_, err := m.CreateKeycloakClientFromLoginPassword(context.Background(), &kc)
	require.NoError(t, err)

	m.On("CreateKeycloakClientFromLoginPassword", &kc).Return(nil, errors.New("fatal")).Once()
	_, err = m.CreateKeycloakClientFromLoginPassword(context.Background(), &kc)
	require.Error(t, err)
}

func TestMock_CreateKeycloakClientFromTokenSecret(t *testing.T) {
	m := Mock{}
	kc := keycloakApi.Keycloak{}
	m.On("CreateKeycloakClientFromTokenSecret", &kc).Return(&adapter.Mock{}, nil).Once()
	_, err := m.CreateKeycloakClientFromTokenSecret(context.Background(), &kc)
	require.NoError(t, err)

	m.On("CreateKeycloakClientFromTokenSecret", &kc).Return(nil, errors.New("fatal")).Once()
	_, err = m.CreateKeycloakClientFromTokenSecret(context.Background(), &kc)
	require.Error(t, err)
}

func TestMock_GetOrCreateRealmOwnerRef(t *testing.T) {
	m := Mock{}
	meta := &metav1.ObjectMeta{}
	r := keycloakApi.KeycloakRealm{}
	m.On("GetOrCreateRealmOwnerRef", nil, meta).Return(&r, nil).Once()
	_, err := m.GetOrCreateRealmOwnerRef(nil, meta)
	require.NoError(t, err)

	m.On("GetOrCreateRealmOwnerRef", nil, meta).Return(nil, errors.New("fatal")).Once()
	_, err = m.GetOrCreateRealmOwnerRef(nil, meta)
	require.Error(t, err)
}

func TestMock_TryToDelete(t *testing.T) {
	m := Mock{}
	m.On("TryToDelete", nil, nil, "!").Return(false, errors.New("fatal")).Once()
	_, err := m.TryToDelete(context.Background(), nil, nil, "!")
	require.Error(t, err)

	m.On("TryToDelete", nil, nil, "!").Return(false, nil).Once()
	_, err = m.TryToDelete(context.Background(), nil, nil, "!")
	require.NoError(t, err)
}

func TestMock_OneLiners(t *testing.T) {
	m := Mock{}
	m.On("SetFailureCount", nil).Return(time.Second)
	if m.SetFailureCount(nil) != time.Second {
		t.Fatal("wrong duration returned")
	}

	m.On("UpdateStatus", nil).Return(nil)
	err := m.UpdateStatus(nil)
	require.NoError(t, err)

	sch := scheme.Scheme
	m.On("GetScheme").Return(sch)
	if m.GetScheme() != sch {
		t.Fatal("wrong scheme")
	}

	m.On("IsOwner", nil, nil).Return(false)
	if m.IsOwner(nil, nil) {
		t.Fatal("wrong owner")
	}
}

func TestMock_TokenSecretLock(t *testing.T) {
	m := Mock{}
	if m.TokenSecretLock() != &m.tokenSecretLock {
		t.Fatal("wrong token secret lock")
	}
}

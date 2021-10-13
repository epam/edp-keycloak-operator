package helper

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/epam/edp-keycloak-operator/pkg/apis/v1/v1alpha1"
	"github.com/epam/edp-keycloak-operator/pkg/client/keycloak/adapter"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/scheme"
)

func TestMock_CreateKeycloakClient(t *testing.T) {
	m := Mock{}
	m.On("CreateKeycloakClient", "", "", "").Return(&adapter.Mock{}, nil).Once()
	if _, err := m.CreateKeycloakClient(context.Background(), "", "", ""); err != nil {
		t.Fatal(err)
	}

	m.On("CreateKeycloakClient", "", "", "").Return(nil, errors.New("fatal")).Once()
	if _, err := m.CreateKeycloakClient(context.Background(), "", "", ""); err == nil {
		t.Fatal("no error returned")
	}
}

func TestMock_CreateKeycloakClientForRealm(t *testing.T) {
	m := Mock{}
	r := v1alpha1.KeycloakRealm{}
	m.On("CreateKeycloakClientForRealm", &r).Return(&adapter.Mock{}, nil).Once()
	if _, err := m.CreateKeycloakClientForRealm(context.Background(), &r); err != nil {
		t.Fatal(err)
	}

	m.On("CreateKeycloakClientForRealm", &r).Return(nil, errors.New("fatal"))
	if _, err := m.CreateKeycloakClientForRealm(context.Background(), &r); err == nil {
		t.Fatal("no error returned")
	}
}

func TestMock_CreateKeycloakClientFromLoginPassword(t *testing.T) {
	m := Mock{}
	kc := v1alpha1.Keycloak{}
	m.On("CreateKeycloakClientFromLoginPassword", &kc).Return(&adapter.Mock{}, nil).Once()
	if _, err := m.CreateKeycloakClientFromLoginPassword(context.Background(), &kc); err != nil {
		t.Fatal(err)
	}

	m.On("CreateKeycloakClientFromLoginPassword", &kc).Return(nil, errors.New("fatal")).Once()
	if _, err := m.CreateKeycloakClientFromLoginPassword(context.Background(), &kc); err == nil {
		t.Fatal("no error returned")
	}
}

func TestMock_CreateKeycloakClientFromTokenSecret(t *testing.T) {
	m := Mock{}
	kc := v1alpha1.Keycloak{}
	m.On("CreateKeycloakClientFromTokenSecret", &kc).Return(&adapter.Mock{}, nil).Once()
	if _, err := m.CreateKeycloakClientFromTokenSecret(context.Background(), &kc); err != nil {
		t.Fatal(err)
	}

	m.On("CreateKeycloakClientFromTokenSecret", &kc).Return(nil, errors.New("fatal")).Once()
	if _, err := m.CreateKeycloakClientFromTokenSecret(context.Background(), &kc); err == nil {
		t.Fatal("no error returned")
	}
}

func TestMock_GetOrCreateRealmOwnerRef(t *testing.T) {
	m := Mock{}
	meta := metav1.ObjectMeta{}
	r := v1alpha1.KeycloakRealm{}
	m.On("GetOrCreateRealmOwnerRef", nil, meta).Return(&r, nil).Once()
	if _, err := m.GetOrCreateRealmOwnerRef(nil, meta); err != nil {
		t.Fatal(err)
	}

	m.On("GetOrCreateRealmOwnerRef", nil, meta).Return(nil, errors.New("fatal")).Once()
	if _, err := m.GetOrCreateRealmOwnerRef(nil, meta); err == nil {
		t.Fatal("no error returned")
	}

}

func TestMock_TryToDelete(t *testing.T) {
	m := Mock{}
	m.On("TryToDelete", nil, nil, "!").Return(false, errors.New("fatal")).Once()
	if _, err := m.TryToDelete(context.Background(), nil, nil, "!"); err == nil {
		t.Fatal("no error")
	}

	m.On("TryToDelete", nil, nil, "!").Return(false, nil).Once()
	if _, err := m.TryToDelete(context.Background(), nil, nil, "!"); err != nil {
		t.Fatal(err)
	}
}

func TestMock_OneLiners(t *testing.T) {
	m := Mock{}
	m.On("SetFailureCount", nil).Return(time.Second)
	if m.SetFailureCount(nil) != time.Second {
		t.Fatal("wrong duration returned")
	}

	m.On("UpdateStatus", nil).Return(nil)
	if err := m.UpdateStatus(nil); err != nil {
		t.Fatal(err)
	}

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

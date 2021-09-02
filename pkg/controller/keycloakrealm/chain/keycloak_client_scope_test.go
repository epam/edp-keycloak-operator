package chain

import (
	"strings"
	"testing"

	"github.com/epam/edp-keycloak-operator/pkg/apis/v1/v1alpha1"
	"github.com/epam/edp-keycloak-operator/pkg/client/keycloak/adapter"
	"github.com/pkg/errors"
	v1 "k8s.io/api/apps/v1"
	"k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func TestPutKeycloakClientScope_ServeRequest(t *testing.T) {
	s := scheme.Scheme
	realm := v1alpha1.KeycloakRealm{
		Spec: v1alpha1.KeycloakRealmSpec{RealmName: "realm1"},
	}
	s.AddKnownTypes(v1.SchemeGroupVersion, &realm)
	client := fake.NewClientBuilder().WithRuntimeObjects(&realm).Build()

	kClient := adapter.Mock{}
	ps := PutKeycloakClientScope{
		client: client,
	}

	kClient.On("GetClientScope", "edp", "realm1").
		Return(nil, adapter.ErrNotFound("not found"))
	kClient.On("CreateClientScope", "realm1", getDefClientScope()).
		Return("scope_id", nil)
	if err := ps.ServeRequest(&realm, &kClient); err != nil {
		t.Fatal(err)
	}
}

func TestPutKeycloakClientScope_ServeRequest_Failure_GetClientScope(t *testing.T) {
	s := scheme.Scheme
	realm := v1alpha1.KeycloakRealm{
		Spec: v1alpha1.KeycloakRealmSpec{RealmName: "realm1"},
	}
	s.AddKnownTypes(v1.SchemeGroupVersion, &realm)
	client := fake.NewClientBuilder().WithRuntimeObjects(&realm).Build()

	kClient := adapter.Mock{}
	ps := PutKeycloakClientScope{
		client: client,
	}

	kClient.On("GetClientScope", "edp", "realm1").
		Return(nil, errors.New("get scope fatal"))

	err := ps.ServeRequest(&realm, &kClient)
	if err == nil {
		t.Fatal("no error returned")
	}

	if !strings.Contains(err.Error(), "get scope fatal") {
		t.Fatalf("wrong error returned: %s", err.Error())
	}
}

func TestPutKeycloakClientScope_ServeRequest_FailureCreateClientScope(t *testing.T) {
	s := scheme.Scheme
	realm := v1alpha1.KeycloakRealm{
		Spec: v1alpha1.KeycloakRealmSpec{RealmName: "realm1"},
	}
	s.AddKnownTypes(v1.SchemeGroupVersion, &realm)
	client := fake.NewClientBuilder().WithRuntimeObjects(&realm).Build()

	kClient := adapter.Mock{}
	ps := PutKeycloakClientScope{
		client: client,
	}

	kClient.On("GetClientScope", "edp", "realm1").
		Return(nil, adapter.ErrNotFound("not found"))
	kClient.On("CreateClientScope", "realm1", getDefClientScope()).
		Return("", errors.New("create client scope fatal"))
	err := ps.ServeRequest(&realm, &kClient)
	if err == nil {
		t.Fatal("no error returned")
	}

	if !strings.Contains(err.Error(), "create client scope fatal") {
		t.Fatalf("wrong error returned: %s", err.Error())
	}
}

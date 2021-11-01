package chain

import (
	"context"
	"testing"

	"github.com/pkg/errors"

	"github.com/Nerzal/gocloak/v8"
	"github.com/epam/edp-keycloak-operator/pkg/apis/v1/v1alpha1"
	"github.com/epam/edp-keycloak-operator/pkg/client/keycloak/adapter"
)

func TestAuthFlow_ServeRequest(t *testing.T) {
	kc := adapter.Mock{}
	af := AuthFlow{}

	realm := v1alpha1.KeycloakRealm{
		Spec: v1alpha1.KeycloakRealmSpec{
			RealmName: "realm1",
		},
	}

	ctx := context.Background()

	if err := af.ServeRequest(ctx, &realm, &kc); err != nil {
		t.Fatal(err)
	}

	kc.On("SetRealmBrowserFlow", "realm1", "flow-alias-1").Return(nil)
	realm.Spec.BrowserFlow = gocloak.StringP("flow-alias-1")
	if err := af.ServeRequest(ctx, &realm, &kc); err != nil {
		t.Fatal(err)
	}
}

func TestAuthFlow_ServeRequest_Failure(t *testing.T) {
	kc := adapter.Mock{}
	af := AuthFlow{}

	realm := v1alpha1.KeycloakRealm{
		Spec: v1alpha1.KeycloakRealmSpec{
			RealmName: "realm1",
		},
	}

	mockErr := errors.New("fatal")

	kc.On("SetRealmBrowserFlow", "realm1", "flow-alias-1").Return(mockErr)
	realm.Spec.BrowserFlow = gocloak.StringP("flow-alias-1")
	err := af.ServeRequest(context.Background(), &realm, &kc)
	if err == nil {
		t.Fatal("no error on mock fatal")
	}

	if errors.Cause(err) != mockErr {
		t.Log(err)
		t.Fatal("wrong error returned")
	}
}

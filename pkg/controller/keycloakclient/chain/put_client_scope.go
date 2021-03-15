package chain

import (
	v1v1alpha1 "github.com/epmd-edp/keycloak-operator/pkg/apis/v1/v1alpha1"
	"github.com/epmd-edp/keycloak-operator/pkg/consts"
	"github.com/pkg/errors"
)

type PutClientScope struct {
	BaseElement
	next Element
}

func (el *PutClientScope) Serve(keycloakClient *v1v1alpha1.KeycloakClient) error {
	if err := el.putClientScope(keycloakClient); err != nil {
		return errors.Wrap(err, "error during putClientScope")
	}

	return el.NextServeOrNil(el.next, keycloakClient)
}

func (el *PutClientScope) putClientScope(keycloakClient *v1v1alpha1.KeycloakClient) error {
	if !keycloakClient.Spec.AudRequired {
		return nil
	}

	realmName := el.State.KeycloakRealm.Spec.RealmName

	scope, err := el.State.AdapterClient.GetClientScope(consts.DefaultClientScopeName, realmName)
	if err != nil {
		return errors.Wrap(err, "error during GetClientScope")
	}

	if err := el.State.AdapterClient.PutClientScopeMapper(
		keycloakClient.Spec.ClientId, *scope.ID, realmName); err != nil {
		return errors.Wrap(err, "error during PutClientScopeMapper")
	}

	if err := el.State.AdapterClient.LinkClientScopeToClient(
		keycloakClient.Spec.ClientId, *scope.ID, realmName); err != nil {
		return errors.Wrap(err, "error during LinkClientScopeToClient")
	}

	return nil
}

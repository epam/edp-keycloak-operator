package chain

import (
	v1v1alpha1 "github.com/epam/edp-keycloak-operator/pkg/apis/v1/v1alpha1"
	"github.com/epam/edp-keycloak-operator/pkg/client/keycloak"
	"github.com/epam/edp-keycloak-operator/pkg/consts"
	"github.com/pkg/errors"
)

type PutClientScope struct {
	BaseElement
	next Element
}

func (el *PutClientScope) Serve(keycloakClient *v1v1alpha1.KeycloakClient, adapterClient keycloak.Client) error {
	if err := el.putClientScope(keycloakClient, adapterClient); err != nil {
		return errors.Wrap(err, "error during putClientScope")
	}

	return el.NextServeOrNil(el.next, keycloakClient, adapterClient)
}

func (el *PutClientScope) putClientScope(keycloakClient *v1v1alpha1.KeycloakClient, adapterClient keycloak.Client) error {
	if !keycloakClient.Spec.AudRequired {
		return nil
	}

	scope, err := adapterClient.GetClientScope(consts.DefaultClientScopeName, keycloakClient.Spec.TargetRealm)
	if err != nil {
		return errors.Wrap(err, "error during GetClientScope")
	}

	if err := adapterClient.PutClientScopeMapper(
		keycloakClient.Spec.ClientId, *scope.ID, keycloakClient.Spec.TargetRealm); err != nil {
		return errors.Wrap(err, "error during PutClientScopeMapper")
	}

	if err := adapterClient.LinkClientScopeToClient(
		keycloakClient.Spec.ClientId, *scope.ID, keycloakClient.Spec.TargetRealm); err != nil {
		return errors.Wrap(err, "error during LinkClientScopeToClient")
	}

	return nil
}

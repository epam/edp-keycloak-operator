package chain

import (
	v1v1alpha1 "github.com/epmd-edp/keycloak-operator/pkg/apis/v1/v1alpha1"
	"github.com/epmd-edp/keycloak-operator/pkg/client/keycloak"
	"github.com/pkg/errors"
)

type CreateAdapter struct {
	BaseElement
	factory keycloak.ClientFactory
	next    Element
}

func (el *CreateAdapter) Serve(keycloakClient *v1v1alpha1.KeycloakClient) error {
	adapterClient, err := el.Helper.CreateKeycloakClient(el.State.KeycloakRealm, el.factory)
	if err != nil {
		return errors.Wrap(err, "unable to create keycloak adapter")
	}

	el.State.AdapterClient = adapterClient

	return el.NextServeOrNil(el.next, keycloakClient)
}

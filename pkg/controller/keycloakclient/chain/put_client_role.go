package chain

import (
	v1v1alpha1 "github.com/epmd-edp/keycloak-operator/pkg/apis/v1/v1alpha1"
	"github.com/epmd-edp/keycloak-operator/pkg/client/keycloak/dto"
	"github.com/pkg/errors"
)

type PutClientRole struct {
	BaseElement
	next Element
}

func (el *PutClientRole) Serve(keycloakClient *v1v1alpha1.KeycloakClient) error {
	if err := el.putKeycloakClientRole(keycloakClient); err != nil {
		return errors.Wrap(err, "unable to put keycloak client role")
	}

	return el.NextServeOrNil(el.next, keycloakClient)
}

func (el *PutClientRole) putKeycloakClientRole(keycloakClient *v1v1alpha1.KeycloakClient) error {
	reqLog := el.Logger.WithValues("keycloak client cr", keycloakClient)
	reqLog.Info("Start put keycloak client role...")

	clientDto := dto.ConvertSpecToClient(&keycloakClient.Spec, "")

	for _, role := range clientDto.Roles {
		exist, err := el.State.AdapterClient.ExistClientRole(clientDto, role)
		if err != nil {
			return errors.Wrap(err, "error during ExistClientRole")
		}

		if exist {
			reqLog.Info("Client role already exists", "role", role)
			continue
		}

		if err := el.State.AdapterClient.CreateClientRole(clientDto, role); err != nil {
			return errors.Wrap(err, "unable to create client role")
		}
	}

	reqLog.Info("End put keycloak client role")
	return nil
}

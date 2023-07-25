package chain

import (
	"context"

	"github.com/pkg/errors"

	keycloakApi "github.com/epam/edp-keycloak-operator/api/v1"
	"github.com/epam/edp-keycloak-operator/pkg/client/keycloak"
	"github.com/epam/edp-keycloak-operator/pkg/client/keycloak/dto"
)

type PutClientRole struct {
	BaseElement
	next Element
}

func (el *PutClientRole) Serve(ctx context.Context, keycloakClient *keycloakApi.KeycloakClient, adapterClient keycloak.Client, realmName string) error {
	if err := el.putKeycloakClientRole(keycloakClient, adapterClient, realmName); err != nil {
		return errors.Wrap(err, "unable to put keycloak client role")
	}

	return el.NextServeOrNil(ctx, el.next, keycloakClient, adapterClient, realmName)
}

func (el *PutClientRole) putKeycloakClientRole(keycloakClient *keycloakApi.KeycloakClient, adapterClient keycloak.Client, realmName string) error {
	reqLog := el.Logger.WithValues("keycloak client cr", keycloakClient)
	reqLog.Info("Start put keycloak client role...")

	clientDto := dto.ConvertSpecToClient(&keycloakClient.Spec, "", realmName)

	for _, role := range clientDto.Roles {
		exist, err := adapterClient.ExistClientRole(clientDto, role)
		if err != nil {
			return errors.Wrap(err, "error during ExistClientRole")
		}

		if exist {
			reqLog.Info("Client role already exists", "role", role)
			continue
		}

		if err := adapterClient.CreateClientRole(clientDto, role); err != nil {
			return errors.Wrap(err, "unable to create client role")
		}
	}

	reqLog.Info("End put keycloak client role")

	return nil
}

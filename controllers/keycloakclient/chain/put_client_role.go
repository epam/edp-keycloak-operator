package chain

import (
	"context"

	"github.com/pkg/errors"
	ctrl "sigs.k8s.io/controller-runtime"

	keycloakApi "github.com/epam/edp-keycloak-operator/api/v1"
	"github.com/epam/edp-keycloak-operator/pkg/client/keycloak"
	"github.com/epam/edp-keycloak-operator/pkg/client/keycloak/dto"
)

type PutClientRole struct {
	keycloakApiClient keycloak.Client
}

func NewPutClientRole(keycloakApiClient keycloak.Client) *PutClientRole {
	return &PutClientRole{keycloakApiClient: keycloakApiClient}
}

func (el *PutClientRole) Serve(ctx context.Context, keycloakClient *keycloakApi.KeycloakClient, realmName string) error {
	if err := el.putKeycloakClientRole(ctx, keycloakClient, realmName); err != nil {
		return errors.Wrap(err, "unable to put keycloak client role")
	}

	return nil
}

func (el *PutClientRole) putKeycloakClientRole(ctx context.Context, keycloakClient *keycloakApi.KeycloakClient, realmName string) error {
	reqLog := ctrl.LoggerFrom(ctx)
	reqLog.Info("Start put keycloak client role")

	clientDto := dto.ConvertSpecToClient(&keycloakClient.Spec, "", realmName)

	for _, role := range clientDto.Roles {
		exist, err := el.keycloakApiClient.ExistClientRole(clientDto, role)
		if err != nil {
			return errors.Wrap(err, "error during ExistClientRole")
		}

		if exist {
			reqLog.Info("Client role already exists", "role", role)
			continue
		}

		if err := el.keycloakApiClient.CreateClientRole(clientDto, role); err != nil {
			return errors.Wrap(err, "unable to create client role")
		}
	}

	reqLog.Info("End put keycloak client role")

	return nil
}

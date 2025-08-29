package chain

import (
	"context"
	"fmt"

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
		return fmt.Errorf("unable to put keycloak client role: %w", err)
	}

	return nil
}

func (el *PutClientRole) putKeycloakClientRole(ctx context.Context, keycloakClient *keycloakApi.KeycloakClient, realmName string) error {
	reqLog := ctrl.LoggerFrom(ctx)
	reqLog.Info("Start put keycloak client role")

	clientDto := dto.ConvertSpecToClient(&keycloakClient.Spec, "", realmName, nil)

	if err := el.keycloakApiClient.SyncClientRoles(ctx, realmName, clientDto); err != nil {
		return fmt.Errorf("unable to sync client roles: %w", err)
	}

	reqLog.Info("End put keycloak client role")

	return nil
}

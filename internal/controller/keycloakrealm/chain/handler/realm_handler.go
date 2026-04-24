package handler

import (
	"context"

	keycloakApi "github.com/epam/edp-keycloak-operator/api/v1"
	"github.com/epam/edp-keycloak-operator/pkg/client/keycloakapi"
)

type RealmHandler interface {
	ServeRequest(ctx context.Context, realm *keycloakApi.KeycloakRealm, kClient *keycloakapi.KeycloakClient) error
}

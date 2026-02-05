package handler

import (
	"context"

	keycloakApi "github.com/epam/edp-keycloak-operator/api/v1"
	"github.com/epam/edp-keycloak-operator/pkg/client/keycloak"
	keycloakv2 "github.com/epam/edp-keycloak-operator/pkg/client/keycloakv2"
)

type RealmHandler interface {
	ServeRequest(ctx context.Context, realm *keycloakApi.KeycloakRealm, kClient keycloak.Client, kClientV2 *keycloakv2.KeycloakClient) error
}

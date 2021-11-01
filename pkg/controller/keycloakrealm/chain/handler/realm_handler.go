package handler

import (
	"context"

	"github.com/epam/edp-keycloak-operator/pkg/apis/v1/v1alpha1"
	"github.com/epam/edp-keycloak-operator/pkg/client/keycloak"
)

type RealmHandler interface {
	ServeRequest(ctx context.Context, realm *v1alpha1.KeycloakRealm, kClient keycloak.Client) error
}

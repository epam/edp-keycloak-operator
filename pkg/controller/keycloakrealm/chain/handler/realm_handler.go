package handler

import (
	"github.com/epam/keycloak-operator/v2/pkg/apis/v1/v1alpha1"
	"github.com/epam/keycloak-operator/v2/pkg/client/keycloak"
)

type RealmHandler interface {
	ServeRequest(realm *v1alpha1.KeycloakRealm, kClient keycloak.Client) error
}

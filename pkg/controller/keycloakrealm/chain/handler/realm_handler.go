package handler

import (
	"github.com/epam/edp-keycloak-operator/pkg/apis/v1/v1alpha1"
	"github.com/epam/edp-keycloak-operator/pkg/client/keycloak"
)

type RealmHandler interface {
	ServeRequest(realm *v1alpha1.KeycloakRealm, kClient keycloak.Client) error
}

package chain

import (
	"context"
	"fmt"

	ctrl "sigs.k8s.io/controller-runtime"

	keycloakApi "github.com/epam/edp-keycloak-operator/api/v1"
	"github.com/epam/edp-keycloak-operator/pkg/client/keycloak"
	"github.com/epam/edp-keycloak-operator/pkg/client/keycloak/dto"
)

type PutRealmRole struct {
	keycloakApiClient keycloak.Client
}

func NewPutRealmRole(keycloakApiClient keycloak.Client) *PutRealmRole {
	return &PutRealmRole{keycloakApiClient: keycloakApiClient}
}

func (el *PutRealmRole) Serve(ctx context.Context, keycloakClient *keycloakApi.KeycloakClient, realmName string) error {
	if err := el.putRealmRoles(ctx, keycloakClient, realmName); err != nil {
		return fmt.Errorf("unable to put realm roles: %w", err)
	}

	return nil
}

func (el *PutRealmRole) putRealmRoles(ctx context.Context, keycloakClient *keycloakApi.KeycloakClient, realmName string) error {
	reqLog := ctrl.LoggerFrom(ctx)
	reqLog.Info("Start put realm roles")

	if keycloakClient.Spec.RealmRoles == nil || len(*keycloakClient.Spec.RealmRoles) == 0 {
		reqLog.Info("Keycloak client does not have realm roles")
		return nil
	}

	for _, role := range *keycloakClient.Spec.RealmRoles {
		roleDto := &dto.IncludedRealmRole{
			Name:      role.Name,
			Composite: role.Composite,
		}

		exist, err := el.keycloakApiClient.ExistRealmRole(realmName, roleDto.Name)
		if err != nil {
			return fmt.Errorf("error during ExistRealmRole: %w", err)
		}

		if exist {
			reqLog.Info("Client already exists")
			return nil
		}

		err = el.keycloakApiClient.CreateIncludedRealmRole(realmName, roleDto)
		if err != nil {
			return fmt.Errorf("error during CreateRealmRole: %w", err)
		}
	}

	reqLog.Info("End put realm roles")

	return nil
}

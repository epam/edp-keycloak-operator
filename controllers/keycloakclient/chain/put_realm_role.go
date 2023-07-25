package chain

import (
	"context"

	"github.com/pkg/errors"

	keycloakApi "github.com/epam/edp-keycloak-operator/api/v1"
	"github.com/epam/edp-keycloak-operator/pkg/client/keycloak"
	"github.com/epam/edp-keycloak-operator/pkg/client/keycloak/dto"
)

type PutRealmRole struct {
	BaseElement
	next Element
}

func (el *PutRealmRole) Serve(ctx context.Context, keycloakClient *keycloakApi.KeycloakClient, adapterClient keycloak.Client, realmName string) error {
	if err := el.putRealmRoles(keycloakClient, adapterClient, realmName); err != nil {
		return errors.Wrap(err, "unable to put realm roles")
	}

	return el.NextServeOrNil(ctx, el.next, keycloakClient, adapterClient, realmName)
}

func (el *PutRealmRole) putRealmRoles(keycloakClient *keycloakApi.KeycloakClient, adapterClient keycloak.Client, realmName string) error {
	reqLog := el.Logger.WithValues("keycloak client cr", keycloakClient)
	reqLog.Info("Start put realm roles...")

	if keycloakClient.Spec.RealmRoles == nil || len(*keycloakClient.Spec.RealmRoles) == 0 {
		reqLog.Info("Keycloak client does not have realm roles")
		return nil
	}

	for _, role := range *keycloakClient.Spec.RealmRoles {
		roleDto := &dto.IncludedRealmRole{
			Name:      role.Name,
			Composite: role.Composite,
		}

		exist, err := adapterClient.ExistRealmRole(realmName, roleDto.Name)
		if err != nil {
			return errors.Wrap(err, "error during ExistRealmRole")
		}

		if exist {
			reqLog.Info("Client already exists")
			return nil
		}

		err = adapterClient.CreateIncludedRealmRole(realmName, roleDto)
		if err != nil {
			return errors.Wrap(err, "error during CreateRealmRole")
		}
	}

	reqLog.Info("End put realm roles")

	return nil
}

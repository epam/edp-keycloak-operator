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

func (el *PutRealmRole) Serve(ctx context.Context, keycloakClient *keycloakApi.KeycloakClient, adapterClient keycloak.Client) error {
	if err := el.putRealmRoles(keycloakClient, adapterClient); err != nil {
		return errors.Wrap(err, "unable to put realm roles")
	}

	return el.NextServeOrNil(ctx, el.next, keycloakClient, adapterClient)
}

func (el *PutRealmRole) putRealmRoles(keycloakClient *keycloakApi.KeycloakClient, adapterClient keycloak.Client) error {
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

		exist, err := adapterClient.ExistRealmRole(keycloakClient.Spec.TargetRealm, roleDto.Name)
		if err != nil {
			return errors.Wrap(err, "error during ExistRealmRole")
		}

		if exist {
			reqLog.Info("Client already exists")
			return nil
		}

		err = adapterClient.CreateIncludedRealmRole(keycloakClient.Spec.TargetRealm, roleDto)
		if err != nil {
			return errors.Wrap(err, "error during CreateRealmRole")
		}
	}

	reqLog.Info("End put realm roles")

	return nil
}

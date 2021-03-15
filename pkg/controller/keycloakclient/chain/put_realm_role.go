package chain

import (
	v1v1alpha1 "github.com/epmd-edp/keycloak-operator/pkg/apis/v1/v1alpha1"
	"github.com/epmd-edp/keycloak-operator/pkg/client/keycloak/dto"
	"github.com/pkg/errors"
)

type PutRealmRole struct {
	BaseElement
	next Element
}

func (el *PutRealmRole) Serve(keycloakClient *v1v1alpha1.KeycloakClient) error {
	if err := el.putRealmRoles(keycloakClient); err != nil {
		return errors.Wrap(err, "unable to put realm roles")
	}

	return el.NextServeOrNil(el.next, keycloakClient)
}

func (el *PutRealmRole) putRealmRoles(keycloakClient *v1v1alpha1.KeycloakClient) error {
	reqLog := el.Logger.WithValues("keycloak client cr", keycloakClient)
	reqLog.Info("Start put realm roles...")

	if keycloakClient.Spec.RealmRoles == nil || len(*keycloakClient.Spec.RealmRoles) == 0 {
		reqLog.Info("Keycloak client does not have realm roles")
		return nil
	}

	realmDto := dto.ConvertSpecToRealm(el.State.KeycloakRealm.Spec)

	for _, role := range *keycloakClient.Spec.RealmRoles {
		roleDto := &dto.RealmRole{
			Name:        role.Name,
			Composites:  []string{role.Composite},
			IsComposite: role.Composite != "",
		}
		exist, err := el.State.AdapterClient.ExistRealmRole(realmDto.Name, roleDto.Name)
		if err != nil {
			return errors.Wrap(err, "error during ExistRealmRole")
		}
		if exist {
			reqLog.Info("Client already exists")
			return nil
		}
		err = el.State.AdapterClient.CreateRealmRole(realmDto.Name, roleDto)
		if err != nil {
			return errors.Wrap(err, "error during CreateRealmRole")
		}
	}

	reqLog.Info("End put realm roles")
	return nil
}

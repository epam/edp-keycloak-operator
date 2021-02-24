package chain

import (
	"github.com/epmd-edp/keycloak-operator/pkg/apis/v1/v1alpha1"
	"github.com/epmd-edp/keycloak-operator/pkg/client/keycloak"
	"github.com/epmd-edp/keycloak-operator/pkg/client/keycloak/dto"
	"github.com/epmd-edp/keycloak-operator/pkg/controller/keycloakrealm/chain/handler"
	"github.com/pkg/errors"
)

type PutUsersRoles struct {
	next handler.RealmHandler
}

func (h PutUsersRoles) ServeRequest(realm *v1alpha1.KeycloakRealm, kClient keycloak.Client) error {
	rLog := log.WithValues("keycloak users", realm.Spec.Users)
	rLog.Info("Start putting roles to users")
	rDto := dto.ConvertSpecToRealm(realm.Spec)
	err := putRolesToUsers(rDto, kClient)
	if err != nil {
		return errors.Wrap(err, "error during putRolesToUsers")
	}
	rLog.Info("End put role to users")
	return nextServeOrNil(h.next, realm, kClient)
}

func putRolesToUsers(realm dto.Realm, kClient keycloak.Client) error {
	for _, user := range realm.Users {
		err := putRolesToOneUser(realm, user, kClient)
		if err != nil {
			return errors.Wrap(err, "error during putRolesToOneUser")
		}
	}
	return nil
}

func putRolesToOneUser(realm dto.Realm, user dto.User, kClient keycloak.Client) error {
	for _, role := range user.RealmRoles {
		err := putOneRoleToOneUser(realm, user, role, kClient)
		if err != nil {
			return errors.Wrap(err, "error during putOneRoleToOneUser")
		}
	}
	return nil
}

func putOneRoleToOneUser(realm dto.Realm, user dto.User, role string, kClient keycloak.Client) error {
	realmName := realm.Name
	if realm.SsoRealmEnabled {
		realmName = realm.SsoRealmName
	}

	exist, err := kClient.HasUserClientRole(realmName, realm.Name, user, role)
	if err != nil {
		return errors.Wrap(err, "error during check of client role")
	}
	if *exist {
		log.Info("Role already exists", "user", user, "role", role)
		return nil
	}

	if err := kClient.AddClientRoleToUser(realmName, realm.Name, user, role); err != nil {
		return errors.Wrap(err, "unable to add client role to user")
	}

	return nil
}

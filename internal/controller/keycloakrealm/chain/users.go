package chain

import (
	"context"

	"github.com/pkg/errors"

	keycloakApi "github.com/epam/edp-keycloak-operator/api/v1"
	"github.com/epam/edp-keycloak-operator/internal/controller/keycloakrealm/chain/handler"
	"github.com/epam/edp-keycloak-operator/pkg/client/keycloak"
	"github.com/epam/edp-keycloak-operator/pkg/client/keycloak/dto"
)

type PutUsers struct {
	next handler.RealmHandler
}

func (h PutUsers) ServeRequest(ctx context.Context, realm *keycloakApi.KeycloakRealm, kClient keycloak.Client) error {
	rLog := log.WithValues("keycloak users", realm.Spec.Users)
	rLog.Info("Start putting users to realm")

	rDto := dto.ConvertSpecToRealm(&realm.Spec)

	err := createUsers(rDto, kClient)
	if err != nil {
		return errors.Wrap(err, "error during createUsers")
	}

	rLog.Info("End put users to realm")

	return nextServeOrNil(ctx, h.next, realm, kClient)
}

func createUsers(realm *dto.Realm, kClient keycloak.Client) error {
	for _, user := range realm.Users {
		err := createOneUser(&user, realm, kClient)
		if err != nil {
			return errors.Wrap(err, "error during createOneUser")
		}
	}

	return nil
}

func createOneUser(user *dto.User, realm *dto.Realm, kClient keycloak.Client) error {
	realmName := realm.Name

	exist, err := kClient.ExistRealmUser(realmName, user)
	if err != nil {
		return errors.Wrap(err, "error during exist ream user check")
	}

	if exist {
		log.Info("User already exists", "user", user)
		return nil
	}

	if err := kClient.CreateRealmUser(realmName, user); err != nil {
		return errors.Wrap(err, "unable to create user in realm")
	}

	return nil
}

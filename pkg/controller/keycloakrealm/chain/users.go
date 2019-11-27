package chain

import (
	"github.com/epmd-edp/keycloak-operator/pkg/apis/v1/v1alpha1"
	"github.com/epmd-edp/keycloak-operator/pkg/client/keycloak"
	"github.com/epmd-edp/keycloak-operator/pkg/client/keycloak/dto"
	"github.com/epmd-edp/keycloak-operator/pkg/controller/keycloakrealm/chain/handler"
)

type PutUsers struct {
	next handler.RealmHandler
}

func (h PutUsers) ServeRequest(realm *v1alpha1.KeycloakRealm, kClient keycloak.Client) error {
	rLog := log.WithValues("keycloak users", realm.Spec.Users)
	rLog.Info("Start putting users to realm")
	rDto := dto.ConvertSpecToRealm(realm.Spec)
	err := createUsers(rDto, kClient)
	if err != nil {
		return err
	}
	rLog.Info("End put users to realm")
	return nextServeOrNil(h.next, realm, kClient)
}

func createUsers(realm dto.Realm, kClient keycloak.Client) error {
	for _, user := range realm.Users {
		err := createOneUser(user, realm, kClient)
		if err != nil {
			return err
		}
	}
	return nil
}

func createOneUser(user dto.User, realm dto.Realm, kClient keycloak.Client) error {
	exist, err := kClient.ExistRealmUser(realm.SsoRealmName, user)
	if err != nil {
		return err
	}
	if *exist {
		log.Info("User already exists", "user", user)
		return nil
	}
	return kClient.CreateRealmUser(realm.SsoRealmName, user)
}

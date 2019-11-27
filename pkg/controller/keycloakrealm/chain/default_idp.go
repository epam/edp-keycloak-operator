package chain

import (
	"github.com/epmd-edp/keycloak-operator/pkg/apis/v1/v1alpha1"
	"github.com/epmd-edp/keycloak-operator/pkg/client/keycloak"
	"github.com/epmd-edp/keycloak-operator/pkg/client/keycloak/dto"
	"github.com/epmd-edp/keycloak-operator/pkg/controller/keycloakrealm/chain/handler"
)

type PutDefaultIdP struct {
	next handler.RealmHandler
}

func (h PutDefaultIdP) ServeRequest(realm *v1alpha1.KeycloakRealm, kClient keycloak.Client) error {
	rLog := log.WithValues("realm name", realm.Spec.RealmName)
	rLog.Info("Start putting default identity provider...")

	rDto := dto.ConvertSpecToRealm(realm.Spec)
	err := kClient.PutDefaultIdp(rDto)
	if err != nil {
		return err
	}
	rLog.Info("Default identity provider has been successfully configured!")
	return nextServeOrNil(h.next, realm, kClient)
}

package chain

import (
	"context"

	keycloakApi "github.com/epam/edp-keycloak-operator/pkg/apis/v1/v1"
	"github.com/epam/edp-keycloak-operator/pkg/client/keycloak"
	"github.com/epam/edp-keycloak-operator/pkg/client/keycloak/dto"
	"github.com/epam/edp-keycloak-operator/pkg/controller/keycloakrealm/chain/handler"
)

type PutDefaultIdP struct {
	next handler.RealmHandler
}

func (h PutDefaultIdP) ServeRequest(ctx context.Context, realm *keycloakApi.KeycloakRealm, kClient keycloak.Client) error {
	rLog := log.WithValues("realm name", realm.Spec.RealmName)
	rLog.Info("Start putting default identity provider...")

	rDto := dto.ConvertSpecToRealm(realm.Spec)
	if !rDto.SsoRealmEnabled {
		rLog.Info("sso integration disabled, skip putting default identity provider")
		return nextServeOrNil(ctx, h.next, realm, kClient)
	}

	err := kClient.PutDefaultIdp(rDto)
	if err != nil {
		return err
	}
	rLog.Info("Default identity provider has been successfully configured!")
	return nextServeOrNil(ctx, h.next, realm, kClient)
}

package chain

import (
	"context"
	"fmt"

	keycloakApi "github.com/epam/edp-keycloak-operator/api/v1"
	"github.com/epam/edp-keycloak-operator/controllers/keycloakrealm/chain/handler"
	"github.com/epam/edp-keycloak-operator/pkg/client/keycloak"
	"github.com/epam/edp-keycloak-operator/pkg/client/keycloak/dto"
)

type PutDefaultIdP struct {
	next handler.RealmHandler
}

func (h PutDefaultIdP) ServeRequest(ctx context.Context, realm *keycloakApi.KeycloakRealm, kClient keycloak.Client) error {
	rLog := log.WithValues("realm name", realm.Spec.RealmName)
	rLog.Info("Start putting default identity provider...")

	rDto := dto.ConvertSpecToRealm(&realm.Spec)
	if !rDto.SsoRealmEnabled {
		rLog.Info("sso integration disabled, skip putting default identity provider")
		return nextServeOrNil(ctx, h.next, realm, kClient)
	}

	err := kClient.PutDefaultIdp(rDto)
	if err != nil {
		return fmt.Errorf("failed to put default edp: %w", err)
	}

	rLog.Info("Default identity provider has been successfully configured!")

	return nextServeOrNil(ctx, h.next, realm, kClient)
}

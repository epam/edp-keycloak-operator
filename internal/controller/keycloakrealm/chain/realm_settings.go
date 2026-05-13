package chain

import (
	"context"

	keycloakApi "github.com/epam/edp-keycloak-operator/api/v1"
	"github.com/epam/edp-keycloak-operator/internal/controller/keycloakrealm/chain/handler"
	"github.com/epam/edp-keycloak-operator/pkg/client/keycloakapi"
	"github.com/epam/edp-keycloak-operator/pkg/realmbuilder"
)

type RealmSettings struct {
	next handler.RealmHandler
}

func (h RealmSettings) ServeRequest(ctx context.Context, realm *keycloakApi.KeycloakRealm, kClient *keycloakapi.KeycloakClient) error {
	rLog := log.WithValues("realm name", realm.Spec.RealmName)
	rLog.Info("Start updating of Keycloak realm settings")

	if err := realmbuilder.ApplyRealmEventConfig(ctx, realm.Spec.RealmName, realm.Spec.RealmEventConfig, kClient.Events); err != nil {
		return err
	}

	overlay := realmbuilder.BuildRealmRepresentationFromV1(realm)

	if err := realmbuilder.ApplyRealmSettings(ctx, realm.Spec.RealmName, overlay, kClient.Realms); err != nil {
		return err
	}

	rLog.Info("Realm settings is updating done.")

	return nextServeOrNil(ctx, h.next, realm, kClient)
}

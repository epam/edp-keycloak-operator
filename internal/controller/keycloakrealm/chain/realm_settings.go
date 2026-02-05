package chain

import (
	"context"
	"fmt"

	keycloakApi "github.com/epam/edp-keycloak-operator/api/v1"
	"github.com/epam/edp-keycloak-operator/internal/controller/keycloakrealm/chain/handler"
	"github.com/epam/edp-keycloak-operator/pkg/client/keycloak"
	keycloakv2 "github.com/epam/edp-keycloak-operator/pkg/client/keycloakv2"
	"github.com/epam/edp-keycloak-operator/pkg/realmbuilder"
)

type RealmSettings struct {
	next            handler.RealmHandler
	settingsBuilder *realmbuilder.SettingsBuilder
}

func (h RealmSettings) ServeRequest(ctx context.Context, realm *keycloakApi.KeycloakRealm, kClient keycloak.Client, kClientV2 *keycloakv2.KeycloakClient) error {
	rLog := log.WithValues("realm name", realm.Spec.RealmName)
	rLog.Info("Start updating of Keycloak realm settings")

	if err := h.settingsBuilder.SetRealmEventConfigFromV1(kClient, realm.Spec.RealmName, realm.Spec.RealmEventConfig); err != nil {
		return fmt.Errorf("unable to set realm event config: %w", err)
	}

	settings := h.settingsBuilder.BuildFromV1(realm)

	if err := kClient.UpdateRealmSettings(realm.Spec.RealmName, &settings); err != nil {
		return fmt.Errorf("unable to update realm settings: %w", err)
	}

	if err := kClient.SetRealmOrganizationsEnabled(ctx, realm.Spec.RealmName, realm.Spec.OrganizationsEnabled); err != nil {
		return fmt.Errorf("unable to set realm organizations enabled: %w", err)
	}

	rLog.Info("Realm settings is updating done.")

	return nextServeOrNil(ctx, h.next, realm, kClient, kClientV2)
}

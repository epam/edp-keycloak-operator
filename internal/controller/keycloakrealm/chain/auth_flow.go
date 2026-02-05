package chain

import (
	"context"
	"fmt"

	keycloakApi "github.com/epam/edp-keycloak-operator/api/v1"
	"github.com/epam/edp-keycloak-operator/internal/controller/keycloakrealm/chain/handler"
	"github.com/epam/edp-keycloak-operator/pkg/client/keycloak"
	keycloakv2 "github.com/epam/edp-keycloak-operator/pkg/client/keycloakv2"
)

type AuthFlow struct {
	next handler.RealmHandler
}

func (a AuthFlow) ServeRequest(ctx context.Context, realm *keycloakApi.KeycloakRealm, kClient keycloak.Client, kClientV2 *keycloakv2.KeycloakClient) error {
	rLog := log.WithValues("realm name", realm.Spec.RealmName)
	rLog.Info("Start configuring keycloak realm auth flow", "flow", realm.Spec.BrowserFlow)

	if realm.Spec.BrowserFlow == nil {
		rLog.Info("Browser flow is empty, exit")
		return nextServeOrNil(ctx, a.next, realm, kClient, kClientV2)
	}

	if err := kClient.SetRealmBrowserFlow(ctx, realm.Spec.RealmName, *realm.Spec.BrowserFlow); err != nil {
		return fmt.Errorf("unable to set realm auth flow: %w", err)
	}

	rLog.Info("End of configuring keycloak realm auth flow")

	return nextServeOrNil(ctx, a.next, realm, kClient, kClientV2)
}

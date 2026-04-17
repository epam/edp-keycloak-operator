package chain

import (
	"context"
	"fmt"

	keycloakApi "github.com/epam/edp-keycloak-operator/api/v1"
	"github.com/epam/edp-keycloak-operator/internal/controller/keycloakrealm/chain/handler"
	keycloakapi "github.com/epam/edp-keycloak-operator/pkg/client/keycloakapi"
)

type AuthFlow struct {
	next handler.RealmHandler
}

func (a AuthFlow) ServeRequest(ctx context.Context, realm *keycloakApi.KeycloakRealm, keycloakAPIClient *keycloakapi.APIClient) error {
	rLog := log.WithValues("realm name", realm.Spec.RealmName)
	rLog.Info("Start configuring keycloak realm auth flow", "flow", realm.Spec.BrowserFlow)

	if realm.Spec.BrowserFlow == nil {
		rLog.Info("Browser flow is empty, exit")
		return nextServeOrNil(ctx, a.next, realm, keycloakAPIClient)
	}

	if _, err := keycloakAPIClient.Realms.SetRealmBrowserFlow(ctx, realm.Spec.RealmName, *realm.Spec.BrowserFlow); err != nil {
		return fmt.Errorf("unable to set realm auth flow: %w", err)
	}

	rLog.Info("End of configuring keycloak realm auth flow")

	return nextServeOrNil(ctx, a.next, realm, keycloakAPIClient)
}

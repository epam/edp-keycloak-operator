package chain

import (
	"context"

	"github.com/epam/edp-keycloak-operator/pkg/apis/v1/v1alpha1"
	"github.com/epam/edp-keycloak-operator/pkg/client/keycloak"
	"github.com/epam/edp-keycloak-operator/pkg/controller/keycloakrealm/chain/handler"
	"github.com/pkg/errors"
)

type AuthFlow struct {
	next handler.RealmHandler
}

func (a AuthFlow) ServeRequest(ctx context.Context, realm *v1alpha1.KeycloakRealm, kClient keycloak.Client) error {
	rLog := log.WithValues("realm name", realm.Spec.RealmName)
	rLog.Info("Start configuring keycloak realm auth flow", "flow", realm.Spec.BrowserFlow)

	if realm.Spec.BrowserFlow == nil {
		rLog.Info("Browser flow is empty, exit")
		return nextServeOrNil(ctx, a.next, realm, kClient)
	}

	if err := kClient.SetRealmBrowserFlow(realm.Spec.RealmName, *realm.Spec.BrowserFlow); err != nil {
		return errors.Wrap(err, "unable to set realm auth flow")
	}

	rLog.Info("End of configuring keycloak realm auth flow")
	return nextServeOrNil(ctx, a.next, realm, kClient)
}

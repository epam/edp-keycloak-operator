package chain

import (
	"context"
	"fmt"

	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	keycloakApi "github.com/epam/edp-keycloak-operator/api/v1"
	"github.com/epam/edp-keycloak-operator/pkg/client/keycloak"
	"github.com/epam/edp-keycloak-operator/pkg/secretref"
)

type ClientHandler interface {
	Serve(
		ctx context.Context,
		keycloakClient *keycloakApi.KeycloakClient,
		realmName string,
	) error
}

type Chain struct {
	handlers []ClientHandler
}

func (ch *Chain) Use(handlers ...ClientHandler) {
	ch.handlers = append(ch.handlers, handlers...)
}

func (ch *Chain) Serve(
	ctx context.Context,
	keycloakClient *keycloakApi.KeycloakClient,
	realmName string,
) error {
	log := ctrl.LoggerFrom(ctx)

	log.Info("Starting KeycloakClient chain")

	for i := 0; i < len(ch.handlers); i++ {
		h := ch.handlers[i]

		err := h.Serve(ctx, keycloakClient, realmName)
		if err != nil {
			log.Info("KeycloakClient chain finished with error")

			return fmt.Errorf("failed to serve handler: %w", err)
		}
	}

	log.Info("Handling of KeycloakClient has been finished")

	return nil
}

func MakeChain(
	keycloakApiClient keycloak.Client,
	k8sClient client.Client,
) *Chain {
	c := &Chain{}

	c.Use(
		NewPutClient(keycloakApiClient, k8sClient, secretref.NewSecretRef(k8sClient)),
		NewPutClientRole(keycloakApiClient),
		NewPutRealmRole(keycloakApiClient),
		NewPutClientScope(keycloakApiClient),
		NewPutProtocolMappers(keycloakApiClient),
		NewServiceAccount(keycloakApiClient),
		NewProcessScope(keycloakApiClient),
		NewProcessPolicy(keycloakApiClient),
		NewProcessPermissions(keycloakApiClient),
	)

	return c
}

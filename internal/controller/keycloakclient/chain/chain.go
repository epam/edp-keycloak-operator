package chain

import (
	"context"
	"fmt"

	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	keycloakApi "github.com/epam/edp-keycloak-operator/api/v1"
	keycloakv2 "github.com/epam/edp-keycloak-operator/pkg/client/keycloakv2"
	"github.com/epam/edp-keycloak-operator/pkg/secretref"
)

// ClientContext holds data that is passed between chain handlers.
type ClientContext struct {
	// ClientUUID is the Keycloak client UUID, set by PutClient handler.
	ClientUUID string
}

type ClientHandler interface {
	Serve(
		ctx context.Context,
		keycloakClient *keycloakApi.KeycloakClient,
		realmName string,
		clientCtx *ClientContext,
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

	clientCtx := &ClientContext{}

	for i := 0; i < len(ch.handlers); i++ {
		h := ch.handlers[i]

		err := h.Serve(ctx, keycloakClient, realmName, clientCtx)
		if err != nil {
			log.Info("KeycloakClient chain finished with error")

			return fmt.Errorf("failed to serve handler: %w", err)
		}
	}

	log.Info("Handling of KeycloakClient has been finished")

	return nil
}

func MakeChain(
	kClient *keycloakv2.KeycloakClient,
	k8sClient client.Client,
) *Chain {
	c := &Chain{}

	c.Use(
		NewPutClient(kClient, k8sClient, secretref.NewSecretRef(k8sClient)),
		NewPutClientRole(kClient, k8sClient),
		NewPutRealmRole(kClient, k8sClient),
		NewPutClientScope(kClient, k8sClient),
		NewPutProtocolMappers(kClient, k8sClient),
		NewServiceAccount(kClient, k8sClient),
		NewProcessScope(kClient, k8sClient),
		NewProcessResources(kClient, k8sClient),
		NewProcessPolicy(kClient, k8sClient),
		NewProcessPermissions(kClient, k8sClient),
		NewPutAdminFineGrainedPermissions(kClient, k8sClient),
	)

	return c
}

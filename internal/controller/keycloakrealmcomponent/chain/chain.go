package chain

import (
	"context"
	"fmt"

	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	keycloakApi "github.com/epam/edp-keycloak-operator/api/v1"
	keycloakapi "github.com/epam/edp-keycloak-operator/pkg/client/keycloakapi"
)

// SecretRefClient resolves secret references in component config.
type SecretRefClient interface {
	MapComponentConfigSecretsRefs(ctx context.Context, config map[string][]string, namespace string) error
}

// RealmComponentHandler is a handler in the chain of responsibility for KeycloakRealmComponent.
type RealmComponentHandler interface {
	Serve(ctx context.Context, component *keycloakApi.KeycloakRealmComponent, realmName string) error
}

// Chain executes a sequence of RealmComponentHandler instances.
type Chain struct {
	handlers []RealmComponentHandler
}

func (ch *Chain) Use(handlers ...RealmComponentHandler) {
	ch.handlers = append(ch.handlers, handlers...)
}

func (ch *Chain) Serve(ctx context.Context, component *keycloakApi.KeycloakRealmComponent, realmName string) error {
	log := ctrl.LoggerFrom(ctx)
	log.Info("Starting KeycloakRealmComponent chain")

	for _, h := range ch.handlers {
		if err := h.Serve(ctx, component, realmName); err != nil {
			log.Info("KeycloakRealmComponent chain finished with error")

			return fmt.Errorf("failed to serve handler: %w", err)
		}
	}

	log.Info("Handling of KeycloakRealmComponent has been finished")

	return nil
}

// MakeChain creates the reconciliation chain for KeycloakRealmComponent.
func MakeChain(k8sClient client.Client, keycloakAPIClient *keycloakapi.APIClient, secretRefClient SecretRefClient) *Chain {
	ch := &Chain{}

	ch.Use(
		NewCreateOrUpdateComponent(k8sClient, keycloakAPIClient, secretRefClient),
	)

	return ch
}

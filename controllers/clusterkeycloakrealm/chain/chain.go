package chain

import (
	"context"
	"fmt"

	ctrl "sigs.k8s.io/controller-runtime"

	"github.com/epam/edp-keycloak-operator/api/v1alpha1"
	"github.com/epam/edp-keycloak-operator/pkg/client/keycloak"
)

type RealmHandler interface {
	ServeRequest(ctx context.Context, realm *v1alpha1.ClusterKeycloakRealm, kClient keycloak.Client) error
}

type chain struct {
	handlers []RealmHandler
}

func (ch *chain) Use(handlers ...RealmHandler) {
	ch.handlers = append(ch.handlers, handlers...)
}

func (ch *chain) ServeRequest(ctx context.Context, realm *v1alpha1.ClusterKeycloakRealm, kClient keycloak.Client) error {
	log := ctrl.LoggerFrom(ctx)

	log.Info("Starting ClusterKeycloak chain")

	for i := 0; i < len(ch.handlers); i++ {
		h := ch.handlers[i]

		err := h.ServeRequest(ctx, realm, kClient)
		if err != nil {
			log.Info("ClusterKeycloak chain finished with error")

			return fmt.Errorf("failed to serve handler: %w", err)
		}
	}

	log.Info("Handling of ClusterKeycloak has been finished")

	return nil
}

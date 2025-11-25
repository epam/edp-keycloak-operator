package chain

import (
	"context"
	"fmt"

	"github.com/Nerzal/gocloak/v12"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	keycloakApi "github.com/epam/edp-keycloak-operator/api/v1"
	"github.com/epam/edp-keycloak-operator/pkg/client/keycloak"
)

type CleanupResource struct {
	k8sClient client.Client
}

func NewCleanupResource(k8sClient client.Client) *CleanupResource {
	return &CleanupResource{k8sClient: k8sClient}
}

func (h *CleanupResource) Serve(
	ctx context.Context,
	user *keycloakApi.KeycloakRealmUser,
	kClient keycloak.Client,
	realm *gocloak.RealmRepresentation,
	userCtx *UserContext,
) error {
	if user.Spec.KeepResource {
		return nil
	}

	log := ctrl.LoggerFrom(ctx)
	log.Info("Deleting KeycloakRealmUser resource as KeepResource is false")

	if err := h.k8sClient.Delete(ctx, user); client.IgnoreNotFound(err) != nil {
		return fmt.Errorf("unable to delete instance of keycloak realm user: %w", err)
	}

	return nil
}

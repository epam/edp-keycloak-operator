package chain

import (
	"context"
	"fmt"

	ctrl "sigs.k8s.io/controller-runtime"

	keycloakApi "github.com/epam/edp-keycloak-operator/api/v1"
	keycloakapi "github.com/epam/edp-keycloak-operator/pkg/client/keycloakapi"
	"github.com/epam/edp-keycloak-operator/pkg/objectmeta"
)

type RemoveUser struct {
	kClientV2 *keycloakapi.APIClient
}

func NewRemoveUser(kClientV2 *keycloakapi.APIClient) *RemoveUser {
	return &RemoveUser{kClientV2: kClientV2}
}

func (h *RemoveUser) ServeRequest(ctx context.Context, user *keycloakApi.KeycloakRealmUser, realmName string) error {
	log := ctrl.LoggerFrom(ctx)

	log.Info("Start removing user")

	if objectmeta.PreserveResourcesOnDeletion(user) {
		log.Info("Preserve resources on deletion, skipping")

		return nil
	}

	keycloakUser, _, err := h.kClientV2.Users.FindUserByUsername(ctx, realmName, user.Spec.Username)
	if err != nil {
		if keycloakapi.IsNotFound(err) {
			log.Info("User not found, skipping")

			return nil
		}

		return fmt.Errorf("failed to find user %s: %w", user.Spec.Username, err)
	}

	if _, err := h.kClientV2.Users.DeleteUser(ctx, realmName, *keycloakUser.Id); err != nil {
		if keycloakapi.IsNotFound(err) {
			log.Info("User not found, skipping")

			return nil
		}

		return fmt.Errorf("failed to delete user %s: %w", user.Spec.Username, err)
	}

	log.Info("User deleted successfully")

	return nil
}

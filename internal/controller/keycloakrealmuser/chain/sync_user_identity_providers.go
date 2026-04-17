package chain

import (
	"context"
	"fmt"
	"slices"

	"k8s.io/utils/ptr"
	ctrl "sigs.k8s.io/controller-runtime"

	keycloakApi "github.com/epam/edp-keycloak-operator/api/v1"
	keycloakapi "github.com/epam/edp-keycloak-operator/pkg/client/keycloakapi"
)

type SyncUserIdentityProviders struct {
	kClientV2 *keycloakapi.APIClient
}

func NewSyncUserIdentityProviders(kClientV2 *keycloakapi.APIClient) *SyncUserIdentityProviders {
	return &SyncUserIdentityProviders{kClientV2: kClientV2}
}

func (h *SyncUserIdentityProviders) Serve(
	ctx context.Context,
	user *keycloakApi.KeycloakRealmUser,
	realmName string,
	userCtx *UserContext,
) error {
	if user.Spec.IdentityProviders == nil {
		return nil
	}

	log := ctrl.LoggerFrom(ctx)
	log.Info("Syncing user identity providers")

	providers := *user.Spec.IdentityProviders

	existing, _, err := h.kClientV2.Users.GetUserFederatedIdentities(ctx, realmName, userCtx.UserID)
	if err != nil {
		return fmt.Errorf("unable to get user federated identities: %w", err)
	}

	existingByAlias := make(map[string]struct{}, len(existing))

	for _, id := range existing {
		if id.IdentityProvider != nil {
			existingByAlias[*id.IdentityProvider] = struct{}{}
		}
	}

	// Add missing providers
	for _, provider := range providers {
		if _, exists := existingByAlias[provider]; exists {
			continue
		}

		// Check the identity provider exists in Keycloak before linking
		if _, _, err := h.kClientV2.IdentityProviders.GetIdentityProvider(ctx, realmName, provider); err != nil {
			if keycloakapi.IsNotFound(err) {
				return fmt.Errorf("identity provider %q does not exist", provider)
			}

			return fmt.Errorf("unable to check if identity provider %q exists: %w", provider, err)
		}

		_, err := h.kClientV2.Users.CreateUserFederatedIdentity(
			ctx,
			realmName,
			userCtx.UserID,
			provider,
			keycloakapi.FederatedIdentityRepresentation{
				IdentityProvider: ptr.To(provider),
				UserId:           ptr.To(userCtx.UserID),
				UserName:         ptr.To(user.Spec.Username),
			},
		)
		if err != nil {
			return fmt.Errorf("unable to add user to identity provider %q: %w", provider, err)
		}
	}

	// Remove providers no longer desired
	for alias := range existingByAlias {
		if !slices.Contains(providers, alias) {
			if _, err := h.kClientV2.Users.DeleteUserFederatedIdentity(ctx, realmName, userCtx.UserID, alias); err != nil {
				return fmt.Errorf("unable to remove user from identity provider %q: %w", alias, err)
			}
		}
	}

	log.Info("User identity providers synced successfully")

	return nil
}

package chain

import (
	"context"
	"fmt"

	"github.com/Nerzal/gocloak/v12"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	keycloakApi "github.com/epam/edp-keycloak-operator/api/v1"
	"github.com/epam/edp-keycloak-operator/pkg/client/keycloak"
	"github.com/epam/edp-keycloak-operator/pkg/client/keycloak/adapter"
)

type CreateOrUpdateUser struct {
	k8sClient client.Client
}

func NewCreateOrUpdateUser(k8sClient client.Client) *CreateOrUpdateUser {
	return &CreateOrUpdateUser{k8sClient: k8sClient}
}

func (h *CreateOrUpdateUser) Serve(
	ctx context.Context,
	user *keycloakApi.KeycloakRealmUser,
	kClient keycloak.Client,
	realm *gocloak.RealmRepresentation,
	userCtx *UserContext,
) error {
	log := ctrl.LoggerFrom(ctx)
	log.Info("Creating or updating user in Keycloak")

	userSpec := user.Spec.DeepCopy()

	userID, err := kClient.CreateOrUpdateUser(ctx, gocloak.PString(realm.Realm), &adapter.KeycloakUser{
		Username:            userSpec.Username,
		Enabled:             userSpec.Enabled,
		EmailVerified:       userSpec.EmailVerified,
		Email:               userSpec.Email,
		FirstName:           userSpec.FirstName,
		LastName:            userSpec.LastName,
		RequiredUserActions: userSpec.RequiredUserActions,
		Attributes:          userSpec.AttributesV2,
	}, user.IsReconciliationStrategyAddOnly())
	if err != nil {
		return fmt.Errorf("unable to create or update user: %w", err)
	}

	userCtx.UserID = userID

	log.Info("User created or updated successfully", "userID", userID)

	return nil
}

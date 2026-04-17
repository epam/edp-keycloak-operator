package chain

import (
	"context"
	"fmt"

	ctrl "sigs.k8s.io/controller-runtime"

	keycloakApi "github.com/epam/edp-keycloak-operator/api/v1"
	keycloakapi "github.com/epam/edp-keycloak-operator/pkg/client/keycloakapi"
)

type MakeDefault struct {
	kClientV2 *keycloakapi.APIClient
}

func NewMakeDefault(kClientV2 *keycloakapi.APIClient) *MakeDefault {
	return &MakeDefault{kClientV2: kClientV2}
}

func (h *MakeDefault) Serve(
	ctx context.Context,
	role *keycloakApi.KeycloakRealmRole,
	realmName string,
	roleCtx *RoleContext,
) error {
	if !role.Spec.IsDefault {
		return nil
	}

	log := ctrl.LoggerFrom(ctx)
	log.Info("Making role default")

	name := role.Spec.Name
	defaultRoleName := "default-roles-" + realmName

	if _, err := h.kClientV2.Roles.AddRealmRoleComposites(ctx, realmName, defaultRoleName, []keycloakapi.RoleRepresentation{
		{
			Id:   &roleCtx.RoleID,
			Name: &name,
		},
	}); err != nil {
		return fmt.Errorf("failed to add role to default-roles: %w", err)
	}

	log.Info("Role has been made default")

	return nil
}

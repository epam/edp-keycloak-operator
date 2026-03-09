package chain

import (
	"context"
	"fmt"

	ctrl "sigs.k8s.io/controller-runtime"

	keycloakApi "github.com/epam/edp-keycloak-operator/api/v1"
	keycloakv2 "github.com/epam/edp-keycloak-operator/pkg/client/keycloakv2"
)

type CreateOrUpdateRole struct {
	kClientV2 *keycloakv2.KeycloakClient
}

func NewCreateOrUpdateRole(kClientV2 *keycloakv2.KeycloakClient) *CreateOrUpdateRole {
	return &CreateOrUpdateRole{kClientV2: kClientV2}
}

func (h *CreateOrUpdateRole) Serve(
	ctx context.Context,
	role *keycloakApi.KeycloakRealmRole,
	realmName string,
	roleCtx *RoleContext,
) error {
	log := ctrl.LoggerFrom(ctx)
	log.Info("Creating or updating realm role")

	spec := role.Spec
	rolesClient := h.kClientV2.Roles

	existingRole, _, err := rolesClient.GetRealmRole(ctx, realmName, spec.Name)
	if err != nil && !keycloakv2.IsNotFound(err) {
		return fmt.Errorf("failed to get realm role: %w", err)
	}

	isComposite := spec.Composite
	attrs := spec.Attributes
	desc := spec.Description

	if existingRole == nil {
		if _, err = rolesClient.CreateRealmRole(ctx, realmName, keycloakv2.RoleRepresentation{
			Name:        &spec.Name,
			Description: &desc,
			Composite:   &isComposite,
			Attributes:  &attrs,
		}); err != nil {
			return fmt.Errorf("failed to create realm role: %w", err)
		}

		existingRole, _, err = rolesClient.GetRealmRole(ctx, realmName, spec.Name)
		if err != nil {
			return fmt.Errorf("failed to get created realm role: %w", err)
		}
	} else {
		existingRole.Description = &desc
		existingRole.Composite = &isComposite
		existingRole.Attributes = &attrs

		if _, err = rolesClient.UpdateRealmRole(ctx, realmName, spec.Name, *existingRole); err != nil {
			return fmt.Errorf("failed to update realm role: %w", err)
		}
	}

	if existingRole.Id != nil {
		roleCtx.RoleID = *existingRole.Id
	}

	log.Info("Realm role has been synced")

	return nil
}

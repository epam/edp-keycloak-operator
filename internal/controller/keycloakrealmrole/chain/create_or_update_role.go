package chain

import (
	"context"
	"fmt"

	ctrl "sigs.k8s.io/controller-runtime"

	keycloakApi "github.com/epam/edp-keycloak-operator/api/v1"
	"github.com/epam/edp-keycloak-operator/pkg/client/keycloakapi"
)

type CreateOrUpdateRole struct {
	kClient *keycloakapi.KeycloakClient
}

func NewCreateOrUpdateRole(kClient *keycloakapi.KeycloakClient) *CreateOrUpdateRole {
	return &CreateOrUpdateRole{kClient: kClient}
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
	rolesClient := h.kClient.Roles

	existingRole, _, err := rolesClient.GetRealmRole(ctx, realmName, spec.Name)
	if err != nil && !keycloakapi.IsNotFound(err) {
		return fmt.Errorf("failed to get realm role: %w", err)
	}

	isComposite := spec.Composite
	attrs := spec.Attributes
	desc := spec.Description

	if existingRole == nil {
		if _, err = rolesClient.CreateRealmRole(ctx, realmName, keycloakapi.RoleRepresentation{
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

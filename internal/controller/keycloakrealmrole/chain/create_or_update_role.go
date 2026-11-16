package chain

import (
	"context"
	"fmt"
	"maps"

	"k8s.io/utils/ptr"
	ctrl "sigs.k8s.io/controller-runtime"

	keycloakApi "github.com/epam/edp-keycloak-operator/api/v1"
	"github.com/epam/edp-keycloak-operator/pkg/client/keycloakapi"
	"github.com/epam/edp-keycloak-operator/pkg/maputil"
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
	} else if !roleMatchesSpec(existingRole, &spec) {
		existingRole.Description = &desc
		existingRole.Composite = &isComposite
		existingRole.Attributes = &attrs

		if _, err = rolesClient.UpdateRealmRole(ctx, realmName, spec.Name, *existingRole); err != nil {
			return fmt.Errorf("failed to update realm role: %w", err)
		}
	}

	// MakeDefault maps RoleID as a composite, and Keycloak resolves a composite by id alone.
	// An unset RoleID would reach it as an empty id.
	syncedRole, err := keycloakapi.RequireRealmRoleWithID(existingRole, realmName, spec.Name)
	if err != nil {
		return fmt.Errorf("realm role is not addressable after sync: %w", err)
	}

	roleCtx.RoleID = *syncedRole.Id

	log.Info("Realm role has been synced")

	return nil
}

// roleMatchesSpec reports whether the role in Keycloak already matches the spec, so a periodic
// reconcile does not issue an update (and an admin event) when nothing changed.
// Nil spec attributes are not managed: the update request sends them as null.
func roleMatchesSpec(existing *keycloakapi.RoleRepresentation, spec *keycloakApi.KeycloakRealmRoleSpec) bool {
	if ptr.Deref(existing.Description, "") != spec.Description || ptr.Deref(existing.Composite, false) != spec.Composite {
		return false
	}

	return spec.Attributes == nil || roleAttributesMatch(ptr.Deref(existing.Attributes, nil), spec.Attributes)
}

// roleAttributesMatch compares key sets and per-key value multisets.
// A key without values equals an absent key: Keycloak stores one row per value.
func roleAttributesMatch(existing, desired map[string][]string) bool {
	have, want := nonEmptyAttributes(existing), nonEmptyAttributes(desired)

	return len(have) == len(want) && maputil.ContainsSubsetMulti(have, want)
}

func nonEmptyAttributes(m map[string][]string) map[string][]string {
	out := maps.Clone(m)
	maps.DeleteFunc(out, func(_ string, v []string) bool { return len(v) == 0 })

	return out
}

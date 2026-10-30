package chain

import (
	"context"
	"fmt"
	"slices"

	ctrl "sigs.k8s.io/controller-runtime"

	keycloakApi "github.com/epam/edp-keycloak-operator/api/v1"
	"github.com/epam/edp-keycloak-operator/pkg/client/keycloakapi"
)

type MakeDefault struct {
	kClient *keycloakapi.KeycloakClient
}

func NewMakeDefault(kClient *keycloakapi.KeycloakClient) *MakeDefault {
	return &MakeDefault{kClient: kClient}
}

// Serve syncs the role's membership in the realm default role with spec.isDefault; writes only on a difference.
func (h *MakeDefault) Serve(
	ctx context.Context,
	role *keycloakApi.KeycloakRealmRole,
	realmName string,
	roleCtx *RoleContext,
) error {
	log := ctrl.LoggerFrom(ctx)

	defaultRoleName, err := h.defaultRoleName(ctx, realmName)
	if err != nil {
		return err
	}

	composites, _, err := h.kClient.Roles.GetRealmRoleComposites(ctx, realmName, defaultRoleName)
	if err != nil {
		return fmt.Errorf("failed to get default-roles composites: %w", err)
	}

	// Match by id: the list mixes realm and client roles.
	member := slices.ContainsFunc(composites, func(r keycloakapi.RoleRepresentation) bool {
		return r.Id != nil && *r.Id == roleCtx.RoleID
	})

	name := role.Spec.Name
	rep := []keycloakapi.RoleRepresentation{{Id: &roleCtx.RoleID, Name: &name}}

	switch {
	case role.Spec.IsDefault && !member:
		if _, err = h.kClient.Roles.AddRealmRoleComposites(ctx, realmName, defaultRoleName, rep); err != nil {
			return fmt.Errorf("failed to add role to default-roles: %w", err)
		}

		log.Info("Role has been made default")
	case !role.Spec.IsDefault && member:
		_, err = h.kClient.Roles.DeleteRealmRoleComposites(ctx, realmName, defaultRoleName, rep)

		switch {
		case err == nil:
			log.Info("Role has been removed from default roles")
		case !keycloakapi.IsNotFound(err):
			return fmt.Errorf("failed to remove role from default-roles: %w", err)
		}
	}

	return nil
}

// defaultRoleName reads the name from the realm.
func (h *MakeDefault) defaultRoleName(ctx context.Context, realmName string) (string, error) {
	realm, _, err := h.kClient.Realms.GetRealm(ctx, realmName)
	if err != nil {
		return "", fmt.Errorf("failed to get realm %q to resolve its default role: %w", realmName, err)
	}

	if realm == nil {
		return "", fmt.Errorf("failed to get realm %q: %w", realmName, keycloakapi.ErrNilRealmRepresentation)
	}

	if realm.DefaultRole == nil || realm.DefaultRole.Name == nil || *realm.DefaultRole.Name == "" {
		return "", fmt.Errorf("realm %q has no default role in the Keycloak response", realmName)
	}

	return *realm.DefaultRole.Name, nil
}

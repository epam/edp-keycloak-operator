package adapter

import (
	"context"
	"fmt"

	"github.com/Nerzal/gocloak/v12"
	"github.com/pkg/errors"

	"github.com/epam/edp-keycloak-operator/pkg/client/keycloak/dto"
)

func (a GoCloakAdapter) SyncRealmRole(ctx context.Context, realmName string, role *dto.PrimaryRealmRole) error {
	if err := a.createOrUpdateRealmRole(ctx, realmName, role); err != nil {
		return errors.Wrap(err, "error during createOrUpdateRealmRole")
	}

	if err := a.makeRoleDefault(ctx, realmName, role); err != nil {
		return errors.Wrap(err, "error during makeRoleDefault")
	}

	return nil
}

func (a GoCloakAdapter) createOrUpdateRealmRole(
	ctx context.Context,
	realmName string,
	role *dto.PrimaryRealmRole,
) error {
	exists := true

	currentRealmRole, err := a.client.GetRealmRole(ctx, a.token.AccessToken, realmName, role.Name)
	if err != nil {
		if !IsErrNotFound(err) {
			return fmt.Errorf("failed to get realm role: %w", err)
		}

		exists = false
	}

	if exists {
		role.ID = currentRealmRole.ID
	}

	if !exists {
		var roleID string

		if roleID, err = a.CreatePrimaryRealmRole(ctx, realmName, role); err != nil {
			return err
		}

		role.ID = &roleID
	}

	if role.IsComposite {
		if err = a.syncRoleComposites(ctx, realmName, role); err != nil {
			return err
		}
	}

	if exists {
		currentRealmRole.Composite = &role.IsComposite
		currentRealmRole.Attributes = &role.Attributes
		currentRealmRole.Description = &role.Description

		if err = a.client.UpdateRealmRole(ctx, a.token.AccessToken, realmName, role.Name, *currentRealmRole); err != nil {
			return errors.Wrap(err, "unable to update realm role")
		}
	}

	return nil
}

func (a GoCloakAdapter) ExistRealmRole(realmName string, roleName string) (bool, error) {
	reqLog := a.log.WithValues("realm name", realmName, "role name", roleName)
	reqLog.Info("Start check existing realm role...")

	_, err := a.client.GetRealmRole(context.Background(), a.token.AccessToken, realmName, roleName)

	res, err := strip404(err)
	if err != nil {
		return false, err
	}

	reqLog.Info("Check existing realm role has been finished", "result", res)

	return res, nil
}

func (a GoCloakAdapter) DeleteRealmRole(ctx context.Context, realm, roleName string) error {
	if err := a.client.DeleteRealmRole(ctx, a.token.AccessToken, realm, roleName); err != nil {
		return errors.Wrap(err, "unable to delete realm role")
	}

	return nil
}

// makeRoleDefault makes the role default if it is required.
// For this purpose, the role is added to the composite role with the name "default-roles-{realmName}".
func (a GoCloakAdapter) makeRoleDefault(ctx context.Context, realmName string, role *dto.PrimaryRealmRole) error {
	if !role.IsDefault {
		return nil
	}

	if err := a.client.AddRealmRoleComposite(
		ctx,
		a.token.AccessToken,
		realmName,
		GetDefaultCompositeRoleName(realmName),
		[]gocloak.Role{
			{
				ID:   role.ID,
				Name: &role.Name,
			},
		},
	); err != nil {
		return fmt.Errorf("failed to make the role default: %w", err)
	}

	return nil
}

// GetDefaultCompositeRoleName returns the name of the composite role,
// which stores all default roles for the given realm.
// The name is generated according to the Keycloak documentation:
// https://www.keycloak.org/docs/22.0.5/release_notes/#default-roles-processing-improvement
func GetDefaultCompositeRoleName(realmName string) string {
	return "default-roles-" + realmName
}

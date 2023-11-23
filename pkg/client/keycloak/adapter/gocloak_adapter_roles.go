package adapter

import (
	"context"
	"fmt"

	"github.com/Nerzal/gocloak/v12"
	"github.com/pkg/errors"

	"github.com/epam/edp-keycloak-operator/pkg/client/keycloak/dto"
)

type DuplicatedError string

func (e DuplicatedError) Error() string {
	return string(e)
}

func IsErrDuplicated(err error) bool {
	errDuplicate := DuplicatedError("")

	return errors.As(err, &errDuplicate)
}

func (a GoCloakAdapter) SyncRealmRole(ctx context.Context, realmName string, role *dto.PrimaryRealmRole) error {
	if err := a.createOrUpdateRealmRole(realmName, role); err != nil {
		return errors.Wrap(err, "error during createOrUpdateRealmRole")
	}

	if err := a.makeRoleDefault(ctx, realmName, role); err != nil {
		return errors.Wrap(err, "error during makeRoleDefault")
	}

	return nil
}

func (a GoCloakAdapter) createOrUpdateRealmRole(realmName string, role *dto.PrimaryRealmRole) error {
	currentRealmRole, err := a.client.GetRealmRole(context.Background(), a.token.AccessToken, realmName, role.Name)

	exists, err := strip404(err)
	if err != nil {
		return errors.Wrap(err, "unable to get realm role")
	}

	if !exists {
		_, err := a.CreatePrimaryRealmRole(realmName, role)
		if err != nil {
			return errors.Wrap(err, "unable to create realm role during sync")
		}

		currentRealmRole, err = a.client.GetRealmRole(context.Background(), a.token.AccessToken, realmName, role.Name)
		if err != nil {
			return errors.Wrap(err, "unable to get realm role")
		}

		role.ID = currentRealmRole.ID

		return nil
	}

	if role.ID == nil {
		return DuplicatedError("role is duplicated")
	}

	if err := a.syncRoleComposites(realmName, role, currentRealmRole); err != nil {
		return errors.Wrap(err, "error during syncRoleComposites")
	}

	currentRealmRole.Composite = &role.IsComposite
	currentRealmRole.Attributes = &role.Attributes
	currentRealmRole.Description = &role.Description

	if err := a.client.UpdateRealmRole(context.Background(), a.token.AccessToken, realmName, role.Name,
		*currentRealmRole); err != nil {
		return errors.Wrap(err, "unable to update realm role")
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

func (a GoCloakAdapter) syncRoleComposites(realmName string, role *dto.PrimaryRealmRole, currentRealmRole *gocloak.Role) error {
	currentComposites, err := a.client.GetCompositeRealmRolesByRoleID(context.Background(), a.token.AccessToken, realmName, *currentRealmRole.ID)
	if err != nil {
		return errors.Wrap(err, "unable to get realm role composites")
	}

	if err := a.syncCreateNewComposites(realmName, role, currentComposites); err != nil {
		return errors.Wrap(err, "error during SyncCreateNewComposites")
	}

	// temporary disable deletion of old composites to remove conflict with keycloak client roles

	// if err := a.syncDeleteOldComposites(realmName, role, currentComposites); err != nil {
	//	return errors.Wrap(err, "error during SyncDeleteOldComposites")
	//}

	return nil
}

func (a GoCloakAdapter) syncCreateNewComposites(realmName string, role *dto.PrimaryRealmRole, currentComposites []*gocloak.Role) error {
	currentCompositesMap := make(map[string]string)

	for _, currentComposite := range currentComposites {
		currentCompositesMap[*currentComposite.Name] = *currentComposite.Name
	}

	rolesToAdd := make([]gocloak.Role, 0, len(role.Composites))

	for _, claimedComposite := range role.Composites {
		if _, ok := currentCompositesMap[claimedComposite]; !ok {
			compRole, err := a.client.GetRealmRole(context.Background(), a.token.AccessToken, realmName,
				claimedComposite)
			if err != nil {
				return errors.Wrap(err, "unable to get realm role")
			}

			rolesToAdd = append(rolesToAdd, *compRole)
		}
	}

	if len(rolesToAdd) > 0 {
		if err := a.client.AddRealmRoleComposite(context.Background(), a.token.AccessToken, realmName,
			role.Name, rolesToAdd); err != nil {
			return errors.Wrap(err, "unable to add role composite")
		}
	}

	return nil
}

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

// GetDefaultCompositeRoleName returns the name of the composite role, which stores all default roles for the given realm.
// The name is generated according to the Keycloak documentation: https://www.keycloak.org/docs/22.0.5/release_notes/#default-roles-processing-improvement
func GetDefaultCompositeRoleName(realmName string) string {
	return "default-roles-" + realmName
}

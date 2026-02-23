package chain

import (
	"context"
	"fmt"

	keycloakApi "github.com/epam/edp-keycloak-operator/api/v1"
	"github.com/epam/edp-keycloak-operator/internal/controller/keycloakrealm/chain/handler"
	keycloakv2 "github.com/epam/edp-keycloak-operator/pkg/client/keycloakv2"
)

type PutUsersRoles struct {
	next handler.RealmHandler
}

func (h PutUsersRoles) ServeRequest(ctx context.Context, realm *keycloakApi.KeycloakRealm, kClientV2 *keycloakv2.KeycloakClient) error {
	rLog := log.WithValues("keycloak users", realm.Spec.Users)
	rLog.Info("Start putting roles to users")

	if err := putRolesToUsers(ctx, realm.Spec.RealmName, realm.Spec.Users, kClientV2); err != nil {
		return fmt.Errorf("error during putRolesToUsers: %w", err)
	}

	rLog.Info("End put role to users")

	return nextServeOrNil(ctx, h.next, realm, kClientV2)
}

func putRolesToUsers(ctx context.Context, realmName string, users []keycloakApi.User, kClientV2 *keycloakv2.KeycloakClient) error {
	for _, user := range users {
		if err := putRolesToOneUser(ctx, realmName, user.Username, user.RealmRoles, kClientV2); err != nil {
			return fmt.Errorf("error during putRolesToOneUser: %w", err)
		}
	}

	return nil
}

func putRolesToOneUser(ctx context.Context, realmName, username string, realmRoles []string, kClientV2 *keycloakv2.KeycloakClient) error {
	for _, role := range realmRoles {
		if err := putOneRealmRoleToOneUser(ctx, realmName, username, role, kClientV2); err != nil {
			return fmt.Errorf("error during putOneRoleToOneUser: %w", err)
		}
	}

	return nil
}

func putOneRealmRoleToOneUser(ctx context.Context, realmName, username, role string, kClientV2 *keycloakv2.KeycloakClient) error {
	user, _, err := kClientV2.Users.FindUserByUsername(ctx, realmName, username)
	if err != nil {
		return fmt.Errorf("unable to find user by username: %w", err)
	}

	if user == nil {
		return fmt.Errorf("user %s not found in realm %s", username, realmName)
	}

	existingRoles, _, err := kClientV2.Users.GetUserRealmRoleMappings(ctx, realmName, *user.Id)
	if err != nil {
		return fmt.Errorf("unable to get user realm role mappings: %w", err)
	}

	for _, r := range existingRoles {
		if r.Name != nil && *r.Name == role {
			log.Info("Role already exists", "user", username, "role", role)
			return nil
		}
	}

	realmRole, _, err := kClientV2.Roles.GetRealmRole(ctx, realmName, role)
	if err != nil {
		return fmt.Errorf("unable to get realm role: %w", err)
	}

	if _, err := kClientV2.Users.AddUserRealmRoles(ctx, realmName, *user.Id, []keycloakv2.RoleRepresentation{*realmRole}); err != nil {
		return fmt.Errorf("unable to add realm role to user: %w", err)
	}

	return nil
}

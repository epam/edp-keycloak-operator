package chain

import (
	"context"
	"fmt"

	keycloakApi "github.com/epam/edp-keycloak-operator/api/v1"
	"github.com/epam/edp-keycloak-operator/internal/controller/keycloakrealm/chain/handler"
	"github.com/epam/edp-keycloak-operator/pkg/client/keycloak"
	"github.com/epam/edp-keycloak-operator/pkg/client/keycloak/dto"
	keycloakv2 "github.com/epam/edp-keycloak-operator/pkg/client/keycloakv2"
)

type PutUsersRoles struct {
	next handler.RealmHandler
}

func (h PutUsersRoles) ServeRequest(ctx context.Context, realm *keycloakApi.KeycloakRealm, kClient keycloak.Client, kClientV2 *keycloakv2.KeycloakClient) error {
	rLog := log.WithValues("keycloak users", realm.Spec.Users)
	rLog.Info("Start putting roles to users")

	rDto := dto.ConvertSpecToRealm(&realm.Spec)

	err := putRolesToUsers(ctx, rDto, kClient)
	if err != nil {
		return fmt.Errorf("error during putRolesToUsers: %w", err)
	}

	rLog.Info("End put role to users")

	return nextServeOrNil(ctx, h.next, realm, kClient, kClientV2)
}

func putRolesToUsers(ctx context.Context, realm *dto.Realm, kClient keycloak.Client) error {
	for _, user := range realm.Users {
		err := putRolesToOneUser(ctx, realm, &user, kClient)
		if err != nil {
			return fmt.Errorf("error during putRolesToOneUser: %w", err)
		}
	}

	return nil
}

func putRolesToOneUser(ctx context.Context, realm *dto.Realm, user *dto.User, kClient keycloak.Client) error {
	for _, role := range user.RealmRoles {
		if err := putOneRealmRoleToOneUser(ctx, realm, user, role, kClient); err != nil {
			return fmt.Errorf("error during putOneRoleToOneUser: %w", err)
		}
	}

	return nil
}

func putOneRealmRoleToOneUser(ctx context.Context, realm *dto.Realm, user *dto.User, role string, kClient keycloak.Client) error {
	exist, err := kClient.HasUserRealmRole(realm.Name, user, role)
	if err != nil {
		return fmt.Errorf("error during check of client role: %w", err)
	}

	if exist {
		log.Info("Role already exists", "user", user, "role", role)
		return nil
	}

	if err := kClient.AddRealmRoleToUser(ctx, realm.Name, user.Username, role); err != nil {
		return fmt.Errorf("unable to add realm role to user: %w", err)
	}

	return nil
}

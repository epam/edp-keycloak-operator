package chain

import (
	"context"
	"fmt"

	"k8s.io/utils/ptr"

	keycloakApi "github.com/epam/edp-keycloak-operator/api/v1"
	"github.com/epam/edp-keycloak-operator/internal/controller/keycloakrealm/chain/handler"
	keycloakv2 "github.com/epam/edp-keycloak-operator/pkg/client/keycloakv2"
)

type PutUsers struct {
	next handler.RealmHandler
}

func (h PutUsers) ServeRequest(ctx context.Context, realm *keycloakApi.KeycloakRealm, kClientV2 *keycloakv2.KeycloakClient) error {
	rLog := log.WithValues("keycloak users", realm.Spec.Users)
	rLog.Info("Start putting users to realm")

	if err := createUsers(ctx, realm.Spec.RealmName, realm.Spec.Users, kClientV2); err != nil {
		return fmt.Errorf("error during createUsers: %w", err)
	}

	rLog.Info("End put users to realm")

	return nextServeOrNil(ctx, h.next, realm, kClientV2)
}

func createUsers(ctx context.Context, realmName string, users []keycloakApi.User, kClientV2 *keycloakv2.KeycloakClient) error {
	for _, user := range users {
		if err := createOneUser(ctx, realmName, user.Username, kClientV2); err != nil {
			return fmt.Errorf("error during createOneUser: %w", err)
		}
	}

	return nil
}

func createOneUser(ctx context.Context, realmName, username string, kClientV2 *keycloakv2.KeycloakClient) error {
	existing, _, err := kClientV2.Users.FindUserByUsername(ctx, realmName, username)
	if err != nil {
		return fmt.Errorf("error during exist realm user check: %w", err)
	}

	if existing != nil {
		log.Info("User already exists", "user", username)
		return nil
	}

	if _, err := kClientV2.Users.CreateUser(ctx, realmName, keycloakv2.UserRepresentation{
		Username: &username,
		Email:    &username,
		Enabled:  ptr.To(true),
	}); err != nil {
		return fmt.Errorf("unable to create user in realm: %w", err)
	}

	return nil
}

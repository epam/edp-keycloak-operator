package chain

import (
	"context"
	"fmt"

	"k8s.io/utils/ptr"

	keycloakApi "github.com/epam/edp-keycloak-operator/api/v1"
	"github.com/epam/edp-keycloak-operator/internal/controller/keycloakrealm/chain/handler"
	"github.com/epam/edp-keycloak-operator/pkg/client/keycloakapi"
)

type PutUsers struct {
	next handler.RealmHandler
}

func (h PutUsers) ServeRequest(ctx context.Context, realm *keycloakApi.KeycloakRealm, kClient *keycloakapi.KeycloakClient) error {
	rLog := log.WithValues("keycloak users", realm.Spec.Users)
	rLog.Info("Start putting users to realm")

	if err := createUsers(ctx, realm.Spec.RealmName, realm.Spec.Users, kClient); err != nil {
		return fmt.Errorf("error during createUsers: %w", err)
	}

	rLog.Info("End put users to realm")

	return nextServeOrNil(ctx, h.next, realm, kClient)
}

func createUsers(ctx context.Context, realmName string, users []keycloakApi.User, kClient *keycloakapi.KeycloakClient) error {
	for _, user := range users {
		if err := createOneUser(ctx, realmName, user.Username, kClient); err != nil {
			return fmt.Errorf("error during createOneUser: %w", err)
		}
	}

	return nil
}

func createOneUser(ctx context.Context, realmName, username string, kClient *keycloakapi.KeycloakClient) error {
	_, _, err := kClient.Users.FindUserByUsername(ctx, realmName, username)
	if err != nil && !keycloakapi.IsNotFound(err) {
		return fmt.Errorf("error during exist realm user check: %w", err)
	}

	if err == nil {
		log.Info("User already exists", "user", username)
		return nil
	}

	if _, err := kClient.Users.CreateUser(ctx, realmName, keycloakapi.UserRepresentation{
		Username: &username,
		Email:    &username,
		Enabled:  ptr.To(true),
	}); err != nil {
		return fmt.Errorf("unable to create user in realm: %w", err)
	}

	return nil
}

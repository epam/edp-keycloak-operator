package chain

import (
	"context"
	"fmt"

	"github.com/pkg/errors"

	keycloakApi "github.com/epam/edp-keycloak-operator/api/v1"
	"github.com/epam/edp-keycloak-operator/pkg/client/keycloak"
)

type PutClientScope struct {
	keycloakApiClient keycloak.Client
}

func NewPutClientScope(keycloakApiClient keycloak.Client) *PutClientScope {
	return &PutClientScope{keycloakApiClient: keycloakApiClient}
}

func (el *PutClientScope) Serve(ctx context.Context, keycloakClient *keycloakApi.KeycloakClient, realmName string) error {
	if err := el.putClientScope(ctx, keycloakClient, realmName); err != nil {
		return errors.Wrap(err, "error during putClientScope")
	}

	return nil
}

func (el *PutClientScope) putClientScope(ctx context.Context, keycloakClient *keycloakApi.KeycloakClient, realmName string) error {
	if err := el.putDefaultClientScope(ctx, keycloakClient, realmName); err != nil {
		return err
	}

	if err := el.putOptionalClientScope(ctx, keycloakClient, realmName); err != nil {
		return err
	}

	return nil
}

func (el *PutClientScope) putDefaultClientScope(ctx context.Context, keycloakClient *keycloakApi.KeycloakClient, realmName string) error {
	kCloakSpec := keycloakClient.Spec

	if len(kCloakSpec.DefaultClientScopes) == 0 {
		return nil
	}

	defaultScopes, err := el.keycloakApiClient.GetClientScopesByNames(ctx, realmName, kCloakSpec.DefaultClientScopes)
	if err != nil {
		return errors.Wrap(err, "error during GetClientScope")
	}

	err = el.keycloakApiClient.AddDefaultScopeToClient(ctx, realmName, kCloakSpec.ClientId, defaultScopes)
	if err != nil {
		return fmt.Errorf("failed to add default scope to client %s: %w", keycloakClient.Name, err)
	}

	return nil
}

func (el *PutClientScope) putOptionalClientScope(ctx context.Context, keycloakClient *keycloakApi.KeycloakClient, realmName string) error {
	kCloakSpec := keycloakClient.Spec

	if len(kCloakSpec.OptionalClientScopes) == 0 {
		return nil
	}

	optionalScopes, err := el.keycloakApiClient.GetClientScopesByNames(ctx, realmName, kCloakSpec.OptionalClientScopes)
	if err != nil {
		return errors.Wrap(err, "error during GetClientScope")
	}

	err = el.keycloakApiClient.AddOptionalScopeToClient(ctx, realmName, kCloakSpec.ClientId, optionalScopes)
	if err != nil {
		return fmt.Errorf("failed to add default scope to client %s: %w", keycloakClient.Name, err)
	}

	return nil
}

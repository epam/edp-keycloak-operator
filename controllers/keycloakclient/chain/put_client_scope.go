package chain

import (
	"context"
	"fmt"

	"github.com/pkg/errors"

	keycloakApi "github.com/epam/edp-keycloak-operator/api/v1"
	"github.com/epam/edp-keycloak-operator/pkg/client/keycloak"
)

type PutClientScope struct {
	BaseElement
	next Element
}

func (el *PutClientScope) Serve(ctx context.Context, keycloakClient *keycloakApi.KeycloakClient, adapterClient keycloak.Client, realmName string) error {
	if err := el.putClientScope(ctx, keycloakClient, adapterClient, realmName); err != nil {
		return errors.Wrap(err, "error during putClientScope")
	}

	return el.NextServeOrNil(ctx, el.next, keycloakClient, adapterClient, realmName)
}

func (el *PutClientScope) putClientScope(ctx context.Context, keycloakClient *keycloakApi.KeycloakClient, adapterClient keycloak.Client, realmName string) error {
	kCloakSpec := keycloakClient.Spec
	if len(kCloakSpec.DefaultClientScopes) == 0 {
		return nil
	}

	scopes, err := adapterClient.GetClientScopesByNames(ctx, realmName, kCloakSpec.DefaultClientScopes)
	if err != nil {
		return errors.Wrap(err, "error during GetClientScope")
	}

	err = adapterClient.AddDefaultScopeToClient(ctx, realmName, kCloakSpec.ClientId, scopes)
	if err != nil {
		return fmt.Errorf("failed to add default scope to client %s: %w", keycloakClient.Name, err)
	}

	return nil
}

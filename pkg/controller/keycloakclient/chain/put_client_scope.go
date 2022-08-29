package chain

import (
	"context"
	"fmt"

	"github.com/pkg/errors"

	keycloakApi "github.com/epam/edp-keycloak-operator/pkg/apis/v1/v1"
	"github.com/epam/edp-keycloak-operator/pkg/client/keycloak"
)

type PutClientScope struct {
	BaseElement
	next Element
}

func (el *PutClientScope) Serve(ctx context.Context, keycloakClient *keycloakApi.KeycloakClient, adapterClient keycloak.Client) error {
	if err := el.putClientScope(ctx, keycloakClient, adapterClient); err != nil {
		return errors.Wrap(err, "error during putClientScope")
	}

	return el.NextServeOrNil(ctx, el.next, keycloakClient, adapterClient)
}

func (el *PutClientScope) putClientScope(ctx context.Context, keycloakClient *keycloakApi.KeycloakClient, adapterClient keycloak.Client) error {
	kCloakSpec := keycloakClient.Spec
	if len(kCloakSpec.DefaultClientScopes) == 0 {
		return nil
	}

	scopes, err := adapterClient.GetClientScopesByNames(ctx, kCloakSpec.TargetRealm, kCloakSpec.DefaultClientScopes)
	if err != nil {
		return errors.Wrap(err, "error during GetClientScope")
	}

	err = adapterClient.AddDefaultScopeToClient(ctx, kCloakSpec.TargetRealm, kCloakSpec.ClientId, scopes)
	if err != nil {
		return fmt.Errorf("failed to add default scope to client %s: %w", keycloakClient.Name, err)
	}

	return nil
}

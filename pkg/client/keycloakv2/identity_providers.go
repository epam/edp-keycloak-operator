package keycloakv2

import (
	"context"

	"github.com/epam/edp-keycloak-operator/pkg/client/keycloakv2/generated"
)

type IdentityProviderRepresentation = generated.IdentityProviderRepresentation

type identityProvidersClient struct {
	client generated.ClientWithResponsesInterface
}

var _ IdentityProvidersClient = (*identityProvidersClient)(nil)

func (c *identityProvidersClient) CreateIdentityProvider(
	ctx context.Context,
	realm string,
	idp IdentityProviderRepresentation,
) (*Response, error) {
	res, err := c.client.PostAdminRealmsRealmIdentityProviderInstancesWithResponse(ctx, realm, idp)
	if err != nil {
		return nil, err
	}

	if res == nil {
		return nil, ErrNilResponse
	}

	response := &Response{HTTPResponse: res.HTTPResponse, Body: res.Body}

	if err := checkResponseError(res.HTTPResponse, res.Body); err != nil {
		return response, err
	}

	return response, nil
}

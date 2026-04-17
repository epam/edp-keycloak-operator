package keycloakapi

import (
	"context"

	"github.com/epam/edp-keycloak-operator/pkg/client/keycloakapi/generated"
)

type GlobalRequestResult = generated.GlobalRequestResult

// SessionsClient defines operations for managing Keycloak realm sessions.
type SessionsClient interface {
	// GetRealmSessionStats returns per-client active and offline session counts for a realm.
	GetRealmSessionStats(ctx context.Context, realm string) ([]map[string]string, *Response, error)
	// LogoutAllSessions terminates all user sessions in a realm.
	LogoutAllSessions(ctx context.Context, realm string) (*Response, error)
}

type sessionsClient struct {
	client generated.ClientWithResponsesInterface
}

var _ SessionsClient = (*sessionsClient)(nil)

func (c *sessionsClient) GetRealmSessionStats(
	ctx context.Context,
	realm string,
) ([]map[string]string, *Response, error) {
	res, err := c.client.GetAdminRealmsRealmClientSessionStatsWithResponse(ctx, realm)
	if err != nil {
		return nil, nil, err
	}

	if res == nil {
		return nil, nil, ErrNilResponse
	}

	response := &Response{HTTPResponse: res.HTTPResponse, Body: res.Body}

	if err := checkResponseError(res.HTTPResponse, res.Body); err != nil {
		return nil, response, err
	}

	if res.JSON200 == nil {
		return nil, response, nil
	}

	return *res.JSON200, response, nil
}

func (c *sessionsClient) LogoutAllSessions(
	ctx context.Context,
	realm string,
) (*Response, error) {
	res, err := c.client.PostAdminRealmsRealmLogoutAllWithResponse(ctx, realm)
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

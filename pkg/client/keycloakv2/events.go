package keycloakv2

import (
	"context"

	"github.com/epam/edp-keycloak-operator/pkg/client/keycloakv2/generated"
)

type (
	EventRepresentation      = generated.EventRepresentation
	AdminEventRepresentation = generated.AdminEventRepresentation
	GetEventsParams          = generated.GetAdminRealmsRealmEventsParams
	GetAdminEventsParams     = generated.GetAdminRealmsRealmAdminEventsParams
)

// EventsClient defines operations for querying and managing Keycloak realm events
// and admin events, as well as brute-force attack detection status.
type EventsClient interface {
	// GetEvents returns user events (login, logout, etc.) for a realm.
	GetEvents(ctx context.Context, realm string, params *GetEventsParams) ([]EventRepresentation, *Response, error)
	// GetAdminEvents returns admin audit events for a realm.
	GetAdminEvents(
		ctx context.Context, realm string, params *GetAdminEventsParams,
	) ([]AdminEventRepresentation, *Response, error)
	// DeleteEvents clears all user events for a realm.
	DeleteEvents(ctx context.Context, realm string) (*Response, error)
	// DeleteAdminEvents clears all admin events for a realm.
	DeleteAdminEvents(ctx context.Context, realm string) (*Response, error)
	// GetEventsConfig returns the events configuration for a realm.
	GetEventsConfig(ctx context.Context, realm string) (*RealmEventsConfigRepresentation, *Response, error)
	// GetBruteForceStatus returns the brute-force detection status for a user.
	GetBruteForceStatus(ctx context.Context, realm, userID string) (map[string]any, *Response, error)
	// ClearBruteForceForUser clears brute-force lockout for a specific user.
	ClearBruteForceForUser(ctx context.Context, realm, userID string) (*Response, error)
	// ClearAllBruteForce clears all brute-force lockouts for a realm.
	ClearAllBruteForce(ctx context.Context, realm string) (*Response, error)
}

type eventsClient struct {
	client generated.ClientWithResponsesInterface
}

var _ EventsClient = (*eventsClient)(nil)

func (c *eventsClient) GetEvents(
	ctx context.Context,
	realm string,
	params *GetEventsParams,
) ([]EventRepresentation, *Response, error) {
	res, err := c.client.GetAdminRealmsRealmEventsWithResponse(ctx, realm, params)
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

func (c *eventsClient) GetAdminEvents(
	ctx context.Context,
	realm string,
	params *GetAdminEventsParams,
) ([]AdminEventRepresentation, *Response, error) {
	res, err := c.client.GetAdminRealmsRealmAdminEventsWithResponse(ctx, realm, params)
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

func (c *eventsClient) DeleteEvents(ctx context.Context, realm string) (*Response, error) {
	res, err := c.client.DeleteAdminRealmsRealmEventsWithResponse(ctx, realm)
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

func (c *eventsClient) DeleteAdminEvents(ctx context.Context, realm string) (*Response, error) {
	res, err := c.client.DeleteAdminRealmsRealmAdminEventsWithResponse(ctx, realm)
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

func (c *eventsClient) GetEventsConfig(
	ctx context.Context,
	realm string,
) (*RealmEventsConfigRepresentation, *Response, error) {
	res, err := c.client.GetAdminRealmsRealmEventsConfigWithResponse(ctx, realm)
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

	return res.JSON200, response, nil
}

func (c *eventsClient) GetBruteForceStatus(
	ctx context.Context,
	realm, userID string,
) (map[string]any, *Response, error) {
	res, err := c.client.GetAdminRealmsRealmAttackDetectionBruteForceUsersUserIdWithResponse(ctx, realm, userID)
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

func (c *eventsClient) ClearBruteForceForUser(ctx context.Context, realm, userID string) (*Response, error) {
	res, err := c.client.DeleteAdminRealmsRealmAttackDetectionBruteForceUsersUserIdWithResponse(ctx, realm, userID)
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

func (c *eventsClient) ClearAllBruteForce(ctx context.Context, realm string) (*Response, error) {
	res, err := c.client.DeleteAdminRealmsRealmAttackDetectionBruteForceUsersWithResponse(ctx, realm)
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

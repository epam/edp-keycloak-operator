package keycloakv2

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/epam/edp-keycloak-operator/pkg/client/keycloakv2/generated"
)

// ErrNilResponse is returned when the Keycloak API returns a nil response.
var ErrNilResponse = errors.New("nil response from Keycloak")

type ApiError struct {
	Code         int
	Message      string
	Body         []byte
	HTTPResponse *http.Response
}

func (e *ApiError) Error() string {
	return e.Message
}

// IsClientError returns true if the error is a 4xx client error
func (e *ApiError) IsClientError() bool {
	return e.Code >= 400 && e.Code < 500
}

// IsServerError returns true if the error is a 5xx server error
func (e *ApiError) IsServerError() bool {
	return e.Code >= 500
}

// IsNotFound returns true if the error is a 404 Not Found
func (e *ApiError) IsNotFound() bool {
	return e.Code == http.StatusNotFound
}

// IsConflict returns true if the error is a 409 Conflict
func (e *ApiError) IsConflict() bool {
	return e.Code == http.StatusConflict
}

// parseKeycloakError parses Keycloak error responses with a three-tier fallback strategy
func parseKeycloakError(statusCode int, body []byte) *ApiError {
	// 1. Try parsing as ErrorRepresentation (complex format)
	var errRep generated.ErrorRepresentation
	if json.Unmarshal(body, &errRep) == nil && errRep.ErrorMessage != nil {
		return &ApiError{
			Code:    statusCode,
			Message: *errRep.ErrorMessage,
			Body:    body,
		}
	}

	// 2. Try parsing as simple {"error": "message"} format
	var simpleErr struct {
		Error string `json:"error"`
	}

	if json.Unmarshal(body, &simpleErr) == nil && simpleErr.Error != "" {
		return &ApiError{
			Code:    statusCode,
			Message: simpleErr.Error,
			Body:    body,
		}
	}

	// 3. Fallback to HTTP status text with raw body
	message := http.StatusText(statusCode)
	if len(body) > 0 && len(body) < 1024 { // Only include body if it's reasonably small
		message += ": " + string(body)
	}

	return &ApiError{
		Code:    statusCode,
		Message: message,
		Body:    body,
	}
}

// checkResponseError checks if an HTTP response indicates an error and returns an ApiError if so
func checkResponseError(httpResp *http.Response, body []byte) error {
	if httpResp == nil {
		return nil
	}

	// Only 2xx status codes are success
	if httpResp.StatusCode >= 200 && httpResp.StatusCode < 300 {
		return nil
	}

	apiErr := parseKeycloakError(httpResp.StatusCode, body)
	apiErr.HTTPResponse = httpResp

	return apiErr
}

// Helper functions for error checking

// IsNotFound returns true if the error is a 404 Not Found ApiError
func IsNotFound(err error) bool {
	var apiErr *ApiError
	if errors.As(err, &apiErr) {
		return apiErr.IsNotFound()
	}

	return false
}

// IsConflict returns true if the error is a 409 Conflict ApiError
func IsConflict(err error) bool {
	var apiErr *ApiError
	if errors.As(err, &apiErr) {
		return apiErr.IsConflict()
	}

	return false
}

// IsClientError returns true if the error is a 4xx client error ApiError
func IsClientError(err error) bool {
	var apiErr *ApiError
	if errors.As(err, &apiErr) {
		return apiErr.IsClientError()
	}

	return false
}

// IsServerError returns true if the error is a 5xx server error ApiError
func IsServerError(err error) bool {
	var apiErr *ApiError
	if errors.As(err, &apiErr) {
		return apiErr.IsServerError()
	}

	return false
}

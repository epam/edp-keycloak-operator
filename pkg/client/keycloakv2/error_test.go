package keycloakv2

import (
	"errors"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseKeycloakError_ErrorRepresentation(t *testing.T) {
	// Test parsing complex ErrorRepresentation format
	body := []byte(`{"errorMessage":"Invalid realm configuration","field":"realmName"}`)
	apiErr := parseKeycloakError(http.StatusBadRequest, body)

	assert.Equal(t, http.StatusBadRequest, apiErr.Code)
	assert.Equal(t, "Invalid realm configuration", apiErr.Message)
	assert.Equal(t, body, apiErr.Body)
}

func TestParseKeycloakError_SimpleFormat(t *testing.T) {
	// Test parsing simple {"error":"message"} format (common Keycloak format)
	body := []byte(`{"error":"Realm not found."}`)
	apiErr := parseKeycloakError(http.StatusNotFound, body)

	assert.Equal(t, http.StatusNotFound, apiErr.Code)
	assert.Equal(t, "Realm not found.", apiErr.Message)
	assert.Equal(t, body, apiErr.Body)
}

func TestParseKeycloakError_Fallback(t *testing.T) {
	// Test fallback to HTTP status text for non-JSON responses
	body := []byte("Internal Server Error")
	apiErr := parseKeycloakError(http.StatusInternalServerError, body)

	assert.Equal(t, http.StatusInternalServerError, apiErr.Code)
	assert.Contains(t, apiErr.Message, "Internal Server Error")
	assert.Equal(t, body, apiErr.Body)
}

func TestParseKeycloakError_EmptyBody(t *testing.T) {
	// Test with empty body
	body := []byte{}
	apiErr := parseKeycloakError(http.StatusNotFound, body)

	assert.Equal(t, http.StatusNotFound, apiErr.Code)
	assert.Equal(t, "Not Found", apiErr.Message)
	assert.Equal(t, body, apiErr.Body)
}

func TestParseKeycloakError_LargeBody(t *testing.T) {
	// Test with large body (should not include body in message)
	body := make([]byte, 2000)
	for i := range body {
		body[i] = 'x'
	}

	apiErr := parseKeycloakError(http.StatusBadRequest, body)

	assert.Equal(t, http.StatusBadRequest, apiErr.Code)
	assert.Equal(t, "Bad Request", apiErr.Message) // Should not include body
	assert.Equal(t, body, apiErr.Body)
}

func TestCheckResponseError_Success(t *testing.T) {
	// Test that 2xx status codes return no error
	resp := &http.Response{StatusCode: http.StatusOK}
	body := []byte(`{"success":true}`)

	err := checkResponseError(resp, body)
	assert.NoError(t, err)

	resp = &http.Response{StatusCode: http.StatusCreated}
	err = checkResponseError(resp, body)
	assert.NoError(t, err)

	resp = &http.Response{StatusCode: http.StatusNoContent}
	err = checkResponseError(resp, body)
	assert.NoError(t, err)
}

func TestCheckResponseError_ClientError(t *testing.T) {
	// Test 4xx client errors
	resp := &http.Response{StatusCode: http.StatusBadRequest}
	body := []byte(`{"error":"Invalid request"}`)

	err := checkResponseError(resp, body)
	require.Error(t, err)

	var apiErr *ApiError

	require.ErrorAs(t, err, &apiErr)
	assert.Equal(t, http.StatusBadRequest, apiErr.Code)
	assert.Contains(t, apiErr.Message, "Invalid request")
	assert.Equal(t, resp, apiErr.HTTPResponse)
}

func TestCheckResponseError_ServerError(t *testing.T) {
	// Test 5xx server errors
	resp := &http.Response{StatusCode: http.StatusInternalServerError}
	body := []byte(`{"error":"Internal error"}`)

	err := checkResponseError(resp, body)
	require.Error(t, err)

	var apiErr *ApiError

	require.ErrorAs(t, err, &apiErr)
	assert.Equal(t, http.StatusInternalServerError, apiErr.Code)
	assert.True(t, apiErr.IsServerError())
}

func TestCheckResponseError_NilResponse(t *testing.T) {
	// Test that nil response returns no error
	err := checkResponseError(nil, []byte{})
	assert.NoError(t, err)
}

func TestApiError_Helpers(t *testing.T) {
	t.Run("IsClientError", func(t *testing.T) {
		apiErr := &ApiError{Code: http.StatusBadRequest}
		assert.True(t, apiErr.IsClientError())

		apiErr = &ApiError{Code: http.StatusNotFound}
		assert.True(t, apiErr.IsClientError())

		apiErr = &ApiError{Code: http.StatusInternalServerError}
		assert.False(t, apiErr.IsClientError())
	})

	t.Run("IsServerError", func(t *testing.T) {
		apiErr := &ApiError{Code: http.StatusInternalServerError}
		assert.True(t, apiErr.IsServerError())

		apiErr = &ApiError{Code: http.StatusBadGateway}
		assert.True(t, apiErr.IsServerError())

		apiErr = &ApiError{Code: http.StatusBadRequest}
		assert.False(t, apiErr.IsServerError())
	})

	t.Run("IsNotFound", func(t *testing.T) {
		apiErr := &ApiError{Code: http.StatusNotFound}
		assert.True(t, apiErr.IsNotFound())

		apiErr = &ApiError{Code: http.StatusBadRequest}
		assert.False(t, apiErr.IsNotFound())
	})

	t.Run("IsConflict", func(t *testing.T) {
		apiErr := &ApiError{Code: http.StatusConflict}
		assert.True(t, apiErr.IsConflict())

		apiErr = &ApiError{Code: http.StatusBadRequest}
		assert.False(t, apiErr.IsConflict())
	})

	t.Run("Error", func(t *testing.T) {
		apiErr := &ApiError{
			Code:    http.StatusNotFound,
			Message: "Resource not found",
		}
		assert.Equal(t, "Resource not found", apiErr.Error())
	})
}

func TestHelperFunctions(t *testing.T) {
	t.Run("IsNotFound", func(t *testing.T) {
		// Test with ApiError
		apiErr := &ApiError{Code: http.StatusNotFound}
		assert.True(t, IsNotFound(apiErr))

		// Test with non-404 ApiError
		apiErr = &ApiError{Code: http.StatusBadRequest}
		assert.False(t, IsNotFound(apiErr))

		// Test with non-ApiError
		otherErr := errors.New("some other error")
		assert.False(t, IsNotFound(otherErr))

		// Test with nil
		assert.False(t, IsNotFound(nil))
	})

	t.Run("IsConflict", func(t *testing.T) {
		// Test with ApiError
		apiErr := &ApiError{Code: http.StatusConflict}
		assert.True(t, IsConflict(apiErr))

		// Test with non-409 ApiError
		apiErr = &ApiError{Code: http.StatusBadRequest}
		assert.False(t, IsConflict(apiErr))

		// Test with non-ApiError
		otherErr := errors.New("some other error")
		assert.False(t, IsConflict(otherErr))

		// Test with nil
		assert.False(t, IsConflict(nil))
	})

	t.Run("IsClientError", func(t *testing.T) {
		// Test with 4xx ApiError
		apiErr := &ApiError{Code: http.StatusBadRequest}
		assert.True(t, IsClientError(apiErr))

		apiErr = &ApiError{Code: http.StatusNotFound}
		assert.True(t, IsClientError(apiErr))

		// Test with 5xx ApiError
		apiErr = &ApiError{Code: http.StatusInternalServerError}
		assert.False(t, IsClientError(apiErr))

		// Test with non-ApiError
		otherErr := errors.New("some other error")
		assert.False(t, IsClientError(otherErr))

		// Test with nil
		assert.False(t, IsClientError(nil))
	})

	t.Run("IsServerError", func(t *testing.T) {
		// Test with 5xx ApiError
		apiErr := &ApiError{Code: http.StatusInternalServerError}
		assert.True(t, IsServerError(apiErr))

		apiErr = &ApiError{Code: http.StatusBadGateway}
		assert.True(t, IsServerError(apiErr))

		// Test with 4xx ApiError
		apiErr = &ApiError{Code: http.StatusBadRequest}
		assert.False(t, IsServerError(apiErr))

		// Test with non-ApiError
		otherErr := errors.New("some other error")
		assert.False(t, IsServerError(otherErr))

		// Test with nil
		assert.False(t, IsServerError(nil))
	})
}

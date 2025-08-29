package adapter

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestGoCloakAdapter_GetIdentityProvider(t *testing.T) {
	testCases := []struct {
		name         string
		alias        string
		status       int
		response     string
		expectError  bool
		errorMessage string
	}{
		{
			name:        "success",
			alias:       "alias1",
			status:      200,
			response:    "",
			expectError: false,
		},
		{
			name:        "not found",
			alias:       "alias2",
			status:      404,
			response:    "",
			expectError: true,
		},
		{
			name:         "server error",
			alias:        "alias3",
			status:       500,
			response:     "fatal",
			expectError:  true,
			errorMessage: "unable to get idp: status: 500 Internal Server Error, body: fatal",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				expectedPath := strings.Replace(
					strings.Replace(identityProviderEntity, "{realm}", "realm1", 1), "{alias}", tc.alias, 1)
				if r.URL.Path == expectedPath {
					w.WriteHeader(tc.status)
					_, _ = w.Write([]byte(tc.response))
				}
			}))
			defer server.Close()

			kc, _, restyClient := initAdapter(t, nil)
			restyClient.SetBaseURL(server.URL)

			_, err := kc.GetIdentityProvider(context.Background(), "realm1", tc.alias)

			if tc.expectError {
				require.Error(t, err)

				if tc.errorMessage != "" && tc.alias == "alias3" {
					require.Equal(t, tc.errorMessage, err.Error())
				} else if tc.alias == "alias2" {
					require.True(t, IsErrNotFound(err))
				}
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestGoCloakAdapter_CreateIdentityProvider(t *testing.T) {
	testCases := []struct {
		name         string
		realm        string
		status       int
		response     string
		expectError  bool
		errorMessage string
	}{
		{
			name:        "success",
			realm:       "realm1",
			status:      200,
			response:    "",
			expectError: false,
		},
		{
			name:         "server error",
			realm:        "realm2",
			status:       500,
			response:     "fatal",
			expectError:  true,
			errorMessage: "unable to create idp: status: 500 Internal Server Error, body: fatal",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				expectedPath := strings.Replace(idPResource, "{realm}", tc.realm, 1)
				if r.URL.Path == expectedPath && r.Method == http.MethodPost {
					w.WriteHeader(tc.status)
					_, _ = w.Write([]byte(tc.response))
				}
			}))
			defer server.Close()

			kc, _, restyClient := initAdapter(t, nil)
			restyClient.SetBaseURL(server.URL)

			err := kc.CreateIdentityProvider(context.Background(), tc.realm, &IdentityProvider{})

			if tc.expectError {
				require.Error(t, err)

				if tc.errorMessage != "" {
					require.Equal(t, tc.errorMessage, err.Error())
				}
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestGoCloakAdapter_UpdateIdentityProvider(t *testing.T) {
	testCases := []struct {
		name         string
		alias        string
		status       int
		response     string
		expectError  bool
		errorMessage string
	}{
		{
			name:        "success",
			alias:       "alias1",
			status:      200,
			response:    "",
			expectError: false,
		},
		{
			name:         "server error",
			alias:        "alias2",
			status:       500,
			response:     "fatal",
			expectError:  true,
			errorMessage: "unable to update idp: status: 500 Internal Server Error, body: fatal",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				expectedPath := strings.Replace(
					strings.Replace(identityProviderEntity, "{realm}", "realm1", 1), "{alias}", tc.alias, 1)
				if r.URL.Path == expectedPath && r.Method == http.MethodPut {
					w.WriteHeader(tc.status)
					_, _ = w.Write([]byte(tc.response))
				}
			}))
			defer server.Close()

			kc, _, restyClient := initAdapter(t, nil)
			restyClient.SetBaseURL(server.URL)

			err := kc.UpdateIdentityProvider(context.Background(), "realm1", &IdentityProvider{Alias: tc.alias})

			if tc.expectError {
				require.Error(t, err)

				if tc.errorMessage != "" {
					require.Equal(t, tc.errorMessage, err.Error())
				}
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestGoCloakAdapter_DeleteIdentityProvider(t *testing.T) {
	testCases := []struct {
		name         string
		alias        string
		status       int
		response     string
		expectError  bool
		errorMessage string
	}{
		{
			name:        "success",
			alias:       "alias1",
			status:      200,
			response:    "",
			expectError: false,
		},
		{
			name:         "server error",
			alias:        "alias2",
			status:       500,
			response:     "fatal",
			expectError:  true,
			errorMessage: "unable to delete idp: status: 500 Internal Server Error, body: fatal",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				expectedPath := strings.Replace(
					strings.Replace(identityProviderEntity, "{realm}", "realm1", 1), "{alias}", tc.alias, 1)
				if r.URL.Path == expectedPath && r.Method == http.MethodDelete {
					w.WriteHeader(tc.status)
					_, _ = w.Write([]byte(tc.response))
				}
			}))
			defer server.Close()

			kc, _, restyClient := initAdapter(t, nil)
			restyClient.SetBaseURL(server.URL)

			err := kc.DeleteIdentityProvider(context.Background(), "realm1", tc.alias)

			if tc.expectError {
				require.Error(t, err)

				if tc.errorMessage != "" {
					require.Equal(t, tc.errorMessage, err.Error())
				}
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestGoCloakAdapter_CreateIDPMapper(t *testing.T) {
	testCases := []struct {
		name         string
		alias        string
		status       int
		response     string
		location     string
		expectError  bool
		errorMessage string
	}{
		{
			name:        "success",
			alias:       "alias1",
			status:      200,
			response:    "",
			location:    "id/new-id",
			expectError: false,
		},
		{
			name:         "server error",
			alias:        "alias2",
			status:       500,
			response:     "fatal",
			expectError:  true,
			errorMessage: "unable to create idp mapper: status: 500 Internal Server Error, body: fatal",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				expectedPath := strings.Replace(
					strings.Replace(idpMapperCreateList, "{realm}", "realm1", 1), "{alias}", tc.alias, 1)
				if r.URL.Path == expectedPath && r.Method == http.MethodPost {
					if tc.location != "" {
						w.Header().Set("Location", tc.location)
					}

					w.WriteHeader(tc.status)
					_, _ = w.Write([]byte(tc.response))
				}
			}))
			defer server.Close()

			kc, _, restyClient := initAdapter(t, nil)
			restyClient.SetBaseURL(server.URL)

			_, err := kc.CreateIDPMapper(context.Background(), "realm1", tc.alias, &IdentityProviderMapper{})

			if tc.expectError {
				require.Error(t, err)

				if tc.errorMessage != "" {
					require.Equal(t, tc.errorMessage, err.Error())
				}
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestGoCloakAdapter_UpdateIDPMapper(t *testing.T) {
	testCases := []struct {
		name         string
		alias        string
		status       int
		response     string
		expectError  bool
		errorMessage string
	}{
		{
			name:        "success",
			alias:       "alias1",
			status:      200,
			response:    "",
			expectError: false,
		},
		{
			name:         "server error",
			alias:        "alias2",
			status:       500,
			response:     "fatal",
			expectError:  true,
			errorMessage: "unable to update idp mapper: status: 500 Internal Server Error, body: fatal",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				expectedPath := strings.Replace(
					strings.Replace(
						strings.Replace(idpMapperEntity, "{realm}", "realm1", 1), "{alias}", tc.alias, 1), "{id}", "id11", 1)
				if r.URL.Path == expectedPath && r.Method == http.MethodPut {
					w.WriteHeader(tc.status)
					_, _ = w.Write([]byte(tc.response))
				}
			}))
			defer server.Close()

			kc, _, restyClient := initAdapter(t, nil)
			restyClient.SetBaseURL(server.URL)

			err := kc.UpdateIDPMapper(context.Background(), "realm1", tc.alias, &IdentityProviderMapper{ID: "id11"})

			if tc.expectError {
				require.Error(t, err)

				if tc.errorMessage != "" {
					require.Equal(t, tc.errorMessage, err.Error())
				}
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestGoCloakAdapter_DeleteIDPMapper(t *testing.T) {
	testCases := []struct {
		name         string
		mapperID     string
		status       int
		response     string
		expectError  bool
		errorMessage string
	}{
		{
			name:        "success",
			mapperID:    "mapper1",
			status:      200,
			response:    "",
			expectError: false,
		},
		{
			name:         "server error",
			mapperID:     "mapper2",
			status:       500,
			response:     "fatal",
			expectError:  true,
			errorMessage: "unable to delete idp mapper: status: 500 Internal Server Error, body: fatal",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				expectedPath := strings.Replace(
					strings.Replace(
						strings.Replace(idpMapperEntity, "{realm}", "realm1", 1), "{alias}", "alias1", 1), "{id}", tc.mapperID, 1)
				if r.URL.Path == expectedPath && r.Method == http.MethodDelete {
					w.WriteHeader(tc.status)
					_, _ = w.Write([]byte(tc.response))
				}
			}))
			defer server.Close()

			kc, _, restyClient := initAdapter(t, nil)
			restyClient.SetBaseURL(server.URL)

			err := kc.DeleteIDPMapper(context.Background(), "realm1", "alias1", tc.mapperID)

			if tc.expectError {
				require.Error(t, err)

				if tc.errorMessage != "" {
					require.Equal(t, tc.errorMessage, err.Error())
				}
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestGoCloakAdapter_GetIDPMappers(t *testing.T) {
	testCases := []struct {
		name         string
		alias        string
		status       int
		response     string
		expectError  bool
		errorMessage string
	}{
		{
			name:        "success",
			alias:       "alias1",
			status:      200,
			response:    "",
			expectError: false,
		},
		{
			name:         "server error",
			alias:        "alias2",
			status:       500,
			response:     "fatal",
			expectError:  true,
			errorMessage: "unable to get idp mappers: status: 500 Internal Server Error, body: fatal",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				expectedPath := strings.Replace(
					strings.Replace(idpMapperCreateList, "{realm}", "realm1", 1), "{alias}", tc.alias, 1)
				if r.URL.Path == expectedPath && r.Method == http.MethodGet {
					w.WriteHeader(tc.status)
					_, _ = w.Write([]byte(tc.response))
				}
			}))
			defer server.Close()

			kc, _, restyClient := initAdapter(t, nil)
			restyClient.SetBaseURL(server.URL)

			_, err := kc.GetIDPMappers(context.Background(), "realm1", tc.alias)

			if tc.expectError {
				require.Error(t, err)

				if tc.errorMessage != "" {
					require.Equal(t, tc.errorMessage, err.Error())
				}
			} else {
				require.NoError(t, err)
			}
		})
	}
}

package adapter

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Nerzal/gocloak/v12"
	"github.com/go-resty/resty/v2"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/epam/edp-keycloak-operator/pkg/client/keycloak/adapter/mocks"
	logmock "github.com/epam/edp-keycloak-operator/pkg/client/keycloak/mock"
)

func initClientScopeAdapter(
	t *testing.T, server *httptest.Server) *GoCloakAdapter {
	t.Helper()

	mockClient := mocks.NewMockGoCloak(t)

	restyClient := resty.New()
	if server != nil {
		restyClient.SetBaseURL(server.URL)
	}

	mockClient.On("RestyClient").Return(restyClient).Maybe()

	logger := logmock.NewLogr()

	return &GoCloakAdapter{
		client:   mockClient,
		basePath: "",
		token:    &gocloak.JWT{AccessToken: "token"},
		log:      logger,
	}
}

func TestGoCloakAdapter_CreateClientScope(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		expectedPostPath := strings.Replace(getRealmClientScopes, "{realm}", "realm1", 1)
		expectedPutPath := strings.Replace(
			strings.Replace(putDefaultClientScope, "{realm}", "realm1", 1), "{clientScopeID}", "new-scope-id", 1)

		switch {
		case r.Method == http.MethodPost && r.URL.Path == expectedPostPath:
			w.Header().Set("Location", "id/new-scope-id")
			w.WriteHeader(http.StatusOK)
		case r.Method == http.MethodPut && r.URL.Path == expectedPutPath:
			w.WriteHeader(http.StatusOK)
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	mockClient := mocks.NewMockGoCloak(t)
	restyClient := resty.New()
	restyClient.SetBaseURL(server.URL)
	mockClient.On("RestyClient").Return(restyClient)

	adapter := GoCloakAdapter{
		client:   mockClient,
		token:    &gocloak.JWT{AccessToken: "token"},
		basePath: "",
		log:      logmock.NewLogr(),
	}

	id, err := adapter.CreateClientScope(context.Background(), "realm1",
		&ClientScope{Name: "demo", Default: true})
	require.NoError(t, err)
	require.NotEmpty(t, id)
}

func TestGoCloakAdapter_CreateClientScope_FailureSetDefault(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		expectedPostPath := strings.Replace(getRealmClientScopes, "{realm}", "realm1", 1)
		expectedPutPath := strings.Replace(
			strings.Replace(putDefaultClientScope, "{realm}", "realm1", 1), "{clientScopeID}", "new-scope-id", 1)

		switch {
		case r.Method == http.MethodPost && r.URL.Path == expectedPostPath:
			w.Header().Set("Location", "id/new-scope-id")
			w.WriteHeader(http.StatusOK)
		case r.Method == http.MethodPut && r.URL.Path == expectedPutPath:
			w.WriteHeader(http.StatusInternalServerError)
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	mockClient := mocks.NewMockGoCloak(t)
	restyClient := resty.New()
	restyClient.SetBaseURL(server.URL)
	mockClient.On("RestyClient").Return(restyClient)

	adapter := GoCloakAdapter{
		client:   mockClient,
		token:    &gocloak.JWT{AccessToken: "token"},
		basePath: "",
		log:      logmock.NewLogr(),
	}

	_, err := adapter.CreateClientScope(context.Background(), "realm1",
		&ClientScope{Name: "demo", Default: true})
	require.Error(t, err)
	require.Contains(t, err.Error(), "unable to set default client scope for realm")
}

func TestGoCloakAdapter_CreateClientScope_FailureCreate(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		expectedPostPath := strings.Replace(getRealmClientScopes, "{realm}", "realm1", 1)
		if r.Method == http.MethodPost && r.URL.Path == expectedPostPath {
			w.WriteHeader(http.StatusInternalServerError)
			_, _ = w.Write([]byte("internal error"))

			return
		}

		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	mockClient := mocks.NewMockGoCloak(t)
	restyClient := resty.New()
	restyClient.SetBaseURL(server.URL)
	mockClient.On("RestyClient").Return(restyClient)

	adapter := GoCloakAdapter{
		client: mockClient,
		token:  &gocloak.JWT{AccessToken: "token"},
	}

	_, err := adapter.CreateClientScope(context.Background(), "realm1",
		&ClientScope{Name: "demo", Default: true})

	require.Error(t, err)
	require.Contains(t, err.Error(), "unable to create client scope")
}

func TestGoCloakAdapter_CreateClientScope_FailureGetID(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		expectedPostPath := strings.Replace(getRealmClientScopes, "{realm}", "realm1", 1)
		if r.Method == http.MethodPost && r.URL.Path == expectedPostPath {
			// Don't set Location header to simulate failure
			w.WriteHeader(http.StatusOK)
			return
		}

		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	mockClient := mocks.NewMockGoCloak(t)
	restyClient := resty.New()
	restyClient.SetBaseURL(server.URL)
	mockClient.On("RestyClient").Return(restyClient)

	adapter := GoCloakAdapter{
		client: mockClient,
		token:  &gocloak.JWT{AccessToken: "token"},
	}

	_, err := adapter.CreateClientScope(context.Background(), "realm1",
		&ClientScope{Name: "demo", Default: true})

	require.Error(t, err)
	require.Contains(t, err.Error(), "location header is not set or empty")
}

func TestGoCloakAdapter_UpdateClientScope(t *testing.T) {
	var (
		realmName = "realm1"
		scopeID   = "scope1"
	)

	tests := []struct {
		name             string
		clientScope      *ClientScope
		defaultScopes    []ClientScope
		expectedRequests []string
	}{
		{
			name: "update without default",
			clientScope: &ClientScope{
				Name: "scope1",
				ProtocolMappers: []ProtocolMapper{{
					Name: "mp2",
				}},
			},
			defaultScopes: []ClientScope{},
			expectedRequests: []string{
				"PUT /admin/realms/realm1/client-scopes/scope1",
				"DELETE /admin/realms/realm1/client-scopes/scope1/protocol-mappers/mp_id1",
				"POST /admin/realms/realm1/client-scopes/scope1/protocol-mappers/models",
			},
		},
		{
			name: "update with default true",
			clientScope: &ClientScope{
				Name: "scope1",
				ProtocolMappers: []ProtocolMapper{{
					Name: "mp2",
				}},
				Default: true,
			},
			defaultScopes: []ClientScope{},
			expectedRequests: []string{
				"PUT /admin/realms/realm1/client-scopes/scope1",
				"DELETE /admin/realms/realm1/client-scopes/scope1/protocol-mappers/mp_id1",
				"POST /admin/realms/realm1/client-scopes/scope1/protocol-mappers/models",
				"GET /admin/realms/realm1/default-default-client-scopes",
				"PUT /admin/realms/realm1/default-default-client-scopes/scope1",
			},
		},
		{
			name: "update with default false",
			clientScope: &ClientScope{
				Name: "scope1",
				ProtocolMappers: []ProtocolMapper{{
					Name: "mp2",
				}},
				Default: false,
			},
			defaultScopes: []ClientScope{{Name: "scope1"}},
			expectedRequests: []string{
				"PUT /admin/realms/realm1/client-scopes/scope1",
				"DELETE /admin/realms/realm1/client-scopes/scope1/protocol-mappers/mp_id1",
				"POST /admin/realms/realm1/client-scopes/scope1/protocol-mappers/models",
				"GET /admin/realms/realm1/default-default-client-scopes",
				"DELETE /admin/realms/realm1/default-default-client-scopes/scope1",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			requestLog := make([]string, 0)

			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				requestLog = append(requestLog, r.Method+" "+r.URL.Path)

				expectedPutPath := strings.Replace(putClientScope, "{realm}", realmName, 1)
				expectedPutPath = strings.Replace(expectedPutPath, "{id}", scopeID, 1)
				expectedGetDefaultPath := strings.Replace(getDefaultClientScopes, "{realm}", "realm1", 1)

				switch {
				case r.Method == http.MethodPut && r.URL.Path == expectedPutPath:
					w.WriteHeader(http.StatusOK)
				case r.Method == http.MethodDelete && strings.Contains(r.URL.Path, "/protocol-mappers/"):
					w.WriteHeader(http.StatusOK)
				case r.Method == http.MethodPost && strings.Contains(r.URL.Path, "/protocol-mappers/models"):
					w.WriteHeader(http.StatusOK)
				case r.Method == http.MethodGet && r.URL.Path == expectedGetDefaultPath:
					setJSONContentType(w)
					w.WriteHeader(http.StatusOK)
					_ = json.NewEncoder(w).Encode(tt.defaultScopes)
				case r.Method == http.MethodPut && strings.Contains(r.URL.Path, "/default-default-client-scopes/"):
					w.WriteHeader(http.StatusOK)
				case r.Method == http.MethodDelete && strings.Contains(r.URL.Path, "/default-default-client-scopes/"):
					w.WriteHeader(http.StatusOK)
				default:
					w.WriteHeader(http.StatusNotFound)
				}
			}))
			defer server.Close()

			mockClient := mocks.NewMockGoCloak(t)
			restyClient := resty.New()
			restyClient.SetBaseURL(server.URL)
			mockClient.On("RestyClient").Return(restyClient)
			mockClient.On("GetClientScope", mock.Anything, "token", realmName, scopeID).Return(&gocloak.ClientScope{
				ID: gocloak.StringP("scope1"),
				ProtocolMappers: &[]gocloak.ProtocolMappers{
					{
						Name: gocloak.StringP("mp1"),
						ID:   gocloak.StringP("mp_id1"),
					},
				},
			}, nil)

			adapter := GoCloakAdapter{
				client:   mockClient,
				token:    &gocloak.JWT{AccessToken: "token"},
				basePath: "",
				log:      logmock.NewLogr(),
			}

			err := adapter.UpdateClientScope(context.Background(), realmName, scopeID, tt.clientScope)
			require.NoError(t, err)
		})
	}
}

func TestGoCloakAdapter_GetClientScope(t *testing.T) {
	result := []ClientScope{{Name: "name1"}}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		expectedPath := strings.Replace(getRealmClientScopes, "{realm}", "realm1", 1)
		if r.Method == http.MethodGet && r.URL.Path == expectedPath {
			setJSONContentType(w)
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(&result)

			return
		}

		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	mockClient := mocks.NewMockGoCloak(t)
	restyClient := resty.New()
	restyClient.SetBaseURL(server.URL)
	mockClient.On("RestyClient").Return(restyClient)

	adapter := GoCloakAdapter{
		client:   mockClient,
		token:    &gocloak.JWT{AccessToken: "token"},
		basePath: "",
		log:      logmock.NewLogr(),
	}

	_, err := adapter.GetClientScope("name1", "realm1")
	require.NoError(t, err)
}

func TestGoCloakAdapter_DeleteClientScope(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		expectedPath := strings.Replace(
			strings.Replace(deleteDefaultClientScope, "{realm}", "realm1", 1), "{clientScopeID}", "scope1", 1)

		if r.Method == http.MethodDelete && r.URL.Path == expectedPath {
			w.WriteHeader(http.StatusOK)
			return
		}

		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	mockClient := mocks.NewMockGoCloak(t)
	restyClient := resty.New()
	restyClient.SetBaseURL(server.URL)
	mockClient.On("RestyClient").Return(restyClient)
	mockClient.On("DeleteClientScope", mock.Anything, "token", "realm1", "scope1").Return(nil)

	adapter := GoCloakAdapter{
		client:   mockClient,
		token:    &gocloak.JWT{AccessToken: "token"},
		basePath: "",
		log:      logmock.NewLogr(),
	}

	err := adapter.DeleteClientScope(context.Background(), "realm1", "scope1")
	require.NoError(t, err)
}

func TestGetClientScope(t *testing.T) {
	_, err := getClientScope("scope1", []ClientScope{})
	if !IsErrNotFound(err) {
		t.Fatalf("wrong error returned: %s", err.Error())
	}
}

func TestGoCloakAdapter_DeleteClientScope_Failure(t *testing.T) {
	tests := []struct {
		name          string
		setupServer   func() *httptest.Server
		mockGoCloak   func(t *testing.T) *mocks.MockGoCloak
		expectedError string
	}{
		{
			name: "unset default failure",
			setupServer: func() *httptest.Server {
				return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					w.WriteHeader(http.StatusInternalServerError)
				}))
			},
			mockGoCloak: func(t *testing.T) *mocks.MockGoCloak {
				mockClient := mocks.NewMockGoCloak(t)
				return mockClient
			},
			expectedError: "unable to unset default client scope for realm",
		},
		{
			name: "delete client scope failure",
			setupServer: func() *httptest.Server {
				return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					expectedPath := strings.Replace(
						strings.Replace(deleteDefaultClientScope, "{realm}", "realm1", 1), "{clientScopeID}", "scope1", 1)
					if r.Method == http.MethodDelete && r.URL.Path == expectedPath {
						w.WriteHeader(http.StatusOK)
						return
					}
					w.WriteHeader(http.StatusNotFound)
				}))
			},
			mockGoCloak: func(t *testing.T) *mocks.MockGoCloak {
				mockClient := mocks.NewMockGoCloak(t)
				mockClient.On("DeleteClientScope", mock.Anything, "token", "realm1", "scope1").Return(errors.New("logmock fatal"))
				return mockClient
			},
			expectedError: "unable to delete client scope",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := tt.setupServer()
			defer server.Close()

			mockClient := tt.mockGoCloak(t)
			restyClient := resty.New()
			restyClient.SetBaseURL(server.URL)
			mockClient.On("RestyClient").Return(restyClient)

			adapter := GoCloakAdapter{
				client:   mockClient,
				token:    &gocloak.JWT{AccessToken: "token"},
				basePath: "",
				log:      logmock.NewLogr(),
			}

			err := adapter.DeleteClientScope(context.Background(), "realm1", "scope1")
			require.Error(t, err)
			require.Contains(t, err.Error(), tt.expectedError)
		})
	}
}

func TestGoCloakAdapter_GetClientScopeMappers(t *testing.T) {
	tests := []struct {
		name      string
		realmName string
		scopeID   string
		response  int
		body      string
		wantErr   bool
		errMsg    string
	}{
		{
			name:      "success",
			realmName: "realm1",
			scopeID:   "scope1",
			response:  200,
			body:      "",
			wantErr:   false,
		},
		{
			name:      "error",
			realmName: "realm1",
			scopeID:   "scope2",
			response:  422,
			body:      "forbidden",
			wantErr:   true,
			errMsg:    "unable to get client scope mappers: status: 422 Unprocessable Entity, body: forbidden",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				expectedPath := strings.Replace(postClientScopeMapper, "{realm}", tt.realmName, 1)
				expectedPath = strings.Replace(expectedPath, "{scopeId}", tt.scopeID, 1)

				if r.Method == http.MethodGet && r.URL.Path == expectedPath {
					w.WriteHeader(tt.response)
					_, _ = w.Write([]byte(tt.body))

					return
				}

				w.WriteHeader(http.StatusNotFound)
			}))
			defer server.Close()

			kcClient := initClientScopeAdapter(t, server)

			_, err := kcClient.GetClientScopeMappers(context.Background(), tt.realmName, tt.scopeID)
			if tt.wantErr {
				require.Error(t, err)

				if tt.errMsg != "" {
					require.Equal(t, tt.errMsg, err.Error())
				}
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestGoCloakAdapter_PutClientScopeMapper(t *testing.T) {
	tests := []struct {
		name      string
		realmName string
		scopeID   string
		response  int
		body      string
		wantErr   bool
		errMsg    string
	}{
		{
			name:      "success",
			realmName: "realm1",
			scopeID:   "scope1",
			response:  200,
			body:      "",
			wantErr:   false,
		},
		{
			name:      "error",
			realmName: "realm1",
			scopeID:   "scope2",
			response:  422,
			body:      "forbidden",
			wantErr:   true,
			errMsg:    "unable to put client scope mapper: status: 422 Unprocessable Entity, body: forbidden",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				expectedPath := strings.Replace(postClientScopeMapper, "{realm}", tt.realmName, 1)
				expectedPath = strings.Replace(expectedPath, "{scopeId}", tt.scopeID, 1)

				if r.Method == http.MethodPost && r.URL.Path == expectedPath {
					w.WriteHeader(tt.response)
					_, _ = w.Write([]byte(tt.body))

					return
				}

				w.WriteHeader(http.StatusNotFound)
			}))
			defer server.Close()

			kcClient := initClientScopeAdapter(t, server)

			err := kcClient.PutClientScopeMapper(tt.realmName, tt.scopeID, &ProtocolMapper{})
			if tt.wantErr {
				require.Error(t, err)

				if tt.errMsg != "" {
					require.Equal(t, tt.errMsg, err.Error())
				}
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestGoCloakAdapter_GetClientScopesByNames(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		realm        string
		statusCode   int
		responseData []ClientScope
		responseBody string
		scopeNames   []string
		want         []ClientScope
		wantErr      require.ErrorAssertionFunc
	}{
		{
			name:       "should get client scope",
			realm:      "realm1",
			statusCode: http.StatusOK,
			responseData: []ClientScope{
				{
					ID:   "testScope",
					Name: "scope1",
				},
			},
			scopeNames: []string{"scope1"},
			want: []ClientScope{
				{
					ID:   "testScope",
					Name: "scope1",
				},
			},
			wantErr: require.NoError,
		},
		{
			name:       "should not find the client scope",
			realm:      "realm2",
			statusCode: http.StatusOK,
			responseData: []ClientScope{
				{
					ID:   "testScope",
					Name: "scope2",
				},
			},
			scopeNames: []string{"scope1", "scope"},
			want:       nil,
			wantErr:    require.Error,
		},
		{
			name:         "should fail to get scopes",
			realm:        "realm3",
			statusCode:   http.StatusBadRequest,
			responseBody: "",
			scopeNames:   []string{"scope1"},
			want:         nil,
			wantErr:      require.Error,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				expectedPath := strings.Replace(getRealmClientScopes, "{realm}", tt.realm, 1)
				if r.Method == http.MethodGet && r.URL.Path == expectedPath {
					if tt.statusCode == http.StatusOK && tt.responseData != nil {
						setJSONContentType(w)
						w.WriteHeader(tt.statusCode)
						_ = json.NewEncoder(w).Encode(tt.responseData)
					} else {
						w.WriteHeader(tt.statusCode)

						if tt.responseBody != "" {
							_, _ = w.Write([]byte(tt.responseBody))
						}
					}

					return
				}

				w.WriteHeader(http.StatusNotFound)
			}))
			defer server.Close()

			adapter := initClientScopeAdapter(t, server)

			got, err := adapter.GetClientScopesByNames(context.Background(), tt.realm, tt.scopeNames)
			tt.wantErr(t, err)
			require.Equal(t, tt.want, got)
		})
	}
}

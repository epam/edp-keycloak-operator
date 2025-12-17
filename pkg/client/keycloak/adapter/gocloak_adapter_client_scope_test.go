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

	var mockClient *mocks.MockGoCloak

	if server != nil {
		mockClient = newMockClientWithResty(t, server.URL)
	} else {
		mockClient = mocks.NewMockGoCloak(t)
		restyClient := resty.New()
		mockClient.On("RestyClient").Return(restyClient).Maybe()
	}

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

	mockClient := newMockClientWithResty(t, server.URL)

	adapter := GoCloakAdapter{
		client:   mockClient,
		token:    &gocloak.JWT{AccessToken: "token"},
		basePath: "",
		log:      logmock.NewLogr(),
	}

	id, err := adapter.CreateClientScope(context.Background(), "realm1",
		&ClientScope{Name: "demo", Type: ClientScopeTypeDefault})
	require.NoError(t, err)
	require.NotEmpty(t, id)
}

func TestGoCloakAdapter_CreateClientScope_FailureSetDefault(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		expectedPostPath := strings.Replace(getRealmClientScopes, "{realm}", "realm1", 1)
		expectedPutPath := strings.Replace(
			strings.Replace(putDefaultClientScope, "{realm}", "realm1", 1), "{clientScopeID}", "new-scope-id", 1)
		expectedDeleteOptional := strings.Replace(
			strings.Replace(deleteOptionalClientScope, "{realm}", "realm1", 1), "{clientScopeID}", "new-scope-id", 1)

		switch {
		case r.Method == http.MethodPost && r.URL.Path == expectedPostPath:
			w.Header().Set("Location", "id/new-scope-id")
			w.WriteHeader(http.StatusOK)
		case r.Method == http.MethodPut && r.URL.Path == expectedPutPath:
			w.WriteHeader(http.StatusInternalServerError)
		case r.Method == http.MethodDelete && r.URL.Path == expectedDeleteOptional:
			w.WriteHeader(http.StatusNotFound) // 404 is ok, should be ignored
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	mockClient := newMockClientWithResty(t, server.URL)

	adapter := GoCloakAdapter{
		client:   mockClient,
		token:    &gocloak.JWT{AccessToken: "token"},
		basePath: "",
		log:      logmock.NewLogr(),
	}

	_, err := adapter.CreateClientScope(context.Background(), "realm1",
		&ClientScope{Name: "demo", Type: "default"})
	require.Error(t, err)
	require.Contains(t, err.Error(), "unable to set client scope type")
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

	mockClient := newMockClientWithResty(t, server.URL)

	adapter := GoCloakAdapter{
		client: mockClient,
		token:  &gocloak.JWT{AccessToken: "token"},
	}

	_, err := adapter.CreateClientScope(context.Background(), "realm1",
		&ClientScope{Name: "demo", Type: ClientScopeTypeDefault})

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

	mockClient := newMockClientWithResty(t, server.URL)

	adapter := GoCloakAdapter{
		client: mockClient,
		token:  &gocloak.JWT{AccessToken: "token"},
	}

	_, err := adapter.CreateClientScope(context.Background(), "realm1",
		&ClientScope{Name: "demo", Type: ClientScopeTypeDefault})

	require.Error(t, err)
	require.Contains(t, err.Error(), "location header is not set or empty")
}

func TestGoCloakAdapter_UpdateClientScope(t *testing.T) {
	var (
		realmName = "realm1"
		scopeID   = "scope1"
	)

	tests := []struct {
		name           string
		clientScope    *ClientScope
		defaultScopes  []ClientScope
		optionalScopes []ClientScope
	}{
		{
			name: "update with type none",
			clientScope: &ClientScope{
				Name: "scope1",
				ProtocolMappers: []ProtocolMapper{{
					Name: "mp2",
				}},
				Type: "none",
			},
			defaultScopes:  []ClientScope{},
			optionalScopes: []ClientScope{},
		},
		{
			name: "update with type default",
			clientScope: &ClientScope{
				Name: "scope1",
				ProtocolMappers: []ProtocolMapper{{
					Name: "mp2",
				}},
				Type: "default",
			},
			defaultScopes:  []ClientScope{},
			optionalScopes: []ClientScope{},
		},
		{
			name: "update with type optional",
			clientScope: &ClientScope{
				Name: "scope1",
				ProtocolMappers: []ProtocolMapper{{
					Name: "mp2",
				}},
				Type: "optional",
			},
			defaultScopes:  []ClientScope{},
			optionalScopes: []ClientScope{},
		},
		{
			name: "update from default to none",
			clientScope: &ClientScope{
				Name: "scope1",
				ProtocolMappers: []ProtocolMapper{{
					Name: "mp2",
				}},
				Type: "none",
			},
			defaultScopes:  []ClientScope{{Name: "scope1"}},
			optionalScopes: []ClientScope{},
		},
		{
			name: "update from optional to default",
			clientScope: &ClientScope{
				Name: "scope1",
				ProtocolMappers: []ProtocolMapper{{
					Name: "mp2",
				}},
				Type: "default",
			},
			defaultScopes:  []ClientScope{},
			optionalScopes: []ClientScope{{Name: "scope1"}},
		},
		{
			name: "update from default to optional",
			clientScope: &ClientScope{
				Name: "scope1",
				ProtocolMappers: []ProtocolMapper{{
					Name: "mp2",
				}},
				Type: "optional",
			},
			defaultScopes:  []ClientScope{{Name: "scope1"}},
			optionalScopes: []ClientScope{},
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
				expectedGetOptionalPath := strings.Replace(getOptionalClientScopes, "{realm}", "realm1", 1)

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
				case r.Method == http.MethodGet && r.URL.Path == expectedGetOptionalPath:
					setJSONContentType(w)
					w.WriteHeader(http.StatusOK)
					_ = json.NewEncoder(w).Encode(tt.optionalScopes)
				case r.Method == http.MethodPut && strings.Contains(r.URL.Path, "/default-default-client-scopes/"):
					w.WriteHeader(http.StatusOK)
				case r.Method == http.MethodDelete && strings.Contains(r.URL.Path, "/default-default-client-scopes/"):
					w.WriteHeader(http.StatusOK)
				case r.Method == http.MethodPut && strings.Contains(r.URL.Path, "/default-optional-client-scopes/"):
					w.WriteHeader(http.StatusOK)
				case r.Method == http.MethodDelete && strings.Contains(r.URL.Path, "/default-optional-client-scopes/"):
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

func TestGoCloakAdapter_UpdateClientScope_Errors(t *testing.T) {
	var (
		realmName = "realm1"
		scopeID   = "scope1"
	)

	tests := []struct {
		name          string
		setupMock     func(t *testing.T) *mocks.MockGoCloak
		setupServer   func() *httptest.Server
		clientScope   *ClientScope
		expectedError string
	}{
		{
			name: "protocol mappers sync error",
			setupMock: func(t *testing.T) *mocks.MockGoCloak {
				mockClient := mocks.NewMockGoCloak(t)
				mockClient.On("GetClientScope", mock.Anything, "token", realmName, scopeID).Return(nil, errors.New("sync error"))
				return mockClient
			},
			clientScope: &ClientScope{
				Name: "scope1",
				ProtocolMappers: []ProtocolMapper{{
					Name: "mp1",
				}},
			},
			expectedError: "unable to sync client scope protocol mappers",
		},
		{
			name: "client scope update error",
			setupMock: func(t *testing.T) *mocks.MockGoCloak {
				mockClient := mocks.NewMockGoCloak(t)
				mockClient.On("GetClientScope", mock.Anything, "token", realmName, scopeID).Return(&gocloak.ClientScope{
					ID:              gocloak.StringP(scopeID),
					ProtocolMappers: &[]gocloak.ProtocolMappers{},
				}, nil)
				return mockClient
			},
			setupServer: func() *httptest.Server {
				return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					expectedPutPath := strings.Replace(putClientScope, "{realm}", realmName, 1)
					expectedPutPath = strings.Replace(expectedPutPath, "{id}", scopeID, 1)

					if r.Method == http.MethodPut && r.URL.Path == expectedPutPath {
						w.WriteHeader(http.StatusInternalServerError)
						_, _ = w.Write([]byte("update error"))
						return
					}
					w.WriteHeader(http.StatusNotFound)
				}))
			},
			clientScope: &ClientScope{
				Name: "scope1",
			},
			expectedError: "unable to update client scope",
		},
		{
			name: "check if need to update default error",
			setupMock: func(t *testing.T) *mocks.MockGoCloak {
				mockClient := mocks.NewMockGoCloak(t)
				mockClient.On("GetClientScope", mock.Anything, "token", realmName, scopeID).Return(&gocloak.ClientScope{
					ID:              gocloak.StringP(scopeID),
					ProtocolMappers: &[]gocloak.ProtocolMappers{},
				}, nil)
				return mockClient
			},
			setupServer: func() *httptest.Server {
				return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					expectedPutPath := strings.Replace(putClientScope, "{realm}", realmName, 1)
					expectedPutPath = strings.Replace(expectedPutPath, "{id}", scopeID, 1)
					expectedGetDefaultPath := strings.Replace(getDefaultClientScopes, "{realm}", realmName, 1)

					switch {
					case r.Method == http.MethodPut && r.URL.Path == expectedPutPath:
						w.WriteHeader(http.StatusOK)
					case r.Method == http.MethodGet && r.URL.Path == expectedGetDefaultPath:
						w.WriteHeader(http.StatusInternalServerError)
						_, _ = w.Write([]byte("default check error"))
					default:
						w.WriteHeader(http.StatusNotFound)
					}
				}))
			},
			clientScope: &ClientScope{
				Name: "scope1",
				Type: "default",
			},
			expectedError: "unable to check if need to update type",
		},
		{
			name: "unset default client scope error",
			setupMock: func(t *testing.T) *mocks.MockGoCloak {
				mockClient := mocks.NewMockGoCloak(t)
				mockClient.On("GetClientScope", mock.Anything, "token", realmName, scopeID).Return(&gocloak.ClientScope{
					ID:              gocloak.StringP(scopeID),
					ProtocolMappers: &[]gocloak.ProtocolMappers{},
				}, nil)
				return mockClient
			},
			setupServer: func() *httptest.Server {
				return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					expectedPutPath := strings.Replace(putClientScope, "{realm}", realmName, 1)
					expectedPutPath = strings.Replace(expectedPutPath, "{id}", scopeID, 1)
					expectedGetDefaultPath := strings.Replace(getDefaultClientScopes, "{realm}", realmName, 1)
					expectedGetOptionalPath := strings.Replace(getOptionalClientScopes, "{realm}", realmName, 1)
					expectedDeletePath := strings.Replace(deleteDefaultClientScope, "{realm}", realmName, 1)
					expectedDeletePath = strings.Replace(expectedDeletePath, "{clientScopeID}", scopeID, 1)

					switch {
					case r.Method == http.MethodPut && r.URL.Path == expectedPutPath:
						w.WriteHeader(http.StatusOK)
					case r.Method == http.MethodGet && r.URL.Path == expectedGetDefaultPath:
						setJSONContentType(w)
						w.WriteHeader(http.StatusOK)
						_ = json.NewEncoder(w).Encode([]ClientScope{{Name: "scope1"}})
					case r.Method == http.MethodGet && r.URL.Path == expectedGetOptionalPath:
						setJSONContentType(w)
						w.WriteHeader(http.StatusOK)
						_ = json.NewEncoder(w).Encode([]ClientScope{})
					case r.Method == http.MethodDelete && r.URL.Path == expectedDeletePath:
						w.WriteHeader(http.StatusInternalServerError)
						_, _ = w.Write([]byte("unset default error"))
					default:
						w.WriteHeader(http.StatusNotFound)
					}
				}))
			},
			clientScope: &ClientScope{
				Name: "scope1",
				Type: "none",
			},
			expectedError: "unable to set client scope type",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var server *httptest.Server

			var adapter GoCloakAdapter

			mockClient := tt.setupMock(t)

			if tt.setupServer != nil {
				server = tt.setupServer()
				defer server.Close()

				restyClient := resty.New()
				restyClient.SetBaseURL(server.URL)
				mockClient.On("RestyClient").Return(restyClient)
			}

			adapter = GoCloakAdapter{
				client:   mockClient,
				token:    &gocloak.JWT{AccessToken: "token"},
				basePath: "",
				log:      logmock.NewLogr(),
			}

			err := adapter.UpdateClientScope(context.Background(), realmName, scopeID, tt.clientScope)
			require.Error(t, err)
			require.Contains(t, err.Error(), tt.expectedError)
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

	mockClient := newMockClientWithResty(t, server.URL)

	adapter := GoCloakAdapter{
		client:   mockClient,
		token:    &gocloak.JWT{AccessToken: "token"},
		basePath: "",
		log:      logmock.NewLogr(),
	}

	_, err := adapter.GetClientScope(context.Background(), "name1", "realm1")
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

	mockClient := newMockClientWithResty(t, server.URL)
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

func TestGoCloakAdapter_syncClientScopeProtocolMappers(t *testing.T) {
	tests := []struct {
		name                  string
		setupMockClient       func(t *testing.T) *mocks.MockGoCloak
		setupServer           func() *httptest.Server
		protocolMappers       []ProtocolMapper
		expectedError         string
		expectedErrorContains []string
		useServer             bool
	}{
		{
			name: "get client scope error",
			setupMockClient: func(t *testing.T) *mocks.MockGoCloak {
				mockClient := mocks.NewMockGoCloak(t)
				mockClient.On(
					"GetClientScope",
					mock.Anything,
					"token",
					"realm1",
					"scope1",
				).Return(nil, errors.New("get client scope error"))
				return mockClient
			},
			protocolMappers: []ProtocolMapper{},
			expectedErrorContains: []string{
				"unable to get client scope",
				"get client scope error",
			},
			useServer: false,
		},
		{
			name: "delete mapper error",
			setupMockClient: func(t *testing.T) *mocks.MockGoCloak {
				return nil // Will be set up with server
			},
			setupServer: func() *httptest.Server {
				return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					if r.Method == http.MethodDelete && strings.Contains(r.URL.Path, "/protocol-mappers/") {
						w.WriteHeader(http.StatusInternalServerError)
						_, _ = w.Write([]byte("delete mapper error"))
						return
					}
					w.WriteHeader(http.StatusNotFound)
				}))
			},
			protocolMappers: []ProtocolMapper{},
			expectedErrorContains: []string{
				"error during client scope protocol mapper deletion",
			},
			useServer: true,
		},
		{
			name: "create mapper error",
			setupMockClient: func(t *testing.T) *mocks.MockGoCloak {
				return nil // Will be set up with server
			},
			setupServer: func() *httptest.Server {
				return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					if r.Method == http.MethodPost && strings.Contains(r.URL.Path, "/protocol-mappers/models") {
						w.WriteHeader(http.StatusBadRequest)
						_, _ = w.Write([]byte("create mapper error"))
						return
					}
					if r.Method == http.MethodDelete && strings.Contains(r.URL.Path, "/protocol-mappers/") {
						w.WriteHeader(http.StatusOK)
						return
					}
					w.WriteHeader(http.StatusNotFound)
				}))
			},
			protocolMappers: []ProtocolMapper{
				{
					Name:     "new_mapper",
					Protocol: "openid-connect",
				},
			},
			expectedErrorContains: []string{
				"error during client scope protocol mapper creation",
			},
			useServer: true,
		},
		{
			name: "success case",
			setupMockClient: func(t *testing.T) *mocks.MockGoCloak {
				return nil // Will be set up with server
			},
			setupServer: func() *httptest.Server {
				return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					if r.Method == http.MethodDelete && strings.Contains(r.URL.Path, "/protocol-mappers/") {
						w.WriteHeader(http.StatusOK)
						return
					}
					if r.Method == http.MethodPost && strings.Contains(r.URL.Path, "/protocol-mappers/models") {
						w.WriteHeader(http.StatusOK)
						return
					}
					w.WriteHeader(http.StatusNotFound)
				}))
			},
			protocolMappers: []ProtocolMapper{
				{
					Name:     "new_mapper",
					Protocol: "openid-connect",
				},
			},
			expectedErrorContains: nil,
			useServer:             true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var adapter GoCloakAdapter

			var server *httptest.Server

			if tt.useServer {
				server = tt.setupServer()
				defer server.Close()

				mockClient := newMockClientWithResty(t, server.URL)
				mockClient.On("GetClientScope", mock.Anything, "token", "realm1", "scope1").Return(&gocloak.ClientScope{
					ID: gocloak.StringP("scope1"),
					ProtocolMappers: &[]gocloak.ProtocolMappers{
						{
							Name: gocloak.StringP("mapper1"),
							ID:   gocloak.StringP("mapper_id1"),
						},
					},
				}, nil)

				adapter = GoCloakAdapter{
					client:   mockClient,
					token:    &gocloak.JWT{AccessToken: "token"},
					basePath: "",
					log:      logmock.NewLogr(),
				}
			} else {
				mockClient := tt.setupMockClient(t)
				adapter = GoCloakAdapter{
					client: mockClient,
					token:  &gocloak.JWT{AccessToken: "token"},
					log:    logmock.NewLogr(),
				}
			}

			err := adapter.syncClientScopeProtocolMappers(context.Background(), "realm1", "scope1", tt.protocolMappers)

			if tt.expectedErrorContains != nil {
				require.Error(t, err)

				for _, expectedMsg := range tt.expectedErrorContains {
					require.Contains(t, err.Error(), expectedMsg)
				}
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestGoCloakAdapter_GetDefaultClientScopesForRealm_Error(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		expectedPath := strings.Replace(getDefaultClientScopes, "{realm}", "realm1", 1)
		if r.Method == http.MethodGet && r.URL.Path == expectedPath {
			w.WriteHeader(http.StatusForbidden)
			_, _ = w.Write([]byte("access denied"))

			return
		}

		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	adapter := initClientScopeAdapter(t, server)

	_, err := adapter.GetDefaultClientScopesForRealm(context.Background(), "realm1")
	require.Error(t, err)
	require.Contains(t, err.Error(), "unable to get default client scopes for realm")
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

// httpResponse represents a configurable HTTP response for testing.
type httpResponse struct {
	statusCode int
	body       string
}

// setupClientScopeTypeServer creates an HTTP test server for testing client scope type operations.
func setupClientScopeTypeServer(realm, scopeID string, responses map[string]httpResponse) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Build expected paths
		putDefaultPath := strings.Replace(putDefaultClientScope, "{realm}", realm, 1)
		putDefaultPath = strings.Replace(putDefaultPath, "{clientScopeID}", scopeID, 1)
		putOptionalPath := strings.Replace(putOptionalClientScope, "{realm}", realm, 1)
		putOptionalPath = strings.Replace(putOptionalPath, "{clientScopeID}", scopeID, 1)
		deleteDefaultPath := strings.Replace(deleteDefaultClientScope, "{realm}", realm, 1)
		deleteDefaultPath = strings.Replace(deleteDefaultPath, "{clientScopeID}", scopeID, 1)
		deleteOptionalPath := strings.Replace(deleteOptionalClientScope, "{realm}", realm, 1)
		deleteOptionalPath = strings.Replace(deleteOptionalPath, "{clientScopeID}", scopeID, 1)

		// Determine the key for this request
		var key string

		switch {
		case r.Method == http.MethodPut && r.URL.Path == putDefaultPath:
			key = "putDefault"
		case r.Method == http.MethodPut && r.URL.Path == putOptionalPath:
			key = "putOptional"
		case r.Method == http.MethodDelete && r.URL.Path == deleteDefaultPath:
			key = "deleteDefault"
		case r.Method == http.MethodDelete && r.URL.Path == deleteOptionalPath:
			key = "deleteOptional"
		default:
			w.WriteHeader(http.StatusNotFound)
			return
		}

		// Write the configured response
		if resp, ok := responses[key]; ok {
			w.WriteHeader(resp.statusCode)

			if resp.body != "" {
				_, _ = w.Write([]byte(resp.body))
			}
		} else {
			w.WriteHeader(http.StatusNotFound)
		}
	}))
}

func TestGoCloakAdapter_setClientScopeType(t *testing.T) {
	var (
		realmName = "realm1"
		scopeID   = "scope1"
	)

	tests := []struct {
		name          string
		scopeType     string
		setupServer   func() *httptest.Server
		expectedError string
	}{
		{
			name:      "set default type - success",
			scopeType: ClientScopeTypeDefault,
			setupServer: func() *httptest.Server {
				return setupClientScopeTypeServer(realmName, scopeID, map[string]httpResponse{
					"putDefault":     {statusCode: http.StatusOK},
					"deleteOptional": {statusCode: http.StatusNotFound},
				})
			},
		},
		{
			name:      "set optional type - success",
			scopeType: ClientScopeTypeOptional,
			setupServer: func() *httptest.Server {
				return setupClientScopeTypeServer(realmName, scopeID, map[string]httpResponse{
					"putOptional":   {statusCode: http.StatusOK},
					"deleteDefault": {statusCode: http.StatusNotFound},
				})
			},
		},
		{
			name:      "set none type - success",
			scopeType: ClientScopeTypeNone,
			setupServer: func() *httptest.Server {
				return setupClientScopeTypeServer(realmName, scopeID, map[string]httpResponse{
					"deleteDefault":  {statusCode: http.StatusNotFound},
					"deleteOptional": {statusCode: http.StatusNotFound},
				})
			},
		},
		{
			name:      "set default type - failure on set default",
			scopeType: ClientScopeTypeDefault,
			setupServer: func() *httptest.Server {
				return setupClientScopeTypeServer(realmName, scopeID, map[string]httpResponse{
					"putDefault": {statusCode: http.StatusInternalServerError, body: "set default error"},
				})
			},
			expectedError: "unable to set default client scope for realm",
		},
		{
			name:      "set default type - failure on unset optional",
			scopeType: ClientScopeTypeDefault,
			setupServer: func() *httptest.Server {
				return setupClientScopeTypeServer(realmName, scopeID, map[string]httpResponse{
					"putDefault":     {statusCode: http.StatusOK},
					"deleteOptional": {statusCode: http.StatusInternalServerError, body: "unset optional error"},
				})
			},
			expectedError: "unable to unset optional client scope for realm",
		},
		{
			name:      "set optional type - failure on set optional",
			scopeType: ClientScopeTypeOptional,
			setupServer: func() *httptest.Server {
				return setupClientScopeTypeServer(realmName, scopeID, map[string]httpResponse{
					"putOptional": {statusCode: http.StatusInternalServerError, body: "set optional error"},
				})
			},
			expectedError: "unable to set optional client scope for realm",
		},
		{
			name:      "set optional type - failure on unset default",
			scopeType: ClientScopeTypeOptional,
			setupServer: func() *httptest.Server {
				return setupClientScopeTypeServer(realmName, scopeID, map[string]httpResponse{
					"putOptional":   {statusCode: http.StatusOK},
					"deleteDefault": {statusCode: http.StatusInternalServerError, body: "unset default error"},
				})
			},
			expectedError: "unable to unset default client scope for realm",
		},
		{
			name:      "set none type - failure on unset default",
			scopeType: ClientScopeTypeNone,
			setupServer: func() *httptest.Server {
				return setupClientScopeTypeServer(realmName, scopeID, map[string]httpResponse{
					"deleteDefault": {statusCode: http.StatusInternalServerError, body: "unset default error"},
				})
			},
			expectedError: "unable to unset default client scope for realm",
		},
		{
			name:      "set none type - failure on unset optional",
			scopeType: ClientScopeTypeNone,
			setupServer: func() *httptest.Server {
				return setupClientScopeTypeServer(realmName, scopeID, map[string]httpResponse{
					"deleteDefault":  {statusCode: http.StatusNotFound},
					"deleteOptional": {statusCode: http.StatusInternalServerError, body: "unset optional error"},
				})
			},
			expectedError: "unable to unset optional client scope for realm",
		},
		{
			name:          "invalid type",
			scopeType:     "invalid",
			setupServer:   func() *httptest.Server { return nil },
			expectedError: "invalid client scope type",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var adapter GoCloakAdapter

			if tt.setupServer != nil {
				server := tt.setupServer()
				if server != nil {
					defer server.Close()

					mockClient := newMockClientWithResty(t, server.URL)
					adapter = GoCloakAdapter{
						client:   mockClient,
						token:    &gocloak.JWT{AccessToken: "token"},
						basePath: "",
						log:      logmock.NewLogr(),
					}
				} else {
					mockClient := mocks.NewMockGoCloak(t)
					adapter = GoCloakAdapter{
						client:   mockClient,
						token:    &gocloak.JWT{AccessToken: "token"},
						basePath: "",
						log:      logmock.NewLogr(),
					}
				}
			}

			err := adapter.setClientScopeType(context.Background(), realmName, scopeID, tt.scopeType)

			if tt.expectedError != "" {
				require.Error(t, err)
				require.Contains(t, err.Error(), tt.expectedError)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

// clientScopeGetResponse represents the configuration for getting client scopes in tests.
type clientScopeGetResponse struct {
	defaultScopes  []ClientScope
	defaultError   bool
	optionalScopes []ClientScope
	optionalError  bool
}

// setupNeedToUpdateTypeServer creates an HTTP test server for testing needToUpdateType operations.
func setupNeedToUpdateTypeServer(realm string, response clientScopeGetResponse) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		getDefaultPath := strings.Replace(getDefaultClientScopes, "{realm}", realm, 1)
		getOptionalPath := strings.Replace(getOptionalClientScopes, "{realm}", realm, 1)

		switch {
		case r.Method == http.MethodGet && r.URL.Path == getDefaultPath:
			if response.defaultError {
				w.WriteHeader(http.StatusInternalServerError)
				_, _ = w.Write([]byte("get default error"))

				return
			}

			setJSONContentType(w)
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(response.defaultScopes)
		case r.Method == http.MethodGet && r.URL.Path == getOptionalPath:
			if response.optionalError {
				w.WriteHeader(http.StatusInternalServerError)
				_, _ = w.Write([]byte("get optional error"))

				return
			}

			setJSONContentType(w)
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(response.optionalScopes)
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
}

func TestGoCloakAdapter_needToUpdateType(t *testing.T) {
	var (
		realmName = "realm1"
		scope1    = ClientScope{Name: "scope1"}
	)

	tests := []struct {
		name           string
		clientScope    *ClientScope
		setupServer    func() *httptest.Server
		expectedResult bool
		expectedError  string
	}{
		{
			name: "type default - already in default list",
			clientScope: &ClientScope{
				Name: "scope1",
				Type: ClientScopeTypeDefault,
			},
			setupServer: func() *httptest.Server {
				return setupNeedToUpdateTypeServer(realmName, clientScopeGetResponse{
					defaultScopes: []ClientScope{scope1},
				})
			},
			expectedResult: false,
		},
		{
			name: "type default - not in default list",
			clientScope: &ClientScope{
				Name: "scope1",
				Type: ClientScopeTypeDefault,
			},
			setupServer: func() *httptest.Server {
				return setupNeedToUpdateTypeServer(realmName, clientScopeGetResponse{
					defaultScopes: []ClientScope{},
				})
			},
			expectedResult: true,
		},
		{
			name: "type default - error getting default scopes",
			clientScope: &ClientScope{
				Name: "scope1",
				Type: ClientScopeTypeDefault,
			},
			setupServer: func() *httptest.Server {
				return setupNeedToUpdateTypeServer(realmName, clientScopeGetResponse{
					defaultError: true,
				})
			},
			expectedError: "unable to get default client scopes",
		},
		{
			name: "type optional - already in optional list",
			clientScope: &ClientScope{
				Name: "scope1",
				Type: ClientScopeTypeOptional,
			},
			setupServer: func() *httptest.Server {
				return setupNeedToUpdateTypeServer(realmName, clientScopeGetResponse{
					optionalScopes: []ClientScope{scope1},
				})
			},
			expectedResult: false,
		},
		{
			name: "type optional - not in optional list",
			clientScope: &ClientScope{
				Name: "scope1",
				Type: ClientScopeTypeOptional,
			},
			setupServer: func() *httptest.Server {
				return setupNeedToUpdateTypeServer(realmName, clientScopeGetResponse{
					optionalScopes: []ClientScope{},
				})
			},
			expectedResult: true,
		},
		{
			name: "type optional - error getting optional scopes",
			clientScope: &ClientScope{
				Name: "scope1",
				Type: ClientScopeTypeOptional,
			},
			setupServer: func() *httptest.Server {
				return setupNeedToUpdateTypeServer(realmName, clientScopeGetResponse{
					optionalError: true,
				})
			},
			expectedError: "unable to get optional client scopes",
		},
		{
			name: "type none - in default list",
			clientScope: &ClientScope{
				Name: "scope1",
				Type: ClientScopeTypeNone,
			},
			setupServer: func() *httptest.Server {
				return setupNeedToUpdateTypeServer(realmName, clientScopeGetResponse{
					defaultScopes: []ClientScope{scope1},
				})
			},
			expectedResult: true,
		},
		{
			name: "type none - in optional list",
			clientScope: &ClientScope{
				Name: "scope1",
				Type: ClientScopeTypeNone,
			},
			setupServer: func() *httptest.Server {
				return setupNeedToUpdateTypeServer(realmName, clientScopeGetResponse{
					defaultScopes:  []ClientScope{},
					optionalScopes: []ClientScope{scope1},
				})
			},
			expectedResult: true,
		},
		{
			name: "type none - not in any list",
			clientScope: &ClientScope{
				Name: "scope1",
				Type: ClientScopeTypeNone,
			},
			setupServer: func() *httptest.Server {
				return setupNeedToUpdateTypeServer(realmName, clientScopeGetResponse{
					defaultScopes:  []ClientScope{},
					optionalScopes: []ClientScope{},
				})
			},
			expectedResult: false,
		},
		{
			name: "type none - error getting default scopes",
			clientScope: &ClientScope{
				Name: "scope1",
				Type: ClientScopeTypeNone,
			},
			setupServer: func() *httptest.Server {
				return setupNeedToUpdateTypeServer(realmName, clientScopeGetResponse{
					defaultError: true,
				})
			},
			expectedError: "unable to get default client scopes",
		},
		{
			name: "type none - error getting optional scopes",
			clientScope: &ClientScope{
				Name: "scope1",
				Type: ClientScopeTypeNone,
			},
			setupServer: func() *httptest.Server {
				return setupNeedToUpdateTypeServer(realmName, clientScopeGetResponse{
					defaultScopes: []ClientScope{},
					optionalError: true,
				})
			},
			expectedError: "unable to get optional client scopes",
		},
		{
			name: "unknown type",
			clientScope: &ClientScope{
				Name: "scope1",
				Type: "unknown",
			},
			setupServer: func() *httptest.Server {
				return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					w.WriteHeader(http.StatusNotFound)
				}))
			},
			expectedResult: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := tt.setupServer()
			defer server.Close()

			adapter := initClientScopeAdapter(t, server)

			result, err := adapter.needToUpdateType(context.Background(), realmName, tt.clientScope)

			if tt.expectedError != "" {
				require.Error(t, err)
				require.Contains(t, err.Error(), tt.expectedError)
			} else {
				require.NoError(t, err)
				require.Equal(t, tt.expectedResult, result)
			}
		})
	}
}

func TestGoCloakAdapter_GetOptionalClientScopesForRealm_Error(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		expectedPath := strings.Replace(getOptionalClientScopes, "{realm}", "realm1", 1)
		if r.Method == http.MethodGet && r.URL.Path == expectedPath {
			w.WriteHeader(http.StatusForbidden)
			_, _ = w.Write([]byte("access denied"))

			return
		}

		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	adapter := initClientScopeAdapter(t, server)

	_, err := adapter.GetOptionalClientScopesForRealm(context.Background(), "realm1")
	require.Error(t, err)
	require.Contains(t, err.Error(), "unable to get optional client scopes for realm")
}

func testHasClientScope(
	t *testing.T,
	checkFunc func(*GoCloakAdapter, context.Context, string, string) (bool, error),
	tests []struct {
		name      string
		scopeName string
		scopes    []ClientScope
		want      bool
		wantErr   bool
	},
) {
	t.Helper()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusOK)
				err := json.NewEncoder(w).Encode(tt.scopes)
				require.NoError(t, err)
			}))
			defer server.Close()

			adapter := initClientScopeAdapter(t, server)

			got, err := checkFunc(adapter, context.Background(), "realm1", tt.scopeName)
			if tt.wantErr {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)
			require.Equal(t, tt.want, got)
		})
	}
}

func TestGoCloakAdapter_HasDefaultClientScope(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		scopeName string
		scopes    []ClientScope
		want      bool
		wantErr   bool
	}{
		{
			name:      "scope exists in default list",
			scopeName: "email",
			scopes: []ClientScope{
				{Name: "profile"},
				{Name: "email"},
				{Name: "roles"},
			},
			want:    true,
			wantErr: false,
		},
		{
			name:      "scope does not exist in default list",
			scopeName: "custom-scope",
			scopes: []ClientScope{
				{Name: "profile"},
				{Name: "email"},
				{Name: "roles"},
			},
			want:    false,
			wantErr: false,
		},
		{
			name:      "empty default scopes list",
			scopeName: "email",
			scopes:    []ClientScope{},
			want:      false,
			wantErr:   false,
		},
	}

	testHasClientScope(t, (*GoCloakAdapter).HasDefaultClientScope, tests)
}

func TestGoCloakAdapter_HasDefaultClientScope_Error(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	adapter := initClientScopeAdapter(t, server)

	_, err := adapter.HasDefaultClientScope(context.Background(), "realm1", "email")
	require.Error(t, err)
}

func TestGoCloakAdapter_HasOptionalClientScope(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		scopeName string
		scopes    []ClientScope
		want      bool
		wantErr   bool
	}{
		{
			name:      "scope exists in optional list",
			scopeName: "address",
			scopes: []ClientScope{
				{Name: "phone"},
				{Name: "address"},
				{Name: "microprofile-jwt"},
			},
			want:    true,
			wantErr: false,
		},
		{
			name:      "scope does not exist in optional list",
			scopeName: "custom-scope",
			scopes: []ClientScope{
				{Name: "phone"},
				{Name: "address"},
				{Name: "microprofile-jwt"},
			},
			want:    false,
			wantErr: false,
		},
		{
			name:      "empty optional scopes list",
			scopeName: "address",
			scopes:    []ClientScope{},
			want:      false,
			wantErr:   false,
		},
	}

	testHasClientScope(t, (*GoCloakAdapter).HasOptionalClientScope, tests)
}

func TestGoCloakAdapter_HasOptionalClientScope_Error(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	adapter := initClientScopeAdapter(t, server)

	_, err := adapter.HasOptionalClientScope(context.Background(), "realm1", "address")
	require.Error(t, err)
}

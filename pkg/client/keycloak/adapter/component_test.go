package adapter

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Nerzal/gocloak/v12"
	"github.com/go-resty/resty/v2"
	"github.com/stretchr/testify/require"

	"github.com/epam/edp-keycloak-operator/pkg/client/keycloak/adapter/mocks"
	"github.com/epam/edp-keycloak-operator/pkg/client/keycloak/mock"
)

func initAdapter(t *testing.T, server *httptest.Server) (*GoCloakAdapter, *mocks.MockGoCloak, *resty.Client) {
	t.Helper()

	var mockClient *mocks.MockGoCloak

	var restyClient *resty.Client

	if server != nil {
		mockClient = newMockClientWithResty(t, server.URL)
		restyClient = resty.New()
		restyClient.SetBaseURL(server.URL)
	} else {
		mockClient = mocks.NewMockGoCloak(t)
		restyClient = resty.New()
		mockClient.On("RestyClient").Return(restyClient).Maybe()
	}

	logger := mock.NewLogr()

	return &GoCloakAdapter{
		client:   mockClient,
		basePath: "",
		token:    &gocloak.JWT{AccessToken: "token"},
		log:      logger,
	}, mockClient, restyClient
}

func testComponent() *Component {
	return &Component{
		Name:         "test-name",
		ProviderType: "test-provider-type",
		Config: map[string][]string{
			"foo": {"bar", "vaz"},
		},
	}
}

func TestGoCloakAdapter_CreateComponent(t *testing.T) {
	tests := []struct {
		name       string
		realmName  string
		statusCode int
		body       string
		wantErr    bool
		errMsg     string
	}{
		{
			name:       "success",
			realmName:  "realm-name",
			statusCode: 200,
			body:       "",
			wantErr:    false,
		},
		{
			name:       "error",
			realmName:  "realm-name-error",
			statusCode: 500,
			body:       "fatal",
			wantErr:    true,
			errMsg:     "error during request: status: 500 Internal Server Error, body: fatal",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				expectedPath := strings.Replace(realmComponent, "{realm}", tt.realmName, 1)
				if r.Method == http.MethodPost && r.URL.Path == expectedPath {
					w.WriteHeader(tt.statusCode)
					_, _ = w.Write([]byte(tt.body))

					return
				}

				w.WriteHeader(http.StatusNotFound)
			}))
			defer server.Close()

			kcAdapter, _, _ := initAdapter(t, server)

			err := kcAdapter.CreateComponent(context.Background(), tt.realmName, testComponent())
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

func TestMock_UpdateComponent(t *testing.T) {
	testCmp := testComponent()
	testCmp.ID = "test-id"

	tests := []struct {
		name         string
		realmName    string
		components   []Component
		updateStatus int
		updateBody   string
		wantErr      bool
		errMsg       string
	}{
		{
			name:         "success",
			realmName:    "realm-name",
			components:   []Component{*testCmp},
			updateStatus: 200,
			updateBody:   "",
			wantErr:      false,
		},
		{
			name:       "component not found",
			realmName:  "realm-name-no-components",
			components: []Component{},
			wantErr:    true,
			errMsg:     "unable to get component id: component not found",
		},
		{
			name:         "update failure",
			realmName:    "realm-name-update-failure",
			components:   []Component{*testCmp},
			updateStatus: 404,
			updateBody:   "not found",
			wantErr:      true,
			errMsg:       "error during update component request: status: 404 Not Found, body: not found",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				expectedGetPath := strings.Replace(realmComponent, "{realm}", tt.realmName, 1)
				expectedPutPath := strings.Replace(realmComponentEntity, "{realm}", tt.realmName, 1)
				expectedPutPath = strings.Replace(expectedPutPath, "{id}", "test-id", 1)

				if r.Method == http.MethodGet && r.URL.Path == expectedGetPath {
					setJSONContentType(w)
					w.WriteHeader(http.StatusOK)
					_ = json.NewEncoder(w).Encode(tt.components)

					return
				}

				if r.Method == http.MethodPut && r.URL.Path == expectedPutPath {
					w.WriteHeader(tt.updateStatus)
					_, _ = w.Write([]byte(tt.updateBody))

					return
				}

				w.WriteHeader(http.StatusNotFound)
			}))
			defer server.Close()

			kcAdapter, _, _ := initAdapter(t, server)

			err := kcAdapter.UpdateComponent(context.Background(), tt.realmName, testComponent())
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

func TestGoCloakAdapter_DeleteComponent(t *testing.T) {
	testCmp := testComponent()
	testCmp.ID = "test-id"

	tests := []struct {
		name         string
		realmName    string
		components   []Component
		deleteStatus int
		deleteBody   string
		wantErr      bool
	}{
		{
			name:         "success",
			realmName:    "realm-name",
			components:   []Component{*testCmp},
			deleteStatus: 200,
			deleteBody:   "",
			wantErr:      false,
		},
		{
			name:       "no components",
			realmName:  "realm-name-no-components",
			components: []Component{},
			wantErr:    false,
		},
		{
			name:         "delete failure ignored",
			realmName:    "realm-name-delete-failure",
			components:   []Component{*testCmp},
			deleteStatus: 404,
			deleteBody:   "delete not found",
			wantErr:      false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				expectedGetPath := strings.Replace(realmComponent, "{realm}", tt.realmName, 1)
				expectedDeletePath := strings.Replace(realmComponentEntity, "{realm}", tt.realmName, 1)
				expectedDeletePath = strings.Replace(expectedDeletePath, "{id}", "test-id", 1)

				if r.Method == http.MethodGet && r.URL.Path == expectedGetPath {
					setJSONContentType(w)
					w.WriteHeader(http.StatusOK)
					_ = json.NewEncoder(w).Encode(tt.components)

					return
				}

				if r.Method == http.MethodDelete && r.URL.Path == expectedDeletePath {
					w.WriteHeader(tt.deleteStatus)
					_, _ = w.Write([]byte(tt.deleteBody))

					return
				}

				w.WriteHeader(http.StatusNotFound)
			}))
			defer server.Close()

			kcAdapter, _, _ := initAdapter(t, server)

			err := kcAdapter.DeleteComponent(context.Background(), tt.realmName, testCmp.Name)
			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestGoCloakAdapter_GetComponent_Failure(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		expectedPath := strings.Replace(realmComponent, "{realm}", "realm-name", 1)
		if r.Method == http.MethodGet && r.URL.Path == expectedPath {
			w.WriteHeader(http.StatusUnprocessableEntity)
			_, _ = w.Write([]byte("forbidden"))

			return
		}

		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	kcAdapter, _, _ := initAdapter(t, server)

	_, err := kcAdapter.GetComponent(context.Background(), "realm-name", "test-name")
	require.Error(t, err)
	require.Equal(t, "error during get component request: status: 422 Unprocessable Entity, body: forbidden", err.Error())
}

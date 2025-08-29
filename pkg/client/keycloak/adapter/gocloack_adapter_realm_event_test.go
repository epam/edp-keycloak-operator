package adapter

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Nerzal/gocloak/v12"
	"github.com/go-resty/resty/v2"
	"github.com/stretchr/testify/require"

	"github.com/epam/edp-keycloak-operator/pkg/client/keycloak/adapter/mocks"
)

func TestGoCloakAdapter_SetRealmEventConfig(t *testing.T) {
	tests := []struct {
		name        string
		realmName   string
		setupServer func() *httptest.Server
		wantErr     bool
		errMsg      string
	}{
		{
			name:      "failure - no server response",
			realmName: "realm1",
			setupServer: func() *httptest.Server {
				return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					w.WriteHeader(http.StatusInternalServerError)
				}))
			},
			wantErr: true,
			errMsg:  "failed to set realm event config request",
		},
		{
			name:      "success",
			realmName: "r1",
			setupServer: func() *httptest.Server {
				return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					if r.Method == http.MethodPut && r.URL.Path == strings.Replace(realmEventConfigPut, "{realm}", "r1", 1) {
						w.WriteHeader(http.StatusOK)
						return
					}
					w.WriteHeader(http.StatusNotFound)
				}))
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := tt.setupServer()
			defer server.Close()

			mockClient := mocks.NewMockGoCloak(t)
			restyClient := resty.New()
			restyClient.SetBaseURL(server.URL)
			mockClient.On("RestyClient").Return(restyClient)

			adapter := GoCloakAdapter{
				client:   mockClient,
				basePath: "",
				token:    &gocloak.JWT{AccessToken: "token"},
			}

			var config *RealmEventConfig
			if tt.name == "success" {
				config = &RealmEventConfig{EventsListeners: []string{"foo", "bar"}}
			} else {
				config = &RealmEventConfig{}
			}

			err := adapter.SetRealmEventConfig(tt.realmName, config)
			if tt.wantErr {
				require.Error(t, err)

				if tt.errMsg != "" {
					require.Contains(t, err.Error(), tt.errMsg)
				}
			} else {
				require.NoError(t, err)
			}
		})
	}
}

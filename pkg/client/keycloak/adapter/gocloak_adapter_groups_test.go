package adapter

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Nerzal/gocloak/v12"
	"github.com/go-logr/logr"
	"github.com/go-resty/resty/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	keycloakApi "github.com/epam/edp-keycloak-operator/api/v1"
	kcmocks "github.com/epam/edp-keycloak-operator/pkg/client/keycloak/adapter/mocks"
)

func TestIsErrNotFound(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		err  error
		want bool
	}{
		{
			name: "test error is NotFoundError",
			err:  NotFoundError(""),
			want: true,
		},
		{
			name: "test error is api error not found",
			err:  gocloak.APIError{Code: http.StatusNotFound},
			want: true,
		},
		{
			name: "test error is pointer to api error not found",
			err:  &gocloak.APIError{Code: http.StatusNotFound},
			want: true,
		},
		{
			name: "test error is not api error not found",
			err:  gocloak.APIError{Code: http.StatusBadRequest},
			want: false,
		},
		{
			name: "test error is not NotFoundError",
			err:  errors.New("error"),
			want: false,
		},
		{
			name: "test error is nil",
			err:  nil,
			want: false,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, tt.want, IsErrNotFound(tt.err))
		})
	}
}

func TestGoCloakAdapter_SyncRealmGroup(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "/groups/test-group-old-endpoint-id/children") {
			w.WriteHeader(http.StatusMethodNotAllowed)
			_, err := w.Write([]byte(`No resource method found for GET, return 405 with Allow header`))
			assert.NoError(t, err)

			return
		}

		if strings.Contains(r.URL.Path, "/groups/test-group-old-endpoint-id") {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_, err := w.Write([]byte(`{"id": "test-group-old-endpoint-id", "name": "test-group"}`))
			assert.NoError(t, err)

			return
		}

		if strings.Contains(r.URL.Path, "/children") {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_, err := w.Write([]byte(`[]`))
			assert.NoError(t, err)

			return
		}
	}))
	t.Cleanup(server.Close)

	tests := []struct {
		name    string
		spec    *keycloakApi.KeycloakRealmGroupSpec
		client  func(t *testing.T) GoCloak
		want    string
		wantErr require.ErrorAssertionFunc
	}{
		{
			name: "create realm group successfully",
			spec: &keycloakApi.KeycloakRealmGroupSpec{
				Name:      "test-group",
				SubGroups: []string{"sub-group1", "sub-group2"},
			},
			client: func(t *testing.T) GoCloak {
				m := kcmocks.NewMockGoCloak(t)

				m.On("GetGroups", mock.Anything, mock.Anything, "master", mock.Anything).
					Return(func(ctx context.Context, accessToken, realm string, params gocloak.GetGroupsParams) []*gocloak.Group {
						if params.Search != nil && *params.Search == "test-group" {
							return []*gocloak.Group{}
						}

						if params.Search != nil && *params.Search == "sub-group1" {
							return []*gocloak.Group{{
								ID:   gocloak.StringP("sub-group1-id"),
								Name: gocloak.StringP("sub-group1"),
							}}
						}

						if params.Search != nil && *params.Search == "sub-group2" {
							return []*gocloak.Group{{
								ID:   gocloak.StringP("sub-group2-id"),
								Name: gocloak.StringP("sub-group2"),
							}}
						}

						return []*gocloak.Group{}
					}, nil)

				m.On("CreateGroup", mock.Anything, mock.Anything, "master", mock.Anything).
					Return("test-group-id", nil)

				m.On("GetRoleMappingByGroupID", mock.Anything, mock.Anything, "master", "test-group-id").
					Return(&gocloak.MappingsRepresentation{}, nil)

				m.On("RestyClient").Return(resty.New().SetBaseURL(server.URL))

				m.On("CreateChildGroup", mock.Anything, mock.Anything, "master", "test-group-id", mock.Anything).
					Return("", nil)

				return m
			},
			wantErr: require.NoError,
			want:    "test-group-id",
		},
		{
			name: "update realm group successfully",
			spec: &keycloakApi.KeycloakRealmGroupSpec{
				Name: "test-group",
			},
			client: func(t *testing.T) GoCloak {
				m := kcmocks.NewMockGoCloak(t)

				m.On("GetGroups", mock.Anything, mock.Anything, "master", mock.Anything).
					Return(func(ctx context.Context, accessToken, realm string, params gocloak.GetGroupsParams) []*gocloak.Group {
						return []*gocloak.Group{{
							ID:   gocloak.StringP("test-group-id"),
							Name: gocloak.StringP("test-group"),
						}}
					}, nil)

				m.On("UpdateGroup", mock.Anything, mock.Anything, "master", mock.Anything).
					Return(nil)

				m.On("GetRoleMappingByGroupID", mock.Anything, mock.Anything, "master", "test-group-id").
					Return(&gocloak.MappingsRepresentation{}, nil)

				m.On("RestyClient").Return(resty.New().SetBaseURL(server.URL))

				return m
			},
			wantErr: require.NoError,
			want:    "test-group-id",
		},
		{
			name: "use old endpoint to get child groups",
			spec: &keycloakApi.KeycloakRealmGroupSpec{
				Name:      "test-group",
				SubGroups: []string{"sub-group1"},
			},
			client: func(t *testing.T) GoCloak {
				m := kcmocks.NewMockGoCloak(t)

				m.On("GetGroups", mock.Anything, mock.Anything, "master", mock.Anything).
					Return(func(ctx context.Context, accessToken, realm string, params gocloak.GetGroupsParams) []*gocloak.Group {
						if params.Search != nil && *params.Search == "sub-group1" {
							return []*gocloak.Group{{
								ID:   gocloak.StringP("sub-group1-id"),
								Name: gocloak.StringP("sub-group1"),
							}}
						}

						return []*gocloak.Group{}
					}, nil)

				m.On("CreateGroup", mock.Anything, mock.Anything, "master", mock.Anything).
					Return("test-group-old-endpoint-id", nil)

				m.On("GetRoleMappingByGroupID", mock.Anything, mock.Anything, "master", "test-group-old-endpoint-id").
					Return(&gocloak.MappingsRepresentation{}, nil)

				m.On("RestyClient").Return(resty.New().SetBaseURL(server.URL))

				m.On("CreateChildGroup", mock.Anything, mock.Anything, "master", "test-group-old-endpoint-id", mock.Anything).
					Return("", nil)

				return m
			},
			wantErr: require.NoError,
			want:    "test-group-old-endpoint-id",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			a := GoCloakAdapter{
				client: tt.client(t),
				token: &gocloak.JWT{
					AccessToken: "token",
				},
				log: logr.Discard(),
			}
			got, err := a.SyncRealmGroup(context.Background(), "master", tt.spec)

			tt.wantErr(t, err)
			assert.Equal(t, tt.want, got)

		})
	}
}

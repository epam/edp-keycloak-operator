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
	ctrl "sigs.k8s.io/controller-runtime"

	keycloakApi "github.com/epam/edp-keycloak-operator/api/v1"
	"github.com/epam/edp-keycloak-operator/pkg/client/keycloak/adapter/mocks"
)

const testGroupName = "test-group"

func TestIsErrNotFound(t *testing.T) {
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
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, IsErrNotFound(tt.err))
		})
	}
}

func TestGoCloakAdapter_GetGroups(t *testing.T) {
	tests := []struct {
		name    string
		realm   string
		client  func(t *testing.T) GoCloak
		want    map[string]*gocloak.Group
		wantErr require.ErrorAssertionFunc
	}{
		{
			name:  "successfully get groups",
			realm: "test-realm",
			client: func(t *testing.T) GoCloak {
				m := mocks.NewMockGoCloak(t)
				m.On("GetGroups", mock.Anything, "token", "test-realm", mock.MatchedBy(func(params gocloak.GetGroupsParams) bool {
					return params.Max != nil && *params.Max == 100
				})).Return([]*gocloak.Group{
					{
						ID:   gocloak.StringP("group1-id"),
						Name: gocloak.StringP("group1"),
					},
					{
						ID:   gocloak.StringP("group2-id"),
						Name: gocloak.StringP("group2"),
					},
				}, nil)
				return m
			},
			want: map[string]*gocloak.Group{
				"group1": {
					ID:   gocloak.StringP("group1-id"),
					Name: gocloak.StringP("group1"),
				},
				"group2": {
					ID:   gocloak.StringP("group2-id"),
					Name: gocloak.StringP("group2"),
				},
			},
			wantErr: require.NoError,
		},
		{
			name:  "successfully get empty groups",
			realm: "empty-realm",
			client: func(t *testing.T) GoCloak {
				m := mocks.NewMockGoCloak(t)
				m.On("GetGroups", mock.Anything, "token", "empty-realm", mock.MatchedBy(func(params gocloak.GetGroupsParams) bool {
					return params.Max != nil && *params.Max == 100
				})).Return([]*gocloak.Group{}, nil)
				return m
			},
			want:    map[string]*gocloak.Group{},
			wantErr: require.NoError,
		},
		{
			name:  "handle nil groups in response",
			realm: "nil-groups-realm",
			client: func(t *testing.T) GoCloak {
				m := mocks.NewMockGoCloak(t)
				m.On(
					"GetGroups",
					mock.Anything,
					"token",
					"nil-groups-realm",
					mock.MatchedBy(func(params gocloak.GetGroupsParams) bool {
						return params.Max != nil && *params.Max == 100
					}),
				).Return([]*gocloak.Group{
					nil,
					{
						ID:   gocloak.StringP("valid-group-id"),
						Name: gocloak.StringP("valid-group"),
					},
					{
						ID:   gocloak.StringP("nil-name-group-id"),
						Name: nil,
					},
				}, nil)
				return m
			},
			want: map[string]*gocloak.Group{
				"valid-group": {
					ID:   gocloak.StringP("valid-group-id"),
					Name: gocloak.StringP("valid-group"),
				},
			},
			wantErr: require.NoError,
		},
		{
			name:  "return error when client fails",
			realm: "error-realm",
			client: func(t *testing.T) GoCloak {
				m := mocks.NewMockGoCloak(t)
				m.On("GetGroups", mock.Anything, "token", "error-realm", mock.MatchedBy(func(params gocloak.GetGroupsParams) bool {
					return params.Max != nil && *params.Max == 100
				})).Return(nil, errors.New("client error"))
				return m
			},
			want:    nil,
			wantErr: require.Error,
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

			got, err := a.GetGroups(context.Background(), tt.realm)
			tt.wantErr(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestGetGroupByName(t *testing.T) {
	tests := []struct {
		name      string
		groups    []gocloak.Group
		groupName string
		want      *gocloak.Group
	}{
		{
			name: "find group by name",
			groups: []gocloak.Group{
				{Name: gocloak.StringP("group1")},
				{Name: gocloak.StringP("group2")},
			},
			groupName: "group2",
			want:      &gocloak.Group{Name: gocloak.StringP("group2")},
		},
		{
			name: "find group in subgroups",
			groups: []gocloak.Group{
				{
					Name: gocloak.StringP("parent"),
					SubGroups: &[]gocloak.Group{
						{Name: gocloak.StringP("child1")},
						{Name: gocloak.StringP("target")},
					},
				},
			},
			groupName: "target",
			want:      &gocloak.Group{Name: gocloak.StringP("target")},
		},
		{
			name:      "group not found",
			groups:    []gocloak.Group{{Name: gocloak.StringP("other")}},
			groupName: "notfound",
			want:      nil,
		},
		{
			name:      "empty groups",
			groups:    []gocloak.Group{},
			groupName: "any",
			want:      nil,
		},
		{
			name: "nil subgroups",
			groups: []gocloak.Group{
				{Name: gocloak.StringP("parent"), SubGroups: nil},
			},
			groupName: "child",
			want:      nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := getGroupByName(tt.groups, tt.groupName)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestNotFoundError(t *testing.T) {
	err := NotFoundError("test error")
	assert.Equal(t, "test error", err.Error())
	assert.True(t, IsErrNotFound(err))
}

func TestGoCloakAdapter_getGroup(t *testing.T) {
	tests := []struct {
		name      string
		realm     string
		groupName string
		client    func(t *testing.T) GoCloak
		want      *gocloak.Group
		wantErr   require.ErrorAssertionFunc
	}{
		{
			name:      "successfully get group",
			realm:     "test-realm",
			groupName: testGroupName,
			client: func(t *testing.T) GoCloak {
				groupName := testGroupName
				m := mocks.NewMockGoCloak(t)
				m.On("GetGroups", mock.Anything, "token", "test-realm", mock.MatchedBy(func(params gocloak.GetGroupsParams) bool {
					return params.Search != nil && *params.Search == groupName
				})).Return([]*gocloak.Group{
					{
						ID:   gocloak.StringP("test-group-id"),
						Name: gocloak.StringP(groupName),
					},
				}, nil)
				return m
			},
			want: &gocloak.Group{
				ID:   gocloak.StringP("test-group-id"),
				Name: gocloak.StringP(testGroupName),
			},
			wantErr: require.NoError,
		},
		{
			name:      "group not found",
			realm:     "test-realm",
			groupName: "nonexistent",
			client: func(t *testing.T) GoCloak {
				m := mocks.NewMockGoCloak(t)
				m.On("GetGroups", mock.Anything, "token", "test-realm", mock.MatchedBy(func(params gocloak.GetGroupsParams) bool {
					return params.Search != nil && *params.Search == "nonexistent"
				})).Return([]*gocloak.Group{}, nil)
				return m
			},
			want:    nil,
			wantErr: require.Error,
		},
		{
			name:      "client error",
			realm:     "test-realm",
			groupName: testGroupName,
			client: func(t *testing.T) GoCloak {
				groupName := testGroupName
				m := mocks.NewMockGoCloak(t)
				m.On("GetGroups", mock.Anything, "token", "test-realm", mock.MatchedBy(func(params gocloak.GetGroupsParams) bool {
					return params.Search != nil && *params.Search == groupName
				})).Return(nil, errors.New("client error"))
				return m
			},
			want:    nil,
			wantErr: require.Error,
		},
		{
			name:      "handle only valid groups in response",
			realm:     "test-realm",
			groupName: testGroupName,
			client: func(t *testing.T) GoCloak {
				groupName := testGroupName
				m := mocks.NewMockGoCloak(t)
				m.On("GetGroups", mock.Anything, "token", "test-realm", mock.MatchedBy(func(params gocloak.GetGroupsParams) bool {
					return params.Search != nil && *params.Search == groupName
				})).Return([]*gocloak.Group{
					{
						ID:   gocloak.StringP("test-group-id"),
						Name: gocloak.StringP(groupName),
					},
				}, nil)
				return m
			},
			want: &gocloak.Group{
				ID:   gocloak.StringP("test-group-id"),
				Name: gocloak.StringP(testGroupName),
			},
			wantErr: require.NoError,
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

			got, err := a.getGroup(context.Background(), tt.realm, tt.groupName)
			tt.wantErr(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestGoCloakAdapter_getGroupsByNames(t *testing.T) {
	tests := []struct {
		name       string
		realm      string
		groupNames []string
		client     func(t *testing.T) GoCloak
		want       map[string]gocloak.Group
		wantErr    require.ErrorAssertionFunc
	}{
		{
			name:       "successfully get groups by names",
			realm:      "test-realm",
			groupNames: []string{"group1", "group2"},
			client: func(t *testing.T) GoCloak {
				m := mocks.NewMockGoCloak(t)
				m.On("GetGroups", mock.Anything, "token", "test-realm", mock.MatchedBy(func(params gocloak.GetGroupsParams) bool {
					return params.Search != nil && *params.Search == "group1"
				})).Return([]*gocloak.Group{
					{ID: gocloak.StringP("group1-id"), Name: gocloak.StringP("group1")},
				}, nil)
				m.On("GetGroups", mock.Anything, "token", "test-realm", mock.MatchedBy(func(params gocloak.GetGroupsParams) bool {
					return params.Search != nil && *params.Search == "group2"
				})).Return([]*gocloak.Group{
					{ID: gocloak.StringP("group2-id"), Name: gocloak.StringP("group2")},
				}, nil)
				return m
			},
			want: map[string]gocloak.Group{
				"group1": {ID: gocloak.StringP("group1-id"), Name: gocloak.StringP("group1")},
				"group2": {ID: gocloak.StringP("group2-id"), Name: gocloak.StringP("group2")},
			},
			wantErr: require.NoError,
		},
		{
			name:       "error getting one group",
			realm:      "test-realm",
			groupNames: []string{"group1", "nonexistent"},
			client: func(t *testing.T) GoCloak {
				m := mocks.NewMockGoCloak(t)
				m.On("GetGroups", mock.Anything, "token", "test-realm", mock.MatchedBy(func(params gocloak.GetGroupsParams) bool {
					return params.Search != nil && *params.Search == "group1"
				})).Return([]*gocloak.Group{
					{ID: gocloak.StringP("group1-id"), Name: gocloak.StringP("group1")},
				}, nil)
				m.On("GetGroups", mock.Anything, "token", "test-realm", mock.MatchedBy(func(params gocloak.GetGroupsParams) bool {
					return params.Search != nil && *params.Search == "nonexistent"
				})).Return([]*gocloak.Group{}, nil)
				return m
			},
			want:    nil,
			wantErr: require.Error,
		},
		{
			name:       "empty group names",
			realm:      "test-realm",
			groupNames: []string{},
			client: func(t *testing.T) GoCloak {
				return mocks.NewMockGoCloak(t)
			},
			want:    map[string]gocloak.Group{},
			wantErr: require.NoError,
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

			got, err := a.getGroupsByNames(context.Background(), tt.realm, tt.groupNames)
			tt.wantErr(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestGoCloakAdapter_SyncRealmGroup(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "/groups/test-group-old-endpoint-id/children") {
			w.WriteHeader(http.StatusMethodNotAllowed)
			_, err := w.Write([]byte(`No resource method found for GET, return 405 with Allow header`))
			assert.NoError(t, err)

			return
		}

		if strings.Contains(r.URL.Path, "/groups/test-group-old-endpoint-id") {
			setJSONContentType(w)
			w.WriteHeader(http.StatusOK)
			_, err := w.Write([]byte(`{"id": "test-group-old-endpoint-id", "name": "` + testGroupName + `"}`))
			assert.NoError(t, err)

			return
		}

		if strings.Contains(r.URL.Path, "/children") {
			setJSONContentType(w)
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
				Name:      testGroupName,
				SubGroups: []string{"sub-group1", "sub-group2"},
			},
			client: func(t *testing.T) GoCloak {
				m := mocks.NewMockGoCloak(t)

				m.On("GetGroups", mock.Anything, mock.Anything, "master", mock.Anything).
					Return(func(ctx context.Context, accessToken, realm string, params gocloak.GetGroupsParams) []*gocloak.Group {
						if params.Search != nil && *params.Search == testGroupName {
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
				Name: testGroupName,
			},
			client: func(t *testing.T) GoCloak {
				m := mocks.NewMockGoCloak(t)

				m.On("GetGroups", mock.Anything, mock.Anything, "master", mock.Anything).
					Return(func(ctx context.Context, accessToken, realm string, params gocloak.GetGroupsParams) []*gocloak.Group {
						return []*gocloak.Group{{
							ID:   gocloak.StringP("test-group-id"),
							Name: gocloak.StringP(testGroupName),
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
				Name:      testGroupName,
				SubGroups: []string{"sub-group1"},
			},
			client: func(t *testing.T) GoCloak {
				m := mocks.NewMockGoCloak(t)

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
		{
			name: "fail to get group returns error",
			spec: &keycloakApi.KeycloakRealmGroupSpec{
				Name: testGroupName,
			},
			client: func(t *testing.T) GoCloak {
				m := mocks.NewMockGoCloak(t)

				m.On("GetGroups", mock.Anything, mock.Anything, "master", mock.Anything).
					Return(nil, errors.New("failed to get groups"))

				return m
			},
			wantErr: require.Error,
			want:    "",
		},
		{
			name: "fail to create group returns error",
			spec: &keycloakApi.KeycloakRealmGroupSpec{
				Name: testGroupName,
			},
			client: func(t *testing.T) GoCloak {
				m := mocks.NewMockGoCloak(t)

				m.On("GetGroups", mock.Anything, mock.Anything, "master", mock.Anything).
					Return([]*gocloak.Group{}, nil)

				m.On("CreateGroup", mock.Anything, mock.Anything, "master", mock.Anything).
					Return("", errors.New("failed to create group"))

				return m
			},
			wantErr: require.Error,
			want:    "",
		},
		{
			name: "fail to update group returns error",
			spec: &keycloakApi.KeycloakRealmGroupSpec{
				Name: testGroupName,
			},
			client: func(t *testing.T) GoCloak {
				m := mocks.NewMockGoCloak(t)

				m.On("GetGroups", mock.Anything, mock.Anything, "master", mock.Anything).
					Return([]*gocloak.Group{{
						ID:   gocloak.StringP("test-group-id"),
						Name: gocloak.StringP(testGroupName),
					}}, nil)

				m.On("UpdateGroup", mock.Anything, mock.Anything, "master", mock.Anything).
					Return(errors.New("failed to update group"))

				return m
			},
			wantErr: require.Error,
			want:    "",
		},
		{
			name: "fail to get role mappings returns error",
			spec: &keycloakApi.KeycloakRealmGroupSpec{
				Name: testGroupName,
			},
			client: func(t *testing.T) GoCloak {
				m := mocks.NewMockGoCloak(t)

				m.On("GetGroups", mock.Anything, mock.Anything, "master", mock.Anything).
					Return([]*gocloak.Group{{
						ID:   gocloak.StringP("test-group-id"),
						Name: gocloak.StringP(testGroupName),
					}}, nil)

				m.On("UpdateGroup", mock.Anything, mock.Anything, "master", mock.Anything).
					Return(nil)

				m.On("GetRoleMappingByGroupID", mock.Anything, mock.Anything, "master", "test-group-id").
					Return(nil, errors.New("failed to get role mappings"))

				return m
			},
			wantErr: require.Error,
			want:    "",
		},
		{
			name: "group with simple configuration",
			spec: &keycloakApi.KeycloakRealmGroupSpec{
				Name: testGroupName,
				Path: "/test-group",
			},
			client: func(t *testing.T) GoCloak {
				m := mocks.NewMockGoCloak(t)

				m.On("GetGroups", mock.Anything, mock.Anything, "master", mock.Anything).
					Return([]*gocloak.Group{{
						ID:   gocloak.StringP("test-group-id"),
						Name: gocloak.StringP(testGroupName),
					}}, nil)

				m.On("UpdateGroup", mock.Anything, mock.Anything, "master", mock.Anything).
					Return(nil)

				m.On("GetRoleMappingByGroupID", mock.Anything, mock.Anything, "master", "test-group-id").
					Return(&gocloak.MappingsRepresentation{
						RealmMappings:  &[]gocloak.Role{},
						ClientMappings: map[string]*gocloak.ClientMappingsRepresentation{},
					}, nil)

				m.On("RestyClient").Return(resty.New().SetBaseURL(server.URL))

				return m
			},
			wantErr: require.NoError,
			want:    "test-group-id",
		},
		{
			name: "unable to sync subgroups error",
			spec: &keycloakApi.KeycloakRealmGroupSpec{
				Name:      testGroupName,
				SubGroups: []string{"failing-subgroup"},
			},
			client: func(t *testing.T) GoCloak {
				m := mocks.NewMockGoCloak(t)

				// Mock group already exists
				m.On("GetGroups", mock.Anything, mock.Anything, "master", mock.Anything).
					Return(func(ctx context.Context, accessToken, realm string, params gocloak.GetGroupsParams) []*gocloak.Group {
						if params.Search != nil && *params.Search == testGroupName {
							return []*gocloak.Group{{
								ID:   gocloak.StringP("test-group-123"),
								Name: gocloak.StringP(testGroupName),
							}}
						}
						if params.Search != nil && *params.Search == "failing-subgroup" {
							return nil
						}
						return []*gocloak.Group{}
					}, func(ctx context.Context, accessToken, realm string, params gocloak.GetGroupsParams) error {
						if params.Search != nil && *params.Search == "failing-subgroup" {
							return errors.New("subgroup lookup failed")
						}
						return nil
					})

				// Mock UpdateGroup success
				m.On("UpdateGroup", mock.Anything, mock.Anything, "master", mock.Anything).
					Return(nil)

				// Mock GetRoleMappingByGroupID success
				m.On("GetRoleMappingByGroupID", mock.Anything, mock.Anything, "master", "test-group-123").
					Return(&gocloak.MappingsRepresentation{
						RealmMappings:  &[]gocloak.Role{},
						ClientMappings: map[string]*gocloak.ClientMappingsRepresentation{},
					}, nil)

				// Mock RestyClient for getChildGroups
				m.On("RestyClient").Return(resty.New().SetBaseURL(server.URL))

				return m
			},
			wantErr: require.Error,
			want:    "",
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

func TestGoCloakAdapter_DeleteGroup(t *testing.T) {
	tests := []struct {
		name       string
		realm      string
		groupName  string
		setupMocks func(t *testing.T) GoCloak
		wantErr    require.ErrorAssertionFunc
		errMsg     string
	}{
		{
			name:      "successful group deletion",
			realm:     "test-realm",
			groupName: testGroupName,
			setupMocks: func(t *testing.T) GoCloak {
				m := mocks.NewMockGoCloak(t)
				m.On("GetGroups", mock.Anything, "token", "test-realm",
					mock.MatchedBy(func(params gocloak.GetGroupsParams) bool {
						return params.Search != nil && *params.Search == testGroupName
					})).Return([]*gocloak.Group{
					{
						ID:   gocloak.StringP("group-id-123"),
						Name: gocloak.StringP(testGroupName),
					},
				}, nil)

				m.On("DeleteGroup", mock.Anything, "token", "test-realm", "group-id-123").
					Return(nil)

				return m
			},
			wantErr: require.NoError,
		},
		{
			name:      "group not found",
			realm:     "test-realm",
			groupName: "nonexistent-group",
			setupMocks: func(t *testing.T) GoCloak {
				m := mocks.NewMockGoCloak(t)
				m.On("GetGroups", mock.Anything, "token", "test-realm",
					mock.MatchedBy(func(params gocloak.GetGroupsParams) bool {
						return params.Search != nil && *params.Search == "nonexistent-group"
					})).Return([]*gocloak.Group{}, nil)

				return m
			},
			wantErr: require.Error,
			errMsg:  "group not found",
		},
		{
			name:      "getGroups API failure",
			realm:     "test-realm",
			groupName: testGroupName,
			setupMocks: func(t *testing.T) GoCloak {
				m := mocks.NewMockGoCloak(t)
				m.On("GetGroups", mock.Anything, "token", "test-realm",
					mock.MatchedBy(func(params gocloak.GetGroupsParams) bool {
						return params.Search != nil && *params.Search == testGroupName
					})).Return([]*gocloak.Group{}, errors.New("API error"))

				return m
			},
			wantErr: require.Error,
			errMsg:  "unable to search groups",
		},
		{
			name:      "deleteGroup API failure",
			realm:     "test-realm",
			groupName: testGroupName,
			setupMocks: func(t *testing.T) GoCloak {
				m := mocks.NewMockGoCloak(t)
				m.On("GetGroups", mock.Anything, "token", "test-realm",
					mock.MatchedBy(func(params gocloak.GetGroupsParams) bool {
						return params.Search != nil && *params.Search == testGroupName
					})).Return([]*gocloak.Group{
					{
						ID:   gocloak.StringP("group-id-123"),
						Name: gocloak.StringP(testGroupName),
					},
				}, nil)

				m.On("DeleteGroup", mock.Anything, "token", "test-realm", "group-id-123").
					Return(errors.New("delete failed"))

				return m
			},
			wantErr: require.Error,
			errMsg:  "unable to delete group",
		},
		{
			name:      "empty realm",
			realm:     "",
			groupName: testGroupName,
			setupMocks: func(t *testing.T) GoCloak {
				m := mocks.NewMockGoCloak(t)
				m.On("GetGroups", mock.Anything, "token", "",
					mock.MatchedBy(func(params gocloak.GetGroupsParams) bool {
						return params.Search != nil && *params.Search == testGroupName
					})).Return([]*gocloak.Group{}, errors.New("invalid realm"))

				return m
			},
			wantErr: require.Error,
			errMsg:  "unable to search groups",
		},
		{
			name:      "empty group name",
			realm:     "test-realm",
			groupName: "",
			setupMocks: func(t *testing.T) GoCloak {
				m := mocks.NewMockGoCloak(t)
				m.On("GetGroups", mock.Anything, "token", "test-realm",
					mock.MatchedBy(func(params gocloak.GetGroupsParams) bool {
						return params.Search != nil && *params.Search == ""
					})).Return([]*gocloak.Group{}, nil)

				return m
			},
			wantErr: require.Error,
			errMsg:  "group not found",
		},
		{
			name:      "group found in subgroups",
			realm:     "test-realm",
			groupName: "sub-group",
			setupMocks: func(t *testing.T) GoCloak {
				m := mocks.NewMockGoCloak(t)
				subGroups := []gocloak.Group{
					{
						ID:   gocloak.StringP("sub-group-id-456"),
						Name: gocloak.StringP("sub-group"),
					},
				}
				m.On("GetGroups", mock.Anything, "token", "test-realm",
					mock.MatchedBy(func(params gocloak.GetGroupsParams) bool {
						return params.Search != nil && *params.Search == "sub-group"
					})).Return([]*gocloak.Group{
					{
						ID:        gocloak.StringP("parent-group-id"),
						Name:      gocloak.StringP("parent-group"),
						SubGroups: &subGroups,
					},
				}, nil)

				m.On("DeleteGroup", mock.Anything, "token", "test-realm", "sub-group-id-456").
					Return(nil)

				return m
			},
			wantErr: require.NoError,
		},
		{
			name:      "multiple groups with same name",
			realm:     "test-realm",
			groupName: "duplicate-group",
			setupMocks: func(t *testing.T) GoCloak {
				m := mocks.NewMockGoCloak(t)
				m.On("GetGroups", mock.Anything, "token", "test-realm",
					mock.MatchedBy(func(params gocloak.GetGroupsParams) bool {
						return params.Search != nil && *params.Search == "duplicate-group"
					})).Return([]*gocloak.Group{
					{
						ID:   gocloak.StringP("group-id-1"),
						Name: gocloak.StringP("duplicate-group"),
					},
					{
						ID:   gocloak.StringP("group-id-2"),
						Name: gocloak.StringP("duplicate-group"),
					},
				}, nil)

				m.On("DeleteGroup", mock.Anything, "token", "test-realm", "group-id-1").
					Return(nil)

				return m
			},
			wantErr: require.NoError,
		},
		{
			name:      "nil group ID causes panic",
			realm:     "test-realm",
			groupName: testGroupName,
			setupMocks: func(t *testing.T) GoCloak {
				m := mocks.NewMockGoCloak(t)
				m.On("GetGroups", mock.Anything, "token", "test-realm",
					mock.MatchedBy(func(params gocloak.GetGroupsParams) bool {
						return params.Search != nil && *params.Search == testGroupName
					})).Return([]*gocloak.Group{
					{
						ID:   nil, // This will cause a panic when dereferencing *group.ID
						Name: gocloak.StringP(testGroupName),
					},
				}, nil)

				return m
			},
			wantErr: func(t require.TestingT, err error, _ ...interface{}) {
				// This test should panic due to nil pointer dereference
				// We'll handle this in the test runner
			},
		},
		{
			name:      "network timeout simulation",
			realm:     "test-realm",
			groupName: testGroupName,
			setupMocks: func(t *testing.T) GoCloak {
				m := mocks.NewMockGoCloak(t)
				m.On("GetGroups", mock.Anything, "token", "test-realm",
					mock.MatchedBy(func(params gocloak.GetGroupsParams) bool {
						return params.Search != nil && *params.Search == testGroupName
					})).Return([]*gocloak.Group{
					{
						ID:   gocloak.StringP("group-id-123"),
						Name: gocloak.StringP(testGroupName),
					},
				}, nil)

				m.On("DeleteGroup", mock.Anything, "token", "test-realm", "group-id-123").
					Return(context.DeadlineExceeded)

				return m
			},
			wantErr: require.Error,
			errMsg:  "context deadline exceeded",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			adapter := &GoCloakAdapter{
				client: tt.setupMocks(t),
				token:  &gocloak.JWT{AccessToken: "token"},
				log:    ctrl.Log.WithName("test"),
			}

			// Handle panic case for nil group ID
			if tt.name == "nil group ID causes panic" {
				assert.Panics(t, func() {
					_ = adapter.DeleteGroup(context.Background(), tt.realm, tt.groupName)
				}, "Expected panic due to nil group ID dereference")

				return
			}

			err := adapter.DeleteGroup(context.Background(), tt.realm, tt.groupName)

			tt.wantErr(t, err)

			if tt.errMsg != "" && err != nil {
				assert.Contains(t, err.Error(), tt.errMsg)
			}
		})
	}
}

func TestGoCloakAdapter_syncGroupRoles_Errors(t *testing.T) {
	tests := []struct {
		name       string
		spec       *keycloakApi.KeycloakRealmGroupSpec
		setupMocks func(t *testing.T) GoCloak
		wantErrMsg string
	}{
		{
			name: "unable to sync group realm roles error",
			spec: &keycloakApi.KeycloakRealmGroupSpec{
				Name:       "test-group",
				RealmRoles: []string{"admin"},
			},
			setupMocks: func(t *testing.T) GoCloak {
				m := mocks.NewMockGoCloak(t)

				// Mock GetRoleMappingByGroupID to return successful response
				m.On("GetRoleMappingByGroupID", mock.Anything, "token", "test-realm", "group-123").
					Return(&gocloak.MappingsRepresentation{
						RealmMappings:  &[]gocloak.Role{},
						ClientMappings: map[string]*gocloak.ClientMappingsRepresentation{},
					}, nil)

				// Mock GetRealmRole to fail, which will cause syncEntityRealmRoles to fail
				m.On("GetRealmRole", mock.Anything, "token", "test-realm", "admin").
					Return(nil, errors.New("failed to get realm role"))

				return m
			},
			wantErrMsg: "unable to sync group realm roles, groupID: group-123 with spec",
		},
		{
			name: "unable to sync client roles for group error",
			spec: &keycloakApi.KeycloakRealmGroupSpec{
				Name: "test-group",
				ClientRoles: []keycloakApi.UserClientRole{
					{
						ClientID: "test-client",
						Roles:    []string{"client-admin"},
					},
				},
			},
			setupMocks: func(t *testing.T) GoCloak {
				m := mocks.NewMockGoCloak(t)

				// Mock GetRoleMappingByGroupID to return successful response
				m.On("GetRoleMappingByGroupID", mock.Anything, "token", "test-realm", "group-123").
					Return(&gocloak.MappingsRepresentation{
						RealmMappings:  &[]gocloak.Role{},
						ClientMappings: map[string]*gocloak.ClientMappingsRepresentation{},
					}, nil)

				// Mock GetClients to fail during client role sync
				m.On("GetClients", mock.Anything, "token", "test-realm", mock.MatchedBy(func(params gocloak.GetClientsParams) bool {
					return params.ClientID != nil && *params.ClientID == "test-client"
				})).Return(nil, errors.New("client not found"))

				return m
			},
			wantErrMsg: "unable to sync client roles for group:",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			adapter := GoCloakAdapter{
				client: tt.setupMocks(t),
				token:  &gocloak.JWT{AccessToken: "token"},
				log:    logr.Discard(),
			}

			err := adapter.syncGroupRoles("test-realm", "group-123", tt.spec)

			require.Error(t, err)
			assert.Contains(t, err.Error(), tt.wantErrMsg)
		})
	}
}

func TestGoCloakAdapter_syncSubGroups_Errors(t *testing.T) {
	tests := []struct {
		name        string
		subGroups   []string
		setupMocks  func(t *testing.T, server *httptest.Server) GoCloak
		setupServer func(w http.ResponseWriter, r *http.Request)
		wantErrMsg  string
	}{
		{
			name:      "unable to get group error in syncSubGroups",
			subGroups: []string{"non-existent-subgroup"},
			setupServer: func(w http.ResponseWriter, r *http.Request) {
				if strings.Contains(r.URL.Path, "/children") {
					setJSONContentType(w)
					w.WriteHeader(http.StatusOK)
					_, err := w.Write([]byte(`[]`))
					assert.NoError(t, err)
					return
				}
			},
			setupMocks: func(t *testing.T, server *httptest.Server) GoCloak {
				m := mocks.NewMockGoCloak(t)

				// Mock RestyClient for getChildGroups
				m.On("RestyClient").Return(resty.New().SetBaseURL(server.URL))

				// Mock GetGroups to fail for subgroup lookup
				m.On("GetGroups", mock.Anything, "token", "test-realm", mock.MatchedBy(func(params gocloak.GetGroupsParams) bool {
					return params.Search != nil && *params.Search == "non-existent-subgroup"
				})).Return(nil, errors.New("group search failed"))

				return m
			},
			wantErrMsg: "unable to get group, realm: test-realm, group: non-existent-subgroup",
		},
		{
			name:      "unable to detach subgroup from group error",
			subGroups: []string{}, // Empty to trigger detach of existing subgroup
			setupServer: func(w http.ResponseWriter, r *http.Request) {
				if strings.Contains(r.URL.Path, "/children") {
					setJSONContentType(w)
					w.WriteHeader(http.StatusOK)
					// Return existing subgroup that should be detached
					_, err := w.Write([]byte(`[{"id": "subgroup-to-detach-123", "name": "subgroup-to-detach"}]`))
					assert.NoError(t, err)
					return
				}
			},
			setupMocks: func(t *testing.T, server *httptest.Server) GoCloak {
				m := mocks.NewMockGoCloak(t)

				// Mock RestyClient for getChildGroups
				m.On("RestyClient").Return(resty.New().SetBaseURL(server.URL))

				// Mock CreateGroup to fail during detach operation
				m.On("CreateGroup", mock.Anything, "token", "test-realm", mock.MatchedBy(func(group gocloak.Group) bool {
					return group.Name != nil && *group.Name == "subgroup-to-detach"
				})).Return("", errors.New("failed to detach subgroup"))

				return m
			},
			wantErrMsg: "unable to detach subgroup from group, realm: test-realm, subgroup: subgroup-to-detach, group:",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(tt.setupServer))
			t.Cleanup(server.Close)

			adapter := GoCloakAdapter{
				client: tt.setupMocks(t, server),
				token:  &gocloak.JWT{AccessToken: "token"},
				log:    logr.Discard(),
			}

			parentGroup := &gocloak.Group{
				ID:   gocloak.StringP("parent-group-123"),
				Name: gocloak.StringP("parent-group"),
			}

			err := adapter.syncSubGroups(context.Background(), "test-realm", parentGroup, tt.subGroups)

			require.Error(t, err)
			assert.Contains(t, err.Error(), tt.wantErrMsg)
		})
	}
}

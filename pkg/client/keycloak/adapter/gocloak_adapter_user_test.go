package adapter

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Nerzal/gocloak/v12"
	"github.com/go-logr/logr"
	"github.com/go-resty/resty/v2"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/epam/edp-keycloak-operator/pkg/client/keycloak/adapter/mocks"
)

func TestGoCloakAdapter_SyncRealmUser(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "users/user-with-groups-id/groups") {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)

			_, err := w.Write([]byte(`[{"id":"group1-id","name":"group1"},{"id":"group2-id","name":"group2"}]`))
			assert.NoError(t, err)

			return
		}

		if strings.Contains(r.URL.Path, "identity-provider/instances/idp1") {
			w.WriteHeader(http.StatusOK)
			_, err := w.Write([]byte(`{"alias":"idp1"}`))
			assert.NoError(t, err)

			return
		}

		if strings.Contains(r.URL.Path, "identity-provider/instances/non-existent-idp") {
			w.WriteHeader(http.StatusNotFound)
			_, err := w.Write([]byte(`{"error":"idp not found"}`))
			assert.NoError(t, err)

			return
		}

		w.WriteHeader(http.StatusOK)
	}))

	t.Cleanup(func() {
		server.Close()
	})

	tests := []struct {
		name    string
		userDto *KeycloakUser
		client  func(t *testing.T) *mocks.MockGoCloak
		wantErr require.ErrorAssertionFunc
	}{
		{
			name: "create user success",
			userDto: &KeycloakUser{
				Username:            "user",
				Enabled:             true,
				EmailVerified:       true,
				Email:               "mail@mail.com",
				FirstName:           "first-name",
				LastName:            "last-name",
				RequiredUserActions: []string{"change-password"},
				Roles:               []string{"role1"},
				Groups:              []string{"group1"},
				Attributes:          map[string][]string{"attr1": {"attr1value"}},
				Password:            "password",
				IdentityProviders:   &[]string{"idp1"},
			},
			client: func(t *testing.T) *mocks.MockGoCloak {
				m := mocks.NewMockGoCloak(t)

				m.On("GetUsers", mock.Anything, "", "realm", mock.Anything).
					Return(nil, nil)
				m.On("CreateUser",
					mock.Anything,
					"",
					"realm",
					mock.MatchedBy(func(user gocloak.User) bool {
						return assert.Equal(t, "user", *user.Username)
					})).
					Return("user-id", nil)
				m.On("GetRoleMappingByUserID", mock.Anything, "", "realm", "user-id").
					Return(&gocloak.MappingsRepresentation{
						RealmMappings:  &[]gocloak.Role{},
						ClientMappings: map[string]*gocloak.ClientMappingsRepresentation{},
					}, nil)
				m.On("GetRealmRole", mock.Anything, "", "realm", "role1").
					Return(&gocloak.Role{
						Name: gocloak.StringP("role1"),
						ID:   gocloak.StringP("role1-id"),
					}, nil)
				m.On("AddRealmRoleToUser",
					mock.Anything,
					"",
					"realm",
					"user-id",
					mock.MatchedBy(func(roles []gocloak.Role) bool {
						return assert.Len(t, roles, 1) &&
							assert.Equal(t, "role1-id", *roles[0].ID)
					})).
					Return(nil)
				m.On("GetGroups",
					mock.Anything,
					"",
					"realm",
					mock.Anything).
					Return([]*gocloak.Group{{
						Name: gocloak.StringP("group1"),
						ID:   gocloak.StringP("group1-id"),
					}}, nil)
				m.On("RestyClient").Return(resty.New())
				m.On("GetUserFederatedIdentities",
					mock.Anything,
					"",
					"realm",
					"user-id").
					Return([]*gocloak.FederatedIdentityRepresentation{{IdentityProvider: gocloak.StringP("idp2")}}, nil)
				m.On("CreateUserFederatedIdentity",
					mock.Anything,
					"",
					"realm",
					"user-id",
					"idp1",
					mock.Anything).
					Return(nil)
				m.On("DeleteUserFederatedIdentity",
					mock.Anything,
					"",
					"realm",
					"user-id",
					"idp2",
					mock.Anything).
					Return(nil)

				return m
			},
			wantErr: require.NoError,
		},
		{
			name: "update user success",
			userDto: &KeycloakUser{
				Username:            "user",
				Enabled:             true,
				EmailVerified:       true,
				Email:               "mail@mail.com",
				FirstName:           "first-name",
				LastName:            "last-name",
				RequiredUserActions: []string{"change-password"},
				Roles:               []string{"role1"},
				Groups:              []string{"group1", "group3"},
				Attributes:          map[string][]string{"attr1": {"attr1value"}},
				Password:            "password",
			},
			client: func(t *testing.T) *mocks.MockGoCloak {
				m := mocks.NewMockGoCloak(t)

				m.On("GetUsers", mock.Anything, "", "realm", mock.Anything).
					Return([]*gocloak.User{{
						ID:       gocloak.StringP("user-with-groups-id"),
						Username: gocloak.StringP("user"),
					}}, nil)
				m.On("UpdateUser",
					mock.Anything,
					"",
					"realm",
					mock.MatchedBy(func(user gocloak.User) bool {
						return assert.Equal(t, "user", *user.Username)
					})).
					Return(nil)
				m.On("GetRoleMappingByUserID", mock.Anything, "", "realm", "user-with-groups-id").
					Return(&gocloak.MappingsRepresentation{
						RealmMappings:  &[]gocloak.Role{},
						ClientMappings: map[string]*gocloak.ClientMappingsRepresentation{},
					}, nil)
				m.On("GetRealmRole", mock.Anything, "", "realm", "role1").
					Return(&gocloak.Role{
						Name: gocloak.StringP("role1"),
						ID:   gocloak.StringP("role1-id"),
					}, nil)
				m.On("AddRealmRoleToUser",
					mock.Anything,
					"",
					"realm",
					"user-with-groups-id",
					mock.MatchedBy(func(roles []gocloak.Role) bool {
						return assert.Len(t, roles, 1) &&
							assert.Equal(t, "role1-id", *roles[0].ID)
					})).
					Return(nil)
				m.On("GetGroups",
					mock.Anything,
					"",
					"realm",
					mock.Anything).
					Return([]*gocloak.Group{
						{
							Name: gocloak.StringP("group1"),
							ID:   gocloak.StringP("group1-id"),
						},
						{
							Name: gocloak.StringP("group2"),
							ID:   gocloak.StringP("group2-id"),
						},
						{
							Name: gocloak.StringP("group3"),
							ID:   gocloak.StringP("group3-id"),
						},
					}, nil)
				m.On("RestyClient").Return(resty.New())

				return m
			},
			wantErr: require.NoError,
		},
		{
			name: "failed to get groups",
			userDto: &KeycloakUser{
				Username:            "user",
				Enabled:             true,
				EmailVerified:       true,
				Email:               "mail@mail.com",
				FirstName:           "first-name",
				LastName:            "last-name",
				RequiredUserActions: []string{"change-password"},
				Roles:               []string{"role1"},
				Groups:              []string{"group1"},
				Attributes:          map[string][]string{"attr1": {"attr1value"}},
				Password:            "password",
			},
			client: func(t *testing.T) *mocks.MockGoCloak {
				m := mocks.NewMockGoCloak(t)

				m.On("GetUsers", mock.Anything, "", "realm", mock.Anything).
					Return(nil, nil)
				m.On("CreateUser",
					mock.Anything,
					"",
					"realm",
					mock.MatchedBy(func(user gocloak.User) bool {
						return assert.Equal(t, "user", *user.Username)
					})).
					Return("user-id", nil)
				m.On("GetRoleMappingByUserID", mock.Anything, "", "realm", "user-id").
					Return(&gocloak.MappingsRepresentation{
						RealmMappings:  &[]gocloak.Role{},
						ClientMappings: map[string]*gocloak.ClientMappingsRepresentation{},
					}, nil)
				m.On("GetRealmRole", mock.Anything, "", "realm", "role1").
					Return(&gocloak.Role{
						Name: gocloak.StringP("role1"),
						ID:   gocloak.StringP("role1-id"),
					}, nil)
				m.On("AddRealmRoleToUser",
					mock.Anything,
					"",
					"realm",
					"user-id",
					mock.MatchedBy(func(roles []gocloak.Role) bool {
						return assert.Len(t, roles, 1) &&
							assert.Equal(t, "role1-id", *roles[0].ID)
					})).
					Return(nil)
				m.On("GetGroups",
					mock.Anything,
					"",
					"realm",
					mock.Anything).
					Return(nil, errors.New("failed to get groups"))
				m.On("RestyClient").Return(resty.New())

				return m
			},
			wantErr: func(t require.TestingT, err error, i ...interface{}) {
				require.Error(t, err)
				assert.Contains(t, err.Error(), "failed to get groups")
			},
		},
		{
			name: "failed to get roles",
			userDto: &KeycloakUser{
				Username:            "user",
				Enabled:             true,
				EmailVerified:       true,
				Email:               "mail@mail.com",
				FirstName:           "first-name",
				LastName:            "last-name",
				RequiredUserActions: []string{"change-password"},
				Roles:               []string{"role1"},
				Groups:              []string{"group1"},
				Attributes:          map[string][]string{"attr1": {"attr1value"}},
				Password:            "password",
			},
			client: func(t *testing.T) *mocks.MockGoCloak {
				m := mocks.NewMockGoCloak(t)

				m.On("GetUsers", mock.Anything, "", "realm", mock.Anything).
					Return([]*gocloak.User{{
						ID:       gocloak.StringP("user-id"),
						Username: gocloak.StringP("user"),
					}}, nil)
				m.On("UpdateUser",
					mock.Anything,
					"",
					"realm",
					mock.MatchedBy(func(user gocloak.User) bool {
						return assert.Equal(t, "user", *user.Username)
					})).
					Return(nil)
				m.On("GetRoleMappingByUserID", mock.Anything, "", "realm", "user-id").
					Return(&gocloak.MappingsRepresentation{
						RealmMappings:  &[]gocloak.Role{},
						ClientMappings: map[string]*gocloak.ClientMappingsRepresentation{},
					}, nil)
				m.On("GetRealmRole", mock.Anything, "", "realm", "role1").
					Return(nil, errors.New("failed to get roles"))
				m.On("RestyClient").Return(resty.New())

				return m
			},
			wantErr: func(t require.TestingT, err error, i ...interface{}) {
				require.Error(t, err)
				assert.Contains(t, err.Error(), "failed to get roles")
			},
		},
		{
			name: "failed to create user",
			userDto: &KeycloakUser{
				Username:            "user",
				Enabled:             true,
				EmailVerified:       true,
				Email:               "mail@mail.com",
				FirstName:           "first-name",
				LastName:            "last-name",
				RequiredUserActions: []string{"change-password"},
				Roles:               []string{"role1"},
				Groups:              []string{"group1"},
				Attributes:          map[string][]string{"attr1": {"attr1value"}},
				Password:            "password",
			},
			client: func(t *testing.T) *mocks.MockGoCloak {
				m := mocks.NewMockGoCloak(t)

				m.On("GetUsers", mock.Anything, "", "realm", mock.Anything).
					Return(nil, nil)
				m.On("CreateUser",
					mock.Anything,
					"",
					"realm",
					mock.MatchedBy(func(user gocloak.User) bool {
						return assert.Equal(t, "user", *user.Username)
					})).
					Return("", errors.New("failed to create user"))

				return m
			},
			wantErr: func(t require.TestingT, err error, i ...interface{}) {
				require.Error(t, err)
				assert.Contains(t, err.Error(), "failed to create user")
			},
		},
		{
			name: "failed to get user",
			userDto: &KeycloakUser{
				Username:            "user",
				Enabled:             true,
				EmailVerified:       true,
				Email:               "mail@mail.com",
				FirstName:           "first-name",
				LastName:            "last-name",
				RequiredUserActions: []string{"change-password"},
				Roles:               []string{"role1"},
				Groups:              []string{"group1"},
				Attributes:          map[string][]string{"attr1": {"attr1value"}},
				Password:            "password",
			},
			client: func(t *testing.T) *mocks.MockGoCloak {
				m := mocks.NewMockGoCloak(t)

				m.On("GetUsers", mock.Anything, "", "realm", mock.Anything).
					Return(nil, errors.New("failed to get user"))

				return m
			},
			wantErr: func(t require.TestingT, err error, i ...interface{}) {
				require.Error(t, err)
				assert.Contains(t, err.Error(), "failed to get user")
			},
		},
		{
			name: "identity provider does not exist",
			userDto: &KeycloakUser{
				Username:            "user",
				Enabled:             true,
				EmailVerified:       true,
				Email:               "mail@mail.com",
				FirstName:           "first-name",
				LastName:            "last-name",
				RequiredUserActions: []string{"change-password"},
				Roles:               []string{"role1"},
				Groups:              []string{"group1"},
				Attributes:          map[string][]string{"attr1": {"attr1value"}},
				Password:            "password",
				IdentityProviders:   &[]string{"non-existent-idp"},
			},
			client: func(t *testing.T) *mocks.MockGoCloak {
				m := mocks.NewMockGoCloak(t)

				m.On("GetUsers", mock.Anything, "", "realm", mock.Anything).
					Return(nil, nil)
				m.On("CreateUser",
					mock.Anything,
					"",
					"realm",
					mock.MatchedBy(func(user gocloak.User) bool {
						return assert.Equal(t, "user", *user.Username)
					})).
					Return("user-id", nil)
				m.On("GetRoleMappingByUserID", mock.Anything, "", "realm", "user-id").
					Return(&gocloak.MappingsRepresentation{
						RealmMappings:  &[]gocloak.Role{},
						ClientMappings: map[string]*gocloak.ClientMappingsRepresentation{},
					}, nil)
				m.On("GetRealmRole", mock.Anything, "", "realm", "role1").
					Return(&gocloak.Role{
						Name: gocloak.StringP("role1"),
						ID:   gocloak.StringP("role1-id"),
					}, nil)
				m.On("AddRealmRoleToUser",
					mock.Anything,
					"",
					"realm",
					"user-id",
					mock.MatchedBy(func(roles []gocloak.Role) bool {
						return assert.Len(t, roles, 1) &&
							assert.Equal(t, "role1-id", *roles[0].ID)
					})).
					Return(nil)
				m.On("GetGroups",
					mock.Anything,
					"",
					"realm",
					mock.Anything).
					Return([]*gocloak.Group{{
						Name: gocloak.StringP("group1"),
						ID:   gocloak.StringP("group1-id"),
					}}, nil)
				m.On("RestyClient").Return(resty.New())
				m.On("GetUserFederatedIdentities",
					mock.Anything,
					"",
					"realm",
					"user-id").
					Return([]*gocloak.FederatedIdentityRepresentation{}, nil)

				return m
			},
			wantErr: func(t require.TestingT, err error, i ...interface{}) {
				require.Error(t, err)
				assert.Contains(t, err.Error(), "identity provider non-existent-idp does not exist")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			a, err := Make(context.Background(), GoCloakConfig{Url: server.URL}, logr.Discard(), nil)
			a.client = tt.client(t)

			require.NoError(t, err)

			tt.wantErr(t, a.SyncRealmUser(
				context.Background(),
				"realm",
				tt.userDto,
				false,
			))
		})
	}
}

package adapter

import (
	"errors"
	"testing"

	"github.com/Nerzal/gocloak/v12"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/epam/edp-keycloak-operator/pkg/client/keycloak/adapter/mocks"
)

func TestGoCloakAdapter_SetServiceAccountAttributes(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		attributes map[string][]string
		addOnly    bool
		client     func(t *testing.T) *mocks.MockGoCloak
		wantErr    require.ErrorAssertionFunc
	}{
		{
			name: "add only mode with existing attributes",
			attributes: map[string][]string{
				"foo": {"bar"},
			},
			addOnly: true,
			client: func(t *testing.T) *mocks.MockGoCloak {
				m := mocks.NewMockGoCloak(t)

				usr1 := gocloak.User{
					Username: gocloak.StringP("user1"),
					Attributes: &map[string][]string{
						"foo1": {"bar1"},
					},
				}

				usr2 := gocloak.User{
					Username: gocloak.StringP("user1"),
					Attributes: &map[string][]string{
						"foo":  {"bar"},
						"foo1": {"bar1"},
					},
				}

				m.On("GetClientServiceAccount", mock.Anything, "token", "realm1", "clientID1").Return(&usr1, nil)
				m.On("UpdateUser", mock.Anything, "token", "realm1", mock.MatchedBy(func(user gocloak.User) bool {
					return assert.Equal(t, *usr2.Username, *user.Username) && assert.Equal(t, *usr2.Attributes, *user.Attributes)
				})).Return(nil)

				return m
			},
			wantErr: require.NoError,
		},
		{
			name: "add only mode with overlapping attributes",
			attributes: map[string][]string{
				"foo": {"bar", "baz"},
			},
			addOnly: true,
			client: func(t *testing.T) *mocks.MockGoCloak {
				m := mocks.NewMockGoCloak(t)

				usr1 := gocloak.User{
					Username: gocloak.StringP("user1"),
					Attributes: &map[string][]string{
						"foo": {"bar"},
					},
				}

				usr2 := gocloak.User{
					Username: gocloak.StringP("user1"),
					Attributes: &map[string][]string{
						"foo": {"bar", "baz"},
					},
				}

				m.On("GetClientServiceAccount", mock.Anything, "token", "realm1", "clientID1").Return(&usr1, nil)
				m.On("UpdateUser", mock.Anything, "token", "realm1", mock.MatchedBy(func(user gocloak.User) bool {
					return assert.Equal(t, *usr2.Username, *user.Username) && assert.Equal(t, *usr2.Attributes, *user.Attributes)
				})).Return(nil)

				return m
			},
			wantErr: require.NoError,
		},
		{
			name: "overwrite mode",
			attributes: map[string][]string{
				"foo": {"bar"},
			},
			addOnly: false,
			client: func(t *testing.T) *mocks.MockGoCloak {
				m := mocks.NewMockGoCloak(t)

				usr1 := gocloak.User{
					Username: gocloak.StringP("user1"),
					Attributes: &map[string][]string{
						"foo1": {"bar1"},
					},
				}

				usr2 := gocloak.User{
					Username: gocloak.StringP("user1"),
					Attributes: &map[string][]string{
						"foo": {"bar"},
					},
				}

				m.On("GetClientServiceAccount", mock.Anything, "token", "realm1", "clientID1").Return(&usr1, nil)
				m.On("UpdateUser", mock.Anything, "token", "realm1", mock.MatchedBy(func(user gocloak.User) bool {
					return assert.Equal(t, *usr2.Username, *user.Username) && assert.Equal(t, *usr2.Attributes, *user.Attributes)
				})).Return(nil)

				return m
			},
			wantErr: require.NoError,
		},
		{
			name: "error getting client service account",
			attributes: map[string][]string{
				"foo": {"bar"},
			},
			addOnly: false,
			client: func(t *testing.T) *mocks.MockGoCloak {
				m := mocks.NewMockGoCloak(t)

				m.On(
					"GetClientServiceAccount",
					mock.Anything,
					"token",
					"realm1",
					"clientID1",
				).Return(nil, errors.New("service account not found"))

				return m
			},
			wantErr: func(t require.TestingT, err error, i ...interface{}) {
				require.Error(t, err)
				assert.Contains(t, err.Error(), "unable to get client service account")
				assert.Contains(t, err.Error(), "service account not found")
			},
		},
		{
			name: "error updating service account user",
			attributes: map[string][]string{
				"foo": {"bar"},
			},
			addOnly: false,
			client: func(t *testing.T) *mocks.MockGoCloak {
				m := mocks.NewMockGoCloak(t)

				usr1 := gocloak.User{
					Username: gocloak.StringP("user1"),
					Attributes: &map[string][]string{
						"foo1": {"bar1"},
					},
				}

				m.On("GetClientServiceAccount", mock.Anything, "token", "realm1", "clientID1").Return(&usr1, nil)
				m.On("UpdateUser", mock.Anything, "token", "realm1", mock.Anything).Return(errors.New("update failed"))

				return m
			},
			wantErr: func(t require.TestingT, err error, i ...interface{}) {
				require.Error(t, err)
				assert.Contains(t, err.Error(), "unable to update service account user: clientID1")
				assert.Contains(t, err.Error(), "update failed")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			adapter := GoCloakAdapter{
				client:   tt.client(t),
				basePath: "",
				token:    &gocloak.JWT{AccessToken: "token"},
			}

			tt.wantErr(t, adapter.SetServiceAccountAttributes("realm1", "clientID1", tt.attributes, tt.addOnly))
		})
	}
}

func TestGoCloakAdapter_SyncServiceAccountRoles(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		realm       string
		clientID    string
		realmRoles  []string
		clientRoles map[string][]string
		addOnly     bool
		client      func(t *testing.T) *mocks.MockGoCloak
		wantErr     require.ErrorAssertionFunc
	}{
		{
			name:        "sync service account roles success",
			realm:       "realm",
			clientID:    "client-id",
			realmRoles:  []string{"role1", "role2"},
			clientRoles: map[string][]string{"client1": {"client-role1"}},
			addOnly:     false,
			client: func(t *testing.T) *mocks.MockGoCloak {
				m := mocks.NewMockGoCloak(t)

				serviceAccountUser := &gocloak.User{
					ID:       gocloak.StringP("service-account-id"),
					Username: gocloak.StringP("service-account-user"),
				}

				m.On("GetClientServiceAccount", mock.Anything, "", "realm", "client-id").
					Return(serviceAccountUser, nil)
				m.On("GetRoleMappingByUserID", mock.Anything, "", "realm", "service-account-id").
					Return(&gocloak.MappingsRepresentation{
						RealmMappings:  &[]gocloak.Role{},
						ClientMappings: map[string]*gocloak.ClientMappingsRepresentation{},
					}, nil)
				m.On("GetRealmRole", mock.Anything, "", "realm", "role1").
					Return(&gocloak.Role{
						Name: gocloak.StringP("role1"),
						ID:   gocloak.StringP("role1-id"),
					}, nil)
				m.On("GetRealmRole", mock.Anything, "", "realm", "role2").
					Return(&gocloak.Role{
						Name: gocloak.StringP("role2"),
						ID:   gocloak.StringP("role2-id"),
					}, nil)
				m.On("AddRealmRoleToUser",
					mock.Anything,
					"",
					"realm",
					"service-account-id",
					mock.MatchedBy(func(roles []gocloak.Role) bool {
						return assert.Len(t, roles, 2) &&
							assert.Equal(t, "role1-id", *roles[0].ID) &&
							assert.Equal(t, "role2-id", *roles[1].ID)
					})).
					Return(nil)
				m.On("GetClients", mock.Anything, "", "realm", mock.Anything).
					Return([]*gocloak.Client{{
						ID:       gocloak.StringP("client1-id"),
						ClientID: gocloak.StringP("client1"),
					}}, nil)
				m.On("GetClientRole", mock.Anything, "", "realm", "client1-id", "client-role1").
					Return(&gocloak.Role{
						Name: gocloak.StringP("client-role1"),
						ID:   gocloak.StringP("client-role1-id"),
					}, nil)
				m.On("AddClientRoleToUser",
					mock.Anything,
					"",
					"realm",
					"client1-id",
					"service-account-id",
					mock.MatchedBy(func(roles []gocloak.Role) bool {
						return assert.Len(t, roles, 1) &&
							assert.Equal(t, "client-role1-id", *roles[0].ID)
					})).
					Return(nil)

				return m
			},
			wantErr: require.NoError,
		},
		{
			name:        "sync service account roles add only",
			realm:       "realm",
			clientID:    "client-id",
			realmRoles:  []string{"role1"},
			clientRoles: map[string][]string{},
			addOnly:     true,
			client: func(t *testing.T) *mocks.MockGoCloak {
				m := mocks.NewMockGoCloak(t)

				serviceAccountUser := &gocloak.User{
					ID:       gocloak.StringP("service-account-id"),
					Username: gocloak.StringP("service-account-user"),
				}

				m.On("GetClientServiceAccount", mock.Anything, "", "realm", "client-id").
					Return(serviceAccountUser, nil)
				m.On("GetRoleMappingByUserID", mock.Anything, "", "realm", "service-account-id").
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
					"service-account-id",
					mock.MatchedBy(func(roles []gocloak.Role) bool {
						return assert.Len(t, roles, 1) &&
							assert.Equal(t, "role1-id", *roles[0].ID)
					})).
					Return(nil)

				return m
			},
			wantErr: require.NoError,
		},
		{
			name:        "failed to get service account",
			realm:       "realm",
			clientID:    "client-id",
			realmRoles:  []string{"role1"},
			clientRoles: map[string][]string{},
			addOnly:     false,
			client: func(t *testing.T) *mocks.MockGoCloak {
				m := mocks.NewMockGoCloak(t)

				m.On("GetClientServiceAccount", mock.Anything, "", "realm", "client-id").
					Return(nil, errors.New("failed to get service account"))

				return m
			},
			wantErr: func(t require.TestingT, err error, i ...interface{}) {
				require.Error(t, err)
				assert.Contains(t, err.Error(), "failed to get service account")
			},
		},
		{
			name:        "failed to get role mapping",
			realm:       "realm",
			clientID:    "client-id",
			realmRoles:  []string{"role1"},
			clientRoles: map[string][]string{},
			addOnly:     false,
			client: func(t *testing.T) *mocks.MockGoCloak {
				m := mocks.NewMockGoCloak(t)

				serviceAccountUser := &gocloak.User{
					ID:       gocloak.StringP("service-account-id"),
					Username: gocloak.StringP("service-account-user"),
				}

				m.On("GetClientServiceAccount", mock.Anything, "", "realm", "client-id").
					Return(serviceAccountUser, nil)
				m.On("GetRoleMappingByUserID", mock.Anything, "", "realm", "service-account-id").
					Return(nil, errors.New("failed to get role mapping"))

				return m
			},
			wantErr: func(t require.TestingT, err error, i ...interface{}) {
				require.Error(t, err)
				assert.Contains(t, err.Error(), "failed to get role mapping")
			},
		},
		{
			name:        "failed to get realm role",
			realm:       "realm",
			clientID:    "client-id",
			realmRoles:  []string{"role1"},
			clientRoles: map[string][]string{},
			addOnly:     false,
			client: func(t *testing.T) *mocks.MockGoCloak {
				m := mocks.NewMockGoCloak(t)

				serviceAccountUser := &gocloak.User{
					ID:       gocloak.StringP("service-account-id"),
					Username: gocloak.StringP("service-account-user"),
				}

				m.On("GetClientServiceAccount", mock.Anything, "", "realm", "client-id").
					Return(serviceAccountUser, nil)
				m.On("GetRoleMappingByUserID", mock.Anything, "", "realm", "service-account-id").
					Return(&gocloak.MappingsRepresentation{
						RealmMappings:  &[]gocloak.Role{},
						ClientMappings: map[string]*gocloak.ClientMappingsRepresentation{},
					}, nil)
				m.On("GetRealmRole", mock.Anything, "", "realm", "role1").
					Return(nil, errors.New("failed to get realm role"))

				return m
			},
			wantErr: func(t require.TestingT, err error, i ...interface{}) {
				require.Error(t, err)
				assert.Contains(t, err.Error(), "failed to get realm role")
			},
		},
		{
			name:        "failed to get client ID",
			realm:       "realm",
			clientID:    "client-id",
			realmRoles:  []string{},
			clientRoles: map[string][]string{"client1": {"client-role1"}},
			addOnly:     false,
			client: func(t *testing.T) *mocks.MockGoCloak {
				m := mocks.NewMockGoCloak(t)

				serviceAccountUser := &gocloak.User{
					ID:       gocloak.StringP("service-account-id"),
					Username: gocloak.StringP("service-account-user"),
				}

				m.On("GetClientServiceAccount", mock.Anything, "", "realm", "client-id").
					Return(serviceAccountUser, nil)
				m.On("GetRoleMappingByUserID", mock.Anything, "", "realm", "service-account-id").
					Return(&gocloak.MappingsRepresentation{
						RealmMappings:  &[]gocloak.Role{},
						ClientMappings: map[string]*gocloak.ClientMappingsRepresentation{},
					}, nil)
				m.On("GetClients", mock.Anything, "", "realm", mock.Anything).
					Return(nil, errors.New("failed to get clients"))

				return m
			},
			wantErr: func(t require.TestingT, err error, i ...interface{}) {
				require.Error(t, err)
				assert.Contains(t, err.Error(), "failed to get clients")
			},
		},
		{
			name:        "failed to get client role",
			realm:       "realm",
			clientID:    "client-id",
			realmRoles:  []string{},
			clientRoles: map[string][]string{"client1": {"client-role1"}},
			addOnly:     false,
			client: func(t *testing.T) *mocks.MockGoCloak {
				m := mocks.NewMockGoCloak(t)

				serviceAccountUser := &gocloak.User{
					ID:       gocloak.StringP("service-account-id"),
					Username: gocloak.StringP("service-account-user"),
				}

				m.On("GetClientServiceAccount", mock.Anything, "", "realm", "client-id").
					Return(serviceAccountUser, nil)
				m.On("GetRoleMappingByUserID", mock.Anything, "", "realm", "service-account-id").
					Return(&gocloak.MappingsRepresentation{
						RealmMappings:  &[]gocloak.Role{},
						ClientMappings: map[string]*gocloak.ClientMappingsRepresentation{},
					}, nil)
				m.On("GetClients", mock.Anything, "", "realm", mock.Anything).
					Return([]*gocloak.Client{{
						ID:       gocloak.StringP("client1-id"),
						ClientID: gocloak.StringP("client1"),
					}}, nil)
				m.On("GetClientRole", mock.Anything, "", "realm", "client1-id", "client-role1").
					Return(nil, errors.New("unable to get client role"))

				return m
			},
			wantErr: func(t require.TestingT, err error, i ...interface{}) {
				require.Error(t, err)
				assert.Contains(t, err.Error(), "unable to get client role")
			},
		},
		{
			name:        "sync with existing roles",
			realm:       "realm",
			clientID:    "client-id",
			realmRoles:  []string{"role1"},
			clientRoles: map[string][]string{},
			addOnly:     false,
			client: func(t *testing.T) *mocks.MockGoCloak {
				m := mocks.NewMockGoCloak(t)

				serviceAccountUser := &gocloak.User{
					ID:       gocloak.StringP("service-account-id"),
					Username: gocloak.StringP("service-account-user"),
				}

				existingRealmRoles := []gocloak.Role{
					{
						Name: gocloak.StringP("existing-role"),
						ID:   gocloak.StringP("existing-role-id"),
					},
				}

				m.On("GetClientServiceAccount", mock.Anything, "", "realm", "client-id").
					Return(serviceAccountUser, nil)
				m.On("GetRoleMappingByUserID", mock.Anything, "", "realm", "service-account-id").
					Return(&gocloak.MappingsRepresentation{
						RealmMappings:  &existingRealmRoles,
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
					"service-account-id",
					mock.MatchedBy(func(roles []gocloak.Role) bool {
						return assert.Len(t, roles, 1) &&
							assert.Equal(t, "role1-id", *roles[0].ID)
					})).
					Return(nil)
				m.On("DeleteRealmRoleFromUser",
					mock.Anything,
					"",
					"realm",
					"service-account-id",
					mock.MatchedBy(func(roles []gocloak.Role) bool {
						return assert.Len(t, roles, 1) &&
							assert.Equal(t, "existing-role-id", *roles[0].ID)
					})).
					Return(nil)

				return m
			},
			wantErr: require.NoError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			a := GoCloakAdapter{
				client:   tt.client(t),
				basePath: "",
				token:    &gocloak.JWT{AccessToken: ""},
			}

			tt.wantErr(t, a.SyncServiceAccountRoles(
				tt.realm,
				tt.clientID,
				tt.realmRoles,
				tt.clientRoles,
				tt.addOnly,
			))
		})
	}
}

package adapter

import (
	"context"
	"errors"
	"testing"

	"github.com/Nerzal/gocloak/v12"
	"github.com/go-logr/logr"
	"github.com/stretchr/testify/assert"
	testifymock "github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	ctrl "sigs.k8s.io/controller-runtime"

	"github.com/epam/edp-keycloak-operator/pkg/client/keycloak/adapter/mocks"
	"github.com/epam/edp-keycloak-operator/pkg/client/keycloak/dto"
)

func TestGoCloakAdapter_SyncRealmRole(t *testing.T) {
	t.Parallel()

	var (
		token = "token"
		realm = "realm"
	)

	tests := []struct {
		name    string
		role    *dto.PrimaryRealmRole
		client  func(t *testing.T) *mocks.MockGoCloak
		wantErr require.ErrorAssertionFunc
	}{
		{
			name: "should update associated roles",
			role: &dto.PrimaryRealmRole{
				Name: "role1",
				// should add role3
				Composites: []string{"role2", "role3"},
				// should add role5
				CompositesClientRoles: map[string][]string{"client1": {"role4", "role5"}},
				IsComposite:           true,
				Description:           "Role description",
				Attributes:            map[string][]string{"foo": {"foo", "bar"}},
				IsDefault:             false,
			},
			client: func(t *testing.T) *mocks.MockGoCloak {
				m := mocks.NewMockGoCloak(t)

				m.On(
					"GetCompositeRolesByRoleID",
					testifymock.Anything,
					token,
					realm,
					"role1-id").
					Return([]*gocloak.Role{
						{
							Name: gocloak.StringP("role2"),
						},
						// should be removed from composites
						{
							Name: gocloak.StringP("role6"),
						},
						{
							Name:        gocloak.StringP("role4"),
							ClientRole:  gocloak.BoolP(true),
							ContainerID: gocloak.StringP("client1-id"),
						},
						// should be removed from composites
						{
							Name:        gocloak.StringP("role7"),
							ClientRole:  gocloak.BoolP(true),
							ContainerID: gocloak.StringP("client1-id"),
						},
					}, nil)

				m.On("GetRealmRole", testifymock.Anything, token, realm, testifymock.Anything).Return(
					func(ctx context.Context, token, realm, roleName string) (*gocloak.Role, error) {
						switch roleName {
						case "role1":
							return &gocloak.Role{Name: gocloak.StringP("role1"), ID: gocloak.StringP("role1-id")}, nil
						case "role3":
							return &gocloak.Role{Name: gocloak.StringP("role3"), ID: gocloak.StringP("role3-id")}, nil
						default:
							return nil, errors.New("unknown role")
						}
					})

				m.On(
					"GetClients",
					testifymock.Anything,
					token,
					realm,
					testifymock.Anything).
					Return([]*gocloak.Client{{
						ID:   gocloak.StringP("client1-id"),
						Name: gocloak.StringP("client1"),
					}}, nil)

				m.On("GetClientRole", testifymock.Anything, token, realm, testifymock.Anything, testifymock.Anything).Return(
					func(ctx context.Context, token, realm, clientID, roleName string) (*gocloak.Role, error) {
						switch roleName {
						case "role5":
							return &gocloak.Role{Name: gocloak.StringP("role5"), ID: gocloak.StringP("role5-id"), ClientRole: gocloak.BoolP(true)}, nil
						default:
							return nil, errors.New("unknown role")
						}
					})

				m.On(
					"AddRealmRoleComposite",
					testifymock.Anything,
					token,
					realm,
					"role1",
					testifymock.MatchedBy(func(roles []gocloak.Role) bool {
						r := make([]string, 0, len(roles))

						for _, role := range roles {
							r = append(r, *role.Name)
						}

						return assert.Len(t, r, 2) &&
							assert.Contains(t, r, "role3") &&
							assert.Contains(t, r, "role5")
					})).
					Return(nil)

				m.On(
					"DeleteRealmRoleComposite",
					testifymock.Anything,
					token,
					realm,
					"role1",
					testifymock.MatchedBy(func(roles []gocloak.Role) bool {
						r := make([]string, 0, len(roles))

						for _, role := range roles {
							r = append(r, *role.Name)
						}

						return assert.Len(t, r, 2) &&
							assert.Contains(t, r, "role6") &&
							assert.Contains(t, r, "role7")
					})).
					Return(nil)

				m.On(
					"UpdateRealmRole",
					testifymock.Anything,
					token,
					realm,
					"role1",
					testifymock.Anything).
					Return(nil)

				return m
			},
			wantErr: require.NoError,
		},
		{
			name: "failed to delete composite roles",
			role: &dto.PrimaryRealmRole{
				Name:        "role1",
				Composites:  []string{},
				IsComposite: true,
			},
			client: func(t *testing.T) *mocks.MockGoCloak {
				m := mocks.NewMockGoCloak(t)

				m.On(
					"GetCompositeRolesByRoleID",
					testifymock.Anything,
					token,
					realm,
					"role1-id").
					Return([]*gocloak.Role{
						{
							Name: gocloak.StringP("role2"),
						},
					}, nil)

				m.On("GetRealmRole", testifymock.Anything, token, realm, testifymock.Anything).
					Return(&gocloak.Role{Name: gocloak.StringP("role1"), ID: gocloak.StringP("role1-id")}, nil)

				m.On(
					"DeleteRealmRoleComposite",
					testifymock.Anything,
					token,
					realm,
					"role1",
					testifymock.Anything).
					Return(errors.New("failed to delete composite roles"))

				return m
			},
			wantErr: func(t require.TestingT, err error, i ...interface{}) {
				require.Error(t, err)
				assert.Contains(t, err.Error(), "failed to delete composite roles")
			},
		},
		{
			name: "failed to add composite roles",
			role: &dto.PrimaryRealmRole{
				Name:        "role1",
				Composites:  []string{"role2"},
				IsComposite: true,
				Description: "Role description",
				IsDefault:   false,
			},
			client: func(t *testing.T) *mocks.MockGoCloak {
				m := mocks.NewMockGoCloak(t)

				m.On(
					"GetCompositeRolesByRoleID",
					testifymock.Anything,
					token,
					realm,
					"role1-id").
					Return([]*gocloak.Role{}, nil)

				m.On("GetRealmRole", testifymock.Anything, token, realm, testifymock.Anything).
					Return(func(ctx context.Context, token, realm, roleName string) (*gocloak.Role, error) {
						switch roleName {
						case "role1":
							return &gocloak.Role{Name: gocloak.StringP("role1"), ID: gocloak.StringP("role1-id")}, nil
						case "role2":
							return &gocloak.Role{Name: gocloak.StringP("role2"), ID: gocloak.StringP("role2-id")}, nil
						default:
							return nil, errors.New("unknown role")
						}
					})

				m.On(
					"AddRealmRoleComposite",
					testifymock.Anything,
					token,
					realm,
					"role1",
					testifymock.Anything).
					Return(errors.New("failed to add composite roles"))

				return m
			},
			wantErr: func(t require.TestingT, err error, i ...interface{}) {
				require.Error(t, err)
				assert.Contains(t, err.Error(), "failed to add composite roles")
			},
		},
		{
			name: "failed get composite associated role",
			role: &dto.PrimaryRealmRole{
				Name:        "role1",
				Composites:  []string{"role2"},
				IsComposite: true,
				Description: "Role description",
				IsDefault:   false,
			},
			client: func(t *testing.T) *mocks.MockGoCloak {
				m := mocks.NewMockGoCloak(t)

				m.On(
					"GetCompositeRolesByRoleID",
					testifymock.Anything,
					token,
					realm,
					"role1-id").
					Return([]*gocloak.Role{}, nil)

				m.On("GetRealmRole", testifymock.Anything, token, realm, testifymock.Anything).
					Return(func(ctx context.Context, token, realm, roleName string) (*gocloak.Role, error) {
						switch roleName {
						case "role1":
							return &gocloak.Role{Name: gocloak.StringP("role1"), ID: gocloak.StringP("role1-id")}, nil
						case "role2":
							return nil, errors.New("failed to get associated role")
						default:
							return nil, errors.New("unknown role")
						}
					})

				return m
			},
			wantErr: func(t require.TestingT, err error, i ...interface{}) {
				require.Error(t, err)
				assert.Contains(t, err.Error(), "failed to get associated role")
			},
		},
		{
			name: "failed get composite associated client role",
			role: &dto.PrimaryRealmRole{
				Name:                  "role1",
				CompositesClientRoles: map[string][]string{"client1": {"role2"}},
				IsComposite:           true,
				Description:           "Role description",
				IsDefault:             false,
			},
			client: func(t *testing.T) *mocks.MockGoCloak {
				m := mocks.NewMockGoCloak(t)

				m.On(
					"GetCompositeRolesByRoleID",
					testifymock.Anything,
					token,
					realm,
					"role1-id").
					Return([]*gocloak.Role{}, nil)

				m.On("GetRealmRole", testifymock.Anything, token, realm, testifymock.Anything).
					Return(&gocloak.Role{Name: gocloak.StringP("role1"), ID: gocloak.StringP("role1-id")}, nil)

				m.On("GetClients", testifymock.Anything, token, realm, testifymock.Anything).
					Return([]*gocloak.Client{{
						ID:   gocloak.StringP("client1-id"),
						Name: gocloak.StringP("client1"),
					}}, nil)

				m.On("GetClientRole", testifymock.Anything, token, realm, "client1-id", "role2").
					Return(nil, errors.New("failed to get associated client role"))

				return m
			},
			wantErr: func(t require.TestingT, err error, i ...interface{}) {
				require.Error(t, err)
				assert.Contains(t, err.Error(), "failed to get associated client role")
			},
		},
		{
			name: "failed get client",
			role: &dto.PrimaryRealmRole{
				Name:                  "role1",
				CompositesClientRoles: map[string][]string{"client1": {"role2"}},
				IsComposite:           true,
				Description:           "Role description",
				IsDefault:             false,
			},
			client: func(t *testing.T) *mocks.MockGoCloak {
				m := mocks.NewMockGoCloak(t)

				m.On(
					"GetCompositeRolesByRoleID",
					testifymock.Anything,
					token,
					realm,
					"role1-id").
					Return([]*gocloak.Role{}, nil)

				m.On("GetRealmRole", testifymock.Anything, token, realm, testifymock.Anything).
					Return(&gocloak.Role{Name: gocloak.StringP("role1"), ID: gocloak.StringP("role1-id")}, nil)

				m.On("GetClients", testifymock.Anything, token, realm, testifymock.Anything).
					Return(nil, errors.New("failed to get client"))

				return m
			},
			wantErr: func(t require.TestingT, err error, i ...interface{}) {
				require.Error(t, err)
				assert.Contains(t, err.Error(), "failed to get client")
			},
		},
		{
			name: "failed get current composite roles roles",
			role: &dto.PrimaryRealmRole{
				Name:                  "role1",
				CompositesClientRoles: map[string][]string{"client1": {"role2"}},
				IsComposite:           true,
				Description:           "Role description",
				IsDefault:             false,
			},
			client: func(t *testing.T) *mocks.MockGoCloak {
				m := mocks.NewMockGoCloak(t)

				m.On(
					"GetCompositeRolesByRoleID",
					testifymock.Anything,
					token,
					realm,
					"role1-id").
					Return(nil, errors.New("failed to get current composite roles"))

				m.On("GetRealmRole", testifymock.Anything, token, realm, testifymock.Anything).
					Return(&gocloak.Role{Name: gocloak.StringP("role1"), ID: gocloak.StringP("role1-id")}, nil)

				return m
			},
			wantErr: func(t require.TestingT, err error, i ...interface{}) {
				require.Error(t, err)
				assert.Contains(t, err.Error(), "failed to get current composite roles")
			},
		},
		{
			name: "should create new default role",
			role: &dto.PrimaryRealmRole{
				Name:      "role1",
				IsDefault: true,
			},
			client: func(t *testing.T) *mocks.MockGoCloak {
				m := mocks.NewMockGoCloak(t)

				getRealmCall := 0
				m.On(
					"GetRealmRole",
					testifymock.Anything,
					token,
					realm,
					"role1").
					Return(func(ctx context.Context, token, realm, roleName string) (*gocloak.Role, error) {
						if getRealmCall == 0 {
							getRealmCall++
							return nil, NotFoundError("role not found")
						}

						return &gocloak.Role{Name: gocloak.StringP("role1"), ID: gocloak.StringP("role1-id")}, nil
					})

				m.On(
					"CreateRealmRole",
					testifymock.Anything,
					token,
					realm,
					testifymock.Anything).
					Return("", nil)

				m.On(
					"AddRealmRoleComposite",
					testifymock.Anything,
					token,
					realm,
					testifymock.Anything,
					testifymock.Anything).
					Return(nil)

				return m
			},
			wantErr: require.NoError,
		},
		{
			name: "failed to update role",
			role: &dto.PrimaryRealmRole{
				Name: "role1",
			},
			client: func(t *testing.T) *mocks.MockGoCloak {
				m := mocks.NewMockGoCloak(t)

				m.On("GetRealmRole", testifymock.Anything, token, realm, testifymock.Anything).
					Return(&gocloak.Role{Name: gocloak.StringP("role1"), ID: gocloak.StringP("role1-id")}, nil)

				m.On(
					"UpdateRealmRole",
					testifymock.Anything,
					token,
					realm,
					"role1",
					testifymock.Anything).
					Return(errors.New("failed to update role"))

				return m
			},
			wantErr: func(t require.TestingT, err error, i ...interface{}) {
				require.Error(t, err)
				assert.Contains(t, err.Error(), "failed to update role")
			},
		},
		{
			name: "failed to create role",
			role: &dto.PrimaryRealmRole{
				Name: "role1",
			},
			client: func(t *testing.T) *mocks.MockGoCloak {
				m := mocks.NewMockGoCloak(t)

				m.On("GetRealmRole", testifymock.Anything, token, realm, testifymock.Anything).
					Return(nil, NotFoundError("role not found"))

				m.On(
					"CreateRealmRole",
					testifymock.Anything,
					token,
					realm,
					testifymock.Anything).
					Return("", errors.New("failed to create role"))

				return m
			},
			wantErr: func(t require.TestingT, err error, i ...interface{}) {
				require.Error(t, err)
				assert.Contains(t, err.Error(), "failed to create role")
			},
		},
		{
			name: "failed to get role",
			role: &dto.PrimaryRealmRole{
				Name: "role1",
			},
			client: func(t *testing.T) *mocks.MockGoCloak {
				m := mocks.NewMockGoCloak(t)

				m.On("GetRealmRole", testifymock.Anything, token, realm, testifymock.Anything).
					Return(nil, errors.New("failed to get role"))

				return m
			},
			wantErr: func(t require.TestingT, err error, i ...interface{}) {
				require.Error(t, err)
				assert.Contains(t, err.Error(), "failed to get role")
			},
		},
		{
			name: "failed to make role default",
			role: &dto.PrimaryRealmRole{
				Name:      "role1",
				IsDefault: true,
			},
			client: func(t *testing.T) *mocks.MockGoCloak {
				m := mocks.NewMockGoCloak(t)

				getRealmCall := 0
				m.On(
					"GetRealmRole",
					testifymock.Anything,
					token,
					realm,
					"role1").
					Return(func(ctx context.Context, token, realm, roleName string) (*gocloak.Role, error) {
						if getRealmCall == 0 {
							getRealmCall++
							return nil, NotFoundError("role not found")
						}

						return &gocloak.Role{Name: gocloak.StringP("role1"), ID: gocloak.StringP("role1-id")}, nil
					})

				m.On(
					"CreateRealmRole",
					testifymock.Anything,
					token,
					realm,
					testifymock.Anything).
					Return("", nil)

				m.On(
					"AddRealmRoleComposite",
					testifymock.Anything,
					token,
					realm,
					testifymock.Anything,
					testifymock.Anything).
					Return(errors.New("failed to make role default"))

				return m
			},
			wantErr: func(t require.TestingT, err error, i ...interface{}) {
				require.Error(t, err)
				assert.Contains(t, err.Error(), "failed to make role default")
			},
		},
	}

	for _, tt := range tests {
		tt := tt

		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			a := GoCloakAdapter{
				client: tt.client(t),
				token:  &gocloak.JWT{AccessToken: token},
				log:    logr.Discard(),
			}

			err := a.SyncRealmRole(ctrl.LoggerInto(context.Background(), logr.Discard()), realm, tt.role)
			tt.wantErr(t, err)
		})
	}
}

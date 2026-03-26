package chain

import (
	"context"
	"fmt"
	"testing"

	"github.com/go-logr/logr"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"k8s.io/utils/ptr"
	ctrl "sigs.k8s.io/controller-runtime"

	keycloakApi "github.com/epam/edp-keycloak-operator/api/v1"
	keycloakv2 "github.com/epam/edp-keycloak-operator/pkg/client/keycloakv2"
	keycloakv2mocks "github.com/epam/edp-keycloak-operator/pkg/client/keycloakv2/mocks"
)

func TestPutAdminFineGrainedPermissions_Serve(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		idp     *keycloakApi.KeycloakRealmIdentityProvider
		kClient func(t *testing.T) *keycloakv2.KeycloakClient
		wantErr require.ErrorAssertionFunc
	}{
		{
			name: "with admin permission enabled",
			idp: &keycloakApi.KeycloakRealmIdentityProvider{
				Spec: keycloakApi.KeycloakRealmIdentityProviderSpec{
					Alias:                              "test-idp",
					AdminFineGrainedPermissionsEnabled: true,
					Permission: &keycloakApi.AdminFineGrainedPermission{
						ScopePermissions: []keycloakApi.ScopePermissions{
							{
								Name:     "map-role",
								Policies: []string{"scope permission"},
							},
						},
					},
				},
			},
			kClient: func(t *testing.T) *keycloakv2.KeycloakClient {
				serverMock := keycloakv2mocks.NewMockServerInfoClient(t)
				serverMock.On("FeatureFlagEnabled", mock.Anything, keycloakv2.FeatureFlagAdminFineGrainedAuthz).
					Return(true, nil).Once()

				idpMock := keycloakv2mocks.NewMockIdentityProvidersClient(t)
				idpMock.On("UpdateIDPManagementPermissions", mock.Anything, "realm", "test-idp",
					keycloakv2.ManagementPermissionReference{Enabled: ptr.To(true)}).
					Return((*keycloakv2.ManagementPermissionReference)(nil), (*keycloakv2.Response)(nil), nil)
				idpMock.On("GetIdentityProvider", mock.Anything, "realm", "test-idp").
					Return(&keycloakv2.IdentityProviderRepresentation{InternalId: ptr.To("12345")}, (*keycloakv2.Response)(nil), nil).Once()
				idpMock.On("GetIDPManagementPermissions", mock.Anything, "realm", "test-idp").
					Return(&keycloakv2.ManagementPermissionReference{
						Enabled:          ptr.To(true),
						ScopePermissions: &map[string]string{"map-role": "321"},
					}, (*keycloakv2.Response)(nil), nil)

				clientsMock := keycloakv2mocks.NewMockClientsClient(t)
				clientsMock.On("GetClientUUID", mock.Anything, "realm", "realm-management").
					Return("567", nil).Once()

				authMock := keycloakv2mocks.NewMockAuthorizationClient(t)
				authMock.On("GetPermissions", mock.Anything, "realm", "567").
					Return([]keycloakv2.AbstractPolicyRepresentation{
						{
							Id:   ptr.To("scope-permission2-id"),
							Name: ptr.To("map-role.permission.idp.12345"),
							Type: ptr.To("scope"),
						},
					}, (*keycloakv2.Response)(nil), nil).Once()
				authMock.On("UpdatePermission", mock.Anything, "realm", "567", "scope", "scope-permission2-id", mock.Anything).
					Return((*keycloakv2.Response)(nil), nil).Once()

				return &keycloakv2.KeycloakClient{
					Server:            serverMock,
					IdentityProviders: idpMock,
					Clients:           clientsMock,
					Authorization:     authMock,
				}
			},
			wantErr: require.NoError,
		},
		{
			name: "with feature flag disabled",
			idp: &keycloakApi.KeycloakRealmIdentityProvider{
				Spec: keycloakApi.KeycloakRealmIdentityProviderSpec{
					Alias:                              "test-idp",
					AdminFineGrainedPermissionsEnabled: true,
				},
			},
			kClient: func(t *testing.T) *keycloakv2.KeycloakClient {
				serverMock := keycloakv2mocks.NewMockServerInfoClient(t)
				serverMock.On("FeatureFlagEnabled", mock.Anything, keycloakv2.FeatureFlagAdminFineGrainedAuthz).
					Return(false, nil).Once()

				return &keycloakv2.KeycloakClient{
					Server: serverMock,
				}
			},
			wantErr: require.NoError,
		},
		{
			name: "with feature flag check error",
			idp: &keycloakApi.KeycloakRealmIdentityProvider{
				Spec: keycloakApi.KeycloakRealmIdentityProviderSpec{
					Alias:                              "test-idp",
					AdminFineGrainedPermissionsEnabled: true,
				},
			},
			kClient: func(t *testing.T) *keycloakv2.KeycloakClient {
				serverMock := keycloakv2mocks.NewMockServerInfoClient(t)
				serverMock.On("FeatureFlagEnabled", mock.Anything, keycloakv2.FeatureFlagAdminFineGrainedAuthz).
					Return(false, fmt.Errorf("feature flag check failed")).Once()

				return &keycloakv2.KeycloakClient{
					Server: serverMock,
				}
			},
			wantErr: require.Error,
		},
		{
			name: "UpdateIDPManagementPermissions fails",
			idp: &keycloakApi.KeycloakRealmIdentityProvider{
				Spec: keycloakApi.KeycloakRealmIdentityProviderSpec{
					Alias:                              "test-idp",
					AdminFineGrainedPermissionsEnabled: true,
				},
			},
			kClient: func(t *testing.T) *keycloakv2.KeycloakClient {
				serverMock := keycloakv2mocks.NewMockServerInfoClient(t)
				serverMock.On("FeatureFlagEnabled", mock.Anything, keycloakv2.FeatureFlagAdminFineGrainedAuthz).
					Return(true, nil).Once()

				idpMock := keycloakv2mocks.NewMockIdentityProvidersClient(t)
				idpMock.On("UpdateIDPManagementPermissions", mock.Anything, "realm", "test-idp",
					keycloakv2.ManagementPermissionReference{Enabled: ptr.To(true)}).
					Return((*keycloakv2.ManagementPermissionReference)(nil), (*keycloakv2.Response)(nil), fmt.Errorf("api error"))

				return &keycloakv2.KeycloakClient{
					Server:            serverMock,
					IdentityProviders: idpMock,
				}
			},
			wantErr: require.Error,
		},
		{
			name: "putKeycloakIDPAdminPermissionPolicies fails",
			idp: &keycloakApi.KeycloakRealmIdentityProvider{
				Spec: keycloakApi.KeycloakRealmIdentityProviderSpec{
					Alias:                              "test-idp",
					AdminFineGrainedPermissionsEnabled: true,
					Permission: &keycloakApi.AdminFineGrainedPermission{
						ScopePermissions: []keycloakApi.ScopePermissions{
							{
								Name:     "map-role",
								Policies: []string{"scope permission"},
							},
						},
					},
				},
			},
			kClient: func(t *testing.T) *keycloakv2.KeycloakClient {
				serverMock := keycloakv2mocks.NewMockServerInfoClient(t)
				serverMock.On("FeatureFlagEnabled", mock.Anything, keycloakv2.FeatureFlagAdminFineGrainedAuthz).
					Return(true, nil).Once()

				idpMock := keycloakv2mocks.NewMockIdentityProvidersClient(t)
				idpMock.On("UpdateIDPManagementPermissions", mock.Anything, "realm", "test-idp",
					keycloakv2.ManagementPermissionReference{Enabled: ptr.To(true)}).
					Return((*keycloakv2.ManagementPermissionReference)(nil), (*keycloakv2.Response)(nil), nil)
				idpMock.On("GetIdentityProvider", mock.Anything, "realm", "test-idp").
					Return((*keycloakv2.IdentityProviderRepresentation)(nil), (*keycloakv2.Response)(nil), fmt.Errorf("get idp error")).Once()

				return &keycloakv2.KeycloakClient{
					Server:            serverMock,
					IdentityProviders: idpMock,
				}
			},
			wantErr: require.Error,
		},
		{
			name: "feature flag enabled but permissions not enabled",
			idp: &keycloakApi.KeycloakRealmIdentityProvider{
				Spec: keycloakApi.KeycloakRealmIdentityProviderSpec{
					Alias:                              "test-idp",
					AdminFineGrainedPermissionsEnabled: false,
				},
			},
			kClient: func(t *testing.T) *keycloakv2.KeycloakClient {
				serverMock := keycloakv2mocks.NewMockServerInfoClient(t)
				serverMock.On("FeatureFlagEnabled", mock.Anything, keycloakv2.FeatureFlagAdminFineGrainedAuthz).
					Return(true, nil).Once()

				idpMock := keycloakv2mocks.NewMockIdentityProvidersClient(t)
				idpMock.On("UpdateIDPManagementPermissions", mock.Anything, "realm", "test-idp",
					keycloakv2.ManagementPermissionReference{Enabled: ptr.To(false)}).
					Return((*keycloakv2.ManagementPermissionReference)(nil), (*keycloakv2.Response)(nil), nil)

				return &keycloakv2.KeycloakClient{
					Server:            serverMock,
					IdentityProviders: idpMock,
				}
			},
			wantErr: require.NoError,
		},
		{
			name: "nil scope permissions returns error",
			idp: &keycloakApi.KeycloakRealmIdentityProvider{
				Spec: keycloakApi.KeycloakRealmIdentityProviderSpec{
					Alias:                              "test-idp",
					AdminFineGrainedPermissionsEnabled: true,
					Permission: &keycloakApi.AdminFineGrainedPermission{
						ScopePermissions: []keycloakApi.ScopePermissions{
							{
								Name:     "map-role",
								Policies: []string{"scope permission"},
							},
						},
					},
				},
			},
			kClient: func(t *testing.T) *keycloakv2.KeycloakClient {
				serverMock := keycloakv2mocks.NewMockServerInfoClient(t)
				serverMock.On("FeatureFlagEnabled", mock.Anything, keycloakv2.FeatureFlagAdminFineGrainedAuthz).
					Return(true, nil).Once()

				idpMock := keycloakv2mocks.NewMockIdentityProvidersClient(t)
				idpMock.On("UpdateIDPManagementPermissions", mock.Anything, "realm", "test-idp",
					keycloakv2.ManagementPermissionReference{Enabled: ptr.To(true)}).
					Return((*keycloakv2.ManagementPermissionReference)(nil), (*keycloakv2.Response)(nil), nil)
				idpMock.On("GetIdentityProvider", mock.Anything, "realm", "test-idp").
					Return(&keycloakv2.IdentityProviderRepresentation{InternalId: ptr.To("12345")}, (*keycloakv2.Response)(nil), nil).Once()
				idpMock.On("GetIDPManagementPermissions", mock.Anything, "realm", "test-idp").
					Return(&keycloakv2.ManagementPermissionReference{
						Enabled:          ptr.To(true),
						ScopePermissions: nil,
					}, (*keycloakv2.Response)(nil), nil)

				clientsMock := keycloakv2mocks.NewMockClientsClient(t)
				clientsMock.On("GetClientUUID", mock.Anything, "realm", "realm-management").
					Return("567", nil).Once()

				authMock := keycloakv2mocks.NewMockAuthorizationClient(t)
				authMock.On("GetPermissions", mock.Anything, "realm", "567").
					Return([]keycloakv2.AbstractPolicyRepresentation{}, (*keycloakv2.Response)(nil), nil).Once()

				return &keycloakv2.KeycloakClient{
					Server:            serverMock,
					IdentityProviders: idpMock,
					Clients:           clientsMock,
					Authorization:     authMock,
				}
			},
			wantErr: require.Error,
		},
		{
			name: "permission with nil Id is skipped",
			idp: &keycloakApi.KeycloakRealmIdentityProvider{
				Spec: keycloakApi.KeycloakRealmIdentityProviderSpec{
					Alias:                              "test-idp",
					AdminFineGrainedPermissionsEnabled: true,
					Permission: &keycloakApi.AdminFineGrainedPermission{
						ScopePermissions: []keycloakApi.ScopePermissions{
							{
								Name:     "map-role",
								Policies: []string{"scope permission"},
							},
						},
					},
				},
			},
			kClient: func(t *testing.T) *keycloakv2.KeycloakClient {
				serverMock := keycloakv2mocks.NewMockServerInfoClient(t)
				serverMock.On("FeatureFlagEnabled", mock.Anything, keycloakv2.FeatureFlagAdminFineGrainedAuthz).
					Return(true, nil).Once()

				idpMock := keycloakv2mocks.NewMockIdentityProvidersClient(t)
				idpMock.On("UpdateIDPManagementPermissions", mock.Anything, "realm", "test-idp",
					keycloakv2.ManagementPermissionReference{Enabled: ptr.To(true)}).
					Return((*keycloakv2.ManagementPermissionReference)(nil), (*keycloakv2.Response)(nil), nil)
				idpMock.On("GetIdentityProvider", mock.Anything, "realm", "test-idp").
					Return(&keycloakv2.IdentityProviderRepresentation{InternalId: ptr.To("12345")}, (*keycloakv2.Response)(nil), nil).Once()
				idpMock.On("GetIDPManagementPermissions", mock.Anything, "realm", "test-idp").
					Return(&keycloakv2.ManagementPermissionReference{
						Enabled:          ptr.To(true),
						ScopePermissions: &map[string]string{"map-role": "321"},
					}, (*keycloakv2.Response)(nil), nil)

				clientsMock := keycloakv2mocks.NewMockClientsClient(t)
				clientsMock.On("GetClientUUID", mock.Anything, "realm", "realm-management").
					Return("567", nil).Once()

				authMock := keycloakv2mocks.NewMockAuthorizationClient(t)
				authMock.On("GetPermissions", mock.Anything, "realm", "567").
					Return([]keycloakv2.AbstractPolicyRepresentation{
						{
							Id:   nil,
							Name: ptr.To("map-role.permission.idp.12345"),
							Type: ptr.To("scope"),
						},
					}, (*keycloakv2.Response)(nil), nil).Once()

				return &keycloakv2.KeycloakClient{
					Server:            serverMock,
					IdentityProviders: idpMock,
					Clients:           clientsMock,
					Authorization:     authMock,
				}
			},
			wantErr: require.NoError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			h := NewPutAdminFineGrainedPermissions(tt.kClient(t))
			err := h.Serve(
				ctrl.LoggerInto(context.Background(), logr.Discard()),
				tt.idp,
				"realm",
			)
			tt.wantErr(t, err)
		})
	}
}

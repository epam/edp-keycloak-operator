package chain

import (
	"context"
	"fmt"
	"testing"

	"github.com/go-logr/logr"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/utils/ptr"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	keycloakApi "github.com/epam/edp-keycloak-operator/api/v1"
	"github.com/epam/edp-keycloak-operator/pkg/client/keycloakapi"
	keycloakapiMocks "github.com/epam/edp-keycloak-operator/pkg/client/keycloakapi/mocks"
)

func TestPutAdminFineGrainedPermissions_Serve(t *testing.T) {
	tests := []struct {
		name              string
		client            func(t *testing.T) client.Client
		keycloakClient    client.ObjectKey
		keycloakApiClient func(t *testing.T) *keycloakapi.KeycloakClient
		wantErr           require.ErrorAssertionFunc
	}{
		{
			name: "with admin permission enabled",
			client: func(t *testing.T) client.Client {
				s := runtime.NewScheme()
				require.NoError(t, keycloakApi.AddToScheme(s))
				require.NoError(t, corev1.AddToScheme(s))

				return fake.NewClientBuilder().
					WithScheme(s).
					WithStatusSubresource(&keycloakApi.KeycloakClient{}).
					WithObjects(
						&keycloakApi.KeycloakClient{
							ObjectMeta: metav1.ObjectMeta{
								Name:      "test-client",
								Namespace: "default",
							},
							Spec: keycloakApi.KeycloakClientSpec{
								ClientId:                           "test-client-id",
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
						}).Build()
			},
			keycloakClient: client.ObjectKey{
				Name:      "test-client",
				Namespace: "default",
			},
			keycloakApiClient: func(t *testing.T) *keycloakapi.KeycloakClient {
				serverMock := keycloakapiMocks.NewMockServerInfoClient(t)
				clientsMock := keycloakapiMocks.NewMockClientsClient(t)
				authzMock := keycloakapiMocks.NewMockAuthorizationClient(t)

				serverMock.On("FeatureFlagEnabled", mock.Anything, "ADMIN_FINE_GRAINED_AUTHZ").
					Return(true, nil).
					Once()

				// UpdateClientManagementPermissions
				clientsMock.On("UpdateClientManagementPermissions", mock.Anything, "realm", "123", mock.MatchedBy(func(p keycloakapi.ManagementPermissionReference) bool {
					return p.Enabled != nil && *p.Enabled == true
				})).
					Return((*keycloakapi.ManagementPermissionReference)(nil), (*keycloakapi.Response)(nil), nil)

				// GetClientUUID for realm-management
				clientsMock.On("GetClientUUID", mock.Anything, "realm", "realm-management").
					Return("567", nil).
					Once()

				// GetPermissions for realm-management client
				authzMock.On("GetPermissions", mock.Anything, "realm", "567").
					Return([]keycloakapi.AbstractPolicyRepresentation{
						{
							Id:   ptr.To("scope-permission-id"),
							Name: ptr.To("scope permission"),
						},
						{
							Id:   ptr.To("scope-permission2-id"),
							Name: ptr.To("map-role.permission.client.123"),
							Type: ptr.To("scope"),
						},
					}, (*keycloakapi.Response)(nil), nil).Once()

				// GetClientManagementPermissions
				scopePerms := map[string]string{
					"map-role": "321",
				}
				clientsMock.On("GetClientManagementPermissions", mock.Anything, "realm", "123").
					Return(&keycloakapi.ManagementPermissionReference{
						Enabled:          ptr.To(true),
						ScopePermissions: &scopePerms,
					}, (*keycloakapi.Response)(nil), nil)

				// UpdatePermission
				authzMock.On("UpdatePermission", mock.Anything, "realm", "567", "scope", "scope-permission2-id", mock.Anything).
					Return((*keycloakapi.Response)(nil), nil)

				return &keycloakapi.KeycloakClient{
					Server:        serverMock,
					Clients:       clientsMock,
					Authorization: authzMock,
				}
			},
			wantErr: require.NoError,
		},
		{
			name: "with feature flag disabled",
			client: func(t *testing.T) client.Client {
				s := runtime.NewScheme()
				require.NoError(t, keycloakApi.AddToScheme(s))
				require.NoError(t, corev1.AddToScheme(s))

				return fake.NewClientBuilder().
					WithScheme(s).
					WithStatusSubresource(&keycloakApi.KeycloakClient{}).
					WithObjects(
						&keycloakApi.KeycloakClient{
							ObjectMeta: metav1.ObjectMeta{
								Name:      "test-client",
								Namespace: "default",
							},
							Spec: keycloakApi.KeycloakClientSpec{
								ClientId:                           "test-client-id",
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
						}).Build()
			},
			keycloakClient: client.ObjectKey{
				Name:      "test-client",
				Namespace: "default",
			},
			keycloakApiClient: func(t *testing.T) *keycloakapi.KeycloakClient {
				serverMock := keycloakapiMocks.NewMockServerInfoClient(t)

				serverMock.On("FeatureFlagEnabled", mock.Anything, "ADMIN_FINE_GRAINED_AUTHZ").
					Return(false, nil).
					Once()

				return &keycloakapi.KeycloakClient{
					Server: serverMock,
				}
			},
			wantErr: require.NoError,
		},
		{
			name: "with feature flag check error",
			client: func(t *testing.T) client.Client {
				s := runtime.NewScheme()
				require.NoError(t, keycloakApi.AddToScheme(s))
				require.NoError(t, corev1.AddToScheme(s))

				return fake.NewClientBuilder().
					WithScheme(s).
					WithStatusSubresource(&keycloakApi.KeycloakClient{}).
					WithObjects(
						&keycloakApi.KeycloakClient{
							ObjectMeta: metav1.ObjectMeta{
								Name:      "test-client",
								Namespace: "default",
							},
							Spec: keycloakApi.KeycloakClientSpec{
								ClientId:                           "test-client-id",
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
						}).Build()
			},
			keycloakClient: client.ObjectKey{
				Name:      "test-client",
				Namespace: "default",
			},
			keycloakApiClient: func(t *testing.T) *keycloakapi.KeycloakClient {
				serverMock := keycloakapiMocks.NewMockServerInfoClient(t)

				serverMock.On("FeatureFlagEnabled", mock.Anything, "ADMIN_FINE_GRAINED_AUTHZ").
					Return(false, fmt.Errorf("feature flag check failed")).
					Once()

				return &keycloakapi.KeycloakClient{
					Server: serverMock,
				}
			},
			wantErr: require.Error,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cl := &keycloakApi.KeycloakClient{}
			require.NoError(t, tt.client(t).Get(context.Background(), tt.keycloakClient, cl))

			el := NewPutAdminFineGrainedPermissions(tt.keycloakApiClient(t), tt.client(t))
			err := el.Serve(
				ctrl.LoggerInto(context.Background(), logr.Discard()),
				cl,
				"realm",
				&ClientContext{ClientUUID: "123"},
			)
			tt.wantErr(t, err)
		})
	}
}

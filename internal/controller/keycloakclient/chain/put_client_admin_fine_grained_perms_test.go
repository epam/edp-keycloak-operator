package chain

import (
	"context"
	"fmt"
	"testing"

	"github.com/go-logr/logr"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/meta"
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
			name: "with admin permission enabled and already in sync",
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
								Name:      testClientName,
								Namespace: testClientNamespace,
							},
							Spec: keycloakApi.KeycloakClientSpec{
								ClientId:                           testClientID,
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
				Name:      testClientName,
				Namespace: testClientNamespace,
			},
			keycloakApiClient: func(t *testing.T) *keycloakapi.KeycloakClient {
				serverMock := keycloakapiMocks.NewMockServerInfoClient(t)
				clientsMock := keycloakapiMocks.NewMockClientsClient(t)
				authzMock := keycloakapiMocks.NewMockAuthorizationClient(t)

				serverMock.On("FeatureFlagEnabled", mock.Anything, "ADMIN_FINE_GRAINED_AUTHZ").
					Return(true, nil).
					Once()

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

				// GetClientManagementPermissions: enabled matches spec, no update call expected
				scopePerms := map[string]string{
					"map-role": "321",
				}
				clientsMock.On("GetClientManagementPermissions", mock.Anything, "realm", "123").
					Return(&keycloakapi.ManagementPermissionReference{
						Enabled:          ptr.To(true),
						ScopePermissions: &scopePerms,
					}, (*keycloakapi.Response)(nil), nil).
					Once()

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
								Name:      testClientName,
								Namespace: testClientNamespace,
							},
							Spec: keycloakApi.KeycloakClientSpec{
								ClientId:                           testClientID,
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
				Name:      testClientName,
				Namespace: testClientNamespace,
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
								Name:      testClientName,
								Namespace: testClientNamespace,
							},
							Spec: keycloakApi.KeycloakClientSpec{
								ClientId:                           testClientID,
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
				Name:      testClientName,
				Namespace: testClientNamespace,
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

func TestPutAdminFineGrainedPermissions_Serve_ManagementPermissionsCompare(t *testing.T) {
	scopePerms := map[string]string{"map-role": "321"}
	realmPerms := []keycloakapi.AbstractPolicyRepresentation{
		{Id: ptr.To("perm-id"), Name: ptr.To("map-role.permission.client.123"), Type: ptr.To("scope")},
	}

	permission := &keycloakApi.AdminFineGrainedPermission{
		ScopePermissions: []keycloakApi.ScopePermissions{{Name: "map-role", Policies: []string{"p"}}},
	}

	tests := []struct {
		name           string
		enabled        bool
		permission     *keycloakApi.AdminFineGrainedPermission
		get            *keycloakapi.ManagementPermissionReference
		getErr         error
		wantPut        *bool
		put            *keycloakapi.ManagementPermissionReference
		wantUpdatePerm bool
		wantErr        string
	}{
		{
			name:    "disabled in spec, never enabled",
			enabled: false,
			get:     &keycloakapi.ManagementPermissionReference{},
		},
		{
			name:           "enabled in sync, policies reuse the fetched reference",
			enabled:        true,
			permission:     permission,
			get:            &keycloakapi.ManagementPermissionReference{Enabled: ptr.To(true), ScopePermissions: &scopePerms},
			wantUpdatePerm: true,
		},
		{
			name:           "enable, policies reuse the PUT response",
			enabled:        true,
			permission:     permission,
			get:            &keycloakapi.ManagementPermissionReference{Enabled: ptr.To(false)},
			wantPut:        ptr.To(true),
			put:            &keycloakapi.ManagementPermissionReference{Enabled: ptr.To(true), ScopePermissions: &scopePerms},
			wantUpdatePerm: true,
		},
		{
			name:    "disable",
			enabled: false,
			get:     &keycloakapi.ManagementPermissionReference{Enabled: ptr.To(true), ScopePermissions: &scopePerms},
			wantPut: ptr.To(false),
			put:     &keycloakapi.ManagementPermissionReference{Enabled: ptr.To(false)},
		},
		{
			name:    "nil GET body, disabled in spec",
			enabled: false,
			get:     nil,
			wantPut: ptr.To(false),
			put:     &keycloakapi.ManagementPermissionReference{Enabled: ptr.To(false)},
		},
		{
			name:    "get error",
			enabled: true,
			getErr:  fmt.Errorf("get failed"),
			wantErr: "failed to get client management permissions",
		},
		{
			name:       "reused reference without scope permissions",
			enabled:    true,
			permission: permission,
			get:        &keycloakapi.ManagementPermissionReference{Enabled: ptr.To(true)},
			wantErr:    "client management permissions or scope permissions are nil",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := runtime.NewScheme()
			require.NoError(t, keycloakApi.AddToScheme(s))
			require.NoError(t, corev1.AddToScheme(s))

			cl := &keycloakApi.KeycloakClient{
				ObjectMeta: metav1.ObjectMeta{Name: testClientName, Namespace: testClientNamespace},
				Spec: keycloakApi.KeycloakClientSpec{
					ClientId:                           testClientID,
					AdminFineGrainedPermissionsEnabled: tt.enabled,
					Permission:                         tt.permission,
				},
			}

			k8sClient := fake.NewClientBuilder().
				WithScheme(s).
				WithStatusSubresource(&keycloakApi.KeycloakClient{}).
				WithObjects(cl.DeepCopy()).
				Build()
			require.NoError(t, k8sClient.Get(context.Background(), client.ObjectKeyFromObject(cl), cl))

			serverMock := keycloakapiMocks.NewMockServerInfoClient(t)
			serverMock.On("FeatureFlagEnabled", mock.Anything, "ADMIN_FINE_GRAINED_AUTHZ").Return(true, nil).Once()

			clientsMock := keycloakapiMocks.NewMockClientsClient(t)
			clientsMock.On("GetClientManagementPermissions", mock.Anything, "realm", "123").
				Return(tt.get, (*keycloakapi.Response)(nil), tt.getErr).
				Once()

			if tt.wantPut != nil {
				clientsMock.On("UpdateClientManagementPermissions", mock.Anything, "realm", "123",
					mock.MatchedBy(func(p keycloakapi.ManagementPermissionReference) bool {
						return p.Enabled != nil && *p.Enabled == *tt.wantPut
					})).
					Return(tt.put, (*keycloakapi.Response)(nil), nil).
					Once()
			}

			authzMock := keycloakapiMocks.NewMockAuthorizationClient(t)

			if tt.permission != nil && tt.getErr == nil {
				clientsMock.On("GetClientUUID", mock.Anything, "realm", "realm-management").Return("567", nil).Once()
				authzMock.On("GetPermissions", mock.Anything, "realm", "567").
					Return(realmPerms, (*keycloakapi.Response)(nil), nil).
					Once()
			}

			if tt.wantUpdatePerm {
				authzMock.On("UpdatePermission", mock.Anything, "realm", "567", "scope", "perm-id", mock.Anything).
					Return((*keycloakapi.Response)(nil), nil).
					Once()
			}

			el := NewPutAdminFineGrainedPermissions(
				&keycloakapi.KeycloakClient{Server: serverMock, Clients: clientsMock, Authorization: authzMock}, k8sClient)

			err := el.Serve(ctrl.LoggerInto(context.Background(), logr.Discard()), cl, "realm", &ClientContext{ClientUUID: "123"})

			cond := meta.FindStatusCondition(cl.Status.Conditions, ConditionAdminFineGrainedPermissionsV1Synced)
			require.NotNil(t, cond)

			if tt.wantErr != "" {
				require.ErrorContains(t, err, tt.wantErr)
				require.Equal(t, metav1.ConditionFalse, cond.Status)

				return
			}

			require.NoError(t, err)
			require.Equal(t, metav1.ConditionTrue, cond.Status)
		})
	}
}

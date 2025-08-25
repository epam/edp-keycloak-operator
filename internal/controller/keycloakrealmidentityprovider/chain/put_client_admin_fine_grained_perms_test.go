package chain

import (
	"context"
	"fmt"
	"testing"

	"github.com/Nerzal/gocloak/v12"
	"github.com/go-logr/logr"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	keycloakApi "github.com/epam/edp-keycloak-operator/api/v1"
	"github.com/epam/edp-keycloak-operator/pkg/client/keycloak/adapter"
	"github.com/epam/edp-keycloak-operator/pkg/client/keycloak/mocks"
)

func TestPutAdminFineGrainedPermissions_Serve(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name              string
		client            func(t *testing.T) client.Client
		keycloakIDP       client.ObjectKey
		keycloakApiClient func(t *testing.T) *mocks.MockClient
		wantErr           require.ErrorAssertionFunc
	}{
		{
			name: "with admin permission enabled",
			client: func(t *testing.T) client.Client {
				s := runtime.NewScheme()
				require.NoError(t, keycloakApi.AddToScheme(s))
				require.NoError(t, corev1.AddToScheme(s))

				return fake.NewClientBuilder().WithScheme(s).WithObjects(
					&keycloakApi.KeycloakRealmIdentityProvider{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "test-idp",
							Namespace: "default",
						},
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
					}).Build()
			},
			keycloakIDP: client.ObjectKey{
				Name:      "test-idp",
				Namespace: "default",
			},
			keycloakApiClient: func(t *testing.T) *mocks.MockClient {
				m := mocks.NewMockClient(t)

				scopePermissions := map[string]string{
					"map-role": "321",
				}

				m.On("FeatureFlagEnabled", ctrl.LoggerInto(context.Background(), logr.Discard()), adapter.FeatureFlagAdminFineGrainedAuthz).
					Return(true, nil).
					Once()

				m.On("GetClientID", "realm-management", "realm").
					Return("567", nil).
					Once()

				m.On("UpdateIDPManagementPermissions", "realm", "test-idp", adapter.ManagementPermissionRepresentation{
					Enabled: gocloak.BoolP(true),
				}).
					Return(nil)

				m.On("GetIDPManagementPermissions", "realm", "test-idp").
					Return(&adapter.ManagementPermissionRepresentation{
						Enabled:          gocloak.BoolP(true),
						ScopePermissions: &scopePermissions,
					}, nil)

				m.On("GetPermissions", ctrl.LoggerInto(context.Background(), logr.Discard()), "realm", "567").
					Return(map[string]gocloak.PermissionRepresentation{
						"token-exchange": {
							ID:   gocloak.StringP("scope-permission-id"),
							Name: gocloak.StringP("scope permission"),
						},
						"map-role": {
							ID:   gocloak.StringP("scope-permission2-id"),
							Name: gocloak.StringP("scope-permission2"),
						},
					}, nil).Once()

				return m
			},
			wantErr: require.NoError,
		},
		{
			name: "with feature flag disabled",
			client: func(t *testing.T) client.Client {
				s := runtime.NewScheme()
				require.NoError(t, keycloakApi.AddToScheme(s))
				require.NoError(t, corev1.AddToScheme(s))

				return fake.NewClientBuilder().WithScheme(s).WithObjects(
					&keycloakApi.KeycloakRealmIdentityProvider{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "test-idp",
							Namespace: "default",
						},
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
					}).Build()
			},
			keycloakIDP: client.ObjectKey{
				Name:      "test-idp",
				Namespace: "default",
			},
			keycloakApiClient: func(t *testing.T) *mocks.MockClient {
				m := mocks.NewMockClient(t)

				m.On("FeatureFlagEnabled", ctrl.LoggerInto(context.Background(), logr.Discard()), adapter.FeatureFlagAdminFineGrainedAuthz).
					Return(false, nil).
					Once()

				return m
			},
			wantErr: require.NoError,
		},
		{
			name: "with feature flag check error",
			client: func(t *testing.T) client.Client {
				s := runtime.NewScheme()
				require.NoError(t, keycloakApi.AddToScheme(s))
				require.NoError(t, corev1.AddToScheme(s))

				return fake.NewClientBuilder().WithScheme(s).WithObjects(
					&keycloakApi.KeycloakRealmIdentityProvider{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "test-idp",
							Namespace: "default",
						},
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
					}).Build()
			},
			keycloakIDP: client.ObjectKey{
				Name:      "test-idp",
				Namespace: "default",
			},
			keycloakApiClient: func(t *testing.T) *mocks.MockClient {
				m := mocks.NewMockClient(t)

				m.On("FeatureFlagEnabled", ctrl.LoggerInto(context.Background(), logr.Discard()), adapter.FeatureFlagAdminFineGrainedAuthz).
					Return(false, fmt.Errorf("feature flag check failed")).
					Once()

				return m
			},
			wantErr: require.Error,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			idp := &keycloakApi.KeycloakRealmIdentityProvider{}
			require.NoError(t, tt.client(t).Get(context.Background(), tt.keycloakIDP, idp))

			el := NewPutAdminFineGrainedPermissions(tt.keycloakApiClient(t))
			err := el.Serve(
				ctrl.LoggerInto(context.Background(), logr.Discard()),
				idp,
				"realm",
			)
			tt.wantErr(t, err)
		})
	}
}

package chain

import (
	"context"
	"testing"

	"github.com/go-logr/logr"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"k8s.io/utils/ptr"
	ctrl "sigs.k8s.io/controller-runtime"

	"github.com/epam/edp-keycloak-operator/api/common"
	keycloakApi "github.com/epam/edp-keycloak-operator/api/v1alpha1"
	"github.com/epam/edp-keycloak-operator/pkg/client/keycloak"
	"github.com/epam/edp-keycloak-operator/pkg/client/keycloak/mocks"
	keycloakv2 "github.com/epam/edp-keycloak-operator/pkg/client/keycloakv2"
	keycloakv2mocks "github.com/epam/edp-keycloak-operator/pkg/client/keycloakv2/mocks"
)

func TestUserProfile_ServeRequest(t *testing.T) {
	tests := []struct {
		name      string
		realm     *keycloakApi.ClusterKeycloakRealm
		kClient   func(t *testing.T) keycloak.Client
		kClientV2 func(t *testing.T) *keycloakv2.KeycloakClient
		wantErr   require.ErrorAssertionFunc
	}{
		{
			name: "should update user profile successfully",
			realm: &keycloakApi.ClusterKeycloakRealm{
				Spec: keycloakApi.ClusterKeycloakRealmSpec{
					RealmName: "realm",
					UserProfileConfig: &common.UserProfileConfig{
						UnmanagedAttributePolicy: "ENABLED",
						Attributes: []common.UserProfileAttribute{
							{
								DisplayName: "Attribute 2",
								Group:       "test-group",
								Name:        "attr2",
							},
						},
						Groups: []common.UserProfileGroup{
							{
								Name: "test-group2",
							},
						},
					},
				},
			},
			kClient: func(t *testing.T) keycloak.Client {
				return mocks.NewMockClient(t)
			},
			kClientV2: func(t *testing.T) *keycloakv2.KeycloakClient {
				mockUsers := keycloakv2mocks.NewMockUsersClient(t)

				mockUsers.On("GetUsersProfile", mock.Anything, "realm").
					Return(&keycloakv2.UserProfileConfig{
						Attributes: &[]keycloakv2.UserProfileAttribute{
							{
								DisplayName: ptr.To("Attribute 1"),
								Group:       ptr.To("test-group"),
								Name:        ptr.To("attr1"),
							},
						},
						Groups: &[]keycloakv2.UserProfileGroup{
							{
								Name:               ptr.To("test-group"),
								DisplayDescription: ptr.To("Group description"),
								DisplayHeader:      ptr.To("Group header"),
							},
						},
					}, nil, nil)

				mockUsers.On("UpdateUsersProfile", mock.Anything, "realm", mock.Anything).
					Return(&keycloakv2.UserProfileConfig{}, nil, nil)

				return &keycloakv2.KeycloakClient{Users: mockUsers}
			},
			wantErr: require.NoError,
		},
		{
			name:  "empty user profile config",
			realm: &keycloakApi.ClusterKeycloakRealm{},
			kClient: func(t *testing.T) keycloak.Client {
				return mocks.NewMockClient(t)
			},
			kClientV2: func(t *testing.T) *keycloakv2.KeycloakClient {
				return nil
			},
			wantErr: require.NoError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := NewUserProfile()
			tt.wantErr(t, h.ServeRequest(
				ctrl.LoggerInto(context.Background(), logr.Discard()),
				tt.realm,
				tt.kClient(t),
				tt.kClientV2(t),
			))
		})
	}
}

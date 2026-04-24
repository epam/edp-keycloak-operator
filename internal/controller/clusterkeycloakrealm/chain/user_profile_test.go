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
	"github.com/epam/edp-keycloak-operator/pkg/client/keycloakapi"
	keycloakapimocks "github.com/epam/edp-keycloak-operator/pkg/client/keycloakapi/mocks"
)

func TestUserProfile_ServeRequest(t *testing.T) {
	tests := []struct {
		name    string
		realm   *keycloakApi.ClusterKeycloakRealm
		kClient func(t *testing.T) *keycloakapi.KeycloakClient
		wantErr require.ErrorAssertionFunc
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
			kClient: func(t *testing.T) *keycloakapi.KeycloakClient {
				mockUsers := keycloakapimocks.NewMockUsersClient(t)

				mockUsers.On("GetUsersProfile", mock.Anything, "realm").
					Return(&keycloakapi.UserProfileConfig{
						Attributes: &[]keycloakapi.UserProfileAttribute{
							{
								DisplayName: ptr.To("Attribute 1"),
								Group:       ptr.To("test-group"),
								Name:        ptr.To("attr1"),
							},
						},
						Groups: &[]keycloakapi.UserProfileGroup{
							{
								Name:               ptr.To("test-group"),
								DisplayDescription: ptr.To("Group description"),
								DisplayHeader:      ptr.To("Group header"),
							},
						},
					}, nil, nil)

				mockUsers.On("UpdateUsersProfile", mock.Anything, "realm", mock.Anything).
					Return(&keycloakapi.UserProfileConfig{}, nil, nil)

				return &keycloakapi.KeycloakClient{Users: mockUsers}
			},
			wantErr: require.NoError,
		},
		{
			name:  "empty user profile config",
			realm: &keycloakApi.ClusterKeycloakRealm{},
			kClient: func(t *testing.T) *keycloakapi.KeycloakClient {
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
			))
		})
	}
}

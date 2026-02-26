package chain

import (
	"context"
	"testing"

	"github.com/go-logr/logr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"k8s.io/utils/ptr"
	ctrl "sigs.k8s.io/controller-runtime"

	"github.com/epam/edp-keycloak-operator/api/common"
	keycloakApi "github.com/epam/edp-keycloak-operator/api/v1"
	keycloakv2 "github.com/epam/edp-keycloak-operator/pkg/client/keycloakv2"
	keycloakv2mocks "github.com/epam/edp-keycloak-operator/pkg/client/keycloakv2/mocks"
)

func TestUserProfileConfigSpecToModel(t *testing.T) {
	tests := []struct {
		name string
		spec *common.UserProfileConfig
		want keycloakv2.UserProfileConfig
	}{
		{
			name: "should convert spec to model",
			spec: &common.UserProfileConfig{
				UnmanagedAttributePolicy: "ENABLED",
				Attributes: []common.UserProfileAttribute{
					{
						DisplayName: "Attribute 1",
						Group:       "test-group",
						Name:        "attr1",
						Multivalued: true,
						Permissions: &common.UserProfileAttributePermissions{
							Edit: []string{"edit"},
							View: []string{"view"},
						},
						Required: &common.UserProfileAttributeRequired{
							Roles:  []string{"role"},
							Scopes: []string{"scope"},
						},
						Selector: &common.UserProfileAttributeSelector{
							Scopes: []string{"scope"},
						},
						Annotations: map[string]string{
							"inputType": "text",
						},
						Validations: map[string]map[string]common.UserProfileAttributeValidation{
							"email": {
								"max-local-length": {
									IntVal: 64,
								},
							},
							"local-date": {},
							"multivalued": {
								"min": {
									StringVal: "1",
								},
								"max": {
									StringVal: "10",
								},
							},
							"options": {
								"options": {
									SliceVal: []string{"option1", "option2"},
								},
							},
						},
					},
				},
				Groups: []common.UserProfileGroup{
					{
						Annotations:        map[string]string{"group": "test"},
						DisplayDescription: "Group description",
						DisplayHeader:      "Group header",
						Name:               "Group",
					},
				},
			},
			want: keycloakv2.UserProfileConfig{
				UnmanagedAttributePolicy: ptr.To(keycloakv2.UnmanagedAttributePolicy("ENABLED")),
				Attributes: &[]keycloakv2.UserProfileAttribute{
					{
						DisplayName: ptr.To("Attribute 1"),
						Group:       ptr.To("test-group"),
						Name:        ptr.To("attr1"),
						Multivalued: ptr.To(true),
						Permissions: &keycloakv2.UserProfileAttributePermissions{
							Edit: &[]string{"edit"},
							View: &[]string{"view"},
						},
						Required: &keycloakv2.UserProfileAttributeRequired{
							Roles:  &[]string{"role"},
							Scopes: &[]string{"scope"},
						},
						Selector: &keycloakv2.UserProfileAttributeSelector{
							Scopes: &[]string{"scope"},
						},
						Annotations: &map[string]any{
							"inputType": "text",
						},
						Validations: &map[string]map[string]any{
							"email": {
								"max-local-length": 64,
							},
							"local-date": {},
							"multivalued": {
								"min": "1",
								"max": "10",
							},
							"options": {
								"options": []string{"option1", "option2"},
							},
						},
					},
				},
				Groups: &[]keycloakv2.UserProfileGroup{
					{
						Annotations:        &map[string]any{"group": "test"},
						DisplayDescription: ptr.To("Group description"),
						DisplayHeader:      ptr.To("Group header"),
						Name:               ptr.To("Group"),
					},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := userProfileConfigSpecToModel(tt.spec)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestUserProfile_ServeRequest(t *testing.T) {
	tests := []struct {
		name      string
		realm     *keycloakApi.KeycloakRealm
		kClientV2 func(t *testing.T) *keycloakv2.KeycloakClient
		wantErr   require.ErrorAssertionFunc
	}{
		{
			name: "should update user profile successfully",
			realm: &keycloakApi.KeycloakRealm{
				Spec: keycloakApi.KeycloakRealmSpec{
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
			realm: &keycloakApi.KeycloakRealm{},
			kClientV2: func(t *testing.T) *keycloakv2.KeycloakClient {
				return nil
			},
			wantErr: require.NoError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			a := &UserProfile{}
			tt.wantErr(t, a.ServeRequest(
				ctrl.LoggerInto(context.Background(), logr.Discard()),
				tt.realm,
				tt.kClientV2(t),
			))
		})
	}
}

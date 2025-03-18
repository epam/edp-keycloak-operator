package chain

import (
	"context"
	"testing"

	"github.com/go-logr/logr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	keycloak_go_client "github.com/zmotso/keycloak-go-client"
	"k8s.io/utils/ptr"
	ctrl "sigs.k8s.io/controller-runtime"

	"github.com/epam/edp-keycloak-operator/api/common"
	keycloakApi "github.com/epam/edp-keycloak-operator/api/v1"
	"github.com/epam/edp-keycloak-operator/pkg/client/keycloak"
	"github.com/epam/edp-keycloak-operator/pkg/client/keycloak/mocks"
)

func TestUserProfileConfigSpecToModel(t *testing.T) {
	tests := []struct {
		name string
		spec *common.UserProfileConfig
		want keycloak_go_client.UserProfileConfig
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
			want: keycloak_go_client.UserProfileConfig{
				UnmanagedAttributePolicy: ptr.To(keycloak_go_client.UnmanagedAttributePolicy("ENABLED")),
				Attributes: &[]keycloak_go_client.UserProfileAttribute{
					{
						DisplayName: ptr.To("Attribute 1"),
						Group:       ptr.To("test-group"),
						Name:        ptr.To("attr1"),
						Multivalued: ptr.To(true),
						Permissions: &keycloak_go_client.UserProfileAttributePermissions{
							Edit: &[]string{"edit"},
							View: &[]string{"view"},
						},
						Required: &keycloak_go_client.UserProfileAttributeRequired{
							Roles:  &[]string{"role"},
							Scopes: &[]string{"scope"},
						},
						Selector: &keycloak_go_client.UserProfileAttributeSelector{
							Scopes: &[]string{"scope"},
						},
						Annotations: &map[string]interface{}{
							"inputType": "text",
						},
						Validations: &map[string]map[string]interface{}{
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
				Groups: &[]keycloak_go_client.UserProfileGroup{
					{
						Annotations:        &map[string]interface{}{"group": "test"},
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
		name    string
		realm   *keycloakApi.KeycloakRealm
		kClient func(t *testing.T) keycloak.Client
		wantErr require.ErrorAssertionFunc
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
			kClient: func(t *testing.T) keycloak.Client {
				m := mocks.NewMockClient(t)

				m.On("GetUsersProfile", mock.Anything, "realm").
					Return(&keycloak_go_client.UserProfileConfig{
						Attributes: &[]keycloak_go_client.UserProfileAttribute{
							{
								DisplayName: ptr.To("Attribute 1"),
								Group:       ptr.To("test-group"),
								Name:        ptr.To("attr1"),
							},
						},
						Groups: &[]keycloak_go_client.UserProfileGroup{
							{
								Name:               ptr.To("test-group"),
								DisplayDescription: ptr.To("Group description"),
								DisplayHeader:      ptr.To("Group header"),
							},
						},
					}, nil)

				m.On("UpdateUsersProfile", mock.Anything, "realm", mock.Anything).
					Return(&keycloak_go_client.UserProfileConfig{}, nil)

				return m
			},
			wantErr: require.NoError,
		},
		{
			name:  "empty user profile config",
			realm: &keycloakApi.KeycloakRealm{},
			kClient: func(t *testing.T) keycloak.Client {
				return mocks.NewMockClient(t)
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
				tt.kClient(t),
			))
		})
	}
}

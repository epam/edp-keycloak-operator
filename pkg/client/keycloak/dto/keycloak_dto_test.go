package dto

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"k8s.io/utils/ptr"

	keycloakApi "github.com/epam/edp-keycloak-operator/api/v1"
	keycloakApiV1Alpha1 "github.com/epam/edp-keycloak-operator/api/v1alpha1"
)

func TestConvertSpecToRole(t *testing.T) {
	tests := []struct {
		name     string
		input    *keycloakApi.KeycloakRealmRole
		expected *PrimaryRealmRole
	}{
		{
			name: "basic role conversion",
			input: &keycloakApi.KeycloakRealmRole{
				Spec: keycloakApi.KeycloakRealmRoleSpec{
					Name:        "test-role",
					Description: "Test role description",
					Composite:   false,
					IsDefault:   true,
				},
				Status: keycloakApi.KeycloakRealmRoleStatus{
					ID: "role-123",
				},
			},
			expected: &PrimaryRealmRole{
				ID:                    ptr.To("role-123"),
				Name:                  "test-role",
				Description:           "Test role description",
				IsComposite:           false,
				IsDefault:             true,
				Composites:            []string{},
				CompositesClientRoles: map[string][]string{},
			},
		},
		{
			name: "composite role with realm composites",
			input: &keycloakApi.KeycloakRealmRole{
				Spec: keycloakApi.KeycloakRealmRoleSpec{
					Name:        "composite-role",
					Description: "Composite role",
					Composite:   true,
					Composites: []keycloakApi.Composite{
						{Name: "role1"},
						{Name: "role2"},
					},
					Attributes: map[string][]string{
						"attr1": {"value1", "value2"},
						"attr2": {"value3"},
					},
				},
				Status: keycloakApi.KeycloakRealmRoleStatus{
					ID: "composite-123",
				},
			},
			expected: &PrimaryRealmRole{
				ID:          ptr.To("composite-123"),
				Name:        "composite-role",
				Description: "Composite role",
				IsComposite: true,
				Composites:  []string{"role1", "role2"},
				Attributes: map[string][]string{
					"attr1": {"value1", "value2"},
					"attr2": {"value3"},
				},
				CompositesClientRoles: map[string][]string{},
			},
		},
		{
			name: "role with client composites",
			input: &keycloakApi.KeycloakRealmRole{
				Spec: keycloakApi.KeycloakRealmRoleSpec{
					Name:      "role-with-client-composites",
					Composite: true,
					CompositesClientRoles: map[string][]keycloakApi.Composite{
						"client1": {
							{Name: "client-role1"},
							{Name: "client-role2"},
						},
						"client2": {
							{Name: "client-role3"},
						},
					},
				},
			},
			expected: &PrimaryRealmRole{
				Name:        "role-with-client-composites",
				IsComposite: true,
				Composites:  []string{},
				CompositesClientRoles: map[string][]string{
					"client1": {"client-role1", "client-role2"},
					"client2": {"client-role3"},
				},
			},
		},
		{
			name: "role without status ID",
			input: &keycloakApi.KeycloakRealmRole{
				Spec: keycloakApi.KeycloakRealmRoleSpec{
					Name: "role-no-id",
				},
				Status: keycloakApi.KeycloakRealmRoleStatus{
					ID: "", // empty ID
				},
			},
			expected: &PrimaryRealmRole{
				Name:                  "role-no-id",
				ID:                    nil, // should be nil when status ID is empty
				Composites:            []string{},
				CompositesClientRoles: map[string][]string{},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ConvertSpecToRole(tt.input)

			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestConvertSpecToRealm(t *testing.T) {
	tests := []struct {
		name     string
		input    *keycloakApi.KeycloakRealmSpec
		expected *Realm
	}{
		{
			name: "basic realm conversion",
			input: &keycloakApi.KeycloakRealmSpec{
				RealmName: "test-realm",
				ID:        ptr.To("realm-123"),
			},
			expected: &Realm{
				Name:  "test-realm",
				ID:    ptr.To("realm-123"),
				Users: []User{},
			},
		},
		{
			name: "realm with users",
			input: &keycloakApi.KeycloakRealmSpec{
				RealmName: "realm-with-users",
				Users: []keycloakApi.User{
					{
						Username:   "user1",
						RealmRoles: []string{"role1", "role2"},
					},
					{
						Username:   "user2",
						RealmRoles: []string{"role3"},
					},
				},
			},
			expected: &Realm{
				Name: "realm-with-users",
				Users: []User{
					{
						Username:   "user1",
						RealmRoles: []string{"role1", "role2"},
					},
					{
						Username:   "user2",
						RealmRoles: []string{"role3"},
					},
				},
			},
		},
		{
			name: "realm with empty users slice",
			input: &keycloakApi.KeycloakRealmSpec{
				RealmName: "empty-users-realm",
				Users:     []keycloakApi.User{},
			},
			expected: &Realm{
				Name:  "empty-users-realm",
				Users: []User{},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ConvertSpecToRealm(tt.input)

			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestConvertSpecToClient(t *testing.T) {
	tests := []struct {
		name              string
		spec              *keycloakApi.KeycloakClientSpec
		clientSecret      string
		realmName         string
		authFlowOverrides map[string]string
		expected          *Client
	}{
		{
			name: "basic client conversion",
			spec: &keycloakApi.KeycloakClientSpec{
				ClientId: "test-client",
				Public:   true,
				Enabled:  true,
			},
			clientSecret: "secret123",
			realmName:    "test-realm",
			expected: &Client{
				ClientId:                           "test-client",
				ClientSecret:                       "secret123",
				RealmName:                          "test-realm",
				PublicClient:                       true,
				Protocol:                           defaultClientProtocol,
				Enabled:                            true,
				Roles:                              []ClientRole{},
				AuthenticationFlowBindingOverrides: nil,
			},
		},
		{
			name: "client with custom protocol",
			spec: &keycloakApi.KeycloakClientSpec{
				ClientId: "saml-client",
				Protocol: ptr.To("saml"),
				Enabled:  true,
			},
			clientSecret: "",
			realmName:    "test-realm",
			expected: &Client{
				ClientId:                           "saml-client",
				ClientSecret:                       "",
				RealmName:                          "test-realm",
				Protocol:                           "saml",
				Enabled:                            true,
				Roles:                              []ClientRole{},
				AuthenticationFlowBindingOverrides: nil,
			},
		},
		{
			name: "client with service account enabled",
			spec: &keycloakApi.KeycloakClientSpec{
				ClientId: "service-client",
				ServiceAccount: &keycloakApi.ServiceAccount{
					Enabled: true,
				},
				Enabled: true,
			},
			clientSecret: "service-secret",
			realmName:    "test-realm",
			expected: &Client{
				ClientId:                           "service-client",
				ClientSecret:                       "service-secret",
				RealmName:                          "test-realm",
				ServiceAccountEnabled:              true,
				Protocol:                           defaultClientProtocol,
				Enabled:                            true,
				Roles:                              []ClientRole{},
				AuthenticationFlowBindingOverrides: nil,
			},
		},
		{
			name: "client with service account disabled",
			spec: &keycloakApi.KeycloakClientSpec{
				ClientId: "no-service-client",
				ServiceAccount: &keycloakApi.ServiceAccount{
					Enabled: false,
				},
				Enabled: true,
			},
			clientSecret: "secret",
			realmName:    "test-realm",
			expected: &Client{
				ClientId:                           "no-service-client",
				ClientSecret:                       "secret",
				RealmName:                          "test-realm",
				ServiceAccountEnabled:              false,
				Protocol:                           defaultClientProtocol,
				Enabled:                            true,
				Roles:                              []ClientRole{},
				AuthenticationFlowBindingOverrides: nil,
			},
		},
		{
			name: "client with roles",
			spec: &keycloakApi.KeycloakClientSpec{
				ClientId: "client-with-roles",
				ClientRolesV2: []keycloakApi.ClientRole{
					{
						Name:                  "role1",
						Description:           "First role",
						AssociatedClientRoles: []string{"associated-role1"},
					},
					{
						Name:        "role2",
						Description: "Second role",
					},
					{
						Name: "", // empty name should be skipped
					},
				},
				Enabled: true,
			},
			clientSecret: "secret",
			realmName:    "test-realm",
			expected: &Client{
				ClientId:     "client-with-roles",
				ClientSecret: "secret",
				RealmName:    "test-realm",
				Protocol:     defaultClientProtocol,
				Enabled:      true,
				Roles: []ClientRole{
					{
						Name:                  "role1",
						Description:           "First role",
						AssociatedClientRoles: []string{"associated-role1"},
					},
					{
						Name:        "role2",
						Description: "Second role",
					},
				},
				AuthenticationFlowBindingOverrides: nil,
			},
		},
		{
			name: "client with all properties",
			spec: &keycloakApi.KeycloakClientSpec{
				ClientId:                     "full-client",
				Public:                       false,
				DirectAccess:                 true,
				WebUrl:                       "https://example.com",
				AdminUrl:                     "https://admin.example.com",
				HomeUrl:                      "https://home.example.com",
				Protocol:                     ptr.To("openid-connect"),
				Attributes:                   map[string]string{"key1": "value1"},
				AdvancedProtocolMappers:      true,
				FrontChannelLogout:           true,
				RedirectUris:                 []string{"https://example.com/callback"},
				WebOrigins:                   []string{"https://example.com"},
				ImplicitFlowEnabled:          true,
				AuthorizationServicesEnabled: true,
				BearerOnly:                   false,
				ClientAuthenticatorType:      "client-secret",
				ConsentRequired:              false,
				Description:                  "Full featured client",
				Enabled:                      true,
				FullScopeAllowed:             true,
				Name:                         "Full Client",
				StandardFlowEnabled:          true,
				SurrogateAuthRequired:        false,
			},
			clientSecret: "full-secret",
			realmName:    "full-realm",
			authFlowOverrides: map[string]string{
				"browser":     "custom-browser-flow",
				"directGrant": "custom-direct-grant-flow",
			},
			expected: &Client{
				ClientId:                     "full-client",
				ClientSecret:                 "full-secret",
				RealmName:                    "full-realm",
				PublicClient:                 false,
				DirectAccess:                 true,
				WebUrl:                       "https://example.com",
				AdminUrl:                     "https://admin.example.com",
				HomeUrl:                      "https://home.example.com",
				Protocol:                     "openid-connect",
				Attributes:                   map[string]string{"key1": "value1"},
				AdvancedProtocolMappers:      true,
				ServiceAccountEnabled:        false,
				FrontChannelLogout:           true,
				RedirectUris:                 []string{"https://example.com/callback"},
				WebOrigins:                   []string{"https://example.com"},
				ImplicitFlowEnabled:          true,
				AuthorizationServicesEnabled: true,
				BearerOnly:                   false,
				ClientAuthenticatorType:      "client-secret",
				ConsentRequired:              false,
				Description:                  "Full featured client",
				Enabled:                      true,
				FullScopeAllowed:             true,
				Name:                         "Full Client",
				StandardFlowEnabled:          true,
				SurrogateAuthRequired:        false,
				Roles:                        []ClientRole{},
				AuthenticationFlowBindingOverrides: map[string]string{
					"browser":     "custom-browser-flow",
					"directGrant": "custom-direct-grant-flow",
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ConvertSpecToClient(tt.spec, tt.clientSecret, tt.realmName, tt.authFlowOverrides)

			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestConvertSpecToOrganization(t *testing.T) {
	tests := []struct {
		name     string
		input    *keycloakApiV1Alpha1.KeycloakOrganization
		expected *Organization
	}{
		{
			name: "basic organization conversion",
			input: &keycloakApiV1Alpha1.KeycloakOrganization{
				Spec: keycloakApiV1Alpha1.KeycloakOrganizationSpec{
					Name:  "test-org",
					Alias: "test-alias",
				},
			},
			expected: &Organization{
				Name:    "test-org",
				Alias:   "test-alias",
				Domains: nil,
			},
		},
		{
			name: "organization with all fields",
			input: &keycloakApiV1Alpha1.KeycloakOrganization{
				Spec: keycloakApiV1Alpha1.KeycloakOrganizationSpec{
					Name:        "full-org",
					Alias:       "full-alias",
					Description: "Full organization description",
					RedirectURL: "https://redirect.example.com",
					Domains:     []string{"example.com", "test.com"},
					Attributes: map[string][]string{
						"attr1": {"value1", "value2"},
						"attr2": {"value3"},
					},
				},
				Status: keycloakApiV1Alpha1.KeycloakOrganizationStatus{
					OrganizationID: "org-123",
				},
			},
			expected: &Organization{
				ID:          "org-123",
				Name:        "full-org",
				Alias:       "full-alias",
				Description: "Full organization description",
				RedirectURL: "https://redirect.example.com",
				Attributes: map[string][]string{
					"attr1": {"value1", "value2"},
					"attr2": {"value3"},
				},
				Domains: []OrganizationDomain{
					{Name: "example.com"},
					{Name: "test.com"},
				},
			},
		},
		{
			name: "organization with empty domains",
			input: &keycloakApiV1Alpha1.KeycloakOrganization{
				Spec: keycloakApiV1Alpha1.KeycloakOrganizationSpec{
					Name:    "empty-domains-org",
					Alias:   "empty-alias",
					Domains: []string{},
				},
			},
			expected: &Organization{
				Name:    "empty-domains-org",
				Alias:   "empty-alias",
				Domains: nil,
			},
		},
		{
			name: "organization without status ID",
			input: &keycloakApiV1Alpha1.KeycloakOrganization{
				Spec: keycloakApiV1Alpha1.KeycloakOrganizationSpec{
					Name:  "no-id-org",
					Alias: "no-id-alias",
				},
				Status: keycloakApiV1Alpha1.KeycloakOrganizationStatus{
					OrganizationID: "", // empty ID
				},
			},
			expected: &Organization{
				Name:    "no-id-org",
				Alias:   "no-id-alias",
				Domains: nil,
				// ID should not be set when OrganizationID is empty
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ConvertSpecToOrganization(tt.input)

			assert.Equal(t, tt.expected, result)
		})
	}
}

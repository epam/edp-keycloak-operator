package dto

import (
	keycloakApi "github.com/epam/edp-keycloak-operator/api/v1"
	keycloakApiV1Alpha1 "github.com/epam/edp-keycloak-operator/api/v1alpha1"
)

const defaultClientProtocol = "openid-connect"

type Keycloak struct {
	Url  string
	User string
	Pwd  string `json:"-"`
}

type Realm struct {
	Name  string
	Users []User
	ID    *string
}

type User struct {
	Username   string   `json:"username"`
	RealmRoles []string `json:"realmRoles"`
}

type ServerInfo struct {
	SystemInfo SystemInfo      `json:"systemInfo"`
	Features   []ServerFeature `json:"features"`
}

type ServerFeature struct {
	Name    string `json:"name"`
	Enabled bool   `json:"enabled"`
}

type SystemInfo struct {
	Version string `json:"version"`
}

func ConvertSpecToRole(roleInstance *keycloakApi.KeycloakRealmRole) *PrimaryRealmRole {
	rr := PrimaryRealmRole{
		Name:                  roleInstance.Spec.Name,
		Description:           roleInstance.Spec.Description,
		IsComposite:           roleInstance.Spec.Composite,
		Attributes:            roleInstance.Spec.Attributes,
		Composites:            make([]string, 0, len(roleInstance.Spec.Composites)),
		CompositesClientRoles: make(map[string][]string, len(roleInstance.Spec.CompositesClientRoles)),
		IsDefault:             roleInstance.Spec.IsDefault,
	}

	for _, comp := range roleInstance.Spec.Composites {
		rr.Composites = append(rr.Composites, comp.Name)
	}

	for k, v := range roleInstance.Spec.CompositesClientRoles {
		rr.CompositesClientRoles[k] = make([]string, 0, len(v))
		for _, comp := range v {
			rr.CompositesClientRoles[k] = append(rr.CompositesClientRoles[k], comp.Name)
		}
	}

	if roleInstance.Status.ID != "" {
		rr.ID = &roleInstance.Status.ID
	}

	return &rr
}

func ConvertSpecToRealm(spec *keycloakApi.KeycloakRealmSpec) *Realm {
	var users []User
	for _, item := range spec.Users {
		users = append(users, User(item))
	}

	return &Realm{
		Name:  spec.RealmName,
		Users: users,
		ID:    spec.ID,
	}
}

type Client struct {
	ID                                 string
	ClientId                           string
	ClientSecret                       string `json:"-"`
	RealmName                          string
	Roles                              []ClientRole
	PublicClient                       bool
	DirectAccess                       bool
	WebUrl                             string
	AdminUrl                           string
	HomeUrl                            string
	Protocol                           string
	Attributes                         map[string]string
	AdvancedProtocolMappers            bool
	ServiceAccountEnabled              bool
	FrontChannelLogout                 bool
	RedirectUris                       []string
	BaseUrl                            string
	WebOrigins                         []string
	AuthorizationServicesEnabled       bool
	BearerOnly                         bool
	ClientAuthenticatorType            string
	ConsentRequired                    bool
	Description                        string
	Enabled                            bool
	FullScopeAllowed                   bool
	ImplicitFlowEnabled                bool
	Name                               string
	Origin                             string
	RegistrationAccessToken            string
	StandardFlowEnabled                bool
	SurrogateAuthRequired              bool
	AuthenticationFlowBindingOverrides map[string]string
}

type ClientRole struct {
	Name                  string   `json:"name"`
	Description           string   `json:"description"`
	AssociatedClientRoles []string `json:"associatedClientRoles"`
}

type PrimaryRealmRole struct {
	ID                    *string
	Name                  string
	Composites            []string
	CompositesClientRoles map[string][]string
	IsComposite           bool
	Description           string
	Attributes            map[string][]string
	IsDefault             bool
}

type IncludedRealmRole struct {
	Name      string
	Composite string
}

func ConvertSpecToClient(spec *keycloakApi.KeycloakClientSpec, clientSecret, realmName string, authFlowOverrides map[string]string) *Client {
	// Convert ClientRolesV2 to DTO ClientRole format
	roles := make([]ClientRole, 0, len(spec.ClientRolesV2))

	for _, role := range spec.ClientRolesV2 {
		if role.Name != "" {
			dtoRole := ClientRole{
				Name:                  role.Name,
				Description:           role.Description,
				AssociatedClientRoles: role.AssociatedClientRoles,
			}
			roles = append(roles, dtoRole)
		}
	}

	return &Client{
		RealmName:                          realmName,
		ClientId:                           spec.ClientId,
		ClientSecret:                       clientSecret,
		Roles:                              roles,
		PublicClient:                       spec.Public,
		DirectAccess:                       spec.DirectAccess,
		WebUrl:                             spec.WebUrl,
		AdminUrl:                           spec.AdminUrl,
		HomeUrl:                            spec.HomeUrl,
		Protocol:                           getValueOrDefault(spec.Protocol),
		Attributes:                         spec.Attributes,
		AdvancedProtocolMappers:            spec.AdvancedProtocolMappers,
		ServiceAccountEnabled:              spec.ServiceAccount != nil && spec.ServiceAccount.Enabled,
		FrontChannelLogout:                 spec.FrontChannelLogout,
		RedirectUris:                       spec.RedirectUris,
		WebOrigins:                         spec.WebOrigins,
		ImplicitFlowEnabled:                spec.ImplicitFlowEnabled,
		AuthorizationServicesEnabled:       spec.AuthorizationServicesEnabled,
		BearerOnly:                         spec.BearerOnly,
		ClientAuthenticatorType:            spec.ClientAuthenticatorType,
		ConsentRequired:                    spec.ConsentRequired,
		Description:                        spec.Description,
		Enabled:                            spec.Enabled,
		FullScopeAllowed:                   spec.FullScopeAllowed,
		Name:                               spec.Name,
		StandardFlowEnabled:                spec.StandardFlowEnabled,
		SurrogateAuthRequired:              spec.SurrogateAuthRequired,
		AuthenticationFlowBindingOverrides: authFlowOverrides,
	}
}

func getValueOrDefault(protocol *string) string {
	if protocol == nil {
		return defaultClientProtocol
	}

	return *protocol
}

type IdentityProviderMapper struct {
	IdentityProviderMapper string            `json:"identityProviderMapper"`
	IdentityProviderAlias  string            `json:"identityProviderAlias,omitempty"`
	Name                   string            `json:"name"`
	Config                 map[string]string `json:"config"`
	ID                     string            `json:"id"`
}

// Organization represents a Keycloak Organization.
type Organization struct {
	ID          string               `json:"id,omitempty"`
	Name        string               `json:"name"`
	Alias       string               `json:"alias"`
	Description string               `json:"description,omitempty"`
	RedirectURL string               `json:"redirectUrl,omitempty"`
	Attributes  map[string][]string  `json:"attributes,omitempty"`
	Domains     []OrganizationDomain `json:"domains,omitempty"`
}

// OrganizationDomain represents a domain within an Organization.
type OrganizationDomain struct {
	Name string `json:"name"`
}

// OrganizationIdentityProvider represents the link between an Organization and Identity Provider.
type OrganizationIdentityProvider struct {
	Alias string `json:"alias"`
}

// ConvertSpecToOrganization converts a KeycloakOrganization spec to an Organization.
func ConvertSpecToOrganization(org *keycloakApiV1Alpha1.KeycloakOrganization) *Organization {
	orgAdapter := &Organization{
		Name:        org.Spec.Name,
		Alias:       org.Spec.Alias,
		Description: org.Spec.Description,
		RedirectURL: org.Spec.RedirectURL,
		Attributes:  org.Spec.Attributes,
	}

	// Convert domains to OrganizationDomain format
	for _, domain := range org.Spec.Domains {
		orgAdapter.Domains = append(orgAdapter.Domains, OrganizationDomain{
			Name: domain,
		})
	}

	// Set ID from status if available
	if org.Status.OrganizationID != "" {
		orgAdapter.ID = org.Status.OrganizationID
	}

	return orgAdapter
}

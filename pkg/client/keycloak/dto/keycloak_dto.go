package dto

import (
	keycloakApi "github.com/epam/edp-keycloak-operator/api/v1"
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
	ID                           string
	ClientId                     string
	ClientSecret                 string `json:"-"`
	RealmName                    string
	Roles                        []string
	PublicClient                 bool
	DirectAccess                 bool
	WebUrl                       string
	Protocol                     string
	Attributes                   map[string]string
	AdvancedProtocolMappers      bool
	ServiceAccountEnabled        bool
	FrontChannelLogout           bool
	RedirectUris                 []string
	BaseUrl                      string
	WebOrigins                   []string
	AuthorizationServicesEnabled bool
	BearerOnly                   bool
	ClientAuthenticatorType      string
	ConsentRequired              bool
	Description                  string
	Enabled                      bool
	FullScopeAllowed             bool
	ImplicitFlowEnabled          bool
	Name                         string
	Origin                       string
	RegistrationAccessToken      string
	StandardFlowEnabled          bool
	SurrogateAuthRequired        bool
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

func ConvertSpecToClient(spec *keycloakApi.KeycloakClientSpec, clientSecret, realmName string) *Client {
	return &Client{
		RealmName:                    realmName,
		ClientId:                     spec.ClientId,
		ClientSecret:                 clientSecret,
		Roles:                        spec.ClientRoles,
		PublicClient:                 spec.Public,
		DirectAccess:                 spec.DirectAccess,
		WebUrl:                       spec.WebUrl,
		Protocol:                     getValueOrDefault(spec.Protocol),
		Attributes:                   spec.Attributes,
		AdvancedProtocolMappers:      spec.AdvancedProtocolMappers,
		ServiceAccountEnabled:        spec.ServiceAccount != nil && spec.ServiceAccount.Enabled,
		FrontChannelLogout:           spec.FrontChannelLogout,
		RedirectUris:                 spec.RedirectUris,
		WebOrigins:                   spec.WebOrigins,
		ImplicitFlowEnabled:          spec.ImplicitFlowEnabled,
		AuthorizationServicesEnabled: spec.AuthorizationServicesEnabled,
		BearerOnly:                   spec.BearerOnly,
		ClientAuthenticatorType:      spec.ClientAuthenticatorType,
		ConsentRequired:              spec.ConsentRequired,
		Description:                  spec.Description,
		Enabled:                      spec.Enabled,
		FullScopeAllowed:             spec.FullScopeAllowed,
		Name:                         spec.Name,
		StandardFlowEnabled:          spec.StandardFlowEnabled,
		SurrogateAuthRequired:        spec.SurrogateAuthRequired,
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

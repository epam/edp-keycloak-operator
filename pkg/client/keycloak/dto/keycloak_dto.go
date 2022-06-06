package dto

import (
	keycloakApi "github.com/epam/edp-keycloak-operator/pkg/apis/v1/v1"
)

const defaultClientProtocol = "openid-connect"

type Keycloak struct {
	Url  string
	User string
	Pwd  string `json:"-"`
}

type Realm struct {
	Name                     string
	Users                    []User
	SsoRealmName             string
	SsoRealmEnabled          bool
	SsoAutoRedirectEnabled   bool
	ID                       *string
	DisableCentralIDPMappers bool
}

type User struct {
	Username   string   `json:"username"`
	RealmRoles []string `json:"realmRoles"`
}

func ConvertSpecToRole(roleInstance *keycloakApi.KeycloakRealmRole) *PrimaryRealmRole {
	rr := PrimaryRealmRole{
		Name:        roleInstance.Spec.Name,
		Description: roleInstance.Spec.Description,
		IsComposite: roleInstance.Spec.Composite,
		Attributes:  roleInstance.Spec.Attributes,
		Composites:  make([]string, 0, len(roleInstance.Spec.Composites)),
		IsDefault:   roleInstance.Spec.IsDefault,
	}

	for _, comp := range roleInstance.Spec.Composites {
		rr.Composites = append(rr.Composites, comp.Name)
	}

	if roleInstance.Status.ID != "" {
		rr.ID = &roleInstance.Status.ID
	}

	return &rr
}

func ConvertSpecToRealm(spec keycloakApi.KeycloakRealmSpec) *Realm {
	var users []User
	for _, item := range spec.Users {
		users = append(users, User(item))
	}

	return &Realm{
		Name:                     spec.RealmName,
		Users:                    users,
		SsoRealmName:             spec.SsoRealmName,
		SsoRealmEnabled:          spec.SSOEnabled(),
		SsoAutoRedirectEnabled:   spec.SSOAutoRedirectEnabled(),
		ID:                       spec.ID,
		DisableCentralIDPMappers: spec.DisableCentralIDPMappers,
	}
}

type Client struct {
	ClientId                string
	ClientSecret            string `json:"-"`
	RealmName               string
	Roles                   []string
	RealmRole               IncludedRealmRole // what this for ? does not used anywhere
	Public                  bool
	DirectAccess            bool
	WebUrl                  string
	Protocol                string
	Attributes              map[string]string
	AdvancedProtocolMappers bool
	ServiceAccountEnabled   bool
	FrontChannelLogout      bool
}

type PrimaryRealmRole struct {
	ID          *string
	Name        string
	Composites  []string
	IsComposite bool
	Description string
	Attributes  map[string][]string
	IsDefault   bool
}

type IncludedRealmRole struct {
	Name      string
	Composite string
}

func ConvertSpecToClient(spec *keycloakApi.KeycloakClientSpec, clientSecret string) *Client {
	return &Client{
		RealmName:               spec.TargetRealm,
		ClientId:                spec.ClientId,
		ClientSecret:            clientSecret,
		Roles:                   spec.ClientRoles,
		Public:                  spec.Public,
		DirectAccess:            spec.DirectAccess,
		WebUrl:                  spec.WebUrl,
		Protocol:                getValueOrDefault(spec.Protocol),
		Attributes:              spec.Attributes,
		AdvancedProtocolMappers: spec.AdvancedProtocolMappers,
		ServiceAccountEnabled:   spec.ServiceAccount != nil && spec.ServiceAccount.Enabled,
		FrontChannelLogout:      spec.FrontChannelLogout,
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

func ConvertSSOMappersToIdentityProviderMappers(idpAlias string,
	ssoMappers []keycloakApi.SSORealmMapper) []IdentityProviderMapper {
	idpMappers := make([]IdentityProviderMapper, 0, len(ssoMappers))
	for _, sm := range ssoMappers {
		idpMappers = append(idpMappers, IdentityProviderMapper{
			IdentityProviderAlias:  idpAlias,
			IdentityProviderMapper: sm.IdentityProviderMapper,
			Config:                 sm.Config,
			Name:                   sm.Name,
		})
	}

	return idpMappers
}

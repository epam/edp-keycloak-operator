package dto

import (
	"github.com/epmd-edp/keycloak-operator/pkg/apis/v1/v1alpha1"
)

const defaultClientProtocol = "openid-connect"

type Keycloak struct {
	Url  string
	User string
	Pwd  string `json:"-"`
}

func ConvertSpecToKeycloak(spec v1alpha1.KeycloakSpec, user string, pwd string) Keycloak {
	return Keycloak{
		Url:  spec.Url,
		User: user,
		Pwd:  pwd,
	}
}

type Realm struct {
	Name                   string
	Users                  []User
	SsoRealmName           string
	SsoRealmEnabled        bool
	SsoAutoRedirectEnabled bool
}

type User struct {
	Username   string   `json:"username"`
	RealmRoles []string `json:"realmRoles"`
}

func ConvertSpecToRole(spec *v1alpha1.KeycloakRealmRoleSpec) *PrimaryRealmRole {
	rr := PrimaryRealmRole{
		Name:        spec.Name,
		Description: spec.Description,
		IsComposite: spec.Composite,
		Attributes:  spec.Attributes,
		Composites:  make([]string, 0, len(spec.Composites)),
		IsDefault:   spec.IsDefault,
	}

	for _, comp := range spec.Composites {
		rr.Composites = append(rr.Composites, comp.Name)
	}

	return &rr
}

func ConvertSpecToRealm(spec v1alpha1.KeycloakRealmSpec) *Realm {
	var users []User
	for _, item := range spec.Users {
		users = append(users, User(item))
	}

	return &Realm{
		Name:                   spec.RealmName,
		Users:                  users,
		SsoRealmName:           spec.SsoRealmName,
		SsoRealmEnabled:        spec.SSOEnabled(),
		SsoAutoRedirectEnabled: spec.SSOAutoRedirectEnabled(),
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

func ConvertSpecToClient(spec *v1alpha1.KeycloakClientSpec, clientSecret string) *Client {
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
	}
}

func getValueOrDefault(protocol *string) string {
	if protocol == nil {
		return defaultClientProtocol
	}
	return *protocol
}

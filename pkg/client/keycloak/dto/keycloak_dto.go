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
	Name            string
	Users           []User
	SsoRealmName    string
	SsoRealmEnabled bool
	ACReaderPass    string `json:"-"`
	ACCreatorPass   string `json:"-"`
}

type User struct {
	Username   string   `json:"username"`
	RealmRoles []string `json:"realmRoles"`
}

func ConvertSpecToRealm(spec v1alpha1.KeycloakRealmSpec) Realm {
	var users []User
	for _, item := range spec.Users {
		users = append(users, User(item))
	}

	return Realm{
		Name:            spec.RealmName,
		Users:           users,
		SsoRealmName:    spec.SsoRealmName,
		SsoRealmEnabled: spec.SsoRealmEnabled == nil || *spec.SsoRealmEnabled,
	}
}

type Client struct {
	ClientId                string
	ClientSecret            string `json:"-"`
	RealmName               string
	Roles                   []string
	RealmRole               RealmRole
	Public                  bool
	DirectAccess            bool
	WebUrl                  string
	Protocol                string
	Attributes              map[string]string
	AdvancedProtocolMappers bool
}

type RealmRole struct {
	Name        string
	Composite   string
	IsComposite bool
	Description string
	Attributes  map[string][]string
}

func ConvertSpecToClient(spec v1alpha1.KeycloakClientSpec, clientSecret string) Client {
	cl := Client{
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
	}
	return cl
}

func getValueOrDefault(protocol *string) string {
	if protocol == nil {
		return defaultClientProtocol
	}
	return *protocol
}

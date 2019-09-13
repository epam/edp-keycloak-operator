package dto

import (
	"github.com/epmd-edp/keycloak-operator/pkg/apis/v1/v1alpha1"
)

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
	Name  string
	Users []v1alpha1.User
}

func ConvertSpecToRealm(spec v1alpha1.KeycloakRealmSpec) Realm {
	return Realm{
		Name:  spec.RealmName,
		Users: spec.Users,
	}
}

type Client struct {
	ClientId     string
	ClientSecret string `json:"-"`
	RealmName    string
	RealmRole    RealmRole
}

type RealmRole struct {
	Name      string
	Composite string
}

func ConvertSpecToClient(spec v1alpha1.KeycloakClientSpec, clientId string, clientSecret string) Client {
	return Client{
		RealmName:    spec.TargetRealm,
		ClientId:     clientId,
		ClientSecret: clientSecret,
	}
}

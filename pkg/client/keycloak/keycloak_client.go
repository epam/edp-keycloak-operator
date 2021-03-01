package keycloak

import (
	"github.com/epmd-edp/keycloak-operator/pkg/client/keycloak/dto"
	"github.com/epmd-edp/keycloak-operator/pkg/model"
)

type Client interface {
	ExistRealm(realm dto.Realm) (*bool, error)

	CreateRealmWithDefaultConfig(realm dto.Realm) error

	DeleteRealm(realmName string) error

	ExistCentralIdentityProvider(realm dto.Realm) (*bool, error)

	CreateCentralIdentityProvider(realm dto.Realm, client dto.Client) error

	ExistClient(client dto.Client) (*bool, error)

	CreateClient(client dto.Client) error

	DeleteClient(kkClientID string, client dto.Client) error

	ExistClientRole(role dto.Client, clientRole string) (*bool, error)

	CreateClientRole(role dto.Client, clientRole string) error

	ExistRealmRole(realm dto.Realm, role dto.RealmRole) (*bool, error)

	CreateRealmRole(realm dto.Realm, role dto.RealmRole) error

	ExistRealmUser(realmName string, user dto.User) (*bool, error)

	CreateRealmUser(realmName string, user dto.User) error

	HasUserClientRole(realmName string, clientId string, user dto.User, role string) (*bool, error)

	HasUserRealmRole(realmName string, user dto.User, role string) (bool, error)

	AddRealmRoleToUser(realmName string, user dto.User, roleName string) error

	GetOpenIdConfig(realm dto.Realm) (*string, error)

	AddClientRoleToUser(realmName string, clientId string, user dto.User, role string) error

	GetClientId(client dto.Client) (*string, error)

	PutDefaultIdp(realm dto.Realm) error

	PutClientScopeMapper(clientName, scopeId, realmName string) error

	GetClientScope(scopeName, realmName string) (*model.ClientScope, error)

	LinkClientScopeToClient(clientName, scopeId, realmName string) error

	CreateClientScope(realmName string, scope model.ClientScope) error
}

type ClientFactory interface {
	New(keycloak dto.Keycloak) (Client, error)
}

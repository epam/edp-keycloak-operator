package keycloak

import (
	"github.com/Nerzal/gocloak/v8"
	"github.com/epmd-edp/keycloak-operator/pkg/client/keycloak/dto"
	"github.com/epmd-edp/keycloak-operator/pkg/model"
)

type Client interface {
	ExistRealm(realm string) (bool, error)

	CreateRealmWithDefaultConfig(realm *dto.Realm) error

	DeleteRealm(realmName string) error

	ExistCentralIdentityProvider(realm *dto.Realm) (bool, error)

	CreateCentralIdentityProvider(realm *dto.Realm, client *dto.Client) error

	ExistClient(client *dto.Client) (bool, error)

	CreateClient(client *dto.Client) error

	DeleteClient(kkClientID, realmName string) error

	ExistClientRole(role *dto.Client, clientRole string) (bool, error)

	CreateClientRole(role *dto.Client, clientRole string) error

	ExistRealmRole(realmName string, roleName string) (bool, error)

	CreateRealmRole(realmName string, role *dto.RealmRole) error

	ExistRealmUser(realmName string, user *dto.User) (bool, error)

	CreateRealmUser(realmName string, user *dto.User) error

	HasUserClientRole(realmName string, clientId string, user *dto.User, role string) (bool, error)

	HasUserRealmRole(realmName string, user *dto.User, role string) (bool, error)

	AddRealmRoleToUser(realmName string, user *dto.User, roleName string) error

	GetOpenIdConfig(realm *dto.Realm) (string, error)

	AddClientRoleToUser(realmName string, clientId string, user *dto.User, role string) error

	GetClientID(client *dto.Client) (string, error)

	PutDefaultIdp(realm *dto.Realm) error

	PutClientScopeMapper(clientName, scopeId, realmName string) error

	GetClientScope(scopeName, realmName string) (*model.ClientScope, error)

	LinkClientScopeToClient(clientName, scopeId, realmName string) error

	CreateClientScope(realmName string, scope model.ClientScope) error

	SyncClientProtocolMapper(
		client *dto.Client, crMappers []gocloak.ProtocolMapperRepresentation) error

	SyncRealmRole(realm *dto.Realm, role *dto.RealmRole) error

	DeleteRealmRole(realm, roleName string) error

	SyncServiceAccountRoles(realm, clientID string, realmRoles []string,
		clientRoles map[string][]string) error
}

type ClientFactory interface {
	New(keycloak dto.Keycloak) (Client, error)
}

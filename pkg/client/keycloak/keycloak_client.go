package keycloak

import (
	"github.com/Nerzal/gocloak/v8"
	"github.com/epmd-edp/keycloak-operator/pkg/apis/v1/v1alpha1"
	"github.com/epmd-edp/keycloak-operator/pkg/client/keycloak/dto"
	"github.com/epmd-edp/keycloak-operator/pkg/model"
)

type Client interface {
	KCloakGroups
	KCloakUsers
	KCloakRealms
	KCloakClients
	KCloakRealmRoles
	KCloakClientRoles

	ExistCentralIdentityProvider(realm *dto.Realm) (bool, error)
	CreateCentralIdentityProvider(realm *dto.Realm, client *dto.Client) error
	GetOpenIdConfig(realm *dto.Realm) (string, error)
	PutDefaultIdp(realm *dto.Realm) error
	SyncServiceAccountRoles(realm, clientID string, realmRoles []string,
		clientRoles map[string][]string) error
}

type KCloakGroups interface {
	SyncRealmGroup(realm string, spec *v1alpha1.KeycloakRealmGroupSpec) (string, error)
	DeleteGroup(realm, groupName string) error
}

type KCloakUsers interface {
	ExistRealmUser(realmName string, user *dto.User) (bool, error)
	CreateRealmUser(realmName string, user *dto.User) error
}

type KCloakRealms interface {
	ExistRealm(realm string) (bool, error)
	CreateRealmWithDefaultConfig(realm *dto.Realm) error
	DeleteRealm(realmName string) error
}

type KCloakClients interface {
	ExistClient(client *dto.Client) (bool, error)
	CreateClient(client *dto.Client) error
	DeleteClient(kkClientID, realmName string) error
	CreateClientScope(realmName string, scope model.ClientScope) error
	SyncClientProtocolMapper(
		client *dto.Client, crMappers []gocloak.ProtocolMapperRepresentation) error
	GetClientID(client *dto.Client) (string, error)
	PutClientScopeMapper(clientName, scopeId, realmName string) error
	GetClientScope(scopeName, realmName string) (*model.ClientScope, error)
	LinkClientScopeToClient(clientName, scopeId, realmName string) error
}

type KCloakRealmRoles interface {
	ExistRealmRole(realmName string, roleName string) (bool, error)
	CreateRealmRole(realmName string, role *dto.RealmRole) error
	HasUserRealmRole(realmName string, user *dto.User, role string) (bool, error)
	AddRealmRoleToUser(realmName string, user *dto.User, roleName string) error
	SyncRealmRole(realm *dto.Realm, role *dto.RealmRole) error
	DeleteRealmRole(realm, roleName string) error
}

type KCloakClientRoles interface {
	ExistClientRole(role *dto.Client, clientRole string) (bool, error)
	CreateClientRole(role *dto.Client, clientRole string) error
	HasUserClientRole(realmName string, clientId string, user *dto.User, role string) (bool, error)
	AddClientRoleToUser(realmName string, clientId string, user *dto.User, role string) error
}

type ClientFactory interface {
	New(keycloak dto.Keycloak) (Client, error)
}

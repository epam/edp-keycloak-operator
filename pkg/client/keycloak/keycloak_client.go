package keycloak

import (
	"context"

	"github.com/Nerzal/gocloak/v8"
	"github.com/epam/edp-keycloak-operator/pkg/apis/v1/v1alpha1"
	"github.com/epam/edp-keycloak-operator/pkg/client/keycloak/adapter"
	"github.com/epam/edp-keycloak-operator/pkg/client/keycloak/dto"
	"github.com/epam/edp-keycloak-operator/pkg/model"
)

type Client interface {
	KCloakGroups
	KCloakUsers
	KCloakRealms
	KCloakClients
	KCloakRealmRoles
	KCloakClientRoles
	KAuthFlow

	ExistCentralIdentityProvider(realm *dto.Realm) (bool, error)
	CreateCentralIdentityProvider(realm *dto.Realm, client *dto.Client) error
	GetOpenIdConfig(realm *dto.Realm) (string, error)
	PutDefaultIdp(realm *dto.Realm) error
	SyncServiceAccountRoles(realm, clientID string, realmRoles []string,
		clientRoles map[string][]string) error
	SetServiceAccountAttributes(realm, clientID string, attributes map[string]string) error
}

type KAuthFlow interface {
	SyncAuthFlow(realmName string, flow *adapter.KeycloakAuthFlow) error
	DeleteAuthFlow(realmName, alias string) error
	SetRealmBrowserFlow(realmName string, flowAlias string) error
}

type KCloakGroups interface {
	SyncRealmGroup(realm string, spec *v1alpha1.KeycloakRealmGroupSpec) (string, error)
	DeleteGroup(realm, groupName string) error
}

type KCloakUsers interface {
	ExistRealmUser(realmName string, user *dto.User) (bool, error)
	CreateRealmUser(realmName string, user *dto.User) error
	SyncRealmUser(realmName string, user *adapter.KeycloakUser) error
}

type KCloakRealms interface {
	ExistRealm(realm string) (bool, error)
	CreateRealmWithDefaultConfig(realm *dto.Realm) error
	DeleteRealm(realmName string) error
	SyncRealmIdentityProviderMappers(realmName string, mappers []dto.IdentityProviderMapper) error
	UpdateRealmSettings(realmName string, realmSettings *adapter.RealmSettings) error
}

type KCloakClients interface {
	ExistClient(clientID, realm string) (bool, error)
	CreateClient(client *dto.Client) error
	DeleteClient(kkClientID, realmName string) error
	CreateClientScope(ctx context.Context, realmName string, scope *adapter.ClientScope) (string, error)
	SyncClientProtocolMapper(
		client *dto.Client, crMappers []gocloak.ProtocolMapperRepresentation) error
	GetClientID(clientID, realm string) (string, error)
	PutClientScopeMapper(clientName, scopeId, realmName string) error
	GetClientScope(scopeName, realmName string) (*model.ClientScope, error)
	LinkClientScopeToClient(clientName, scopeId, realmName string) error
	UpdateClientScope(ctx context.Context, realmName, scopeID string, scope *adapter.ClientScope) error
	DeleteClientScope(ctx context.Context, realmName, scopeID string) error
}

type KCloakRealmRoles interface {
	ExistRealmRole(realmName string, roleName string) (bool, error)
	CreateIncludedRealmRole(realmName string, role *dto.IncludedRealmRole) error
	CreatePrimaryRealmRole(realmName string, role *dto.PrimaryRealmRole) (string, error)
	HasUserRealmRole(realmName string, user *dto.User, role string) (bool, error)
	AddRealmRoleToUser(realmName, username, roleName string) error
	SyncRealmRole(realmName string, role *dto.PrimaryRealmRole) error
	DeleteRealmRole(realm, roleName string) error
}

type KCloakClientRoles interface {
	ExistClientRole(role *dto.Client, clientRole string) (bool, error)
	CreateClientRole(role *dto.Client, clientRole string) error
	HasUserClientRole(realmName string, clientId string, user *dto.User, role string) (bool, error)
	AddClientRoleToUser(realmName string, clientId string, user *dto.User, role string) error
}

package keycloak

import (
	"context"

	"github.com/Nerzal/gocloak/v12"

	keycloakApi "github.com/epam/edp-keycloak-operator/api/v1"
	"github.com/epam/edp-keycloak-operator/pkg/client/keycloak/adapter"
	"github.com/epam/edp-keycloak-operator/pkg/client/keycloak/dto"
)

type Client interface {
	KCloakGroups
	KCloakUsers
	KCloakRealms
	KCloakClients
	KCloakRealmRoles
	KCloakClientRoles
	KAuthFlow
	KCloakComponents
	KCloakClientScope
	KIdentityProvider

	GetOpenIdConfig(realm *dto.Realm) (string, error)
	SyncServiceAccountRoles(realm, clientID string, realmRoles []string,
		clientRoles map[string][]string, addOnly bool) error
	SetServiceAccountAttributes(realm, clientID string, attributes map[string]string, addOnly bool) error
	ExportToken() ([]byte, error)
}

type KIdentityProvider interface {
	CreateIdentityProvider(ctx context.Context, realm string, idp *adapter.IdentityProvider) error
	UpdateIdentityProvider(ctx context.Context, realm string, idp *adapter.IdentityProvider) error
	GetIdentityProvider(ctx context.Context, realm, alias string) (*adapter.IdentityProvider, error)
	IdentityProviderExists(ctx context.Context, realm, alias string) (bool, error)
	DeleteIdentityProvider(ctx context.Context, realm, alias string) error

	CreateIDPMapper(ctx context.Context, realm, idpAlias string, mapper *adapter.IdentityProviderMapper) (string, error)
	UpdateIDPMapper(ctx context.Context, realm, idpAlias string, mapper *adapter.IdentityProviderMapper) error
	DeleteIDPMapper(ctx context.Context, realm, idpAlias, mapperID string) error
	GetIDPMappers(ctx context.Context, realm, idpAlias string) ([]adapter.IdentityProviderMapper, error)
}

type KAuthFlow interface {
	SyncAuthFlow(realmName string, flow *adapter.KeycloakAuthFlow) error
	DeleteAuthFlow(realmName string, flow *adapter.KeycloakAuthFlow) error
	SetRealmBrowserFlow(ctx context.Context, realmName string, flowAlias string) error
}

type KCloakGroups interface {
	SyncRealmGroup(realm string, spec *keycloakApi.KeycloakRealmGroupSpec) (string, error)
	DeleteGroup(ctx context.Context, realm, groupName string) error
	GetGroups(ctx context.Context, realm string) (map[string]*gocloak.Group, error)
}

type KCloakUsers interface {
	ExistRealmUser(realmName string, user *dto.User) (bool, error)
	CreateRealmUser(realmName string, user *dto.User) error
	SyncRealmUser(ctx context.Context, realmName string, user *adapter.KeycloakUser, addOnly bool) error
	DeleteRealmUser(ctx context.Context, realmName, username string) error
	GetUsersByNames(ctx context.Context, realm string, names []string) (map[string]gocloak.User, error)
}

type KCloakRealms interface {
	GetRealm(ctx context.Context, realm string) (*gocloak.RealmRepresentation, error)
	ExistRealm(realm string) (bool, error)
	CreateRealmWithDefaultConfig(realm *dto.Realm) error
	DeleteRealm(ctx context.Context, realmName string) error
	SyncRealmIdentityProviderMappers(realmName string, mappers []dto.IdentityProviderMapper) error
	UpdateRealmSettings(realmName string, realmSettings *adapter.RealmSettings) error
	SetRealmEventConfig(realmName string, eventConfig *adapter.RealmEventConfig) error
}

type KCloakClients interface {
	ExistClient(clientID, realm string) (bool, error)
	CreateClient(ctx context.Context, client *dto.Client) error
	DeleteClient(ctx context.Context, kcClientID, realmName string) error
	UpdateClient(ctx context.Context, client *dto.Client) error
	GetClients(ctx context.Context, realm string) (map[string]*gocloak.Client, error)
	GetClient(ctx context.Context, realm, client string) (*gocloak.Client, error)
	SyncClientProtocolMapper(
		client *dto.Client, crMappers []gocloak.ProtocolMapperRepresentation, addOnly bool) error
	GetClientID(clientID, realm string) (string, error)
	AddDefaultScopeToClient(ctx context.Context, realmName, clientName string, scopes []adapter.ClientScope) error
	AddOptionalScopeToClient(ctx context.Context, realmName, clientName string, scopes []adapter.ClientScope) error

	GetScopes(ctx context.Context, realm, idOfClient string) (map[string]gocloak.ScopeRepresentation, error)
	CreateScope(ctx context.Context, realm, idOfClient string, scope string) (*gocloak.ScopeRepresentation, error)
	DeleteScope(ctx context.Context, realm, idOfClient string, scope string) error

	GetPolicies(ctx context.Context, realm, idOfClient string) (map[string]*gocloak.PolicyRepresentation, error)
	CreatePolicy(ctx context.Context, realm, idOfClient string, policy gocloak.PolicyRepresentation) (*gocloak.PolicyRepresentation, error)
	UpdatePolicy(ctx context.Context, realm, idOfClient string, policy gocloak.PolicyRepresentation) error
	DeletePolicy(ctx context.Context, realm, idOfClient, policyID string) error

	GetPermissions(ctx context.Context, realm, idOfClient string) (map[string]gocloak.PermissionRepresentation, error)
	CreatePermission(ctx context.Context, realm, idOfClient string, permission gocloak.PermissionRepresentation) (*gocloak.PermissionRepresentation, error)
	UpdatePermission(ctx context.Context, realm, idOfClient string, permission gocloak.PermissionRepresentation) error
	DeletePermission(ctx context.Context, realm, idOfClient, permissionID string) error

	GetResources(ctx context.Context, realm, idOfClient string) (map[string]gocloak.ResourceRepresentation, error)
}

type KCloakClientScope interface {
	PutClientScopeMapper(realmName, scopeID string, protocolMapper *adapter.ProtocolMapper) error
	GetClientScope(scopeName, realmName string) (*adapter.ClientScope, error)
	GetClientScopesByNames(ctx context.Context, realmName string, scopeNames []string) ([]adapter.ClientScope, error)
	UpdateClientScope(ctx context.Context, realmName, scopeID string, scope *adapter.ClientScope) error
	DeleteClientScope(ctx context.Context, realmName, scopeID string) error
	GetDefaultClientScopesForRealm(ctx context.Context, realm string) ([]adapter.ClientScope, error)
	CreateClientScope(ctx context.Context, realmName string, scope *adapter.ClientScope) (string, error)
	GetClientScopeMappers(ctx context.Context, realmName, scopeID string) ([]adapter.ProtocolMapper, error)
	GetClientScopes(ctx context.Context, realm string) (map[string]gocloak.ClientScope, error)
}

type KCloakRealmRoles interface {
	ExistRealmRole(realmName string, roleName string) (bool, error)
	CreateIncludedRealmRole(realmName string, role *dto.IncludedRealmRole) error
	CreatePrimaryRealmRole(ctx context.Context, realmName string, role *dto.PrimaryRealmRole) (string, error)
	HasUserRealmRole(realmName string, user *dto.User, role string) (bool, error)
	AddRealmRoleToUser(ctx context.Context, realmName, username, roleName string) error
	SyncRealmRole(ctx context.Context, realmName string, role *dto.PrimaryRealmRole) error
	DeleteRealmRole(ctx context.Context, realm, roleName string) error
}

type KCloakClientRoles interface {
	ExistClientRole(role *dto.Client, clientRole string) (bool, error)
	CreateClientRole(role *dto.Client, clientRole string) error
	HasUserClientRole(realmName string, clientId string, user *dto.User, role string) (bool, error)
	AddClientRoleToUser(realmName string, clientId string, user *dto.User, role string) error
	GetRealmRoles(ctx context.Context, realm string) (map[string]gocloak.Role, error)
}

type KCloakComponents interface {
	CreateComponent(ctx context.Context, realmName string, component *adapter.Component) error
	UpdateComponent(ctx context.Context, realmName string, component *adapter.Component) error
	DeleteComponent(ctx context.Context, realmName, componentName string) error
	GetComponent(ctx context.Context, realmName, componentName string) (*adapter.Component, error)
}

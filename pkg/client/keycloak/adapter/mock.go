package adapter

import (
	"context"

	"github.com/Nerzal/gocloak/v8"
	"github.com/epam/edp-keycloak-operator/pkg/apis/v1/v1alpha1"
	"github.com/epam/edp-keycloak-operator/pkg/client/keycloak/dto"
	"github.com/stretchr/testify/mock"
)

type Mock struct {
	mock.Mock
	ExportTokenResult []byte
	ExportTokenErr    error
}

func (m *Mock) PutDefaultIdp(realm *dto.Realm) error {
	return m.Called(realm).Error(0)
}

func (m *Mock) ExistRealm(realm string) (bool, error) {
	args := m.Called(realm)
	if args.Get(0) == nil {
		return false, args.Error(1)
	}

	return args.Bool(0), args.Error(1)
}

func (m *Mock) CreateRealmWithDefaultConfig(realm *dto.Realm) error {
	args := m.Called(realm)
	return args.Error(0)
}

func (m *Mock) ExistCentralIdentityProvider(realm *dto.Realm) (bool, error) {
	args := m.Called(realm)
	if args.Get(0) == nil {
		return false, args.Error(1)
	}
	res := args.Bool(0)
	return res, args.Error(1)
}

func (m *Mock) CreateCentralIdentityProvider(realm *dto.Realm, client *dto.Client) error {
	return m.Called(realm, client).Error(0)
}

func (m *Mock) ExistClient(clientID, realm string) (bool, error) {
	args := m.Called(clientID, realm)
	if args.Get(0) == nil {
		return false, args.Error(1)
	}
	res := args.Bool(0)
	return res, args.Error(1)
}

func (m *Mock) CreateClient(client *dto.Client) error {
	args := m.Called(client)
	return args.Error(0)
}

func (m *Mock) ExistClientRole(role *dto.Client, clientRole string) (bool, error) {
	panic("implement me")
}

func (m *Mock) CreateClientRole(role *dto.Client, clientRole string) error {
	panic("implement me")
}

func (m *Mock) ExistRealmRole(realm string, role string) (bool, error) {
	args := m.Called(realm, role)
	return args.Bool(0), args.Error(1)
}

func (m *Mock) CreateIncludedRealmRole(realm string, role *dto.IncludedRealmRole) error {
	args := m.Called(realm, role)
	return args.Error(0)
}

func (m *Mock) CreatePrimaryRealmRole(realm string, role *dto.PrimaryRealmRole) (string, error) {
	args := m.Called(realm, role)
	return args.String(0), args.Error(1)
}

func (m *Mock) ExistRealmUser(realmName string, user *dto.User) (bool, error) {
	called := m.Called(realmName, user)
	return called.Bool(0), called.Error(1)
}

func (m *Mock) CreateRealmUser(realmName string, user *dto.User) error {
	return m.Called(realmName, user).Error(0)
}

func (m *Mock) HasUserClientRole(realmName string, clientId string, user *dto.User, role string) (bool, error) {
	called := m.Called(realmName, clientId, user, role)
	return called.Bool(0), called.Error(1)
}

func (m *Mock) GetOpenIdConfig(realm *dto.Realm) (string, error) {
	args := m.Called(realm)
	if args.Get(0) == nil {
		return "", args.Error(1)
	}

	return args.String(0), args.Error(1)
}

func (m *Mock) AddClientRoleToUser(realmName string, clientId string, user *dto.User, role string) error {
	return m.Called(realmName, clientId, user, role).Error(0)
}

func (m *Mock) GetClientID(clientID, realm string) (string, error) {
	args := m.Called(clientID, realm)
	if args.Get(0) == nil {
		return "", args.Error(1)
	}
	res := args.String(0)
	return res, args.Error(1)
}

func (m *Mock) MapRoleToUser(realmName string, user dto.User, role string) error {
	panic("implement me")
}

func (m *Mock) ExistMapRoleToUser(realmName string, user dto.User, role string) (*bool, error) {
	panic("implement me")
}

func (m *Mock) AddRealmRoleToUser(ctx context.Context, realmName, username, roleName string) error {
	return m.Called(realmName, username, roleName).Error(0)
}

func (m *Mock) DeleteClient(ctx context.Context, kcClientID string, realName string) error {
	return m.Called(kcClientID, realName).Error(0)
}

func (m *Mock) DeleteRealmRole(ctx context.Context, realm, roleName string) error {
	return m.Called(realm, roleName).Error(0)
}

func (m *Mock) DeleteRealm(ctx context.Context, realmName string) error {
	return m.Called(realmName).Error(0)
}

func (m *Mock) GetClientScope(scopeName, realmName string) (*ClientScope, error) {
	called := m.Called(scopeName, realmName)
	if err := called.Error(1); err != nil {
		return nil, err
	}

	return called.Get(0).(*ClientScope), nil
}

func (m *Mock) HasUserRealmRole(realmName string, user *dto.User, role string) (bool, error) {
	called := m.Called(realmName, user, role)
	return called.Bool(0), called.Error(1)
}

func (m *Mock) LinkClientScopeToClient(clientName, scopeId, realmName string) error {
	return m.Called(clientName, scopeId, realmName).Error(0)
}

func (m *Mock) PutClientScopeMapper(realmName, scopeID string, protocolMapper *ProtocolMapper) error {
	return m.Called(realmName, scopeID, protocolMapper).Error(0)
}

func (m *Mock) SyncClientProtocolMapper(
	client *dto.Client, crMappers []gocloak.ProtocolMapperRepresentation, addOnly bool) error {
	return m.Called(client, crMappers, addOnly).Error(0)
}

func (m *Mock) SyncRealmRole(realmName string, role *dto.PrimaryRealmRole) error {
	return m.Called(realmName, role).Error(0)
}

func (m *Mock) SyncServiceAccountRoles(realm, clientID string, realmRoles []string,
	clientRoles map[string][]string, addOnly bool) error {
	return m.Called(realm, clientID, realmRoles, clientRoles, addOnly).Error(0)
}

func (m *Mock) SyncRealmGroup(realmName string, spec *v1alpha1.KeycloakRealmGroupSpec) (string, error) {
	called := m.Called(realmName, spec)
	return called.String(0), called.Error(1)
}

func (m *Mock) DeleteGroup(ctx context.Context, realm, groupName string) error {
	return m.Called(realm, groupName).Error(0)
}

func (m *Mock) SyncRealmIdentityProviderMappers(realmName string,
	mappers []dto.IdentityProviderMapper) error {
	return m.Called(realmName, mappers).Error(0)
}

func (m *Mock) DeleteAuthFlow(realmName, alias string) error {
	return m.Called(realmName, alias).Error(0)
}

func (m *Mock) SyncAuthFlow(realmName string, flow *KeycloakAuthFlow) error {
	return m.Called(realmName, flow).Error(0)
}

func (m *Mock) SetRealmBrowserFlow(realmName string, flowAlias string) error {
	return m.Called(realmName, flowAlias).Error(0)
}
func (m *Mock) UpdateRealmSettings(realmName string, realmSettings *RealmSettings) error {
	return m.Called(realmName, realmSettings).Error(0)
}

func (m *Mock) SyncRealmUser(ctx context.Context, realmName string, user *KeycloakUser, addOnly bool) error {
	return m.Called(realmName, user, addOnly).Error(0)
}

func (m *Mock) SetServiceAccountAttributes(realm, clientID string, attributes map[string]string, addOnly bool) error {
	return m.Called(realm, clientID, attributes, addOnly).Error(0)
}

func (m *Mock) CreateClientScope(ctx context.Context, realmName string, scope *ClientScope) (string, error) {
	called := m.Called(realmName, scope)
	if err := called.Error(1); err != nil {
		return "", err
	}

	return called.String(0), nil
}

func (m *Mock) DeleteClientScope(ctx context.Context, realmName, scopeID string) error {
	return m.Called(realmName, scopeID).Error(0)
}

func (m *Mock) UpdateClientScope(ctx context.Context, realmName, scopeID string, scope *ClientScope) error {
	return m.Called(realmName, scopeID, scope).Error(0)
}

func (m *Mock) SetRealmEventConfig(realmName string, eventConfig *RealmEventConfig) error {
	return m.Called(realmName, eventConfig).Error(0)
}

func (m *Mock) ExportToken() ([]byte, error) {
	return m.ExportTokenResult, m.ExportTokenErr
}

func (m *Mock) CreateComponent(ctx context.Context, realmName string, component *Component) error {
	return m.Called(realmName, component).Error(0)
}

func (m *Mock) UpdateComponent(ctx context.Context, realmName string, component *Component) error {
	return m.Called(realmName, component).Error(0)
}

func (m *Mock) DeleteComponent(ctx context.Context, realmName, componentName string) error {
	return m.Called(realmName, componentName).Error(0)
}

func (m *Mock) GetComponent(ctx context.Context, realmName, componentName string) (*Component, error) {
	called := m.Called(realmName, componentName)
	if err := called.Error(1); err != nil {
		return nil, err
	}

	return called.Get(0).(*Component), nil
}

func (m *Mock) GetDefaultClientScopesForRealm(ctx context.Context, realm string) ([]ClientScope, error) {
	called := m.Called(realm)
	if err := called.Error(1); err != nil {
		return nil, err
	}

	return called.Get(0).([]ClientScope), nil
}

func (m *Mock) GetClientScopeMappers(ctx context.Context, realmName, scopeID string) ([]ProtocolMapper, error) {
	called := m.Called(realmName, scopeID)
	if err := called.Error(1); err != nil {
		return nil, err
	}

	return called.Get(0).([]ProtocolMapper), nil
}

func (m *Mock) CreateIDPMapper(ctx context.Context, realm, idpAlias string, mapper *IdentityProviderMapper) (string, error) {
	called := m.Called(realm, idpAlias, mapper)
	return called.String(0), called.Error(1)
}

func (m *Mock) UpdateIDPMapper(ctx context.Context, realm, idpAlias string, mapper *IdentityProviderMapper) error {
	return m.Called(realm, idpAlias, mapper).Error(0)
}

func (m *Mock) DeleteIDPMapper(ctx context.Context, realm, idpAlias, mapperID string) error {
	return m.Called(realm, idpAlias, mapperID).Error(0)
}

func (m *Mock) GetIDPMappers(ctx context.Context, realm, idpAlias string) ([]IdentityProviderMapper, error) {
	called := m.Called(realm, idpAlias)
	if err := called.Error(1); err != nil {
		return nil, err
	}

	return called.Get(0).([]IdentityProviderMapper), nil
}

func (m *Mock) DeleteIdentityProvider(ctx context.Context, realm, alias string) error {
	return m.Called(realm, alias).Error(0)
}

func (m *Mock) GetIdentityProvider(ctx context.Context, realm, alias string) (*IdentityProvider, error) {
	called := m.Called(realm, alias)
	if err := called.Error(1); err != nil {
		return nil, err
	}

	return called.Get(0).(*IdentityProvider), nil
}

func (m *Mock) UpdateIdentityProvider(ctx context.Context, realm string, idp *IdentityProvider) error {
	return m.Called(realm, idp).Error(0)
}

func (m *Mock) CreateIdentityProvider(ctx context.Context, realm string, idp *IdentityProvider) error {
	return m.Called(realm, idp).Error(0)
}

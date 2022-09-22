package adapter

import (
	"context"
	"fmt"
	"strings"

	"github.com/pkg/errors"
)

const (
	OpenIdProtocol     = "openid-connect"
	OIDCAudienceMapper = "oidc-audience-mapper"
)

type ClientScope struct {
	ID              string            `json:"id,omitempty"`
	Name            string            `json:"name"`
	Description     string            `json:"description"`
	Attributes      map[string]string `json:"attributes"`
	Protocol        string            `json:"protocol"`
	ProtocolMappers []ProtocolMapper  `json:"protocolMappers"`
	Default         bool              `json:"-"`
}

type ProtocolMapper struct {
	Name           string            `json:"name"`
	Protocol       string            `json:"protocol"`
	ProtocolMapper string            `json:"protocolMapper"`
	Config         map[string]string `json:"config"`
}

func (a GoCloakAdapter) CreateClientScope(ctx context.Context, realmName string, scope *ClientScope) (string, error) {
	rsp, err := a.startRestyRequest().
		SetContext(ctx).
		SetPathParams(map[string]string{
			keycloakApiParamRealm: realmName,
		}).
		SetBody(scope).
		Post(a.basePath + postClientScope)

	if err = a.checkError(err, rsp); err != nil {
		return "", errors.Wrap(err, "unable to create client scope")
	}

	id, err := getIDFromResponseLocation(rsp.RawResponse)
	if err != nil {
		return "", errors.Wrap(err, "unable to get flow id")
	}

	if scope.Default {
		if err := a.setDefaultClientScopeForRealm(ctx, realmName, id); err != nil {
			return id, errors.Wrap(err, "unable to set default client scope for realm")
		}
	}

	return id, nil
}

func (a GoCloakAdapter) UpdateClientScope(ctx context.Context, realmName, scopeID string, scope *ClientScope) error {
	if err := a.syncClientScopeProtocolMappers(ctx, realmName, scopeID, scope.ProtocolMappers); err != nil {
		return errors.Wrap(err, "unable to sync client scope protocol mappers")
	}

	rsp, err := a.startRestyRequest().
		SetContext(ctx).
		SetPathParams(map[string]string{
			keycloakApiParamRealm: realmName,
			keycloakApiParamId:    scopeID,
		}).
		SetBody(scope).
		Put(a.basePath + putClientScope)

	if err = a.checkError(err, rsp); err != nil {
		return errors.Wrap(err, "unable to update client scope")
	}

	needToUpdateDefault, err := a.needToUpdateDefault(ctx, realmName, scope)
	if err != nil {
		return errors.Wrap(err, "unable to check if need to update default")
	}

	if !needToUpdateDefault {
		return nil
	}

	if scope.Default {
		if err := a.setDefaultClientScopeForRealm(ctx, realmName, scopeID); err != nil {
			return errors.Wrap(err, "unable to set default client scope for realm")
		}

		return nil
	}

	if err := a.unsetDefaultClientScopeForRealm(ctx, realmName, scopeID); err != nil {
		return errors.Wrap(err, "unable to unset default client scope for realm")
	}

	return nil
}

func (a GoCloakAdapter) needToUpdateDefault(ctx context.Context, realmName string, scope *ClientScope) (bool, error) {
	defaultScopes, err := a.GetDefaultClientScopesForRealm(ctx, realmName)
	if err != nil {
		return false, errors.Wrap(err, "unable to get default client scopes")
	}

	currentScopeDefaultState := false

	for _, s := range defaultScopes {
		if s.Name == scope.Name {
			currentScopeDefaultState = true
			break
		}
	}

	return currentScopeDefaultState != scope.Default, nil
}

// TODO: add context.
func (a GoCloakAdapter) GetClientScope(scopeName, realmName string) (*ClientScope, error) {
	log := a.log.WithValues("scopeName", scopeName, logKeyRealm, realmName)
	log.Info("Start get Client Scope...")

	var result []ClientScope
	resp, err := a.client.RestyClient().R().
		SetAuthToken(a.token.AccessToken).
		SetHeader(contentTypeHeader, contentTypeJson).
		SetPathParams(map[string]string{
			keycloakApiParamRealm: realmName,
		}).
		SetResult(&result).
		Get(a.basePath + getRealmClientScopes)

	if err = a.checkError(err, resp); err != nil {
		return nil, err
	}

	if result == nil {
		return nil, NotFoundError(fmt.Sprintf("realm %v doesnt contain client scopes, rsp: %s", realmName, resp.String()))
	}

	scope, err := getClientScope(scopeName, result)
	if err != nil {
		return nil, err
	}

	log.Info("End get Client Scope", "scope", scope)

	return scope, err
}

func getClientScope(name string, clientScopes []ClientScope) (*ClientScope, error) {
	for _, cs := range clientScopes {
		if cs.Name == name {
			return &cs, nil
		}
	}

	return nil, NotFoundError(fmt.Sprintf("scope %v was not found", name))
}

func (a GoCloakAdapter) GetClientScopesByNames(ctx context.Context, realmName string, scopeNames []string) ([]ClientScope, error) {
	log := a.log.WithValues("scopeNames", strings.Join(scopeNames, ","), "realm", realmName)
	log.Info("Start get Client Scopes by name...")

	var result []ClientScope

	resp, err := a.client.RestyClient().R().
		SetContext(ctx).
		SetAuthToken(a.token.AccessToken).
		SetHeader(contentTypeHeader, contentTypeJson).
		SetPathParams(map[string]string{
			keycloakApiParamRealm: realmName,
		}).
		SetResult(&result).
		Get(a.basePath + getRealmClientScopes)

	if err = a.checkError(err, resp); err != nil {
		return nil, err
	}

	log.Info("End get Client Scopes...")

	return a.filterClientScopes(scopeNames, result)
}

func (a GoCloakAdapter) filterClientScopes(scopeNames []string, clientScopes []ClientScope) ([]ClientScope, error) {
	clientScopesMap := make(map[string]ClientScope)
	for _, s := range clientScopes {
		clientScopesMap[s.Name] = s
	}

	result := make([]ClientScope, 0, len(scopeNames))
	missingScopes := make([]string, 0, len(scopeNames))

	for _, scopeName := range scopeNames {
		scope, ok := clientScopesMap[scopeName]
		if ok {
			result = append(result, scope)
			continue
		}

		missingScopes = append(missingScopes, scopeName)
	}

	if len(missingScopes) > 0 {
		return nil, errors.Errorf("failed to get '%s' keycloak client scopes", strings.Join(missingScopes, ","))
	}

	return result, nil
}

func (a GoCloakAdapter) DeleteClientScope(ctx context.Context, realmName, scopeID string) error {
	if err := a.unsetDefaultClientScopeForRealm(ctx, realmName, scopeID); err != nil {
		return errors.Wrap(err, "unable to unset default client scope for realm")
	}

	if err := a.client.DeleteClientScope(ctx, a.token.AccessToken, realmName, scopeID); err != nil {
		return errors.Wrap(err, "unable to delete client scope")
	}

	return nil
}

func (a GoCloakAdapter) syncClientScopeProtocolMappers(ctx context.Context, realm, scopeID string,
	instanceProtocolMappers []ProtocolMapper) error {
	scope, err := a.client.GetClientScope(ctx, a.token.AccessToken, realm, scopeID)
	if err != nil {
		return errors.Wrap(err, "unable to get client scope")
	}

	if scope.ProtocolMappers != nil {
		for _, pm := range *scope.ProtocolMappers {
			rsp, err := a.startRestyRequest().SetContext(ctx).SetPathParams(map[string]string{
				keycloakApiParamRealm:         realm,
				keycloakApiParamClientScopeId: scopeID,
				"protocolMapperID":            *pm.ID,
			}).Delete(a.basePath + deleteClientScopeProtocolMapper)

			if err = a.checkError(err, rsp); err != nil {
				return errors.Wrap(err, "error during client scope protocol mapper deletion")
			}
		}
	}

	for _, pm := range instanceProtocolMappers {
		rsp, err := a.startRestyRequest().SetContext(ctx).SetPathParams(map[string]string{
			keycloakApiParamRealm:         realm,
			keycloakApiParamClientScopeId: scopeID,
		}).SetBody(&pm).Post(a.basePath + createClientScopeProtocolMapper)

		if err = a.checkError(err, rsp); err != nil {
			return errors.Wrap(err, "error during client scope protocol mapper creation")
		}
	}

	return nil
}

func (a GoCloakAdapter) GetDefaultClientScopesForRealm(ctx context.Context, realmName string) ([]ClientScope, error) {
	var scopes []ClientScope

	rsp, err := a.startRestyRequest().SetContext(ctx).SetPathParams(map[string]string{
		keycloakApiParamRealm: realmName,
	}).SetResult(&scopes).Get(a.basePath + getDefaultClientScopes)

	if err = a.checkError(err, rsp); err != nil {
		return nil, errors.Wrap(err, "unable to get default client scopes for realm")
	}

	return scopes, nil
}

func (a GoCloakAdapter) setDefaultClientScopeForRealm(ctx context.Context, realm, scopeID string) error {
	rsp, err := a.startRestyRequest().SetContext(ctx).SetPathParams(map[string]string{
		keycloakApiParamRealm:         realm,
		keycloakApiParamClientScopeId: scopeID,
	}).SetBody(map[string]string{
		keycloakApiParamRealm:         realm,
		keycloakApiParamClientScopeId: scopeID,
	}).Put(a.basePath + putDefaultClientScope)

	if err = a.checkError(err, rsp); err != nil {
		return errors.Wrap(err, "unable to set default client scope for realm")
	}

	return nil
}

func (a GoCloakAdapter) unsetDefaultClientScopeForRealm(ctx context.Context, realm, scopeID string) error {
	rsp, err := a.startRestyRequest().SetContext(ctx).SetPathParams(map[string]string{
		keycloakApiParamRealm:         realm,
		keycloakApiParamClientScopeId: scopeID,
	}).Delete(a.basePath + deleteDefaultClientScope)

	if err = a.checkError(err, rsp); err != nil {
		return errors.Wrap(err, "unable to unset default client scope for realm")
	}

	return nil
}

func (a GoCloakAdapter) GetClientScopeMappers(ctx context.Context, realmName, scopeID string) ([]ProtocolMapper, error) {
	var mappers []ProtocolMapper
	rsp, err := a.startRestyRequest().SetContext(ctx).SetPathParams(map[string]string{
		keycloakApiParamRealm: realmName,
		"scopeId":             scopeID,
	}).SetResult(&mappers).Get(a.basePath + postClientScopeMapper)

	if err = a.checkError(err, rsp); err != nil {
		return nil, errors.Wrap(err, "unable to get client scope mappers")
	}

	return mappers, nil
}

func (a GoCloakAdapter) PutClientScopeMapper(realmName, scopeID string, protocolMapper *ProtocolMapper) error {
	log := a.log.WithValues("scopeId", scopeID, logKeyRealm, realmName)
	log.Info("Start put Client Scope mapper...")

	resp, err := a.client.RestyClient().R().
		SetAuthToken(a.token.AccessToken).
		SetHeader(contentTypeHeader, contentTypeJson).
		SetPathParams(map[string]string{
			keycloakApiParamRealm: realmName,
			"scopeId":             scopeID,
		}).
		SetBody(protocolMapper).
		Post(a.basePath + postClientScopeMapper)
	if err = a.checkError(err, resp); err != nil {
		return errors.Wrap(err, "unable to put client scope mapper")
	}

	log.Info("Client Scope mapper was successfully configured!")

	return nil
}

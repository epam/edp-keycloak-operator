package adapter

import (
	"context"
	"fmt"

	"github.com/epam/edp-keycloak-operator/pkg/model"
	"github.com/pkg/errors"
)

type ClientScope struct {
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
			"realm": realmName,
		}).
		SetBody(scope).
		Post(a.basePath + postClientScope)

	if err := a.checkError(err, rsp); err != nil {
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
			"realm": realmName,
			"id":    scopeID,
		}).
		SetBody(scope).
		Put(a.basePath + putClientScope)

	if err := a.checkError(err, rsp); err != nil {
		return errors.Wrap(err, "unable to update client scope")
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

func (a GoCloakAdapter) GetClientScope(scopeName, realmName string) (*model.ClientScope, error) {
	log := a.log.WithValues("scopeName", scopeName, "realm", realmName)
	log.Info("Start get Client Scope...")
	var result []*model.ClientScope
	resp, err := a.client.RestyClient().R().
		SetAuthToken(a.token.AccessToken).
		SetHeader("Content-Type", "application/json").
		SetPathParams(map[string]string{
			"realm": realmName,
		}).
		SetResult(&result).
		Get(a.basePath + getOneClientScope)
	if err := a.checkError(err, resp); err != nil {
		return nil, err
	}
	if result == nil {
		return nil, ErrNotFound(fmt.Sprintf("realm %v doesnt contain client scopes, rsp: %s", realmName, resp.String()))
	}
	scope, err := getClientScope(scopeName, result)
	if err != nil {
		return nil, err
	}
	log.Info("End get Client Scope", "scope", scope)
	return scope, err
}

func getClientScope(name string, clientScopes []*model.ClientScope) (*model.ClientScope, error) {
	for _, cs := range clientScopes {
		if *cs.Name == name {
			return cs, nil
		}
	}
	return nil, ErrNotFound(fmt.Sprintf("scope %v was not found", name))
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
				"realm":            realm,
				"clientScopeID":    scopeID,
				"protocolMapperID": *pm.ID,
			}).Delete(a.basePath + deleteClientScopeProtocolMapper)

			if err := a.checkError(err, rsp); err != nil {
				return errors.Wrap(err, "error during client scope protocol mapper deletion")
			}

		}
	}

	for _, pm := range instanceProtocolMappers {
		rsp, err := a.startRestyRequest().SetContext(ctx).SetPathParams(map[string]string{
			"realm":         realm,
			"clientScopeID": scopeID,
		}).SetBody(&pm).Post(a.basePath + createClientScopeProtocolMapper)

		if err := a.checkError(err, rsp); err != nil {
			return errors.Wrap(err, "error during client scope protocol mapper creation")
		}
	}

	return nil
}

func (a GoCloakAdapter) setDefaultClientScopeForRealm(ctx context.Context, realm, scopeID string) error {
	rsp, err := a.startRestyRequest().SetContext(ctx).SetPathParams(map[string]string{
		"realm":         realm,
		"clientScopeID": scopeID,
	}).SetBody(map[string]string{
		"realm":         realm,
		"clientScopeId": scopeID,
	}).Put(a.basePath + putDefaultClientScope)

	if err := a.checkError(err, rsp); err != nil {
		return errors.Wrap(err, "unable to set default client scope for realm")
	}

	return nil
}

func (a GoCloakAdapter) unsetDefaultClientScopeForRealm(ctx context.Context, realm, scopeID string) error {
	rsp, err := a.startRestyRequest().SetContext(ctx).SetPathParams(map[string]string{
		"realm":         realm,
		"clientScopeID": scopeID,
	}).Delete(a.basePath + deleteDefaultClientScope)

	if err := a.checkError(err, rsp); err != nil {
		return errors.Wrap(err, "unable to unset default client scope for realm")
	}

	return nil
}

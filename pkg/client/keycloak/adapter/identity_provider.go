package adapter

import (
	"context"
	"fmt"
	"net/http"
)

type IdentityProvider struct {
	ProviderID                string            `json:"providerId"`
	Config                    map[string]string `json:"config"`
	AddReadTokenRoleOnCreate  bool              `json:"addReadTokenRoleOnCreate"`
	Alias                     string            `json:"alias"`
	AuthenticateByDefault     bool              `json:"authenticateByDefault"`
	DisplayName               string            `json:"displayName"`
	Enabled                   bool              `json:"enabled"`
	FirstBrokerLoginFlowAlias string            `json:"firstBrokerLoginFlowAlias"`
	LinkOnly                  bool              `json:"linkOnly"`
	StoreToken                bool              `json:"storeToken"`
	TrustEmail                bool              `json:"trustEmail"`
}

type IdentityProviderMapper struct {
	ID                     string            `json:"id,omitempty"`
	IdentityProviderAlias  string            `json:"identityProviderAlias"`
	IdentityProviderMapper string            `json:"identityProviderMapper"`
	Name                   string            `json:"name"`
	Config                 map[string]string `json:"config"`
}

func (a GoCloakAdapter) CreateIdentityProvider(ctx context.Context, realm string, idp *IdentityProvider) error {
	rsp, err := a.startRestyRequest().
		SetContext(ctx).
		SetPathParams(map[string]string{
			keycloakApiParamRealm: realm,
		}).
		SetBody(idp).
		Post(a.buildPath(identityProviderCreateList))

	if err = a.checkError(err, rsp); err != nil {
		return fmt.Errorf("unable to create idp: %w", err)
	}

	return nil
}

func (a GoCloakAdapter) UpdateIdentityProvider(ctx context.Context, realm string, idp *IdentityProvider) error {
	rsp, err := a.startRestyRequest().
		SetContext(ctx).
		SetPathParams(map[string]string{
			keycloakApiParamRealm: realm,
			keycloakApiParamAlias: idp.Alias,
		}).
		SetBody(idp).
		Put(a.buildPath(identityProviderEntity))

	if err = a.checkError(err, rsp); err != nil {
		return fmt.Errorf("unable to update idp: %w", err)
	}

	return nil
}

func (a GoCloakAdapter) GetIdentityProvider(ctx context.Context, realm, alias string) (*IdentityProvider, error) {
	var idp IdentityProvider
	rsp, err := a.startRestyRequest().
		SetContext(ctx).
		SetPathParams(map[string]string{
			keycloakApiParamRealm: realm,
			keycloakApiParamAlias: alias,
		}).
		SetResult(&idp).
		Get(a.buildPath(identityProviderEntity))

	if err = a.checkError(err, rsp); err != nil {
		if rsp.StatusCode() == http.StatusNotFound {
			return nil, NotFoundError("idp not found")
		}

		return nil, fmt.Errorf("unable to get idp: %w", err)
	}

	return &idp, nil
}

func (a GoCloakAdapter) IdentityProviderExists(ctx context.Context, realm, alias string) (bool, error) {
	_, err := a.GetIdentityProvider(ctx, realm, alias)
	if err != nil {
		if IsErrNotFound(err) {
			return false, nil
		}

		return false, fmt.Errorf("unable to get idp, unexpected error: %w", err)
	}

	return true, nil
}

func (a GoCloakAdapter) DeleteIdentityProvider(ctx context.Context, realm, alias string) error {
	rsp, err := a.startRestyRequest().
		SetContext(ctx).
		SetPathParams(map[string]string{
			keycloakApiParamRealm: realm,
			keycloakApiParamAlias: alias,
		}).
		Delete(a.buildPath(identityProviderEntity))

	if err = a.checkError(err, rsp); err != nil {
		return fmt.Errorf("unable to delete idp: %w", err)
	}

	return nil
}

func (a GoCloakAdapter) CreateIDPMapper(ctx context.Context, realm, idpAlias string,
	mapper *IdentityProviderMapper) (string, error) {
	rsp, err := a.startRestyRequest().
		SetContext(ctx).
		SetPathParams(map[string]string{
			keycloakApiParamRealm: realm,
			keycloakApiParamAlias: idpAlias,
		}).
		SetBody(mapper).
		Post(a.buildPath(idpMapperCreateList))

	if err = a.checkError(err, rsp); err != nil {
		return "", fmt.Errorf("unable to create idp mapper: %w", err)
	}

	id, err := getIDFromResponseLocation(rsp.RawResponse)
	if err != nil {
		return "", fmt.Errorf("no id in response: %w", err)
	}

	return id, nil
}

func (a GoCloakAdapter) UpdateIDPMapper(
	ctx context.Context,
	realm,
	idpAlias string,
	mapper *IdentityProviderMapper,
) error {
	rsp, err := a.startRestyRequest().
		SetContext(ctx).
		SetPathParams(map[string]string{
			keycloakApiParamRealm: realm,
			keycloakApiParamAlias: idpAlias,
			keycloakApiParamId:    mapper.ID,
		}).
		SetBody(mapper).
		Put(a.buildPath(idpMapperEntity))

	if err = a.checkError(err, rsp); err != nil {
		return fmt.Errorf("unable to update idp mapper: %w", err)
	}

	return nil
}

func (a GoCloakAdapter) DeleteIDPMapper(ctx context.Context, realm, idpAlias, mapperID string) error {
	rsp, err := a.startRestyRequest().
		SetContext(ctx).
		SetPathParams(map[string]string{
			keycloakApiParamRealm: realm,
			keycloakApiParamAlias: idpAlias,
			keycloakApiParamId:    mapperID,
		}).
		Delete(a.buildPath(idpMapperEntity))

	if err = a.checkError(err, rsp); err != nil {
		return fmt.Errorf("unable to delete idp mapper: %w", err)
	}

	return nil
}

func (a GoCloakAdapter) GetIDPMappers(ctx context.Context, realm, idpAlias string) ([]IdentityProviderMapper, error) {
	var res []IdentityProviderMapper
	rsp, err := a.startRestyRequest().
		SetContext(ctx).
		SetPathParams(map[string]string{
			keycloakApiParamRealm: realm,
			keycloakApiParamAlias: idpAlias,
		}).
		SetResult(&res).
		Get(a.buildPath(idpMapperCreateList))

	if err = a.checkError(err, rsp); err != nil {
		return nil, fmt.Errorf("unable to get idp mappers: %w", err)
	}

	return res, nil
}

func (a GoCloakAdapter) GetIDPManagementPermissions(
	realm, idpAlias string,
) (*ManagementPermissionRepresentation, error) {
	var result ManagementPermissionRepresentation

	rsp, err := a.startRestyRequest().
		SetPathParams(map[string]string{
			keycloakApiParamRealm: realm,
			keycloakApiParamAlias: idpAlias,
		}).
		SetResult(&result).
		Get(a.buildPath(idpManagementPermissions))

	if err = a.checkError(err, rsp); err != nil {
		return nil, fmt.Errorf("unable to get idp management permissions: %w", err)
	}

	return &result, nil
}

func (a GoCloakAdapter) UpdateIDPManagementPermissions(
	realm, idpAlias string,
	permission ManagementPermissionRepresentation,
) error {
	rsp, err := a.startRestyRequest().
		SetPathParams(map[string]string{
			keycloakApiParamRealm: realm,
			keycloakApiParamAlias: idpAlias,
		}).
		SetBody(permission).
		Put(a.buildPath(idpManagementPermissions))

	if err = a.checkError(err, rsp); err != nil {
		return fmt.Errorf("unable to update idp management permissions: %w", err)
	}

	return nil
}

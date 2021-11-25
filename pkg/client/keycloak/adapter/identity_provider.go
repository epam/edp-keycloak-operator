package adapter

import (
	"context"
	"net/http"

	"github.com/pkg/errors"
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
	rsp, err := a.startRestyRequest().SetContext(ctx).SetPathParams(map[string]string{
		"realm": realm,
	}).SetBody(idp).Post(a.basePath + identityProviderCreateList)

	if err := a.checkError(err, rsp); err != nil {
		return errors.Wrap(err, "unable to create idp")
	}

	return nil
}

func (a GoCloakAdapter) UpdateIdentityProvider(ctx context.Context, realm string, idp *IdentityProvider) error {
	rsp, err := a.startRestyRequest().SetContext(ctx).SetPathParams(map[string]string{
		"realm": realm,
		"alias": idp.Alias,
	}).SetBody(idp).Put(a.basePath + identityProviderEntity)

	if err := a.checkError(err, rsp); err != nil {
		return errors.Wrap(err, "unable to update idp")
	}

	return nil
}

func (a GoCloakAdapter) GetIdentityProvider(ctx context.Context, realm, alias string) (*IdentityProvider, error) {
	var idp IdentityProvider
	rsp, err := a.startRestyRequest().SetContext(ctx).SetPathParams(map[string]string{
		"realm": realm,
		"alias": alias,
	}).SetResult(&idp).Get(a.basePath + identityProviderEntity)

	if err := a.checkError(err, rsp); err != nil {
		if rsp.StatusCode() == http.StatusNotFound {
			return nil, ErrNotFound("idp not found")
		}

		return nil, errors.Wrap(err, "unable to get idp")
	}

	return &idp, nil
}

func (a GoCloakAdapter) DeleteIdentityProvider(ctx context.Context, realm, alias string) error {
	rsp, err := a.startRestyRequest().SetContext(ctx).SetPathParams(map[string]string{
		"realm": realm,
		"alias": alias,
	}).Delete(a.basePath + identityProviderEntity)

	if err := a.checkError(err, rsp); err != nil {
		return errors.Wrap(err, "unable to delete idp")
	}

	return nil
}

func (a GoCloakAdapter) CreateIDPMapper(ctx context.Context, realm, idpAlias string,
	mapper *IdentityProviderMapper) (string, error) {
	rsp, err := a.startRestyRequest().SetContext(ctx).SetPathParams(map[string]string{
		"realm": realm,
		"alias": idpAlias,
	}).SetBody(mapper).Post(a.basePath + idpMapperCreateList)

	if err := a.checkError(err, rsp); err != nil {
		return "", errors.Wrap(err, "unable to create idp mapper")
	}

	id, err := getIDFromResponseLocation(rsp.RawResponse)
	if err != nil {
		return "", errors.Wrap(err, "no id in response")
	}

	return id, nil
}

func (a GoCloakAdapter) UpdateIDPMapper(ctx context.Context, realm, idpAlias string, mapper *IdentityProviderMapper) error {
	rsp, err := a.startRestyRequest().SetContext(ctx).SetPathParams(map[string]string{
		"realm": realm,
		"alias": idpAlias,
		"id":    mapper.ID,
	}).SetBody(mapper).Put(a.basePath + idpMapperEntity)

	if err := a.checkError(err, rsp); err != nil {
		return errors.Wrap(err, "unable to update idp mapper")
	}

	return nil
}

func (a GoCloakAdapter) DeleteIDPMapper(ctx context.Context, realm, idpAlias, mapperID string) error {
	rsp, err := a.startRestyRequest().SetContext(ctx).SetPathParams(map[string]string{
		"realm": realm,
		"alias": idpAlias,
		"id":    mapperID,
	}).Delete(a.basePath + idpMapperEntity)

	if err := a.checkError(err, rsp); err != nil {
		return errors.Wrap(err, "unable to delete idp mapper")
	}

	return nil
}

func (a GoCloakAdapter) GetIDPMappers(ctx context.Context, realm, idpAlias string) ([]IdentityProviderMapper, error) {
	var res []IdentityProviderMapper
	rsp, err := a.startRestyRequest().SetContext(ctx).SetPathParams(map[string]string{
		"realm": realm,
		"alias": idpAlias,
	}).SetResult(&res).Get(a.basePath + idpMapperCreateList)

	if err := a.checkError(err, rsp); err != nil {
		return nil, errors.Wrap(err, "unable to get idp mappers")
	}

	return res, nil
}

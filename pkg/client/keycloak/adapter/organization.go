package adapter

import (
	"context"
	"fmt"
	"net/http"

	"github.com/epam/edp-keycloak-operator/pkg/client/keycloak/dto"
)

// GetOrganizationsParams represents parameters for getting organizations.
type GetOrganizationsParams struct {
	Alias *string
}

const (
	// Organization API endpoints.
	organizationsResource    = "/admin/realms/{realm}/organizations"
	organizationEntity       = "/admin/realms/{realm}/organizations/{id}"
	organizationIdPsResource = "/admin/realms/{realm}/organizations/{id}/identity-providers"
	organizationIdPEntity    = "/admin/realms/{realm}/organizations/{id}/identity-providers/{alias}"
)

// CreateOrganization creates a new organization in the specified realm.
func (a GoCloakAdapter) CreateOrganization(ctx context.Context, realm string, org *dto.Organization) error {
	rsp, err := a.startRestyRequest().
		SetContext(ctx).
		SetPathParams(map[string]string{
			keycloakApiParamRealm: realm,
		}).
		SetBody(org).
		Post(a.buildPath(organizationsResource))

	if err = a.checkError(err, rsp); err != nil {
		return fmt.Errorf("unable to create organization: %w", err)
	}

	return nil
}

// UpdateOrganization updates an existing organization in the specified realm.
func (a GoCloakAdapter) UpdateOrganization(ctx context.Context, realm string, org *dto.Organization) error {
	if org.ID == "" {
		return fmt.Errorf("organization ID is required for update")
	}

	rsp, err := a.startRestyRequest().
		SetContext(ctx).
		SetPathParams(map[string]string{
			keycloakApiParamRealm: realm,
			keycloakApiParamId:    org.ID,
		}).
		SetBody(org).
		Put(a.buildPath(organizationEntity))

	if err = a.checkError(err, rsp); err != nil {
		return fmt.Errorf("unable to update organization: %w", err)
	}

	return nil
}

// GetOrganization retrieves an organization by ID from the specified realm.
func (a GoCloakAdapter) GetOrganization(ctx context.Context, realm, orgID string) (*dto.Organization, error) {
	var org dto.Organization
	rsp, err := a.startRestyRequest().
		SetContext(ctx).
		SetPathParams(map[string]string{
			keycloakApiParamRealm: realm,
			keycloakApiParamId:    orgID,
		}).
		SetResult(&org).
		Get(a.buildPath(organizationEntity))

	if err = a.checkError(err, rsp); err != nil {
		if rsp.StatusCode() == http.StatusNotFound {
			return nil, NotFoundError("organization not found")
		}

		return nil, fmt.Errorf("unable to get organization: %w", err)
	}

	return &org, nil
}

// DeleteOrganization deletes an organization by ID from the specified realm.
func (a GoCloakAdapter) DeleteOrganization(ctx context.Context, realm, orgID string) error {
	rsp, err := a.startRestyRequest().
		SetContext(ctx).
		SetPathParams(map[string]string{
			keycloakApiParamRealm: realm,
			keycloakApiParamId:    orgID,
		}).
		Delete(a.buildPath(organizationEntity))

	if err = a.checkError(err, rsp); err != nil {
		return fmt.Errorf("unable to delete organization: %w", err)
	}

	return nil
}

// GetOrganizations retrieves organizations from the specified realm.
func (a GoCloakAdapter) GetOrganizations(ctx context.Context, realm string, params *GetOrganizationsParams) ([]dto.Organization, error) {
	var orgs []dto.Organization
	req := a.startRestyRequest().
		SetContext(ctx).
		SetPathParams(map[string]string{
			keycloakApiParamRealm: realm,
		}).
		SetResult(&orgs)

	// Add alias query parameter if provided
	if params != nil && params.Alias != nil && *params.Alias != "" {
		req = req.SetQueryParam("q", fmt.Sprintf("alias:%s", *params.Alias))
	}

	rsp, err := req.Get(a.buildPath(organizationsResource))

	if err = a.checkError(err, rsp); err != nil {
		return nil, fmt.Errorf("unable to get organizations: %w", err)
	}

	return orgs, nil
}

// LinkIdentityProviderToOrganization links an identity provider to an organization.
func (a GoCloakAdapter) LinkIdentityProviderToOrganization(ctx context.Context, realm, orgID, idpAlias string) error {
	rsp, err := a.startRestyRequest().
		SetContext(ctx).
		SetPathParams(map[string]string{
			keycloakApiParamRealm: realm,
			keycloakApiParamId:    orgID,
		}).
		SetBody(idpAlias).
		Post(a.buildPath(organizationIdPsResource))

	if err = a.checkError(err, rsp); err != nil {
		return fmt.Errorf("unable to link identity provider to organization: %w", err)
	}

	return nil
}

// UnlinkIdentityProviderFromOrganization unlinks an identity provider from an organization.
func (a GoCloakAdapter) UnlinkIdentityProviderFromOrganization(ctx context.Context, realm, orgID, idpAlias string) error {
	rsp, err := a.startRestyRequest().
		SetContext(ctx).
		SetPathParams(map[string]string{
			keycloakApiParamRealm: realm,
			keycloakApiParamId:    orgID,
			keycloakApiParamAlias: idpAlias,
		}).
		Delete(a.buildPath(organizationIdPEntity))

	if err = a.checkError(err, rsp); err != nil {
		return fmt.Errorf("unable to unlink identity provider from organization: %w", err)
	}

	return nil
}

// GetOrganizationIdentityProviders retrieves all identity providers linked to an organization.
func (a GoCloakAdapter) GetOrganizationIdentityProviders(ctx context.Context, realm, orgID string) ([]dto.OrganizationIdentityProvider, error) {
	var idps []dto.OrganizationIdentityProvider
	rsp, err := a.startRestyRequest().
		SetContext(ctx).
		SetPathParams(map[string]string{
			keycloakApiParamRealm: realm,
			keycloakApiParamId:    orgID,
		}).
		SetResult(&idps).
		Get(a.buildPath(organizationIdPsResource))

	if err = a.checkError(err, rsp); err != nil {
		return nil, fmt.Errorf("unable to get organization identity providers: %w", err)
	}

	return idps, nil
}

// OrganizationExists checks if an organization exists in the specified realm.
func (a GoCloakAdapter) OrganizationExists(ctx context.Context, realm, orgID string) (bool, error) {
	_, err := a.GetOrganization(ctx, realm, orgID)
	if err != nil {
		if IsErrNotFound(err) {
			return false, nil
		}

		return false, fmt.Errorf("unable to get organization, unexpected error: %w", err)
	}

	return true, nil
}

// GetOrganizationByAlias retrieves an organization by alias from the specified realm.
func (a GoCloakAdapter) GetOrganizationByAlias(ctx context.Context, realm, alias string) (*dto.Organization, error) {
	// Get organizations filtered by alias
	orgs, err := a.GetOrganizations(ctx, realm, &GetOrganizationsParams{
		Alias: &alias,
	})
	if err != nil {
		return nil, fmt.Errorf("unable to get organization by alias %s: %w", alias, err)
	}

	// Find the organization by exact alias match
	for _, org := range orgs {
		if org.Alias == alias {
			return &org, nil
		}
	}

	return nil, NotFoundError("organization with alias %s not found")
}

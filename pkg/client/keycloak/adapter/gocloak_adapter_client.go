package adapter

import (
	"context"
	"fmt"

	"github.com/Nerzal/gocloak/v12"
	"github.com/pkg/errors"
)

const defaultMax = 100

func (a GoCloakAdapter) AddDefaultScopeToClient(ctx context.Context, realmName, clientName string, scopes []ClientScope) error {
	log := a.log.WithValues("clientName", clientName, logKeyRealm, realmName)
	log.Info("Start add Default Client Scopes to client...")

	clientID, err := a.GetClientID(clientName, realmName)
	if err != nil {
		return errors.Wrap(err, "error during GetClientId")
	}

	existingScopes, err := a.client.GetClientsDefaultScopes(ctx, a.token.AccessToken, realmName, clientID)
	if err != nil {
		return errors.Wrap(err, fmt.Sprintf("failed to get existing client scope for client %s", clientName))
	}

	existingScopesMap := make(map[string]*gocloak.ClientScope)

	for _, s := range existingScopes {
		if s != nil {
			existingScopesMap[*s.ID] = s
		}
	}

	for _, scope := range scopes {
		if _, ok := existingScopesMap[scope.ID]; ok {
			continue
		}

		err := a.client.AddDefaultScopeToClient(ctx, a.token.AccessToken, realmName, clientID, scope.ID)
		if err != nil {
			a.log.Error(err, fmt.Sprintf("failed link scope %s to client %s", scope.Name, clientName))
		}
	}

	log.Info("End add Default Client Scopes to client...")

	return nil
}

func (a GoCloakAdapter) AddOptionalScopeToClient(ctx context.Context, realmName, clientName string, scopes []ClientScope) error {
	log := a.log.WithValues("clientName", clientName, logKeyRealm, realmName)
	log.Info("Start add Optional Client Scopes to client...")

	clientID, err := a.GetClientID(clientName, realmName)
	if err != nil {
		return errors.Wrap(err, "error during GetClientId")
	}

	existingScopes, err := a.client.GetClientsOptionalScopes(ctx, a.token.AccessToken, realmName, clientID)
	if err != nil {
		return errors.Wrap(err, fmt.Sprintf("failed to get existing client scope for client %s", clientName))
	}

	existingScopesMap := make(map[string]*gocloak.ClientScope)

	for _, s := range existingScopes {
		if s != nil {
			existingScopesMap[*s.ID] = s
		}
	}

	for _, scope := range scopes {
		if _, ok := existingScopesMap[scope.ID]; ok {
			continue
		}

		err := a.client.AddOptionalScopeToClient(ctx, a.token.AccessToken, realmName, clientID, scope.ID)
		if err != nil {
			a.log.Error(err, fmt.Sprintf("failed link scope %s to client %s", scope.Name, clientName))
		}
	}

	log.Info("End add Optional Client Scopes to client...")

	return nil
}

func (a GoCloakAdapter) GetPermissions(ctx context.Context, realm, idOfClient string) (map[string]gocloak.PermissionRepresentation, error) {
	params := gocloak.GetPermissionParams{
		Max: gocloak.IntP(defaultMax),
	}

	p, err := a.client.GetPermissions(ctx, a.token.AccessToken, realm, idOfClient, params)
	if err != nil {
		return nil, fmt.Errorf("failed to get permissions: %w", err)
	}

	permissions := make(map[string]gocloak.PermissionRepresentation, len(p))

	for _, permission := range p {
		if permission == nil || permission.Name == nil {
			continue
		}

		permissions[*permission.Name] = *permission
	}

	return permissions, nil
}

func (a GoCloakAdapter) GetScopes(ctx context.Context, realm, idOfClient string) (map[string]gocloak.ScopeRepresentation, error) {
	params := gocloak.GetScopeParams{
		Max:  gocloak.IntP(defaultMax),
		Deep: gocloak.BoolP(false),
	}

	s, err := a.client.GetScopes(ctx, a.token.AccessToken, realm, idOfClient, params)
	if err != nil {
		return nil, fmt.Errorf("failed to get scopes: %w", err)
	}

	scopes := make(map[string]gocloak.ScopeRepresentation, len(s))

	for _, scope := range s {
		if scope == nil || scope.Name == nil {
			continue
		}

		scopes[*scope.Name] = *scope
	}

	return scopes, nil
}

func (a GoCloakAdapter) GetResources(ctx context.Context, realm, idOfClient string) (map[string]gocloak.ResourceRepresentation, error) {
	params := gocloak.GetResourceParams{
		Max:  gocloak.IntP(defaultMax),
		Deep: gocloak.BoolP(false),
	}

	r, err := a.client.GetResources(ctx, a.token.AccessToken, realm, idOfClient, params)
	if err != nil {
		return nil, fmt.Errorf("failed to get resources: %w", err)
	}

	resources := make(map[string]gocloak.ResourceRepresentation, len(r))

	for _, resource := range r {
		if resource == nil || resource.Name == nil {
			continue
		}

		resources[*resource.Name] = *resource
	}

	return resources, nil
}

// CreateScope creates a client authorization permission.
// nolint:gocritic // gocloak is a third party library, we can't change the function signature
func (a GoCloakAdapter) CreateScope(ctx context.Context, realm, idOfClient string, scope string) (*gocloak.ScopeRepresentation, error) {
	scopeRepresentation := gocloak.ScopeRepresentation{
		Name: &scope,
	}
	p, err := a.client.CreateScope(ctx, a.token.AccessToken, realm, idOfClient, scopeRepresentation)

	if err != nil {
		return nil, fmt.Errorf("failed to create scope: %w", err)
	}

	return p, nil
}

func (a GoCloakAdapter) DeleteScope(ctx context.Context, realm, idOfClient, scope string) error {
	if err := a.client.DeleteScope(ctx, a.token.AccessToken, realm, idOfClient, scope); err != nil {
		return fmt.Errorf("failed to delete scope: %w", err)
	}

	return nil
}

// CreatePermission creates a client authorization permission.
// nolint:gocritic // gocloak is a third party library, we can't change the function signature
func (a GoCloakAdapter) CreatePermission(ctx context.Context, realm, idOfClient string, permission gocloak.PermissionRepresentation) (*gocloak.PermissionRepresentation, error) {
	p, err := a.client.CreatePermission(ctx, a.token.AccessToken, realm, idOfClient, permission)
	if err != nil {
		return nil, fmt.Errorf("failed to create permission: %w", err)
	}

	return p, nil
}

// UpdatePermission updates a client authorization permission.
// nolint:gocritic // gocloak is a third party library, we can't change the function signature
func (a GoCloakAdapter) UpdatePermission(ctx context.Context, realm, idOfClient string, permission gocloak.PermissionRepresentation) error {
	if err := a.client.UpdatePermission(ctx, a.token.AccessToken, realm, idOfClient, permission); err != nil {
		return fmt.Errorf("failed to update permission: %w", err)
	}

	return nil
}

func (a GoCloakAdapter) DeletePermission(ctx context.Context, realm, idOfClient, permissionID string) error {
	if err := a.client.DeletePermission(ctx, a.token.AccessToken, realm, idOfClient, permissionID); err != nil {
		return fmt.Errorf("failed to delete permission: %w", err)
	}

	return nil
}

func (a GoCloakAdapter) GetPolicies(ctx context.Context, realm, idOfClient string) (map[string]*gocloak.PolicyRepresentation, error) {
	params := gocloak.GetPolicyParams{
		Permission: gocloak.BoolP(false),
		Max:        gocloak.IntP(defaultMax),
	}

	p, err := a.client.GetPolicies(ctx, a.token.AccessToken, realm, idOfClient, params)
	if err != nil {
		return nil, fmt.Errorf("failed to get policies: %w", err)
	}

	policies := make(map[string]*gocloak.PolicyRepresentation, len(p))

	for _, policy := range p {
		if policy.Name == nil {
			continue
		}

		policies[*policy.Name] = policy
	}

	return policies, nil
}

// CreatePolicy creates a client authorization policy.
// nolint:gocritic // gocloak is a third party library, we can't change the function signature
func (a GoCloakAdapter) CreatePolicy(ctx context.Context, realm, idOfClient string, policy gocloak.PolicyRepresentation) (*gocloak.PolicyRepresentation, error) {
	pl, err := a.client.CreatePolicy(ctx, a.token.AccessToken, realm, idOfClient, policy)
	if err != nil {
		return nil, fmt.Errorf("failed to create policy: %w", err)
	}

	return pl, nil
}

// UpdatePolicy updates a client authorization policy.
// nolint:gocritic // gocloak is a third party library, we can't change the function signature
func (a GoCloakAdapter) UpdatePolicy(ctx context.Context, realm, idOfClient string, policy gocloak.PolicyRepresentation) error {
	if err := a.client.UpdatePolicy(ctx, a.token.AccessToken, realm, idOfClient, policy); err != nil {
		return fmt.Errorf("failed to update policy: %w", err)
	}

	return nil
}

func (a GoCloakAdapter) DeletePolicy(ctx context.Context, realm, idOfClient, policyID string) error {
	if err := a.client.DeletePolicy(ctx, a.token.AccessToken, realm, idOfClient, policyID); err != nil {
		return fmt.Errorf("failed to delete policy: %w", err)
	}

	return nil
}

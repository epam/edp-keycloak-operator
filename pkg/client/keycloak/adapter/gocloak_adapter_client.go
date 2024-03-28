package adapter

import (
	"context"
	"fmt"

	"github.com/Nerzal/gocloak/v12"
	"github.com/pkg/errors"
)

func (a GoCloakAdapter) AddDefaultScopeToClient(ctx context.Context, realmName, clientName string, scopes []ClientScope) error {
	log := a.log.WithValues("clientName", clientName, logKeyRealm, realmName)
	log.Info("Start add Client Scopes to client...")

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

	log.Info("End add Client Scopes to client...")

	return nil
}

func (a GoCloakAdapter) GetPolicies(ctx context.Context, realm, idOfClient string) (map[string]*gocloak.PolicyRepresentation, error) {
	params := gocloak.GetPolicyParams{
		Permission: gocloak.BoolP(false),
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

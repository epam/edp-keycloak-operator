package chain

import (
	"context"
	"fmt"
	"strings"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	keycloakApi "github.com/epam/edp-keycloak-operator/api/v1"
	"github.com/epam/edp-keycloak-operator/pkg/client/keycloakapi"
)

type ScopeMappings struct {
	kClient   *keycloakapi.KeycloakClient
	k8sClient client.Client
}

func NewScopeMappings(kClient *keycloakapi.KeycloakClient, k8sClient client.Client) *ScopeMappings {
	return &ScopeMappings{kClient: kClient, k8sClient: k8sClient}
}

func (h *ScopeMappings) Serve(
	ctx context.Context,
	keycloakClient *keycloakApi.KeycloakClient,
	realmName string,
	clientCtx *ClientContext,
) error {
	sm := keycloakClient.Spec.ScopeMappings
	if sm == nil {
		return nil
	}

	if err := validateScopeMappings(sm); err != nil {
		h.setCondition(ctx, keycloakClient, metav1.ConditionFalse, ReasonConfigurationError,
			fmt.Sprintf("Invalid scope mappings: %s", err.Error()))

		return err
	}

	addOnly := keycloakClient.GetReconciliationStrategy() == keycloakApi.ReconciliationStrategyAddOnly

	if err := h.sync(ctx, realmName, clientCtx.ClientUUID, sm, addOnly); err != nil {
		h.setCondition(ctx, keycloakClient, metav1.ConditionFalse, ReasonKeycloakAPIError,
			fmt.Sprintf("Failed to sync scope mappings: %s", err.Error()))

		return fmt.Errorf("unable to sync scope mappings: %w", err)
	}

	h.setCondition(ctx, keycloakClient, metav1.ConditionTrue, ReasonScopeMappingsSynced, "Scope mappings synchronized")

	return nil
}

func (h *ScopeMappings) setCondition(
	ctx context.Context,
	keycloakClient *keycloakApi.KeycloakClient,
	status metav1.ConditionStatus,
	reason, message string,
) {
	if err := SetCondition(
		ctx, h.k8sClient, keycloakClient, ConditionScopeMappingsSynced, status, reason, message,
	); err != nil {
		ctrl.LoggerFrom(ctx).Error(err, "Failed to set scope mappings condition")
	}
}

func validateScopeMappings(sm *keycloakApi.ScopeMappings) error {
	if sm.RealmRoles != nil {
		for i, name := range *sm.RealmRoles {
			if name == "" {
				return fmt.Errorf("scopeMappings.realmRoles[%d]: role name must not be empty", i)
			}
		}
	}

	seen := make(map[string]struct{}, len(sm.ClientRoles))

	for i, entry := range sm.ClientRoles {
		if strings.TrimSpace(entry.ClientID) == "" {
			return fmt.Errorf("scopeMappings.clientRoles[%d]: clientId must not be empty", i)
		}

		if _, dup := seen[entry.ClientID]; dup {
			return fmt.Errorf("scopeMappings.clientRoles[%d]: duplicate clientId %q", i, entry.ClientID)
		}

		seen[entry.ClientID] = struct{}{}

		if entry.Roles == nil {
			continue
		}

		for j, name := range *entry.Roles {
			if name == "" {
				return fmt.Errorf("scopeMappings.clientRoles[%d].roles[%d]: role name must not be empty", i, j)
			}
		}
	}

	return nil
}

func (h *ScopeMappings) sync(
	ctx context.Context,
	realmName, clientUUID string,
	sm *keycloakApi.ScopeMappings,
	addOnly bool,
) error {
	if sm.RealmRoles != nil {
		if err := h.syncRealmRoles(ctx, realmName, clientUUID, *sm.RealmRoles, addOnly); err != nil {
			return err
		}
	}

	for _, entry := range sm.ClientRoles {
		if entry.Roles == nil {
			continue
		}

		if err := h.syncClientRoles(ctx, realmName, clientUUID, entry.ClientID, *entry.Roles, addOnly); err != nil {
			return err
		}
	}

	return nil
}

func (h *ScopeMappings) syncRealmRoles(
	ctx context.Context,
	realmName, clientUUID string,
	desired []string,
	addOnly bool,
) error {
	current, _, err := h.kClient.Clients.GetRealmScopeMappings(ctx, realmName, clientUUID)
	if err != nil {
		return fmt.Errorf("unable to get realm scope mappings: %w", err)
	}

	currentByName := roleMapByName(current)
	desiredSet := make(map[string]struct{}, len(desired))

	var toAdd []keycloakapi.RoleRepresentation

	for _, name := range desired {
		if _, queued := desiredSet[name]; queued {
			continue
		}

		desiredSet[name] = struct{}{}

		if _, mapped := currentByName[name]; mapped {
			continue
		}

		role, _, err := h.kClient.Roles.GetRealmRole(ctx, realmName, name)
		if err != nil {
			if keycloakapi.IsNotFound(err) {
				return fmt.Errorf("realm role %q not found in realm %q; create it with a KeycloakRealmRole: %w", name, realmName, err)
			}

			return fmt.Errorf("unable to get realm role %q: %w", name, err)
		}

		mappable, err := keycloakapi.RequireRealmRoleWithID(role, realmName, name)
		if err != nil {
			return err
		}

		toAdd = append(toAdd, mappable)
	}

	if len(toAdd) > 0 {
		if _, err := h.kClient.Clients.AddRealmScopeMappings(ctx, realmName, clientUUID, toAdd); err != nil {
			return fmt.Errorf("unable to add realm scope mappings: %w", err)
		}
	}

	if addOnly {
		return nil
	}

	if toRemove := rolesNotIn(current, desiredSet); len(toRemove) > 0 {
		if _, err := h.kClient.Clients.DeleteRealmScopeMappings(ctx, realmName, clientUUID, toRemove); err != nil {
			return fmt.Errorf("unable to delete realm scope mappings: %w", err)
		}
	}

	return nil
}

func (h *ScopeMappings) syncClientRoles(
	ctx context.Context,
	realmName, clientUUID, roleClientID string,
	desired []string,
	addOnly bool,
) error {
	roleClient, _, err := h.kClient.Clients.GetClientByClientID(ctx, realmName, roleClientID)
	if err != nil {
		if keycloakapi.IsNotFound(err) {
			return fmt.Errorf("client %q not found in realm %q; create it with a KeycloakClient: %w", roleClientID, realmName, err)
		}

		return fmt.Errorf("unable to get client %q: %w", roleClientID, err)
	}

	if roleClient == nil || roleClient.Id == nil || roleClient.ClientId == nil || *roleClient.ClientId != roleClientID {
		return fmt.Errorf("client %q not found in realm %q; create it with a KeycloakClient", roleClientID, realmName)
	}

	roleClientUUID := *roleClient.Id

	current, _, err := h.kClient.Clients.GetClientScopeMappings(ctx, realmName, clientUUID, roleClientUUID)
	if err != nil {
		return fmt.Errorf("unable to get scope mappings of client %q: %w", roleClientID, err)
	}

	currentByName := roleMapByName(current)
	desiredSet := make(map[string]struct{}, len(desired))

	var toAdd []keycloakapi.RoleRepresentation

	for _, name := range desired {
		if _, queued := desiredSet[name]; queued {
			continue
		}

		desiredSet[name] = struct{}{}

		if _, mapped := currentByName[name]; mapped {
			continue
		}

		role, _, err := h.kClient.Clients.GetClientRole(ctx, realmName, roleClientUUID, name)
		if err != nil {
			if keycloakapi.IsNotFound(err) {
				return fmt.Errorf(
					"client role %q not found for client %q; define it in clientRolesV2 of the KeycloakClient for that client: %w",
					name, roleClientID, err,
				)
			}

			return fmt.Errorf("unable to get client role %q of client %q: %w", name, roleClientID, err)
		}

		mappable, err := keycloakapi.RequireClientRoleWithID(role, roleClientID, name)
		if err != nil {
			return err
		}

		toAdd = append(toAdd, mappable)
	}

	if len(toAdd) > 0 {
		if _, err := h.kClient.Clients.AddClientScopeMappings(ctx, realmName, clientUUID, roleClientUUID, toAdd); err != nil {
			return fmt.Errorf("unable to add scope mappings of client %q: %w", roleClientID, err)
		}
	}

	if addOnly {
		return nil
	}

	if toRemove := rolesNotIn(current, desiredSet); len(toRemove) > 0 {
		if _, err := h.kClient.Clients.DeleteClientScopeMappings(ctx, realmName, clientUUID, roleClientUUID, toRemove); err != nil {
			return fmt.Errorf("unable to delete scope mappings of client %q: %w", roleClientID, err)
		}
	}

	return nil
}

func roleMapByName(roles []keycloakapi.RoleRepresentation) map[string]keycloakapi.RoleRepresentation {
	out := make(map[string]keycloakapi.RoleRepresentation, len(roles))

	for _, r := range roles {
		if r.Name != nil {
			out[*r.Name] = r
		}
	}

	return out
}

func rolesNotIn(current []keycloakapi.RoleRepresentation, desired map[string]struct{}) []keycloakapi.RoleRepresentation {
	var out []keycloakapi.RoleRepresentation

	for _, r := range current {
		if r.Name == nil {
			continue
		}

		if _, ok := desired[*r.Name]; !ok {
			out = append(out, r)
		}
	}

	return out
}

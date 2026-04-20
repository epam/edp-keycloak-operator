package chain

import (
	"context"
	"fmt"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	keycloakApi "github.com/epam/edp-keycloak-operator/api/v1"
	"github.com/epam/edp-keycloak-operator/pkg/client/keycloakapi"
	"github.com/epam/edp-keycloak-operator/pkg/maputil"
)

type PutClientRole struct {
	kClient   *keycloakapi.KeycloakClient
	k8sClient client.Client
}

func NewPutClientRole(kClient *keycloakapi.KeycloakClient, k8sClient client.Client) *PutClientRole {
	return &PutClientRole{kClient: kClient, k8sClient: k8sClient}
}

func (h *PutClientRole) Serve(ctx context.Context, keycloakClient *keycloakApi.KeycloakClient, realmName string, clientCtx *ClientContext) error {
	if err := h.putKeycloakClientRole(ctx, keycloakClient, realmName, clientCtx.ClientUUID); err != nil {
		h.setFailureCondition(ctx, keycloakClient, fmt.Sprintf("Failed to sync client roles: %s", err.Error()))

		return fmt.Errorf("unable to put keycloak client role: %w", err)
	}

	h.setSuccessCondition(ctx, keycloakClient, "Client roles synchronized")

	return nil
}

func (h *PutClientRole) setFailureCondition(ctx context.Context, keycloakClient *keycloakApi.KeycloakClient, message string) {
	log := ctrl.LoggerFrom(ctx)

	if err := SetCondition(
		ctx, h.k8sClient, keycloakClient,
		ConditionClientRolesSynced,
		metav1.ConditionFalse,
		ReasonKeycloakAPIError,
		message,
	); err != nil {
		log.Error(err, "Failed to set failure condition")
	}
}

func (h *PutClientRole) setSuccessCondition(ctx context.Context, keycloakClient *keycloakApi.KeycloakClient, message string) {
	log := ctrl.LoggerFrom(ctx)

	if err := SetCondition(
		ctx, h.k8sClient, keycloakClient,
		ConditionClientRolesSynced,
		metav1.ConditionTrue,
		ReasonClientRolesSynced,
		message,
	); err != nil {
		log.Error(err, "Failed to set success condition")
	}
}

func (h *PutClientRole) putKeycloakClientRole(ctx context.Context, keycloakClient *keycloakApi.KeycloakClient, realmName, clientUUID string) error {
	reqLog := ctrl.LoggerFrom(ctx)
	reqLog.Info("Start put keycloak client role")

	if len(keycloakClient.Spec.ClientRolesV2) == 0 {
		reqLog.Info("No client roles to sync")
		return nil
	}

	// Get existing roles
	existingRoles, _, err := h.kClient.Clients.GetClientRoles(ctx, realmName, clientUUID, nil)
	if err != nil {
		return fmt.Errorf("unable to get existing client roles: %w", err)
	}

	existingRoleMap := maputil.SliceToMapSelf(existingRoles, func(r keycloakapi.RoleRepresentation) (string, bool) {
		return *r.Name, r.Name != nil
	})

	// Build desired roles from spec
	desiredRoles := maputil.SliceToMapSelf(keycloakClient.Spec.ClientRolesV2, func(role keycloakApi.ClientRole) (string, bool) {
		return role.Name, role.Name != ""
	})

	addOnly := keycloakClient.GetReconciliationStrategy() == keycloakApi.ReconciliationStrategyAddOnly

	// Create missing roles or update existing ones
	for name, role := range desiredRoles {
		roleRep := keycloakapi.RoleRepresentation{
			Name:        ptr.To(name),
			Description: ptr.To(role.Description),
		}

		if existing, exists := existingRoleMap[name]; !exists {
			if _, err := h.kClient.Clients.CreateClientRole(ctx, realmName, clientUUID, roleRep); err != nil {
				return fmt.Errorf("unable to create client role %s: %w", name, err)
			}

			reqLog.Info("Client role created", "role", name)
		} else if ptr.Deref(existing.Description, "") != role.Description {
			roleRep.Id = existing.Id
			if _, err := h.kClient.Clients.UpdateClientRole(ctx, realmName, clientUUID, name, roleRep); err != nil {
				return fmt.Errorf("unable to update client role %s: %w", name, err)
			}

			reqLog.Info("Client role updated", "role", name)
		}
	}

	// Delete removed roles (unless addOnly)
	if !addOnly {
		for name := range existingRoleMap {
			if _, desired := desiredRoles[name]; !desired {
				if _, err := h.kClient.Clients.DeleteClientRole(ctx, realmName, clientUUID, name); err != nil {
					if !keycloakapi.IsNotFound(err) {
						return fmt.Errorf("unable to delete client role %s: %w", name, err)
					}
				}

				reqLog.Info("Client role deleted", "role", name)
			}
		}
	}

	// Sync composites for roles that have associated client roles
	for _, role := range keycloakClient.Spec.ClientRolesV2 {
		if role.Name == "" || len(role.AssociatedClientRoles) == 0 {
			continue
		}

		if err := h.syncRoleComposites(ctx, realmName, clientUUID, role); err != nil {
			return fmt.Errorf("unable to sync composites for role %s: %w", role.Name, err)
		}
	}

	reqLog.Info("End put keycloak client role")

	return nil
}

func (h *PutClientRole) syncRoleComposites(
	ctx context.Context,
	realmName, clientUUID string,
	role keycloakApi.ClientRole,
) error {
	// Get existing composites
	existingComposites, _, err := h.kClient.Clients.GetClientRoleComposites(ctx, realmName, clientUUID, role.Name)
	if err != nil {
		return fmt.Errorf("unable to get composites for role %s: %w", role.Name, err)
	}

	existingCompositeMap := maputil.SliceToMapSelf(existingComposites, func(c keycloakapi.RoleRepresentation) (string, bool) {
		return *c.Name, c.Name != nil
	})

	desiredComposites := make(map[string]bool, len(role.AssociatedClientRoles))
	for _, name := range role.AssociatedClientRoles {
		desiredComposites[name] = true
	}

	// Add missing composites
	var toAdd []keycloakapi.RoleRepresentation

	for _, compositeName := range role.AssociatedClientRoles {
		if _, exists := existingCompositeMap[compositeName]; !exists {
			compositeRole, _, err := h.kClient.Clients.GetClientRole(ctx, realmName, clientUUID, compositeName)
			if err != nil {
				return fmt.Errorf("unable to get client role %s for composite: %w", compositeName, err)
			}

			toAdd = append(toAdd, *compositeRole)
		}
	}

	if len(toAdd) > 0 {
		if _, err := h.kClient.Clients.AddClientRoleComposites(ctx, realmName, clientUUID, role.Name, toAdd); err != nil {
			return fmt.Errorf("unable to add composites to role %s: %w", role.Name, err)
		}
	}

	// Remove extra composites
	var toRemove []keycloakapi.RoleRepresentation

	for name, compositeRole := range existingCompositeMap {
		if !desiredComposites[name] {
			toRemove = append(toRemove, compositeRole)
		}
	}

	if len(toRemove) > 0 {
		if _, err := h.kClient.Clients.DeleteClientRoleComposites(ctx, realmName, clientUUID, role.Name, toRemove); err != nil {
			return fmt.Errorf("unable to delete composites from role %s: %w", role.Name, err)
		}
	}

	return nil
}

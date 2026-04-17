package chain

import (
	"context"
	"errors"
	"fmt"
	"maps"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	keycloakApi "github.com/epam/edp-keycloak-operator/api/v1"
	keycloakapi "github.com/epam/edp-keycloak-operator/pkg/client/keycloakapi"
	"github.com/epam/edp-keycloak-operator/pkg/maputil"
)

type ServiceAccount struct {
	kClient   *keycloakapi.APIClient
	k8sClient client.Client
}

func NewServiceAccount(kClient *keycloakapi.APIClient, k8sClient client.Client) *ServiceAccount {
	return &ServiceAccount{kClient: kClient, k8sClient: k8sClient}
}

func (h *ServiceAccount) Serve(ctx context.Context, keycloakClient *keycloakApi.KeycloakClient, realmName string, clientCtx *ClientContext) error {
	if keycloakClient.Spec.ServiceAccount == nil || !keycloakClient.Spec.ServiceAccount.Enabled {
		return nil
	}

	if keycloakClient.Spec.ServiceAccount != nil && keycloakClient.Spec.Public {
		return errors.New("service account can not be configured with public client")
	}

	addOnly := keycloakClient.GetReconciliationStrategy() == keycloakApi.ReconciliationStrategyAddOnly

	clientUUID := clientCtx.ClientUUID

	// Get service account user
	saUser, _, err := h.kClient.Clients.GetServiceAccountUser(ctx, realmName, clientUUID)
	if err != nil {
		h.setFailureCondition(ctx, keycloakClient, fmt.Sprintf("Failed to sync service account: %s", err.Error()))
		return fmt.Errorf("unable to get service account user: %w", err)
	}

	if saUser == nil || saUser.Id == nil {
		h.setFailureCondition(ctx, keycloakClient, "Failed to sync service account: service account user not found")
		return fmt.Errorf("service account user for client %s not found", keycloakClient.Spec.ClientId)
	}

	saUserID := *saUser.Id

	// Sync realm roles
	if err := h.syncRealmRoles(ctx, realmName, saUserID, keycloakClient.Spec.ServiceAccount.RealmRoles, addOnly); err != nil {
		h.setFailureCondition(ctx, keycloakClient, fmt.Sprintf("Failed to sync service account: %s", err.Error()))
		return fmt.Errorf("unable to sync service account realm roles: %w", err)
	}

	// Sync client roles
	clientRoles := make(map[string][]string)
	for _, v := range keycloakClient.Spec.ServiceAccount.ClientRoles {
		clientRoles[v.ClientID] = v.Roles
	}

	if err := h.syncClientRoles(ctx, realmName, saUserID, clientRoles, addOnly); err != nil {
		h.setFailureCondition(ctx, keycloakClient, fmt.Sprintf("Failed to sync service account: %s", err.Error()))
		return fmt.Errorf("unable to sync service account client roles: %w", err)
	}

	// Sync groups
	if keycloakClient.Spec.ServiceAccount.Groups != nil {
		if err := h.syncGroups(ctx, realmName, saUserID, keycloakClient.Spec.ServiceAccount.Groups, addOnly); err != nil {
			h.setFailureCondition(ctx, keycloakClient, fmt.Sprintf("Failed to sync service account: %s", err.Error()))
			return fmt.Errorf("unable to sync service account groups: %w", err)
		}
	}

	// Set attributes
	if keycloakClient.Spec.ServiceAccount.AttributesV2 != nil {
		if err := h.setAttributes(ctx, realmName, saUserID, keycloakClient.Spec.ServiceAccount.AttributesV2, saUser.Attributes, addOnly); err != nil {
			h.setFailureCondition(ctx, keycloakClient, fmt.Sprintf("Failed to sync service account: %s", err.Error()))
			return fmt.Errorf("unable to set service account attributes: %w", err)
		}
	}

	h.setSuccessCondition(ctx, keycloakClient, "Service account synchronized")

	return nil
}

func (h *ServiceAccount) setFailureCondition(ctx context.Context, keycloakClient *keycloakApi.KeycloakClient, message string) {
	log := ctrl.LoggerFrom(ctx)

	if err := SetCondition(
		ctx, h.k8sClient, keycloakClient,
		ConditionServiceAccountSynced,
		metav1.ConditionFalse,
		ReasonKeycloakAPIError,
		message,
	); err != nil {
		log.Error(err, "Failed to set failure condition")
	}
}

func (h *ServiceAccount) setSuccessCondition(ctx context.Context, keycloakClient *keycloakApi.KeycloakClient, message string) {
	log := ctrl.LoggerFrom(ctx)

	if err := SetCondition(
		ctx, h.k8sClient, keycloakClient,
		ConditionServiceAccountSynced,
		metav1.ConditionTrue,
		ReasonServiceAccountSynced,
		message,
	); err != nil {
		log.Error(err, "Failed to set success condition")
	}
}

func (h *ServiceAccount) syncRealmRoles(
	ctx context.Context,
	realmName, saUserID string,
	desiredRoleNames []string,
	addOnly bool,
) error {
	// Get current realm role mappings
	currentRoles, _, err := h.kClient.Users.GetUserRealmRoleMappings(ctx, realmName, saUserID)
	if err != nil {
		return fmt.Errorf("unable to get user realm role mappings: %w", err)
	}

	currentRoleMap := maputil.SliceToMapSelf(currentRoles, func(r keycloakapi.RoleRepresentation) (string, bool) {
		return *r.Name, r.Name != nil
	})

	desiredSet := make(map[string]bool, len(desiredRoleNames))
	for _, name := range desiredRoleNames {
		desiredSet[name] = true
	}

	// Add missing roles
	var toAdd []keycloakapi.RoleRepresentation

	for _, roleName := range desiredRoleNames {
		if _, exists := currentRoleMap[roleName]; !exists {
			role, _, err := h.kClient.Roles.GetRealmRole(ctx, realmName, roleName)
			if err != nil {
				return fmt.Errorf("unable to get realm role %s: %w", roleName, err)
			}

			toAdd = append(toAdd, *role)
		}
	}

	if len(toAdd) > 0 {
		if _, err := h.kClient.Users.AddUserRealmRoles(ctx, realmName, saUserID, toAdd); err != nil {
			return fmt.Errorf("unable to add realm roles: %w", err)
		}
	}

	// Remove extra roles (unless addOnly)
	if !addOnly {
		var toRemove []keycloakapi.RoleRepresentation

		for name, role := range currentRoleMap {
			if !desiredSet[name] {
				toRemove = append(toRemove, role)
			}
		}

		if len(toRemove) > 0 {
			if _, err := h.kClient.Users.DeleteUserRealmRoles(ctx, realmName, saUserID, toRemove); err != nil {
				return fmt.Errorf("unable to delete realm roles: %w", err)
			}
		}
	}

	return nil
}

func (h *ServiceAccount) syncClientRoles(
	ctx context.Context,
	realmName, saUserID string,
	desiredClientRoles map[string][]string,
	addOnly bool,
) error {
	for targetClientID, desiredRoleNames := range desiredClientRoles {
		// Get UUID of the target client
		targetClient, _, err := h.kClient.Clients.GetClientByClientID(ctx, realmName, targetClientID)
		if err != nil {
			return fmt.Errorf("unable to get client %s: %w", targetClientID, err)
		}

		if targetClient == nil || targetClient.Id == nil {
			return fmt.Errorf("client %s not found", targetClientID)
		}

		targetClientUUID := *targetClient.Id

		// Get current client role mappings for this user
		currentRoles, _, err := h.kClient.Users.GetUserClientRoleMappings(ctx, realmName, saUserID, targetClientUUID)
		if err != nil {
			return fmt.Errorf("unable to get user client role mappings for client %s: %w", targetClientID, err)
		}

		currentRoleMap := maputil.SliceToMapSelf(currentRoles, func(r keycloakapi.RoleRepresentation) (string, bool) {
			return *r.Name, r.Name != nil
		})

		desiredSet := make(map[string]bool, len(desiredRoleNames))
		for _, name := range desiredRoleNames {
			desiredSet[name] = true
		}

		// Add missing roles
		var toAdd []keycloakapi.RoleRepresentation

		for _, roleName := range desiredRoleNames {
			if _, exists := currentRoleMap[roleName]; !exists {
				role, _, err := h.kClient.Clients.GetClientRole(ctx, realmName, targetClientUUID, roleName)
				if err != nil {
					return fmt.Errorf("unable to get client role %s/%s: %w", targetClientID, roleName, err)
				}

				toAdd = append(toAdd, *role)
			}
		}

		if len(toAdd) > 0 {
			if _, err := h.kClient.Users.AddUserClientRoles(ctx, realmName, saUserID, targetClientUUID, toAdd); err != nil {
				return fmt.Errorf("unable to add client roles for client %s: %w", targetClientID, err)
			}
		}

		// Remove extra roles (unless addOnly)
		if !addOnly {
			var toRemove []keycloakapi.RoleRepresentation

			for name, role := range currentRoleMap {
				if !desiredSet[name] {
					toRemove = append(toRemove, role)
				}
			}

			if len(toRemove) > 0 {
				if _, err := h.kClient.Users.DeleteUserClientRoles(ctx, realmName, saUserID, targetClientUUID, toRemove); err != nil {
					return fmt.Errorf("unable to delete client roles for client %s: %w", targetClientID, err)
				}
			}
		}
	}

	return nil
}

func (h *ServiceAccount) syncGroups(
	ctx context.Context,
	realmName, saUserID string,
	desiredGroupNames []string,
	addOnly bool,
) error {
	// Get current user groups
	currentGroups, _, err := h.kClient.Users.GetUserGroups(ctx, realmName, saUserID)
	if err != nil {
		return fmt.Errorf("unable to get user groups: %w", err)
	}

	currentGroupMap := maputil.SliceToMap(currentGroups, // name -> id
		func(g keycloakapi.GroupRepresentation) (string, bool) { return *g.Name, g.Name != nil && g.Id != nil },
		func(g keycloakapi.GroupRepresentation) string { return *g.Id },
	)

	desiredSet := make(map[string]bool, len(desiredGroupNames))
	for _, name := range desiredGroupNames {
		desiredSet[name] = true
	}

	// Get all groups in realm to look up IDs
	allGroups, _, err := h.kClient.Groups.GetGroups(ctx, realmName, nil)
	if err != nil {
		return fmt.Errorf("unable to get realm groups: %w", err)
	}

	allGroupMap := maputil.SliceToMap(allGroups, // name -> id
		func(g keycloakapi.GroupRepresentation) (string, bool) { return *g.Name, g.Name != nil && g.Id != nil },
		func(g keycloakapi.GroupRepresentation) string { return *g.Id },
	)

	// Add user to missing groups
	for _, groupName := range desiredGroupNames {
		if _, exists := currentGroupMap[groupName]; !exists {
			groupID, ok := allGroupMap[groupName]
			if !ok {
				return fmt.Errorf("group %s not found in realm", groupName)
			}

			if _, err := h.kClient.Users.AddUserToGroup(ctx, realmName, saUserID, groupID); err != nil {
				return fmt.Errorf("unable to add user to group %s: %w", groupName, err)
			}
		}
	}

	// Remove user from extra groups (unless addOnly)
	if !addOnly {
		for name, groupID := range currentGroupMap {
			if !desiredSet[name] {
				if _, err := h.kClient.Users.RemoveUserFromGroup(ctx, realmName, saUserID, groupID); err != nil {
					return fmt.Errorf("unable to remove user from group %s: %w", name, err)
				}
			}
		}
	}

	return nil
}

func (h *ServiceAccount) setAttributes(
	ctx context.Context,
	realmName, saUserID string,
	desiredAttributes map[string][]string,
	currentAttributes *map[string][]string,
	addOnly bool,
) error {
	var updatedAttrs map[string][]string

	if addOnly && currentAttributes != nil {
		updatedAttrs = make(map[string][]string, len(*currentAttributes)+len(desiredAttributes))
		maps.Copy(updatedAttrs, *currentAttributes)
		maps.Copy(updatedAttrs, desiredAttributes)
	} else {
		updatedAttrs = make(map[string][]string, len(desiredAttributes))
		maps.Copy(updatedAttrs, desiredAttributes)
	}

	userUpdate := keycloakapi.UserRepresentation{
		Attributes: &updatedAttrs,
	}

	if _, err := h.kClient.Users.UpdateUser(ctx, realmName, saUserID, userUpdate); err != nil {
		return fmt.Errorf("unable to update service account attributes: %w", err)
	}

	return nil
}

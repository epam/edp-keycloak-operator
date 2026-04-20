package chain

import (
	"context"
	"fmt"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	keycloakApi "github.com/epam/edp-keycloak-operator/api/v1"
	"github.com/epam/edp-keycloak-operator/pkg/client/keycloakapi"
	"github.com/epam/edp-keycloak-operator/pkg/maputil"
)

const permissionLogKey = "permission"

type ProcessPermissions struct {
	kClient   *keycloakapi.KeycloakClient
	k8sClient client.Client
}

func NewProcessPermissions(kClient *keycloakapi.KeycloakClient, k8sClient client.Client) *ProcessPermissions {
	return &ProcessPermissions{kClient: kClient, k8sClient: k8sClient}
}

func (h *ProcessPermissions) Serve(ctx context.Context, keycloakClient *keycloakApi.KeycloakClient, realmName string, clientCtx *ClientContext) error {
	log := ctrl.LoggerFrom(ctx)

	if keycloakClient.Spec.Authorization == nil {
		log.Info("Authorization settings are not specified")
		return nil
	}

	clientUUID := clientCtx.ClientUUID

	permissionsList, _, err := h.kClient.Authorization.GetPermissions(ctx, realmName, clientUUID)
	if err != nil {
		h.setFailureCondition(ctx, keycloakClient, fmt.Sprintf("Failed to sync authorization permissions: %s", err.Error()))

		return fmt.Errorf("failed to get permissions: %w", err)
	}

	existingPermissions := maputil.SliceToMapSelf(permissionsList, func(p keycloakapi.AbstractPolicyRepresentation) (string, bool) {
		return *p.Name, p.Name != nil
	})

	for i := 0; i < len(keycloakClient.Spec.Authorization.Permissions); i++ {
		log.Info("Processing permission", permissionLogKey, keycloakClient.Spec.Authorization.Permissions[i].Name)

		var permissionRepresentation keycloakapi.PolicyRepresentation

		if permissionRepresentation, err = h.toPermissionRepresentation(ctx, &keycloakClient.Spec.Authorization.Permissions[i], clientUUID, realmName); err != nil {
			h.setFailureCondition(ctx, keycloakClient, fmt.Sprintf("Failed to sync authorization permissions: %s", err.Error()))

			return fmt.Errorf("failed to convert permission: %w", err)
		}

		permType := keycloakClient.Spec.Authorization.Permissions[i].Type

		existingPermission, ok := existingPermissions[keycloakClient.Spec.Authorization.Permissions[i].Name]
		if ok {
			if existingPermission.Id == nil {
				return fmt.Errorf("existing permission %s does not have ID", keycloakClient.Spec.Authorization.Permissions[i].Name)
			}

			permissionRepresentation.Id = existingPermission.Id
			if _, err = h.kClient.Authorization.UpdatePermission(ctx, realmName, clientUUID, permType, *existingPermission.Id, permissionRepresentation); err != nil {
				h.setFailureCondition(ctx, keycloakClient, fmt.Sprintf("Failed to sync authorization permissions: %s", err.Error()))

				return fmt.Errorf("failed to update permission: %w", err)
			}

			log.Info("Permission updated", permissionLogKey, keycloakClient.Spec.Authorization.Permissions[i].Name)

			delete(existingPermissions, keycloakClient.Spec.Authorization.Permissions[i].Name)

			continue
		}

		if _, _, err = h.kClient.Authorization.CreatePermission(ctx, realmName, clientUUID, permType, permissionRepresentation); err != nil {
			h.setFailureCondition(ctx, keycloakClient, fmt.Sprintf("Failed to sync authorization permissions: %s", err.Error()))

			return fmt.Errorf("failed to create permission: %w", err)
		}

		log.Info("Permission created", permissionLogKey, keycloakClient.Spec.Authorization.Permissions[i].Name)
	}

	if keycloakClient.Spec.ReconciliationStrategy != keycloakApi.ReconciliationStrategyAddOnly {
		if err = h.deletePermissions(ctx, existingPermissions, realmName, clientUUID); err != nil {
			h.setFailureCondition(ctx, keycloakClient, fmt.Sprintf("Failed to sync authorization permissions: %s", err.Error()))

			return err
		}
	}

	h.setSuccessCondition(ctx, keycloakClient, "Authorization permissions synchronized")

	return nil
}

func (h *ProcessPermissions) setFailureCondition(ctx context.Context, keycloakClient *keycloakApi.KeycloakClient, message string) {
	log := ctrl.LoggerFrom(ctx)

	if err := SetCondition(
		ctx, h.k8sClient, keycloakClient,
		ConditionAuthorizationPermissionsSynced,
		metav1.ConditionFalse,
		ReasonKeycloakAPIError,
		message,
	); err != nil {
		log.Error(err, "Failed to set failure condition")
	}
}

func (h *ProcessPermissions) setSuccessCondition(ctx context.Context, keycloakClient *keycloakApi.KeycloakClient, message string) {
	log := ctrl.LoggerFrom(ctx)

	if err := SetCondition(
		ctx, h.k8sClient, keycloakClient,
		ConditionAuthorizationPermissionsSynced,
		metav1.ConditionTrue,
		ReasonAuthorizationPermissionsSynced,
		message,
	); err != nil {
		log.Error(err, "Failed to set success condition")
	}
}

func (h *ProcessPermissions) deletePermissions(ctx context.Context, existingPermissions map[string]keycloakapi.AbstractPolicyRepresentation, realmName string, clientUUID string) error {
	log := ctrl.LoggerFrom(ctx)

	for name := range existingPermissions {
		if name == "Default Permission" {
			continue
		}

		p := existingPermissions[name]
		if p.Id == nil {
			continue
		}

		if _, err := h.kClient.Authorization.DeletePermission(ctx, realmName, clientUUID, *p.Id); err != nil {
			if !keycloakapi.IsNotFound(err) {
				return fmt.Errorf("failed to delete permission: %w", err)
			}
		}

		log.Info("Permission deleted", permissionLogKey, name)
	}

	return nil
}

// toPermissionRepresentation converts keycloakApi.Permission to keycloakapi.PolicyRepresentation.
func (h *ProcessPermissions) toPermissionRepresentation(ctx context.Context, permission *keycloakApi.Permission, clientUUID, realm string) (keycloakapi.PolicyRepresentation, error) {
	keycloakPermission := getBasePermissionRepresentation(permission)

	if err := h.mapResources(ctx, permission, &keycloakPermission, realm, clientUUID); err != nil {
		return keycloakapi.PolicyRepresentation{}, fmt.Errorf("failed to map resources: %w", err)
	}

	if err := h.mapPolicies(ctx, permission, &keycloakPermission, realm, clientUUID); err != nil {
		return keycloakapi.PolicyRepresentation{}, fmt.Errorf("failed to map policies: %w", err)
	}

	if permission.Type == keycloakApi.PermissionTypeScope {
		if err := h.mapScopes(ctx, permission, &keycloakPermission, realm, clientUUID); err != nil {
			return keycloakapi.PolicyRepresentation{}, fmt.Errorf("failed to map scopes: %w", err)
		}
	}

	return keycloakPermission, nil
}

func (h *ProcessPermissions) mapResources(
	ctx context.Context,
	permission *keycloakApi.Permission,
	keycloakPermission *keycloakapi.PolicyRepresentation,
	realm,
	clientUUID string,
) error {
	if len(permission.Resources) == 0 {
		emptyResources := []string{}
		keycloakPermission.Resources = &emptyResources

		return nil
	}

	resourcesList, _, err := h.kClient.Authorization.GetResources(ctx, realm, clientUUID)
	if err != nil {
		return fmt.Errorf("failed to get resources: %w", err)
	}

	existingResources := maputil.SliceToMapSelf(resourcesList, func(r keycloakapi.ResourceRepresentation) (string, bool) {
		return *r.Name, r.Name != nil
	})

	permissionResources := make([]string, 0, len(permission.Resources))

	for _, r := range permission.Resources {
		existingResource, ok := existingResources[r]
		if !ok {
			return fmt.Errorf("resource %s does not exist", r)
		}

		if existingResource.UnderscoreId == nil {
			return fmt.Errorf("resource %s does not have ID", r)
		}

		permissionResources = append(permissionResources, *existingResource.UnderscoreId)
	}

	keycloakPermission.Resources = &permissionResources

	return nil
}

func (h *ProcessPermissions) mapPolicies(
	ctx context.Context,
	permission *keycloakApi.Permission,
	keycloakPermission *keycloakapi.PolicyRepresentation,
	realm,
	clientUUID string,
) error {
	if len(permission.Policies) == 0 {
		emptyPolicies := []string{}
		keycloakPermission.Policies = &emptyPolicies

		return nil
	}

	policiesList, _, err := h.kClient.Authorization.GetPolicies(ctx, realm, clientUUID)
	if err != nil {
		return fmt.Errorf("failed to get polices: %w", err)
	}

	existingPolicies := maputil.SliceToMapSelf(policiesList, func(p keycloakapi.AbstractPolicyRepresentation) (string, bool) {
		return *p.Name, p.Name != nil
	})

	permissionPolicies := make([]string, 0, len(permission.Policies))

	for _, r := range permission.Policies {
		existingPolicy, ok := existingPolicies[r]
		if !ok {
			return fmt.Errorf("policy %s does not exist", r)
		}

		if existingPolicy.Id == nil {
			return fmt.Errorf("policy %s does not have ID", r)
		}

		permissionPolicies = append(permissionPolicies, *existingPolicy.Id)
	}

	keycloakPermission.Policies = &permissionPolicies

	return nil
}

func (h *ProcessPermissions) mapScopes(
	ctx context.Context,
	permission *keycloakApi.Permission,
	keycloakPermission *keycloakapi.PolicyRepresentation,
	realm,
	clientUUID string,
) error {
	if len(permission.Scopes) == 0 {
		emptyScopes := []string{}
		keycloakPermission.Scopes = &emptyScopes

		return nil
	}

	scopesList, _, err := h.kClient.Authorization.GetScopes(ctx, realm, clientUUID)
	if err != nil {
		return fmt.Errorf("failed to get scopes: %w", err)
	}

	existingScopes := maputil.SliceToMapSelf(scopesList, func(s keycloakapi.ScopeRepresentation) (string, bool) {
		return *s.Name, s.Name != nil
	})

	permissionScopes := make([]string, 0, len(permission.Scopes))

	for _, r := range permission.Scopes {
		existingScope, ok := existingScopes[r]
		if !ok {
			return fmt.Errorf("scope %s does not exist", r)
		}

		if existingScope.Id == nil {
			return fmt.Errorf("scope %s does not have ID", r)
		}

		permissionScopes = append(permissionScopes, *existingScope.Id)
	}

	keycloakPermission.Scopes = &permissionScopes

	return nil
}

func getBasePermissionRepresentation(policy *keycloakApi.Permission) keycloakapi.PolicyRepresentation {
	keycloakPermission := keycloakapi.PolicyRepresentation{}

	name := policy.Name
	keycloakPermission.Name = &name

	pType := policy.Type
	keycloakPermission.Type = &pType

	desc := policy.Description
	decisionStrategy := keycloakapi.DecisionStrategy(policy.DecisionStrategy)

	keycloakPermission.DecisionStrategy = &decisionStrategy
	keycloakPermission.Description = &desc

	logic := keycloakapi.Logic(policy.Logic)
	keycloakPermission.Logic = &logic

	return keycloakPermission
}

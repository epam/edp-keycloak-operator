package chain

import (
	"context"
	"fmt"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	keycloakApi "github.com/epam/edp-keycloak-operator/api/v1"
	keycloakapi "github.com/epam/edp-keycloak-operator/pkg/client/keycloakapi"
	"github.com/epam/edp-keycloak-operator/pkg/maputil"
)

type PutAdminFineGrainedPermissions struct {
	kClient   *keycloakapi.APIClient
	k8sClient client.Client
}

func NewPutAdminFineGrainedPermissions(kClient *keycloakapi.APIClient, k8sClient client.Client) *PutAdminFineGrainedPermissions {
	return &PutAdminFineGrainedPermissions{kClient: kClient, k8sClient: k8sClient}
}

func (h *PutAdminFineGrainedPermissions) Serve(ctx context.Context, keycloakClient *keycloakApi.KeycloakClient, realmName string, clientCtx *ClientContext) error {
	log := ctrl.LoggerFrom(ctx)

	featureFlagEnabled, err := h.kClient.Server.FeatureFlagEnabled(ctx, keycloakapi.FeatureFlagAdminFineGrainedAuthz)
	if err != nil {
		h.setFailureCondition(ctx, keycloakClient, fmt.Sprintf("Failed to sync admin fine-grained permissions: %s", err.Error()))

		return fmt.Errorf("failed to check feature flag ADMIN_FINE_GRAINED_AUTHZ: %w", err)
	}

	if !featureFlagEnabled {
		log.Info("Feature flag is not enabled, skipping admin fine grained permissions", "featureFlag", keycloakapi.FeatureFlagAdminFineGrainedAuthz)

		return nil
	}

	clientUUID := clientCtx.ClientUUID

	if err := h.putKeycloakClientAdminFineGrainedPermissions(ctx, keycloakClient, realmName, clientUUID); err != nil {
		h.setFailureCondition(ctx, keycloakClient, fmt.Sprintf("Failed to sync admin fine-grained permissions: %s", err.Error()))

		return fmt.Errorf("unable to put keycloak client admin fine grained permissions: %w", err)
	}

	if keycloakClient.Spec.AdminFineGrainedPermissionsEnabled && keycloakClient.Spec.Permission != nil {
		if err := h.putKeycloakClientAdminPermissionPolicies(ctx, keycloakClient, realmName, clientUUID); err != nil {
			h.setFailureCondition(ctx, keycloakClient, fmt.Sprintf("Failed to sync admin fine-grained permissions: %s", err.Error()))

			return fmt.Errorf("unable to put keycloak client admin permission policies: %w", err)
		}
	}

	h.setSuccessCondition(ctx, keycloakClient, "Admin fine-grained permissions synchronized")

	return nil
}

func (h *PutAdminFineGrainedPermissions) setFailureCondition(ctx context.Context, keycloakClient *keycloakApi.KeycloakClient, message string) {
	log := ctrl.LoggerFrom(ctx)

	if err := SetCondition(
		ctx, h.k8sClient, keycloakClient,
		ConditionAdminFineGrainedPermissionsV1Synced,
		metav1.ConditionFalse,
		ReasonKeycloakAPIError,
		message,
	); err != nil {
		log.Error(err, "Failed to set failure condition")
	}
}

func (h *PutAdminFineGrainedPermissions) setSuccessCondition(ctx context.Context, keycloakClient *keycloakApi.KeycloakClient, message string) {
	log := ctrl.LoggerFrom(ctx)

	if err := SetCondition(
		ctx, h.k8sClient, keycloakClient,
		ConditionAdminFineGrainedPermissionsV1Synced,
		metav1.ConditionTrue,
		ReasonAdminFineGrainedPermissionsV1Synced,
		message,
	); err != nil {
		log.Error(err, "Failed to set success condition")
	}
}

func (h *PutAdminFineGrainedPermissions) putKeycloakClientAdminFineGrainedPermissions(ctx context.Context, keycloakClient *keycloakApi.KeycloakClient, realmName, clientUUID string) error {
	reqLog := ctrl.LoggerFrom(ctx)
	reqLog.Info("Start put keycloak client admin fine grained permissions")

	managementPermissions := keycloakapi.ManagementPermissionReference{
		Enabled: &keycloakClient.Spec.AdminFineGrainedPermissionsEnabled,
	}

	if _, _, err := h.kClient.Clients.UpdateClientManagementPermissions(ctx, realmName, clientUUID, managementPermissions); err != nil {
		return fmt.Errorf("unable to update client management permissions: %w", err)
	}

	reqLog.Info("End put keycloak client admin fine grained permissions")

	return nil
}

func (h *PutAdminFineGrainedPermissions) putKeycloakClientAdminPermissionPolicies(ctx context.Context, keycloakClient *keycloakApi.KeycloakClient, realmName, clientUUID string) error {
	reqLog := ctrl.LoggerFrom(ctx)
	reqLog.Info("Start put keycloak client admin permission policies")

	realmManagementClientUUID, err := h.kClient.Clients.GetClientUUID(ctx, realmName, keycloakapi.RealmManagementClient)
	if err != nil {
		return fmt.Errorf("failed to get %s client id: %w", keycloakapi.RealmManagementClient, err)
	}

	realmManagementPermissionsList, _, err := h.kClient.Authorization.GetPermissions(ctx, realmName, realmManagementClientUUID)
	if err != nil {
		return fmt.Errorf("failed to get permissions for %s client: %w", keycloakapi.RealmManagementClient, err)
	}

	realmManagementPermissions := maputil.SliceToMapSelf(realmManagementPermissionsList, func(p keycloakapi.AbstractPolicyRepresentation) (string, bool) {
		return *p.Name, p.Name != nil
	})

	existingClientPermissions, _, err := h.kClient.Clients.GetClientManagementPermissions(ctx, realmName, clientUUID)
	if err != nil {
		return fmt.Errorf("failed to get client permissions: %w", err)
	}

	if existingClientPermissions == nil || existingClientPermissions.ScopePermissions == nil {
		return fmt.Errorf("client management permissions or scope permissions are nil")
	}

	existingScopePermissions := *existingClientPermissions.ScopePermissions

	for i := 0; i < len(keycloakClient.Spec.Permission.ScopePermissions); i++ {
		name := keycloakClient.Spec.Permission.ScopePermissions[i].Name
		reqLog.Info("Processing scope permission", scopeLogKey, name)

		if _, ok := existingScopePermissions[name]; !ok {
			return fmt.Errorf("scope %s not found in permissions", name)
		}

		permissionName := fmt.Sprintf("%s.permission.client.%s", name, clientUUID)

		if permission, ok := realmManagementPermissions[permissionName]; ok {
			if permission.Id == nil {
				continue
			}

			// Build the updated permission with policies
			policies := keycloakClient.Spec.Permission.ScopePermissions[i].Policies
			updatedPermission := keycloakapi.PolicyRepresentation{
				Id:       permission.Id,
				Name:     permission.Name,
				Type:     permission.Type,
				Policies: &policies,
			}

			permType := ""
			if permission.Type != nil {
				permType = *permission.Type
			}

			if _, err = h.kClient.Authorization.UpdatePermission(ctx, realmName, realmManagementClientUUID, permType, *permission.Id, updatedPermission); err != nil {
				return fmt.Errorf("failed to update permission %s: %w", permissionName, err)
			}

			reqLog.Info("Scope permission updated", scopeLogKey, name, permissionLogKey, permissionName)
		}
	}

	reqLog.Info("End put keycloak client admin permission policies")

	return nil
}

package chain

import (
	"context"
	"fmt"

	"github.com/pkg/errors"
	ctrl "sigs.k8s.io/controller-runtime"

	keycloakApi "github.com/epam/edp-keycloak-operator/api/v1"
	"github.com/epam/edp-keycloak-operator/pkg/client/keycloak"
	"github.com/epam/edp-keycloak-operator/pkg/client/keycloak/adapter"
)

type PutAdminFineGrainedPermissions struct {
	keycloakApiClient keycloak.Client
}

func NewPutAdminFineGrainedPermissions(keycloakApiClient keycloak.Client) *PutAdminFineGrainedPermissions {
	return &PutAdminFineGrainedPermissions{keycloakApiClient: keycloakApiClient}
}

func (el *PutAdminFineGrainedPermissions) Serve(ctx context.Context, keycloakClient *keycloakApi.KeycloakClient, realmName string) error {
	clientID, err := el.keycloakApiClient.GetClientID(keycloakClient.Spec.ClientId, realmName)
	if err != nil {
		return fmt.Errorf("failed to get client id: %w", err)
	}

	if err := el.putKeycloakClientAdminFineGrainedPermissions(ctx, keycloakClient, realmName, clientID); err != nil {
		return errors.Wrap(err, "unable to put keycloak client admin fine grained permissions")
	}

	if keycloakClient.Spec.AdminFineGrainedPermissions.Enabled {
		if err := el.putKeycloakClientAdminPermissionPolicies(ctx, keycloakClient, realmName, clientID); err != nil {
			return errors.Wrap(err, "unable to put keycloak client admin permission policies")
		}
	}

	return nil
}

func (el *PutAdminFineGrainedPermissions) putKeycloakClientAdminFineGrainedPermissions(ctx context.Context, keycloakClient *keycloakApi.KeycloakClient, realmName, clientID string) error {
	reqLog := ctrl.LoggerFrom(ctx)
	reqLog.Info("Start put keycloak client admin fine grained permissions")

	managementPermissions := &adapter.ManagementPermissionRepresentation{
		Enabled: &keycloakClient.Spec.AdminFineGrainedPermissions.Enabled,
	}

	if err := el.keycloakApiClient.UpdateClientManagementPermissions(realmName, clientID, managementPermissions); err != nil {
		return errors.Wrap(err, "unable to update client management permissions")
	}

	reqLog.Info("End put keycloak client admin fine grained permissions")

	return nil
}

func (el *PutAdminFineGrainedPermissions) putKeycloakClientAdminPermissionPolicies(ctx context.Context, keycloakClient *keycloakApi.KeycloakClient, realmName, clientID string) error {
	reqLog := ctrl.LoggerFrom(ctx)
	reqLog.Info("Start put keycloak client admin permission policies")

	realmManagementClientID, err := el.keycloakApiClient.GetClientID("realm-management", realmName)
	if err != nil {
		return fmt.Errorf("failed to get realm-management client id: %w", err)
	}

	realmManagementPermissions, err := el.keycloakApiClient.GetPermissions(ctx, realmName, realmManagementClientID)
	if err != nil {
		return fmt.Errorf("failed to get permissions for realm-management client: %w", err)
	}

	existingClientPermissions, err := el.keycloakApiClient.GetClientManagementPermissions(realmName, clientID)
	if err != nil {
		return fmt.Errorf("failed to get client permissions: %w", err)
	}

	existingScopePermissions := *existingClientPermissions.ScopePermissions

	for i := 0; i < len(keycloakClient.Spec.AdminFineGrainedPermissions.ScopePermissions); i++ {
		name := keycloakClient.Spec.AdminFineGrainedPermissions.ScopePermissions[i].Name
		reqLog.Info("Processing scope permission", scopeLogKey, name)

		if _, ok := existingScopePermissions[name]; !ok {
			return fmt.Errorf("scope %s not found in permissions", name)
		}

		permissionName := fmt.Sprintf("%s.permission.client.%s", name, clientID)

		if permission, ok := realmManagementPermissions[permissionName]; ok {
			permission.Policies = &keycloakClient.Spec.AdminFineGrainedPermissions.ScopePermissions[i].Policies
			if err = el.keycloakApiClient.UpdatePermission(ctx, realmName, realmManagementClientID, permission); err != nil {
				return fmt.Errorf("failed to update permission %s: %w", permissionName, err)
			}

			reqLog.Info("Scope permission updated", scopeLogKey, name, permissionLogKey, permissionName)
		}
	}

	reqLog.Info("End put keycloak client admin permission policies")

	return nil
}

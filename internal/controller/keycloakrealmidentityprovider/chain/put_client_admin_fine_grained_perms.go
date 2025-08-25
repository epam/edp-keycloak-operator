package chain

import (
	"context"
	"fmt"

	ctrl "sigs.k8s.io/controller-runtime"

	keycloakApi "github.com/epam/edp-keycloak-operator/api/v1"
	"github.com/epam/edp-keycloak-operator/pkg/client/keycloak"
	"github.com/epam/edp-keycloak-operator/pkg/client/keycloak/adapter"
)

const (
	// RealmManagementClient built-in Keycloak client for the realm
	// This client manages admin fine-grained permissions for other clients.
	RealmManagementClient = "realm-management"

	scopeLogKey      = "scope"
	permissionLogKey = "permission"
)

type PutAdminFineGrainedPermissions struct {
	keycloakApiClient keycloak.Client
}

func NewPutAdminFineGrainedPermissions(keycloakApiClient keycloak.Client) *PutAdminFineGrainedPermissions {
	return &PutAdminFineGrainedPermissions{keycloakApiClient: keycloakApiClient}
}

func (el *PutAdminFineGrainedPermissions) Serve(ctx context.Context, keycloakIDP *keycloakApi.KeycloakRealmIdentityProvider, realmName string) error {
	featureFlagEnabled, err := el.keycloakApiClient.FeatureFlagEnabled(ctx, adapter.FeatureFlagAdminFineGrainedAuthz)
	if err != nil {
		return fmt.Errorf("failed to check feature flag ADMIN_FINE_GRAINED_AUTHZ: %w", err)
	}

	if !featureFlagEnabled {
		log := ctrl.LoggerFrom(ctx)
		log.Info("Feature flag is not enabled, skipping admin fine grained permissions", "featureFlag", adapter.FeatureFlagAdminFineGrainedAuthz)

		return nil
	}

	if err = el.putKeycloakClientAdminFineGrainedPermissions(ctx, keycloakIDP, realmName); err != nil {
		return fmt.Errorf("unable to put keycloak idp admin fine grained permissions: %w", err)
	}

	if keycloakIDP.Spec.AdminFineGrainedPermissionsEnabled && keycloakIDP.Spec.Permission != nil {
		if err = el.putKeycloakIDPAdminPermissionPolicies(ctx, keycloakIDP, realmName); err != nil {
			return fmt.Errorf("unable to put keycloak idp admin permission policies: %w", err)
		}
	}

	return nil
}

func (el *PutAdminFineGrainedPermissions) putKeycloakClientAdminFineGrainedPermissions(ctx context.Context, keycloakIDP *keycloakApi.KeycloakRealmIdentityProvider, realmName string) error {
	reqLog := ctrl.LoggerFrom(ctx)
	reqLog.Info("Start put keycloak idp admin fine grained permissions")

	managementPermissions := adapter.ManagementPermissionRepresentation{
		Enabled: &keycloakIDP.Spec.AdminFineGrainedPermissionsEnabled,
	}

	if err := el.keycloakApiClient.UpdateIDPManagementPermissions(realmName, keycloakIDP.Spec.Alias, managementPermissions); err != nil {
		return fmt.Errorf("unable to update idp management permissions: %w", err)
	}

	reqLog.Info("End put keycloak idp admin fine grained permissions")

	return nil
}

func (el *PutAdminFineGrainedPermissions) putKeycloakIDPAdminPermissionPolicies(ctx context.Context, keycloakIDP *keycloakApi.KeycloakRealmIdentityProvider, realmName string) error {
	reqLog := ctrl.LoggerFrom(ctx)
	reqLog.Info("Start put keycloak idp admin permission policies")

	realmManagementClientID, err := el.keycloakApiClient.GetClientID(RealmManagementClient, realmName)
	if err != nil {
		return fmt.Errorf("failed to get %s client id: %w", RealmManagementClient, err)
	}

	realmManagementPermissions, err := el.keycloakApiClient.GetPermissions(ctx, realmName, realmManagementClientID)
	if err != nil {
		return fmt.Errorf("failed to get permissions for %s client: %w", RealmManagementClient, err)
	}

	existingIDPPermissions, err := el.keycloakApiClient.GetIDPManagementPermissions(realmName, keycloakIDP.Spec.Alias)
	if err != nil {
		return fmt.Errorf("failed to get idp permissions: %w", err)
	}

	existingScopePermissions := *existingIDPPermissions.ScopePermissions

	for i := 0; i < len(keycloakIDP.Spec.Permission.ScopePermissions); i++ {
		name := keycloakIDP.Spec.Permission.ScopePermissions[i].Name
		reqLog.Info("Processing scope permission", scopeLogKey, name)

		if _, ok := existingScopePermissions[name]; !ok {
			return fmt.Errorf("scope %s not found in permissions", name)
		}

		permissionName := fmt.Sprintf("%s.permission.idp.%s", name, keycloakIDP.Spec.Alias)

		if permission, ok := realmManagementPermissions[permissionName]; ok {
			permission.Policies = &keycloakIDP.Spec.Permission.ScopePermissions[i].Policies
			if err = el.keycloakApiClient.UpdatePermission(ctx, realmName, realmManagementClientID, permission); err != nil {
				return fmt.Errorf("failed to update permission %s: %w", permissionName, err)
			}

			reqLog.Info("Scope permission updated", scopeLogKey, name, permissionLogKey, permissionName)
		}
	}

	reqLog.Info("End put keycloak idp admin permission policies")

	return nil
}

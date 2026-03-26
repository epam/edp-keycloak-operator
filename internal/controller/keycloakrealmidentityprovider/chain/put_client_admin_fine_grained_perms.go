package chain

import (
	"context"
	"fmt"

	"k8s.io/utils/ptr"
	ctrl "sigs.k8s.io/controller-runtime"

	keycloakApi "github.com/epam/edp-keycloak-operator/api/v1"
	keycloakv2 "github.com/epam/edp-keycloak-operator/pkg/client/keycloakv2"
	"github.com/epam/edp-keycloak-operator/pkg/maputil"
)

const (
	scopeLogKey      = "scope"
	permissionLogKey = "permission"
)

type PutAdminFineGrainedPermissions struct {
	kClient *keycloakv2.KeycloakClient
}

func NewPutAdminFineGrainedPermissions(kClient *keycloakv2.KeycloakClient) *PutAdminFineGrainedPermissions {
	return &PutAdminFineGrainedPermissions{kClient: kClient}
}

func (h *PutAdminFineGrainedPermissions) Serve(ctx context.Context, keycloakIDP *keycloakApi.KeycloakRealmIdentityProvider, realmName string) error {
	featureFlagEnabled, err := h.kClient.Server.FeatureFlagEnabled(ctx, keycloakv2.FeatureFlagAdminFineGrainedAuthz)
	if err != nil {
		return fmt.Errorf("failed to check feature flag ADMIN_FINE_GRAINED_AUTHZ: %w", err)
	}

	if !featureFlagEnabled {
		log := ctrl.LoggerFrom(ctx)
		log.Info("Feature flag is not enabled, skipping admin fine grained permissions", "featureFlag", keycloakv2.FeatureFlagAdminFineGrainedAuthz)

		return nil
	}

	if err = h.putKeycloakClientAdminFineGrainedPermissions(ctx, keycloakIDP, realmName); err != nil {
		return fmt.Errorf("unable to put keycloak idp admin fine grained permissions: %w", err)
	}

	if keycloakIDP.Spec.AdminFineGrainedPermissionsEnabled && keycloakIDP.Spec.Permission != nil {
		if err = h.putKeycloakIDPAdminPermissionPolicies(ctx, keycloakIDP, realmName); err != nil {
			return fmt.Errorf("unable to put keycloak idp admin permission policies: %w", err)
		}
	}

	return nil
}

func (h *PutAdminFineGrainedPermissions) putKeycloakClientAdminFineGrainedPermissions(ctx context.Context, keycloakIDP *keycloakApi.KeycloakRealmIdentityProvider, realmName string) error {
	reqLog := ctrl.LoggerFrom(ctx)
	reqLog.Info("Start put keycloak idp admin fine grained permissions")

	managementPermissions := keycloakv2.ManagementPermissionReference{
		Enabled: &keycloakIDP.Spec.AdminFineGrainedPermissionsEnabled,
	}

	if _, _, err := h.kClient.IdentityProviders.UpdateIDPManagementPermissions(ctx, realmName, keycloakIDP.Spec.Alias, managementPermissions); err != nil {
		return fmt.Errorf("unable to update idp management permissions: %w", err)
	}

	reqLog.Info("End put keycloak idp admin fine grained permissions")

	return nil
}

func (h *PutAdminFineGrainedPermissions) putKeycloakIDPAdminPermissionPolicies(ctx context.Context, keycloakIDP *keycloakApi.KeycloakRealmIdentityProvider, realmName string) error {
	reqLog := ctrl.LoggerFrom(ctx)
	reqLog.Info("Start put keycloak idp admin permission policies")

	idp, _, err := h.kClient.IdentityProviders.GetIdentityProvider(ctx, realmName, keycloakIDP.Spec.Alias)
	if err != nil {
		return fmt.Errorf("failed to get idp: %w", err)
	}

	realmManagementClientUUID, err := h.kClient.Clients.GetClientUUID(ctx, realmName, keycloakv2.RealmManagementClient)
	if err != nil {
		return fmt.Errorf("failed to get %s client id: %w", keycloakv2.RealmManagementClient, err)
	}

	realmManagementPermissionsList, _, err := h.kClient.Authorization.GetPermissions(ctx, realmName, realmManagementClientUUID)
	if err != nil {
		return fmt.Errorf("failed to get permissions for %s client: %w", keycloakv2.RealmManagementClient, err)
	}

	realmManagementPermissions := maputil.SliceToMapSelf(realmManagementPermissionsList, func(p keycloakv2.AbstractPolicyRepresentation) (string, bool) {
		return ptr.Deref(p.Name, ""), p.Name != nil
	})

	existingIDPPermissions, _, err := h.kClient.IdentityProviders.GetIDPManagementPermissions(ctx, realmName, keycloakIDP.Spec.Alias)
	if err != nil {
		return fmt.Errorf("failed to get idp permissions: %w", err)
	}

	if existingIDPPermissions == nil || existingIDPPermissions.ScopePermissions == nil {
		return fmt.Errorf("idp management permissions or scope permissions are nil")
	}

	existingScopePermissions := *existingIDPPermissions.ScopePermissions

	for i := 0; i < len(keycloakIDP.Spec.Permission.ScopePermissions); i++ {
		name := keycloakIDP.Spec.Permission.ScopePermissions[i].Name
		reqLog.Info("Processing scope permission", scopeLogKey, name)

		if _, ok := existingScopePermissions[name]; !ok {
			return fmt.Errorf("scope %s not found in permissions", name)
		}

		permissionName := fmt.Sprintf("%s.permission.idp.%s", name, ptr.Deref(idp.InternalId, ""))

		if permission, ok := realmManagementPermissions[permissionName]; ok {
			if permission.Id == nil {
				continue
			}

			policies := keycloakIDP.Spec.Permission.ScopePermissions[i].Policies
			updatedPermission := keycloakv2.PolicyRepresentation{
				Id:       permission.Id,
				Name:     permission.Name,
				Type:     permission.Type,
				Policies: &policies,
			}

			permType := ptr.Deref(permission.Type, "")

			if _, err = h.kClient.Authorization.UpdatePermission(ctx, realmName, realmManagementClientUUID, permType, *permission.Id, updatedPermission); err != nil {
				return fmt.Errorf("failed to update permission %s: %w", permissionName, err)
			}

			reqLog.Info("Scope permission updated", scopeLogKey, name, permissionLogKey, permissionName)
		}
	}

	reqLog.Info("End put keycloak idp admin permission policies")

	return nil
}

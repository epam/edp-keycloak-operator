package chain

import (
	"context"
	"fmt"

	"k8s.io/utils/ptr"
	ctrl "sigs.k8s.io/controller-runtime"

	keycloakApi "github.com/epam/edp-keycloak-operator/api/v1"
	"github.com/epam/edp-keycloak-operator/pkg/client/keycloakapi"
	"github.com/epam/edp-keycloak-operator/pkg/maputil"
)

const (
	scopeLogKey      = "scope"
	permissionLogKey = "permission"
)

type PutAdminFineGrainedPermissions struct {
	kClient *keycloakapi.KeycloakClient
}

func NewPutAdminFineGrainedPermissions(kClient *keycloakapi.KeycloakClient) *PutAdminFineGrainedPermissions {
	return &PutAdminFineGrainedPermissions{kClient: kClient}
}

func (h *PutAdminFineGrainedPermissions) Serve(ctx context.Context, keycloakIDP *keycloakApi.KeycloakRealmIdentityProvider, realmName string) error {
	featureFlagEnabled, err := h.kClient.Server.FeatureFlagEnabled(ctx, keycloakapi.FeatureFlagAdminFineGrainedAuthz)
	if err != nil {
		return fmt.Errorf("failed to check feature flag ADMIN_FINE_GRAINED_AUTHZ: %w", err)
	}

	if !featureFlagEnabled {
		log := ctrl.LoggerFrom(ctx)
		log.Info("Feature flag is not enabled, skipping admin fine grained permissions", "featureFlag", keycloakapi.FeatureFlagAdminFineGrainedAuthz)

		return nil
	}

	current, err := h.putKeycloakClientAdminFineGrainedPermissions(ctx, keycloakIDP, realmName)
	if err != nil {
		return fmt.Errorf("unable to put keycloak idp admin fine grained permissions: %w", err)
	}

	if keycloakIDP.Spec.AdminFineGrainedPermissionsEnabled && keycloakIDP.Spec.Permission != nil {
		if err = h.putKeycloakIDPAdminPermissionPolicies(ctx, keycloakIDP, realmName, current); err != nil {
			return fmt.Errorf("unable to put keycloak idp admin permission policies: %w", err)
		}
	}

	return nil
}

// putKeycloakClientAdminFineGrainedPermissions returns the current management permissions state
// (fetched, or the fresh state from the PUT response when an update was needed) so the caller can
// reuse it instead of re-fetching.
func (h *PutAdminFineGrainedPermissions) putKeycloakClientAdminFineGrainedPermissions(
	ctx context.Context, keycloakIDP *keycloakApi.KeycloakRealmIdentityProvider, realmName string,
) (*keycloakapi.ManagementPermissionReference, error) {
	reqLog := ctrl.LoggerFrom(ctx)
	reqLog.Info("Start put keycloak idp admin fine grained permissions")

	current, _, err := h.kClient.IdentityProviders.GetIDPManagementPermissions(ctx, realmName, keycloakIDP.Spec.Alias)
	if err != nil {
		return nil, fmt.Errorf("failed to get idp management permissions: %w", err)
	}

	if current != nil && ptr.Deref(current.Enabled, false) == keycloakIDP.Spec.AdminFineGrainedPermissionsEnabled {
		reqLog.Info("Idp admin fine grained permissions are already in sync, skipping update")
		return current, nil
	}

	managementPermissions := keycloakapi.ManagementPermissionReference{
		Enabled: &keycloakIDP.Spec.AdminFineGrainedPermissionsEnabled,
	}

	updated, _, err := h.kClient.IdentityProviders.UpdateIDPManagementPermissions(ctx, realmName, keycloakIDP.Spec.Alias, managementPermissions)
	if err != nil {
		return nil, fmt.Errorf("unable to update idp management permissions: %w", err)
	}

	reqLog.Info("End put keycloak idp admin fine grained permissions")

	return updated, nil
}

func (h *PutAdminFineGrainedPermissions) putKeycloakIDPAdminPermissionPolicies(
	ctx context.Context,
	keycloakIDP *keycloakApi.KeycloakRealmIdentityProvider,
	realmName string,
	existingIDPPermissions *keycloakapi.ManagementPermissionReference,
) error {
	reqLog := ctrl.LoggerFrom(ctx)
	reqLog.Info("Start put keycloak idp admin permission policies")

	idp, _, err := h.kClient.IdentityProviders.GetIdentityProvider(ctx, realmName, keycloakIDP.Spec.Alias)
	if err != nil {
		return fmt.Errorf("failed to get idp: %w", err)
	}

	realmManagementClientUUID, err := h.kClient.Clients.GetClientUUID(ctx, realmName, keycloakapi.RealmManagementClient)
	if err != nil {
		return fmt.Errorf("failed to get %s client id: %w", keycloakapi.RealmManagementClient, err)
	}

	realmManagementPermissionsList, _, err := h.kClient.Authorization.GetPermissions(ctx, realmName, realmManagementClientUUID)
	if err != nil {
		return fmt.Errorf("failed to get permissions for %s client: %w", keycloakapi.RealmManagementClient, err)
	}

	realmManagementPermissions := maputil.SliceToMapSelf(realmManagementPermissionsList, func(p keycloakapi.AbstractPolicyRepresentation) (string, bool) {
		return ptr.Deref(p.Name, ""), p.Name != nil
	})

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
			updatedPermission := keycloakapi.PolicyRepresentation{
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

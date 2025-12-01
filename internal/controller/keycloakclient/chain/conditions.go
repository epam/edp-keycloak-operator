package chain

import (
	"context"
	"fmt"

	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	keycloakApi "github.com/epam/edp-keycloak-operator/api/v1"
)

const (
	// ConditionReady indicates the overall readiness of the KeycloakClient.
	// This is the primary condition that summarizes all chain steps.
	ConditionReady = "Ready"

	// Individual chain step conditions - one per step
	ConditionClientSynced                        = "ClientSynced"                        // PutClient
	ConditionClientRolesSynced                   = "ClientRolesSynced"                   // PutClientRole
	ConditionRealmRolesSynced                    = "RealmRolesSynced"                    // PutRealmRole
	ConditionClientScopesSynced                  = "ClientScopesSynced"                  // PutClientScope
	ConditionProtocolMappersSynced               = "ProtocolMappersSynced"               // PutProtocolMappers
	ConditionServiceAccountSynced                = "ServiceAccountSynced"                // ServiceAccount
	ConditionAuthorizationScopesSynced           = "AuthorizationScopesSynced"           // ProcessScope
	ConditionAuthorizationResourcesSynced        = "AuthorizationResourcesSynced"        // ProcessResources
	ConditionAuthorizationPoliciesSynced         = "AuthorizationPoliciesSynced"         // ProcessPolicy
	ConditionAuthorizationPermissionsSynced      = "AuthorizationPermissionsSynced"      // ProcessPermissions
	ConditionAdminFineGrainedPermissionsV1Synced = "AdminFineGrainedPermissionsV1Synced" // PutAdminFineGrainedPermissions

	// Success reasons - one per step
	ReasonClientCreated                       = "ClientCreated"
	ReasonClientUpdated                       = "ClientUpdated"
	ReasonClientRolesSynced                   = "ClientRolesSynced"
	ReasonRealmRolesSynced                    = "RealmRolesSynced"
	ReasonClientScopesSynced                  = "ClientScopesSynced"
	ReasonProtocolMappersSynced               = "ProtocolMappersSynced"
	ReasonServiceAccountSynced                = "ServiceAccountSynced"
	ReasonAuthorizationScopesSynced           = "AuthorizationScopesSynced"
	ReasonAuthorizationResourcesSynced        = "AuthorizationResourcesSynced"
	ReasonAuthorizationPoliciesSynced         = "AuthorizationPoliciesSynced"
	ReasonAuthorizationPermissionsSynced      = "AuthorizationPermissionsSynced"
	ReasonAdminFineGrainedPermissionsV1Synced = "AdminFineGrainedPermissionsV1Synced"
	ReasonReconciliationSucceeded             = "ReconciliationSucceeded"

	// Failure reasons - generic
	ReasonKeycloakAPIError   = "KeycloakAPIError"
	ReasonConfigurationError = "ConfigurationError"
	ReasonSecretError        = "SecretError"

	// Skipped reasons (for addOnly strategy or not configured)
	ReasonSkippedAddOnly = "SkippedAddOnly"
	ReasonNotConfigured  = "NotConfigured"
)

// SetCondition is a helper to set a condition on KeycloakClient and update status.
// Each chain step calls this when it succeeds or fails.
func SetCondition(
	ctx context.Context,
	k8sClient client.Client,
	keycloakClient *keycloakApi.KeycloakClient,
	conditionType string,
	status metav1.ConditionStatus,
	reason string,
	message string,
) error {
	if changed := meta.SetStatusCondition(&keycloakClient.Status.Conditions, metav1.Condition{
		Type:               conditionType,
		Status:             status,
		Reason:             reason,
		Message:            message,
		ObservedGeneration: keycloakClient.Generation,
	}); !changed {
		return nil
	}

	if err := k8sClient.Status().Update(ctx, keycloakClient); err != nil {
		return fmt.Errorf("failed to update condition %s: %w", conditionType, err)
	}

	return nil
}

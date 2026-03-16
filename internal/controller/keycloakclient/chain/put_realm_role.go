package chain

import (
	"context"
	"fmt"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	keycloakApi "github.com/epam/edp-keycloak-operator/api/v1"
	keycloakv2 "github.com/epam/edp-keycloak-operator/pkg/client/keycloakv2"
)

type PutRealmRole struct {
	kClient   *keycloakv2.KeycloakClient
	k8sClient client.Client
}

func NewPutRealmRole(kClient *keycloakv2.KeycloakClient, k8sClient client.Client) *PutRealmRole {
	return &PutRealmRole{kClient: kClient, k8sClient: k8sClient}
}

func (h *PutRealmRole) Serve(ctx context.Context, keycloakClient *keycloakApi.KeycloakClient, realmName string, _ *ClientContext) error {
	if err := h.putRealmRoles(ctx, keycloakClient, realmName); err != nil {
		h.setFailureCondition(ctx, keycloakClient, fmt.Sprintf("Failed to sync realm roles: %s", err.Error()))

		return fmt.Errorf("unable to put realm roles: %w", err)
	}

	h.setSuccessCondition(ctx, keycloakClient, "Realm roles synchronized")

	return nil
}

func (h *PutRealmRole) setFailureCondition(ctx context.Context, keycloakClient *keycloakApi.KeycloakClient, message string) {
	log := ctrl.LoggerFrom(ctx)

	if err := SetCondition(
		ctx, h.k8sClient, keycloakClient,
		ConditionRealmRolesSynced,
		metav1.ConditionFalse,
		ReasonKeycloakAPIError,
		message,
	); err != nil {
		log.Error(err, "Failed to set failure condition")
	}
}

func (h *PutRealmRole) setSuccessCondition(ctx context.Context, keycloakClient *keycloakApi.KeycloakClient, message string) {
	log := ctrl.LoggerFrom(ctx)

	if err := SetCondition(
		ctx, h.k8sClient, keycloakClient,
		ConditionRealmRolesSynced,
		metav1.ConditionTrue,
		ReasonRealmRolesSynced,
		message,
	); err != nil {
		log.Error(err, "Failed to set success condition")
	}
}

func (h *PutRealmRole) putRealmRoles(ctx context.Context, keycloakClient *keycloakApi.KeycloakClient, realmName string) error {
	reqLog := ctrl.LoggerFrom(ctx)
	reqLog.Info("Start put realm roles")

	if keycloakClient.Spec.RealmRoles == nil || len(*keycloakClient.Spec.RealmRoles) == 0 {
		reqLog.Info("Keycloak client does not have realm roles")
		return nil
	}

	for _, role := range *keycloakClient.Spec.RealmRoles {
		_, _, err := h.kClient.Roles.GetRealmRole(ctx, realmName, role.Name)
		if err == nil {
			reqLog.Info("Realm role already exists", "role", role.Name)
			continue
		}

		if !keycloakv2.IsNotFound(err) {
			return fmt.Errorf("error checking realm role %s: %w", role.Name, err)
		}

		// Role doesn't exist, create it
		composite := role.Composite != ""
		newRole := keycloakv2.RoleRepresentation{
			Name:      ptr.To(role.Name),
			Composite: &composite,
		}

		if _, err := h.kClient.Roles.CreateRealmRole(ctx, realmName, newRole); err != nil {
			return fmt.Errorf("error creating realm role %s: %w", role.Name, err)
		}

		// If composite, add the composite role
		if composite {
			compositeRole, _, err := h.kClient.Roles.GetRealmRole(ctx, realmName, role.Composite)
			if err != nil {
				return fmt.Errorf("error getting composite realm role %s: %w", role.Composite, err)
			}

			if _, err := h.kClient.Roles.AddRealmRoleComposites(ctx, realmName, role.Name, []keycloakv2.RoleRepresentation{*compositeRole}); err != nil {
				return fmt.Errorf("error adding composite to realm role %s: %w", role.Name, err)
			}
		}
	}

	reqLog.Info("End put realm roles")

	return nil
}

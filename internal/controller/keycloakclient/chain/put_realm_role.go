package chain

import (
	"context"
	"fmt"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	keycloakApi "github.com/epam/edp-keycloak-operator/api/v1"
	"github.com/epam/edp-keycloak-operator/pkg/client/keycloak"
	"github.com/epam/edp-keycloak-operator/pkg/client/keycloak/dto"
)

type PutRealmRole struct {
	keycloakApiClient keycloak.Client
	k8sClient         client.Client
}

func NewPutRealmRole(keycloakApiClient keycloak.Client, k8sClient client.Client) *PutRealmRole {
	return &PutRealmRole{keycloakApiClient: keycloakApiClient, k8sClient: k8sClient}
}

func (el *PutRealmRole) Serve(ctx context.Context, keycloakClient *keycloakApi.KeycloakClient, realmName string) error {
	if err := el.putRealmRoles(ctx, keycloakClient, realmName); err != nil {
		el.setFailureCondition(ctx, keycloakClient, fmt.Sprintf("Failed to sync realm roles: %s", err.Error()))

		return fmt.Errorf("unable to put realm roles: %w", err)
	}

	el.setSuccessCondition(ctx, keycloakClient, "Realm roles synchronized")

	return nil
}

func (el *PutRealmRole) setFailureCondition(ctx context.Context, keycloakClient *keycloakApi.KeycloakClient, message string) {
	log := ctrl.LoggerFrom(ctx)

	if err := SetCondition(
		ctx, el.k8sClient, keycloakClient,
		ConditionRealmRolesSynced,
		metav1.ConditionFalse,
		ReasonKeycloakAPIError,
		message,
	); err != nil {
		log.Error(err, "Failed to set failure condition")
	}
}

func (el *PutRealmRole) setSuccessCondition(ctx context.Context, keycloakClient *keycloakApi.KeycloakClient, message string) {
	log := ctrl.LoggerFrom(ctx)

	if err := SetCondition(
		ctx, el.k8sClient, keycloakClient,
		ConditionRealmRolesSynced,
		metav1.ConditionTrue,
		ReasonRealmRolesSynced,
		message,
	); err != nil {
		log.Error(err, "Failed to set success condition")
	}
}

func (el *PutRealmRole) putRealmRoles(ctx context.Context, keycloakClient *keycloakApi.KeycloakClient, realmName string) error {
	reqLog := ctrl.LoggerFrom(ctx)
	reqLog.Info("Start put realm roles")

	if keycloakClient.Spec.RealmRoles == nil || len(*keycloakClient.Spec.RealmRoles) == 0 {
		reqLog.Info("Keycloak client does not have realm roles")
		return nil
	}

	for _, role := range *keycloakClient.Spec.RealmRoles {
		roleDto := &dto.IncludedRealmRole{
			Name:      role.Name,
			Composite: role.Composite,
		}

		exist, err := el.keycloakApiClient.ExistRealmRole(realmName, roleDto.Name)
		if err != nil {
			return fmt.Errorf("error during ExistRealmRole: %w", err)
		}

		if exist {
			reqLog.Info("Client already exists")
			return nil
		}

		err = el.keycloakApiClient.CreateIncludedRealmRole(realmName, roleDto)
		if err != nil {
			return fmt.Errorf("error during CreateRealmRole: %w", err)
		}
	}

	reqLog.Info("End put realm roles")

	return nil
}

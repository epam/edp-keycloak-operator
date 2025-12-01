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

type PutClientRole struct {
	keycloakApiClient keycloak.Client
	k8sClient         client.Client
}

func NewPutClientRole(keycloakApiClient keycloak.Client, k8sClient client.Client) *PutClientRole {
	return &PutClientRole{keycloakApiClient: keycloakApiClient, k8sClient: k8sClient}
}

func (el *PutClientRole) Serve(ctx context.Context, keycloakClient *keycloakApi.KeycloakClient, realmName string) error {
	if err := el.putKeycloakClientRole(ctx, keycloakClient, realmName); err != nil {
		el.setFailureCondition(ctx, keycloakClient, fmt.Sprintf("Failed to sync client roles: %s", err.Error()))

		return fmt.Errorf("unable to put keycloak client role: %w", err)
	}

	el.setSuccessCondition(ctx, keycloakClient, "Client roles synchronized")

	return nil
}

func (el *PutClientRole) setFailureCondition(ctx context.Context, keycloakClient *keycloakApi.KeycloakClient, message string) {
	log := ctrl.LoggerFrom(ctx)

	if err := SetCondition(
		ctx, el.k8sClient, keycloakClient,
		ConditionClientRolesSynced,
		metav1.ConditionFalse,
		ReasonKeycloakAPIError,
		message,
	); err != nil {
		log.Error(err, "Failed to set failure condition")
	}
}

func (el *PutClientRole) setSuccessCondition(ctx context.Context, keycloakClient *keycloakApi.KeycloakClient, message string) {
	log := ctrl.LoggerFrom(ctx)

	if err := SetCondition(
		ctx, el.k8sClient, keycloakClient,
		ConditionClientRolesSynced,
		metav1.ConditionTrue,
		ReasonClientRolesSynced,
		message,
	); err != nil {
		log.Error(err, "Failed to set success condition")
	}
}

func (el *PutClientRole) putKeycloakClientRole(ctx context.Context, keycloakClient *keycloakApi.KeycloakClient, realmName string) error {
	reqLog := ctrl.LoggerFrom(ctx)
	reqLog.Info("Start put keycloak client role")

	clientDto := dto.ConvertSpecToClient(&keycloakClient.Spec, "", realmName, nil)

	if err := el.keycloakApiClient.SyncClientRoles(ctx, realmName, clientDto); err != nil {
		return fmt.Errorf("unable to sync client roles: %w", err)
	}

	reqLog.Info("End put keycloak client role")

	return nil
}

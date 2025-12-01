package chain

import (
	"context"
	"errors"
	"fmt"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	keycloakApi "github.com/epam/edp-keycloak-operator/api/v1"
	"github.com/epam/edp-keycloak-operator/pkg/client/keycloak"
)

type ServiceAccount struct {
	keycloakApiClient keycloak.Client
	k8sClient         client.Client
}

func NewServiceAccount(keycloakApiClient keycloak.Client, k8sClient client.Client) *ServiceAccount {
	return &ServiceAccount{keycloakApiClient: keycloakApiClient, k8sClient: k8sClient}
}

func (el *ServiceAccount) Serve(ctx context.Context, keycloakClient *keycloakApi.KeycloakClient, realmName string) error {
	if keycloakClient.Spec.ServiceAccount == nil || !keycloakClient.Spec.ServiceAccount.Enabled {
		return nil
	}

	if keycloakClient.Spec.ServiceAccount != nil && keycloakClient.Spec.Public {
		return errors.New("service account can not be configured with public client")
	}

	clientRoles := make(map[string][]string)
	for _, v := range keycloakClient.Spec.ServiceAccount.ClientRoles {
		clientRoles[v.ClientID] = v.Roles
	}

	addOnly := keycloakClient.GetReconciliationStrategy() == keycloakApi.ReconciliationStrategyAddOnly

	if err := el.keycloakApiClient.SyncServiceAccountRoles(realmName,
		keycloakClient.Status.ClientID, keycloakClient.Spec.ServiceAccount.RealmRoles, clientRoles, addOnly); err != nil {
		el.setFailureCondition(ctx, keycloakClient, fmt.Sprintf("Failed to sync service account: %s", err.Error()))

		return fmt.Errorf("unable to sync service account roles: %w", err)
	}

	if keycloakClient.Spec.ServiceAccount.Groups != nil {
		if err := el.keycloakApiClient.SyncServiceAccountGroups(realmName,
			keycloakClient.Status.ClientID, keycloakClient.Spec.ServiceAccount.Groups, addOnly); err != nil {
			el.setFailureCondition(ctx, keycloakClient, fmt.Sprintf("Failed to sync service account: %s", err.Error()))

			return fmt.Errorf("unable to sync service account groups: %w", err)
		}
	}

	if keycloakClient.Spec.ServiceAccount.AttributesV2 != nil {
		if err := el.keycloakApiClient.SetServiceAccountAttributes(realmName, keycloakClient.Status.ClientID,
			keycloakClient.Spec.ServiceAccount.AttributesV2, addOnly); err != nil {
			el.setFailureCondition(ctx, keycloakClient, fmt.Sprintf("Failed to sync service account: %s", err.Error()))

			return fmt.Errorf("unable to set service account attributes: %w", err)
		}
	}

	el.setSuccessCondition(ctx, keycloakClient, "Service account synchronized")

	return nil
}

func (el *ServiceAccount) setFailureCondition(ctx context.Context, keycloakClient *keycloakApi.KeycloakClient, message string) {
	log := ctrl.LoggerFrom(ctx)

	if err := SetCondition(
		ctx, el.k8sClient, keycloakClient,
		ConditionServiceAccountSynced,
		metav1.ConditionFalse,
		ReasonKeycloakAPIError,
		message,
	); err != nil {
		log.Error(err, "Failed to set failure condition")
	}
}

func (el *ServiceAccount) setSuccessCondition(ctx context.Context, keycloakClient *keycloakApi.KeycloakClient, message string) {
	log := ctrl.LoggerFrom(ctx)

	if err := SetCondition(
		ctx, el.k8sClient, keycloakClient,
		ConditionServiceAccountSynced,
		metav1.ConditionTrue,
		ReasonServiceAccountSynced,
		message,
	); err != nil {
		log.Error(err, "Failed to set success condition")
	}
}

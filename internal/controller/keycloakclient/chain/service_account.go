package chain

import (
	"context"

	"github.com/pkg/errors"

	keycloakApi "github.com/epam/edp-keycloak-operator/api/v1"
	"github.com/epam/edp-keycloak-operator/pkg/client/keycloak"
)

type ServiceAccount struct {
	keycloakApiClient keycloak.Client
}

func NewServiceAccount(keycloakApiClient keycloak.Client) *ServiceAccount {
	return &ServiceAccount{keycloakApiClient: keycloakApiClient}
}

func (el *ServiceAccount) Serve(_ context.Context, keycloakClient *keycloakApi.KeycloakClient, realmName string) error {
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
		return errors.Wrap(err, "unable to sync service account roles")
	}

	if keycloakClient.Spec.ServiceAccount.Attributes != nil {
		if err := el.keycloakApiClient.SetServiceAccountAttributes(realmName, keycloakClient.Status.ClientID,
			keycloakClient.Spec.ServiceAccount.Attributes, addOnly); err != nil {
			return errors.Wrap(err, "unable to set service account attributes")
		}
	}

	if keycloakClient.Spec.ServiceAccount.Groups != nil {
		if err := el.keycloakApiClient.SetServiceAccountGroups(realmName,
			keycloakClient.Status.ClientID, keycloakClient.Spec.ServiceAccount.Groups, addOnly); err != nil {
			return errors.Wrap(err, "unable to sync service account groups")
		}
	}

	return nil
}

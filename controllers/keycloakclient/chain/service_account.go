package chain

import (
	"context"

	"github.com/pkg/errors"

	keycloakApi "github.com/epam/edp-keycloak-operator/api/v1"
	"github.com/epam/edp-keycloak-operator/pkg/client/keycloak"
)

type ServiceAccount struct {
	BaseElement
	next Element
}

func (el *ServiceAccount) Serve(ctx context.Context, keycloakClient *keycloakApi.KeycloakClient, adapterClient keycloak.Client, realmName string) error {
	if keycloakClient.Spec.ServiceAccount == nil || !keycloakClient.Spec.ServiceAccount.Enabled {
		return el.NextServeOrNil(ctx, el.next, keycloakClient, adapterClient, realmName)
	}

	if keycloakClient.Spec.ServiceAccount != nil && keycloakClient.Spec.Public {
		return errors.New("service account can not be configured with public client")
	}

	clientRoles := make(map[string][]string)
	for _, v := range keycloakClient.Spec.ServiceAccount.ClientRoles {
		clientRoles[v.ClientID] = v.Roles
	}

	addOnly := keycloakClient.GetReconciliationStrategy() == keycloakApi.ReconciliationStrategyAddOnly

	if err := adapterClient.SyncServiceAccountRoles(realmName,
		keycloakClient.Status.ClientID, keycloakClient.Spec.ServiceAccount.RealmRoles, clientRoles, addOnly); err != nil {
		return errors.Wrap(err, "unable to sync service account roles")
	}

	if keycloakClient.Spec.ServiceAccount.Attributes != nil {
		if err := adapterClient.SetServiceAccountAttributes(realmName, keycloakClient.Status.ClientID,
			keycloakClient.Spec.ServiceAccount.Attributes, addOnly); err != nil {
			return errors.Wrap(err, "unable to set service account attributes")
		}
	}

	return el.NextServeOrNil(ctx, el.next, keycloakClient, adapterClient, realmName)
}

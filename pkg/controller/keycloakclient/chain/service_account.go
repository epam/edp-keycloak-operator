package chain

import (
	v1v1alpha1 "github.com/epam/edp-keycloak-operator/pkg/apis/v1/v1alpha1"
	"github.com/epam/edp-keycloak-operator/pkg/client/keycloak"
	"github.com/pkg/errors"
)

type ServiceAccount struct {
	BaseElement
	next Element
}

func (el *ServiceAccount) Serve(keycloakClient *v1v1alpha1.KeycloakClient, adapterClient keycloak.Client) error {
	if keycloakClient.Spec.ServiceAccount == nil || !keycloakClient.Spec.ServiceAccount.Enabled {
		return el.NextServeOrNil(el.next, keycloakClient, adapterClient)
	}

	if keycloakClient.Spec.ServiceAccount != nil && keycloakClient.Spec.Public {
		return errors.New("service account can not be configured with public client")
	}

	clientRoles := make(map[string][]string)
	for _, v := range keycloakClient.Spec.ServiceAccount.ClientRoles {
		clientRoles[v.ClientID] = v.Roles
	}

	if err := adapterClient.SyncServiceAccountRoles(keycloakClient.Spec.TargetRealm,
		keycloakClient.Status.ClientID, keycloakClient.Spec.ServiceAccount.RealmRoles, clientRoles); err != nil {
		return errors.Wrap(err, "unable to sync service account roles")
	}

	if keycloakClient.Spec.ServiceAccount.Attributes != nil {
		if err := adapterClient.SetServiceAccountAttributes(keycloakClient.Spec.TargetRealm, keycloakClient.Status.ClientID,
			keycloakClient.Spec.ServiceAccount.Attributes); err != nil {
			return errors.Wrap(err, "unable to set service account attributes")
		}
	}

	return el.NextServeOrNil(el.next, keycloakClient, adapterClient)
}

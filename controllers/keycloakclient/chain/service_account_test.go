package chain

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	keycloakApi "github.com/epam/edp-keycloak-operator/api/v1"
	"github.com/epam/edp-keycloak-operator/pkg/client/keycloak/adapter"
)

func TestServiceAccount_Serve(t *testing.T) {
	sa := ServiceAccount{}

	kc := keycloakApi.KeycloakClient{
		Spec: keycloakApi.KeycloakClientSpec{
			TargetRealm: "realm1",
			ServiceAccount: &keycloakApi.ServiceAccount{
				Enabled: true,
				Attributes: map[string]string{
					"foo": "bar",
				},
				ClientRoles: []keycloakApi.ClientRole{
					{
						ClientID: "clid2",
						Roles:    []string{"foo", "bar"},
					},
				},
				RealmRoles: []string{"baz", "zaz"},
			},
		},
		Status: keycloakApi.KeycloakClientStatus{
			ClientID: "clid1",
		},
	}
	kClient := new(adapter.Mock)

	kClient.On("SyncServiceAccountRoles", kc.Spec.TargetRealm, kc.Status.ClientID,
		kc.Spec.ServiceAccount.RealmRoles,
		map[string][]string{
			kc.Spec.ServiceAccount.ClientRoles[0].ClientID: kc.Spec.ServiceAccount.ClientRoles[0].Roles}, false).Return(nil)
	kClient.On("SetServiceAccountAttributes", kc.Spec.TargetRealm, kc.Status.ClientID,
		kc.Spec.ServiceAccount.Attributes, false).Return(nil)

	err := sa.Serve(context.Background(), &kc, kClient)
	require.NoError(t, err)
}

package chain

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/epam/edp-keycloak-operator/api/common"
	keycloakApi "github.com/epam/edp-keycloak-operator/api/v1"
	"github.com/epam/edp-keycloak-operator/pkg/client/keycloak/mocks"
)

func TestServiceAccount_Serve(t *testing.T) {
	kc := keycloakApi.KeycloakClient{
		Spec: keycloakApi.KeycloakClientSpec{
			RealmRef: common.RealmRef{
				Kind: keycloakApi.KeycloakRealmKind,
				Name: "realm",
			},
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
				Groups:     []string{"group1", "group2"},
			},
		},
		Status: keycloakApi.KeycloakClientStatus{
			ClientID: "clid1",
		},
	}

	realmName := "realm"
	apiClient := mocks.NewMockClient(t)

	apiClient.On("SyncServiceAccountRoles", realmName, kc.Status.ClientID,
		kc.Spec.ServiceAccount.RealmRoles,
		map[string][]string{
			kc.Spec.ServiceAccount.ClientRoles[0].ClientID: kc.Spec.ServiceAccount.ClientRoles[0].Roles}, false).Return(nil)
	apiClient.On("SetServiceAccountAttributes", realmName, kc.Status.ClientID,
		kc.Spec.ServiceAccount.Attributes, false).Return(nil)
	apiClient.On("SetServiceAccountGroups", realmName, kc.Status.ClientID,
		kc.Spec.ServiceAccount.Groups, false).Return(nil)

	sa := NewServiceAccount(apiClient)

	err := sa.Serve(context.Background(), &kc, realmName)
	require.NoError(t, err)
}

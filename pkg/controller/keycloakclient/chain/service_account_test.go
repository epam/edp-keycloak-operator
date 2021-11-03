package chain

import (
	"testing"

	"github.com/epam/edp-keycloak-operator/pkg/apis/v1/v1alpha1"
	"github.com/epam/edp-keycloak-operator/pkg/client/keycloak/adapter"
)

func TestServiceAccount_Serve(t *testing.T) {
	sa := ServiceAccount{}

	kc := v1alpha1.KeycloakClient{
		Spec: v1alpha1.KeycloakClientSpec{
			TargetRealm: "realm1",
			ServiceAccount: &v1alpha1.ServiceAccount{
				Enabled: true,
				Attributes: map[string]string{
					"foo": "bar",
				},
				ClientRoles: []v1alpha1.ClientRole{
					{
						ClientID: "clid2",
						Roles:    []string{"foo", "bar"},
					},
				},
				RealmRoles: []string{"baz", "zaz"},
			},
		},
		Status: v1alpha1.KeycloakClientStatus{
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

	if err := sa.Serve(&kc, kClient); err != nil {
		t.Fatal(err)
	}
}

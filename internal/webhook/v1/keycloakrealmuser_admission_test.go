package v1

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	k8sErrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/epam/edp-keycloak-operator/api/common"
	keycloakApi "github.com/epam/edp-keycloak-operator/api/v1"
)

var _ = Describe("KeycloakRealmUser clientRoles admission", func() {
	newUser := func(name, clientID string) *keycloakApi.KeycloakRealmUser {
		return &keycloakApi.KeycloakRealmUser{
			ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: "ns1"},
			Spec: keycloakApi.KeycloakRealmUserSpec{
				Username: name,
				RealmRef: common.RealmRef{Kind: keycloakApi.KeycloakRealmKind, Name: "test-realm"},
				ClientRoles: []keycloakApi.UserClientRole{
					{ClientID: clientID, Roles: ptr.To([]string{"manage-account"})},
				},
			},
		}
	}

	It("Should reject an empty clientId in clientRoles", func() {
		err := k8sClient.Create(ctx, newUser("user-empty-client-id", ""))
		Expect(k8sErrors.IsInvalid(err)).To(BeTrue(), "got: %v", err)
		Expect(err.Error()).To(ContainSubstring("spec.clientRoles[0].clientId"))
	})

	It("Should accept a non-empty clientId and an update to another valid clientId", func() {
		user := newUser("user-valid-client-id", "account")
		Expect(k8sClient.Create(ctx, user)).To(Succeed())

		DeferCleanup(func() { Expect(client.IgnoreNotFound(k8sClient.Delete(ctx, user))).To(Succeed()) })

		user.Spec.ClientRoles[0].ClientID = "realm-management"
		Expect(k8sClient.Update(ctx, user)).To(Succeed())
	})
})

package keycloakclientscope

import (
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	k8sErrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	"github.com/epam/edp-keycloak-operator/api/common"
	keycloakApi "github.com/epam/edp-keycloak-operator/api/v1"
)

var _ = Describe("KeycloakClientScope controller", Ordered, func() {
	It("Should create KeycloakClientScope", func() {
		By("Creating a KeycloakClientScope")
		scope := &keycloakApi.KeycloakClientScope{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-client-scope",
				Namespace: ns,
			},
			Spec: keycloakApi.KeycloakClientScopeSpec{
				Name: "groups",
				RealmRef: common.RealmRef{
					Kind: keycloakApi.KeycloakRealmKind,
					Name: KeycloakRealmCR,
				},
				Protocol:    "openid-connect",
				Description: "Group Membership",
				Attributes:  map[string]string{"include.in.token.scope": "true"},
				Default:     false,
				ProtocolMappers: []keycloakApi.ProtocolMapper{{
					Name:           "groups",
					Protocol:       "openid-connect",
					ProtocolMapper: "oidc-group-membership-mapper",
					Config: map[string]string{
						"access.token.claim":   "true",
						"claim.name":           "groups",
						"full.path":            "false",
						"id.token.claim":       "true",
						"userinfo.token.claim": "true",
					},
				}},
			},
		}
		Expect(k8sClient.Create(ctx, scope)).Should(Succeed())
		Eventually(func(g Gomega) {
			createdScope := &keycloakApi.KeycloakClientScope{}
			err := k8sClient.Get(ctx, types.NamespacedName{Name: scope.Name, Namespace: ns}, createdScope)
			g.Expect(err).ShouldNot(HaveOccurred())
			g.Expect(createdScope.Status.Value).Should(Equal(common.StatusOK))
		}).WithTimeout(time.Second * 20).WithPolling(time.Second).Should(Succeed())
	})
	It("Should update KeycloakClientScope", func() {
		By("Getting KeycloakClientScope")
		createdScope := &keycloakApi.KeycloakClientScope{}
		Expect(k8sClient.Get(ctx, types.NamespacedName{Namespace: ns, Name: "test-client-scope"}, createdScope)).Should(Succeed())

		By("Updating a KeycloakClientScope description")
		createdScope.Spec.Description = "new-description"

		Expect(k8sClient.Update(ctx, createdScope)).Should(Succeed())
		Consistently(func(g Gomega) {
			updatedScope := &keycloakApi.KeycloakClientScope{}
			err := k8sClient.Get(ctx, types.NamespacedName{Name: createdScope.Name, Namespace: ns}, updatedScope)
			g.Expect(err).ShouldNot(HaveOccurred())
			g.Expect(updatedScope.Status.Value).Should(Equal(common.StatusOK))
		}, time.Second*5, time.Second).Should(Succeed())
	})
	It("Should delete KeycloakClientScope", func() {
		By("Getting KeycloakClientScope")
		createdScope := &keycloakApi.KeycloakClientScope{}
		Expect(k8sClient.Get(ctx, types.NamespacedName{Namespace: ns, Name: "test-client-scope"}, createdScope)).Should(Succeed())

		By("Deleting KeycloakClientScope")
		Expect(k8sClient.Delete(ctx, createdScope)).Should(Succeed())
		Eventually(func(g Gomega) {
			deletedScope := &keycloakApi.KeycloakClientScope{}
			err := k8sClient.Get(ctx, types.NamespacedName{Name: createdScope.Name, Namespace: ns}, deletedScope)

			g.Expect(k8sErrors.IsNotFound(err)).Should(BeTrue())
		}, time.Second*30, time.Second).Should(Succeed())
	})
	It("Should fail to create KeycloakClientScope with invalid name", func() {
		By("Creating a KeycloakClientScope")
		scope := &keycloakApi.KeycloakClientScope{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-client-scope-with-invalid-name",
				Namespace: ns,
			},
			Spec: keycloakApi.KeycloakClientScopeSpec{
				RealmRef: common.RealmRef{
					Kind: keycloakApi.KeycloakRealmKind,
					Name: KeycloakRealmCR,
				},
				Name:     "invalid name with spaces",
				Protocol: "openid-connect",
			},
		}
		Expect(k8sClient.Create(ctx, scope)).Should(Succeed())

		By("Waiting for KeycloakClientScope reconciliation")
		time.Sleep(time.Second * 3)

		By("Checking KeycloakClientScope status")
		Consistently(func(g Gomega) {
			createdScope := &keycloakApi.KeycloakClientScope{}
			err := k8sClient.Get(ctx, types.NamespacedName{Name: scope.Name, Namespace: ns}, createdScope)
			g.Expect(err).ShouldNot(HaveOccurred())
			g.Expect(createdScope.Status.Value).Should(ContainSubstring("unable to sync client scope"))
		}).WithTimeout(time.Second * 3).WithPolling(time.Second).Should(Succeed())
	})
})

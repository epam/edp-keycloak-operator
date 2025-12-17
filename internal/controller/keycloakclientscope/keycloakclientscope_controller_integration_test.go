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
				Type:        keycloakApi.KeycloakClientScopeTypeOptional,
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
		}).WithTimeout(timeout).WithPolling(interval).Should(Succeed())

		By("Verifying the client scope type via API")
		Eventually(func(g Gomega) {
			adapter := keycloakAdapterManager.GetAdapter()

			// Verify scope is in optional list
			hasOptional, err := adapter.HasOptionalClientScope(ctx, KeycloakRealmCR, "groups")
			g.Expect(err).ShouldNot(HaveOccurred())
			g.Expect(hasOptional).Should(BeTrue(), "scope should be in optional scopes list")

			// Verify scope is NOT in default list
			hasDefault, err := adapter.HasDefaultClientScope(ctx, KeycloakRealmCR, "groups")
			g.Expect(err).ShouldNot(HaveOccurred())
			g.Expect(hasDefault).Should(BeFalse(), "scope should NOT be in default scopes list")
		}, timeout, interval).Should(Succeed())
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
		}, time.Second*5, interval).Should(Succeed())
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
		}, timeout, interval).Should(Succeed())
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
				Type:     keycloakApi.KeycloakClientScopeTypeOptional,
			},
		}
		Expect(k8sClient.Create(ctx, scope)).Should(Succeed())

		By("Checking KeycloakClientScope status")
		Eventually(func(g Gomega) {
			createdScope := &keycloakApi.KeycloakClientScope{}
			err := k8sClient.Get(ctx, types.NamespacedName{Name: scope.Name, Namespace: ns}, createdScope)
			g.Expect(err).ShouldNot(HaveOccurred())
			g.Expect(createdScope.Status.Value).Should(ContainSubstring("unable to sync client scope"))
		}, timeout, interval).Should(Succeed())

		By("Verifying the error status persists")
		Consistently(func(g Gomega) {
			createdScope := &keycloakApi.KeycloakClientScope{}
			err := k8sClient.Get(ctx, types.NamespacedName{Name: scope.Name, Namespace: ns}, createdScope)
			g.Expect(err).ShouldNot(HaveOccurred())
			g.Expect(createdScope.Status.Value).Should(ContainSubstring("unable to sync client scope"))
		}).WithTimeout(time.Second * 3).WithPolling(interval).Should(Succeed())
	})

	It("Should create KeycloakClientScope with type default", func() {
		By("Creating a KeycloakClientScope with type default")
		scope := &keycloakApi.KeycloakClientScope{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-client-scope-default",
				Namespace: ns,
			},
			Spec: keycloakApi.KeycloakClientScopeSpec{
				Name: "test-default-scope",
				RealmRef: common.RealmRef{
					Kind: keycloakApi.KeycloakRealmKind,
					Name: KeycloakRealmCR,
				},
				Protocol:    "openid-connect",
				Description: "Test default scope",
				Type:        keycloakApi.KeycloakClientScopeTypeDefault,
			},
		}
		Expect(k8sClient.Create(ctx, scope)).Should(Succeed())
		Eventually(func(g Gomega) {
			createdScope := &keycloakApi.KeycloakClientScope{}
			err := k8sClient.Get(ctx, types.NamespacedName{Name: scope.Name, Namespace: ns}, createdScope)
			g.Expect(err).ShouldNot(HaveOccurred())
			g.Expect(createdScope.Status.Value).Should(Equal(common.StatusOK))
		}).WithTimeout(timeout).WithPolling(interval).Should(Succeed())

		By("Verifying the client scope is in default list via API")
		Eventually(func(g Gomega) {
			adapter := keycloakAdapterManager.GetAdapter()

			// Verify scope is in default list
			hasDefault, err := adapter.HasDefaultClientScope(ctx, KeycloakRealmCR, "test-default-scope")
			g.Expect(err).ShouldNot(HaveOccurred())
			g.Expect(hasDefault).Should(BeTrue(), "scope should be in default scopes list")

			// Verify scope is NOT in optional list
			hasOptional, err := adapter.HasOptionalClientScope(ctx, KeycloakRealmCR, "test-default-scope")
			g.Expect(err).ShouldNot(HaveOccurred())
			g.Expect(hasOptional).Should(BeFalse(), "scope should NOT be in optional scopes list")
		}, timeout, interval).Should(Succeed())

		By("Deleting the KeycloakClientScope")
		Expect(k8sClient.Delete(ctx, scope)).Should(Succeed())
		Eventually(func(g Gomega) {
			deletedScope := &keycloakApi.KeycloakClientScope{}
			err := k8sClient.Get(ctx, types.NamespacedName{Name: scope.Name, Namespace: ns}, deletedScope)
			g.Expect(k8sErrors.IsNotFound(err)).Should(BeTrue())
		}, timeout, interval).Should(Succeed())
	})

	It("Should create KeycloakClientScope with type none", func() {
		By("Creating a KeycloakClientScope with type none")
		scope := &keycloakApi.KeycloakClientScope{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-client-scope-none",
				Namespace: ns,
			},
			Spec: keycloakApi.KeycloakClientScopeSpec{
				Name: "test-none-scope",
				RealmRef: common.RealmRef{
					Kind: keycloakApi.KeycloakRealmKind,
					Name: KeycloakRealmCR,
				},
				Protocol:    "openid-connect",
				Description: "Test none scope",
				Type:        keycloakApi.KeycloakClientScopeTypeNone,
			},
		}
		Expect(k8sClient.Create(ctx, scope)).Should(Succeed())
		Eventually(func(g Gomega) {
			createdScope := &keycloakApi.KeycloakClientScope{}
			err := k8sClient.Get(ctx, types.NamespacedName{Name: scope.Name, Namespace: ns}, createdScope)
			g.Expect(err).ShouldNot(HaveOccurred())
			g.Expect(createdScope.Status.Value).Should(Equal(common.StatusOK))
		}).WithTimeout(timeout).WithPolling(interval).Should(Succeed())

		By("Verifying the client scope is in neither default nor optional list via API")
		Eventually(func(g Gomega) {
			adapter := keycloakAdapterManager.GetAdapter()

			// Verify scope is NOT in default list
			hasDefault, err := adapter.HasDefaultClientScope(ctx, KeycloakRealmCR, "test-none-scope")
			g.Expect(err).ShouldNot(HaveOccurred())
			g.Expect(hasDefault).Should(BeFalse(), "scope should NOT be in default scopes list")

			// Verify scope is NOT in optional list
			hasOptional, err := adapter.HasOptionalClientScope(ctx, KeycloakRealmCR, "test-none-scope")
			g.Expect(err).ShouldNot(HaveOccurred())
			g.Expect(hasOptional).Should(BeFalse(), "scope should NOT be in optional scopes list")
		}, timeout, interval).Should(Succeed())

		By("Deleting the KeycloakClientScope")
		Expect(k8sClient.Delete(ctx, scope)).Should(Succeed())
		Eventually(func(g Gomega) {
			deletedScope := &keycloakApi.KeycloakClientScope{}
			err := k8sClient.Get(ctx, types.NamespacedName{Name: scope.Name, Namespace: ns}, deletedScope)
			g.Expect(k8sErrors.IsNotFound(err)).Should(BeTrue())
		}, timeout, interval).Should(Succeed())
	})

	It("Should update KeycloakClientScope type from optional to default", func() {
		By("Creating a KeycloakClientScope with type optional")
		scope := &keycloakApi.KeycloakClientScope{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-client-scope-update-opt-to-def",
				Namespace: ns,
			},
			Spec: keycloakApi.KeycloakClientScopeSpec{
				Name: "test-update-opt-def",
				RealmRef: common.RealmRef{
					Kind: keycloakApi.KeycloakRealmKind,
					Name: KeycloakRealmCR,
				},
				Protocol:    "openid-connect",
				Description: "Test update optional to default",
				Type:        keycloakApi.KeycloakClientScopeTypeOptional,
			},
		}
		Expect(k8sClient.Create(ctx, scope)).Should(Succeed())
		Eventually(func(g Gomega) {
			createdScope := &keycloakApi.KeycloakClientScope{}
			err := k8sClient.Get(ctx, types.NamespacedName{Name: scope.Name, Namespace: ns}, createdScope)
			g.Expect(err).ShouldNot(HaveOccurred())
			g.Expect(createdScope.Status.Value).Should(Equal(common.StatusOK))
		}).WithTimeout(timeout).WithPolling(interval).Should(Succeed())

		By("Updating KeycloakClientScope type to default")
		updatableScope := &keycloakApi.KeycloakClientScope{}
		Expect(k8sClient.Get(ctx, types.NamespacedName{Name: scope.Name, Namespace: ns}, updatableScope)).Should(Succeed())
		updatableScope.Spec.Type = keycloakApi.KeycloakClientScopeTypeDefault
		Expect(k8sClient.Update(ctx, updatableScope)).Should(Succeed())

		Eventually(func(g Gomega) {
			updatedScope := &keycloakApi.KeycloakClientScope{}
			err := k8sClient.Get(ctx, types.NamespacedName{Name: scope.Name, Namespace: ns}, updatedScope)
			g.Expect(err).ShouldNot(HaveOccurred())
			g.Expect(updatedScope.Status.Value).Should(Equal(common.StatusOK))
		}, timeout, interval).Should(Succeed())

		By("Verifying the client scope is now in default list via API")
		Eventually(func(g Gomega) {
			adapter := keycloakAdapterManager.GetAdapter()

			// Verify scope is in default list
			hasDefault, err := adapter.HasDefaultClientScope(ctx, KeycloakRealmCR, "test-update-opt-def")
			g.Expect(err).ShouldNot(HaveOccurred())
			g.Expect(hasDefault).Should(BeTrue(), "scope should be in default scopes list")

			// Verify scope is NOT in optional list
			hasOptional, err := adapter.HasOptionalClientScope(ctx, KeycloakRealmCR, "test-update-opt-def")
			g.Expect(err).ShouldNot(HaveOccurred())
			g.Expect(hasOptional).Should(BeFalse(), "scope should NOT be in optional scopes list")
		}, timeout, interval).Should(Succeed())

		By("Deleting the KeycloakClientScope")
		Expect(k8sClient.Delete(ctx, scope)).Should(Succeed())
		Eventually(func(g Gomega) {
			deletedScope := &keycloakApi.KeycloakClientScope{}
			err := k8sClient.Get(ctx, types.NamespacedName{Name: scope.Name, Namespace: ns}, deletedScope)
			g.Expect(k8sErrors.IsNotFound(err)).Should(BeTrue())
		}, timeout, interval).Should(Succeed())
	})

	It("Should update KeycloakClientScope type from default to none", func() {
		By("Creating a KeycloakClientScope with type default")
		scope := &keycloakApi.KeycloakClientScope{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-client-scope-update-def-to-none",
				Namespace: ns,
			},
			Spec: keycloakApi.KeycloakClientScopeSpec{
				Name: "test-update-def-none",
				RealmRef: common.RealmRef{
					Kind: keycloakApi.KeycloakRealmKind,
					Name: KeycloakRealmCR,
				},
				Protocol:    "openid-connect",
				Description: "Test update default to none",
				Type:        keycloakApi.KeycloakClientScopeTypeDefault,
			},
		}
		Expect(k8sClient.Create(ctx, scope)).Should(Succeed())
		Eventually(func(g Gomega) {
			createdScope := &keycloakApi.KeycloakClientScope{}
			err := k8sClient.Get(ctx, types.NamespacedName{Name: scope.Name, Namespace: ns}, createdScope)
			g.Expect(err).ShouldNot(HaveOccurred())
			g.Expect(createdScope.Status.Value).Should(Equal(common.StatusOK))
		}).WithTimeout(timeout).WithPolling(interval).Should(Succeed())

		By("Updating KeycloakClientScope type to none")
		updatableScope := &keycloakApi.KeycloakClientScope{}
		Expect(k8sClient.Get(ctx, types.NamespacedName{Name: scope.Name, Namespace: ns}, updatableScope)).Should(Succeed())
		updatableScope.Spec.Type = keycloakApi.KeycloakClientScopeTypeNone
		Expect(k8sClient.Update(ctx, updatableScope)).Should(Succeed())

		Eventually(func(g Gomega) {
			updatedScope := &keycloakApi.KeycloakClientScope{}
			err := k8sClient.Get(ctx, types.NamespacedName{Name: scope.Name, Namespace: ns}, updatedScope)
			g.Expect(err).ShouldNot(HaveOccurred())
			g.Expect(updatedScope.Status.Value).Should(Equal(common.StatusOK))
		}, timeout, interval).Should(Succeed())

		By("Verifying the client scope is in neither default nor optional list via API")
		Eventually(func(g Gomega) {
			adapter := keycloakAdapterManager.GetAdapter()

			// Verify scope is NOT in default list
			hasDefault, err := adapter.HasDefaultClientScope(ctx, KeycloakRealmCR, "test-update-def-none")
			g.Expect(err).ShouldNot(HaveOccurred())
			g.Expect(hasDefault).Should(BeFalse(), "scope should NOT be in default scopes list")

			// Verify scope is NOT in optional list
			hasOptional, err := adapter.HasOptionalClientScope(ctx, KeycloakRealmCR, "test-update-def-none")
			g.Expect(err).ShouldNot(HaveOccurred())
			g.Expect(hasOptional).Should(BeFalse(), "scope should NOT be in optional scopes list")
		}, timeout, interval).Should(Succeed())

		By("Deleting the KeycloakClientScope")
		Expect(k8sClient.Delete(ctx, scope)).Should(Succeed())
		Eventually(func(g Gomega) {
			deletedScope := &keycloakApi.KeycloakClientScope{}
			err := k8sClient.Get(ctx, types.NamespacedName{Name: scope.Name, Namespace: ns}, deletedScope)
			g.Expect(k8sErrors.IsNotFound(err)).Should(BeTrue())
		}, timeout, interval).Should(Succeed())
	})
})

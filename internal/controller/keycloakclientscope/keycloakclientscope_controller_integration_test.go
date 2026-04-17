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
	"github.com/epam/edp-keycloak-operator/pkg/client/keycloakapi"
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
			// Verify scope is in optional list
			optionalScopes, _, err := keycloakApiClient.ClientScopes.GetRealmOptionalClientScopes(ctx, KeycloakRealmCR)
			g.Expect(err).ShouldNot(HaveOccurred())
			g.Expect(hasDefaultScope(optionalScopes, "groups")).Should(BeTrue(), "scope should be in optional scopes list")

			// Verify scope is NOT in default list
			defaultScopes, _, err := keycloakApiClient.ClientScopes.GetRealmDefaultClientScopes(ctx, KeycloakRealmCR)
			g.Expect(err).ShouldNot(HaveOccurred())
			g.Expect(hasDefaultScope(defaultScopes, "groups")).Should(BeFalse(), "scope should NOT be in default scopes list")
		}, timeout, interval).Should(Succeed())

		By("Verifying the client scope fields via Keycloak API")
		Eventually(func(g Gomega) {
			scopes, _, err := keycloakApiClient.ClientScopes.GetClientScopes(ctx, KeycloakRealmCR)
			g.Expect(err).ShouldNot(HaveOccurred())

			kcScope := findScopeByName(scopes, "groups")
			g.Expect(kcScope).ShouldNot(BeNil(), "scope 'groups' should exist in Keycloak")

			g.Expect(kcScope.Description).ShouldNot(BeNil())
			g.Expect(*kcScope.Description).Should(Equal("Group Membership"))

			g.Expect(kcScope.Protocol).ShouldNot(BeNil())
			g.Expect(*kcScope.Protocol).Should(Equal("openid-connect"))

			g.Expect(kcScope.Attributes).ShouldNot(BeNil())
			g.Expect((*kcScope.Attributes)["include.in.token.scope"]).Should(Equal("true"))

			g.Expect(kcScope.ProtocolMappers).ShouldNot(BeNil())
			g.Expect(*kcScope.ProtocolMappers).Should(HaveLen(1))

			mapper := (*kcScope.ProtocolMappers)[0]
			g.Expect(mapper.Name).ShouldNot(BeNil())
			g.Expect(*mapper.Name).Should(Equal("groups"))
			g.Expect(mapper.Protocol).ShouldNot(BeNil())
			g.Expect(*mapper.Protocol).Should(Equal("openid-connect"))
			g.Expect(mapper.ProtocolMapper).ShouldNot(BeNil())
			g.Expect(*mapper.ProtocolMapper).Should(Equal("oidc-group-membership-mapper"))
			g.Expect(mapper.Config).ShouldNot(BeNil())
			g.Expect((*mapper.Config)["claim.name"]).Should(Equal("groups"))
			g.Expect((*mapper.Config)["full.path"]).Should(Equal("false"))
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

		By("Verifying the updated description via Keycloak API")
		Eventually(func(g Gomega) {
			scopes, _, err := keycloakApiClient.ClientScopes.GetClientScopes(ctx, KeycloakRealmCR)
			g.Expect(err).ShouldNot(HaveOccurred())

			kcScope := findScopeByName(scopes, "groups")
			g.Expect(kcScope).ShouldNot(BeNil())
			g.Expect(kcScope.Description).ShouldNot(BeNil())
			g.Expect(*kcScope.Description).Should(Equal("new-description"))

			g.Expect(kcScope.Attributes).ShouldNot(BeNil())
			g.Expect((*kcScope.Attributes)["include.in.token.scope"]).Should(Equal("true"))
		}, timeout, interval).Should(Succeed())
	})
	It("Should update KeycloakClientScope protocol mappers", func() {
		By("Getting KeycloakClientScope")
		scope := &keycloakApi.KeycloakClientScope{}
		Expect(k8sClient.Get(ctx, types.NamespacedName{Namespace: ns, Name: "test-client-scope"}, scope)).Should(Succeed())

		By("Updating protocol mappers: replace with two new mappers")
		scope.Spec.ProtocolMappers = []keycloakApi.ProtocolMapper{
			{
				Name:           "groups-updated",
				Protocol:       "openid-connect",
				ProtocolMapper: "oidc-group-membership-mapper",
				Config: map[string]string{
					"access.token.claim":   "true",
					"claim.name":           "groups_v2",
					"full.path":            "true",
					"id.token.claim":       "true",
					"userinfo.token.claim": "true",
				},
			},
			{
				Name:           "audience-mapper",
				Protocol:       "openid-connect",
				ProtocolMapper: "oidc-audience-mapper",
				Config: map[string]string{
					"included.client.audience": "account",
					"id.token.claim":           "false",
					"access.token.claim":       "true",
				},
			},
		}
		Expect(k8sClient.Update(ctx, scope)).Should(Succeed())

		Eventually(func(g Gomega) {
			updatedScope := &keycloakApi.KeycloakClientScope{}
			err := k8sClient.Get(ctx, types.NamespacedName{Name: scope.Name, Namespace: ns}, updatedScope)
			g.Expect(err).ShouldNot(HaveOccurred())
			g.Expect(updatedScope.Status.Value).Should(Equal(common.StatusOK))
		}, timeout, interval).Should(Succeed())

		By("Verifying protocol mappers via Keycloak API")
		Eventually(func(g Gomega) {
			scopes, _, err := keycloakApiClient.ClientScopes.GetClientScopes(ctx, KeycloakRealmCR)
			g.Expect(err).ShouldNot(HaveOccurred())

			kcScope := findScopeByName(scopes, "groups")
			g.Expect(kcScope).ShouldNot(BeNil())
			g.Expect(kcScope.ProtocolMappers).ShouldNot(BeNil())
			g.Expect(*kcScope.ProtocolMappers).Should(HaveLen(2))

			mapperNames := make([]string, 0, 2)
			for _, m := range *kcScope.ProtocolMappers {
				g.Expect(m.Name).ShouldNot(BeNil())
				mapperNames = append(mapperNames, *m.Name)
			}

			g.Expect(mapperNames).Should(ContainElements("groups-updated", "audience-mapper"))
		}, timeout, interval).Should(Succeed())

		By("Removing all protocol mappers")
		updatableScope := &keycloakApi.KeycloakClientScope{}
		Expect(k8sClient.Get(ctx, types.NamespacedName{Namespace: ns, Name: "test-client-scope"}, updatableScope)).Should(Succeed())
		updatableScope.Spec.ProtocolMappers = nil
		Expect(k8sClient.Update(ctx, updatableScope)).Should(Succeed())

		Eventually(func(g Gomega) {
			updatedScope := &keycloakApi.KeycloakClientScope{}
			err := k8sClient.Get(ctx, types.NamespacedName{Name: scope.Name, Namespace: ns}, updatedScope)
			g.Expect(err).ShouldNot(HaveOccurred())
			g.Expect(updatedScope.Status.Value).Should(Equal(common.StatusOK))
		}, timeout, interval).Should(Succeed())

		By("Verifying all protocol mappers are removed via Keycloak API")
		Eventually(func(g Gomega) {
			scopes, _, err := keycloakApiClient.ClientScopes.GetClientScopes(ctx, KeycloakRealmCR)
			g.Expect(err).ShouldNot(HaveOccurred())

			kcScope := findScopeByName(scopes, "groups")
			g.Expect(kcScope).ShouldNot(BeNil())

			if kcScope.ProtocolMappers != nil {
				g.Expect(*kcScope.ProtocolMappers).Should(BeEmpty())
			}
		}, timeout, interval).Should(Succeed())
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
	It("Should delete KeycloakClientScope with two finalizers", func() {
		By("Creating a KeycloakClientScope")
		scope := &keycloakApi.KeycloakClientScope{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-client-scope-two-finalizers",
				Namespace: ns,
			},
			Spec: keycloakApi.KeycloakClientScopeSpec{
				Name: "test-two-finalizers-scope",
				RealmRef: common.RealmRef{
					Kind: keycloakApi.KeycloakRealmKind,
					Name: KeycloakRealmCR,
				},
				Protocol: "openid-connect",
				Type:     keycloakApi.KeycloakClientScopeTypeOptional,
			},
		}
		Expect(k8sClient.Create(ctx, scope)).Should(Succeed())
		Eventually(func(g Gomega) {
			createdScope := &keycloakApi.KeycloakClientScope{}
			err := k8sClient.Get(ctx, types.NamespacedName{Name: scope.Name, Namespace: ns}, createdScope)
			g.Expect(err).ShouldNot(HaveOccurred())
			g.Expect(createdScope.Status.Value).Should(Equal(common.StatusOK))
		}).WithTimeout(timeout).WithPolling(interval).Should(Succeed())

		By("Patching the scope to add the legacy finalizer alongside the existing one")
		createdScope := &keycloakApi.KeycloakClientScope{}
		Expect(k8sClient.Get(ctx, types.NamespacedName{Name: scope.Name, Namespace: ns}, createdScope)).Should(Succeed())
		createdScope.Finalizers = append(createdScope.Finalizers, legacyFinalizerName)
		Expect(k8sClient.Update(ctx, createdScope)).Should(Succeed())

		By("Deleting the KeycloakClientScope")
		Expect(k8sClient.Delete(ctx, createdScope)).Should(Succeed())
		Eventually(func(g Gomega) {
			deletedScope := &keycloakApi.KeycloakClientScope{}
			err := k8sClient.Get(ctx, types.NamespacedName{Name: scope.Name, Namespace: ns}, deletedScope)
			g.Expect(k8sErrors.IsNotFound(err)).Should(BeTrue())
		}, timeout, interval).Should(Succeed())

		By("Verifying the scope is removed from Keycloak")
		Eventually(func(g Gomega) {
			scopes, _, err := keycloakApiClient.ClientScopes.GetClientScopes(ctx, KeycloakRealmCR)
			g.Expect(err).ShouldNot(HaveOccurred())
			g.Expect(findScopeByName(scopes, "test-two-finalizers-scope")).Should(BeNil())
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
			g.Expect(createdScope.Status.Value).Should(ContainSubstring("client scope chain processing failed"))
		}, timeout, interval).Should(Succeed())

		By("Verifying the error status persists")
		Consistently(func(g Gomega) {
			createdScope := &keycloakApi.KeycloakClientScope{}
			err := k8sClient.Get(ctx, types.NamespacedName{Name: scope.Name, Namespace: ns}, createdScope)
			g.Expect(err).ShouldNot(HaveOccurred())
			g.Expect(createdScope.Status.Value).Should(ContainSubstring("client scope chain processing failed"))
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
			defaultScopes, _, err := keycloakApiClient.ClientScopes.GetRealmDefaultClientScopes(ctx, KeycloakRealmCR)
			g.Expect(err).ShouldNot(HaveOccurred())
			g.Expect(hasDefaultScope(defaultScopes, "test-default-scope")).Should(BeTrue(), "scope should be in default scopes list")

			optionalScopes, _, err := keycloakApiClient.ClientScopes.GetRealmOptionalClientScopes(ctx, KeycloakRealmCR)
			g.Expect(err).ShouldNot(HaveOccurred())
			g.Expect(hasDefaultScope(optionalScopes, "test-default-scope")).Should(BeFalse(), "scope should NOT be in optional scopes list")
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
			defaultScopes, _, err := keycloakApiClient.ClientScopes.GetRealmDefaultClientScopes(ctx, KeycloakRealmCR)
			g.Expect(err).ShouldNot(HaveOccurred())
			g.Expect(hasDefaultScope(defaultScopes, "test-none-scope")).Should(BeFalse(), "scope should NOT be in default scopes list")

			optionalScopes, _, err := keycloakApiClient.ClientScopes.GetRealmOptionalClientScopes(ctx, KeycloakRealmCR)
			g.Expect(err).ShouldNot(HaveOccurred())
			g.Expect(hasDefaultScope(optionalScopes, "test-none-scope")).Should(BeFalse(), "scope should NOT be in optional scopes list")
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
			defaultScopes, _, err := keycloakApiClient.ClientScopes.GetRealmDefaultClientScopes(ctx, KeycloakRealmCR)
			g.Expect(err).ShouldNot(HaveOccurred())
			g.Expect(hasDefaultScope(defaultScopes, "test-update-opt-def")).Should(BeTrue(), "scope should be in default scopes list")

			optionalScopes, _, err := keycloakApiClient.ClientScopes.GetRealmOptionalClientScopes(ctx, KeycloakRealmCR)
			g.Expect(err).ShouldNot(HaveOccurred())
			g.Expect(hasDefaultScope(optionalScopes, "test-update-opt-def")).Should(BeFalse(), "scope should NOT be in optional scopes list")
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
			defaultScopes, _, err := keycloakApiClient.ClientScopes.GetRealmDefaultClientScopes(ctx, KeycloakRealmCR)
			g.Expect(err).ShouldNot(HaveOccurred())
			g.Expect(hasDefaultScope(defaultScopes, "test-update-def-none")).Should(BeFalse(), "scope should NOT be in default scopes list")

			optionalScopes, _, err := keycloakApiClient.ClientScopes.GetRealmOptionalClientScopes(ctx, KeycloakRealmCR)
			g.Expect(err).ShouldNot(HaveOccurred())
			g.Expect(hasDefaultScope(optionalScopes, "test-update-def-none")).Should(BeFalse(), "scope should NOT be in optional scopes list")
		}, timeout, interval).Should(Succeed())

		By("Deleting the KeycloakClientScope")
		Expect(k8sClient.Delete(ctx, scope)).Should(Succeed())
		Eventually(func(g Gomega) {
			deletedScope := &keycloakApi.KeycloakClientScope{}
			err := k8sClient.Get(ctx, types.NamespacedName{Name: scope.Name, Namespace: ns}, deletedScope)
			g.Expect(k8sErrors.IsNotFound(err)).Should(BeTrue())
		}, timeout, interval).Should(Succeed())
	})

	It("Should create KeycloakClientScope with deprecated Default field", func() {
		By("Creating a KeycloakClientScope with Default: true and Type: none")
		scope := &keycloakApi.KeycloakClientScope{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-client-scope-deprecated-default",
				Namespace: ns,
			},
			Spec: keycloakApi.KeycloakClientScopeSpec{
				Name: "test-deprecated-default",
				RealmRef: common.RealmRef{
					Kind: keycloakApi.KeycloakRealmKind,
					Name: KeycloakRealmCR,
				},
				Protocol:    "openid-connect",
				Description: "Test deprecated Default field",
				Default:     true,
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

		By("Verifying the scope exists in Keycloak and is in the default list (Default: true overrides Type: none)")
		Eventually(func(g Gomega) {
			scopes, _, err := keycloakApiClient.ClientScopes.GetClientScopes(ctx, KeycloakRealmCR)
			g.Expect(err).ShouldNot(HaveOccurred())

			kcScope := findScopeByName(scopes, "test-deprecated-default")
			g.Expect(kcScope).ShouldNot(BeNil(), "scope 'test-deprecated-default' should exist in Keycloak")
			g.Expect(kcScope.Description).ShouldNot(BeNil())
			g.Expect(*kcScope.Description).Should(Equal("Test deprecated Default field"))

			defaultScopes, _, err := keycloakApiClient.ClientScopes.GetRealmDefaultClientScopes(ctx, KeycloakRealmCR)
			g.Expect(err).ShouldNot(HaveOccurred())
			g.Expect(hasDefaultScope(defaultScopes, "test-deprecated-default")).Should(BeTrue(),
				"scope should be in default scopes list because Default: true overrides Type: none")

			optionalScopes, _, err := keycloakApiClient.ClientScopes.GetRealmOptionalClientScopes(ctx, KeycloakRealmCR)
			g.Expect(err).ShouldNot(HaveOccurred())
			g.Expect(hasDefaultScope(optionalScopes, "test-deprecated-default")).Should(BeFalse(),
				"scope should NOT be in optional scopes list")
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

func hasDefaultScope(scopes []keycloakapi.ClientScopeRepresentation, name string) bool {
	for _, s := range scopes {
		if s.Name != nil && *s.Name == name {
			return true
		}
	}

	return false
}

func findScopeByName(scopes []keycloakapi.ClientScopeRepresentation, name string) *keycloakapi.ClientScopeRepresentation {
	for i := range scopes {
		if scopes[i].Name != nil && *scopes[i].Name == name {
			return &scopes[i]
		}
	}

	return nil
}

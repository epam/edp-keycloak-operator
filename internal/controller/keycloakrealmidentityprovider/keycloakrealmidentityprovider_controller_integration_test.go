package keycloakrealmidentityprovider

import (
	"fmt"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	corev1 "k8s.io/api/core/v1"
	k8sErrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/utils/ptr"

	"github.com/epam/edp-keycloak-operator/api/common"
	keycloakApi "github.com/epam/edp-keycloak-operator/api/v1"
	"github.com/epam/edp-keycloak-operator/pkg/objectmeta"
)

var _ = Describe("KeycloakRealmIdentityProvider controller", Ordered, func() {
	const (
		identityProviderCR = "test-keycloak-realm-identity-provider"
	)
	It("Should create KeycloakRealmIdentityProvider", func() {
		By("By creating a client secret")
		clientSecret := &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-keycloak-realm-client-secret",
				Namespace: ns,
			},
			Data: map[string][]byte{
				"secretKey": []byte("secretValue"),
			},
		}
		Expect(k8sClient.Create(ctx, clientSecret)).Should(Succeed())
		By("By creating a KeycloakRealmIdentityProvider")
		provider := &keycloakApi.KeycloakRealmIdentityProvider{
			ObjectMeta: metav1.ObjectMeta{
				Name:      identityProviderCR,
				Namespace: ns,
			},
			Spec: keycloakApi.KeycloakRealmIdentityProviderSpec{
				RealmRef: common.RealmRef{
					Name: KeycloakRealmCR,
					Kind: keycloakApi.KeycloakRealmKind,
				},
				ProviderID: "github",
				Alias:      "new-provider",
				Config: map[string]string{
					"clientId":     "provider-client",
					"clientSecret": fmt.Sprintf("$%s:%s", clientSecret.Name, "secretKey"),
				},
				Enabled:                   true,
				DisplayName:               "New provider",
				FirstBrokerLoginFlowAlias: "first broker login",
				PostBrokerLoginFlowAlias:  "browser",
				HideOnLogin:               ptr.To(true),
			},
		}
		Expect(k8sClient.Create(ctx, provider)).Should(Succeed())
		Eventually(func(g Gomega) {
			createdProvider := &keycloakApi.KeycloakRealmIdentityProvider{}
			err := k8sClient.Get(ctx, types.NamespacedName{Name: provider.Name, Namespace: ns}, createdProvider)
			g.Expect(err).ShouldNot(HaveOccurred())
			g.Expect(createdProvider.Status.Value).Should(Equal(common.StatusOK))
		}).WithTimeout(time.Second * 30).WithPolling(time.Second).Should(Succeed())

		By("Verifying the identity provider was created in Keycloak")
		Eventually(func(g Gomega) {
			idp, _, err := keycloakApiClient.IdentityProviders.GetIdentityProvider(ctx, KeycloakRealmCR, "new-provider")
			g.Expect(err).ShouldNot(HaveOccurred())
			g.Expect(idp).ShouldNot(BeNil())
			g.Expect(idp.Alias).ShouldNot(BeNil())
			g.Expect(*idp.Alias).Should(Equal("new-provider"))
			g.Expect(idp.ProviderId).ShouldNot(BeNil())
			g.Expect(*idp.ProviderId).Should(Equal("github"))
			g.Expect(idp.Enabled).ShouldNot(BeNil())
			g.Expect(*idp.Enabled).Should(BeTrue())
			g.Expect(idp.DisplayName).ShouldNot(BeNil())
			g.Expect(*idp.DisplayName).Should(Equal("New provider"))
			g.Expect(idp.FirstBrokerLoginFlowAlias).ShouldNot(BeNil())
			g.Expect(*idp.FirstBrokerLoginFlowAlias).Should(Equal("first broker login"))
			g.Expect(idp.Config).ShouldNot(BeNil())
			g.Expect((*idp.Config)["clientId"]).Should(Equal("provider-client"))
			g.Expect(idp.HideOnLogin).ShouldNot(BeNil())
			g.Expect(*idp.HideOnLogin).Should(BeTrue())
		}, time.Second*10, time.Second).Should(Succeed())
	})
	It("Should update KeycloakRealmIdentityProvider", func() {
		By("By getting existing KeycloakRealmIdentityProvider")
		provider := &keycloakApi.KeycloakRealmIdentityProvider{}
		Expect(k8sClient.Get(ctx, types.NamespacedName{Namespace: ns, Name: identityProviderCR}, provider)).Should(Succeed())

		By("By updating spec: change display name and add mappers")
		provider.Spec.DisplayName = "Updated provider"
		provider.Spec.Mappers = []keycloakApi.IdentityProviderMapper{
			{
				Name:                   "test-mapper",
				IdentityProviderMapper: "hardcoded-attribute-idp-mapper",
				Config: map[string]string{
					"attribute":       "test-attr",
					"attribute.value": "test-value",
				},
			},
		}
		Expect(k8sClient.Update(ctx, provider)).Should(Succeed())

		By("Waiting for reconciliation to complete")
		Eventually(func(g Gomega) {
			updatedProvider := &keycloakApi.KeycloakRealmIdentityProvider{}
			err := k8sClient.Get(ctx, types.NamespacedName{Name: provider.Name, Namespace: ns}, updatedProvider)
			g.Expect(err).ShouldNot(HaveOccurred())
			g.Expect(updatedProvider.Status.Value).Should(Equal(common.StatusOK))
		}, timeout, interval).Should(Succeed())

		By("Verifying identity provider was updated in Keycloak")
		Eventually(func(g Gomega) {
			idp, _, err := keycloakApiClient.IdentityProviders.GetIdentityProvider(ctx, KeycloakRealmCR, "new-provider")
			g.Expect(err).ShouldNot(HaveOccurred())
			g.Expect(idp).ShouldNot(BeNil())
			g.Expect(idp.DisplayName).ShouldNot(BeNil())
			g.Expect(*idp.DisplayName).Should(Equal("Updated provider"))
		}, time.Second*10, time.Second).Should(Succeed())

		By("Verifying mapper was created in Keycloak")
		Eventually(func(g Gomega) {
			mappers, _, err := keycloakApiClient.IdentityProviders.GetIDPMappers(ctx, KeycloakRealmCR, "new-provider")
			g.Expect(err).ShouldNot(HaveOccurred())
			g.Expect(mappers).Should(HaveLen(1))
			g.Expect(mappers[0].Name).ShouldNot(BeNil())
			g.Expect(*mappers[0].Name).Should(Equal("test-mapper"))
		}, time.Second*10, time.Second).Should(Succeed())
	})
	It("Should delete KeycloakRealmIdentityProvider", func() {
		By("By getting KeycloakRealmIdentityProvider")
		provider := &keycloakApi.KeycloakRealmIdentityProvider{}
		Expect(k8sClient.Get(ctx, types.NamespacedName{Namespace: ns, Name: identityProviderCR}, provider)).Should(Succeed())
		By("By deleting KeycloakRealmIdentityProvider")
		Expect(k8sClient.Delete(ctx, provider)).Should(Succeed())
		Eventually(func() bool {
			deletedProvider := &keycloakApi.KeycloakRealmIdentityProvider{}
			err := k8sClient.Get(ctx, types.NamespacedName{Name: provider.Name, Namespace: ns}, deletedProvider)

			return k8sErrors.IsNotFound(err)
		}, timeout, interval).Should(BeTrue(), "KeycloakRealmIdentityProvider should be deleted")
	})
	It("Should skip keycloak resource removing if preserveResourcesOnDeletion is set", func() {
		By("By creating a KeycloakRealmIdentityProvider")
		provider := &keycloakApi.KeycloakRealmIdentityProvider{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "identity-provider-with-preserve-resources-on-deletion",
				Namespace: ns,
				Annotations: map[string]string{
					objectmeta.PreserveResourcesOnDeletionAnnotation: "true",
				},
			},
			Spec: keycloakApi.KeycloakRealmIdentityProviderSpec{
				RealmRef: common.RealmRef{
					Name: KeycloakRealmCR,
					Kind: keycloakApi.KeycloakRealmKind,
				},
				ProviderID:                "github",
				Alias:                     "identity-provider-with-preserve-resources-on-deletion",
				Enabled:                   true,
				DisplayName:               "New provider",
				FirstBrokerLoginFlowAlias: "first broker login",
				Config: map[string]string{
					"clientId": "identity-provider-with-preserve-resources-on-deletion",
				},
			},
		}
		Expect(k8sClient.Create(ctx, provider)).Should(Succeed())
		Eventually(func(g Gomega) {
			createdProvider := &keycloakApi.KeycloakRealmIdentityProvider{}
			err := k8sClient.Get(ctx, types.NamespacedName{Name: provider.Name, Namespace: ns}, createdProvider)
			g.Expect(err).ShouldNot(HaveOccurred())
			g.Expect(createdProvider.Status.Value).Should(Equal(common.StatusOK))
		}, timeout, interval).Should(Succeed())
		By("By deleting KeycloakRealmIdentityProvider")
		Expect(k8sClient.Delete(ctx, provider)).Should(Succeed())
		By("Waiting for KeycloakRealmIdentityProvider to be deleted")
		Eventually(func() bool {
			deletedProvider := &keycloakApi.KeycloakRealmIdentityProvider{}
			err := k8sClient.Get(ctx, types.NamespacedName{Name: provider.Name, Namespace: ns}, deletedProvider)

			return k8sErrors.IsNotFound(err)
		}, timeout, interval).Should(BeTrue(), "KeycloakRealmIdentityProvider with preserveResourcesOnDeletion annotation should be deleted")
	})
	It("Should successfully delete KeycloakRealmIdentityProvider if realm is deleted first", func() {
		By("By creating a KeycloakRealm")
		testRealm := &keycloakApi.KeycloakRealm{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-idp-realm-for-deletion",
				Namespace: ns,
			},
			Spec: keycloakApi.KeycloakRealmSpec{
				RealmName: "test-idp-realm-for-deletion",
				KeycloakRef: common.KeycloakRef{
					Kind: keycloakApi.KeycloakKind,
					Name: KeycloakCR,
				},
			},
		}
		Expect(k8sClient.Create(ctx, testRealm)).Should(Succeed())
		Eventually(func(g Gomega) {
			createdRealm := &keycloakApi.KeycloakRealm{}
			err := k8sClient.Get(ctx, types.NamespacedName{Name: testRealm.Name, Namespace: ns}, createdRealm)
			g.Expect(err).ShouldNot(HaveOccurred())
			g.Expect(createdRealm.Status.Available).Should(BeTrue())
		}, time.Second*30, interval).Should(Succeed())

		By("By creating a KeycloakRealmIdentityProvider in that realm")
		provider := &keycloakApi.KeycloakRealmIdentityProvider{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "idp-for-realm-deletion-test",
				Namespace: ns,
			},
			Spec: keycloakApi.KeycloakRealmIdentityProviderSpec{
				RealmRef: common.RealmRef{
					Name: testRealm.Name,
					Kind: keycloakApi.KeycloakRealmKind,
				},
				ProviderID: "github",
				Alias:      "idp-for-realm-deletion-test",
				Enabled:    true,
				Config: map[string]string{
					"clientId": "idp-for-realm-deletion-test",
				},
			},
		}
		Expect(k8sClient.Create(ctx, provider)).Should(Succeed())

		By("Waiting for KeycloakRealmIdentityProvider to be ready")
		Eventually(func(g Gomega) {
			createdProvider := &keycloakApi.KeycloakRealmIdentityProvider{}
			err := k8sClient.Get(ctx, types.NamespacedName{Name: provider.Name, Namespace: ns}, createdProvider)
			g.Expect(err).ShouldNot(HaveOccurred())
			g.Expect(createdProvider.Status.Value).Should(Equal(common.StatusOK))
		}, time.Second*30, interval).Should(Succeed())

		By("Deleting the KeycloakRealm first")
		Expect(k8sClient.Delete(ctx, testRealm)).Should(Succeed())
		Eventually(func() bool {
			var r keycloakApi.KeycloakRealm
			err := k8sClient.Get(ctx, types.NamespacedName{Name: testRealm.Name, Namespace: ns}, &r)
			return k8sErrors.IsNotFound(err)
		}, time.Minute, time.Second*5).Should(BeTrue())

		By("Deleting the KeycloakRealmIdentityProvider after realm is gone")
		Expect(k8sClient.Delete(ctx, provider)).Should(Succeed())

		By("Waiting for KeycloakRealmIdentityProvider to be deleted via finalizer cleanup")
		Eventually(func() bool {
			var p keycloakApi.KeycloakRealmIdentityProvider
			err := k8sClient.Get(ctx, types.NamespacedName{Name: provider.Name, Namespace: ns}, &p)
			return k8sErrors.IsNotFound(err)
		}, time.Minute, time.Second*5).Should(BeTrue())
	})
})

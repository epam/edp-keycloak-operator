package keycloakrealm

import (
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/Nerzal/gocloak/v12"
	k8sErrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/utils/pointer"

	"github.com/epam/edp-keycloak-operator/api/common"
	keycloakApi "github.com/epam/edp-keycloak-operator/api/v1"
)

var _ = Describe("KeycloakRealm controller", Ordered, func() {
	const (
		keycloakRealmCR = "test-keycloak-realm-cr"
		ssoRealmName    = "test-sso-realm"
	)
	It("Should create KeycloakRealm", func() {
		By("By creating SSO realm KeycloakRealm")
		ssoKeycloakRealm := &keycloakApi.KeycloakRealm{
			ObjectMeta: metav1.ObjectMeta{
				Name:      ssoRealmName,
				Namespace: ns,
			},
			Spec: keycloakApi.KeycloakRealmSpec{
				RealmName: ssoRealmName,
				KeycloakRef: common.KeycloakRef{
					Name: keycloakCR,
					Kind: keycloakApi.KeycloakKind,
				},
			},
		}
		Expect(k8sClient.Create(ctx, ssoKeycloakRealm)).Should(Succeed())
		Eventually(func() bool {
			createdSSOKeycloakRealm := &keycloakApi.KeycloakRealm{}
			err := k8sClient.Get(ctx, types.NamespacedName{Name: ssoRealmName, Namespace: ns}, createdSSOKeycloakRealm)
			Expect(err).ShouldNot(HaveOccurred())

			return createdSSOKeycloakRealm.Status.Available
		}, timeout, interval).Should(BeTrue())

		By("By creating KeycloakRealm with full config")
		keycloakRealm := &keycloakApi.KeycloakRealm{
			ObjectMeta: metav1.ObjectMeta{
				Name:      keycloakRealmCR,
				Namespace: ns,
			},
			Spec: keycloakApi.KeycloakRealmSpec{
				RealmName: "test-realm-with-full-config",
				KeycloakRef: common.KeycloakRef{
					Name: keycloakCR,
					Kind: keycloakApi.KeycloakKind,
				},
				SsoRealmName:           ssoRealmName,
				SsoRealmEnabled:        pointer.Bool(true),
				SsoAutoRedirectEnabled: pointer.Bool(true),
				SSORealmMappers: &[]keycloakApi.SSORealmMapper{
					{
						IdentityProviderMapper: "hardcoded-attribute-idp-mapper",
						Name:                   "test-mapper",
						Config: map[string]string{
							"attribute.name": "test-attribute",
						},
					},
				},
				BrowserFlow: pointer.String("browser"),
				RealmEventConfig: &keycloakApi.RealmEventConfig{
					AdminEventsDetailsEnabled: false,
					AdminEventsEnabled:        true,
					EnabledEventTypes:         []string{"UPDATE_CONSENT_ERROR", "CLIENT_LOGIN"},
					EventsEnabled:             true,
					EventsExpiration:          15000,
					EventsListeners:           []string{"jboss-logging"},
				},
				DisableCentralIDPMappers: false,
				PasswordPolicies: []keycloakApi.PasswordPolicy{
					{
						Type:  "forceExpiredPasswordChange",
						Value: "365",
					},
				},
				FrontendURL: "https://test.com",
				Users: []keycloakApi.User{
					{
						Username:   "keycloakrealm-user@mail.com",
						RealmRoles: []string{"administrator"},
					},
				},
			},
		}
		Expect(k8sClient.Create(ctx, keycloakRealm)).Should(Succeed())

		By("By disable HTTPS for realm")
		// Wait for KeycloakRealm to be created.
		time.Sleep(time.Second * 5)
		Eventually(func(g Gomega) {
			// We need to disable HTTPS for realm to be able to create
			// IdentityProvider with http AuthorizationUrl (adapter.GoCloakAdapter.CreateCentralIdentityProvider).
			// This is needed for integration tests on the CI where Keycloak is http external URL.
			// KeycloakRealm CR doesn't have a field to disable HTTPS, so we need to do it manually by API.
			client := gocloak.NewClient(keycloakURL)
			token, err := client.LoginAdmin(ctx, "admin", "admin", "master")
			g.Expect(err).ShouldNot(HaveOccurred())

			rl, err := client.GetRealm(ctx, token.AccessToken, keycloakRealm.Spec.RealmName)
			g.Expect(err).ShouldNot(HaveOccurred())
			g.Expect(rl).ShouldNot(BeNil())

			rl.SslRequired = pointer.String("none")

			err = client.UpdateRealm(ctx, token.AccessToken, *rl)
			g.Expect(err).ShouldNot(HaveOccurred())
		}, time.Second*6, time.Second*2).Should(Succeed())

		Eventually(func() bool {
			createdKeycloakRealm := &keycloakApi.KeycloakRealm{}
			err := k8sClient.Get(ctx, types.NamespacedName{Name: keycloakRealmCR, Namespace: ns}, createdKeycloakRealm)
			Expect(err).ShouldNot(HaveOccurred())

			if !createdKeycloakRealm.Status.Available {
				GinkgoWriter.Println("KeycloakRealm status error: ", createdKeycloakRealm.Status.Value)
			}

			return createdKeycloakRealm.Status.Available
		}, time.Minute, time.Second*5).Should(BeTrue())
	})
	It("Should update KeycloakRealm", func() {
		By("By getting KeycloakRealm")
		keycloakRealm := &keycloakApi.KeycloakRealm{}
		err := k8sClient.Get(ctx, types.NamespacedName{Name: keycloakRealmCR, Namespace: ns}, keycloakRealm)
		Expect(err).ShouldNot(HaveOccurred())

		By("By updating KeycloakRealm")
		keycloakRealm.Spec.FrontendURL = "https://test-updated.com"
		Expect(k8sClient.Update(ctx, keycloakRealm)).Should(Succeed())

		Eventually(func() bool {
			updatedKeycloakRealm := &keycloakApi.KeycloakRealm{}
			err := k8sClient.Get(ctx, types.NamespacedName{Name: keycloakRealmCR, Namespace: ns}, updatedKeycloakRealm)
			Expect(err).ShouldNot(HaveOccurred())

			return updatedKeycloakRealm.Status.Available && updatedKeycloakRealm.Spec.FrontendURL == "https://test-updated.com"
		}, timeout, interval).Should(BeTrue())
	})
	It("Should delete KeycloakRealm", func() {
		By("By getting KeycloakRealm")
		keycloakRealm := &keycloakApi.KeycloakRealm{}
		err := k8sClient.Get(ctx, types.NamespacedName{Name: keycloakRealmCR, Namespace: ns}, keycloakRealm)
		Expect(err).ShouldNot(HaveOccurred())

		By("By deleting KeycloakRealm")
		Expect(k8sClient.Delete(ctx, keycloakRealm)).Should(Succeed())
		Eventually(func() bool {
			deletedRealm := &keycloakApi.KeycloakRealm{}
			err := k8sClient.Get(ctx, types.NamespacedName{Name: keycloakRealmCR, Namespace: ns}, deletedRealm)

			return k8sErrors.IsNotFound(err)
		}, timeout, interval).Should(BeTrue(), "KeycloakRealm should be deleted")
	})
})

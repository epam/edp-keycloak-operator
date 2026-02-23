package clusterkeycloakrealm

import (
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	k8sErrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/utils/ptr"

	"github.com/epam/edp-keycloak-operator/api/common"
	keycloakApi "github.com/epam/edp-keycloak-operator/api/v1"
	keycloakAlpha "github.com/epam/edp-keycloak-operator/api/v1alpha1"
	"github.com/epam/edp-keycloak-operator/pkg/objectmeta"
)

var _ = Describe("ClusterKeycloakRealm controller", func() {
	const (
		clusterKeycloakCR = "test-cluster-keycloak-realm"
	)
	It("Should reconcile ClusterKeycloakRealm", func() {
		By("Creating a ClusterKeycloakRealm")
		keycloakRealm := &keycloakAlpha.ClusterKeycloakRealm{
			ObjectMeta: metav1.ObjectMeta{
				Name: clusterKeycloakCR,
			},
			Spec: keycloakAlpha.ClusterKeycloakRealmSpec{
				ClusterKeycloakRef: ClusterKeycloakCR,
				RealmName:          "test-realm",
				FrontendURL:        "https://test.com",
				TokenSettings: &common.TokenSettings{
					DefaultSignatureAlgorithm:           "RS256",
					RevokeRefreshToken:                  true,
					RefreshTokenMaxReuse:                230,
					AccessTokenLifespan:                 231,
					AccessTokenLifespanForImplicitFlow:  232,
					AccessCodeLifespan:                  233,
					ActionTokenGeneratedByUserLifespan:  234,
					ActionTokenGeneratedByAdminLifespan: 235,
				},
				DisplayName:     "Test Realm",
				DisplayHTMLName: "<b>Test Realm</b>",
				RealmEventConfig: &common.RealmEventConfig{
					AdminEventsEnabled:    true,
					AdminEventsExpiration: 100,
				},
			},
		}
		Expect(k8sClient.Create(ctx, keycloakRealm)).Should(Succeed())
		Eventually(func() bool {
			createdKeycloakRealm := &keycloakAlpha.ClusterKeycloakRealm{}
			err := k8sClient.Get(ctx, types.NamespacedName{Name: clusterKeycloakCR}, createdKeycloakRealm)
			Expect(err).ShouldNot(HaveOccurred())

			return createdKeycloakRealm.Status.Available
		}, timeout, interval).Should(BeTrue())

		By("Verifying the realm was created in Keycloak")
		Eventually(func(g Gomega) {
			realm, _, err := keycloakApiClient.Realms.GetRealm(ctx, "test-realm")
			g.Expect(err).ShouldNot(HaveOccurred())
			g.Expect(realm).ShouldNot(BeNil())

			// Verify basic fields
			g.Expect(realm.DisplayName).Should(Equal(ptr.To("Test Realm")))
			g.Expect(realm.DisplayNameHtml).Should(Equal(ptr.To("<b>Test Realm</b>")))
			g.Expect(realm.Attributes).ShouldNot(BeNil())
			g.Expect((*realm.Attributes)["frontendUrl"]).Should(Equal("https://test.com"))

			// Verify token settings
			g.Expect(realm.DefaultSignatureAlgorithm).Should(Equal(ptr.To("RS256")))
			g.Expect(realm.RevokeRefreshToken).Should(Equal(ptr.To(true)))
			g.Expect(realm.RefreshTokenMaxReuse).Should(Equal(ptr.To(int32(230))))
			g.Expect(realm.AccessTokenLifespan).Should(Equal(ptr.To(int32(231))))
			g.Expect(realm.AccessTokenLifespanForImplicitFlow).Should(Equal(ptr.To(int32(232))))
			g.Expect(realm.AccessCodeLifespan).Should(Equal(ptr.To(int32(233))))
			g.Expect(realm.ActionTokenGeneratedByUserLifespan).Should(Equal(ptr.To(int32(234))))
			g.Expect(realm.ActionTokenGeneratedByAdminLifespan).Should(Equal(ptr.To(int32(235))))

			// Verify event config
			g.Expect(realm.AdminEventsEnabled).Should(Equal(ptr.To(true)))
		}, time.Second*10, time.Second).Should(Succeed())

		By("Updating ClusterKeycloakRealm with authentication flow")
		By("Getting ClusterKeycloakRealm")
		createdKeycloakRealm := &keycloakAlpha.ClusterKeycloakRealm{}
		err := k8sClient.Get(ctx, types.NamespacedName{Name: clusterKeycloakCR}, createdKeycloakRealm)
		Expect(err).ShouldNot(HaveOccurred())

		createdKeycloakRealm.Spec.AuthenticationFlow = &keycloakAlpha.AuthenticationFlow{
			BrowserFlow: "browser",
		}
		Expect(k8sClient.Update(ctx, createdKeycloakRealm)).Should(Succeed())
		Consistently(func() bool {
			updatedKeycloakRealm := &keycloakAlpha.ClusterKeycloakRealm{}

			Expect(k8sClient.Get(
				ctx,
				types.NamespacedName{Name: clusterKeycloakCR},
				updatedKeycloakRealm,
			)).ShouldNot(HaveOccurred())

			return updatedKeycloakRealm.Status.Available
		}, time.Second*3, time.Second).Should(BeTrue())

		By("Checking realm configuration")
		Eventually(func(g Gomega) {
			realm, _, err := keycloakApiClient.Realms.GetRealm(ctx, "test-realm")
			g.Expect(err).ShouldNot(HaveOccurred())
			g.Expect(realm.BrowserFlow).Should(Equal(ptr.To("browser")))
		}, time.Second*10, time.Second).Should(Succeed())

		By("By deleting ClusterKeycloakRealm")
		Expect(k8sClient.Delete(ctx, keycloakRealm)).Should(Succeed())
		Eventually(func() bool {
			deletedRealm := &keycloakAlpha.ClusterKeycloakRealm{}
			err := k8sClient.Get(ctx, types.NamespacedName{Name: clusterKeycloakCR}, deletedRealm)

			return k8sErrors.IsNotFound(err)
		}, timeout, interval).Should(BeTrue(), "ClusterKeycloakRealm should be deleted")
	})
	It("Should skip keycloak resource removing if preserveResourcesOnDeletion is set", func() {
		By("By creating a ClusterKeycloakRealm")
		keycloakRealm := &keycloakAlpha.ClusterKeycloakRealm{
			ObjectMeta: metav1.ObjectMeta{
				Name: "test-cluster-keycloak-realm-preserve-resources",
				Annotations: map[string]string{
					objectmeta.PreserveResourcesOnDeletionAnnotation: "true",
				},
			},
			Spec: keycloakAlpha.ClusterKeycloakRealmSpec{
				ClusterKeycloakRef: ClusterKeycloakCR,
				RealmName:          "test-realm2",
				FrontendURL:        "https://test.com",
			},
		}
		Expect(k8sClient.Create(ctx, keycloakRealm)).Should(Succeed())
		Eventually(func() bool {
			createdKeycloakRealm := &keycloakAlpha.ClusterKeycloakRealm{}
			err := k8sClient.Get(ctx, types.NamespacedName{Name: keycloakRealm.Name}, createdKeycloakRealm)
			Expect(err).ShouldNot(HaveOccurred())

			return createdKeycloakRealm.Status.Available
		}, timeout, interval).Should(BeTrue())
		By("By deleting ClusterKeycloakRealm")
		Expect(k8sClient.Delete(ctx, keycloakRealm)).Should(Succeed())
		Eventually(func() bool {
			deletedRealm := &keycloakAlpha.ClusterKeycloakRealm{}
			err := k8sClient.Get(ctx, types.NamespacedName{Name: keycloakRealm.Name}, deletedRealm)

			return k8sErrors.IsNotFound(err)
		}, timeout, interval).Should(BeTrue(), "ClusterKeycloakRealm with preserveResourcesOnDeletion annotation should be deleted")
	})
	It("Should create ClusterKeycloakRealm with Login settings", func() {
		By("Creating ClusterKeycloakRealm with Login settings")
		keycloakRealmWithLogin := &keycloakAlpha.ClusterKeycloakRealm{
			ObjectMeta: metav1.ObjectMeta{
				Name: "test-cluster-keycloak-realm-login",
			},
			Spec: keycloakAlpha.ClusterKeycloakRealmSpec{
				ClusterKeycloakRef: ClusterKeycloakCR,
				RealmName:          "test-realm-login",
				Login: &keycloakApi.RealmLogin{
					UserRegistration: true,
					ForgotPassword:   true,
					RememberMe:       true,
					EmailAsUsername:  false,
					LoginWithEmail:   true,
					DuplicateEmails:  false,
					VerifyEmail:      true,
					EditUsername:     false,
				},
			},
		}
		Expect(k8sClient.Create(ctx, keycloakRealmWithLogin)).Should(Succeed())

		By("Waiting for ClusterKeycloakRealm to be available")
		Eventually(func() bool {
			createdKeycloakRealm := &keycloakAlpha.ClusterKeycloakRealm{}
			err := k8sClient.Get(ctx, types.NamespacedName{Name: "test-cluster-keycloak-realm-login"}, createdKeycloakRealm)
			Expect(err).ShouldNot(HaveOccurred())

			return createdKeycloakRealm.Status.Available
		}, timeout, interval).Should(BeTrue())

		By("Verifying the realm login settings in Keycloak")
		Eventually(func(g Gomega) {
			realm, _, err := keycloakApiClient.Realms.GetRealm(ctx, "test-realm-login")
			g.Expect(err).ShouldNot(HaveOccurred())
			g.Expect(realm).ShouldNot(BeNil())

			// Verify login settings
			g.Expect(realm.RegistrationAllowed).Should(Equal(ptr.To(true)))
			g.Expect(realm.ResetPasswordAllowed).Should(Equal(ptr.To(true)))
			g.Expect(realm.RememberMe).Should(Equal(ptr.To(true)))
			g.Expect(realm.RegistrationEmailAsUsername).Should(Equal(ptr.To(false)))
			g.Expect(realm.LoginWithEmailAllowed).Should(Equal(ptr.To(true)))
			g.Expect(realm.DuplicateEmailsAllowed).Should(Equal(ptr.To(false)))
			g.Expect(realm.VerifyEmail).Should(Equal(ptr.To(true)))
			g.Expect(realm.EditUsernameAllowed).Should(Equal(ptr.To(false)))
		}, time.Second*10, time.Second).Should(Succeed())

		By("Deleting ClusterKeycloakRealm with Login settings")
		Expect(k8sClient.Delete(ctx, keycloakRealmWithLogin)).Should(Succeed())
		Eventually(func() bool {
			deletedRealm := &keycloakAlpha.ClusterKeycloakRealm{}
			err := k8sClient.Get(ctx, types.NamespacedName{Name: "test-cluster-keycloak-realm-login"}, deletedRealm)

			return k8sErrors.IsNotFound(err)
		}, timeout, interval).Should(BeTrue(), "ClusterKeycloakRealm with Login should be deleted")
	})
	It("Should create ClusterKeycloakRealm with SSO Session settings", func() {
		By("Creating ClusterKeycloakRealm with SSO Session settings")
		keycloakRealmWithSSO := &keycloakAlpha.ClusterKeycloakRealm{
			ObjectMeta: metav1.ObjectMeta{
				Name: "test-cluster-keycloak-realm-sso-session",
			},
			Spec: keycloakAlpha.ClusterKeycloakRealmSpec{
				ClusterKeycloakRef: ClusterKeycloakCR,
				RealmName:          "test-realm-sso-session",
				Sessions: &common.RealmSessions{
					SSOSessionSettings: &common.RealmSSOSessionSettings{
						IdleTimeout:           1801,
						MaxLifespan:           36002,
						IdleTimeoutRememberMe: 3603,
						MaxLifespanRememberMe: 72004,
					},
					SSOOfflineSessionSettings: &common.RealmSSOOfflineSessionSettings{
						IdleTimeout:        2592007,
						MaxLifespanEnabled: true,
						MaxLifespan:        5184008,
					},
					SSOLoginSettings: &common.RealmSSOLoginSettings{
						AccessCodeLifespanLogin:      1809,
						AccessCodeLifespanUserAction: 310,
					},
				},
			},
		}
		Expect(k8sClient.Create(ctx, keycloakRealmWithSSO)).Should(Succeed())

		By("Waiting for ClusterKeycloakRealm to be available")
		Eventually(func() bool {
			createdKeycloakRealm := &keycloakAlpha.ClusterKeycloakRealm{}
			err := k8sClient.Get(ctx, types.NamespacedName{Name: "test-cluster-keycloak-realm-sso-session"}, createdKeycloakRealm)
			Expect(err).ShouldNot(HaveOccurred())

			return createdKeycloakRealm.Status.Available
		}, timeout, interval).Should(BeTrue())

		By("Verifying the realm SSO session settings in Keycloak")
		Eventually(func(g Gomega) {
			realm, _, err := keycloakApiClient.Realms.GetRealm(ctx, "test-realm-sso-session")
			g.Expect(err).ShouldNot(HaveOccurred())
			g.Expect(realm).ShouldNot(BeNil())

			// Verify SSO session settings
			g.Expect(realm.SsoSessionIdleTimeout).Should(Equal(ptr.To(int32(1801))))
			g.Expect(realm.SsoSessionMaxLifespan).Should(Equal(ptr.To(int32(36002))))
			g.Expect(realm.SsoSessionIdleTimeoutRememberMe).Should(Equal(ptr.To(int32(3603))))
			g.Expect(realm.SsoSessionMaxLifespanRememberMe).Should(Equal(ptr.To(int32(72004))))

			// Verify Offline session settings
			g.Expect(realm.OfflineSessionIdleTimeout).Should(Equal(ptr.To(int32(2592007))))
			g.Expect(realm.OfflineSessionMaxLifespanEnabled).Should(Equal(ptr.To(true)))
			g.Expect(realm.OfflineSessionMaxLifespan).Should(Equal(ptr.To(int32(5184008))))

			// Verify Login settings
			g.Expect(realm.AccessCodeLifespanLogin).Should(Equal(ptr.To(int32(1809))))
			g.Expect(realm.AccessCodeLifespanUserAction).Should(Equal(ptr.To(int32(310))))
		}, time.Second*10, time.Second).Should(Succeed())

		By("Deleting ClusterKeycloakRealm with SSO Session settings")
		Expect(k8sClient.Delete(ctx, keycloakRealmWithSSO)).Should(Succeed())
		Eventually(func() bool {
			deletedRealm := &keycloakAlpha.ClusterKeycloakRealm{}
			err := k8sClient.Get(ctx, types.NamespacedName{Name: "test-cluster-keycloak-realm-sso-session"}, deletedRealm)

			return k8sErrors.IsNotFound(err)
		}, timeout, interval).Should(BeTrue(), "ClusterKeycloakRealm with SSO Session should be deleted")
	})
})

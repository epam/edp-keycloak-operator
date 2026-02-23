package keycloakrealm

import (
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
)

var _ = Describe("KeycloakRealm controller", Ordered, func() {
	const (
		keycloakRealmCR = "test-keycloak-realm-cr"
	)
	It("Should create KeycloakRealm", func() {
		By("Creating secret for email configuration")
		emailSecret := &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "email-config",
				Namespace: ns,
			},
			StringData: map[string]string{
				"password": "test",
			},
		}
		Expect(k8sClient.Create(ctx, emailSecret)).Should(Succeed())

		By("Creating KeycloakRealm")
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
				BrowserFlow: ptr.To("browser"),
				RealmEventConfig: &common.RealmEventConfig{
					AdminEventsDetailsEnabled: false,
					AdminEventsEnabled:        true,
					EnabledEventTypes:         []string{"UPDATE_CONSENT_ERROR", "CLIENT_LOGIN"},
					EventsEnabled:             true,
					EventsExpiration:          15000,
					EventsListeners:           []string{"jboss-logging"},
					AdminEventsExpiration:     100,
				},
				PasswordPolicies: []common.PasswordPolicy{
					{
						Type:  "forceExpiredPasswordChange",
						Value: "365",
					},
				},
				FrontendURL: "https://test.com",
				Users: []keycloakApi.User{
					{
						Username: "keycloakrealm-user@mail.com",
					},
				},
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
				UserProfileConfig: &common.UserProfileConfig{
					Attributes: []common.UserProfileAttribute{
						{
							DisplayName: "Attribute 1",
							Group:       "test-group",
							Name:        "attr1",
							Multivalued: true,
							Permissions: &common.UserProfileAttributePermissions{
								Edit: []string{"admin"},
								View: []string{"admin"},
							},
							Required: &common.UserProfileAttributeRequired{
								Roles:  []string{"admin", "user"},
								Scopes: []string{"email"},
							},
							Selector: &common.UserProfileAttributeSelector{
								Scopes: []string{"roles"},
							},
							Annotations: map[string]string{
								"inputType": "text",
							},
							Validations: map[string]map[string]common.UserProfileAttributeValidation{
								"email": {
									"max-local-length": {
										IntVal: 64,
									},
								},
								"local-date": {},
								"multivalued": {
									"min": {
										StringVal: "1",
									},
									"max": {
										StringVal: "10",
									},
								},
								"options": {
									"options": {
										SliceVal: []string{"option1", "option2"},
									},
								},
							},
						},
						{
							Name:        "attr2",
							DisplayName: "Attribute 2",
							Permissions: &common.UserProfileAttributePermissions{
								Edit: []string{"admin"},
								View: []string{"admin"},
							},
							Validations: map[string]map[string]common.UserProfileAttributeValidation{
								"options": {
									"options": {
										SliceVal: []string{"option1", "option2"},
									},
								},
							},
						},
					},
					Groups: []common.UserProfileGroup{
						{
							Annotations:        map[string]string{"group": "test"},
							DisplayDescription: "Group description",
							DisplayHeader:      "Group header",
							Name:               "test-group",
						},
					},
				},
				Smtp: &common.SMTP{
					Template: common.EmailTemplate{
						From:               "from@mailcom",
						FromDisplayName:    "from test",
						ReplyTo:            "to@mail.com",
						ReplyToDisplayName: "to test",
						EnvelopeFrom:       "envelope@mail.com",
					},
					Connection: common.EmailConnection{
						Host:           "smtp-host",
						Port:           25,
						EnableSSL:      true,
						EnableStartTLS: true,
						Authentication: &common.EmailAuthentication{
							Username: common.SourceRefOrVal{
								Value: "username",
							},
							Password: common.SourceRef{
								SecretKeyRef: &common.SecretKeySelector{
									LocalObjectReference: corev1.LocalObjectReference{
										Name: "email-config",
									},
									Key: "password",
								},
							},
						},
					},
				},
			},
		}
		Expect(k8sClient.Create(ctx, keycloakRealm)).Should(Succeed())
		Eventually(func() bool {
			createdKeycloakRealm := &keycloakApi.KeycloakRealm{}
			err := k8sClient.Get(ctx, types.NamespacedName{Name: keycloakRealmCR, Namespace: ns}, createdKeycloakRealm)
			Expect(err).ShouldNot(HaveOccurred())

			if !createdKeycloakRealm.Status.Available {
				GinkgoWriter.Println("KeycloakRealm status error: ", createdKeycloakRealm.Status.Value)
			}

			return createdKeycloakRealm.Status.Available
		}, time.Minute, time.Second*5).Should(BeTrue())

		By("Verifying the realm was created in Keycloak")
		Eventually(func(g Gomega) {
			realm, _, err := keycloakApiClient.Realms.GetRealm(ctx, "test-realm-with-full-config")
			g.Expect(err).ShouldNot(HaveOccurred())
			g.Expect(realm).ShouldNot(BeNil())

			// Verify basic fields
			g.Expect(realm.DisplayName).Should(Equal(ptr.To("Test Realm")))
			g.Expect(realm.DisplayNameHtml).Should(Equal(ptr.To("<b>Test Realm</b>")))
			g.Expect(realm.BrowserFlow).Should(Equal(ptr.To("browser")))

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
			g.Expect(realm.AdminEventsDetailsEnabled).Should(Equal(ptr.To(false)))
			g.Expect(realm.AdminEventsEnabled).Should(Equal(ptr.To(true)))
			g.Expect(realm.EventsEnabled).Should(Equal(ptr.To(true)))
			g.Expect(realm.EventsExpiration).Should(Equal(ptr.To(int64(15000))))
			g.Expect(*realm.EventsListeners).Should(ContainElement("jboss-logging"))
			g.Expect(*realm.EnabledEventTypes).Should(ContainElements("UPDATE_CONSENT_ERROR", "CLIENT_LOGIN"))

			// Verify adminEventsExpiration stored as realm attribute
			g.Expect(realm.Attributes).ShouldNot(BeNil())
			g.Expect((*realm.Attributes)["adminEventsExpiration"]).Should(Equal("100"))

			// Verify frontendUrl stored as realm attribute
			g.Expect((*realm.Attributes)["frontendUrl"]).Should(Equal("https://test.com"))

			// Verify password policies
			g.Expect(realm.PasswordPolicy).ShouldNot(BeNil())
			g.Expect(*realm.PasswordPolicy).Should(ContainSubstring("forceExpiredPasswordChange(365)"))

			// Verify SMTP settings
			g.Expect(realm.SmtpServer).ShouldNot(BeNil())
			g.Expect((*realm.SmtpServer)["from"]).Should(Equal("from@mailcom"))
			g.Expect((*realm.SmtpServer)["fromDisplayName"]).Should(Equal("from test"))
			g.Expect((*realm.SmtpServer)["replyTo"]).Should(Equal("to@mail.com"))
			g.Expect((*realm.SmtpServer)["replyToDisplayName"]).Should(Equal("to test"))
			g.Expect((*realm.SmtpServer)["envelopeFrom"]).Should(Equal("envelope@mail.com"))
			g.Expect((*realm.SmtpServer)["host"]).Should(Equal("smtp-host"))
			g.Expect((*realm.SmtpServer)["port"]).Should(Equal("25"))
			g.Expect((*realm.SmtpServer)["ssl"]).Should(Equal("true"))
			g.Expect((*realm.SmtpServer)["starttls"]).Should(Equal("true"))
			g.Expect((*realm.SmtpServer)["auth"]).Should(Equal("true"))
			g.Expect((*realm.SmtpServer)["user"]).Should(Equal("username"))
		}, time.Second*10, time.Second).Should(Succeed())

		By("Verifying the user profile was configured in Keycloak")
		Eventually(func(g Gomega) {
			userProfile, _, err := keycloakApiClient.Users.GetUsersProfile(ctx, "test-realm-with-full-config")
			g.Expect(err).ShouldNot(HaveOccurred())
			g.Expect(userProfile).ShouldNot(BeNil())

			// Verify attributes
			g.Expect(userProfile.Attributes).ShouldNot(BeNil())
			attrNames := make([]string, 0, len(*userProfile.Attributes))
			for _, a := range *userProfile.Attributes {
				attrNames = append(attrNames, *a.Name)
			}
			g.Expect(attrNames).Should(ContainElements("attr1", "attr2"))

			// Verify groups
			g.Expect(userProfile.Groups).ShouldNot(BeNil())
			groupNames := make([]string, 0, len(*userProfile.Groups))
			for _, gr := range *userProfile.Groups {
				groupNames = append(groupNames, *gr.Name)
			}
			g.Expect(groupNames).Should(ContainElement("test-group"))
		}, time.Second*10, time.Second).Should(Succeed())

		By("Verifying the user was created in Keycloak")
		Eventually(func(g Gomega) {
			user, _, err := keycloakApiClient.Users.FindUserByUsername(ctx, "test-realm-with-full-config", "keycloakrealm-user@mail.com")
			g.Expect(err).ShouldNot(HaveOccurred())
			g.Expect(user).ShouldNot(BeNil())
		}, time.Second*10, time.Second).Should(Succeed())
	})
	It("Should update KeycloakRealm", func() {
		By("Getting KeycloakRealm")
		keycloakRealm := &keycloakApi.KeycloakRealm{}
		err := k8sClient.Get(ctx, types.NamespacedName{Name: keycloakRealmCR, Namespace: ns}, keycloakRealm)
		Expect(err).ShouldNot(HaveOccurred())

		By("Updating KeycloakRealm")
		keycloakRealm.Spec.FrontendURL = "https://test-updated.com"
		Expect(k8sClient.Update(ctx, keycloakRealm)).Should(Succeed())

		Eventually(func() bool {
			updatedKeycloakRealm := &keycloakApi.KeycloakRealm{}
			err := k8sClient.Get(ctx, types.NamespacedName{Name: keycloakRealmCR, Namespace: ns}, updatedKeycloakRealm)
			Expect(err).ShouldNot(HaveOccurred())

			return updatedKeycloakRealm.Status.Available && updatedKeycloakRealm.Spec.FrontendURL == "https://test-updated.com"
		}, timeout, interval).Should(BeTrue())

		By("Verifying the realm was updated in Keycloak")
		Eventually(func(g Gomega) {
			realm, _, err := keycloakApiClient.Realms.GetRealm(ctx, "test-realm-with-full-config")
			g.Expect(err).ShouldNot(HaveOccurred())
			g.Expect(realm).ShouldNot(BeNil())
			g.Expect(realm.Attributes).ShouldNot(BeNil())
			g.Expect((*realm.Attributes)["frontendUrl"]).Should(Equal("https://test-updated.com"))
		}, time.Second*10, time.Second).Should(Succeed())
	})
	It("Should delete KeycloakRealm", func() {
		By("Getting KeycloakRealm")
		keycloakRealm := &keycloakApi.KeycloakRealm{}
		err := k8sClient.Get(ctx, types.NamespacedName{Name: keycloakRealmCR, Namespace: ns}, keycloakRealm)
		Expect(err).ShouldNot(HaveOccurred())

		By("Deleting KeycloakRealm")
		Expect(k8sClient.Delete(ctx, keycloakRealm)).Should(Succeed())
		Eventually(func() bool {
			deletedRealm := &keycloakApi.KeycloakRealm{}
			err := k8sClient.Get(ctx, types.NamespacedName{Name: keycloakRealmCR, Namespace: ns}, deletedRealm)

			return k8sErrors.IsNotFound(err)
		}, timeout, interval).Should(BeTrue(), "KeycloakRealm should be deleted")
	})
	It("Should create KeycloakRealm with Login settings", func() {
		By("Creating KeycloakRealm with Login settings")
		keycloakRealmWithLogin := &keycloakApi.KeycloakRealm{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-keycloak-realm-login",
				Namespace: ns,
			},
			Spec: keycloakApi.KeycloakRealmSpec{
				RealmName: "test-realm-login",
				KeycloakRef: common.KeycloakRef{
					Name: keycloakCR,
					Kind: keycloakApi.KeycloakKind,
				},
				Login: &keycloakApi.RealmLogin{
					UserRegistration: true,
					ForgotPassword:   true,
					RememberMe:       true,
					EmailAsUsername:  true,
					LoginWithEmail:   true,
					DuplicateEmails:  false,
					VerifyEmail:      true,
					EditUsername:     true,
				},
			},
		}
		Expect(k8sClient.Create(ctx, keycloakRealmWithLogin)).Should(Succeed())

		By("Waiting for KeycloakRealm to be available")
		Eventually(func() bool {
			createdKeycloakRealm := &keycloakApi.KeycloakRealm{}
			err := k8sClient.Get(ctx, types.NamespacedName{Name: "test-keycloak-realm-login", Namespace: ns}, createdKeycloakRealm)
			Expect(err).ShouldNot(HaveOccurred())

			if !createdKeycloakRealm.Status.Available {
				GinkgoWriter.Println("KeycloakRealm status error: ", createdKeycloakRealm.Status.Value)
			}

			return createdKeycloakRealm.Status.Available
		}, time.Minute, time.Second*5).Should(BeTrue())

		By("Verifying the realm login settings in Keycloak")
		Eventually(func(g Gomega) {
			realm, _, err := keycloakApiClient.Realms.GetRealm(ctx, "test-realm-login")
			g.Expect(err).ShouldNot(HaveOccurred())
			g.Expect(realm).ShouldNot(BeNil())

			// Verify login settings
			g.Expect(realm.RegistrationAllowed).Should(Equal(ptr.To(true)))
			g.Expect(realm.ResetPasswordAllowed).Should(Equal(ptr.To(true)))
			g.Expect(realm.RememberMe).Should(Equal(ptr.To(true)))
			g.Expect(realm.RegistrationEmailAsUsername).Should(Equal(ptr.To(true)))
			g.Expect(realm.LoginWithEmailAllowed).Should(Equal(ptr.To(true)))
			g.Expect(realm.DuplicateEmailsAllowed).Should(Equal(ptr.To(false)))
			g.Expect(realm.VerifyEmail).Should(Equal(ptr.To(true)))
			g.Expect(realm.EditUsernameAllowed).Should(Equal(ptr.To(true)))
		}, time.Second*10, time.Second).Should(Succeed())

		By("Deleting KeycloakRealm with Login settings")
		Expect(k8sClient.Delete(ctx, keycloakRealmWithLogin)).Should(Succeed())
		Eventually(func() bool {
			deletedRealm := &keycloakApi.KeycloakRealm{}
			err := k8sClient.Get(ctx, types.NamespacedName{Name: "test-keycloak-realm-login", Namespace: ns}, deletedRealm)

			return k8sErrors.IsNotFound(err)
		}, timeout, interval).Should(BeTrue(), "KeycloakRealm with Login should be deleted")
	})
	It("Should create KeycloakRealm with SSO Session settings", func() {
		By("Creating KeycloakRealm with SSO Session settings")
		keycloakRealmWithSSO := &keycloakApi.KeycloakRealm{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-keycloak-realm-sso-session",
				Namespace: ns,
			},
			Spec: keycloakApi.KeycloakRealmSpec{
				RealmName: "test-realm-sso-session",
				KeycloakRef: common.KeycloakRef{
					Name: keycloakCR,
					Kind: keycloakApi.KeycloakKind,
				},
				Login: &keycloakApi.RealmLogin{
					RememberMe: true,
				},
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

		By("Waiting for KeycloakRealm to be available")
		Eventually(func() bool {
			createdKeycloakRealm := &keycloakApi.KeycloakRealm{}
			err := k8sClient.Get(ctx, types.NamespacedName{Name: "test-keycloak-realm-sso-session", Namespace: ns}, createdKeycloakRealm)
			Expect(err).ShouldNot(HaveOccurred())

			if !createdKeycloakRealm.Status.Available {
				GinkgoWriter.Println("KeycloakRealm status error: ", createdKeycloakRealm.Status.Value)
			}

			return createdKeycloakRealm.Status.Available
		}, time.Minute, time.Second*5).Should(BeTrue())

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

		By("Deleting KeycloakRealm with SSO Session settings")
		Expect(k8sClient.Delete(ctx, keycloakRealmWithSSO)).Should(Succeed())
		Eventually(func() bool {
			deletedRealm := &keycloakApi.KeycloakRealm{}
			err := k8sClient.Get(ctx, types.NamespacedName{Name: "test-keycloak-realm-sso-session", Namespace: ns}, deletedRealm)

			return k8sErrors.IsNotFound(err)
		}, timeout, interval).Should(BeTrue(), "KeycloakRealm with SSO Session should be deleted")
	})
})

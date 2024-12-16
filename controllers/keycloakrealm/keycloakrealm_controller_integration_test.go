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
				RealmEventConfig: &keycloakApi.RealmEventConfig{
					AdminEventsDetailsEnabled: false,
					AdminEventsEnabled:        true,
					EnabledEventTypes:         []string{"UPDATE_CONSENT_ERROR", "CLIENT_LOGIN"},
					EventsEnabled:             true,
					EventsExpiration:          15000,
					EventsListeners:           []string{"jboss-logging"},
				},
				PasswordPolicies: []keycloakApi.PasswordPolicy{
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
})

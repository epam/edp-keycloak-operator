package keycloakrealm

import (
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

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
	)
	It("Should create KeycloakRealm", func() {
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
				BrowserFlow: pointer.String("browser"),
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

package keycloakauthflow

import (
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	k8sErrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	"github.com/epam/edp-keycloak-operator/api/common"
	keycloakApi "github.com/epam/edp-keycloak-operator/api/v1"
	"github.com/epam/edp-keycloak-operator/controllers/helper"
)

var _ = Describe("KeycloakAuthFlow controller", Ordered, func() {
	It("Should create KeycloakAuthFlow", func() {
		By("Creating a KeycloakAuthFlow")
		authFlow := &keycloakApi.KeycloakAuthFlow{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-auth-flow",
				Namespace: ns,
			},
			Spec: keycloakApi.KeycloakAuthFlowSpec{
				RealmRef: common.RealmRef{
					Kind: keycloakApi.KeycloakRealmKind,
					Name: KeycloakRealmCR,
				},
				Alias:       "test-auth-flow",
				Description: "Test auth flow",
				ProviderID:  "basic-flow",
				TopLevel:    true,
				AuthenticationExecutions: []keycloakApi.AuthenticationExecution{{
					Authenticator: "identity-provider-redirector",
					AuthenticatorConfig: &keycloakApi.AuthenticatorConfig{
						Alias:  "my-alias",
						Config: map[string]string{"defaultProvider": "my-provider"},
					},
					AuthenticatorFlow: false,
					Priority:          0,
					Requirement:       "REQUIRED",
					Alias:             "identity-provider-redirector-alias",
				}},
			},
		}
		Expect(k8sClient.Create(ctx, authFlow)).Should(Succeed())
		Eventually(func(g Gomega) {
			createdAuthFlow := &keycloakApi.KeycloakAuthFlow{}
			err := k8sClient.Get(ctx, types.NamespacedName{Name: authFlow.Name, Namespace: ns}, createdAuthFlow)
			g.Expect(err).ShouldNot(HaveOccurred())
			g.Expect(createdAuthFlow.Status.Value).Should(Equal(helper.StatusOK))
		}).WithTimeout(time.Second * 20).WithPolling(time.Second).Should(Succeed())
	})
	It("Should update KeycloakAuthFlow", func() {
		By("Getting KeycloakAuthFlow")
		createdAuthFlow := &keycloakApi.KeycloakAuthFlow{}
		Expect(k8sClient.Get(ctx, types.NamespacedName{Namespace: ns, Name: "test-auth-flow"}, createdAuthFlow)).Should(Succeed())

		By("Updating a KeycloakAuthFlow description")
		createdAuthFlow.Spec.Description = "new-description"

		Expect(k8sClient.Update(ctx, createdAuthFlow)).Should(Succeed())
		Consistently(func(g Gomega) {
			updatedUser := &keycloakApi.KeycloakAuthFlow{}
			err := k8sClient.Get(ctx, types.NamespacedName{Name: createdAuthFlow.Name, Namespace: ns}, updatedUser)
			g.Expect(err).ShouldNot(HaveOccurred())
			g.Expect(updatedUser.Status.Value).Should(Equal(helper.StatusOK))
		}, time.Second*5, time.Second).Should(Succeed())
	})
	It("Should delete KeycloakAuthFlow", func() {
		By("Getting KeycloakAuthFlow")
		createdAuthFlow := &keycloakApi.KeycloakAuthFlow{}
		Expect(k8sClient.Get(ctx, types.NamespacedName{Namespace: ns, Name: "test-auth-flow"}, createdAuthFlow)).Should(Succeed())

		By("Deleting KeycloakAuthFlow")
		Expect(k8sClient.Delete(ctx, createdAuthFlow)).Should(Succeed())
		Eventually(func(g Gomega) {
			deletedUser := &keycloakApi.KeycloakAuthFlow{}
			err := k8sClient.Get(ctx, types.NamespacedName{Name: createdAuthFlow.Name, Namespace: ns}, deletedUser)

			g.Expect(k8sErrors.IsNotFound(err)).Should(BeTrue())
		}, time.Second*30, time.Second).Should(Succeed())
	})
	It("Should fail to create KeycloakAuthFlow with invalid execution", func() {
		By("Creating a KeycloakAuthFlow")
		authFlow := &keycloakApi.KeycloakAuthFlow{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-auth-flow-with-invalid-execution",
				Namespace: ns,
			},
			Spec: keycloakApi.KeycloakAuthFlowSpec{
				RealmRef: common.RealmRef{
					Kind: keycloakApi.KeycloakRealmKind,
					Name: KeycloakRealmCR,
				},
				Alias:       "test-auth-flow-with-invalid-execution",
				Description: "Test auth flow with invalid execution",
				ProviderID:  "basic-flow-invalid",
				TopLevel:    true,
				AuthenticationExecutions: []keycloakApi.AuthenticationExecution{{
					Authenticator: "invalid-authenticator",
					AuthenticatorConfig: &keycloakApi.AuthenticatorConfig{
						Alias:  "my-alias",
						Config: map[string]string{"defaultProvider": "my-provider"},
					},
					AuthenticatorFlow: true,
					Priority:          0,
					Requirement:       "invalid",
					Alias:             "identity-provider-redirector-alias",
				}},
			},
		}
		Expect(k8sClient.Create(ctx, authFlow)).Should(Succeed())

		By("Waiting for KeycloakAuthFlow reconciliation")
		time.Sleep(time.Second * 5)

		By("Checking KeycloakAuthFlow status")
		Consistently(func(g Gomega) {
			createdAuthFlow := &keycloakApi.KeycloakAuthFlow{}
			err := k8sClient.Get(ctx, types.NamespacedName{Name: authFlow.Name, Namespace: ns}, createdAuthFlow)
			g.Expect(err).ShouldNot(HaveOccurred())
			g.Expect(createdAuthFlow.Status.Value).Should(ContainSubstring("unable to sync auth flow"))
		}).WithTimeout(time.Second * 10).WithPolling(time.Second).Should(Succeed())
	})
})

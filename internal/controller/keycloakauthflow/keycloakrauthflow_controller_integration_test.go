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
			g.Expect(createdAuthFlow.Status.Value).Should(Equal(common.StatusOK))
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
			g.Expect(updatedUser.Status.Value).Should(Equal(common.StatusOK))
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
	It("Should create child KeycloakAuthFlow", func() {
		By("Creating a parent KeycloakAuthFlow")
		parentAuthFlow := &keycloakApi.KeycloakAuthFlow{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-auth-flow-parent",
				Namespace: ns,
			},
			Spec: keycloakApi.KeycloakAuthFlowSpec{
				RealmRef: common.RealmRef{
					Kind: keycloakApi.KeycloakRealmKind,
					Name: KeycloakRealmCR,
				},
				Alias:       "test-auth-flow-parent",
				Description: "test-auth-flow-parent",
				ProviderID:  "basic-flow",
				TopLevel:    true,
			},
		}
		Expect(k8sClient.Create(ctx, parentAuthFlow)).Should(Succeed())
		Eventually(func(g Gomega) {
			createdParentAuthFlow := &keycloakApi.KeycloakAuthFlow{}
			err := k8sClient.Get(ctx, types.NamespacedName{Name: parentAuthFlow.Name, Namespace: ns}, createdParentAuthFlow)
			g.Expect(err).ShouldNot(HaveOccurred())
			g.Expect(createdParentAuthFlow.Status.Value).Should(Equal(common.StatusOK))
		}).WithTimeout(time.Second * 20).WithPolling(time.Second).Should(Succeed())
		By("Creating a child KeycloakAuthFlow")
		childAuthFlow := &keycloakApi.KeycloakAuthFlow{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-auth-flow-child",
				Namespace: ns,
			},
			Spec: keycloakApi.KeycloakAuthFlowSpec{
				RealmRef: common.RealmRef{
					Kind: keycloakApi.KeycloakRealmKind,
					Name: KeycloakRealmCR,
				},
				Alias:            "test-auth-flow-child",
				Description:      "test-auth-flow-child",
				ProviderID:       "basic-flow",
				ParentName:       parentAuthFlow.Name,
				ChildType:        "basic-flow",
				ChildRequirement: "REQUIRED",
			},
		}
		Expect(k8sClient.Create(ctx, childAuthFlow)).Should(Succeed())
		Eventually(func(g Gomega) {
			createdChildAuthFlow := &keycloakApi.KeycloakAuthFlow{}
			err := k8sClient.Get(ctx, types.NamespacedName{Name: childAuthFlow.Name, Namespace: ns}, createdChildAuthFlow)
			g.Expect(err).ShouldNot(HaveOccurred())
			g.Expect(createdChildAuthFlow.Status.Value).Should(Equal(common.StatusOK))
		}).WithTimeout(time.Second * 20).WithPolling(time.Second).Should(Succeed())
	})
})

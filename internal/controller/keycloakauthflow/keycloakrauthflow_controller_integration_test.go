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
	"github.com/epam/edp-keycloak-operator/pkg/client/keycloakapi"
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
			g.Expect(createdAuthFlow.Status.ID).ShouldNot(BeEmpty())
		}).WithTimeout(time.Second * 20).WithPolling(time.Second).Should(Succeed())

		By("Verifying auth flow fields via Keycloak API")
		Eventually(func(g Gomega) {
			flows, _, err := keycloakApiClient.AuthFlows.GetAuthFlows(ctx, KeycloakRealmCR)
			g.Expect(err).ShouldNot(HaveOccurred())

			flow := findAuthFlowByAlias(flows, "test-auth-flow")
			g.Expect(flow).ShouldNot(BeNil())
			g.Expect(flow.Alias).ShouldNot(BeNil())
			g.Expect(*flow.Alias).Should(Equal("test-auth-flow"))
			g.Expect(flow.Description).ShouldNot(BeNil())
			g.Expect(*flow.Description).Should(Equal("Test auth flow"))
			g.Expect(flow.ProviderId).ShouldNot(BeNil())
			g.Expect(*flow.ProviderId).Should(Equal("basic-flow"))
			g.Expect(flow.TopLevel).ShouldNot(BeNil())
			g.Expect(*flow.TopLevel).Should(BeTrue())
			g.Expect(flow.BuiltIn).ShouldNot(BeNil())
			g.Expect(*flow.BuiltIn).Should(BeFalse())
		}, timeout, interval).Should(Succeed())

		By("Verifying auth flow executions via Keycloak API")
		Eventually(func(g Gomega) {
			execs, _, err := keycloakApiClient.AuthFlows.GetFlowExecutions(ctx, KeycloakRealmCR, "test-auth-flow")
			g.Expect(err).ShouldNot(HaveOccurred())

			exec := findExecutionByProviderId(execs, "identity-provider-redirector")
			g.Expect(exec).ShouldNot(BeNil())
			g.Expect(exec.Requirement).ShouldNot(BeNil())
			g.Expect(*exec.Requirement).Should(Equal("REQUIRED"))

			g.Expect(exec.AuthenticationConfig).ShouldNot(BeNil())
			cfg, _, err := keycloakApiClient.AuthFlows.GetAuthenticatorConfig(ctx, KeycloakRealmCR, *exec.AuthenticationConfig)
			g.Expect(err).ShouldNot(HaveOccurred())
			g.Expect(cfg).ShouldNot(BeNil())
			g.Expect(cfg.Alias).ShouldNot(BeNil())
			g.Expect(*cfg.Alias).Should(Equal("my-alias"))
			g.Expect(cfg.Config).ShouldNot(BeNil())
			g.Expect((*cfg.Config)["defaultProvider"]).Should(Equal("my-provider"))
		}, timeout, interval).Should(Succeed())
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

		By("Verifying the updated description via Keycloak API")
		Eventually(func(g Gomega) {
			flows, _, err := keycloakApiClient.AuthFlows.GetAuthFlows(ctx, KeycloakRealmCR)
			g.Expect(err).ShouldNot(HaveOccurred())

			flow := findAuthFlowByAlias(flows, "test-auth-flow")
			g.Expect(flow).ShouldNot(BeNil())
			g.Expect(flow.Description).ShouldNot(BeNil())
			g.Expect(*flow.Description).Should(Equal("new-description"))
		}, timeout, interval).Should(Succeed())
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
			g.Expect(createdAuthFlow.Status.Value).Should(ContainSubstring("auth flow chain processing failed"))
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
			g.Expect(createdChildAuthFlow.Status.ID).ShouldNot(BeEmpty())
		}).WithTimeout(time.Second * 20).WithPolling(time.Second).Should(Succeed())

		By("Verifying child flow is listed in parent's executions via Keycloak API")
		Eventually(func(g Gomega) {
			execs, _, err := keycloakApiClient.AuthFlows.GetFlowExecutions(ctx, KeycloakRealmCR, "test-auth-flow-parent")
			g.Expect(err).ShouldNot(HaveOccurred())

			exec := findExecutionByDisplayName(execs, "test-auth-flow-child")
			g.Expect(exec).ShouldNot(BeNil())
			g.Expect(exec.AuthenticationFlow).ShouldNot(BeNil())
			g.Expect(*exec.AuthenticationFlow).Should(BeTrue())
			g.Expect(exec.Requirement).ShouldNot(BeNil())
			g.Expect(*exec.Requirement).Should(Equal("REQUIRED"))
		}, timeout, interval).Should(Succeed())
	})
})

func findAuthFlowByAlias(flows []keycloakapi.AuthFlowRepresentation, alias string) *keycloakapi.AuthFlowRepresentation {
	for i := range flows {
		if flows[i].Alias != nil && *flows[i].Alias == alias {
			return &flows[i]
		}
	}

	return nil
}

func findExecutionByProviderId(
	execs []keycloakapi.AuthenticationExecutionInfoRepresentation,
	providerID string,
) *keycloakapi.AuthenticationExecutionInfoRepresentation {
	for i := range execs {
		if execs[i].ProviderId != nil && *execs[i].ProviderId == providerID {
			return &execs[i]
		}
	}

	return nil
}

func findExecutionByDisplayName(
	execs []keycloakapi.AuthenticationExecutionInfoRepresentation,
	name string,
) *keycloakapi.AuthenticationExecutionInfoRepresentation {
	for i := range execs {
		if execs[i].DisplayName != nil && *execs[i].DisplayName == name {
			return &execs[i]
		}
	}

	return nil
}

package keycloakauthflow

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	k8sErrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/utils/ptr"

	"github.com/epam/edp-keycloak-operator/api/common"
	keycloakApi "github.com/epam/edp-keycloak-operator/api/v1"
	"github.com/epam/edp-keycloak-operator/pkg/testutils"
)

// Idempotency suite for KeycloakAuthFlow.
var _ = Describe("KeycloakAuthFlow idempotent reconcile", Ordered, func() {
	const (
		crName    = "idem-flow"
		flowAlias = "idem-flow"
		realmCR   = "idem-flow-realm"
		settle    = testutils.Settle
		longWait  = testutils.LongWait
	)

	var recorder *testutils.AdminEventRecorder

	nudge := func() {
		Expect(testutils.Nudge(ctx, k8sClient,
			types.NamespacedName{Name: crName, Namespace: ns}, &keycloakApi.KeycloakAuthFlow{})).To(Succeed())
	}

	updateSpec := func(mutate func(*keycloakApi.KeycloakAuthFlow)) {
		Eventually(func(g Gomega) {
			cr := &keycloakApi.KeycloakAuthFlow{}
			g.Expect(k8sClient.Get(ctx, types.NamespacedName{Name: crName, Namespace: ns}, cr)).To(Succeed())
			mutate(cr)
			g.Expect(k8sClient.Update(ctx, cr)).To(Succeed())
		}, longWait, interval).Should(Succeed())
	}

	waitReady := func() {
		Eventually(func(g Gomega) {
			cr := &keycloakApi.KeycloakAuthFlow{}
			g.Expect(k8sClient.Get(ctx, types.NamespacedName{Name: crName, Namespace: ns}, cr)).To(Succeed())
			g.Expect(cr.Status.Value).To(Equal(common.StatusOK))
			g.Expect(cr.Status.ObservedGeneration).To(Equal(cr.Generation))
		}, longWait, interval).Should(Succeed())
	}

	// execProvider returns the defaultProvider recorded on the flow's single execution,
	// resolved through the live Keycloak state rather than the CR.
	execProvider := func(g Gomega) string {
		execs, _, err := keycloakApiClient.AuthFlows.GetFlowExecutions(ctx, realmCR, flowAlias)
		g.Expect(err).ShouldNot(HaveOccurred())
		g.Expect(execs).ShouldNot(BeEmpty())
		g.Expect(execs[0].AuthenticationConfig).ShouldNot(BeNil(), "execution must carry an authenticator config")

		cfg, _, err := keycloakApiClient.AuthFlows.GetAuthenticatorConfig(ctx, realmCR, *execs[0].AuthenticationConfig)
		g.Expect(err).ShouldNot(HaveOccurred())
		g.Expect(cfg).ShouldNot(BeNil())
		g.Expect(cfg.Config).ShouldNot(BeNil())

		return (*cfg.Config)["defaultProvider"]
	}

	flowExists := func() bool {
		flows, _, err := keycloakApiClient.Realms.GetAuthenticationFlows(ctx, realmCR)
		Expect(err).ShouldNot(HaveOccurred())

		for _, f := range flows {
			if ptr.Deref(f.Alias, "") == flowAlias {
				return true
			}
		}

		return false
	}

	// This suite owns its realm. A flow's admin events land under disjoint resource
	// paths (authentication/flows, authentication/executions, authentication/config),
	// so the recorder cannot be scoped to one flow; realm isolation replaces scoping.
	BeforeAll(func() {
		realm := &keycloakApi.KeycloakRealm{
			ObjectMeta: metav1.ObjectMeta{Name: realmCR, Namespace: ns},
			Spec: keycloakApi.KeycloakRealmSpec{
				RealmName:   realmCR,
				KeycloakRef: common.KeycloakRef{Name: KeycloakCR, Kind: keycloakApi.KeycloakKind},
			},
		}
		Expect(k8sClient.Create(ctx, realm)).To(Succeed())

		Eventually(func(g Gomega) {
			got := &keycloakApi.KeycloakRealm{}
			g.Expect(k8sClient.Get(ctx, types.NamespacedName{Name: realmCR, Namespace: ns}, got)).To(Succeed())
			g.Expect(got.Status.Available).To(BeTrue())
		}, longWait, interval).Should(Succeed())

		recorder = testutils.NewAdminEventRecorder(keycloakApiClient, realmCR)
		Expect(recorder.Enable(ctx)).To(Succeed())
	})

	AfterAll(func() {
		realm := &keycloakApi.KeycloakRealm{}
		if err := k8sClient.Get(ctx, types.NamespacedName{Name: realmCR, Namespace: ns}, realm); err == nil {
			Expect(k8sClient.Delete(ctx, realm)).To(Succeed())
		}
	})

	It("Should create KeycloakAuthFlow with an execution", func() {
		cr := &keycloakApi.KeycloakAuthFlow{
			ObjectMeta: metav1.ObjectMeta{Name: crName, Namespace: ns},
			Spec: keycloakApi.KeycloakAuthFlowSpec{
				RealmRef:    common.RealmRef{Kind: keycloakApi.KeycloakRealmKind, Name: realmCR},
				Alias:       flowAlias,
				Description: "idem flow v1",
				ProviderID:  "basic-flow",
				TopLevel:    true,
				AuthenticationExecutions: []keycloakApi.AuthenticationExecution{{
					Authenticator: "identity-provider-redirector",
					AuthenticatorConfig: &keycloakApi.AuthenticatorConfig{
						Alias:  "idem-flow-cfg",
						Config: map[string]string{"defaultProvider": "provider-v1"},
					},
					AuthenticatorFlow: false,
					Priority:          0,
					Requirement:       "REQUIRED",
					Alias:             "idem-flow-exec",
				}},
			},
		}
		Expect(k8sClient.Create(ctx, cr)).To(Succeed())
		waitReady()

		Expect(flowExists()).To(BeTrue())
		Eventually(func(g Gomega) {
			g.Expect(execProvider(g)).To(Equal("provider-v1"))
		}, longWait, interval).Should(Succeed())
	})

	It("Should not write to Keycloak when a reconcile is forced with no spec change", func() {
		Expect(recorder.WritesDuring(ctx, settle, nudge)).To(BeEmpty())
	})

	It("Should write to Keycloak and converge when an execution config changes", func() {
		// Liveness anchor for this suite's empty-window specs.
		Expect(recorder.WritesDuring(ctx, 0, func() {
			updateSpec(func(cr *keycloakApi.KeycloakAuthFlow) {
				cr.Spec.AuthenticationExecutions[0].AuthenticatorConfig.Config["defaultProvider"] = "provider-v2"
			})
			waitReady()

			Expect(flowExists()).To(BeTrue())

			// The edit must reach the authenticator config in Keycloak, not just bump the flow.
			Eventually(func(g Gomega) {
				g.Expect(execProvider(g)).To(Equal("provider-v2"))
			}, longWait, interval).Should(Succeed())
		})).ShouldNot(BeEmpty(), "admin event log recorded nothing for a known write")
	})

	It("Should not write to Keycloak after the edit settles", func() {
		Expect(recorder.WritesDuring(ctx, settle, nudge)).To(BeEmpty())
	})

	It("Should force one write when observedGeneration is cleared", func() {
		// No drift: Keycloak matches spec here, so a write can only come from the
		// cleared observedGeneration.
		Expect(recorder.WritesDuring(ctx, 0, func() {
			Expect(testutils.ClearObservedGeneration(ctx, k8sClient,
				types.NamespacedName{Name: crName, Namespace: ns},
				&keycloakApi.KeycloakAuthFlow{})).To(Succeed())
		})).ShouldNot(BeEmpty(), "a cleared observedGeneration must force one write")
	})

	It("Should delete KeycloakAuthFlow and remove the flow from Keycloak", func() {
		cr := &keycloakApi.KeycloakAuthFlow{}
		Expect(k8sClient.Get(ctx, types.NamespacedName{Name: crName, Namespace: ns}, cr)).To(Succeed())
		Expect(k8sClient.Delete(ctx, cr)).To(Succeed())

		Eventually(func() bool {
			return k8sErrors.IsNotFound(
				k8sClient.Get(ctx, types.NamespacedName{Name: crName, Namespace: ns}, &keycloakApi.KeycloakAuthFlow{}))
		}, longWait, interval).Should(BeTrue())

		Eventually(func(g Gomega) {
			g.Expect(flowExists()).To(BeFalse(), "flow must be gone from Keycloak")
		}, longWait, interval).Should(Succeed())
	})
})

package keycloakrealmcomponent

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	k8sErrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/utils/ptr"

	"github.com/epam/edp-keycloak-operator/api/common"
	keycloakApi "github.com/epam/edp-keycloak-operator/api/v1"
	"github.com/epam/edp-keycloak-operator/pkg/client/keycloakapi"
	"github.com/epam/edp-keycloak-operator/pkg/testutils"
)

// Idempotency suite for KeycloakRealmComponent.
var _ = Describe("KeycloakRealmComponent idempotent reconcile", Ordered, func() {
	const (
		crName        = "idem-component"
		componentName = "idem-component"
		settle        = testutils.Settle
		longWait      = testutils.LongWait
	)

	var (
		recorder    *testutils.AdminEventRecorder
		componentID string
	)

	nudge := func() {
		Expect(testutils.Nudge(ctx, k8sClient,
			types.NamespacedName{Name: crName, Namespace: ns}, &keycloakApi.KeycloakRealmComponent{})).To(Succeed())
	}

	updateSpec := func(mutate func(*keycloakApi.KeycloakRealmComponent)) {
		Eventually(func(g Gomega) {
			cr := &keycloakApi.KeycloakRealmComponent{}
			g.Expect(k8sClient.Get(ctx, types.NamespacedName{Name: crName, Namespace: ns}, cr)).To(Succeed())
			mutate(cr)
			g.Expect(k8sClient.Update(ctx, cr)).To(Succeed())
		}, longWait, interval).Should(Succeed())
	}

	waitReady := func() {
		Eventually(func(g Gomega) {
			cr := &keycloakApi.KeycloakRealmComponent{}
			g.Expect(k8sClient.Get(ctx, types.NamespacedName{Name: crName, Namespace: ns}, cr)).To(Succeed())
			g.Expect(cr.Status.Value).To(Equal(common.StatusOK))
			g.Expect(cr.Status.ObservedGeneration).To(Equal(cr.Generation))
		}, longWait, interval).Should(Succeed())
	}

	kcComponent := func() *keycloakapi.ComponentRepresentation {
		rep, err := keycloakApiClient.RealmComponents.FindComponentByName(ctx, KeycloakRealmCR, componentName)
		Expect(err).ShouldNot(HaveOccurred())
		Expect(rep).ShouldNot(BeNil())

		return rep
	}

	BeforeAll(func() {
		recorder = testutils.NewAdminEventRecorder(keycloakApiClient, KeycloakRealmCR)
		Expect(recorder.Enable(ctx)).To(Succeed())
	})

	It("Should create KeycloakRealmComponent with config", func() {
		cr := &keycloakApi.KeycloakRealmComponent{
			ObjectMeta: metav1.ObjectMeta{Name: crName, Namespace: ns},
			Spec: keycloakApi.KeycloakComponentSpec{
				Name:         componentName,
				ProviderID:   "max-clients",
				ProviderType: "org.keycloak.services.clientregistration.policy.ClientRegistrationPolicy",
				RealmRef:     common.RealmRef{Kind: keycloakApi.KeycloakRealmKind, Name: KeycloakRealmCR},
				Config: map[string][]string{
					"max-clients": {"100"},
				},
			},
		}
		Expect(k8sClient.Create(ctx, cr)).To(Succeed())
		waitReady()

		created := kcComponent()
		componentID = ptr.Deref(created.Id, "")
		Expect(componentID).ShouldNot(BeEmpty())
		Expect((*created.Config)["max-clients"]).To(ConsistOf("100"))

		// Sibling specs share the realm; report only this component's events.
		recorder = recorder.Scoped("components/" + componentID)
	})

	It("Should not write to Keycloak on a forced reconcile and keep the component ID", func() {
		Expect(recorder.WritesDuring(ctx, settle, nudge)).To(BeEmpty())
		Expect(ptr.Deref(kcComponent().Id, "")).To(Equal(componentID), "component must not be recreated")
	})

	It("Should heal external drift on the component config", func() {
		// Liveness anchor for this suite's empty-window specs. The drift write sits
		// inside the window: the controller reconciles on a timer and can heal it
		// before the nudge lands.
		Expect(recorder.WritesDuring(ctx, 0, func() {
			rep := kcComponent()
			cfg := *rep.Config
			cfg["max-clients"] = []string{"999"}
			rep.Config = &cfg
			_, err := keycloakApiClient.RealmComponents.UpdateComponent(ctx, KeycloakRealmCR, componentID, *rep)
			Expect(err).ShouldNot(HaveOccurred())

			nudge()

			Eventually(func(g Gomega) {
				g.Expect((*kcComponent().Config)["max-clients"]).To(ConsistOf("100"))
			}, longWait, interval).Should(Succeed())
		})).ShouldNot(BeEmpty(), "admin event log recorded nothing for a known write")
	})

	It("Should write to Keycloak and keep the component ID when a config value changes", func() {
		updateSpec(func(cr *keycloakApi.KeycloakRealmComponent) {
			cr.Spec.Config["max-clients"] = []string{"250"}
		})
		waitReady()

		Eventually(func(g Gomega) {
			g.Expect((*kcComponent().Config)["max-clients"]).To(ConsistOf("250"))
		}, longWait, interval).Should(Succeed())

		Expect(ptr.Deref(kcComponent().Id, "")).To(Equal(componentID),
			"an update must not delete and recreate the component")
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
				&keycloakApi.KeycloakRealmComponent{})).To(Succeed())
		})).ShouldNot(BeEmpty(), "a cleared observedGeneration must force one write")
	})

	It("Should delete KeycloakRealmComponent and remove the component from Keycloak", func() {
		cr := &keycloakApi.KeycloakRealmComponent{}
		Expect(k8sClient.Get(ctx, types.NamespacedName{Name: crName, Namespace: ns}, cr)).To(Succeed())
		Expect(k8sClient.Delete(ctx, cr)).To(Succeed())

		Eventually(func() bool {
			return k8sErrors.IsNotFound(
				k8sClient.Get(ctx, types.NamespacedName{Name: crName, Namespace: ns},
					&keycloakApi.KeycloakRealmComponent{}))
		}, longWait, interval).Should(BeTrue())

		Eventually(func(g Gomega) {
			rep, err := keycloakApiClient.RealmComponents.FindComponentByName(ctx, KeycloakRealmCR, componentName)
			g.Expect(err).ShouldNot(HaveOccurred())
			g.Expect(rep).To(BeNil(), "component must be gone from Keycloak")
		}, longWait, interval).Should(Succeed())
	})
})

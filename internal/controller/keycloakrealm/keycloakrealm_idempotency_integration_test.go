package keycloakrealm

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

// Idempotency suite for KeycloakRealm.
//
// The CR owns spec.realmEventConfig. The realm PUT carries adminEventsEnabled and
// resets the event log without it.
var _ = Describe("KeycloakRealm idempotent reconcile", Ordered, func() {
	const (
		crName   = "idem-realm"
		settle   = testutils.Settle
		longWait = testutils.LongWait
	)

	realmName := testutils.RealmName("idem-realm")

	var recorder *testutils.AdminEventRecorder

	nudge := func() {
		Expect(testutils.Nudge(ctx, k8sClient,
			types.NamespacedName{Name: crName, Namespace: ns}, &keycloakApi.KeycloakRealm{})).To(Succeed())
	}

	updateSpec := func(mutate func(*keycloakApi.KeycloakRealm)) {
		Eventually(func(g Gomega) {
			cr := &keycloakApi.KeycloakRealm{}
			g.Expect(k8sClient.Get(ctx, types.NamespacedName{Name: crName, Namespace: ns}, cr)).To(Succeed())
			mutate(cr)
			g.Expect(k8sClient.Update(ctx, cr)).To(Succeed())
		}, longWait, interval).Should(Succeed())
	}

	waitReady := func() {
		Eventually(func(g Gomega) {
			cr := &keycloakApi.KeycloakRealm{}
			g.Expect(k8sClient.Get(ctx, types.NamespacedName{Name: crName, Namespace: ns}, cr)).To(Succeed())
			g.Expect(cr.Status.Available).To(BeTrue())
			g.Expect(cr.Status.ObservedGeneration).To(Equal(cr.Generation))
		}, longWait, interval).Should(Succeed())
	}

	kcRealm := func() *keycloakapi.RealmRepresentation {
		rep, _, err := keycloakApiClient.Realms.GetRealm(ctx, realmName)
		Expect(err).ShouldNot(HaveOccurred())
		Expect(rep).ShouldNot(BeNil())

		return rep
	}

	It("Should create KeycloakRealm with locales and event settings", func() {
		cr := &keycloakApi.KeycloakRealm{
			ObjectMeta: metav1.ObjectMeta{Name: crName, Namespace: ns},
			Spec: keycloakApi.KeycloakRealmSpec{
				RealmName:   realmName,
				KeycloakRef: common.KeycloakRef{Name: keycloakCR, Kind: keycloakApi.KeycloakKind},
				DisplayName: ptr.To("Idem Realm"),
				RealmEventConfig: &common.RealmEventConfig{
					AdminEventsEnabled:        ptr.To(true),
					AdminEventsDetailsEnabled: ptr.To(true),
					EventsEnabled:             ptr.To(true),
					EnabledEventTypes:         []string{"LOGIN", "LOGOUT", "REGISTER"},
					EventsListeners:           []string{"jboss-logging"},
				},
				Localization: &keycloakApi.RealmLocalization{
					InternationalizationEnabled: ptr.To(true),
					SupportedLocales:            []string{"en", "de", "uk"},
				},
			},
		}
		Expect(k8sClient.Create(ctx, cr)).To(Succeed())
		waitReady()

		created := kcRealm()
		Expect(ptr.Deref(created.DisplayName, "")).To(Equal("Idem Realm"))
		Expect(*created.SupportedLocales).To(ConsistOf("en", "de", "uk"))

		recorder = testutils.NewAdminEventRecorder(keycloakApiClient, realmName)
	})

	It("Should not write to Keycloak when a reconcile is forced with no spec change", func() {
		Expect(recorder.WritesDuring(ctx, settle, nudge)).To(BeEmpty())
	})

	It("Should heal external drift on the realm", func() {
		// Liveness anchor for this suite's empty-window specs. The drift write sits
		// inside the window: the controller reconciles on a timer and can heal it
		// before the nudge lands.
		Expect(recorder.WritesDuring(ctx, 0, func() {
			rep := kcRealm()
			rep.DisplayName = ptr.To("drifted-by-hand")
			_, err := keycloakApiClient.Realms.UpdateRealm(ctx, realmName, *rep)
			Expect(err).ShouldNot(HaveOccurred())

			nudge()

			Eventually(func(g Gomega) {
				g.Expect(ptr.Deref(kcRealm().DisplayName, "")).To(Equal("Idem Realm"))
			}, longWait, interval).Should(Succeed())
		})).ShouldNot(BeEmpty(), "admin event log recorded nothing for a known write")
	})

	It("Should settle without further writes when enabledEventTypes is reordered in spec", func() {
		updateSpec(func(cr *keycloakApi.KeycloakRealm) {
			cr.Spec.RealmEventConfig.EnabledEventTypes = []string{"REGISTER", "LOGIN", "LOGOUT"}
		})
		waitReady()

		Expect(recorder.WritesDuring(ctx, settle, func() {})).To(BeEmpty(),
			"an order-only spec edit must not keep writing")

		cfg, _, err := keycloakApiClient.Events.GetEventsConfig(ctx, realmName)
		Expect(err).ShouldNot(HaveOccurred())
		Expect(*cfg.EnabledEventTypes).To(ConsistOf("LOGIN", "LOGOUT", "REGISTER"))
	})

	It("Should keep the existing locales when a locale is added", func() {
		updateSpec(func(cr *keycloakApi.KeycloakRealm) {
			cr.Spec.Localization.SupportedLocales = []string{"en", "de", "uk", "fr"}
		})
		waitReady()

		Eventually(func(g Gomega) {
			g.Expect(*kcRealm().SupportedLocales).To(ConsistOf("en", "de", "uk", "fr"))
		}, longWait, interval).Should(Succeed())
	})

	It("Should remove a locale from Keycloak when it is removed from spec", func() {
		updateSpec(func(cr *keycloakApi.KeycloakRealm) {
			cr.Spec.Localization.SupportedLocales = []string{"en", "de"}
		})
		waitReady()

		Eventually(func(g Gomega) {
			g.Expect(*kcRealm().SupportedLocales).To(ConsistOf("en", "de"),
				"locale lists have set semantics, removals must propagate")
		}, longWait, interval).Should(Succeed())
	})

	It("Should not write to Keycloak after all edits settle", func() {
		Expect(recorder.WritesDuring(ctx, settle, nudge)).To(BeEmpty())
	})

	It("Should delete KeycloakRealm and remove the realm from Keycloak", func() {
		cr := &keycloakApi.KeycloakRealm{}
		Expect(k8sClient.Get(ctx, types.NamespacedName{Name: crName, Namespace: ns}, cr)).To(Succeed())
		Expect(k8sClient.Delete(ctx, cr)).To(Succeed())

		Eventually(func() bool {
			return k8sErrors.IsNotFound(
				k8sClient.Get(ctx, types.NamespacedName{Name: crName, Namespace: ns}, &keycloakApi.KeycloakRealm{}))
		}, longWait, interval).Should(BeTrue())

		Eventually(func(g Gomega) {
			_, _, err := keycloakApiClient.Realms.GetRealm(ctx, realmName)
			g.Expect(err).Should(HaveOccurred())
		}, longWait, interval).Should(Succeed())
	})
})

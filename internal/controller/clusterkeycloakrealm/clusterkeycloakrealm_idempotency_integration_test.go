package clusterkeycloakrealm

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	k8sErrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/utils/ptr"

	"github.com/epam/edp-keycloak-operator/api/common"
	keycloakAlpha "github.com/epam/edp-keycloak-operator/api/v1alpha1"
	"github.com/epam/edp-keycloak-operator/pkg/client/keycloakapi"
	"github.com/epam/edp-keycloak-operator/pkg/testutils"
)

// Idempotency suite for ClusterKeycloakRealm.
var _ = Describe("ClusterKeycloakRealm idempotent reconcile", Ordered, func() {
	const (
		crName    = "idem-cluster-realm"
		realmName = "idem-cluster-realm"
		settle    = testutils.Settle
		longWait  = testutils.LongWait
	)

	var recorder *testutils.AdminEventRecorder

	nudge := func() {
		Expect(testutils.Nudge(ctx, k8sClient,
			types.NamespacedName{Name: crName}, &keycloakAlpha.ClusterKeycloakRealm{})).To(Succeed())
	}

	updateSpec := func(mutate func(*keycloakAlpha.ClusterKeycloakRealm)) {
		Eventually(func(g Gomega) {
			cr := &keycloakAlpha.ClusterKeycloakRealm{}
			g.Expect(k8sClient.Get(ctx, types.NamespacedName{Name: crName}, cr)).To(Succeed())
			mutate(cr)
			g.Expect(k8sClient.Update(ctx, cr)).To(Succeed())
		}, longWait, interval).Should(Succeed())
	}

	waitReady := func() {
		Eventually(func(g Gomega) {
			cr := &keycloakAlpha.ClusterKeycloakRealm{}
			g.Expect(k8sClient.Get(ctx, types.NamespacedName{Name: crName}, cr)).To(Succeed())
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

	It("Should create ClusterKeycloakRealm with locales and event settings", func() {
		cr := &keycloakAlpha.ClusterKeycloakRealm{
			ObjectMeta: metav1.ObjectMeta{Name: crName},
			Spec: keycloakAlpha.ClusterKeycloakRealmSpec{
				ClusterKeycloakRef: ClusterKeycloakCR,
				RealmName:          realmName,
				DisplayName:        ptr.To("Idem Cluster Realm"),
				RealmEventConfig: &common.RealmEventConfig{
					AdminEventsEnabled:        ptr.To(true),
					AdminEventsDetailsEnabled: ptr.To(true),
					EnabledEventTypes:         []string{"LOGIN", "LOGOUT"},
					EventsListeners:           []string{"jboss-logging"},
				},
				Localization: &keycloakAlpha.RealmLocalization{
					InternationalizationEnabled: ptr.To(true),
					SupportedLocales:            []string{"en", "de", "uk"},
				},
			},
		}
		Expect(k8sClient.Create(ctx, cr)).To(Succeed())
		waitReady()

		created := kcRealm()
		Expect(ptr.Deref(created.DisplayName, "")).To(Equal("Idem Cluster Realm"))
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
				g.Expect(ptr.Deref(kcRealm().DisplayName, "")).To(Equal("Idem Cluster Realm"))
			}, longWait, interval).Should(Succeed())
		})).ShouldNot(BeEmpty(), "admin event log recorded nothing for a known write")
	})

	It("Should remove a locale from Keycloak when it is removed from spec", func() {
		updateSpec(func(cr *keycloakAlpha.ClusterKeycloakRealm) {
			cr.Spec.Localization.SupportedLocales = []string{"en", "de"}
		})
		waitReady()

		Eventually(func(g Gomega) {
			g.Expect(*kcRealm().SupportedLocales).To(ConsistOf("en", "de"))
		}, longWait, interval).Should(Succeed())
	})

	It("Should not write to Keycloak after the edit settles", func() {
		Expect(recorder.WritesDuring(ctx, settle, nudge)).To(BeEmpty())
	})

	It("Should delete ClusterKeycloakRealm and remove the realm from Keycloak", func() {
		cr := &keycloakAlpha.ClusterKeycloakRealm{}
		Expect(k8sClient.Get(ctx, types.NamespacedName{Name: crName}, cr)).To(Succeed())
		Expect(k8sClient.Delete(ctx, cr)).To(Succeed())

		Eventually(func() bool {
			return k8sErrors.IsNotFound(
				k8sClient.Get(ctx, types.NamespacedName{Name: crName}, &keycloakAlpha.ClusterKeycloakRealm{}))
		}, longWait, interval).Should(BeTrue())

		Eventually(func(g Gomega) {
			_, _, err := keycloakApiClient.Realms.GetRealm(ctx, realmName)
			g.Expect(err).Should(HaveOccurred(), "realm must be gone from Keycloak")
		}, longWait, interval).Should(Succeed())
	})
})

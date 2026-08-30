package keycloakorganization

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	k8sErrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/utils/ptr"

	"github.com/epam/edp-keycloak-operator/api/common"
	keycloakApi "github.com/epam/edp-keycloak-operator/api/v1"
	"github.com/epam/edp-keycloak-operator/api/v1alpha1"
	"github.com/epam/edp-keycloak-operator/pkg/client/keycloakapi"
	"github.com/epam/edp-keycloak-operator/pkg/testutils"
)

// Idempotency suite for KeycloakOrganization.
var _ = Describe("KeycloakOrganization idempotent reconcile", Ordered, func() {
	const (
		crName   = "idem-org"
		orgAlias = "idem-org"
		settle   = testutils.Settle
		longWait = testutils.LongWait
	)

	var (
		recorder *testutils.AdminEventRecorder
		orgID    string
	)

	nudge := func() {
		Expect(testutils.Nudge(ctx, k8sClient,
			types.NamespacedName{Name: crName, Namespace: ns}, &v1alpha1.KeycloakOrganization{})).To(Succeed())
	}

	updateSpec := func(mutate func(*v1alpha1.KeycloakOrganization)) {
		Eventually(func(g Gomega) {
			cr := &v1alpha1.KeycloakOrganization{}
			g.Expect(k8sClient.Get(ctx, types.NamespacedName{Name: crName, Namespace: ns}, cr)).To(Succeed())
			mutate(cr)
			g.Expect(k8sClient.Update(ctx, cr)).To(Succeed())
		}, longWait, interval).Should(Succeed())
	}

	waitReady := func() {
		Eventually(func(g Gomega) {
			cr := &v1alpha1.KeycloakOrganization{}
			g.Expect(k8sClient.Get(ctx, types.NamespacedName{Name: crName, Namespace: ns}, cr)).To(Succeed())
			g.Expect(cr.Status.Value).To(Equal(common.StatusOK))
			g.Expect(cr.Status.ObservedGeneration).To(Equal(cr.Generation))
		}, longWait, interval).Should(Succeed())
	}

	kcOrg := func() *keycloakapi.OrganizationRepresentation {
		rep, _, err := keycloakAdminClient.Organizations.GetOrganization(ctx, KeycloakRealmCR, orgID)
		Expect(err).ShouldNot(HaveOccurred())
		Expect(rep).ShouldNot(BeNil())

		return rep
	}

	domainNames := func() []string {
		rep := kcOrg()
		if rep.Domains == nil {
			return nil
		}

		names := make([]string, 0, len(*rep.Domains))
		for _, d := range *rep.Domains {
			names = append(names, ptr.Deref(d.Name, ""))
		}

		return names
	}

	BeforeAll(func() {
		recorder = testutils.NewAdminEventRecorder(keycloakAdminClient, KeycloakRealmCR)
		Expect(recorder.Enable(ctx)).To(Succeed())
	})

	It("Should create KeycloakOrganization with domains", func() {
		cr := &v1alpha1.KeycloakOrganization{
			ObjectMeta: metav1.ObjectMeta{Name: crName, Namespace: ns},
			Spec: v1alpha1.KeycloakOrganizationSpec{
				Name:        "Idem Org",
				Alias:       orgAlias,
				Description: "idem org v1",
				Domains:     []string{"idem-a.example.test", "idem-b.example.test"},
				RealmRef:    common.RealmRef{Kind: keycloakApi.KeycloakRealmKind, Name: KeycloakRealmCR},
			},
		}
		Expect(k8sClient.Create(ctx, cr)).To(Succeed())
		waitReady()

		stored := &v1alpha1.KeycloakOrganization{}
		Expect(k8sClient.Get(ctx, types.NamespacedName{Name: crName, Namespace: ns}, stored)).To(Succeed())
		orgID = stored.Status.OrganizationID
		Expect(orgID).ShouldNot(BeEmpty())

		// Sibling specs share the realm; report only this organization's events.
		recorder = recorder.Scoped("organizations/" + orgID)

		Expect(domainNames()).To(ConsistOf("idem-a.example.test", "idem-b.example.test"))
	})

	It("Should not write to Keycloak on a forced reconcile and keep the organization ID", func() {
		Expect(recorder.WritesDuring(ctx, settle, nudge)).To(BeEmpty())

		// Resolve by alias, not by id: GetOrganization echoes the id it was given, so
		// comparing that to orgID could never fail.
		byAlias, _, err := keycloakAdminClient.Organizations.GetOrganizationByAlias(ctx, KeycloakRealmCR, orgAlias)
		Expect(err).ShouldNot(HaveOccurred())
		Expect(byAlias).ShouldNot(BeNil())
		Expect(ptr.Deref(byAlias.Id, "")).To(Equal(orgID), "organization must not be recreated")
	})

	It("Should heal external drift on the organization description", func() {
		// Liveness anchor for this suite's empty-window specs. The drift write sits
		// inside the window: the controller reconciles on a timer and can heal it
		// before the nudge lands.
		Expect(recorder.WritesDuring(ctx, 0, func() {
			rep := kcOrg()
			rep.Description = ptr.To("drifted-by-hand")
			_, err := keycloakAdminClient.Organizations.UpdateOrganization(ctx, KeycloakRealmCR, orgID, *rep)
			Expect(err).ShouldNot(HaveOccurred())

			nudge()

			Eventually(func(g Gomega) {
				g.Expect(ptr.Deref(kcOrg().Description, "")).To(Equal("idem org v1"))
			}, longWait, interval).Should(Succeed())
		})).ShouldNot(BeEmpty(), "admin event log recorded nothing for a known write")
	})

	It("Should keep the existing domains when a domain is added", func() {
		updateSpec(func(cr *v1alpha1.KeycloakOrganization) {
			cr.Spec.Domains = append(cr.Spec.Domains, "idem-c.example.test")
		})
		waitReady()

		Eventually(func(g Gomega) {
			g.Expect(domainNames()).To(ConsistOf(
				"idem-a.example.test", "idem-b.example.test", "idem-c.example.test"))
		}, longWait, interval).Should(Succeed())
	})

	It("Should remove a domain from Keycloak when it is removed from spec", func() {
		updateSpec(func(cr *v1alpha1.KeycloakOrganization) {
			cr.Spec.Domains = []string{"idem-a.example.test"}
		})
		waitReady()

		Eventually(func(g Gomega) {
			g.Expect(domainNames()).To(ConsistOf("idem-a.example.test"))
		}, longWait, interval).Should(Succeed())
	})

	It("Should not write to Keycloak after the edits settle", func() {
		Expect(recorder.WritesDuring(ctx, settle, nudge)).To(BeEmpty())
	})

	It("Should force one write when observedGeneration is cleared", func() {
		// No drift: Keycloak matches spec here, so a write can only come from the
		// cleared observedGeneration.
		Expect(recorder.WritesDuring(ctx, 0, func() {
			Expect(testutils.ClearObservedGeneration(ctx, k8sClient,
				types.NamespacedName{Name: crName, Namespace: ns},
				&v1alpha1.KeycloakOrganization{})).To(Succeed())
		})).ShouldNot(BeEmpty(), "a cleared observedGeneration must force one write")
	})

	It("Should delete KeycloakOrganization and remove the organization from Keycloak", func() {
		cr := &v1alpha1.KeycloakOrganization{}
		Expect(k8sClient.Get(ctx, types.NamespacedName{Name: crName, Namespace: ns}, cr)).To(Succeed())
		Expect(k8sClient.Delete(ctx, cr)).To(Succeed())

		Eventually(func() bool {
			return k8sErrors.IsNotFound(
				k8sClient.Get(ctx, types.NamespacedName{Name: crName, Namespace: ns},
					&v1alpha1.KeycloakOrganization{}))
		}, longWait, interval).Should(BeTrue())

		Eventually(func(g Gomega) {
			_, _, err := keycloakAdminClient.Organizations.GetOrganizationByAlias(ctx, KeycloakRealmCR, orgAlias)
			g.Expect(err).Should(HaveOccurred(), "organization must be gone from Keycloak")
		}, longWait, interval).Should(Succeed())
	})
})

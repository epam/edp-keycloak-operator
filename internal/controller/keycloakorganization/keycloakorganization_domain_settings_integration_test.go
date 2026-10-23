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
	"github.com/epam/edp-keycloak-operator/pkg/testutils"
)

var _ = Describe("KeycloakOrganization domain settings", Ordered, func() {
	const (
		crName   = "domset-org"
		orgAlias = "domset-org"
		domainA  = "domset-a.example.test"
		domainB  = "domset-b.example.test"
		domainC  = "domset-c.example.test"
		settle   = testutils.Settle
		longWait = testutils.LongWait
	)

	var (
		recorder *testutils.AdminEventRecorder
		orgID    string
	)

	crKey := types.NamespacedName{Name: crName, Namespace: ns}

	nudge := func() {
		Expect(testutils.Nudge(ctx, k8sClient, crKey, &v1alpha1.KeycloakOrganization{})).To(Succeed())
	}

	updateSpec := func(mutate func(*v1alpha1.KeycloakOrganization)) {
		Eventually(func(g Gomega) {
			cr := &v1alpha1.KeycloakOrganization{}
			g.Expect(k8sClient.Get(ctx, crKey, cr)).To(Succeed())
			mutate(cr)
			g.Expect(k8sClient.Update(ctx, cr)).To(Succeed())
		}, longWait, interval).Should(Succeed())
	}

	waitReady := func() {
		Eventually(func(g Gomega) {
			cr := &v1alpha1.KeycloakOrganization{}
			g.Expect(k8sClient.Get(ctx, crKey, cr)).To(Succeed())
			g.Expect(cr.Status.Value).To(Equal(common.StatusOK))
			g.Expect(cr.Status.ObservedGeneration).To(Equal(cr.Generation))
		}, longWait, interval).Should(Succeed())
	}

	kcDomains := func() map[string]bool {
		rep, _, err := keycloakAdminClient.Organizations.GetOrganization(ctx, KeycloakRealmCR, orgID)
		Expect(err).ShouldNot(HaveOccurred())
		Expect(rep).ShouldNot(BeNil())

		verified := map[string]bool{}
		for _, d := range ptr.Deref(rep.Domains, nil) {
			verified[ptr.Deref(d.Name, "")] = ptr.Deref(d.Verified, false)
		}

		return verified
	}

	setVerified := func(name string, verified bool) {
		rep, _, err := keycloakAdminClient.Organizations.GetOrganization(ctx, KeycloakRealmCR, orgID)
		Expect(err).ShouldNot(HaveOccurred())

		domains := ptr.Deref(rep.Domains, nil)
		for i := range domains {
			if ptr.Deref(domains[i].Name, "") == name {
				domains[i].Verified = ptr.To(verified)
			}
		}

		rep.Domains = &domains
		_, err = keycloakAdminClient.Organizations.UpdateOrganization(ctx, KeycloakRealmCR, orgID, *rep)
		Expect(err).ShouldNot(HaveOccurred())
	}

	BeforeAll(func() {
		recorder = testutils.NewAdminEventRecorder(keycloakAdminClient, KeycloakRealmCR)
		Expect(recorder.Enable(ctx)).To(Succeed())

		DeferCleanup(func() {
			cr := &v1alpha1.KeycloakOrganization{}
			if err := k8sClient.Get(ctx, crKey, cr); err == nil {
				Expect(k8sClient.Delete(ctx, cr)).To(Succeed())
			}

			Eventually(func() bool {
				return k8sErrors.IsNotFound(k8sClient.Get(ctx, crKey, &v1alpha1.KeycloakOrganization{}))
			}, longWait, interval).Should(BeTrue())
		})

		cr := &v1alpha1.KeycloakOrganization{
			ObjectMeta: metav1.ObjectMeta{Name: crName, Namespace: ns},
			Spec: v1alpha1.KeycloakOrganizationSpec{
				Name:        "Domain Settings Org",
				Alias:       orgAlias,
				Description: "domset v1",
				DomainsV2:   orgDomains(domainA, domainB),
				RealmRef:    common.RealmRef{Kind: keycloakApi.KeycloakRealmKind, Name: KeycloakRealmCR},
			},
		}
		Expect(k8sClient.Create(ctx, cr)).To(Succeed())
		waitReady()

		stored := &v1alpha1.KeycloakOrganization{}
		Expect(k8sClient.Get(ctx, crKey, stored)).To(Succeed())
		orgID = stored.Status.OrganizationID
		Expect(orgID).ShouldNot(BeEmpty())

		// Sibling suites share the realm; report only this organization's events.
		recorder = recorder.Scoped("organizations/" + orgID)
	})

	It("Should keep a verified domain when the spec is edited", func() {
		setVerified(domainA, true)

		updateSpec(func(cr *v1alpha1.KeycloakOrganization) { cr.Spec.Description = "domset v2" })
		waitReady()

		Expect(kcDomains()).To(HaveKeyWithValue(domainA, true))
	})

	It("Should send an added domain by name only and keep the others", func() {
		updateSpec(func(cr *v1alpha1.KeycloakOrganization) {
			cr.Spec.DomainsV2 = orgDomains(domainA, domainB, domainC)
		})
		waitReady()

		Expect(kcDomains()).To(Equal(map[string]bool{domainA: true, domainB: false, domainC: false}))
	})

	It("Should not repair settings changed out of band", func() {
		// Liveness anchor for the empty-window specs of this suite.
		Expect(recorder.WritesDuring(ctx, 0, func() { setVerified(domainA, false) })).
			ShouldNot(BeEmpty(), "admin event log recorded nothing for a known write")

		Expect(recorder.WritesDuring(ctx, settle, nudge)).To(BeEmpty())
		Expect(kcDomains()).To(HaveKeyWithValue(domainA, false))
	})

	It("Should not write on a mixed-case spec domain and keep its settings", func() {
		setVerified(domainA, true)

		updateSpec(func(cr *v1alpha1.KeycloakOrganization) {
			cr.Spec.DomainsV2 = orgDomains("DomSet-A.Example.test", domainB, domainC)
		})
		waitReady()

		Expect(kcDomains()).To(HaveKeyWithValue(domainA, true))
		Expect(recorder.WritesDuring(ctx, settle, nudge)).To(BeEmpty())
	})
})

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

var _ = Describe("KeycloakOrganization link settings", Ordered, func() {
	const (
		crName   = "linkset-org"
		orgAlias = "linkset-org"
		idpAlias = "linkset-idp"
		longWait = testutils.LongWait
	)

	var (
		orgID string
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

	kcLink := func() keycloakapi.OrganizationIdentityProviderLinkRepresentation {
		idps, _, err := keycloakAdminClient.Organizations.GetOrganizationIdentityProviders(ctx, KeycloakRealmCR, orgID)
		Expect(err).ShouldNot(HaveOccurred())
		Expect(idps).To(HaveLen(1))
		Expect(idps[0].OrganizationLinks).ShouldNot(BeNil())
		Expect(*idps[0].OrganizationLinks).To(HaveLen(1))

		return (*idps[0].OrganizationLinks)[0]
	}

	BeforeAll(func() {
		info, err := keycloakAdminClient.GetServerInfo(ctx)
		Expect(err).ShouldNot(HaveOccurred())

		supported, err := testutils.ServerVersionAtLeast(info, 26, 8)
		Expect(err).ShouldNot(HaveOccurred())

		if !supported {
			Skip("organization link settings need Keycloak >= 26.8, server " + info.SystemInfo.ServerVersion)
		}

		_, err = keycloakAdminClient.IdentityProviders.CreateIdentityProvider(ctx, KeycloakRealmCR,
			keycloakapi.IdentityProviderRepresentation{
				Alias:      ptr.To(idpAlias),
				ProviderId: ptr.To("github"),
				Enabled:    ptr.To(true),
				Config: &map[string]string{
					"clientId":     "linkset-client-id",
					"clientSecret": "linkset-client-secret",
				},
			})
		if !keycloakapi.IsConflict(err) {
			Expect(err).ShouldNot(HaveOccurred())
		}
	})

	It("Should reject an unknown membershipType", func() {
		cr := &v1alpha1.KeycloakOrganization{
			ObjectMeta: metav1.ObjectMeta{Name: "linkset-invalid", Namespace: ns},
			Spec: v1alpha1.KeycloakOrganizationSpec{
				Name:     "Invalid",
				Alias:    "linkset-invalid",
				Domains:  []string{"linkset-invalid.example.test"},
				RealmRef: common.RealmRef{Kind: keycloakApi.KeycloakRealmKind, Name: KeycloakRealmCR},
				IdentityProviders: []v1alpha1.OrgIdentityProvider{
					{Alias: idpAlias, MembershipType: ptr.To("SEMI")},
				},
			},
		}

		err := k8sClient.Create(ctx, cr)
		Expect(err).Should(HaveOccurred())
		Expect(k8sErrors.IsInvalid(err)).To(BeTrue())
	})

	It("Should link the identity provider with autoMembership false in one pass", func() {
		cr := &v1alpha1.KeycloakOrganization{
			ObjectMeta: metav1.ObjectMeta{Name: crName, Namespace: ns},
			Spec: v1alpha1.KeycloakOrganizationSpec{
				Name:     "Link Settings Org",
				Alias:    orgAlias,
				Domains:  []string{"linkset.example.test"},
				RealmRef: common.RealmRef{Kind: keycloakApi.KeycloakRealmKind, Name: KeycloakRealmCR},
				IdentityProviders: []v1alpha1.OrgIdentityProvider{
					{Alias: idpAlias, AutoMembership: ptr.To(false)},
				},
			},
		}
		Expect(k8sClient.Create(ctx, cr)).To(Succeed())
		waitReady()

		stored := &v1alpha1.KeycloakOrganization{}
		Expect(k8sClient.Get(ctx, crKey, stored)).To(Succeed())
		orgID = stored.Status.OrganizationID
		Expect(orgID).ShouldNot(BeEmpty())

		link := kcLink()
		Expect(ptr.Deref(link.AutoMembership, true)).To(BeFalse())
		Expect(ptr.Deref(link.MembershipType, "")).To(Equal("UNMANAGED"))
	})

	It("Should repair link drift", func() {
		_, err := keycloakAdminClient.Organizations.UpdateOrganizationIdentityProviderLink(
			ctx, KeycloakRealmCR, orgID, idpAlias,
			keycloakapi.OrganizationIdentityProviderLinkRepresentation{
				AutoMembership: ptr.To(true),
				MembershipType: ptr.To("UNMANAGED"),
			})
		Expect(err).ShouldNot(HaveOccurred())

		nudge()

		Eventually(func(g Gomega) {
			g.Expect(ptr.Deref(kcLink().AutoMembership, true)).To(BeFalse())
		}, longWait, interval).Should(Succeed())
	})

	It("Should keep the Keycloak link when the settings leave the spec", func() {
		updateSpec(func(cr *v1alpha1.KeycloakOrganization) {
			cr.Spec.IdentityProviders = []v1alpha1.OrgIdentityProvider{{Alias: idpAlias}}
		})
		waitReady()

		Expect(ptr.Deref(kcLink().AutoMembership, true)).To(BeFalse())
	})

	It("Should delete KeycloakOrganization", func() {
		cr := &v1alpha1.KeycloakOrganization{}
		Expect(k8sClient.Get(ctx, crKey, cr)).To(Succeed())
		Expect(k8sClient.Delete(ctx, cr)).To(Succeed())

		Eventually(func() bool {
			return k8sErrors.IsNotFound(k8sClient.Get(ctx, crKey, &v1alpha1.KeycloakOrganization{}))
		}, longWait, interval).Should(BeTrue())
	})
})

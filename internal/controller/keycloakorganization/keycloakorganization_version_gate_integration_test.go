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

var _ = Describe("KeycloakOrganization Keycloak 26.8 field gate", Ordered, func() {
	const (
		longWait = testutils.LongWait
		settle   = testutils.Settle
	)

	realmRef := common.RealmRef{Kind: keycloakApi.KeycloakRealmKind, Name: KeycloakRealmCR}

	var (
		serverVersion string
		recorder      *testutils.AdminEventRecorder
	)

	getOrg := func(key types.NamespacedName) *v1alpha1.KeycloakOrganization {
		cr := &v1alpha1.KeycloakOrganization{}
		Expect(k8sClient.Get(ctx, key, cr)).To(Succeed())

		return cr
	}

	remove := func(key types.NamespacedName) {
		cr := &v1alpha1.KeycloakOrganization{}
		if err := k8sClient.Get(ctx, key, cr); err == nil {
			Expect(k8sClient.Delete(ctx, cr)).To(Succeed())
		}

		Eventually(func() bool {
			return k8sErrors.IsNotFound(k8sClient.Get(ctx, key, &v1alpha1.KeycloakOrganization{}))
		}, longWait, interval).Should(BeTrue())
	}

	BeforeAll(func() {
		info, err := keycloakAdminClient.GetServerInfo(ctx)
		Expect(err).ShouldNot(HaveOccurred())

		supported, err := keycloakapi.ServerVersionAtLeast(info, 26, 8)
		Expect(err).ShouldNot(HaveOccurred())

		if supported {
			Skip("the gate covers Keycloak older than 26.8, server " + info.SystemInfo.ServerVersion)
		}

		serverVersion = info.SystemInfo.ServerVersion

		recorder = testutils.NewAdminEventRecorder(keycloakAdminClient, KeycloakRealmCR)
		Expect(recorder.Enable(ctx)).To(Succeed())
	})

	It("Should not create an organization that sets a 26.8-only field", func() {
		name := testutils.UniqueName("vgate-new")
		idpAlias := name + "-idp"
		key := types.NamespacedName{Name: name, Namespace: ns}

		DeferCleanup(func() { remove(key) })

		Expect(recorder.Scoped("organizations").WritesDuring(ctx, settle, func() {
			cr := &v1alpha1.KeycloakOrganization{
				ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: ns},
				Spec: v1alpha1.KeycloakOrganizationSpec{
					Name: name, Alias: name, RealmRef: realmRef,
					DomainsV2:         []v1alpha1.OrgDomain{{Name: name + ".example.test", IdentityProviderAlias: ptr.To(idpAlias)}},
					IdentityProviders: ptr.To([]v1alpha1.OrgIdentityProvider{{Alias: idpAlias}}),
				},
			}
			Expect(k8sClient.Create(ctx, cr)).To(Succeed())

			Eventually(func(g Gomega) {
				g.Expect(getOrg(key).Status.Value).To(And(
					ContainSubstring("identityProviderAlias requires Keycloak 26.8 or later"),
					ContainSubstring("server version is "+serverVersion),
				))
			}, longWait, interval).Should(Succeed())

			Expect(testutils.Nudge(ctx, k8sClient, key, &v1alpha1.KeycloakOrganization{})).To(Succeed())
			Eventually(func(g Gomega) {
				g.Expect(getOrg(key).Status.Value).To(ContainSubstring("requires Keycloak 26.8 or later"))
			}, longWait, interval).Should(Succeed())
		})).To(BeEmpty(), "the gate must fail before any Keycloak write")

		Expect(getOrg(key).Status.OrganizationID).To(BeEmpty())

		_, _, err := keycloakAdminClient.Organizations.GetOrganizationByAlias(ctx, KeycloakRealmCR, name)
		Expect(keycloakapi.IsNotFound(err)).To(BeTrue(), "the organization must not exist")
	})

	It("Should leave an existing organization untouched when a 26.8-only field is added", func() {
		name := testutils.UniqueName("vgate-edit")
		idpAlias := name + "-idp"
		key := types.NamespacedName{Name: name, Namespace: ns}

		DeferCleanup(func() { remove(key) })

		cr := &v1alpha1.KeycloakOrganization{
			ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: ns},
			Spec: v1alpha1.KeycloakOrganizationSpec{
				Name: name, Alias: name, RealmRef: realmRef,
				DomainsV2: orgDomains(name + ".example.test"),
			},
		}
		Expect(k8sClient.Create(ctx, cr)).To(Succeed())

		Eventually(func(g Gomega) {
			stored := getOrg(key)
			g.Expect(stored.Status.Value).To(Equal(common.StatusOK))
			g.Expect(stored.Status.ObservedGeneration).To(Equal(stored.Generation))
		}, longWait, interval).Should(Succeed())

		orgID := getOrg(key).Status.OrganizationID
		Expect(orgID).ShouldNot(BeEmpty())

		var observedBefore int64

		Expect(recorder.Scoped("organizations/"+orgID).WritesDuring(ctx, settle, func() {
			observedBefore = getOrg(key).Status.ObservedGeneration

			Eventually(func(g Gomega) {
				stored := getOrg(key)
				stored.Spec.IdentityProviders = ptr.To([]v1alpha1.OrgIdentityProvider{{Alias: idpAlias}})
				stored.Spec.DomainsV2[0].IdentityProviderAlias = ptr.To(idpAlias)
				stored.Spec.DomainsV2[0].AutoRedirect = ptr.To(true)
				g.Expect(k8sClient.Update(ctx, stored)).To(Succeed())
			}, longWait, interval).Should(Succeed())

			Eventually(func(g Gomega) {
				g.Expect(getOrg(key).Status.Value).To(And(
					ContainSubstring("domainsV2[0]"),
					ContainSubstring("identityProviderAlias requires Keycloak 26.8 or later"),
					ContainSubstring("server version is "+serverVersion),
				))
			}, longWait, interval).Should(Succeed())
		})).To(BeEmpty(), "the gate must fail before any Keycloak write")

		stored := getOrg(key)
		Expect(stored.Status.ObservedGeneration).To(Equal(observedBefore))
		Expect(stored.Status.ObservedGeneration).To(BeNumerically("<", stored.Generation))
		Expect(stored.Status.OrganizationID).To(Equal(orgID))
	})
})

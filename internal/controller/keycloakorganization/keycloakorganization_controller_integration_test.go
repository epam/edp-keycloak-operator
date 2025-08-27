package keycloakorganization

import (
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/Nerzal/gocloak/v12"
	k8sErrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	"github.com/epam/edp-keycloak-operator/api/common"
	v1 "github.com/epam/edp-keycloak-operator/api/v1"
	"github.com/epam/edp-keycloak-operator/api/v1alpha1"
	"github.com/epam/edp-keycloak-operator/internal/controller/helper"
	"github.com/epam/edp-keycloak-operator/pkg/client/keycloak"
	"github.com/epam/edp-keycloak-operator/pkg/client/keycloak/adapter"
	"github.com/epam/edp-keycloak-operator/pkg/objectmeta"
)

var _ = Describe("Organization controller", Ordered, func() {
	const (
		orgCR = "test-keycloak-organization"
	)

	var (
		keycloakClient keycloak.Client
		h              helper.ControllerHelper
	)

	BeforeAll(func() {
		h = helper.MakeHelper(k8sClient, k8sClient.Scheme(), "default")
	})

	It("Should create Organization", func() {
		By("Creating an Identity Provider for the organization")
		_, err := keycloakApiClient.CreateIdentityProvider(
			ctx,
			getKeyCloakToken(),
			KeycloakRealmCR,
			gocloak.IdentityProviderRepresentation{
				Alias:       gocloak.StringP("test-org-idp"),
				DisplayName: gocloak.StringP("Test Organization Identity Provider"),
				ProviderID:  gocloak.StringP("github"),
				Enabled:     gocloak.BoolP(true),
				Config: &map[string]string{
					"clientId":     "test-org-client-id",
					"clientSecret": "test-org-client-secret",
				},
			},
		)
		Expect(adapter.SkipAlreadyExistsErr(err)).ShouldNot(HaveOccurred())

		By("Creating an Organization")
		org := &v1alpha1.KeycloakOrganization{
			ObjectMeta: metav1.ObjectMeta{
				Name:      orgCR,
				Namespace: ns,
			},
			Spec: v1alpha1.KeycloakOrganizationSpec{
				RealmRef: common.RealmRef{
					Kind: v1.KeycloakRealmKind,
					Name: KeycloakRealmCR,
				},
				Name:        "Test Organization",
				Alias:       "test-org",
				Domains:     []string{"example.com", "test.com"},
				RedirectURL: "https://example.com/redirect",
				Description: "Test organization for integration tests",
				Attributes: map[string][]string{
					"department": {"engineering", "qa"},
					"location":   {"us-east"},
				},
				IdentityProviders: []v1alpha1.OrgIdentityProvider{
					{
						Alias: "test-org-idp",
					},
				},
			},
		}
		Expect(k8sClient.Create(ctx, org)).Should(Succeed())

		Eventually(func(g Gomega) {
			createdOrg := &v1alpha1.KeycloakOrganization{}
			err = k8sClient.Get(ctx, types.NamespacedName{Name: orgCR, Namespace: ns}, createdOrg)
			g.Expect(err).ShouldNot(HaveOccurred())
			g.Expect(createdOrg.Status.Value).Should(Equal(common.StatusOK))
			g.Expect(createdOrg.Status.OrganizationID).ShouldNot(BeEmpty())

			keycloakClient, err = h.CreateKeycloakClientFromRealmRef(ctx, createdOrg)
			g.Expect(err).ShouldNot(HaveOccurred())
		}).WithTimeout(time.Second * 30).WithPolling(time.Second).Should(Succeed())

		By("Verifying the organization was created in Keycloak")
		Eventually(func(g Gomega) {
			testOrg, err := keycloakClient.GetOrganizationByAlias(ctx, KeycloakRealmCR, "test-org")
			g.Expect(err).ShouldNot(HaveOccurred())

			g.Expect(testOrg).ShouldNot(BeNil())
			g.Expect(testOrg.Name).Should(Equal("Test Organization"))
			g.Expect(testOrg.Alias).Should(Equal("test-org"))
			g.Expect(testOrg.Domains).Should(HaveLen(2))
			domainNames := make([]string, len(testOrg.Domains))
			for i, domain := range testOrg.Domains {
				domainNames[i] = domain.Name
			}
			g.Expect(domainNames).Should(ContainElements("example.com", "test.com"))
			g.Expect(testOrg.RedirectURL).Should(Equal("https://example.com/redirect"))
			g.Expect(testOrg.Description).Should(Equal("Test organization for integration tests"))
		}, time.Second*10, time.Second).Should(Succeed())
	})

	It("Should update Organization", func() {
		By("Getting Organization")
		org := &v1alpha1.KeycloakOrganization{}
		Expect(k8sClient.Get(ctx, types.NamespacedName{Namespace: ns, Name: orgCR}, org)).Should(Succeed())

		By("Updating Organization")
		org.Spec.Name = "Updated Test Organization"
		org.Spec.Description = "Updated test organization description"
		org.Spec.Domains = []string{"example.com", "updated.com"}
		org.Spec.RedirectURL = "https://updated.com/redirect"
		org.Spec.Attributes = map[string][]string{
			"department": {"engineering"},
			"region":     {"us-west"},
		}

		Expect(k8sClient.Update(ctx, org)).Should(Succeed())
		Eventually(func(g Gomega) {
			updatedOrg := &v1alpha1.KeycloakOrganization{}
			err := k8sClient.Get(ctx, types.NamespacedName{Name: org.Name, Namespace: ns}, updatedOrg)
			g.Expect(err).ShouldNot(HaveOccurred())
			g.Expect(updatedOrg.Status.Value).Should(Equal(common.StatusOK))
		}, time.Second*10, time.Second).Should(Succeed())

		By("Verifying the organization was updated in Keycloak")
		Eventually(func(g Gomega) {
			testOrg, err := keycloakClient.GetOrganizationByAlias(ctx, KeycloakRealmCR, "test-org")
			g.Expect(err).ShouldNot(HaveOccurred())

			g.Expect(testOrg).ShouldNot(BeNil())
			g.Expect(testOrg.Name).Should(Equal("Updated Test Organization"))
			g.Expect(testOrg.Description).Should(Equal("Updated test organization description"))
			g.Expect(testOrg.Domains).Should(HaveLen(2))
			domainNames := make([]string, len(testOrg.Domains))
			for i, domain := range testOrg.Domains {
				domainNames[i] = domain.Name
			}
			g.Expect(domainNames).Should(ContainElements("example.com", "updated.com"))
			g.Expect(testOrg.RedirectURL).Should(Equal("https://updated.com/redirect"))
		}, time.Second*10, time.Second).Should(Succeed())
	})

	It("Should delete Organization", func() {
		By("Getting Organization")
		org := &v1alpha1.KeycloakOrganization{}
		Expect(k8sClient.Get(ctx, types.NamespacedName{Namespace: ns, Name: orgCR}, org)).Should(Succeed())

		By("Deleting Organization")
		Expect(k8sClient.Delete(ctx, org)).Should(Succeed())
		Eventually(func(g Gomega) {
			deletedOrg := &v1alpha1.KeycloakOrganization{}
			err := k8sClient.Get(ctx, types.NamespacedName{Name: org.Name, Namespace: ns}, deletedOrg)
			g.Expect(k8sErrors.IsNotFound(err)).Should(BeTrue())
		}, timeout, interval).Should(Succeed())

		By("Verifying the organization was deleted from Keycloak")
		Eventually(func(g Gomega) {
			testOrg, err := keycloakClient.GetOrganizationByAlias(ctx, KeycloakRealmCR, "test-org")
			g.Expect(err).Should(HaveOccurred())
			g.Expect(testOrg).Should(BeNil())
		}, time.Second*10, time.Second).Should(Succeed())
	})

	It("Should preserve organization with annotation", func() {
		By("Creating an Organization with preserve annotation")
		org := &v1alpha1.KeycloakOrganization{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-preserve-organization",
				Namespace: ns,
				Annotations: map[string]string{
					objectmeta.PreserveResourcesOnDeletionAnnotation: "true",
				},
			},
			Spec: v1alpha1.KeycloakOrganizationSpec{
				RealmRef: common.RealmRef{
					Kind: "KeycloakRealm",
					Name: KeycloakRealmCR,
				},
				Name:    "Preserve Test Organization",
				Alias:   "preserve-test-org",
				Domains: []string{"preserve.com"},
			},
		}
		Expect(k8sClient.Create(ctx, org)).Should(Succeed())

		Eventually(func(g Gomega) {
			createdOrg := &v1alpha1.KeycloakOrganization{}
			err := k8sClient.Get(ctx, types.NamespacedName{Name: org.Name, Namespace: ns}, createdOrg)
			g.Expect(err).ShouldNot(HaveOccurred())
			g.Expect(createdOrg.Status.Value).Should(Equal(common.StatusOK))
		}).WithTimeout(time.Second * 30).WithPolling(time.Second).Should(Succeed())

		By("Deleting Organization with preserve annotation")
		Expect(k8sClient.Delete(ctx, org)).Should(Succeed())

		By("Verifying the organization still exists in Keycloak")
		Eventually(func(g Gomega) {
			preservedOrg, err := keycloakClient.GetOrganizationByAlias(ctx, KeycloakRealmCR, "preserve-test-org")
			g.Expect(err).ShouldNot(HaveOccurred())

			g.Expect(preservedOrg).ShouldNot(BeNil())
			g.Expect(preservedOrg.Name).Should(Equal("Preserve Test Organization"))
		}, time.Second*10, time.Second).Should(Succeed())
	})

	It("Should fail to create Organization with invalid realm reference", func() {
		By("Creating an Organization with invalid realm reference")
		org := &v1alpha1.KeycloakOrganization{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-invalid-realm-org",
				Namespace: ns,
			},
			Spec: v1alpha1.KeycloakOrganizationSpec{
				RealmRef: common.RealmRef{
					Kind: "KeycloakRealm",
					Name: "invalid-realm",
				},
				Name:    "Invalid Realm Organization",
				Alias:   "invalid-realm-org",
				Domains: []string{"invalid.com"},
			},
		}
		Expect(k8sClient.Create(ctx, org)).Should(Succeed())

		By("Waiting for Organization to be processed")
		time.Sleep(time.Second * 2)

		By("Checking Organization status")
		Consistently(func(g Gomega) {
			createdOrg := &v1alpha1.KeycloakOrganization{}
			err := k8sClient.Get(ctx, types.NamespacedName{Name: org.Name, Namespace: ns}, createdOrg)
			g.Expect(err).ShouldNot(HaveOccurred())
			g.Expect(createdOrg.Status.Value).ShouldNot(Equal(common.StatusOK))
		}, time.Second*10, time.Second).Should(Succeed())
	})

	It("Should fail to create Organization with duplicate domain", func() {
		By("Creating first organization with domain")
		org1 := &v1alpha1.KeycloakOrganization{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-duplicate-domain-org-1",
				Namespace: ns,
			},
			Spec: v1alpha1.KeycloakOrganizationSpec{
				RealmRef: common.RealmRef{
					Kind: v1.KeycloakRealmKind,
					Name: KeycloakRealmCR,
				},
				Name:    "Duplicate Domain Test Organization 1",
				Alias:   "duplicate-domain-test-org-1",
				Domains: []string{"duplicate-domain.com"},
			},
		}
		Expect(k8sClient.Create(ctx, org1)).Should(Succeed())

		Eventually(func(g Gomega) {
			createdOrg := &v1alpha1.KeycloakOrganization{}
			err := k8sClient.Get(ctx, types.NamespacedName{Name: org1.Name, Namespace: ns}, createdOrg)
			g.Expect(err).ShouldNot(HaveOccurred())
			g.Expect(createdOrg.Status.Value).Should(Equal(common.StatusOK))
		}).WithTimeout(time.Second * 10).WithPolling(time.Second).Should(Succeed())

		By("Creating second organization with same domain")
		org2 := &v1alpha1.KeycloakOrganization{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-duplicate-domain-org-2",
				Namespace: ns,
			},
			Spec: v1alpha1.KeycloakOrganizationSpec{
				RealmRef: common.RealmRef{
					Kind: v1.KeycloakRealmKind,
					Name: KeycloakRealmCR,
				},
				Name:    "Duplicate Domain Test Organization 2",
				Alias:   "duplicate-domain-test-org-2",
				Domains: []string{"duplicate-domain.com", "unique-domain.com"},
			},
		}
		Expect(k8sClient.Create(ctx, org2)).Should(Succeed())

		By("Waiting for second Organization to be processed")
		time.Sleep(time.Second * 5)

		By("Checking second Organization status shows duplicate domain error")
		Consistently(func(g Gomega) {
			createdOrg := &v1alpha1.KeycloakOrganization{}
			err := k8sClient.Get(ctx, types.NamespacedName{Name: org2.Name, Namespace: ns}, createdOrg)
			g.Expect(err).ShouldNot(HaveOccurred())
			g.Expect(createdOrg.Status.Value).Should(ContainSubstring("Domain duplicate-domain.com is already linked to another organization"))
		}, time.Second*3, time.Second).Should(Succeed())

		By("Cleaning up first organization")
		Expect(k8sClient.Delete(ctx, org1)).Should(Succeed())
	})
})

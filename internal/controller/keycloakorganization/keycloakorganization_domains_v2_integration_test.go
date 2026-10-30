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

var _ = Describe("KeycloakOrganization domainsV2", Ordered, func() {
	const (
		longWait = testutils.LongWait
		settle   = testutils.Settle
	)

	realmRef := common.RealmRef{Kind: keycloakApi.KeycloakRealmKind, Name: KeycloakRealmCR}

	waitReady := func(key types.NamespacedName) {
		Eventually(func(g Gomega) {
			cr := &v1alpha1.KeycloakOrganization{}
			g.Expect(k8sClient.Get(ctx, key, cr)).To(Succeed())
			g.Expect(cr.Status.Value).To(Equal(common.StatusOK))
			g.Expect(cr.Status.ObservedGeneration).To(Equal(cr.Generation))
		}, longWait, interval).Should(Succeed())
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

	kcDomain := func(orgID, name string) keycloakapi.OrganizationDomainRepresentation {
		rep, _, err := keycloakAdminClient.Organizations.GetOrganization(ctx, KeycloakRealmCR, orgID)
		Expect(err).ShouldNot(HaveOccurred())

		for _, d := range ptr.Deref(rep.Domains, nil) {
			if ptr.Deref(d.Name, "") == name {
				return d
			}
		}

		Fail("domain " + name + " not found in Keycloak")

		return keycloakapi.OrganizationDomainRepresentation{}
	}

	It("Should reject a spec with neither domains nor domainsV2", func() {
		cr := &v1alpha1.KeycloakOrganization{
			ObjectMeta: metav1.ObjectMeta{Name: "domv2-none", Namespace: ns},
			Spec: v1alpha1.KeycloakOrganizationSpec{
				Name: "None", Alias: "domv2-none", RealmRef: realmRef,
			},
		}

		err := k8sClient.Create(ctx, cr)
		Expect(err).Should(HaveOccurred())
		Expect(k8sErrors.IsInvalid(err)).To(BeTrue())
		Expect(err.Error()).To(ContainSubstring("one of domains or domainsV2 must be set"))
	})

	Context("legacy domains", Ordered, func() {
		const (
			crName = "domv2-legacy"
			domain = "domv2-legacy.example.test"
		)

		key := types.NamespacedName{Name: crName, Namespace: ns}

		var (
			recorder *testutils.AdminEventRecorder
			orgID    string
		)

		BeforeAll(func() {
			DeferCleanup(func() { remove(key) })

			recorder = testutils.NewAdminEventRecorder(keycloakAdminClient, KeycloakRealmCR)
			Expect(recorder.Enable(ctx)).To(Succeed())

			_, err := keycloakAdminClient.Organizations.CreateOrganization(ctx, KeycloakRealmCR,
				keycloakapi.OrganizationRepresentation{
					Name:  ptr.To("Legacy Org"),
					Alias: ptr.To(crName),
					Domains: &[]keycloakapi.OrganizationDomainRepresentation{
						{Name: ptr.To(domain), Verified: ptr.To(true)},
					},
				})
			Expect(err).ShouldNot(HaveOccurred())

			org, _, err := keycloakAdminClient.Organizations.GetOrganizationByAlias(ctx, KeycloakRealmCR, crName)
			Expect(err).ShouldNot(HaveOccurred())

			orgID = ptr.Deref(org.Id, "")
			recorder = recorder.Scoped("organizations/" + orgID)
		})

		nudge := func() {
			Expect(testutils.Nudge(ctx, k8sClient, key, &v1alpha1.KeycloakOrganization{})).To(Succeed())
		}

		It("Should migrate domains to domainsV2 and keep the domain verified", func() {
			// The forced write repeats after a failed or stale observedGeneration stamp: assert at least one write, never an exact count.
			Expect(recorder.WritesDuring(ctx, 0, func() {
				cr := &v1alpha1.KeycloakOrganization{
					ObjectMeta: metav1.ObjectMeta{Name: crName, Namespace: ns},
					Spec: v1alpha1.KeycloakOrganizationSpec{
						Name:     "Legacy Org",
						Alias:    crName,
						RealmRef: realmRef,
						//nolint:staticcheck // legacy field under test
						Domains: []string{domain},
					},
				}
				Expect(k8sClient.Create(ctx, cr)).To(Succeed())
				waitReady(key)
			})).ShouldNot(BeEmpty(), "migrating domains must write the organization")

			stored := &v1alpha1.KeycloakOrganization{}
			Expect(k8sClient.Get(ctx, key, stored)).To(Succeed())
			Expect(stored.Spec.DomainsV2).To(Equal(orgDomains(domain)))
			Expect(ptr.Deref(kcDomain(orgID, domain).Verified, false)).To(BeTrue())
		})

		It("Should not write to Keycloak after settling on a migrated CR", func() {
			Expect(recorder.WritesDuring(ctx, settle, nudge)).To(BeEmpty())
		})

		It("Should ignore later edits of domains", func() {
			Eventually(func(g Gomega) {
				cr := &v1alpha1.KeycloakOrganization{}
				g.Expect(k8sClient.Get(ctx, key, cr)).To(Succeed())
				//nolint:staticcheck // legacy field under test
				cr.Spec.Domains = append(cr.Spec.Domains, "domv2-ignored.example.test")
				g.Expect(k8sClient.Update(ctx, cr)).To(Succeed())
			}, longWait, interval).Should(Succeed())
			waitReady(key)

			rep, _, err := keycloakAdminClient.Organizations.GetOrganization(ctx, KeycloakRealmCR, orgID)
			Expect(err).ShouldNot(HaveOccurred())
			Expect(ptr.Deref(rep.Domains, nil)).To(HaveLen(1))
			Expect(ptr.Deref(rep.Domains, nil)[0].Name).To(HaveValue(Equal(domain)))
		})
	})

	Context("domain routing", Ordered, func() {
		const (
			crName   = "domv2-routing"
			idpAlias = "domv2-idp"
			domain   = "domv2-routing.example.test"
		)

		key := types.NamespacedName{Name: crName, Namespace: ns}

		var (
			recorder *testutils.AdminEventRecorder
			orgID    string
		)

		nudge := func() {
			Expect(testutils.Nudge(ctx, k8sClient, key, &v1alpha1.KeycloakOrganization{})).To(Succeed())
		}

		BeforeAll(func() {
			info, err := keycloakAdminClient.GetServerInfo(ctx)
			Expect(err).ShouldNot(HaveOccurred())

			supported, err := keycloakapi.ServerVersionAtLeast(info, 26, 8)
			Expect(err).ShouldNot(HaveOccurred())

			if !supported {
				Skip("domain routing needs Keycloak >= 26.8, server " + info.SystemInfo.ServerVersion)
			}

			DeferCleanup(func() { remove(key) })

			_, err = keycloakAdminClient.IdentityProviders.CreateIdentityProvider(ctx, KeycloakRealmCR,
				keycloakapi.IdentityProviderRepresentation{
					Alias:      ptr.To(idpAlias),
					ProviderId: ptr.To("github"),
					Enabled:    ptr.To(true),
					Config: &map[string]string{
						"clientId":     "domv2-client-id",
						"clientSecret": "domv2-client-secret",
					},
				})
			if !keycloakapi.IsConflict(err) {
				Expect(err).ShouldNot(HaveOccurred())
			}

			recorder = testutils.NewAdminEventRecorder(keycloakAdminClient, KeycloakRealmCR)
			Expect(recorder.Enable(ctx)).To(Succeed())
		})

		It("Should reject an alias that is not in identityProviders", func() {
			cr := &v1alpha1.KeycloakOrganization{
				ObjectMeta: metav1.ObjectMeta{Name: crName, Namespace: ns},
				Spec: v1alpha1.KeycloakOrganizationSpec{
					Name: "Routing Org", Alias: crName, RealmRef: realmRef,
					DomainsV2: []v1alpha1.OrgDomain{{Name: domain, IdentityProviderAlias: ptr.To(idpAlias)}},
				},
			}
			Expect(k8sClient.Create(ctx, cr)).To(Succeed())

			Eventually(func(g Gomega) {
				stored := &v1alpha1.KeycloakOrganization{}
				g.Expect(k8sClient.Get(ctx, key, stored)).To(Succeed())
				g.Expect(stored.Status.Value).To(ContainSubstring("is not in spec.identityProviders"))
			}, longWait, interval).Should(Succeed())

			_, _, err := keycloakAdminClient.Organizations.GetOrganizationByAlias(ctx, KeycloakRealmCR, crName)
			Expect(keycloakapi.IsNotFound(err)).To(BeTrue(), "an invalid spec must not create the organization")
		})

		It("Should write link and domain settings in one pass", func() {
			Eventually(func(g Gomega) {
				cr := &v1alpha1.KeycloakOrganization{}
				g.Expect(k8sClient.Get(ctx, key, cr)).To(Succeed())
				cr.Spec.IdentityProviders = ptr.To([]v1alpha1.OrgIdentityProvider{{Alias: idpAlias}})
				cr.Spec.DomainsV2 = []v1alpha1.OrgDomain{{
					Name:                  domain,
					Verified:              ptr.To(true),
					IdentityProviderAlias: ptr.To(idpAlias),
					AutoRedirect:          ptr.To(true),
				}}
				g.Expect(k8sClient.Update(ctx, cr)).To(Succeed())
			}, longWait, interval).Should(Succeed())
			waitReady(key)

			stored := &v1alpha1.KeycloakOrganization{}
			Expect(k8sClient.Get(ctx, key, stored)).To(Succeed())
			orgID = stored.Status.OrganizationID
			Expect(orgID).ShouldNot(BeEmpty())

			recorder = recorder.Scoped("organizations/" + orgID)

			d := kcDomain(orgID, domain)
			Expect(ptr.Deref(d.Verified, false)).To(BeTrue())
			Expect(ptr.Deref(d.IdentityProviderAlias, "")).To(Equal(idpAlias))
			Expect(ptr.Deref(d.AutoRedirect, false)).To(BeTrue())
		})

		It("Should not write to Keycloak after settling", func() {
			Expect(recorder.WritesDuring(ctx, settle, nudge)).To(BeEmpty())
		})

		It("Should repair drift of domain settings", func() {
			Expect(recorder.WritesDuring(ctx, 0, func() {
				rep, _, err := keycloakAdminClient.Organizations.GetOrganization(ctx, KeycloakRealmCR, orgID)
				Expect(err).ShouldNot(HaveOccurred())

				domains := ptr.Deref(rep.Domains, nil)
				domains[0].AutoRedirect = ptr.To(false)
				rep.Domains = &domains

				_, err = keycloakAdminClient.Organizations.UpdateOrganization(ctx, KeycloakRealmCR, orgID, *rep)
				Expect(err).ShouldNot(HaveOccurred())

				nudge()

				Eventually(func(g Gomega) {
					g.Expect(ptr.Deref(kcDomain(orgID, domain).AutoRedirect, false)).To(BeTrue())
				}, longWait, interval).Should(Succeed())
			})).ShouldNot(BeEmpty())
		})

		It("Should clear the routing with an empty alias", func() {
			Eventually(func(g Gomega) {
				cr := &v1alpha1.KeycloakOrganization{}
				g.Expect(k8sClient.Get(ctx, key, cr)).To(Succeed())
				cr.Spec.DomainsV2[0].IdentityProviderAlias = ptr.To("")
				cr.Spec.DomainsV2[0].AutoRedirect = ptr.To(false)
				g.Expect(k8sClient.Update(ctx, cr)).To(Succeed())
			}, longWait, interval).Should(Succeed())
			waitReady(key)

			Expect(ptr.Deref(kcDomain(orgID, domain).IdentityProviderAlias, "")).To(BeEmpty())
			Expect(ptr.Deref(kcDomain(orgID, domain).AutoRedirect, false)).To(BeFalse())
		})

		It("Should not write to Keycloak after the routing is cleared", func() {
			Expect(recorder.WritesDuring(ctx, settle, nudge)).To(BeEmpty())
		})
	})

	Context("autoRedirect false before Keycloak 26.8", Ordered, func() {
		const (
			crName = "domv2-no-redirect"
			domain = "domv2-no-redirect.example.test"
		)

		key := types.NamespacedName{Name: crName, Namespace: ns}

		var recorder *testutils.AdminEventRecorder

		BeforeAll(func() {
			info, err := keycloakAdminClient.GetServerInfo(ctx)
			Expect(err).ShouldNot(HaveOccurred())

			supported, err := keycloakapi.ServerVersionAtLeast(info, 26, 8)
			Expect(err).ShouldNot(HaveOccurred())

			if supported {
				Skip("covers Keycloak older than 26.8, server " + info.SystemInfo.ServerVersion)
			}

			DeferCleanup(func() { remove(key) })

			recorder = testutils.NewAdminEventRecorder(keycloakAdminClient, KeycloakRealmCR)
			Expect(recorder.Enable(ctx)).To(Succeed())
		})

		It("Should apply verified without sending autoRedirect", func() {
			cr := &v1alpha1.KeycloakOrganization{
				ObjectMeta: metav1.ObjectMeta{Name: crName, Namespace: ns},
				Spec: v1alpha1.KeycloakOrganizationSpec{
					Name: "No Redirect Org", Alias: crName, RealmRef: realmRef,
					DomainsV2: []v1alpha1.OrgDomain{{Name: domain, Verified: ptr.To(true), AutoRedirect: ptr.To(false)}},
				},
			}
			Expect(k8sClient.Create(ctx, cr)).To(Succeed())
			waitReady(key)

			stored := &v1alpha1.KeycloakOrganization{}
			Expect(k8sClient.Get(ctx, key, stored)).To(Succeed())
			Expect(stored.Status.OrganizationID).ShouldNot(BeEmpty())

			recorder = recorder.Scoped("organizations/" + stored.Status.OrganizationID)
			Expect(ptr.Deref(kcDomain(stored.Status.OrganizationID, domain).Verified, false)).To(BeTrue())
		})

		It("Should not write to Keycloak after settling", func() {
			Expect(recorder.WritesDuring(ctx, settle, func() {
				Expect(testutils.Nudge(ctx, k8sClient, key, &v1alpha1.KeycloakOrganization{})).To(Succeed())
			})).To(BeEmpty())
		})
	})
})

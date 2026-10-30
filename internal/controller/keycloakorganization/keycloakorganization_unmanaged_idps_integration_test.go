package keycloakorganization

import (
	"strconv"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	k8sErrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/epam/edp-keycloak-operator/api/common"
	keycloakApi "github.com/epam/edp-keycloak-operator/api/v1"
	"github.com/epam/edp-keycloak-operator/api/v1alpha1"
	"github.com/epam/edp-keycloak-operator/pkg/client/keycloakapi"
	"github.com/epam/edp-keycloak-operator/pkg/testutils"
)

var _ = Describe("KeycloakOrganization identityProviders as an optional list", func() {
	const longWait = testutils.LongWait

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

	createIdP := func(alias string) {
		_, err := keycloakAdminClient.IdentityProviders.CreateIdentityProvider(ctx, KeycloakRealmCR,
			keycloakapi.IdentityProviderRepresentation{
				Alias:      ptr.To(alias),
				ProviderId: ptr.To("github"),
				Enabled:    ptr.To(true),
				Config: &map[string]string{
					"clientId":     "unmanaged-idps-client-id",
					"clientSecret": "unmanaged-idps-client-secret",
				},
			})
		Expect(err).ShouldNot(HaveOccurred())

		DeferCleanup(func() {
			_, _ = keycloakAdminClient.IdentityProviders.DeleteIdentityProvider(ctx, KeycloakRealmCR, alias)
		})
	}

	links := func(g Gomega, orgID string) []string {
		idps, _, err := keycloakAdminClient.Organizations.GetOrganizationIdentityProviders(ctx, KeycloakRealmCR, orgID)
		g.Expect(err).ShouldNot(HaveOccurred())

		aliases := make([]string, 0, len(idps))
		for _, idp := range idps {
			aliases = append(aliases, ptr.Deref(idp.Alias, ""))
		}

		return aliases
	}

	Context("typed CR", Ordered, func() {
		var (
			key     types.NamespacedName
			orgID   string
			idp1    string
			idp2    string
			editSeq int
		)

		patchSpec := func(patch client.Patch) {
			cr := &v1alpha1.KeycloakOrganization{ObjectMeta: metav1.ObjectMeta{Name: key.Name, Namespace: key.Namespace}}
			Expect(k8sClient.Patch(ctx, cr, patch)).To(Succeed())
		}

		descriptionPatch := func() client.Patch {
			editSeq++

			return client.RawPatch(types.MergePatchType,
				[]byte(`{"spec":{"description":"edit-`+strconv.Itoa(editSeq)+`"}}`))
		}

		BeforeAll(func() {
			name := testutils.UniqueName("org-unmanaged")
			key = types.NamespacedName{Name: name, Namespace: ns}
			idp1 = testutils.UniqueName("org-unmanaged-idp")
			idp2 = testutils.UniqueName("org-unmanaged-idp")

			createIdP(idp1)
			createIdP(idp2)

			cr := &v1alpha1.KeycloakOrganization{
				ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: ns},
				Spec: v1alpha1.KeycloakOrganizationSpec{
					Name: name, Alias: name, RealmRef: realmRef,
					DomainsV2: orgDomains(name + ".example.test"),
				},
			}
			Expect(k8sClient.Create(ctx, cr)).To(Succeed())
			DeferCleanup(func() { remove(key) })

			waitReady(key)
			Expect(k8sClient.Get(ctx, key, cr)).To(Succeed())
			orgID = cr.Status.OrganizationID
			Expect(orgID).ToNot(BeEmpty())
		})

		It("Should keep an identity provider linked outside the CR when identityProviders is absent", func() {
			_, err := keycloakAdminClient.Organizations.LinkIdentityProviderToOrganization(ctx, KeycloakRealmCR, orgID, idp1)
			Expect(err).ShouldNot(HaveOccurred())

			patchSpec(descriptionPatch())
			waitReady(key)

			Expect(links(Default, orgID)).To(ConsistOf(idp1))
		})

		It("Should unlink every identity provider when identityProviders is []", func() {
			patchSpec(client.RawPatch(types.MergePatchType, []byte(`{"spec":{"identityProviders":[]}}`)))

			Eventually(func(g Gomega) []string { return links(g, orgID) }, longWait, interval).Should(BeEmpty())
		})

		It("Should link exactly the listed identity provider", func() {
			_, err := keycloakAdminClient.Organizations.LinkIdentityProviderToOrganization(ctx, KeycloakRealmCR, orgID, idp2)
			Expect(err).ShouldNot(HaveOccurred())

			patchSpec(client.RawPatch(types.MergePatchType,
				[]byte(`{"spec":{"identityProviders":[{"alias":"`+idp1+`"}]}}`)))

			Eventually(func(g Gomega) []string { return links(g, orgID) }, longWait, interval).Should(ConsistOf(idp1))
		})

		It("Should keep the links when identityProviders is removed from the spec", func() {
			waitReady(key)

			patchSpec(client.RawPatch(types.MergePatchType,
				[]byte(`{"spec":{"identityProviders":null,"description":"removed"}}`)))
			waitReady(key)

			Expect(links(Default, orgID)).To(ConsistOf(idp1))
		})
	})

	Context("legacy domains with identityProviders: []", Ordered, func() {
		It("Should keep [] through the domains migration and unlink every identity provider", func() {
			name := testutils.UniqueName("org-unmanaged-legacy")
			domain := name + ".example.test"
			legacyIdP := testutils.UniqueName("org-unmanaged-idp")
			key := types.NamespacedName{Name: name, Namespace: ns}

			createIdP(legacyIdP)

			_, err := keycloakAdminClient.Organizations.CreateOrganization(ctx, KeycloakRealmCR,
				keycloakapi.OrganizationRepresentation{
					Name:  ptr.To(name),
					Alias: ptr.To(name),
					Domains: &[]keycloakapi.OrganizationDomainRepresentation{
						{Name: ptr.To(domain), Verified: ptr.To(true)},
					},
				})
			Expect(err).ShouldNot(HaveOccurred())

			DeferCleanup(func() {
				org, _, getErr := keycloakAdminClient.Organizations.GetOrganizationByAlias(ctx, KeycloakRealmCR, name)
				if getErr != nil || org == nil {
					return
				}

				_, _ = keycloakAdminClient.Organizations.DeleteOrganization(ctx, KeycloakRealmCR, ptr.Deref(org.Id, ""))
			})

			org, _, err := keycloakAdminClient.Organizations.GetOrganizationByAlias(ctx, KeycloakRealmCR, name)
			Expect(err).ShouldNot(HaveOccurred())

			orgID := ptr.Deref(org.Id, "")

			_, err = keycloakAdminClient.Organizations.LinkIdentityProviderToOrganization(ctx, KeycloakRealmCR, orgID, legacyIdP)
			Expect(err).ShouldNot(HaveOccurred())

			cr := &unstructured.Unstructured{Object: map[string]any{
				"apiVersion": v1alpha1.GroupVersion.String(),
				"kind":       "KeycloakOrganization",
				"metadata":   map[string]any{"name": name, "namespace": ns},
				"spec": map[string]any{
					"name":              name,
					"alias":             name,
					"domains":           []any{domain},
					"identityProviders": []any{},
					"realmRef":          map[string]any{"kind": keycloakApi.KeycloakRealmKind, "name": KeycloakRealmCR},
				},
			}}
			Expect(k8sClient.Create(ctx, cr)).To(Succeed())
			DeferCleanup(func() { remove(key) })

			Eventually(func(g Gomega) {
				stored := &unstructured.Unstructured{}
				stored.SetGroupVersionKind(v1alpha1.GroupVersion.WithKind("KeycloakOrganization"))
				g.Expect(k8sClient.Get(ctx, key, stored)).To(Succeed())

				domainsV2, _, err := unstructured.NestedSlice(stored.Object, "spec", "domainsV2")
				g.Expect(err).ShouldNot(HaveOccurred())
				g.Expect(domainsV2).To(HaveLen(1))

				idps, found, err := unstructured.NestedSlice(stored.Object, "spec", "identityProviders")
				g.Expect(err).ShouldNot(HaveOccurred())
				g.Expect(found).To(BeTrue())
				g.Expect(idps).To(BeEmpty())

				g.Expect(links(g, orgID)).To(BeEmpty())
			}, longWait, interval).Should(Succeed())
		})
	})
})

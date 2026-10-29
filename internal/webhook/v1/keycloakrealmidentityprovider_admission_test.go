package v1

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	k8sErrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/epam/edp-keycloak-operator/api/common"
	keycloakApi "github.com/epam/edp-keycloak-operator/api/v1"
	"github.com/epam/edp-keycloak-operator/pkg/testutils"
)

var _ = Describe("KeycloakRealmIdentityProvider mappers admission", func() {
	newIDP := func(name string, mapper keycloakApi.IdentityProviderMapper) *keycloakApi.KeycloakRealmIdentityProvider {
		return &keycloakApi.KeycloakRealmIdentityProvider{
			ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: "ns1"},
			Spec: keycloakApi.KeycloakRealmIdentityProviderSpec{
				Alias:      name,
				ProviderID: "oidc",
				Config:     map[string]string{"clientId": "test"},
				RealmRef:   common.RealmRef{Kind: keycloakApi.KeycloakRealmKind, Name: "test-realm"},
				Mappers:    []keycloakApi.IdentityProviderMapper{mapper},
			},
		}
	}

	validMapper := func() keycloakApi.IdentityProviderMapper {
		return keycloakApi.IdentityProviderMapper{
			Name:                   "m1",
			IdentityProviderMapper: "hardcoded-attribute-idp-mapper",
		}
	}

	toWire := func(idp *keycloakApi.KeycloakRealmIdentityProvider, mutate func(item map[string]any)) *unstructured.Unstructured {
		GinkgoHelper()

		obj, err := runtime.DefaultUnstructuredConverter.ToUnstructured(idp)
		Expect(err).NotTo(HaveOccurred())

		u := &unstructured.Unstructured{Object: obj}
		u.SetGroupVersionKind(keycloakApi.GroupVersion.WithKind("KeycloakRealmIdentityProvider"))

		mappers, found, err := unstructured.NestedSlice(u.Object, "spec", "mappers")
		Expect(err).NotTo(HaveOccurred())
		Expect(found).To(BeTrue())

		item, ok := mappers[0].(map[string]any)
		Expect(ok).To(BeTrue())

		mutate(item)

		Expect(unstructured.SetNestedSlice(u.Object, mappers, "spec", "mappers")).To(Succeed())

		return u
	}

	expectInvalid := func(err error, substrings ...string) {
		GinkgoHelper()
		Expect(k8sErrors.IsInvalid(err)).To(BeTrue(), "got: %v", err)

		for _, s := range substrings {
			Expect(err.Error()).To(ContainSubstring(s))
		}
	}

	It("Should reject an empty mapper name", func() {
		u := toWire(newIDP(testutils.UniqueName("idp-empty-name"), validMapper()), func(item map[string]any) { item["name"] = "" })

		expectInvalid(k8sClient.Create(ctx, u), "spec.mappers[0].name", "at least 1 chars long")
	})

	It("Should reject an omitted mapper name", func() {
		u := toWire(newIDP(testutils.UniqueName("idp-no-name"), validMapper()), func(item map[string]any) { delete(item, "name") })

		expectInvalid(k8sClient.Create(ctx, u), "spec.mappers[0].name: Required value")
	})

	It("Should reject an empty identityProviderMapper", func() {
		u := toWire(newIDP(testutils.UniqueName("idp-empty-type"), validMapper()), func(item map[string]any) { item["identityProviderMapper"] = "" })

		expectInvalid(k8sClient.Create(ctx, u), "spec.mappers[0].identityProviderMapper", "at least 1 chars long")
	})

	It("Should reject an omitted identityProviderMapper", func() {
		u := toWire(newIDP(testutils.UniqueName("idp-no-type"), validMapper()), func(item map[string]any) { delete(item, "identityProviderMapper") })

		expectInvalid(k8sClient.Create(ctx, u), "spec.mappers[0].identityProviderMapper: Required value")
	})

	It("Should reject a typed create with an empty mapper name", func() {
		mapper := validMapper()
		mapper.Name = ""

		expectInvalid(k8sClient.Create(ctx, newIDP(testutils.UniqueName("idp-typed-empty-name"), mapper)), "spec.mappers[0].name: Required value")
	})

	It("Should reject a typed create with an empty identityProviderMapper", func() {
		mapper := validMapper()
		mapper.IdentityProviderMapper = ""

		expectInvalid(k8sClient.Create(ctx, newIDP(testutils.UniqueName("idp-typed-empty-type"), mapper)), "spec.mappers[0].identityProviderMapper: Required value")
	})

	It("Should reject an update that empties the mapper name", func() {
		idp := newIDP(testutils.UniqueName("idp-update-empty"), validMapper())
		Expect(k8sClient.Create(ctx, idp)).To(Succeed())

		DeferCleanup(func() { Expect(client.IgnoreNotFound(k8sClient.Delete(ctx, idp))).To(Succeed()) })

		u := toWire(idp, func(item map[string]any) { item["name"] = "" })

		expectInvalid(k8sClient.Update(ctx, u), "spec.mappers[0].name", "at least 1 chars long")
	})

	It("Should accept a valid mapper and an update to another valid name", func() {
		idp := newIDP(testutils.UniqueName("idp-valid"), validMapper())
		Expect(k8sClient.Create(ctx, idp)).To(Succeed())

		DeferCleanup(func() { Expect(client.IgnoreNotFound(k8sClient.Delete(ctx, idp))).To(Succeed()) })

		idp.Spec.Mappers[0].Name = "m2"
		Expect(k8sClient.Update(ctx, idp)).To(Succeed())
	})
})

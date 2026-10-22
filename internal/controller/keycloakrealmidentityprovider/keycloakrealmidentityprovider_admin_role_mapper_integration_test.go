package keycloakrealmidentityprovider

import (
	"maps"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	k8sErrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/epam/edp-keycloak-operator/api/common"
	keycloakApi "github.com/epam/edp-keycloak-operator/api/v1"
	"github.com/epam/edp-keycloak-operator/pkg/testutils"
)

// Keycloak 26.8 rejects a mapper that grants an admin role unless the identity provider
// config carries allowAdminRoleMapping=true.
var _ = Describe("KeycloakRealmIdentityProvider admin role mapper", Ordered, func() {
	const (
		longWait = testutils.LongWait
		settle   = testutils.Settle
	)

	var recorder *testutils.AdminEventRecorder

	newIDP := func(name string, extra map[string]string) *keycloakApi.KeycloakRealmIdentityProvider {
		config := map[string]string{
			"authorizationUrl": "https://idp.example.test/authorize",
			"tokenUrl":         "https://idp.example.test/token",
			"clientId":         "admin-role-client",
			"clientSecret":     "admin-role-secret",
		}
		maps.Copy(config, extra)

		return &keycloakApi.KeycloakRealmIdentityProvider{
			ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: ns},
			Spec: keycloakApi.KeycloakRealmIdentityProviderSpec{
				RealmRef:   common.RealmRef{Kind: keycloakApi.KeycloakRealmKind, Name: KeycloakRealmCR},
				ProviderID: "oidc",
				Alias:      name,
				Enabled:    true,
				Config:     config,
				Mappers: []keycloakApi.IdentityProviderMapper{{
					Name:                   "admin-role",
					IdentityProviderMapper: "oidc-hardcoded-role-idp-mapper",
					Config:                 map[string]string{"role": "realm-management.realm-admin"},
				}},
			},
		}
	}

	deleteIDP := func(name string) {
		key := types.NamespacedName{Name: name, Namespace: ns}

		cr := &keycloakApi.KeycloakRealmIdentityProvider{}
		if err := k8sClient.Get(ctx, key, cr); err != nil {
			return
		}

		Expect(client.IgnoreNotFound(k8sClient.Delete(ctx, cr))).To(Succeed())
		Eventually(func() bool {
			return k8sErrors.IsNotFound(k8sClient.Get(ctx, key, &keycloakApi.KeycloakRealmIdentityProvider{}))
		}, longWait, interval).Should(BeTrue())
	}

	BeforeAll(func() {
		recorder = testutils.NewAdminEventRecorder(keycloakApiClient, KeycloakRealmCR)
		Expect(recorder.Enable(ctx)).To(Succeed())
	})

	It("Should report the Keycloak rejection when allowAdminRoleMapping is absent", func() {
		info, err := keycloakApiClient.GetServerInfo(ctx)
		Expect(err).NotTo(HaveOccurred())

		supported, err := testutils.ServerVersionAtLeast(info, 26, 8)
		Expect(err).NotTo(HaveOccurred())

		if !supported {
			Skip("Keycloak >= 26.8 required, server " + info.SystemInfo.ServerVersion)
		}

		const name = "admin-role-denied"

		DeferCleanup(func() { deleteIDP(name) })

		Expect(k8sClient.Create(ctx, newIDP(name, nil))).To(Succeed())

		Eventually(func(g Gomega) {
			cr := &keycloakApi.KeycloakRealmIdentityProvider{}
			g.Expect(k8sClient.Get(ctx, types.NamespacedName{Name: name, Namespace: ns}, cr)).To(Succeed())
			g.Expect(cr.Status.Value).To(ContainSubstring("not configured to allow granting admin roles"))
		}, longWait, interval).Should(Succeed())
	})

	It("Should create the mapper and settle when allowAdminRoleMapping is true", func() {
		const name = "admin-role-allowed"

		DeferCleanup(func() { deleteIDP(name) })

		scoped := recorder.Scoped("identity-provider/instances/" + name)

		// Liveness anchor for the empty-window assertion below.
		Expect(scoped.WritesDuring(ctx, 0, func() {
			Expect(k8sClient.Create(ctx, newIDP(name, map[string]string{"allowAdminRoleMapping": "true"}))).To(Succeed())

			Eventually(func(g Gomega) {
				cr := &keycloakApi.KeycloakRealmIdentityProvider{}
				g.Expect(k8sClient.Get(ctx, types.NamespacedName{Name: name, Namespace: ns}, cr)).To(Succeed())
				g.Expect(cr.Status.Value).To(Equal(common.StatusOK))
				g.Expect(cr.Status.ObservedGeneration).To(Equal(cr.Generation))
			}, longWait, interval).Should(Succeed())
		})).ShouldNot(BeEmpty(), "admin event log recorded nothing for a known write")

		mappers, _, err := keycloakApiClient.IdentityProviders.GetIDPMappers(ctx, KeycloakRealmCR, name)
		Expect(err).NotTo(HaveOccurred())
		Expect(mappers).To(HaveLen(1))

		Expect(scoped.WritesDuring(ctx, settle, func() {
			Expect(testutils.Nudge(ctx, k8sClient, types.NamespacedName{Name: name, Namespace: ns},
				&keycloakApi.KeycloakRealmIdentityProvider{})).To(Succeed())
		})).To(BeEmpty())
	})
})

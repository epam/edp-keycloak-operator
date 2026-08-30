package keycloakclientscope

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	k8sErrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/utils/ptr"

	"github.com/epam/edp-keycloak-operator/api/common"
	keycloakApi "github.com/epam/edp-keycloak-operator/api/v1"
	"github.com/epam/edp-keycloak-operator/pkg/client/keycloakapi"
	"github.com/epam/edp-keycloak-operator/pkg/testutils"
)

// Idempotency suite for KeycloakClientScope.
var _ = Describe("KeycloakClientScope idempotent reconcile", Ordered, func() {
	const (
		crName    = "idem-scope"
		scopeName = "idem-scope"
		settle    = testutils.Settle
		longWait  = testutils.LongWait
	)

	var (
		recorder *testutils.AdminEventRecorder
		scopeID  string
	)

	nudge := func() {
		Expect(testutils.Nudge(ctx, k8sClient,
			types.NamespacedName{Name: crName, Namespace: ns}, &keycloakApi.KeycloakClientScope{})).To(Succeed())
	}

	updateSpec := func(mutate func(*keycloakApi.KeycloakClientScope)) {
		Eventually(func(g Gomega) {
			cr := &keycloakApi.KeycloakClientScope{}
			g.Expect(k8sClient.Get(ctx, types.NamespacedName{Name: crName, Namespace: ns}, cr)).To(Succeed())
			mutate(cr)
			g.Expect(k8sClient.Update(ctx, cr)).To(Succeed())
		}, longWait, interval).Should(Succeed())
	}

	waitReady := func() {
		Eventually(func(g Gomega) {
			cr := &keycloakApi.KeycloakClientScope{}
			g.Expect(k8sClient.Get(ctx, types.NamespacedName{Name: crName, Namespace: ns}, cr)).To(Succeed())
			g.Expect(cr.Status.Value).To(Equal(common.StatusOK))
			g.Expect(cr.Status.ObservedGeneration).To(Equal(cr.Generation))
		}, longWait, interval).Should(Succeed())
	}

	kcScope := func() *keycloakapi.ClientScopeRepresentation {
		rep, _, err := keycloakApiClient.ClientScopes.GetClientScope(ctx, KeycloakRealmCR, scopeID)
		Expect(err).ShouldNot(HaveOccurred())
		Expect(rep).ShouldNot(BeNil())

		return rep
	}

	mapperIDsByName := func() map[string]string {
		rep := kcScope()
		ids := map[string]string{}

		if rep.ProtocolMappers == nil {
			return ids
		}

		for _, m := range *rep.ProtocolMappers {
			ids[ptr.Deref(m.Name, "")] = ptr.Deref(m.Id, "")
		}

		return ids
	}

	mapper := func(name, claim string) keycloakApi.ProtocolMapper {
		return keycloakApi.ProtocolMapper{
			Name:           name,
			Protocol:       "openid-connect",
			ProtocolMapper: "oidc-hardcoded-claim-mapper",
			Config: map[string]string{
				"claim.name":                 claim,
				"claim.value":                "v1",
				"access.token.claim":         "true",
				"id.token.claim":             "true",
				"jsonType.label":             "String",
				"userinfo.token.claim":       "true",
				"introspection.token.claim":  "true",
				"access.tokenResponse.claim": "false",
			},
		}
	}

	BeforeAll(func() {
		recorder = testutils.NewAdminEventRecorder(keycloakApiClient, KeycloakRealmCR)
		Expect(recorder.Enable(ctx)).To(Succeed())
	})

	It("Should create KeycloakClientScope with protocol mappers", func() {
		cr := &keycloakApi.KeycloakClientScope{
			ObjectMeta: metav1.ObjectMeta{Name: crName, Namespace: ns},
			Spec: keycloakApi.KeycloakClientScopeSpec{
				Name:        scopeName,
				RealmRef:    common.RealmRef{Kind: keycloakApi.KeycloakRealmKind, Name: KeycloakRealmCR},
				Protocol:    "openid-connect",
				Description: "idem scope v1",
				Type:        "none",
				Attributes:  map[string]string{"include.in.token.scope": "true"},
				ProtocolMappers: []keycloakApi.ProtocolMapper{
					mapper("pm-a", "claim-a"),
					mapper("pm-b", "claim-b"),
					mapper("pm-c", "claim-c"),
				},
			},
		}
		Expect(k8sClient.Create(ctx, cr)).To(Succeed())
		waitReady()

		stored := &keycloakApi.KeycloakClientScope{}
		Expect(k8sClient.Get(ctx, types.NamespacedName{Name: crName, Namespace: ns}, stored)).To(Succeed())
		scopeID = stored.Status.ID
		Expect(scopeID).ShouldNot(BeEmpty())

		// Sibling specs share the realm; report only this scope's events.
		recorder = recorder.Scoped("client-scopes/" + scopeID)

		Expect(mapperIDsByName()).To(HaveLen(3))
		Expect(ptr.Deref(kcScope().Description, "")).To(Equal("idem scope v1"))
	})

	It("Should not write to Keycloak on a forced reconcile and keep mapper IDs stable", func() {
		before := mapperIDsByName()

		Expect(recorder.WritesDuring(ctx, settle, nudge)).To(BeEmpty())
		Expect(mapperIDsByName()).To(Equal(before), "mappers must not be recreated")
	})

	It("Should heal external drift on the scope description", func() {
		// Liveness anchor for this suite's empty-window specs. The drift write sits
		// inside the window: the controller reconciles on a timer and can heal it
		// before the nudge lands.
		Expect(recorder.WritesDuring(ctx, 0, func() {
			rep := kcScope()
			rep.Description = ptr.To("drifted-by-hand")
			_, err := keycloakApiClient.ClientScopes.UpdateClientScope(ctx, KeycloakRealmCR, scopeID, *rep)
			Expect(err).ShouldNot(HaveOccurred())

			nudge()

			Eventually(func(g Gomega) {
				g.Expect(ptr.Deref(kcScope().Description, "")).To(Equal("idem scope v1"))
			}, longWait, interval).Should(Succeed())
		})).ShouldNot(BeEmpty(), "admin event log recorded nothing for a known write")
	})

	It("Should update one mapper in place and keep the other mapper IDs", func() {
		before := mapperIDsByName()

		updateSpec(func(cr *keycloakApi.KeycloakClientScope) {
			cr.Spec.ProtocolMappers[0].Config["claim.value"] = "v2"
		})
		waitReady()

		Eventually(func(g Gomega) {
			rep := kcScope()
			g.Expect(rep.ProtocolMappers).ShouldNot(BeNil())

			configs := map[string]map[string]string{}
			for _, m := range *rep.ProtocolMappers {
				configs[ptr.Deref(m.Name, "")] = ptr.Deref(m.Config, nil)
			}

			g.Expect(configs).To(HaveKey("pm-a"))
			g.Expect(configs["pm-a"]).To(HaveKeyWithValue("claim.value", "v2"))
		}, longWait, interval).Should(Succeed())

		after := mapperIDsByName()
		Expect(after).To(HaveLen(3))
		for _, name := range []string{"pm-a", "pm-b", "pm-c"} {
			Expect(after[name]).To(Equal(before[name]), "mapper %s must be updated, not recreated", name)
		}
	})

	It("Should delete only the removed mapper", func() {
		before := mapperIDsByName()

		updateSpec(func(cr *keycloakApi.KeycloakClientScope) {
			kept := make([]keycloakApi.ProtocolMapper, 0, len(cr.Spec.ProtocolMappers))
			for _, m := range cr.Spec.ProtocolMappers {
				if m.Name != "pm-c" {
					kept = append(kept, m)
				}
			}
			cr.Spec.ProtocolMappers = kept
		})
		waitReady()

		Eventually(func(g Gomega) {
			g.Expect(mapperIDsByName()).To(HaveLen(2))
		}, longWait, interval).Should(Succeed())

		after := mapperIDsByName()
		Expect(after).ShouldNot(HaveKey("pm-c"))
		for _, name := range []string{"pm-a", "pm-b"} {
			Expect(after[name]).To(Equal(before[name]), "surviving mapper %s must keep its ID", name)
		}
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
				&keycloakApi.KeycloakClientScope{})).To(Succeed())
		})).ShouldNot(BeEmpty(), "a cleared observedGeneration must force one write")
	})

	It("Should delete KeycloakClientScope and remove the scope from Keycloak", func() {
		cr := &keycloakApi.KeycloakClientScope{}
		Expect(k8sClient.Get(ctx, types.NamespacedName{Name: crName, Namespace: ns}, cr)).To(Succeed())
		Expect(k8sClient.Delete(ctx, cr)).To(Succeed())

		Eventually(func() bool {
			return k8sErrors.IsNotFound(
				k8sClient.Get(ctx, types.NamespacedName{Name: crName, Namespace: ns},
					&keycloakApi.KeycloakClientScope{}))
		}, longWait, interval).Should(BeTrue())

		Eventually(func(g Gomega) {
			scopes, _, err := keycloakApiClient.ClientScopes.GetClientScopes(ctx, KeycloakRealmCR)
			g.Expect(err).ShouldNot(HaveOccurred())

			for _, s := range scopes {
				g.Expect(ptr.Deref(s.Name, "")).ShouldNot(Equal(scopeName), "scope must be gone from Keycloak")
			}
		}, longWait, interval).Should(Succeed())
	})
})

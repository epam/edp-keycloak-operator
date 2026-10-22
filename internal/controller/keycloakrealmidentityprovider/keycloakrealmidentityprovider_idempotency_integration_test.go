package keycloakrealmidentityprovider

import (
	"fmt"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	corev1 "k8s.io/api/core/v1"
	k8sErrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/epam/edp-keycloak-operator/api/common"
	keycloakApi "github.com/epam/edp-keycloak-operator/api/v1"
	"github.com/epam/edp-keycloak-operator/pkg/testutils"
)

// clientSecret is ref-backed first, then literal: the first form is skipped by the
// compare as a reference, the second as the masked value Keycloak returns on GET.
var _ = Describe("KeycloakRealmIdentityProvider idempotent reconcile", Ordered, func() {
	const (
		crName     = "idem-idp"
		idpAlias   = "idem-idp"
		secretName = "idem-idp-secret"
		secretKey  = "clientSecret"
		settle     = testutils.Settle
		longWait   = testutils.LongWait
	)

	crKey := types.NamespacedName{Name: crName, Namespace: ns}

	var recorder *testutils.AdminEventRecorder

	getCR := func(g Gomega) *keycloakApi.KeycloakRealmIdentityProvider {
		cr := &keycloakApi.KeycloakRealmIdentityProvider{}
		g.Expect(k8sClient.Get(ctx, crKey, cr)).To(Succeed())

		return cr
	}

	// nudge stamps a fresh value by merge patch; safe to repeat inside Eventually.
	nudge := func(g Gomega) {
		cr := &keycloakApi.KeycloakRealmIdentityProvider{ObjectMeta: metav1.ObjectMeta{Name: crName, Namespace: ns}}
		patch := client.RawPatch(types.MergePatchType, []byte(fmt.Sprintf(`{"metadata":{"annotations":{%q:%q}}}`,
			testutils.NudgeAnnotation, time.Now().Format(time.RFC3339Nano))))
		g.Expect(k8sClient.Patch(ctx, cr, patch)).To(Succeed())
	}

	updateSpec := func(mutate func(*keycloakApi.KeycloakRealmIdentityProvider)) {
		Eventually(func(g Gomega) {
			cr := getCR(g)
			mutate(cr)
			g.Expect(k8sClient.Update(ctx, cr)).To(Succeed())
		}, longWait, interval).Should(Succeed())
	}

	waitReady := func() {
		Eventually(func(g Gomega) {
			cr := getCR(g)
			g.Expect(cr.Status.Value).To(Equal(common.StatusOK))
			g.Expect(cr.Status.ObservedGeneration).To(Equal(cr.Generation))
		}, longWait, interval).Should(Succeed())
	}

	BeforeAll(func() {
		recorder = testutils.NewAdminEventRecorder(keycloakApiClient, KeycloakRealmCR)
		Expect(recorder.Enable(ctx)).To(Succeed())
		recorder = recorder.Scoped("identity-provider/instances/" + idpAlias)

		DeferCleanup(func() {
			cr := &keycloakApi.KeycloakRealmIdentityProvider{}
			if err := k8sClient.Get(ctx, crKey, cr); err == nil {
				Expect(k8sClient.Delete(ctx, cr)).To(Succeed())
				Eventually(func() bool {
					return k8sErrors.IsNotFound(k8sClient.Get(ctx, crKey, &keycloakApi.KeycloakRealmIdentityProvider{}))
				}, longWait, interval).Should(BeTrue())
			}

			secret := &corev1.Secret{ObjectMeta: metav1.ObjectMeta{Name: secretName, Namespace: ns}}
			Expect(client.IgnoreNotFound(k8sClient.Delete(ctx, secret))).To(Succeed())
		})

		secret := &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{Name: secretName, Namespace: ns},
			StringData: map[string]string{secretKey: "idem-secret-v1"},
		}
		Expect(k8sClient.Create(ctx, secret)).To(Succeed())

		cr := &keycloakApi.KeycloakRealmIdentityProvider{
			ObjectMeta: metav1.ObjectMeta{Name: crName, Namespace: ns},
			Spec: keycloakApi.KeycloakRealmIdentityProviderSpec{
				RealmRef:    common.RealmRef{Kind: keycloakApi.KeycloakRealmKind, Name: KeycloakRealmCR},
				ProviderID:  "oidc",
				Alias:       idpAlias,
				Enabled:     true,
				DisplayName: "Idem IdP",
				Config: map[string]string{
					"authorizationUrl": "https://idp.example.test/authorize",
					"tokenUrl":         "https://idp.example.test/token",
					"clientId":         "idem-client",
					"clientSecret":     fmt.Sprintf("$%s:%s", secretName, secretKey),
				},
			},
		}
		Expect(k8sClient.Create(ctx, cr)).To(Succeed())
		waitReady()
	})

	It("Should not write to Keycloak on a nudge while nothing changed", func() {
		Expect(recorder.WritesDuring(ctx, settle, func() { nudge(Default) })).To(BeEmpty())
	})

	It("Should write on a config edit, then not write on a nudge", func() {
		Expect(recorder.WritesDuring(ctx, 0, func() {
			updateSpec(func(cr *keycloakApi.KeycloakRealmIdentityProvider) {
				cr.Spec.Config["tokenUrl"] = "https://idp.example.test/token-v2"
			})
			waitReady()
		})).NotTo(BeEmpty(), "a config edit must write")

		Eventually(func(g Gomega) {
			idp, _, err := keycloakApiClient.IdentityProviders.GetIdentityProvider(ctx, KeycloakRealmCR, idpAlias)
			g.Expect(err).NotTo(HaveOccurred())
			g.Expect((*idp.Config)["tokenUrl"]).To(Equal("https://idp.example.test/token-v2"))
		}, longWait, interval).Should(Succeed())

		Expect(recorder.WritesDuring(ctx, settle, func() { nudge(Default) })).To(BeEmpty())
	})

	It("Should write the rotated secret without a generation change", func() {
		before := getCR(Default)

		secret := &corev1.Secret{}
		Expect(k8sClient.Get(ctx, types.NamespacedName{Name: secretName, Namespace: ns}, secret)).To(Succeed())
		secret.Data[secretKey] = []byte("idem-secret-v2")
		Expect(k8sClient.Update(ctx, secret)).To(Succeed())

		// Secrets are not watched: re-nudge on each attempt. A forced write may repeat once
		// after a status-update conflict: assert at least one write.
		writes, err := recorder.WritesDuring(ctx, 0, func() {
			Eventually(func(g Gomega) {
				nudge(g)

				got := getCR(g)
				g.Expect(got.Status.Value).To(Equal(common.StatusOK))
				g.Expect(got.Generation).To(Equal(before.Generation))
				g.Expect(got.Status.ConfigSecretsHash).NotTo(Equal(before.Status.ConfigSecretsHash))
			}, longWait, interval).Should(Succeed())
		})
		Expect(err).NotTo(HaveOccurred())
		Expect(writes).NotTo(BeEmpty(), "a rotated secret must write")

		Expect(recorder.WritesDuring(ctx, settle, func() { nudge(Default) })).To(BeEmpty())
	})

	It("Should write when observedGeneration is cleared, then not write on a nudge", func() {
		Expect(recorder.WritesDuring(ctx, 0, func() {
			Expect(testutils.ClearObservedGeneration(ctx, k8sClient, crKey,
				&keycloakApi.KeycloakRealmIdentityProvider{})).To(Succeed())
		})).NotTo(BeEmpty(), "a cleared observedGeneration must force one write")

		Expect(recorder.WritesDuring(ctx, settle, func() { nudge(Default) })).To(BeEmpty())
	})

	It("Should not write on a nudge when clientSecret is a masked literal", func() {
		Expect(recorder.WritesDuring(ctx, 0, func() {
			updateSpec(func(cr *keycloakApi.KeycloakRealmIdentityProvider) {
				cr.Spec.Config["clientSecret"] = "idem-literal-secret"
			})
			waitReady()
		})).NotTo(BeEmpty(), "a spec edit must write")

		Expect(recorder.WritesDuring(ctx, settle, func() { nudge(Default) })).To(BeEmpty())
	})
})

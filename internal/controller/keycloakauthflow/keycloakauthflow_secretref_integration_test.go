package keycloakauthflow

import (
	"fmt"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	corev1 "k8s.io/api/core/v1"
	k8sErrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/epam/edp-keycloak-operator/api/common"
	keycloakApi "github.com/epam/edp-keycloak-operator/api/v1"
	"github.com/epam/edp-keycloak-operator/pkg/secretref"
	"github.com/epam/edp-keycloak-operator/pkg/testutils"
)

// Secret references in authenticatorConfig.config. defaultProvider is the only key
// identity-provider-redirector declares, so it is the only key Keycloak returns in clear;
// a reference there proves resolution through the admin API. apiToken is undeclared and
// masked on GET; a reference there proves the rotation-driven write through admin events.
var _ = Describe("KeycloakAuthFlow secret references", Ordered, func() {
	const (
		crName     = "secretref-flow"
		flowAlias  = "secretref-flow"
		secretName = "secretref-flow-secret"
		secretKey  = "provider"
		tokenKey   = "token"
		settle     = testutils.Settle
		longWait   = testutils.LongWait
	)

	realmCR := testutils.RealmName("secretref-flow-realm")
	crKey := types.NamespacedName{Name: crName, Namespace: ns}
	secretRef := fmt.Sprintf("$%s:%s", secretName, secretKey)
	tokenRef := fmt.Sprintf("$%s:%s", secretName, tokenKey)

	var recorder *testutils.AdminEventRecorder

	getCR := func(g Gomega) *keycloakApi.KeycloakAuthFlow {
		cr := &keycloakApi.KeycloakAuthFlow{}
		g.Expect(k8sClient.Get(ctx, crKey, cr)).To(Succeed())

		return cr
	}

	// nudge stamps a fresh nudge value by merge patch; safe to repeat inside Eventually because
	// it never carries a stale resourceVersion.
	nudge := func(g Gomega) {
		cr := &keycloakApi.KeycloakAuthFlow{ObjectMeta: metav1.ObjectMeta{Name: crName, Namespace: ns}}
		patch := client.RawPatch(types.MergePatchType, []byte(fmt.Sprintf(`{"metadata":{"annotations":{%q:%q}}}`,
			testutils.NudgeAnnotation, time.Now().Format(time.RFC3339Nano))))
		g.Expect(k8sClient.Patch(ctx, cr, patch)).To(Succeed())
	}

	// expectedHash is the hash the handler stamps for the two references, derived from the
	// Secret's current version the same way the resolver derives it.
	expectedHash := func(g Gomega) string {
		secret := &corev1.Secret{}
		g.Expect(k8sClient.Get(ctx, types.NamespacedName{Name: secretName, Namespace: ns}, secret)).To(Succeed())

		return secretref.ValuesHashSingle(map[string]string{
			"0/defaultProvider": secretref.SecretKeyVersion(secret, secretKey),
			"0/apiToken":        secretref.SecretKeyVersion(secret, tokenKey),
		})
	}

	BeforeAll(func() {
		realm := &keycloakApi.KeycloakRealm{
			ObjectMeta: metav1.ObjectMeta{Name: realmCR, Namespace: ns},
			Spec: keycloakApi.KeycloakRealmSpec{
				RealmName:   realmCR,
				KeycloakRef: common.KeycloakRef{Name: KeycloakCR, Kind: keycloakApi.KeycloakKind},
			},
		}
		Expect(k8sClient.Create(ctx, realm)).To(Succeed())

		Eventually(func(g Gomega) {
			got := &keycloakApi.KeycloakRealm{}
			g.Expect(k8sClient.Get(ctx, types.NamespacedName{Name: realmCR, Namespace: ns}, got)).To(Succeed())
			g.Expect(got.Status.Available).To(BeTrue())
		}, longWait, interval).Should(Succeed())

		recorder = testutils.NewAdminEventRecorder(keycloakApiClient, realmCR)
		Expect(recorder.Enable(ctx)).To(Succeed())

		secret := &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{Name: secretName, Namespace: ns},
			StringData: map[string]string{secretKey: "provider-v1", tokenKey: "token-v1"},
		}
		Expect(k8sClient.Create(ctx, secret)).To(Succeed())
	})

	AfterAll(func() {
		cr := &keycloakApi.KeycloakAuthFlow{}
		if err := k8sClient.Get(ctx, crKey, cr); err == nil {
			Expect(k8sClient.Delete(ctx, cr)).To(Succeed())
			Eventually(func() bool {
				return k8sErrors.IsNotFound(k8sClient.Get(ctx, crKey, &keycloakApi.KeycloakAuthFlow{}))
			}, longWait, interval).Should(BeTrue())
		}

		secret := &corev1.Secret{ObjectMeta: metav1.ObjectMeta{Name: secretName, Namespace: ns}}
		Expect(client.IgnoreNotFound(k8sClient.Delete(ctx, secret))).To(Succeed())

		realm := &keycloakApi.KeycloakRealm{}
		if err := k8sClient.Get(ctx, types.NamespacedName{Name: realmCR, Namespace: ns}, realm); err == nil {
			Expect(k8sClient.Delete(ctx, realm)).To(Succeed())
		}
	})

	It("Should resolve the reference and record the secret versions hash", func() {
		cr := &keycloakApi.KeycloakAuthFlow{
			ObjectMeta: metav1.ObjectMeta{Name: crName, Namespace: ns},
			Spec: keycloakApi.KeycloakAuthFlowSpec{
				RealmRef:   common.RealmRef{Kind: keycloakApi.KeycloakRealmKind, Name: realmCR},
				Alias:      flowAlias,
				ProviderID: "basic-flow",
				TopLevel:   true,
				AuthenticationExecutions: []keycloakApi.AuthenticationExecution{{
					Authenticator: "identity-provider-redirector",
					AuthenticatorConfig: &keycloakApi.AuthenticatorConfig{
						Alias:  "secretref-flow-cfg",
						Config: map[string]string{"defaultProvider": secretRef, "apiToken": tokenRef},
					},
					Priority:    0,
					Requirement: "REQUIRED",
				}},
			},
		}
		Expect(k8sClient.Create(ctx, cr)).To(Succeed())

		Eventually(func(g Gomega) {
			got := getCR(g)
			g.Expect(got.Status.Value).To(Equal(common.StatusOK))
			g.Expect(got.Status.ObservedGeneration).To(Equal(got.Generation))
			g.Expect(got.Status.ConfigSecretsHash).To(Equal(expectedHash(g)))
			g.Expect(execProvider(g, realmCR, flowAlias)).To(Equal("provider-v1"))
		}, longWait, interval).Should(Succeed())

		got := getCR(Default)
		Expect(got.Spec.AuthenticationExecutions[0].AuthenticatorConfig.Config["defaultProvider"]).
			To(Equal(secretRef), "the stored spec must keep the reference, not the resolved value")
	})

	It("Should not write to Keycloak on a nudge while the secret is unchanged", func() {
		before := getCR(Default)

		Expect(recorder.WritesDuring(ctx, settle, func() { nudge(Default) })).To(BeEmpty())

		after := getCR(Default)
		Expect(after.Status.ConfigSecretsHash).To(Equal(before.Status.ConfigSecretsHash))
		Expect(after.Status.ID).To(Equal(before.Status.ID))
	})

	It("Should write the rotated value of a masked key without a generation change", func() {
		before := getCR(Default)

		secret := &corev1.Secret{}
		Expect(k8sClient.Get(ctx, types.NamespacedName{Name: secretName, Namespace: ns}, secret)).To(Succeed())
		secret.Data[tokenKey] = []byte("token-v2")
		Expect(k8sClient.Update(ctx, secret)).To(Succeed())

		// apiToken is masked on GET: the config write is the evidence. Secrets are not watched:
		// re-nudge on each attempt. A forced write may repeat once after a status-update 409:
		// assert at least one config UPDATE and no other write.
		configWrites := recorder.Scoped("authentication/config")

		writes, err := configWrites.WritesDuring(ctx, 0, func() {
			Eventually(func(g Gomega) {
				nudge(g)

				got := getCR(g)
				g.Expect(got.Status.Value).To(Equal(common.StatusOK))
				g.Expect(got.Generation).To(Equal(before.Generation))
				g.Expect(got.Status.ConfigSecretsHash).To(Equal(expectedHash(g)))
				g.Expect(got.Status.ConfigSecretsHash).NotTo(Equal(before.Status.ConfigSecretsHash))
			}, longWait, interval).Should(Succeed())
		})
		Expect(err).NotTo(HaveOccurred())
		Expect(writes).NotTo(BeEmpty())

		for _, w := range writes {
			Expect(ptr.Deref(w.OperationType, "")).To(Equal("UPDATE"))
			Expect(ptr.Deref(w.ResourcePath, "")).To(ContainSubstring("authentication/config"))
		}

		Expect(execProvider(Default, realmCR, flowAlias)).To(Equal("provider-v1"))

		// The rotation has settled: a nudge with the new hash persisted writes nothing.
		Expect(configWrites.WritesDuring(ctx, settle, func() { nudge(Default) })).To(BeEmpty())
	})

	It("Should report a missing secret key in status and keep the last value in Keycloak", func() {
		secret := &corev1.Secret{}
		Expect(k8sClient.Get(ctx, types.NamespacedName{Name: secretName, Namespace: ns}, secret)).To(Succeed())
		delete(secret.Data, secretKey)
		Expect(k8sClient.Update(ctx, secret)).To(Succeed())

		Eventually(func(g Gomega) {
			nudge(g)

			got := getCR(g)
			g.Expect(got.Status.Value).To(ContainSubstring(`execution 0 ("identity-provider-redirector"): unable to map config secrets`))
			g.Expect(got.Status.Value).To(ContainSubstring("does not contain key " + secretKey))
		}, longWait, interval).Should(Succeed())

		Expect(execProvider(Default, realmCR, flowAlias)).To(Equal("provider-v1"))
	})
})

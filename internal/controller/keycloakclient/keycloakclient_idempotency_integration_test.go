package keycloakclient

import (
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
	"github.com/epam/edp-keycloak-operator/pkg/client/keycloakapi"
	"github.com/epam/edp-keycloak-operator/pkg/secretref"
	"github.com/epam/edp-keycloak-operator/pkg/testutils"
)

// Idempotency suite for the KeycloakClient client PUT. Keycloak's admin event log is the write
// oracle: each client PUT emits one UPDATE on clients/<uuid>. The management permissions PUT emits
// no event and is covered by unit tests.
// The admin GET returns no masked client value, so the suite has no masked key; the masked branch is
// covered by the chain unit tests "fetched secret masked" and "masked fetched attribute".
var _ = Describe("KeycloakClient idempotent reconcile", Ordered, func() {
	const (
		secretKey = "secret"
		settle    = testutils.Settle
		longWait  = testutils.LongWait
		descV1    = "idem client v1"
	)

	var (
		crName     = testutils.UniqueName("idem-client")
		secretName = testutils.UniqueName("idem-client-sec")
		crKey      = types.NamespacedName{Name: crName, Namespace: ns}
		secretKeyN = types.NamespacedName{Name: secretName, Namespace: ns}

		recorder   *testutils.AdminEventRecorder
		clientUUID string
	)

	nudge := func() {
		Expect(testutils.Nudge(ctx, k8sClient, crKey, &keycloakApi.KeycloakClient{})).To(Succeed())
	}

	getCR := func(g Gomega) *keycloakApi.KeycloakClient {
		cr := &keycloakApi.KeycloakClient{}
		g.Expect(k8sClient.Get(ctx, crKey, cr)).To(Succeed())

		return cr
	}

	waitReady := func() {
		Eventually(func(g Gomega) {
			cr := getCR(g)
			g.Expect(cr.Status.Value).To(Equal(common.StatusOK))
			g.Expect(cr.Status.ObservedGeneration).To(Equal(cr.Generation))
		}, longWait, interval).Should(Succeed())
	}

	updateSpec := func(mutate func(*keycloakApi.KeycloakClient)) {
		Eventually(func(g Gomega) {
			cr := getCR(g)
			mutate(cr)
			g.Expect(k8sClient.Update(ctx, cr)).To(Succeed())
		}, longWait, interval).Should(Succeed())
	}

	kcClient := func() *keycloakapi.ClientRepresentation {
		rep, _, err := keycloakAdmin.Clients.GetClientByClientID(ctx, KeycloakRealmCR, crName)
		Expect(err).ShouldNot(HaveOccurred())
		Expect(rep).ShouldNot(BeNil())

		return rep
	}

	putKeycloak := func(mutate func(*keycloakapi.ClientRepresentation)) {
		rep := kcClient()
		mutate(rep)

		_, err := keycloakAdmin.Clients.UpdateClient(ctx, KeycloakRealmCR, clientUUID, *rep)
		Expect(err).ShouldNot(HaveOccurred())
	}

	BeforeAll(func() {
		recorder = testutils.NewAdminEventRecorder(keycloakAdmin, KeycloakRealmCR)
		Expect(recorder.Enable(ctx)).To(Succeed())

		Expect(k8sClient.Create(ctx, &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{Name: secretName, Namespace: ns},
			Data:       map[string][]byte{secretKey: []byte("idem-v1")},
		})).To(Succeed())
	})

	AfterAll(func() {
		cr := &keycloakApi.KeycloakClient{}
		if err := k8sClient.Get(ctx, crKey, cr); err == nil {
			Expect(k8sClient.Delete(ctx, cr)).To(Succeed())

			Eventually(func() bool {
				return k8sErrors.IsNotFound(k8sClient.Get(ctx, crKey, &keycloakApi.KeycloakClient{}))
			}, longWait, interval).Should(BeTrue())
		}

		secret := &corev1.Secret{}
		if err := k8sClient.Get(ctx, secretKeyN, secret); err == nil {
			Expect(k8sClient.Delete(ctx, secret)).To(Succeed())
		}
	})

	It("Should create the client and stamp observedGeneration", func() {
		cr := &keycloakApi.KeycloakClient{
			ObjectMeta: metav1.ObjectMeta{Name: crName, Namespace: ns},
			Spec: keycloakApi.KeycloakClientSpec{
				ClientId:                crName,
				RealmRef:                common.RealmRef{Name: KeycloakRealmCR, Kind: keycloakApi.KeycloakRealmKind},
				Secret:                  secretref.GenerateSecretRef(secretName, secretKey),
				Description:             descV1,
				Enabled:                 true,
				StandardFlowEnabled:     true,
				FullScopeAllowed:        true,
				ClientAuthenticatorType: "client-secret",
				WebUrl:                  "https://idem-client.example.com",
				RedirectUris: ptr.To([]string{
					"https://idem-client.example.com/z/*",
					"https://idem-client.example.com/a/*",
				}),
				Attributes: map[string]string{
					"idem.attr":  "one",
					"idem.blank": "",
				},
			},
		}
		Expect(k8sClient.Create(ctx, cr)).To(Succeed())
		waitReady()

		Eventually(func(g Gomega) {
			g.Expect(getCR(g).Status.ConfigSecretsHash).ShouldNot(BeEmpty())
		}, longWait, interval).Should(Succeed())

		uuid, err := keycloakAdmin.Clients.GetClientUUID(ctx, KeycloakRealmCR, crName)
		Expect(err).ShouldNot(HaveOccurred())
		Expect(uuid).ShouldNot(BeEmpty())

		clientUUID = uuid
		recorder = recorder.Scoped("clients/" + clientUUID)
	})

	It("Should not write to Keycloak on a forced reconcile", func() {
		Expect(recorder.WritesDuring(ctx, settle, nudge)).To(BeEmpty())
	})

	It("Should heal external drift on the description", func() {
		// Liveness anchor for the empty-window specs.
		Expect(recorder.WritesDuring(ctx, 0, func() {
			putKeycloak(func(rep *keycloakapi.ClientRepresentation) {
				rep.Description = ptr.To("drifted-by-hand")
			})

			nudge()

			Eventually(func(g Gomega) {
				g.Expect(ptr.Deref(kcClient().Description, "")).To(Equal(descV1))
			}, longWait, interval).Should(Succeed())
		})).ShouldNot(BeEmpty(), "admin event log recorded nothing for a known write")
	})

	It("Should propagate Secret rotation end to end", func() {
		before := ""

		Eventually(func(g Gomega) {
			before = getCR(g).Status.ConfigSecretsHash
			g.Expect(before).ShouldNot(BeEmpty())
		}, longWait, interval).Should(Succeed())

		Expect(recorder.WritesDuring(ctx, 0, func() {
			Eventually(func(g Gomega) {
				secret := &corev1.Secret{}
				g.Expect(k8sClient.Get(ctx, secretKeyN, secret)).To(Succeed())

				patch := client.MergeFrom(secret.DeepCopy())
				secret.Data[secretKey] = []byte("idem-v2")
				g.Expect(k8sClient.Patch(ctx, secret, patch)).To(Succeed())

				nudge()

				g.Expect(getCR(g).Status.ConfigSecretsHash).ShouldNot(Equal(before))
			}, longWait, interval).Should(Succeed())
		})).ShouldNot(BeEmpty(), "a changed Secret version must force one write")

		Eventually(func(g Gomega) {
			rep, _, err := keycloakAdmin.Clients.GetClientByClientID(ctx, KeycloakRealmCR, crName)
			g.Expect(err).ShouldNot(HaveOccurred())
			g.Expect(rep).ShouldNot(BeNil())
			g.Expect(ptr.Deref(rep.Secret, "")).To(Equal("idem-v2"))
		}, longWait, interval).Should(Succeed())
	})

	It("Should write on a spec edit and advance observedGeneration", func() {
		Expect(recorder.WritesDuring(ctx, 0, func() {
			updateSpec(func(cr *keycloakApi.KeycloakClient) {
				cr.Spec.Description = "idem client v2"
			})
			waitReady()

			Eventually(func(g Gomega) {
				g.Expect(ptr.Deref(kcClient().Description, "")).To(Equal("idem client v2"))
			}, longWait, interval).Should(Succeed())
		})).ShouldNot(BeEmpty())
	})

	It("Should force one write when observedGeneration is cleared", func() {
		// No drift: Keycloak matches spec, so a write can only come from the cleared observedGeneration.
		Expect(recorder.WritesDuring(ctx, 0, func() {
			Expect(testutils.ClearObservedGeneration(ctx, k8sClient, crKey, &keycloakApi.KeycloakClient{})).To(Succeed())
		})).ShouldNot(BeEmpty(), "a cleared observedGeneration must force one write")
	})

	It("Should not write to Keycloak after the edits settle", func() {
		Expect(recorder.WritesDuring(ctx, settle, nudge)).To(BeEmpty())
	})
})

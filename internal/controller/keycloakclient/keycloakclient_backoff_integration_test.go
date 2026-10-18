package keycloakclient

import (
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	corev1 "k8s.io/api/core/v1"
	k8sErrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/epam/edp-keycloak-operator/api/common"
	keycloakApi "github.com/epam/edp-keycloak-operator/api/v1"
	"github.com/epam/edp-keycloak-operator/internal/controller/keycloakclient/chain"
	"github.com/epam/edp-keycloak-operator/pkg/secretref"
)

// The failure status write must not enqueue the next reconcile; the first backoff is 10s.
var _ = Describe("KeycloakClient failure backoff", Ordered, func() {
	const (
		crName     = "backoff-keycloak-client"
		secretName = "backoff-keycloak-client-secret"
		window     = 5 * time.Second
		// Below the next timed requeue (>= 20s) so recovery proves the spec edit triggered it.
		recoverWithin = 15 * time.Second
	)

	key := types.NamespacedName{Name: crName, Namespace: ns}

	failureCount := func(g Gomega) int64 {
		cr := &keycloakApi.KeycloakClient{}
		g.Expect(k8sClient.Get(ctx, key, cr)).To(Succeed())

		return cr.Status.FailureCount
	}

	AfterAll(func() {
		cr := &keycloakApi.KeycloakClient{ObjectMeta: metav1.ObjectMeta{Name: crName, Namespace: ns}}
		Expect(client.IgnoreNotFound(k8sClient.Delete(ctx, cr))).To(Succeed())

		Eventually(func() bool {
			return k8sErrors.IsNotFound(k8sClient.Get(ctx, key, &keycloakApi.KeycloakClient{}))
		}, timeout, interval).Should(BeTrue())

		secret := &corev1.Secret{ObjectMeta: metav1.ObjectMeta{Name: secretName, Namespace: ns}}
		Expect(client.IgnoreNotFound(k8sClient.Delete(ctx, secret))).To(Succeed())
	})

	It("Should back off after a chain failure instead of reconciling on its own status write", func() {
		cr := &keycloakApi.KeycloakClient{
			ObjectMeta: metav1.ObjectMeta{Name: crName, Namespace: ns},
			Spec: keycloakApi.KeycloakClientSpec{
				ClientId: crName,
				RealmRef: common.RealmRef{
					Name: KeycloakRealmCR,
					Kind: keycloakApi.KeycloakRealmKind,
				},
				Secret: secretref.GenerateSecretRef("backoff-missing-secret", "secretKey"),
			},
		}
		Expect(k8sClient.Create(ctx, cr)).To(Succeed())

		var first int64

		Eventually(func(g Gomega) {
			got := &keycloakApi.KeycloakClient{}
			g.Expect(k8sClient.Get(ctx, key, got)).To(Succeed())

			first = got.Status.FailureCount
			g.Expect(first).To(BeNumerically(">=", 1))
			g.Expect(got.Status.Value).To(ContainSubstring("backoff-missing-secret"))

			ready := meta.FindStatusCondition(got.Status.Conditions, chain.ConditionReady)
			g.Expect(ready).NotTo(BeNil())
			g.Expect(ready.Status).To(Equal(metav1.ConditionFalse))
		}, timeout, interval).Should(Succeed())

		// The finalizer patch and the first ClientSynced=False condition write pass the predicate
		// as separate events; each may enqueue one more reconcile.
		Consistently(failureCount, window, interval).Should(BeNumerically("<=", first+2))
	})

	It("Should reconcile a spec change immediately while backing off", func() {
		g := NewWithT(GinkgoT())
		last := failureCount(g)

		// Wait for a timed requeue so the next one is at least 20s away.
		Eventually(failureCount, 35*time.Second, interval).Should(BeNumerically(">", last))

		secret := &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{Name: secretName, Namespace: ns},
			Data:       map[string][]byte{"secretKey": []byte("secretValue")},
		}
		Expect(k8sClient.Create(ctx, secret)).To(Succeed())

		Eventually(func(g Gomega) {
			cr := &keycloakApi.KeycloakClient{}
			g.Expect(k8sClient.Get(ctx, key, cr)).To(Succeed())
			cr.Spec.Secret = secretref.GenerateSecretRef(secretName, "secretKey")
			g.Expect(k8sClient.Update(ctx, cr)).To(Succeed())
		}, timeout, interval).Should(Succeed())

		Eventually(func(g Gomega) {
			cr := &keycloakApi.KeycloakClient{}
			g.Expect(k8sClient.Get(ctx, key, cr)).To(Succeed())
			g.Expect(cr.Status.Value).To(Equal(common.StatusOK))
			g.Expect(cr.Status.FailureCount).To(BeZero())

			ready := meta.FindStatusCondition(cr.Status.Conditions, chain.ConditionReady)
			g.Expect(ready).NotTo(BeNil())
			g.Expect(ready.Status).To(Equal(metav1.ConditionTrue))
		}, recoverWithin, interval).Should(Succeed())
	})
})

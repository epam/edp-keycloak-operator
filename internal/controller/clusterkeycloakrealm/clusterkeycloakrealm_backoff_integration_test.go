package clusterkeycloakrealm

import (
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	corev1 "k8s.io/api/core/v1"
	k8sErrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/epam/edp-keycloak-operator/api/common"
	keycloakAlpha "github.com/epam/edp-keycloak-operator/api/v1alpha1"
	"github.com/epam/edp-keycloak-operator/pkg/testutils"
)

// The failure status write must not enqueue the next reconcile; the first backoff is 10s.
var _ = Describe("ClusterKeycloakRealm failure backoff", Ordered, func() {
	const (
		crName = "backoff-cluster-realm"
		window = 5 * time.Second
	)

	realmName := testutils.RealmName("backoff-cluster-realm")
	key := types.NamespacedName{Name: crName}

	failureCount := func(g Gomega) int64 {
		cr := &keycloakAlpha.ClusterKeycloakRealm{}
		g.Expect(k8sClient.Get(ctx, key, cr)).To(Succeed())

		return cr.Status.FailureCount
	}

	AfterAll(func() {
		cr := &keycloakAlpha.ClusterKeycloakRealm{ObjectMeta: metav1.ObjectMeta{Name: crName}}
		Expect(client.IgnoreNotFound(k8sClient.Delete(ctx, cr))).To(Succeed())

		Eventually(func() bool {
			return k8sErrors.IsNotFound(k8sClient.Get(ctx, key, &keycloakAlpha.ClusterKeycloakRealm{}))
		}, timeout, interval).Should(BeTrue())
	})

	It("Should back off after a chain failure instead of reconciling on its own status write", func() {
		cr := &keycloakAlpha.ClusterKeycloakRealm{
			ObjectMeta: metav1.ObjectMeta{Name: crName},
			Spec: keycloakAlpha.ClusterKeycloakRealmSpec{
				ClusterKeycloakRef: ClusterKeycloakCR,
				RealmName:          realmName,
				Smtp: &common.SMTP{
					Template: common.EmailTemplate{From: "from@mail.com"},
					Connection: common.EmailConnection{
						Host: "smtp-host",
						Authentication: &common.EmailAuthentication{
							Username: common.SourceRefOrVal{Value: "user"},
							Password: common.SourceRef{
								SecretKeyRef: &common.SecretKeySelector{
									LocalObjectReference: corev1.LocalObjectReference{Name: "missing-smtp-secret"},
									Key:                  "password",
								},
							},
						},
					},
				},
			},
		}
		Expect(k8sClient.Create(ctx, cr)).To(Succeed())

		var first int64

		Eventually(func(g Gomega) {
			first = failureCount(g)
			g.Expect(first).To(BeNumerically(">=", 1))
		}, timeout, interval).Should(Succeed())

		got := &keycloakAlpha.ClusterKeycloakRealm{}
		Expect(k8sClient.Get(ctx, key, got)).To(Succeed())
		Expect(got.Status.Available).To(BeFalse())
		Expect(got.Status.Value).To(ContainSubstring("missing-smtp-secret"))

		// Finalizer and owner-reference writes in the first reconcile may enqueue one more.
		Consistently(failureCount, window, interval).Should(BeNumerically("<=", first+1))
	})

	It("Should reconcile a spec change immediately while backing off", func() {
		g := NewWithT(GinkgoT())
		last := failureCount(g)

		// Wait for a timed requeue so the next one is at least 20s away.
		Eventually(failureCount, 35*time.Second, interval).Should(BeNumerically(">", last))

		Eventually(func(g Gomega) {
			cr := &keycloakAlpha.ClusterKeycloakRealm{}
			g.Expect(k8sClient.Get(ctx, key, cr)).To(Succeed())
			cr.Spec.Smtp = nil
			g.Expect(k8sClient.Update(ctx, cr)).To(Succeed())
		}, timeout, interval).Should(Succeed())

		Eventually(func(g Gomega) {
			cr := &keycloakAlpha.ClusterKeycloakRealm{}
			g.Expect(k8sClient.Get(ctx, key, cr)).To(Succeed())
			g.Expect(cr.Status.Available).To(BeTrue())
			g.Expect(cr.Status.FailureCount).To(BeZero())
			g.Expect(cr.Status.ObservedGeneration).To(Equal(cr.Generation))
		}, 5*time.Second, interval).Should(Succeed())
	})
})

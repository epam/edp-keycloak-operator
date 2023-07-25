package clusterkeycloakrealm

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	keycloakAlpha "github.com/epam/edp-keycloak-operator/api/v1alpha1"
)

var _ = Describe("ClusterKeycloakRealm controller", func() {
	const (
		clusterKeycloakCR = "test-cluster-keycloak-realm"
	)
	It("Should reconcile ClusterKeycloakRealm", func() {
		By("By creating a ClusterKeycloakRealm")
		keycloakRealm := &keycloakAlpha.ClusterKeycloakRealm{
			ObjectMeta: metav1.ObjectMeta{
				Name: clusterKeycloakCR,
			},
			Spec: keycloakAlpha.ClusterKeycloakRealmSpec{
				ClusterKeycloakRef: ClusterKeycloakCR,
				RealmName:          "test-realm",
				FrontendURL:        "https://test.com",
			},
		}
		Expect(k8sClient.Create(ctx, keycloakRealm)).Should(Succeed())
		Eventually(func() bool {
			createdKeycloakRealm := &keycloakAlpha.ClusterKeycloakRealm{}
			err := k8sClient.Get(ctx, types.NamespacedName{Name: clusterKeycloakCR}, createdKeycloakRealm)
			Expect(err).ShouldNot(HaveOccurred())

			return createdKeycloakRealm.Status.Available
		}, timeout, interval).Should(BeTrue())
	})
})

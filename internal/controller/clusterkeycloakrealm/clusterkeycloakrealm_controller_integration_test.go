package clusterkeycloakrealm

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	k8sErrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	"github.com/epam/edp-keycloak-operator/api/common"
	keycloakAlpha "github.com/epam/edp-keycloak-operator/api/v1alpha1"
	"github.com/epam/edp-keycloak-operator/pkg/objectmeta"
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
				TokenSettings: &common.TokenSettings{
					DefaultSignatureAlgorithm:           "RS256",
					RevokeRefreshToken:                  true,
					RefreshTokenMaxReuse:                230,
					AccessTokenLifespan:                 231,
					AccessTokenLifespanForImplicitFlow:  232,
					AccessCodeLifespan:                  233,
					ActionTokenGeneratedByUserLifespan:  234,
					ActionTokenGeneratedByAdminLifespan: 235,
				},
				DisplayName:     "Test Realm",
				DisplayHTMLName: "<b>Test Realm</b>",
				RealmEventConfig: &keycloakAlpha.RealmEventConfig{
					AdminEventsEnabled:    true,
					AdminEventsExpiration: 100,
				},
			},
		}
		Expect(k8sClient.Create(ctx, keycloakRealm)).Should(Succeed())
		Eventually(func() bool {
			createdKeycloakRealm := &keycloakAlpha.ClusterKeycloakRealm{}
			err := k8sClient.Get(ctx, types.NamespacedName{Name: clusterKeycloakCR}, createdKeycloakRealm)
			Expect(err).ShouldNot(HaveOccurred())

			return createdKeycloakRealm.Status.Available
		}, timeout, interval).Should(BeTrue())
		By("By deleting ClusterKeycloakRealm")
		Expect(k8sClient.Delete(ctx, keycloakRealm)).Should(Succeed())
		Eventually(func() bool {
			deletedRealm := &keycloakAlpha.ClusterKeycloakRealm{}
			err := k8sClient.Get(ctx, types.NamespacedName{Name: clusterKeycloakCR}, deletedRealm)

			return k8sErrors.IsNotFound(err)
		}, timeout, interval).Should(BeTrue(), "ClusterKeycloakRealm should be deleted")
	})
	It("Should skip keycloak resource removing if preserveResourcesOnDeletion is set", func() {
		By("By creating a ClusterKeycloakRealm")
		keycloakRealm := &keycloakAlpha.ClusterKeycloakRealm{
			ObjectMeta: metav1.ObjectMeta{
				Name: "test-cluster-keycloak-realm",
				Annotations: map[string]string{
					objectmeta.PreserveResourcesOnDeletionAnnotation: "true",
				},
			},
			Spec: keycloakAlpha.ClusterKeycloakRealmSpec{
				ClusterKeycloakRef: ClusterKeycloakCR,
				RealmName:          "test-realm2",
				FrontendURL:        "https://test.com",
			},
		}
		Expect(k8sClient.Create(ctx, keycloakRealm)).Should(Succeed())
		Eventually(func() bool {
			createdKeycloakRealm := &keycloakAlpha.ClusterKeycloakRealm{}
			err := k8sClient.Get(ctx, types.NamespacedName{Name: keycloakRealm.Name}, createdKeycloakRealm)
			Expect(err).ShouldNot(HaveOccurred())

			return createdKeycloakRealm.Status.Available
		}, timeout, interval).Should(BeTrue())
		By("By deleting ClusterKeycloakRealm")
		Expect(k8sClient.Delete(ctx, keycloakRealm)).Should(Succeed())
		Eventually(func() bool {
			deletedRealm := &keycloakAlpha.ClusterKeycloakRealm{}
			err := k8sClient.Get(ctx, types.NamespacedName{Name: keycloakRealm.Name}, deletedRealm)

			return k8sErrors.IsNotFound(err)
		}, timeout, interval).Should(BeTrue(), "ClusterKeycloakRealm with preserveResourcesOnDeletion annotation should be deleted")
	})
})

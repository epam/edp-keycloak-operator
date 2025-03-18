package clusterkeycloak

import (
	"context"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	keycloakAlpha "github.com/epam/edp-keycloak-operator/api/v1alpha1"
)

var _ = Describe("ClusterKeycloak controller", func() {
	const (
		timeout            = time.Second * 10
		interval           = time.Millisecond * 250
		keycloakName       = "test-keycloak"
		keycloakSecretName = "keycloak-auth-secret"
	)

	ctx := context.Background()

	It("Should create ClusterKeycloak object with secret auth", func() {
		By("By creating a secret")
		secret := &v1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      keycloakSecretName,
				Namespace: "default",
			},
			Data: map[string][]byte{
				"username": []byte("admin"),
				"password": []byte("admin"),
			},
		}
		Expect(k8sClient.Create(ctx, secret)).Should(Succeed())
		By("By creating a new ClusterKeycloak object")
		newKeycloak := &keycloakAlpha.ClusterKeycloak{
			ObjectMeta: metav1.ObjectMeta{
				Name: keycloakName,
			},
			Spec: keycloakAlpha.ClusterKeycloakSpec{
				Url:    keycloakURL,
				Secret: keycloakSecretName,
			},
		}
		Expect(k8sClient.Create(ctx, newKeycloak)).Should(Succeed())
		Eventually(func() bool {
			createdKeycloak := &keycloakAlpha.ClusterKeycloak{}
			err := k8sClient.Get(ctx, types.NamespacedName{Name: keycloakName}, createdKeycloak)
			if err != nil {
				return false
			}
			return createdKeycloak.Status.Connected
		}, timeout, interval).Should(BeTrue())
	})
})

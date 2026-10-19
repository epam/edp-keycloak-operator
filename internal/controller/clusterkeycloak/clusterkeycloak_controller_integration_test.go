package clusterkeycloak

import (
	"context"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	"github.com/epam/edp-keycloak-operator/api/common"
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
			return createdKeycloak.Status.Connected && createdKeycloak.Status.Value == common.StatusOK
		}, timeout, interval).Should(BeTrue())
	})

	It("Should reject tlsServerName with a non-https url", func() {
		newKeycloak := &keycloakAlpha.ClusterKeycloak{
			ObjectMeta: metav1.ObjectMeta{
				Name: "test-keycloak-tls-server-name-http",
			},
			Spec: keycloakAlpha.ClusterKeycloakSpec{
				Url:           "http://keycloak.keycloak.svc.cluster.local:8080",
				Secret:        "absent-auth-secret",
				TLSServerName: "keycloak.example.com",
			},
		}

		err := k8sClient.Create(ctx, newKeycloak)
		Expect(err).Should(HaveOccurred())
		Expect(err.Error()).Should(ContainSubstring("tlsServerName requires an https url"))
	})

	It("Should reject tlsServerName with a port", func() {
		newKeycloak := &keycloakAlpha.ClusterKeycloak{
			ObjectMeta: metav1.ObjectMeta{
				Name: "test-keycloak-tls-server-name-port",
			},
			Spec: keycloakAlpha.ClusterKeycloakSpec{
				Url:           "https://keycloak.keycloak.svc.cluster.local:8443",
				Secret:        "absent-auth-secret",
				TLSServerName: "keycloak:8443",
			},
		}

		err := k8sClient.Create(ctx, newKeycloak)
		Expect(err).Should(HaveOccurred())
		Expect(err.Error()).Should(ContainSubstring("spec.tlsServerName"))
	})

	It("Should accept tlsServerName with an https url", func() {
		newKeycloak := &keycloakAlpha.ClusterKeycloak{
			ObjectMeta: metav1.ObjectMeta{
				Name: "test-keycloak-tls-server-name-https",
			},
			Spec: keycloakAlpha.ClusterKeycloakSpec{
				Url:           "HTTPS://keycloak.keycloak.svc.cluster.local:8443",
				Secret:        "absent-auth-secret",
				TLSServerName: "keycloak.example.com",
			},
		}
		Expect(k8sClient.Create(ctx, newKeycloak)).Should(Succeed())
		Expect(k8sClient.Delete(ctx, newKeycloak)).Should(Succeed())
	})
})

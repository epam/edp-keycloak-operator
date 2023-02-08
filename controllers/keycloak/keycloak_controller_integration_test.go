package keycloak

import (
	"context"
	"os"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	keycloakApi "github.com/epam/edp-keycloak-operator/api/v1"
)

var _ = Describe("Keycloak controller", func() {
	const (
		KeycloakName      = "test-keycloak"
		KeycloakNamespace = "test-keycloak"

		timeout  = time.Second * 10
		interval = time.Millisecond * 250
	)

	ctx := context.Background()

	namespace := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: KeycloakNamespace,
		},
	}

	BeforeEach(func() {
		By("Creating the Namespace to perform the tests")
		err := k8sClient.Create(ctx, namespace)
		Expect(err).To(Not(HaveOccurred()))
	})

	AfterEach(func() {
		By("Deleting the Namespace to perform the tests")
		_ = k8sClient.Delete(ctx, namespace)
	})

	It("Should create Keycloak object with secret auth", func() {
		By("By creating a secret")
		ctx := context.Background()
		secret := &v1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "keycloak-auth-secret",
				Namespace: KeycloakNamespace,
			},
			Data: map[string][]byte{
				"username": []byte("admin"),
				"password": []byte("admin"),
			},
		}
		Expect(k8sClient.Create(ctx, secret)).Should(Succeed())
		By("By creating a new Keycloak object")
		newKeycloak := &keycloakApi.Keycloak{
			ObjectMeta: metav1.ObjectMeta{
				Name:      KeycloakName,
				Namespace: KeycloakNamespace,
			},
			Spec: keycloakApi.KeycloakSpec{
				Url:    os.Getenv("TEST_KEYCLOAK_URL"),
				Secret: secret.Name,
			},
		}
		Expect(k8sClient.Create(ctx, newKeycloak)).Should(Succeed())
		Eventually(func() bool {
			createdKeycloak := &keycloakApi.Keycloak{}
			err := k8sClient.Get(ctx, types.NamespacedName{Name: KeycloakName, Namespace: KeycloakNamespace}, createdKeycloak)
			if err != nil {
				return false
			}
			return createdKeycloak.Status.Connected
		}, timeout, interval).Should(BeTrue())
	})
})

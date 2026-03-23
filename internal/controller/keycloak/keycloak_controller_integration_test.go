package keycloak

import (
	"context"
	"os"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	"github.com/epam/edp-keycloak-operator/api/common"
	keycloakApi "github.com/epam/edp-keycloak-operator/api/v1"
	keycloakv2 "github.com/epam/edp-keycloak-operator/pkg/client/keycloakv2"
)

var _ = Describe("Keycloak controller", Ordered, func() {
	const (
		testNamespace = "test-keycloak-auth"

		timeout  = time.Second * 20
		interval = time.Millisecond * 250
	)

	ctx := context.Background()
	keycloakURL := os.Getenv("TEST_KEYCLOAK_URL")

	BeforeAll(func() {
		ns := &corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name: testNamespace,
			},
		}
		Expect(k8sClient.Create(ctx, ns)).Should(Succeed())
	})

	AfterAll(func() {
		ns := &corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name: testNamespace,
			},
		}
		_ = k8sClient.Delete(ctx, ns)
	})

	assertKeycloakConnected := func(name string) {
		Eventually(func() bool {
			kc := &keycloakApi.Keycloak{}
			err := k8sClient.Get(ctx, types.NamespacedName{Name: name, Namespace: testNamespace}, kc)
			if err != nil {
				return false
			}

			return kc.Status.Connected
		}, timeout, interval).Should(BeTrue())
	}

	It("Should connect with legacy secret auth", func() {
		secret := &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "kc-legacy-auth-secret",
				Namespace: testNamespace,
			},
			Data: map[string][]byte{
				"username": []byte("admin"),
				"password": []byte("admin"),
			},
		}
		Expect(k8sClient.Create(ctx, secret)).Should(Succeed())

		kc := &keycloakApi.Keycloak{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "kc-legacy-secret",
				Namespace: testNamespace,
			},
			Spec: keycloakApi.KeycloakSpec{
				Url:    keycloakURL,
				Secret: secret.Name,
			},
		}
		Expect(k8sClient.Create(ctx, kc)).Should(Succeed())
		assertKeycloakConnected("kc-legacy-secret")
	})

	It("Should connect with password grant using direct username value", func() {
		secret := &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "pw-grant-direct-secret",
				Namespace: testNamespace,
			},
			Data: map[string][]byte{
				"password": []byte("admin"),
			},
		}
		Expect(k8sClient.Create(ctx, secret)).Should(Succeed())

		kc := &keycloakApi.Keycloak{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "kc-pw-grant-direct",
				Namespace: testNamespace,
			},
			Spec: keycloakApi.KeycloakSpec{
				Url: keycloakURL,
				Auth: &common.AuthSpec{
					PasswordGrant: &common.PasswordGrantConfig{
						Username: common.SourceRefOrVal{
							Value: "admin",
						},
						PasswordRef: common.SecretKeySelector{
							LocalObjectReference: corev1.LocalObjectReference{
								Name: "pw-grant-direct-secret",
							},
							Key: "password",
						},
					},
				},
			},
		}
		Expect(k8sClient.Create(ctx, kc)).Should(Succeed())
		assertKeycloakConnected("kc-pw-grant-direct")
	})

	It("Should connect with password grant using username from SecretKeyRef", func() {
		secret := &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "pw-grant-secretref-secret",
				Namespace: testNamespace,
			},
			Data: map[string][]byte{
				"username": []byte("admin"),
				"password": []byte("admin"),
			},
		}
		Expect(k8sClient.Create(ctx, secret)).Should(Succeed())

		kc := &keycloakApi.Keycloak{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "kc-pw-grant-secretref",
				Namespace: testNamespace,
			},
			Spec: keycloakApi.KeycloakSpec{
				Url: keycloakURL,
				Auth: &common.AuthSpec{
					PasswordGrant: &common.PasswordGrantConfig{
						Username: common.SourceRefOrVal{
							SourceRef: common.SourceRef{
								SecretKeyRef: &common.SecretKeySelector{
									LocalObjectReference: corev1.LocalObjectReference{
										Name: "pw-grant-secretref-secret",
									},
									Key: "username",
								},
							},
						},
						PasswordRef: common.SecretKeySelector{
							LocalObjectReference: corev1.LocalObjectReference{
								Name: "pw-grant-secretref-secret",
							},
							Key: "password",
						},
					},
				},
			},
		}
		Expect(k8sClient.Create(ctx, kc)).Should(Succeed())
		assertKeycloakConnected("kc-pw-grant-secretref")
	})

	It("Should connect with password grant using username from ConfigMapKeyRef", func() {
		configMap := &corev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "pw-grant-configmap",
				Namespace: testNamespace,
			},
			Data: map[string]string{
				"username": "admin",
			},
		}
		Expect(k8sClient.Create(ctx, configMap)).Should(Succeed())

		secret := &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "pw-grant-configmap-secret",
				Namespace: testNamespace,
			},
			Data: map[string][]byte{
				"password": []byte("admin"),
			},
		}
		Expect(k8sClient.Create(ctx, secret)).Should(Succeed())

		kc := &keycloakApi.Keycloak{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "kc-pw-grant-configmap",
				Namespace: testNamespace,
			},
			Spec: keycloakApi.KeycloakSpec{
				Url: keycloakURL,
				Auth: &common.AuthSpec{
					PasswordGrant: &common.PasswordGrantConfig{
						Username: common.SourceRefOrVal{
							SourceRef: common.SourceRef{
								ConfigMapKeyRef: &common.ConfigMapKeySelector{
									LocalObjectReference: corev1.LocalObjectReference{
										Name: "pw-grant-configmap",
									},
									Key: "username",
								},
							},
						},
						PasswordRef: common.SecretKeySelector{
							LocalObjectReference: corev1.LocalObjectReference{
								Name: "pw-grant-configmap-secret",
							},
							Key: "password",
						},
					},
				},
			},
		}
		Expect(k8sClient.Create(ctx, kc)).Should(Succeed())
		assertKeycloakConnected("kc-pw-grant-configmap")
	})

	It("Should connect with client credentials using direct client ID value", func() {
		clientID := "test-sa-client"
		clientSecretVal := "test-sa-client-secret"

		cleanup, err := provisionServiceAccountClient(ctx, keycloakURL, clientID, clientSecretVal)
		Expect(err).ShouldNot(HaveOccurred())
		DeferCleanup(func() { cleanup() })

		secret := &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "client-creds-secret",
				Namespace: testNamespace,
			},
			Data: map[string][]byte{
				"clientSecret": []byte(clientSecretVal),
			},
		}
		Expect(k8sClient.Create(ctx, secret)).Should(Succeed())

		kc := &keycloakApi.Keycloak{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "kc-client-creds",
				Namespace: testNamespace,
			},
			Spec: keycloakApi.KeycloakSpec{
				Url: keycloakURL,
				Auth: &common.AuthSpec{
					ClientCredentials: &common.ClientCredentialsConfig{
						ClientID: common.SourceRefOrVal{
							Value: clientID,
						},
						ClientSecretRef: common.SecretKeySelector{
							LocalObjectReference: corev1.LocalObjectReference{
								Name: "client-creds-secret",
							},
							Key: "clientSecret",
						},
					},
				},
			},
		}
		Expect(k8sClient.Create(ctx, kc)).Should(Succeed())
		assertKeycloakConnected("kc-client-creds")
	})

	It("Should connect with client credentials using clientId from SecretKeyRef", func() {
		clientID := "test-sa-client-secretref"
		clientSecretVal := "test-sa-client-secretref-secret"

		cleanup, err := provisionServiceAccountClient(ctx, keycloakURL, clientID, clientSecretVal)
		Expect(err).ShouldNot(HaveOccurred())
		DeferCleanup(func() { cleanup() })

		secret := &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "client-creds-secretref-secret",
				Namespace: testNamespace,
			},
			Data: map[string][]byte{
				"clientId":     []byte(clientID),
				"clientSecret": []byte(clientSecretVal),
			},
		}
		Expect(k8sClient.Create(ctx, secret)).Should(Succeed())

		kc := &keycloakApi.Keycloak{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "kc-client-creds-secretref",
				Namespace: testNamespace,
			},
			Spec: keycloakApi.KeycloakSpec{
				Url: keycloakURL,
				Auth: &common.AuthSpec{
					ClientCredentials: &common.ClientCredentialsConfig{
						ClientID: common.SourceRefOrVal{
							SourceRef: common.SourceRef{
								SecretKeyRef: &common.SecretKeySelector{
									LocalObjectReference: corev1.LocalObjectReference{
										Name: "client-creds-secretref-secret",
									},
									Key: "clientId",
								},
							},
						},
						ClientSecretRef: common.SecretKeySelector{
							LocalObjectReference: corev1.LocalObjectReference{
								Name: "client-creds-secretref-secret",
							},
							Key: "clientSecret",
						},
					},
				},
			},
		}
		Expect(k8sClient.Create(ctx, kc)).Should(Succeed())
		assertKeycloakConnected("kc-client-creds-secretref")
	})
})

func provisionServiceAccountClient(ctx context.Context, keycloakURL, clientID, clientSecret string) (cleanup func(), err error) {
	adminKC, err := keycloakv2.NewKeycloakClient(
		ctx,
		keycloakURL,
		keycloakv2.DefaultAdminClientID,
		keycloakv2.WithPasswordGrant(keycloakv2.DefaultAdminUsername, keycloakv2.DefaultAdminPassword),
	)
	if err != nil {
		return nil, err
	}

	saEnabled := true
	publicClient := false
	enabled := true
	protocol := "openid-connect"

	_, err = adminKC.Clients.CreateClient(ctx, keycloakv2.MasterRealm, keycloakv2.ClientRepresentation{
		ClientId:               &clientID,
		Secret:                 &clientSecret,
		ServiceAccountsEnabled: &saEnabled,
		PublicClient:           &publicClient,
		Enabled:                &enabled,
		Protocol:               &protocol,
	})
	if err != nil {
		return nil, err
	}

	clientUUID, err := adminKC.Clients.GetClientUUID(ctx, keycloakv2.MasterRealm, clientID)
	if err != nil {
		return nil, err
	}

	saUser, _, err := adminKC.Clients.GetServiceAccountUser(ctx, keycloakv2.MasterRealm, clientUUID)
	if err != nil {
		return nil, err
	}

	adminRole, _, err := adminKC.Roles.GetRealmRole(ctx, keycloakv2.MasterRealm, "admin")
	if err != nil {
		return nil, err
	}

	_, err = adminKC.Users.AddUserRealmRoles(ctx, keycloakv2.MasterRealm, *saUser.Id, []keycloakv2.RoleRepresentation{*adminRole})
	if err != nil {
		return nil, err
	}

	cleanup = func() {
		_, _ = adminKC.Clients.DeleteClient(ctx, keycloakv2.MasterRealm, clientUUID)
	}

	return cleanup, nil
}

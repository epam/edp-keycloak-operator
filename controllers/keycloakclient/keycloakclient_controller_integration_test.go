package keycloakclient

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	corev1 "k8s.io/api/core/v1"
	k8sErrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	"github.com/epam/edp-keycloak-operator/api/common"
	keycloakApi "github.com/epam/edp-keycloak-operator/api/v1"
	"github.com/epam/edp-keycloak-operator/controllers/helper"
	"github.com/epam/edp-keycloak-operator/pkg/secretref"
)

var _ = Describe("KeycloakClient controller", func() {
	It("Should create KeycloakClient with secret reference", func() {
		By("By creating a client secret")
		clientSecret := &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-keycloak-client-secret",
				Namespace: ns,
			},
			Data: map[string][]byte{
				"secretKey": []byte("secretValue"),
			},
		}
		Expect(k8sClient.Create(ctx, clientSecret)).Should(Succeed())
		By("By creating a KeycloakClient")
		keycloakClient := &keycloakApi.KeycloakClient{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-keycloak-client-with-secret-ref",
				Namespace: ns,
			},
			Spec: keycloakApi.KeycloakClientSpec{
				ClientId: "test-keycloak-client-with-secret-ref",
				RealmRef: common.RealmRef{
					Name: KeycloakRealmCR,
					Kind: keycloakApi.KeycloakRealmKind,
				},
				Secret: secretref.GenerateSecretRef(clientSecret.Name, "secretKey"),
				Public: true,
				WebUrl: "https://test-keycloak-client-with-secret-ref",
				Attributes: map[string]string{
					"post.logout.redirect.uris": "+",
				},
				DirectAccess:                 false,
				AdvancedProtocolMappers:      false,
				ClientRoles:                  []string{"administrator", "developer"},
				FrontChannelLogout:           false,
				RedirectUris:                 []string{"https://test-keycloak-client-with-secret-ref"},
				WebOrigins:                   []string{"https://test-keycloak-client-with-secret-ref"},
				ImplicitFlowEnabled:          false,
				AuthorizationServicesEnabled: false,
				BearerOnly:                   false,
				ConsentRequired:              false,
				Description:                  "test description",
				Enabled:                      true,
				FullScopeAllowed:             true,
				Name:                         "test name",
				StandardFlowEnabled:          true,
				SurrogateAuthRequired:        false,
				ClientAuthenticatorType:      "client-secret",
			},
		}
		Expect(k8sClient.Create(ctx, keycloakClient)).Should(Succeed())
		Eventually(func() bool {
			createdKeycloakClient := &keycloakApi.KeycloakClient{}
			err := k8sClient.Get(ctx, types.NamespacedName{Name: keycloakClient.Name, Namespace: ns}, createdKeycloakClient)
			if err != nil {
				return false
			}

			return createdKeycloakClient.Status.Value == helper.StatusOK &&
				createdKeycloakClient.Status.ClientID != ""
		}, timeout, interval).Should(BeTrue(), "KeycloakClient should be created successfully")
	})
	It("Should delete KeycloakClient", func() {
		By("By getting KeycloakClient")
		keycloakClient := &keycloakApi.KeycloakClient{}
		Expect(k8sClient.Get(ctx, types.NamespacedName{Namespace: ns, Name: "test-keycloak-client-with-secret-ref"}, keycloakClient)).Should(Succeed())
		By("By deleting KeycloakClient")
		Expect(k8sClient.Delete(ctx, keycloakClient)).Should(Succeed())
		Eventually(func() bool {
			deletedKeycloakClient := &keycloakApi.KeycloakClient{}
			err := k8sClient.Get(ctx, types.NamespacedName{Name: keycloakClient.Name, Namespace: ns}, deletedKeycloakClient)

			return k8sErrors.IsNotFound(err)
		}, timeout, interval).Should(BeTrue(), "KeycloakClient should be deleted")
	})
	It("Should create KeycloakClient with empty secret", func() {
		By("By creating a KeycloakClient")
		keycloakClient := &keycloakApi.KeycloakClient{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-keycloak-client-with-empty-secret",
				Namespace: ns,
			},
			Spec: keycloakApi.KeycloakClientSpec{
				ClientId: "test-keycloak-client-with-empty-secret",
				RealmRef: common.RealmRef{
					Name: KeycloakRealmCR,
					Kind: keycloakApi.KeycloakRealmKind,
				},
			},
		}
		Expect(k8sClient.Create(ctx, keycloakClient)).Should(Succeed())
		Eventually(func() bool {
			createdKeycloakClient := &keycloakApi.KeycloakClient{}
			err := k8sClient.Get(ctx, types.NamespacedName{Name: keycloakClient.Name, Namespace: ns}, createdKeycloakClient)
			if err != nil {
				return false
			}

			return createdKeycloakClient.Status.Value == helper.StatusOK &&
				createdKeycloakClient.Status.ClientID != ""
		}, timeout, interval).Should(BeTrue(), "KeycloakClient should be created successfully")
	})
	It("Should create KeycloakClient with direct secret name", func() {
		By("By creating a client secret")
		clientSecret := &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-keycloak-client-secret2",
				Namespace: ns,
			},
			Data: map[string][]byte{
				keycloakApi.ClientSecretKey: []byte("secretValue"),
			},
		}
		Expect(k8sClient.Create(ctx, clientSecret)).Should(Succeed())
		By("By creating a KeycloakClient")
		keycloakClient := &keycloakApi.KeycloakClient{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-keycloak-client-with-direct-secret-name",
				Namespace: ns,
			},
			Spec: keycloakApi.KeycloakClientSpec{
				ClientId: "test-keycloak-client-with-direct-secret-name",
				RealmRef: common.RealmRef{
					Name: KeycloakRealmCR,
					Kind: keycloakApi.KeycloakRealmKind,
				},
				Secret: clientSecret.Name,
			},
		}
		Expect(k8sClient.Create(ctx, keycloakClient)).Should(Succeed())
		Eventually(func() bool {
			createdKeycloakClient := &keycloakApi.KeycloakClient{}
			err := k8sClient.Get(ctx, types.NamespacedName{Name: keycloakClient.Name, Namespace: ns}, createdKeycloakClient)
			if err != nil {
				return false
			}

			return createdKeycloakClient.Status.Value == helper.StatusOK &&
				createdKeycloakClient.Status.ClientID != ""
		}, timeout, interval).Should(BeTrue(), "KeycloakClient should be created successfully")
	})
})

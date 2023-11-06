package keycloakrealmidentityprovider

import (
	"fmt"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	corev1 "k8s.io/api/core/v1"
	k8sErrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	"github.com/epam/edp-keycloak-operator/api/common"
	keycloakApi "github.com/epam/edp-keycloak-operator/api/v1"
	"github.com/epam/edp-keycloak-operator/controllers/helper"
)

var _ = Describe("KeycloakRealmIdentityProvider controller", func() {
	const (
		identityProviderCR = "test-keycloak-realm-identity-provider"
	)
	It("Should create KeycloakRealmIdentityProvider", func() {
		By("By creating a client secret")
		clientSecret := &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-keycloak-realm-client-secret",
				Namespace: ns,
			},
			Data: map[string][]byte{
				"secretKey": []byte("secretValue"),
			},
		}
		Expect(k8sClient.Create(ctx, clientSecret)).Should(Succeed())
		By("By creating a KeycloakRealmIdentityProvider")
		provider := &keycloakApi.KeycloakRealmIdentityProvider{
			ObjectMeta: metav1.ObjectMeta{
				Name:      identityProviderCR,
				Namespace: ns,
			},
			Spec: keycloakApi.KeycloakRealmIdentityProviderSpec{
				RealmRef: common.RealmRef{
					Name: KeycloakRealmCR,
					Kind: keycloakApi.KeycloakRealmKind,
				},
				ProviderID: "instagram",
				Alias:      "new-provider",
				Config: map[string]string{
					"clientId":     "provider-client",
					"clientSecret": fmt.Sprintf("$%s:%s", clientSecret.Name, "secretKey"),
				},
				Enabled:                   true,
				DisplayName:               "New provider",
				FirstBrokerLoginFlowAlias: "first broker login",
			},
		}
		Expect(k8sClient.Create(ctx, provider)).Should(Succeed())
		Eventually(func() bool {
			createdProvider := &keycloakApi.KeycloakRealmIdentityProvider{}
			err := k8sClient.Get(ctx, types.NamespacedName{Name: provider.Name, Namespace: ns}, createdProvider)
			if err != nil {
				return false
			}

			return createdProvider.Status.Value == helper.StatusOK
		}, timeout, interval).Should(BeTrue())
	})
	It("Should delete KeycloakRealmIdentityProvider", func() {
		By("By getting KeycloakRealmIdentityProvider")
		provider := &keycloakApi.KeycloakRealmIdentityProvider{}
		Expect(k8sClient.Get(ctx, types.NamespacedName{Namespace: ns, Name: identityProviderCR}, provider)).Should(Succeed())
		By("By deleting KeycloakRealmIdentityProvider")
		Expect(k8sClient.Delete(ctx, provider)).Should(Succeed())
		Eventually(func() bool {
			deletedProvider := &keycloakApi.KeycloakRealmIdentityProvider{}
			err := k8sClient.Get(ctx, types.NamespacedName{Name: provider.Name, Namespace: ns}, deletedProvider)

			return k8sErrors.IsNotFound(err)
		}, timeout, interval).Should(BeTrue(), "KeycloakRealmIdentityProvider should be deleted")
	})
})

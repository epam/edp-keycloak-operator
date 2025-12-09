package v1

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/epam/edp-keycloak-operator/api/common"
	keycloakApi "github.com/epam/edp-keycloak-operator/api/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var _ = Describe("KeycloakRealm Webhook", func() {
	var objInNs1 *keycloakApi.KeycloakRealm
	BeforeEach(func() {
		objInNs1 = &keycloakApi.KeycloakRealm{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-realm",
				Namespace: "ns1",
			},
			Spec: keycloakApi.KeycloakRealmSpec{
				RealmName: "test-realm",
				KeycloakRef: common.KeycloakRef{
					Name: "test-keycloak",
				},
			},
		}

		Expect(k8sClient.Create(ctx, objInNs1)).Should(Succeed(), "failed to create initial KeycloakRealm in ns1")
	})
	AfterEach(func() {
		Expect(client.IgnoreNotFound(k8sClient.Delete(ctx, objInNs1))).Should(Succeed(), "failed to delete initial KeycloakRealm in ns1")
	})

	Context("When creating KeycloakRealm", func() {
		It("Should deny creation with the same RealmName in the same namespace", func() {
			duplicateObj := &keycloakApi.KeycloakRealm{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "duplicate-realm",
					Namespace: "ns1",
				},
				Spec: keycloakApi.KeycloakRealmSpec{
					RealmName: "test-realm", // Same RealmName as the existing one in ns1
					KeycloakRef: common.KeycloakRef{
						Name: "test-keycloak",
					},
				},
			}

			err := k8sClient.Create(ctx, duplicateObj)
			Expect(err).Should(HaveOccurred(), "expected error when creating KeycloakRealm with duplicate RealmName in the same namespace")
		})

		It("Should deny creation with the same RealmName in a different namespace", func() {
			objInNs2 := &keycloakApi.KeycloakRealm{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-realm-ns2",
					Namespace: "ns2",
				},
				Spec: keycloakApi.KeycloakRealmSpec{
					RealmName: "test-realm", // Same RealmName as the existing one but in a different namespace
					KeycloakRef: common.KeycloakRef{
						Name: "test-keycloak",
					},
				},
			}

			err := k8sClient.Create(ctx, objInNs2)
			Expect(err).Should(HaveOccurred(), "failed to create KeycloakRealm with same RealmName in different namespace")
		})
	})
})

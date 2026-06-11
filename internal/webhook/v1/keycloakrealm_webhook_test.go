package v1

import (
	"context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	"github.com/epam/edp-keycloak-operator/api/common"
	keycloakApi "github.com/epam/edp-keycloak-operator/api/v1"
	keycloakAlpha "github.com/epam/edp-keycloak-operator/api/v1alpha1"
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
					Kind: keycloakApi.KeycloakKind,
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
		It("Should deny creation with the same RealmName for the same Keycloak in the same namespace", func() {
			duplicateObj := &keycloakApi.KeycloakRealm{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "duplicate-realm",
					Namespace: "ns1",
				},
				Spec: keycloakApi.KeycloakRealmSpec{
					RealmName: "test-realm", // Same RealmName as the existing one in ns1
					KeycloakRef: common.KeycloakRef{
						Kind: keycloakApi.KeycloakKind,
						Name: "test-keycloak",
					},
				},
			}

			err := k8sClient.Create(ctx, duplicateObj)
			Expect(err).Should(HaveOccurred(), "expected error when creating KeycloakRealm with duplicate RealmName in the same namespace")
			Expect(err.Error()).To(ContainSubstring(`realm name "test-realm" is already used by KeycloakRealm ns1/test-realm targeting Keycloak/ns1/test-keycloak`))
		})

		It("Should allow creation with the same RealmName for a different Keycloak in another namespace", func() {
			objInNs2 := &keycloakApi.KeycloakRealm{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-realm-ns2",
					Namespace: "ns2",
				},
				Spec: keycloakApi.KeycloakRealmSpec{
					RealmName: "test-realm", // Same RealmName as the existing one but in a different namespace
					KeycloakRef: common.KeycloakRef{
						Kind: keycloakApi.KeycloakKind,
						Name: "test-keycloak-2",
					},
				},
			}

			err := k8sClient.Create(ctx, objInNs2)
			Expect(err).ShouldNot(HaveOccurred(), "unexpected error when creating KeycloakRealm with same RealmName for a different Keycloak")
			Expect(client.IgnoreNotFound(k8sClient.Delete(ctx, objInNs2))).Should(Succeed(), "failed to delete KeycloakRealm in ns2")
		})

		It("Should allow creation with the same RealmName and same Keycloak name in another namespace (namespaced kind)", func() {
			// The namespaced Keycloak kind resolves in the realm's own namespace, so
			// "test-keycloak" in ns2 is a different instance than "test-keycloak" in ns1.
			objInNs2 := &keycloakApi.KeycloakRealm{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-realm-ns2",
					Namespace: "ns2",
				},
				Spec: keycloakApi.KeycloakRealmSpec{
					RealmName: "test-realm",
					KeycloakRef: common.KeycloakRef{
						Kind: keycloakApi.KeycloakKind,
						Name: "test-keycloak", // Same KeycloakRef name, but resolved in ns2
					},
				},
			}

			err := k8sClient.Create(ctx, objInNs2)
			Expect(err).ShouldNot(HaveOccurred(), "unexpected error when creating KeycloakRealm referencing a namespaced Keycloak with the same name in a different namespace")
			Expect(client.IgnoreNotFound(k8sClient.Delete(ctx, objInNs2))).Should(Succeed(), "failed to delete KeycloakRealm in ns2")
		})

		It("Should deny creation with the same RealmName for the same ClusterKeycloak across namespaces", func() {
			// ClusterKeycloak is cluster-scoped, so the same ref name across namespaces
			// targets the same instance and must conflict.
			clusterRealmNs1 := &keycloakApi.KeycloakRealm{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "cluster-realm-ns1",
					Namespace: "ns1",
				},
				Spec: keycloakApi.KeycloakRealmSpec{
					RealmName: "shared-realm",
					KeycloakRef: common.KeycloakRef{
						Kind: keycloakAlpha.ClusterKeycloakKind,
						Name: "shared-keycloak",
					},
				},
			}
			Expect(k8sClient.Create(ctx, clusterRealmNs1)).Should(Succeed(), "failed to create cluster-scoped KeycloakRealm in ns1")
			defer func() {
				Expect(client.IgnoreNotFound(k8sClient.Delete(ctx, clusterRealmNs1))).Should(Succeed())
			}()

			clusterRealmNs2 := &keycloakApi.KeycloakRealm{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "cluster-realm-ns2",
					Namespace: "ns2",
				},
				Spec: keycloakApi.KeycloakRealmSpec{
					RealmName: "shared-realm",
					KeycloakRef: common.KeycloakRef{
						Kind: keycloakAlpha.ClusterKeycloakKind,
						Name: "shared-keycloak",
					},
				},
			}

			err := k8sClient.Create(ctx, clusterRealmNs2)
			Expect(err).Should(HaveOccurred(), "expected error when creating KeycloakRealm with same RealmName for the same ClusterKeycloak in a different namespace")
			Expect(err.Error()).To(ContainSubstring(`realm name "shared-realm" is already used by KeycloakRealm ns1/cluster-realm-ns1 targeting ClusterKeycloak/shared-keycloak`))
		})
	})

	Context("Internationalization duplicate fields (admission warning)", func() {
		It("returns a warning from ValidateCreate when both themes and localization toggles are set", func() {
			v := NewKeycloakRealmCustomValidator(fake.NewClientBuilder().Build())
			t := true
			realm := &keycloakApi.KeycloakRealm{
				Spec: keycloakApi.KeycloakRealmSpec{
					Themes: &keycloakApi.RealmThemes{
						InternationalizationEnabled: &t,
					},
					Localization: &keycloakApi.RealmLocalization{
						InternationalizationEnabled: &t,
					},
				},
			}
			warnings, err := v.ValidateCreate(context.Background(), realm)
			Expect(err).NotTo(HaveOccurred())
			Expect(warnings).To(HaveLen(1))
			Expect(warnings[0]).To(ContainSubstring("deprecated"))
			Expect(warnings[0]).To(ContainSubstring("spec.localization wins"))
		})

		It("returns no warning when only localization internationalizationEnabled is set", func() {
			v := NewKeycloakRealmCustomValidator(fake.NewClientBuilder().Build())
			t := true
			realm := &keycloakApi.KeycloakRealm{
				Spec: keycloakApi.KeycloakRealmSpec{
					Themes: &keycloakApi.RealmThemes{},
					Localization: &keycloakApi.RealmLocalization{
						InternationalizationEnabled: &t,
					},
				},
			}
			warnings, err := v.ValidateCreate(context.Background(), realm)
			Expect(err).NotTo(HaveOccurred())
			Expect(warnings).To(BeEmpty())
		})

		It("returns a warning from ValidateUpdate when both toggles are set on the new object", func() {
			v := NewKeycloakRealmCustomValidator(fake.NewClientBuilder().Build())
			t := true
			oldRealm := &keycloakApi.KeycloakRealm{}
			newRealm := &keycloakApi.KeycloakRealm{
				Spec: keycloakApi.KeycloakRealmSpec{
					Themes: &keycloakApi.RealmThemes{
						InternationalizationEnabled: &t,
					},
					Localization: &keycloakApi.RealmLocalization{
						InternationalizationEnabled: &t,
					},
				},
			}
			warnings, err := v.ValidateUpdate(context.Background(), oldRealm, newRealm)
			Expect(err).NotTo(HaveOccurred())
			Expect(warnings).To(HaveLen(1))
		})
	})
})

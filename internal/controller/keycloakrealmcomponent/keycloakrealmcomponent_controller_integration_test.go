package keycloakrealmcomponent

import (
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	corev1 "k8s.io/api/core/v1"
	k8sErrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	"github.com/epam/edp-keycloak-operator/api/common"
	keycloakApi "github.com/epam/edp-keycloak-operator/api/v1"
	"github.com/epam/edp-keycloak-operator/pkg/client/keycloakapi"
	"github.com/epam/edp-keycloak-operator/pkg/objectmeta"
)

// keyProviderID / keyProviderType are used for tests that verify Config persistence.
// The rsa-generated key provider actually stores its config (priority, keySize) in Keycloak,
// unlike the scope/ClientRegistrationPolicy provider which silently drops unknown keys.
const (
	keyProviderID   = "rsa-generated"
	keyProviderType = "org.keycloakapi.keys.KeyProvider"

	scopeProviderID   = "scope"
	scopeProviderType = "org.keycloakapi.services.clientregistration.policy.ClientRegistrationPolicy"
)

var _ = Describe("KeycloakRealmComponent controller", func() {
	const (
		componentCR      = "test-keycloak-realm-component"
		childComponentCR = "test-keycloak-realm-component-child"
	)

	It("Should create KeycloakRealmComponents", func() {
		By("By creating a KeycloakRealmComponent")
		component := &keycloakApi.KeycloakRealmComponent{
			ObjectMeta: metav1.ObjectMeta{
				Name:      componentCR,
				Namespace: ns,
			},
			Spec: keycloakApi.KeycloakComponentSpec{
				Name:         "test-keycloak-realm-component",
				ProviderID:   scopeProviderID,
				ProviderType: scopeProviderType,
				RealmRef: common.RealmRef{
					Kind: keycloakApi.KeycloakRealmKind,
					Name: KeycloakRealmCR,
				},
			},
		}
		Expect(k8sClient.Create(ctx, component)).Should(Succeed())
		Eventually(func(g Gomega) {
			cr := &keycloakApi.KeycloakRealmComponent{}
			g.Expect(k8sClient.Get(ctx, types.NamespacedName{Name: component.Name, Namespace: ns}, cr)).Should(Succeed())
			g.Expect(cr.Status.Value).Should(Equal(common.StatusOK))
			g.Expect(cr.Status.ID).ShouldNot(BeEmpty())
		}, timeout, interval).Should(Succeed())

		By("By verifying Name, ProviderId and ProviderType in Keycloak")
		Eventually(func(g Gomega) {
			cr := &keycloakApi.KeycloakRealmComponent{}
			g.Expect(k8sClient.Get(ctx, types.NamespacedName{Name: component.Name, Namespace: ns}, cr)).Should(Succeed())

			kcComp, _, err := keycloakApiClient.RealmComponents.GetComponent(ctx, KeycloakRealmCR, cr.Status.ID)
			g.Expect(err).ShouldNot(HaveOccurred())
			g.Expect(kcComp).ShouldNot(BeNil())
			g.Expect(kcComp.Name).ShouldNot(BeNil())
			g.Expect(*kcComp.Name).Should(Equal(component.Spec.Name))
			g.Expect(kcComp.ProviderId).ShouldNot(BeNil())
			g.Expect(*kcComp.ProviderId).Should(Equal(component.Spec.ProviderID))
			g.Expect(kcComp.ProviderType).ShouldNot(BeNil())
			g.Expect(*kcComp.ProviderType).Should(Equal(component.Spec.ProviderType))
		}, timeout, interval).Should(Succeed())

		By("By creating a child KeycloakRealmComponent with ParentRef.Kind=KeycloakRealmComponent")
		childComponent := &keycloakApi.KeycloakRealmComponent{
			ObjectMeta: metav1.ObjectMeta{
				Name:      childComponentCR,
				Namespace: ns,
			},
			Spec: keycloakApi.KeycloakComponentSpec{
				Name:         "test-keycloak-realm-component-child",
				ProviderID:   scopeProviderID,
				ProviderType: scopeProviderType,
				RealmRef: common.RealmRef{
					Kind: keycloakApi.KeycloakRealmKind,
					Name: KeycloakRealmCR,
				},
				ParentRef: &keycloakApi.ParentComponent{
					Kind: keycloakApi.KeycloakRealmComponentKind,
					Name: component.Name,
				},
			},
		}
		Expect(k8sClient.Create(ctx, childComponent)).Should(Succeed())
		Eventually(func(g Gomega) {
			cr := &keycloakApi.KeycloakRealmComponent{}
			g.Expect(k8sClient.Get(ctx, types.NamespacedName{Name: childComponent.Name, Namespace: ns}, cr)).Should(Succeed())
			g.Expect(cr.Status.Value).Should(Equal(common.StatusOK))
			g.Expect(cr.Status.ID).ShouldNot(BeEmpty())
		}, timeout, interval).Should(Succeed())

		By("By verifying child component's parentId equals parent component's status.ID in Keycloak")
		Eventually(func(g Gomega) {
			parentCR := &keycloakApi.KeycloakRealmComponent{}
			g.Expect(k8sClient.Get(ctx, types.NamespacedName{Name: component.Name, Namespace: ns}, parentCR)).Should(Succeed())

			childCR := &keycloakApi.KeycloakRealmComponent{}
			g.Expect(k8sClient.Get(ctx, types.NamespacedName{Name: childComponent.Name, Namespace: ns}, childCR)).Should(Succeed())

			kcChild, _, err := keycloakApiClient.RealmComponents.GetComponent(ctx, KeycloakRealmCR, childCR.Status.ID)
			g.Expect(err).ShouldNot(HaveOccurred())
			g.Expect(kcChild).ShouldNot(BeNil())
			g.Expect(kcChild.ParentId).ShouldNot(BeNil())
			g.Expect(*kcChild.ParentId).Should(Equal(parentCR.Status.ID))
		}, timeout, interval).Should(Succeed())

		By("By creating a KeycloakRealmComponent with ParentRef.Kind=KeycloakRealm")
		componentWithParentRealm := &keycloakApi.KeycloakRealmComponent{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-keycloak-realm-component-with-parent-realm",
				Namespace: ns,
			},
			Spec: keycloakApi.KeycloakComponentSpec{
				Name:         "test-keycloak-realm-component-with-parent-realm",
				ProviderID:   scopeProviderID,
				ProviderType: scopeProviderType,
				RealmRef: common.RealmRef{
					Kind: keycloakApi.KeycloakRealmKind,
					Name: KeycloakRealmCR,
				},
				ParentRef: &keycloakApi.ParentComponent{
					Kind: keycloakApi.KeycloakRealmKind,
					Name: KeycloakRealmCR,
				},
			},
		}
		Expect(k8sClient.Create(ctx, componentWithParentRealm)).Should(Succeed())
		Eventually(func(g Gomega) {
			cr := &keycloakApi.KeycloakRealmComponent{}
			g.Expect(k8sClient.Get(ctx, types.NamespacedName{Name: componentWithParentRealm.Name, Namespace: ns}, cr)).Should(Succeed())
			g.Expect(cr.Status.Value).Should(Equal(common.StatusOK))
			g.Expect(cr.Status.ID).ShouldNot(BeEmpty())
		}, timeout, interval).Should(Succeed())

		By("By verifying the component with parent realm has a non-empty parentId in Keycloak")
		Eventually(func(g Gomega) {
			cr := &keycloakApi.KeycloakRealmComponent{}
			g.Expect(k8sClient.Get(ctx, types.NamespacedName{Name: componentWithParentRealm.Name, Namespace: ns}, cr)).Should(Succeed())

			kcComp, _, err := keycloakApiClient.RealmComponents.GetComponent(ctx, KeycloakRealmCR, cr.Status.ID)
			g.Expect(err).ShouldNot(HaveOccurred())
			g.Expect(kcComp).ShouldNot(BeNil())
			g.Expect(kcComp.ParentId).ShouldNot(BeNil())
			g.Expect(*kcComp.ParentId).ShouldNot(BeEmpty())
		}, timeout, interval).Should(Succeed())
	})

	It("Should delete KeycloakRealmComponents and verify removal from Keycloak", func() {
		By("By getting the parent KeycloakRealmComponent")
		component := &keycloakApi.KeycloakRealmComponent{}
		Expect(k8sClient.Get(ctx, types.NamespacedName{Namespace: ns, Name: componentCR}, component)).Should(Succeed())
		parentID := component.Status.ID

		By("By deleting the parent KeycloakRealmComponent")
		Expect(k8sClient.Delete(ctx, component)).Should(Succeed())
		Eventually(func() bool {
			err := k8sClient.Get(ctx, types.NamespacedName{Name: component.Name, Namespace: ns}, &keycloakApi.KeycloakRealmComponent{})
			return k8sErrors.IsNotFound(err)
		}, timeout, interval).Should(BeTrue(), "parent KeycloakRealmComponent should be deleted from k8s")

		By("By verifying the parent component is removed from Keycloak")
		Eventually(func(g Gomega) {
			_, resp, err := keycloakApiClient.RealmComponents.GetComponent(ctx, KeycloakRealmCR, parentID)
			g.Expect(err).Should(HaveOccurred())
			g.Expect(resp).ShouldNot(BeNil())
			g.Expect(resp.HTTPResponse.StatusCode).Should(Equal(404))
		}, timeout, interval).Should(Succeed())

		By("By getting the child KeycloakRealmComponent")
		childComponent := &keycloakApi.KeycloakRealmComponent{}
		Expect(k8sClient.Get(ctx, types.NamespacedName{Namespace: ns, Name: childComponentCR}, childComponent)).Should(Succeed())
		childID := childComponent.Status.ID

		By("By deleting the child KeycloakRealmComponent")
		Expect(k8sClient.Delete(ctx, childComponent)).Should(Succeed())
		Eventually(func() bool {
			err := k8sClient.Get(ctx, types.NamespacedName{Name: childComponent.Name, Namespace: ns}, &keycloakApi.KeycloakRealmComponent{})
			return k8sErrors.IsNotFound(err)
		}, timeout, interval).Should(BeTrue(), "child KeycloakRealmComponent should be deleted from k8s")

		By("By verifying the child component is removed from Keycloak")
		Eventually(func(g Gomega) {
			_, resp, err := keycloakApiClient.RealmComponents.GetComponent(ctx, KeycloakRealmCR, childID)
			g.Expect(err).Should(HaveOccurred())
			g.Expect(resp).ShouldNot(BeNil())
			g.Expect(resp.HTTPResponse.StatusCode).Should(Equal(404))
		}, timeout, interval).Should(Succeed())
	})

	It("Should fail with invalid realm", func() {
		const (
			timeout  = time.Second * 5
			interval = time.Second
		)
		By("By creating a KeycloakRealmComponent with a non-existent realm")
		component := &keycloakApi.KeycloakRealmComponent{
			ObjectMeta: metav1.ObjectMeta{
				Name:      componentCR,
				Namespace: ns,
			},
			Spec: keycloakApi.KeycloakComponentSpec{
				Name:         "test-keycloak-realm-component",
				ProviderID:   scopeProviderID,
				ProviderType: scopeProviderType,
				RealmRef: common.RealmRef{
					Kind: keycloakApi.KeycloakRealmKind,
					Name: "invalid-realm",
				},
			},
		}
		Expect(k8sClient.Create(ctx, component)).Should(Succeed())
		Consistently(func() bool {
			cr := &keycloakApi.KeycloakRealmComponent{}
			err := k8sClient.Get(ctx, types.NamespacedName{Name: component.Name, Namespace: ns}, cr)
			if err != nil {
				return false
			}

			return cr.Status.Value != common.StatusOK
		}, timeout, interval).Should(BeTrue(), "KeycloakRealmComponent should remain in failed state")
	})

	It("Should fail with invalid parent component", func() {
		const (
			timeout  = time.Second * 5
			interval = time.Second
		)
		By("By creating a KeycloakRealmComponent with a non-existent parent component")
		component := &keycloakApi.KeycloakRealmComponent{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-keycloak-realm-component-invalid-parent-component",
				Namespace: ns,
			},
			Spec: keycloakApi.KeycloakComponentSpec{
				Name:         "test-keycloak-realm-component-invalid-parent-component",
				ProviderID:   scopeProviderID,
				ProviderType: scopeProviderType,
				RealmRef: common.RealmRef{
					Kind: keycloakApi.KeycloakRealmKind,
					Name: KeycloakRealmCR,
				},
				ParentRef: &keycloakApi.ParentComponent{
					Kind: keycloakApi.KeycloakRealmComponentKind,
					Name: "invalid-parent-component",
				},
			},
		}
		Expect(k8sClient.Create(ctx, component)).Should(Succeed())
		Consistently(func() bool {
			cr := &keycloakApi.KeycloakRealmComponent{}
			err := k8sClient.Get(ctx, types.NamespacedName{Name: component.Name, Namespace: ns}, cr)
			if err != nil {
				return false
			}

			return cr.Status.Value != common.StatusOK
		}, timeout, interval).Should(BeTrue(), "KeycloakRealmComponent with invalid parent component should remain in failed state")
	})

	It("Should fail with invalid parent realm", func() {
		const (
			timeout  = time.Second * 5
			interval = time.Second
		)
		By("By creating a KeycloakRealmComponent with a non-existent parent realm")
		component := &keycloakApi.KeycloakRealmComponent{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-keycloak-realm-component-invalid-parent-realm",
				Namespace: ns,
			},
			Spec: keycloakApi.KeycloakComponentSpec{
				Name:         "test-keycloak-realm-component-invalid-parent-realm",
				ProviderID:   scopeProviderID,
				ProviderType: scopeProviderType,
				RealmRef: common.RealmRef{
					Kind: keycloakApi.KeycloakRealmKind,
					Name: KeycloakRealmCR,
				},
				ParentRef: &keycloakApi.ParentComponent{
					Kind: keycloakApi.KeycloakRealmKind,
					Name: "invalid-parent-realm",
				},
			},
		}
		Expect(k8sClient.Create(ctx, component)).Should(Succeed())
		Consistently(func() bool {
			cr := &keycloakApi.KeycloakRealmComponent{}
			err := k8sClient.Get(ctx, types.NamespacedName{Name: component.Name, Namespace: ns}, cr)
			if err != nil {
				return false
			}

			return cr.Status.Value != common.StatusOK
		}, time.Second*3, interval).Should(BeTrue(), "KeycloakRealmComponent with invalid parent realm should remain in failed state")
	})

	It("Should skip Keycloak resource removal when preserveResourcesOnDeletion is set", func() {
		By("By creating a KeycloakRealmComponent with preserveResourcesOnDeletion annotation")
		component := &keycloakApi.KeycloakRealmComponent{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "component-with-preserve-resources-on-deletion",
				Namespace: ns,
				Annotations: map[string]string{
					objectmeta.PreserveResourcesOnDeletionAnnotation: "true",
				},
			},
			Spec: keycloakApi.KeycloakComponentSpec{
				Name:         "component-with-preserve-resources-on-deletion",
				ProviderID:   scopeProviderID,
				ProviderType: scopeProviderType,
				RealmRef: common.RealmRef{
					Kind: keycloakApi.KeycloakRealmKind,
					Name: KeycloakRealmCR,
				},
			},
		}
		Expect(k8sClient.Create(ctx, component)).Should(Succeed())
		Eventually(func(g Gomega) {
			cr := &keycloakApi.KeycloakRealmComponent{}
			g.Expect(k8sClient.Get(ctx, types.NamespacedName{Name: component.Name, Namespace: ns}, cr)).Should(Succeed())
			g.Expect(cr.Status.Value).Should(Equal(common.StatusOK))
			g.Expect(cr.Status.ID).ShouldNot(BeEmpty())
		}, timeout, interval).Should(Succeed())

		cr := &keycloakApi.KeycloakRealmComponent{}
		Expect(k8sClient.Get(ctx, types.NamespacedName{Name: component.Name, Namespace: ns}, cr)).Should(Succeed())
		componentID := cr.Status.ID

		By("By deleting the KeycloakRealmComponent from k8s")
		Expect(k8sClient.Delete(ctx, cr)).Should(Succeed())
		Eventually(func() bool {
			err := k8sClient.Get(ctx, types.NamespacedName{Name: component.Name, Namespace: ns}, &keycloakApi.KeycloakRealmComponent{})
			return k8sErrors.IsNotFound(err)
		}, timeout, interval).Should(BeTrue(), "KeycloakRealmComponent with preserveResourcesOnDeletion should be deleted from k8s")

		By("By verifying the component still exists in Keycloak")
		kcComp, _, err := keycloakApiClient.RealmComponents.GetComponent(ctx, KeycloakRealmCR, componentID)
		Expect(err).ShouldNot(HaveOccurred())
		Expect(kcComp).ShouldNot(BeNil())

		By("By cleaning up the component from Keycloak")
		_, err = keycloakApiClient.RealmComponents.DeleteComponent(ctx, KeycloakRealmCR, componentID)
		Expect(err).ShouldNot(HaveOccurred())
	})

	It("Should create component resource with secret reference in config", func() {
		By("By creating a secret")
		clientSecret := &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-keycloak-realm-component-secret",
				Namespace: ns,
			},
			Data: map[string][]byte{
				"secretKey": []byte("secretValue"),
			},
		}
		Expect(k8sClient.Create(ctx, clientSecret)).Should(Succeed())

		By("By creating a KeycloakRealmComponent with a secret reference in config")
		component := &keycloakApi.KeycloakRealmComponent{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "component-with-secret",
				Namespace: ns,
			},
			Spec: keycloakApi.KeycloakComponentSpec{
				Name:         "component-with-secret",
				ProviderID:   keyProviderID,
				ProviderType: keyProviderType,
				RealmRef: common.RealmRef{
					Kind: keycloakApi.KeycloakRealmKind,
					Name: KeycloakRealmCR,
				},
				Config: map[string][]string{
					"priority":   {"100"},
					"privateKey": {"$test-keycloak-realm-component-secret:secretKey"},
				},
			},
		}
		Expect(k8sClient.Create(ctx, component)).Should(Succeed())
		Eventually(func(g Gomega) {
			cr := &keycloakApi.KeycloakRealmComponent{}
			g.Expect(k8sClient.Get(ctx, types.NamespacedName{Name: component.Name, Namespace: ns}, cr)).Should(Succeed())
			g.Expect(cr.Status.Value).Should(Equal(common.StatusOK))
			g.Expect(cr.Status.ID).ShouldNot(BeEmpty())
		}, timeout, interval).Should(Succeed())

		By("By verifying the plain config value is stored in Keycloak")
		Eventually(func(g Gomega) {
			cr := &keycloakApi.KeycloakRealmComponent{}
			g.Expect(k8sClient.Get(ctx, types.NamespacedName{Name: component.Name, Namespace: ns}, cr)).Should(Succeed())

			kcComp, _, err := keycloakApiClient.RealmComponents.GetComponent(ctx, KeycloakRealmCR, cr.Status.ID)
			g.Expect(err).ShouldNot(HaveOccurred())
			g.Expect(kcComp).ShouldNot(BeNil())
			g.Expect(kcComp.Config).ShouldNot(BeNil())
			g.Expect((*kcComp.Config)["priority"]).Should(ContainElement("100"))
		}, timeout, interval).Should(Succeed())
	})

	It("Should fail to create resource with nonexistent secret reference in config", func() {
		By("By creating a KeycloakRealmComponent with a non-existent secret reference")
		component := &keycloakApi.KeycloakRealmComponent{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "component-with-secret-should-fail",
				Namespace: ns,
			},
			Spec: keycloakApi.KeycloakComponentSpec{
				Name:         "component-with-secret-should-fail",
				ProviderID:   scopeProviderID,
				ProviderType: scopeProviderType,
				RealmRef: common.RealmRef{
					Kind: keycloakApi.KeycloakRealmKind,
					Name: KeycloakRealmCR,
				},
				Config: map[string][]string{
					"bindDn":         {"uid=serviceaccount,cn=users,dc=example,dc=com"},
					"bindCredential": {"$nonexistent-secret:secretKey"},
				},
			},
		}
		Expect(k8sClient.Create(ctx, component)).Should(Succeed())
		Eventually(func(g Gomega) {
			cr := &keycloakApi.KeycloakRealmComponent{}
			g.Expect(k8sClient.Get(ctx, types.NamespacedName{Name: component.Name, Namespace: ns}, cr)).Should(Succeed())
			g.Expect(cr.Status.Value).Should(Equal(
				`realm component chain processing failed: failed to serve handler: unable to map config secrets: failed to get secret nonexistent-secret: Secret "nonexistent-secret" not found`,
			))
		}, timeout, interval).Should(Succeed())
	})

	It("Should create component without Config", func() {
		By("By creating a KeycloakRealmComponent with no config")
		component := &keycloakApi.KeycloakRealmComponent{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "component-without-config",
				Namespace: ns,
			},
			Spec: keycloakApi.KeycloakComponentSpec{
				Name:         "component-without-config",
				ProviderID:   scopeProviderID,
				ProviderType: scopeProviderType,
				RealmRef: common.RealmRef{
					Kind: keycloakApi.KeycloakRealmKind,
					Name: KeycloakRealmCR,
				},
			},
		}
		Expect(k8sClient.Create(ctx, component)).Should(Succeed())
		Eventually(func(g Gomega) {
			cr := &keycloakApi.KeycloakRealmComponent{}
			g.Expect(k8sClient.Get(ctx, types.NamespacedName{Name: component.Name, Namespace: ns}, cr)).Should(Succeed())
			g.Expect(cr.Status.Value).Should(Equal(common.StatusOK))
			g.Expect(cr.Status.ID).ShouldNot(BeEmpty())
		}, timeout, interval).Should(Succeed())

		By("By verifying Name, ProviderId and ProviderType in Keycloak")
		Eventually(func(g Gomega) {
			cr := &keycloakApi.KeycloakRealmComponent{}
			g.Expect(k8sClient.Get(ctx, types.NamespacedName{Name: component.Name, Namespace: ns}, cr)).Should(Succeed())

			kcComp, _, err := keycloakApiClient.RealmComponents.GetComponent(ctx, KeycloakRealmCR, cr.Status.ID)
			g.Expect(err).ShouldNot(HaveOccurred())
			g.Expect(kcComp).ShouldNot(BeNil())
			g.Expect(kcComp.Name).ShouldNot(BeNil())
			g.Expect(*kcComp.Name).Should(Equal(component.Spec.Name))
			g.Expect(kcComp.ProviderId).ShouldNot(BeNil())
			g.Expect(*kcComp.ProviderId).Should(Equal(component.Spec.ProviderID))
			g.Expect(kcComp.ProviderType).ShouldNot(BeNil())
			g.Expect(*kcComp.ProviderType).Should(Equal(component.Spec.ProviderType))
		}, timeout, interval).Should(Succeed())
	})

	It("Should create component with multiple Config values and verify them in Keycloak", func() {
		By("By creating a key provider component with multi-value config")
		component := &keycloakApi.KeycloakRealmComponent{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "component-with-multi-config",
				Namespace: ns,
			},
			Spec: keycloakApi.KeycloakComponentSpec{
				Name:         "component-with-multi-config",
				ProviderID:   keyProviderID,
				ProviderType: keyProviderType,
				RealmRef: common.RealmRef{
					Kind: keycloakApi.KeycloakRealmKind,
					Name: KeycloakRealmCR,
				},
				Config: map[string][]string{
					"priority": {"100"},
					"keySize":  {"2048"},
				},
			},
		}
		Expect(k8sClient.Create(ctx, component)).Should(Succeed())
		Eventually(func(g Gomega) {
			cr := &keycloakApi.KeycloakRealmComponent{}
			g.Expect(k8sClient.Get(ctx, types.NamespacedName{Name: component.Name, Namespace: ns}, cr)).Should(Succeed())
			g.Expect(cr.Status.Value).Should(Equal(common.StatusOK))
			g.Expect(cr.Status.ID).ShouldNot(BeEmpty())
		}, timeout, interval).Should(Succeed())

		By("By verifying all config values are present in Keycloak")
		Eventually(func(g Gomega) {
			cr := &keycloakApi.KeycloakRealmComponent{}
			g.Expect(k8sClient.Get(ctx, types.NamespacedName{Name: component.Name, Namespace: ns}, cr)).Should(Succeed())

			kcComp, _, err := keycloakApiClient.RealmComponents.GetComponent(ctx, KeycloakRealmCR, cr.Status.ID)
			g.Expect(err).ShouldNot(HaveOccurred())
			g.Expect(kcComp).ShouldNot(BeNil())
			g.Expect(kcComp.Config).ShouldNot(BeNil())
			g.Expect((*kcComp.Config)["priority"]).Should(ContainElement("100"))
			g.Expect((*kcComp.Config)["keySize"]).Should(ContainElement("2048"))
		}, timeout, interval).Should(Succeed())
	})

	It("Should update a KeycloakRealmComponent config and reflect changes in Keycloak", func() {
		By("By creating a key provider component")
		component := &keycloakApi.KeycloakRealmComponent{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-component-for-update",
				Namespace: ns,
			},
			Spec: keycloakApi.KeycloakComponentSpec{
				Name:         "test-component-for-update",
				ProviderID:   keyProviderID,
				ProviderType: keyProviderType,
				RealmRef: common.RealmRef{
					Kind: keycloakApi.KeycloakRealmKind,
					Name: KeycloakRealmCR,
				},
				Config: map[string][]string{
					"priority": {"50"},
					"keySize":  {"2048"},
				},
			},
		}
		Expect(k8sClient.Create(ctx, component)).Should(Succeed())
		Eventually(func(g Gomega) {
			cr := &keycloakApi.KeycloakRealmComponent{}
			g.Expect(k8sClient.Get(ctx, types.NamespacedName{Name: component.Name, Namespace: ns}, cr)).Should(Succeed())
			g.Expect(cr.Status.Value).Should(Equal(common.StatusOK))
			g.Expect(cr.Status.ID).ShouldNot(BeEmpty())
		}, timeout, interval).Should(Succeed())

		By("By updating the priority config value")
		updatedCR := &keycloakApi.KeycloakRealmComponent{}
		Expect(k8sClient.Get(ctx, types.NamespacedName{Name: component.Name, Namespace: ns}, updatedCR)).Should(Succeed())
		updatedCR.Spec.Config = map[string][]string{
			"priority": {"200"},
			"keySize":  {"2048"},
		}
		Expect(k8sClient.Update(ctx, updatedCR)).Should(Succeed())

		By("By verifying the updated priority is reflected in Keycloak")
		Eventually(func(g Gomega) {
			cr := &keycloakApi.KeycloakRealmComponent{}
			g.Expect(k8sClient.Get(ctx, types.NamespacedName{Name: component.Name, Namespace: ns}, cr)).Should(Succeed())
			g.Expect(cr.Status.Value).Should(Equal(common.StatusOK))

			kcComp, _, err := keycloakApiClient.RealmComponents.GetComponent(ctx, KeycloakRealmCR, cr.Status.ID)
			g.Expect(err).ShouldNot(HaveOccurred())
			g.Expect(kcComp).ShouldNot(BeNil())
			g.Expect(kcComp.Config).ShouldNot(BeNil())
			g.Expect((*kcComp.Config)["priority"]).Should(ContainElement("200"))
			g.Expect((*kcComp.Config)["priority"]).ShouldNot(ContainElement("50"))
		}, timeout, interval).Should(Succeed())
	})

	It("Should reconcile idempotently — same status.ID after re-reconciliation", func() {
		By("By creating a KeycloakRealmComponent")
		component := &keycloakApi.KeycloakRealmComponent{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "component-idempotent",
				Namespace: ns,
			},
			Spec: keycloakApi.KeycloakComponentSpec{
				Name:         "component-idempotent",
				ProviderID:   scopeProviderID,
				ProviderType: scopeProviderType,
				RealmRef: common.RealmRef{
					Kind: keycloakApi.KeycloakRealmKind,
					Name: KeycloakRealmCR,
				},
			},
		}
		Expect(k8sClient.Create(ctx, component)).Should(Succeed())

		var firstID string
		Eventually(func(g Gomega) {
			cr := &keycloakApi.KeycloakRealmComponent{}
			g.Expect(k8sClient.Get(ctx, types.NamespacedName{Name: component.Name, Namespace: ns}, cr)).Should(Succeed())
			g.Expect(cr.Status.Value).Should(Equal(common.StatusOK))
			g.Expect(cr.Status.ID).ShouldNot(BeEmpty())
			firstID = cr.Status.ID
		}, timeout, interval).Should(Succeed())

		By("By triggering re-reconciliation via annotation update")
		cr := &keycloakApi.KeycloakRealmComponent{}
		Expect(k8sClient.Get(ctx, types.NamespacedName{Name: component.Name, Namespace: ns}, cr)).Should(Succeed())
		if cr.Annotations == nil {
			cr.Annotations = make(map[string]string)
		}
		cr.Annotations["test-trigger"] = "1"
		Expect(k8sClient.Update(ctx, cr)).Should(Succeed())

		By("By verifying status.ID remains stable and component still exists in Keycloak")
		Consistently(func(g Gomega) {
			cr := &keycloakApi.KeycloakRealmComponent{}
			g.Expect(k8sClient.Get(ctx, types.NamespacedName{Name: component.Name, Namespace: ns}, cr)).Should(Succeed())
			g.Expect(cr.Status.Value).Should(Equal(common.StatusOK))
			g.Expect(cr.Status.ID).Should(Equal(firstID))

			kcComp, _, err := keycloakApiClient.RealmComponents.GetComponent(ctx, KeycloakRealmCR, firstID)
			g.Expect(err).ShouldNot(HaveOccurred())
			g.Expect(kcComp).ShouldNot(BeNil())
		}, time.Second*3, interval).Should(Succeed())
	})

	It("Should adopt an existing Keycloak component when spec.Name matches", func() {
		By("By pre-creating a component directly in Keycloak")
		name := "component-precreated"
		resp, err := keycloakApiClient.RealmComponents.CreateComponent(ctx, KeycloakRealmCR, keycloakapi.ComponentRepresentation{
			Name:         &name,
			ProviderId:   func() *string { s := scopeProviderID; return &s }(),
			ProviderType: func() *string { s := scopeProviderType; return &s }(),
		})
		Expect(err).ShouldNot(HaveOccurred())
		Expect(resp).ShouldNot(BeNil())

		By("By creating a KeycloakRealmComponent CR with the same name")
		component := &keycloakApi.KeycloakRealmComponent{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "component-precreated-cr",
				Namespace: ns,
			},
			Spec: keycloakApi.KeycloakComponentSpec{
				Name:         name,
				ProviderID:   scopeProviderID,
				ProviderType: scopeProviderType,
				RealmRef: common.RealmRef{
					Kind: keycloakApi.KeycloakRealmKind,
					Name: KeycloakRealmCR,
				},
			},
		}
		Expect(k8sClient.Create(ctx, component)).Should(Succeed())
		Eventually(func(g Gomega) {
			cr := &keycloakApi.KeycloakRealmComponent{}
			g.Expect(k8sClient.Get(ctx, types.NamespacedName{Name: component.Name, Namespace: ns}, cr)).Should(Succeed())
			g.Expect(cr.Status.Value).Should(Equal(common.StatusOK))
			g.Expect(cr.Status.ID).ShouldNot(BeEmpty())

			kcComp, _, err := keycloakApiClient.RealmComponents.GetComponent(ctx, KeycloakRealmCR, cr.Status.ID)
			g.Expect(err).ShouldNot(HaveOccurred())
			g.Expect(kcComp).ShouldNot(BeNil())
			g.Expect(*kcComp.Name).Should(Equal(name))
		}, timeout, interval).Should(Succeed())
	})
})

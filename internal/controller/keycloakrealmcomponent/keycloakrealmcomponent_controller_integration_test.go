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
	"github.com/epam/edp-keycloak-operator/internal/controller/helper"
	"github.com/epam/edp-keycloak-operator/pkg/objectmeta"
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
				Name: "test-keycloak-realm-component",
				RealmRef: common.RealmRef{
					Kind: keycloakApi.KeycloakRealmKind,
					Name: KeycloakRealmCR,
				},
				ProviderID:   "scope",
				ProviderType: "org.keycloak.services.clientregistration.policy.ClientRegistrationPolicy",
			},
		}
		Expect(k8sClient.Create(ctx, component)).Should(Succeed())
		Eventually(func() bool {
			createdComponent := &keycloakApi.KeycloakRealmComponent{}
			err := k8sClient.Get(ctx, types.NamespacedName{Name: component.Name, Namespace: ns}, createdComponent)
			if err != nil {
				return false
			}

			return createdComponent.Status.Value == helper.StatusOK
		}, timeout, interval).Should(BeTrue())
		By("By creating a child KeycloakRealmComponent")
		childComponent := &keycloakApi.KeycloakRealmComponent{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-keycloak-realm-component-child",
				Namespace: ns,
			},
			Spec: keycloakApi.KeycloakComponentSpec{
				Name: "test-keycloak-realm-component-child",
				RealmRef: common.RealmRef{
					Kind: keycloakApi.KeycloakRealmKind,
					Name: KeycloakRealmCR,
				},
				ProviderID:   "scope",
				ProviderType: "org.keycloak.services.clientregistration.policy.ClientRegistrationPolicy",
				ParentRef: &keycloakApi.ParentComponent{
					Kind: keycloakApi.KeycloakRealmComponentKind,
					Name: component.Name,
				},
			},
		}
		Expect(k8sClient.Create(ctx, childComponent)).Should(Succeed())
		Eventually(func() bool {
			createdChildComponent := &keycloakApi.KeycloakRealmComponent{}
			err := k8sClient.Get(ctx, types.NamespacedName{Name: childComponent.Name, Namespace: ns}, createdChildComponent)
			if err != nil {
				return false
			}

			return createdChildComponent.Status.Value == helper.StatusOK
		}, timeout, interval).Should(BeTrue())
		By("By creating a KeycloakRealmComponent with parent realm")
		componentWithParentRealm := &keycloakApi.KeycloakRealmComponent{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-keycloak-realm-component-with-parent-realm",
				Namespace: ns,
			},
			Spec: keycloakApi.KeycloakComponentSpec{
				Name: "test-keycloak-realm-component-with-parent-realm",
				RealmRef: common.RealmRef{
					Kind: keycloakApi.KeycloakRealmKind,
					Name: KeycloakRealmCR,
				},
				ProviderID:   "scope",
				ProviderType: "org.keycloak.services.clientregistration.policy.ClientRegistrationPolicy",
				ParentRef: &keycloakApi.ParentComponent{
					Kind: keycloakApi.KeycloakRealmKind,
					Name: KeycloakRealmCR,
				},
			},
		}
		Expect(k8sClient.Create(ctx, componentWithParentRealm)).Should(Succeed())
		Eventually(func() bool {
			createdComponentWithParentRealm := &keycloakApi.KeycloakRealmComponent{}
			err := k8sClient.Get(ctx, types.NamespacedName{Name: childComponent.Name, Namespace: ns}, createdComponentWithParentRealm)
			if err != nil {
				return false
			}

			return createdComponentWithParentRealm.Status.Value == helper.StatusOK
		}, timeout, interval).Should(BeTrue())
	})
	It("Should delete KeycloakRealmComponents", func() {
		By("By getting a parent KeycloakRealmComponent")
		component := &keycloakApi.KeycloakRealmComponent{}
		Expect(k8sClient.Get(ctx, types.NamespacedName{Namespace: ns, Name: componentCR}, component)).Should(Succeed())
		By("By deleting a parent KeycloakRealmComponent")
		Expect(k8sClient.Delete(ctx, component)).Should(Succeed())
		Eventually(func() bool {
			deletedComponent := &keycloakApi.KeycloakRealmComponent{}
			err := k8sClient.Get(ctx, types.NamespacedName{Name: component.Name, Namespace: ns}, deletedComponent)

			return k8sErrors.IsNotFound(err)
		}, timeout, interval).Should(BeTrue(), "parent KeycloakRealmComponent should be deleted")
		By("By getting a parent KeycloakRealmComponent")
		childComponent := &keycloakApi.KeycloakRealmComponent{}
		Expect(k8sClient.Get(ctx, types.NamespacedName{Namespace: ns, Name: childComponentCR}, childComponent)).Should(Succeed())
		By("By deleting a child KeycloakRealmComponent")
		Expect(k8sClient.Delete(ctx, childComponent)).Should(Succeed())
		Eventually(func() bool {
			deletedChildComponent := &keycloakApi.KeycloakRealmComponent{}
			err := k8sClient.Get(ctx, types.NamespacedName{Name: component.Name, Namespace: ns}, deletedChildComponent)

			return k8sErrors.IsNotFound(err)
		}, timeout, interval).Should(BeTrue(), "child KeycloakRealmComponent should be deleted")
	})
	It("Should fail with invalid realm", func() {
		const (
			timeout  = time.Second * 5
			interval = time.Second
		)
		By("By creating a KeycloakRealmComponent")
		component := &keycloakApi.KeycloakRealmComponent{
			ObjectMeta: metav1.ObjectMeta{
				Name:      componentCR,
				Namespace: ns,
			},
			Spec: keycloakApi.KeycloakComponentSpec{
				Name:         "test-keycloak-realm-component",
				Realm:        "invalid-realm",
				ProviderID:   "scope",
				ProviderType: "org.keycloak.services.clientregistration.policy.ClientRegistrationPolicy",
			},
		}
		Expect(k8sClient.Create(ctx, component)).Should(Succeed())
		Consistently(func() bool {
			createdComponent := &keycloakApi.KeycloakRealmComponent{}
			err := k8sClient.Get(ctx, types.NamespacedName{Name: component.Name, Namespace: ns}, createdComponent)
			if err != nil {
				return false
			}

			return createdComponent.Status.Value != helper.StatusOK
		}, timeout, interval).Should(BeTrue(), "KeycloakRealmComponent should be in failed state")
	})
	It("Should fail with invalid parent component", func() {
		const (
			timeout  = time.Second * 5
			interval = time.Second
		)
		By("By creating a KeycloakRealmComponent with invalid parent component")
		component := &keycloakApi.KeycloakRealmComponent{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-keycloak-realm-component-invalid-parent-component",
				Namespace: ns,
			},
			Spec: keycloakApi.KeycloakComponentSpec{
				Name: "test-keycloak-realm-component-invalid-parent-component",
				RealmRef: common.RealmRef{
					Kind: keycloakApi.KeycloakRealmKind,
					Name: KeycloakRealmCR,
				},
				ProviderID:   "scope",
				ProviderType: "org.keycloak.services.clientregistration.policy.ClientRegistrationPolicy",
				ParentRef: &keycloakApi.ParentComponent{
					Kind: keycloakApi.KeycloakRealmComponentKind,
					Name: "invalid-parent-component",
				},
			},
		}
		Expect(k8sClient.Create(ctx, component)).Should(Succeed())
		Consistently(func() bool {
			createdComponent := &keycloakApi.KeycloakRealmComponent{}
			err := k8sClient.Get(ctx, types.NamespacedName{Name: component.Name, Namespace: ns}, createdComponent)
			if err != nil {
				return false
			}

			return createdComponent.Status.Value != helper.StatusOK
		}, timeout, interval).Should(BeTrue(), "KeycloakRealmComponent with invalid parent component should be in failed state")
	})
	It("Should fail with invalid parent realm", func() {
		const (
			timeout  = time.Second * 5
			interval = time.Second
		)
		By("By creating a KeycloakRealmComponent with invalid parent realm")
		component := &keycloakApi.KeycloakRealmComponent{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-keycloak-realm-component-invalid-parent-realm",
				Namespace: ns,
			},
			Spec: keycloakApi.KeycloakComponentSpec{
				Name: "test-keycloak-realm-component-invalid-parent-realm",
				RealmRef: common.RealmRef{
					Kind: keycloakApi.KeycloakRealmKind,
					Name: KeycloakRealmCR,
				},
				ProviderID:   "scope",
				ProviderType: "org.keycloak.services.clientregistration.policy.ClientRegistrationPolicy",
				ParentRef: &keycloakApi.ParentComponent{
					Kind: keycloakApi.KeycloakRealmKind,
					Name: "invalid-parent-component",
				},
			},
		}
		Expect(k8sClient.Create(ctx, component)).Should(Succeed())
		Consistently(func() bool {
			createdComponent := &keycloakApi.KeycloakRealmComponent{}
			err := k8sClient.Get(ctx, types.NamespacedName{Name: component.Name, Namespace: ns}, createdComponent)
			if err != nil {
				return false
			}

			return createdComponent.Status.Value != helper.StatusOK
		}, time.Second*3, interval).Should(BeTrue(), "KeycloakRealmComponent with invalid parent realm should be in failed state")
	})
	It("Should skip keycloak resource removing if preserveResourcesOnDeletion is set", func() {
		By("By creating a KeycloakRealmComponent")
		component := &keycloakApi.KeycloakRealmComponent{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "component-with-preserve-resources-on-deletion",
				Namespace: ns,
				Annotations: map[string]string{
					objectmeta.PreserveResourcesOnDeletionAnnotation: "true",
				},
			},
			Spec: keycloakApi.KeycloakComponentSpec{
				Name: "component-with-preserve-resources-on-deletion",
				RealmRef: common.RealmRef{
					Kind: keycloakApi.KeycloakRealmKind,
					Name: KeycloakRealmCR,
				},
				ProviderID:   "scope",
				ProviderType: "org.keycloak.services.clientregistration.policy.ClientRegistrationPolicy",
			},
		}
		Expect(k8sClient.Create(ctx, component)).Should(Succeed())
		Eventually(func() bool {
			createdComponent := &keycloakApi.KeycloakRealmComponent{}
			err := k8sClient.Get(ctx, types.NamespacedName{Name: component.Name, Namespace: ns}, createdComponent)
			if err != nil {
				return false
			}

			return createdComponent.Status.Value == helper.StatusOK
		}, timeout, interval).Should(BeTrue())
		By("By deleting KeycloakRealmComponent")
		Expect(k8sClient.Delete(ctx, component)).Should(Succeed())
		Eventually(func() bool {
			deletedComponent := &keycloakApi.KeycloakRealmComponent{}
			err := k8sClient.Get(ctx, types.NamespacedName{Name: component.Name, Namespace: ns}, deletedComponent)

			return k8sErrors.IsNotFound(err)
		}, timeout, interval).Should(BeTrue(), "KeycloakRealmComponent with preserveResourcesOnDeletion annotation should be deleted")
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
		By("By creating a KeycloakRealmComponent")
		component := &keycloakApi.KeycloakRealmComponent{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "component-with-secret",
				Namespace: ns,
			},
			Spec: keycloakApi.KeycloakComponentSpec{
				Name: "component-with-secret",
				RealmRef: common.RealmRef{
					Kind: keycloakApi.KeycloakRealmKind,
					Name: KeycloakRealmCR,
				},
				ProviderID:   "scope",
				ProviderType: "org.keycloak.services.clientregistration.policy.ClientRegistrationPolicy",
				Config: map[string][]string{
					"bindDn":         {"uid=serviceaccount,cn=users,dc=example,dc=com"},
					"bindCredential": {"$test-keycloak-realm-component-secret:secretKey"},
				},
			},
		}
		Expect(k8sClient.Create(ctx, component)).Should(Succeed())
		Eventually(func() bool {
			createdComponent := &keycloakApi.KeycloakRealmComponent{}
			err := k8sClient.Get(ctx, types.NamespacedName{Name: component.Name, Namespace: ns}, createdComponent)
			if err != nil {
				return false
			}

			return createdComponent.Status.Value == helper.StatusOK
		}, timeout, interval).Should(BeTrue())
	})
	It("Should fail to create resource with nonexistent secret reference in config", func() {
		By("By creating a KeycloakRealmComponent")
		component := &keycloakApi.KeycloakRealmComponent{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "component-with-secret-should-fail",
				Namespace: ns,
			},
			Spec: keycloakApi.KeycloakComponentSpec{
				Name: "component-with-secret-should-fail",
				RealmRef: common.RealmRef{
					Kind: keycloakApi.KeycloakRealmKind,
					Name: KeycloakRealmCR,
				},
				ProviderID:   "scope",
				ProviderType: "org.keycloak.services.clientregistration.policy.ClientRegistrationPolicy",
				Config: map[string][]string{
					"bindDn":         {"uid=serviceaccount,cn=users,dc=example,dc=com"},
					"bindCredential": {"$nonexistent-secret:secretKey"},
				},
			},
		}
		Expect(k8sClient.Create(ctx, component)).Should(Succeed())
		Eventually(func() bool {
			createdComponent := &keycloakApi.KeycloakRealmComponent{}
			err := k8sClient.Get(ctx, types.NamespacedName{Name: component.Name, Namespace: ns}, createdComponent)
			if err != nil {
				return false
			}

			return createdComponent.Status.Value == "unable to map config secrets: failed to get secret nonexistent-secret: Secret \"nonexistent-secret\" not found"
		}, timeout, interval).Should(BeTrue())
	})
})

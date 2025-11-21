package keycloakrealmgroup

import (
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gstruct"

	"github.com/Nerzal/gocloak/v12"
	k8sErrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	"github.com/epam/edp-keycloak-operator/api/common"
	keycloakApi "github.com/epam/edp-keycloak-operator/api/v1"
	"github.com/epam/edp-keycloak-operator/pkg/client/keycloak/adapter"
	"github.com/epam/edp-keycloak-operator/pkg/objectmeta"
)

var _ = Describe("KeycloakRealmGroup controller", Ordered, func() {
	const (
		parentGroupCR = "test-parent-group"
		childGroupCR  = "test-child-group"
		childGroup2CR = "test-child-group2"
	)
	It("Should create KeycloakRealmGroup with parent-child hierarchy", func() {
		By("Creating role for the group")
		_, err := keycloakApiClient.CreateRealmRole(ctx, getKeyCloakToken(), KeycloakRealmCR, gocloak.Role{
			Name: gocloak.StringP("test-group-role"),
		})
		Expect(adapter.SkipAlreadyExistsErr(err)).ShouldNot(HaveOccurred())

		By("Creating a parent KeycloakRealmGroup")
		parentGroup := &keycloakApi.KeycloakRealmGroup{
			ObjectMeta: metav1.ObjectMeta{
				Name:      parentGroupCR,
				Namespace: ns,
			},
			Spec: keycloakApi.KeycloakRealmGroupSpec{
				Name: "parent-group",
				RealmRef: common.RealmRef{
					Kind: keycloakApi.KeycloakRealmKind,
					Name: KeycloakRealmCR,
				},
				Path:       "/parent-group",
				RealmRoles: []string{"test-group-role"},
			},
		}
		Expect(k8sClient.Create(ctx, parentGroup)).Should(Succeed())
		Eventually(func(g Gomega) {
			createdGroup := &keycloakApi.KeycloakRealmGroup{}
			err = k8sClient.Get(ctx, types.NamespacedName{Name: parentGroup.Name, Namespace: ns}, createdGroup)
			g.Expect(err).ShouldNot(HaveOccurred())
			g.Expect(createdGroup.Status.Value).Should(Equal(common.StatusOK))
			g.Expect(createdGroup.Status.ID).ShouldNot(BeEmpty())
		}).WithTimeout(time.Second * 20).WithPolling(time.Second).Should(Succeed())

		By("Creating a child KeycloakRealmGroup")
		childGroup := &keycloakApi.KeycloakRealmGroup{
			ObjectMeta: metav1.ObjectMeta{
				Name:      childGroupCR,
				Namespace: ns,
			},
			Spec: keycloakApi.KeycloakRealmGroupSpec{
				Name: "child-group",
				RealmRef: common.RealmRef{
					Kind: keycloakApi.KeycloakRealmKind,
					Name: KeycloakRealmCR,
				},
				ParentGroup: &common.GroupRef{
					Name: parentGroupCR,
				},
			},
		}
		Expect(k8sClient.Create(ctx, childGroup)).Should(Succeed())
		Eventually(func(g Gomega) {
			createdGroup := &keycloakApi.KeycloakRealmGroup{}
			err = k8sClient.Get(ctx, types.NamespacedName{Name: childGroup.Name, Namespace: ns}, createdGroup)
			g.Expect(err).ShouldNot(HaveOccurred())
			g.Expect(createdGroup.Status.Value).Should(Equal(common.StatusOK))
			g.Expect(createdGroup.Status.ID).ShouldNot(BeEmpty())
		}).WithTimeout(time.Second * 20).WithPolling(time.Second).Should(Succeed())

		By("Creating another child KeycloakRealmGroup")
		childGroup2 := &keycloakApi.KeycloakRealmGroup{
			ObjectMeta: metav1.ObjectMeta{
				Name:      childGroup2CR,
				Namespace: ns,
			},
			Spec: keycloakApi.KeycloakRealmGroupSpec{
				Name: "child-group2",
				RealmRef: common.RealmRef{
					Kind: keycloakApi.KeycloakRealmKind,
					Name: KeycloakRealmCR,
				},
				ParentGroup: &common.GroupRef{
					Name: parentGroupCR,
				},
			},
		}
		Expect(k8sClient.Create(ctx, childGroup2)).Should(Succeed())
		Eventually(func(g Gomega) {
			createdGroup := &keycloakApi.KeycloakRealmGroup{}
			err = k8sClient.Get(ctx, types.NamespacedName{Name: childGroup2.Name, Namespace: ns}, createdGroup)
			g.Expect(err).ShouldNot(HaveOccurred())
			g.Expect(createdGroup.Status.Value).Should(Equal(common.StatusOK))
			g.Expect(createdGroup.Status.ID).ShouldNot(BeEmpty())
		}).WithTimeout(time.Second * 20).WithPolling(time.Second).Should(Succeed())

		gr, err := keycloakApiClient.GetGroups(ctx, getKeyCloakToken(), KeycloakRealmCR, gocloak.GetGroupsParams{})
		Expect(err).ShouldNot(HaveOccurred())
		// Check if the parent group is in the list (top-level groups only).
		// childGroup and childGroup2 should not be in the list since they are children.
		Expect(gr).Should(ContainElement(HaveField("Name", PointTo(Equal(parentGroup.Spec.Name)))))
		Expect(gr).ShouldNot(ContainElement(HaveField("Name", PointTo(Equal(childGroup.Spec.Name)))))
		Expect(gr).ShouldNot(ContainElement(HaveField("Name", PointTo(Equal(childGroup2.Spec.Name)))))
	})
	It("Should delete KeycloakRealmGroup hierarchy", func() {
		By("Deleting child groups first")
		childGroup := &keycloakApi.KeycloakRealmGroup{}
		Expect(k8sClient.Get(ctx, types.NamespacedName{Namespace: ns, Name: childGroupCR}, childGroup)).Should(Succeed())
		Expect(k8sClient.Delete(ctx, childGroup)).Should(Succeed())
		Eventually(func(g Gomega) {
			deletedGroup := &keycloakApi.KeycloakRealmGroup{}
			err := k8sClient.Get(ctx, types.NamespacedName{Name: childGroup.Name, Namespace: ns}, deletedGroup)
			g.Expect(k8sErrors.IsNotFound(err)).Should(BeTrue())
		}, timeout, interval).Should(Succeed())

		childGroup2 := &keycloakApi.KeycloakRealmGroup{}
		Expect(k8sClient.Get(ctx, types.NamespacedName{Namespace: ns, Name: childGroup2CR}, childGroup2)).Should(Succeed())
		Expect(k8sClient.Delete(ctx, childGroup2)).Should(Succeed())
		Eventually(func(g Gomega) {
			deletedGroup := &keycloakApi.KeycloakRealmGroup{}
			err := k8sClient.Get(ctx, types.NamespacedName{Name: childGroup2.Name, Namespace: ns}, deletedGroup)
			g.Expect(k8sErrors.IsNotFound(err)).Should(BeTrue())
		}, timeout, interval).Should(Succeed())

		By("Deleting parent group")
		parentGroup := &keycloakApi.KeycloakRealmGroup{}
		Expect(k8sClient.Get(ctx, types.NamespacedName{Namespace: ns, Name: parentGroupCR}, parentGroup)).Should(Succeed())
		Expect(k8sClient.Delete(ctx, parentGroup)).Should(Succeed())
		Eventually(func(g Gomega) {
			deletedGroup := &keycloakApi.KeycloakRealmGroup{}
			err := k8sClient.Get(ctx, types.NamespacedName{Name: parentGroup.Name, Namespace: ns}, deletedGroup)
			g.Expect(k8sErrors.IsNotFound(err)).Should(BeTrue())
		}, timeout, interval).Should(Succeed())
	})
	It("Should fail to create group with non-existent parent", func() {
		By("Creating a group with non-existent parent")
		group := &keycloakApi.KeycloakRealmGroup{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-group-with-invalid-parent",
				Namespace: ns,
			},
			Spec: keycloakApi.KeycloakRealmGroupSpec{
				Name: "group-with-invalid-parent",
				RealmRef: common.RealmRef{
					Kind: keycloakApi.KeycloakRealmKind,
					Name: KeycloakRealmCR,
				},
				ParentGroup: &common.GroupRef{
					Name: "non-existent-parent",
				},
			},
		}
		Expect(k8sClient.Create(ctx, group)).Should(Succeed())

		By("Checking group fails with parent not found error")
		Eventually(func(g Gomega) {
			createdGroup := &keycloakApi.KeycloakRealmGroup{}
			err := k8sClient.Get(ctx, types.NamespacedName{Name: group.Name, Namespace: ns}, createdGroup)
			g.Expect(err).ShouldNot(HaveOccurred())
			g.Expect(createdGroup.Status.Value).Should(ContainSubstring("unable to get parent KeycloakRealmGroup"))
		}, time.Second*10, time.Second).Should(Succeed())

		By("Cleaning up")
		Expect(k8sClient.Delete(ctx, group)).Should(Succeed())
	})
	It("Should preserve group with annotation", func() {
		By("Creating a KeycloakRealmGroup")
		group := &keycloakApi.KeycloakRealmGroup{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-keycloak-realm-group-preserve",
				Namespace: ns,
				Annotations: map[string]string{
					objectmeta.PreserveResourcesOnDeletionAnnotation: "true",
				},
			},
			Spec: keycloakApi.KeycloakRealmGroupSpec{
				RealmRef: common.RealmRef{
					Kind: keycloakApi.KeycloakRealmKind,
					Name: KeycloakRealmCR,
				},
				Name: "test-group-preserve",
			},
		}
		Expect(k8sClient.Create(ctx, group)).Should(Succeed())
		Eventually(func(g Gomega) {
			createdGroup := &keycloakApi.KeycloakRealmGroup{}
			err := k8sClient.Get(ctx, types.NamespacedName{Name: group.Name, Namespace: ns}, createdGroup)
			g.Expect(err).ShouldNot(HaveOccurred())
			g.Expect(createdGroup.Status.Value).Should(Equal(common.StatusOK))
		}).WithTimeout(time.Second * 20).WithPolling(time.Second).Should(Succeed())

		By("Deleting KeycloakRealmGroup")
		Expect(k8sClient.Delete(ctx, group)).Should(Succeed())
		Eventually(func(g Gomega) {
			groups, err := keycloakApiClient.GetGroups(ctx, getKeyCloakToken(), KeycloakRealmCR, gocloak.GetGroupsParams{
				Search: gocloak.StringP("test-group-preserve"),
			})
			g.Expect(err).ShouldNot(HaveOccurred())
			g.Expect(groups).Should(HaveLen(1))
		}, time.Minute, time.Second*5).Should(Succeed())
	})
})

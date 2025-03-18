package keycloakrealmgroup

import (
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/Nerzal/gocloak/v12"
	k8sErrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	"github.com/epam/edp-keycloak-operator/api/common"
	keycloakApi "github.com/epam/edp-keycloak-operator/api/v1"
	"github.com/epam/edp-keycloak-operator/internal/controller/helper"
	"github.com/epam/edp-keycloak-operator/pkg/client/keycloak/adapter"
	"github.com/epam/edp-keycloak-operator/pkg/objectmeta"
)

var _ = Describe("KeycloakRealmGroup controller", Ordered, func() {
	const (
		groupCR = "test-keycloak-realm-group"
	)
	It("Should create KeycloakRealmGroup", func() {
		By("Creating role for the group")
		_, err := keycloakApiClient.CreateRealmRole(ctx, getKeyCloakToken(), KeycloakRealmCR, gocloak.Role{
			Name: gocloak.StringP("test-group-role"),
		})
		Expect(adapter.SkipAlreadyExistsErr(err)).ShouldNot(HaveOccurred())

		By("Creating a KeycloakRealmGroup subgroup")
		subgroup := &keycloakApi.KeycloakRealmGroup{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-subgroup",
				Namespace: ns,
			},
			Spec: keycloakApi.KeycloakRealmGroupSpec{
				Name: "test-subgroup",
				RealmRef: common.RealmRef{
					Kind: keycloakApi.KeycloakRealmKind,
					Name: KeycloakRealmCR,
				},
				Path: "/test-subgroup",
			},
		}
		Expect(k8sClient.Create(ctx, subgroup)).Should(Succeed())
		Eventually(func(g Gomega) {
			createdGroup := &keycloakApi.KeycloakRealmGroup{}
			err = k8sClient.Get(ctx, types.NamespacedName{Name: subgroup.Name, Namespace: ns}, createdGroup)
			g.Expect(err).ShouldNot(HaveOccurred())
			g.Expect(createdGroup.Status.Value).Should(Equal(helper.StatusOK))
		}).WithTimeout(time.Second * 20).WithPolling(time.Second).Should(Succeed())

		By("Creating a KeycloakRealmGroup subgroup2")
		subgroup2 := &keycloakApi.KeycloakRealmGroup{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-subgroup2",
				Namespace: ns,
			},
			Spec: keycloakApi.KeycloakRealmGroupSpec{
				Name: "test-subgroup2",
				RealmRef: common.RealmRef{
					Kind: keycloakApi.KeycloakRealmKind,
					Name: KeycloakRealmCR,
				},
				Path: "/test-subgroup2",
			},
		}
		Expect(k8sClient.Create(ctx, subgroup2)).Should(Succeed())
		Eventually(func(g Gomega) {
			createdGroup := &keycloakApi.KeycloakRealmGroup{}
			err = k8sClient.Get(ctx, types.NamespacedName{Name: subgroup2.Name, Namespace: ns}, createdGroup)
			g.Expect(err).ShouldNot(HaveOccurred())
			g.Expect(createdGroup.Status.Value).Should(Equal(helper.StatusOK))
		}).WithTimeout(time.Second * 20).WithPolling(time.Second).Should(Succeed())

		By("Creating a KeycloakRealmGroup")
		group := &keycloakApi.KeycloakRealmGroup{
			ObjectMeta: metav1.ObjectMeta{
				Name:      groupCR,
				Namespace: ns,
			},
			Spec: keycloakApi.KeycloakRealmGroupSpec{
				Name: "test-group",
				RealmRef: common.RealmRef{
					Kind: keycloakApi.KeycloakRealmKind,
					Name: KeycloakRealmCR,
				},
				Path:       "/test-group",
				RealmRoles: []string{"test-group-role"},
				SubGroups:  []string{"test-subgroup"},
			},
		}
		Expect(k8sClient.Create(ctx, group)).Should(Succeed())
		Eventually(func(g Gomega) {
			createdGroup := &keycloakApi.KeycloakRealmGroup{}
			err = k8sClient.Get(ctx, types.NamespacedName{Name: groupCR, Namespace: ns}, createdGroup)
			g.Expect(err).ShouldNot(HaveOccurred())
			g.Expect(createdGroup.Status.Value).Should(Equal(helper.StatusOK))
		}).WithTimeout(time.Second * 20).WithPolling(time.Second).Should(Succeed())
	})
	It("Should update KeycloakRealmGroup", func() {
		By("Getting KeycloakRealmGroup")
		group := &keycloakApi.KeycloakRealmGroup{}
		Expect(k8sClient.Get(ctx, types.NamespacedName{Namespace: ns, Name: groupCR}, group)).Should(Succeed())

		By("Updating a parent KeycloakRealmGroup")
		group.Spec.SubGroups = []string{"test-subgroup2"}

		Expect(k8sClient.Update(ctx, group)).Should(Succeed())
		Eventually(func(g Gomega) {
			updatedGroup := &keycloakApi.KeycloakRealmGroup{}
			err := k8sClient.Get(ctx, types.NamespacedName{Name: group.Name, Namespace: ns}, updatedGroup)
			g.Expect(err).ShouldNot(HaveOccurred())
			g.Expect(updatedGroup.Status.Value).Should(Equal(helper.StatusOK))
		}, time.Minute, time.Second*5).Should(Succeed())
	})
	It("Should delete KeycloakRealmGroup and subgroup", func() {
		By("Getting KeycloakRealmGroup")
		group := &keycloakApi.KeycloakRealmGroup{}
		Expect(k8sClient.Get(ctx, types.NamespacedName{Namespace: ns, Name: groupCR}, group)).Should(Succeed())
		By("Deleting KeycloakRealmGroup")
		Expect(k8sClient.Delete(ctx, group)).Should(Succeed())
		Eventually(func(g Gomega) {
			deletedGroup := &keycloakApi.KeycloakRealmGroup{}
			err := k8sClient.Get(ctx, types.NamespacedName{Name: group.Name, Namespace: ns}, deletedGroup)

			g.Expect(k8sErrors.IsNotFound(err)).Should(BeTrue())
		}, timeout, interval).Should(Succeed())

		By("Getting KeycloakRealmGroup subgroup")
		subgroup := &keycloakApi.KeycloakRealmGroup{}
		Expect(k8sClient.Get(ctx, types.NamespacedName{Namespace: ns, Name: "test-subgroup2"}, subgroup)).Should(Succeed())
		By("Deleting KeycloakRealmGroup subgroup")
		Expect(k8sClient.Delete(ctx, subgroup)).Should(Succeed())
		Eventually(func(g Gomega) {
			deletedSubGroup := &keycloakApi.KeycloakRealmGroup{}
			err := k8sClient.Get(ctx, types.NamespacedName{Name: subgroup.Name, Namespace: ns}, deletedSubGroup)

			g.Expect(k8sErrors.IsNotFound(err)).Should(BeTrue())
		}, timeout, interval).Should(Succeed())
	})
	It("Should delete KeycloakRealmGroup if subgroup is deleted", func() {
		By("Creating a subgroup")
		subgroup := &keycloakApi.KeycloakRealmGroup{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-subgroup-for-deletion",
				Namespace: ns,
			},
			Spec: keycloakApi.KeycloakRealmGroupSpec{
				Name: "test-subgroup-for-deletion",
				RealmRef: common.RealmRef{
					Kind: keycloakApi.KeycloakRealmKind,
					Name: KeycloakRealmCR,
				},
			},
		}
		Expect(k8sClient.Create(ctx, subgroup)).Should(Succeed())
		Eventually(func(g Gomega) {
			createdSubGroup := &keycloakApi.KeycloakRealmGroup{}
			err := k8sClient.Get(ctx, types.NamespacedName{Name: subgroup.Name, Namespace: ns}, createdSubGroup)
			g.Expect(err).ShouldNot(HaveOccurred())
			g.Expect(createdSubGroup.Status.Value).Should(Equal(helper.StatusOK))
		}).WithTimeout(time.Second * 20).WithPolling(time.Second).Should(Succeed())

		By("Creating a group with subgroup")
		group := &keycloakApi.KeycloakRealmGroup{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-group-for-deletion",
				Namespace: ns,
			},
			Spec: keycloakApi.KeycloakRealmGroupSpec{
				Name: "test-group-for-deletion",
				RealmRef: common.RealmRef{
					Kind: keycloakApi.KeycloakRealmKind,
					Name: KeycloakRealmCR,
				},
				SubGroups: []string{"test-subgroup-for-deletion"},
			},
		}
		Expect(k8sClient.Create(ctx, group)).Should(Succeed())
		Eventually(func(g Gomega) {
			createdGroup := &keycloakApi.KeycloakRealmGroup{}
			err := k8sClient.Get(ctx, types.NamespacedName{Name: group.Name, Namespace: ns}, createdGroup)
			g.Expect(err).ShouldNot(HaveOccurred())
			g.Expect(createdGroup.Status.Value).Should(Equal(helper.StatusOK))
		}).WithTimeout(time.Second * 20).WithPolling(time.Second).Should(Succeed())

		By("Deleting subgroup")
		Expect(k8sClient.Delete(ctx, subgroup)).Should(Succeed())
		Eventually(func(g Gomega) {
			deletedSubGroup := &keycloakApi.KeycloakRealmGroup{}
			err := k8sClient.Get(ctx, types.NamespacedName{Name: subgroup.Name, Namespace: ns}, deletedSubGroup)

			g.Expect(k8sErrors.IsNotFound(err)).Should(BeTrue())
		}, timeout, interval).Should(Succeed())

		By("Deleting group")
		Expect(k8sClient.Delete(ctx, group)).Should(Succeed())
		Eventually(func(g Gomega) {
			deletedGroup := &keycloakApi.KeycloakRealmGroup{}
			err := k8sClient.Get(ctx, types.NamespacedName{Name: group.Name, Namespace: ns}, deletedGroup)

			g.Expect(k8sErrors.IsNotFound(err)).Should(BeTrue())
		}, timeout, interval).Should(Succeed())
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
			g.Expect(createdGroup.Status.Value).Should(Equal(helper.StatusOK))
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
	It("Should fail to create KeycloakRealmGroup with not existing subgroup", func() {
		By("Creating a KeycloakRealmGroup")
		group := &keycloakApi.KeycloakRealmGroup{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-keycloak-realm-group-with-invalid-subgroup",
				Namespace: ns,
			},
			Spec: keycloakApi.KeycloakRealmGroupSpec{
				RealmRef: common.RealmRef{
					Kind: keycloakApi.KeycloakRealmKind,
					Name: KeycloakRealmCR,
				},
				Name:      "test-group-with-invalid-subgroup",
				SubGroups: []string{"not-existing-subgroup"},
			},
		}
		Expect(k8sClient.Create(ctx, group)).Should(Succeed())

		By("Waiting for KeycloakRealmGroup to be processed")
		time.Sleep(time.Second * 3)

		By("Checking KeycloakRealmGroup status")
		Consistently(func(g Gomega) {
			createdGroup := &keycloakApi.KeycloakRealmGroup{}
			err := k8sClient.Get(ctx, types.NamespacedName{Name: group.Name, Namespace: ns}, createdGroup)
			g.Expect(err).ShouldNot(HaveOccurred())
			g.Expect(createdGroup.Status.Value).Should(ContainSubstring("unable to sync realm group"))
		}, time.Second*3, time.Second).Should(Succeed())
	})
})

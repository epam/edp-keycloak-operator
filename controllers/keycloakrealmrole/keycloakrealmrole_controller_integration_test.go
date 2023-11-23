package keycloakrealmrole

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	k8sErrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	"github.com/epam/edp-keycloak-operator/api/common"
	keycloakApi "github.com/epam/edp-keycloak-operator/api/v1"
	"github.com/epam/edp-keycloak-operator/controllers/helper"
)

var _ = Describe("KeycloakRealmRole controller", func() {
	const (
		roleCR = "test-keycloak-realm-role"
	)
	It("Should create KeycloakRealmRole", func() {
		By("By creating a KeycloakRealmRole")
		role := &keycloakApi.KeycloakRealmRole{
			ObjectMeta: metav1.ObjectMeta{
				Name:      roleCR,
				Namespace: ns,
			},
			Spec: keycloakApi.KeycloakRealmRoleSpec{
				Name: "test-keycloak-realm-role",
				RealmRef: common.RealmRef{
					Kind: keycloakApi.KeycloakRealmKind,
					Name: KeycloakRealmCR,
				},
				Description: "test role",
				Attributes: map[string][]string{
					"test": {"test"},
				},
				IsDefault: true,
			},
		}
		Expect(k8sClient.Create(ctx, role)).Should(Succeed())
		Eventually(func() bool {
			createdRole := &keycloakApi.KeycloakRealmRole{}
			err := k8sClient.Get(ctx, types.NamespacedName{Name: role.Name, Namespace: ns}, createdRole)
			if err != nil {
				return false
			}

			return createdRole.Status.Value == helper.StatusOK
		}, timeout, interval).Should(BeTrue())
	})
	It("Should updated KeycloakRealmRole", func() {
		By("By getting KeycloakRealmRole")
		role := &keycloakApi.KeycloakRealmRole{}
		Expect(k8sClient.Get(ctx, types.NamespacedName{Namespace: ns, Name: roleCR}, role)).Should(Succeed())
		By("By updating KeycloakRealmRole")
		role.Spec.Description = "updated description"
		Expect(k8sClient.Update(ctx, role)).Should(Succeed())
		Eventually(func() bool {
			updatedRole := &keycloakApi.KeycloakRealmRole{}
			if err := k8sClient.Get(ctx, types.NamespacedName{Name: role.Name, Namespace: ns}, updatedRole); err != nil {
				return false
			}

			return updatedRole.Spec.Description == "updated description" && updatedRole.Status.Value == helper.StatusOK
		}, timeout, interval).Should(BeTrue(), "KeycloakRealmRole should be deleted")
	})
	It("Should delete KeycloakRealmRole", func() {
		By("By getting KeycloakRealmRole")
		role := &keycloakApi.KeycloakRealmRole{}
		Expect(k8sClient.Get(ctx, types.NamespacedName{Namespace: ns, Name: roleCR}, role)).Should(Succeed())
		By("By deleting KeycloakRealmRole")
		Expect(k8sClient.Delete(ctx, role)).Should(Succeed())
		Eventually(func() bool {
			deletedRole := &keycloakApi.KeycloakRealmRole{}
			err := k8sClient.Get(ctx, types.NamespacedName{Name: role.Name, Namespace: ns}, deletedRole)

			return k8sErrors.IsNotFound(err)
		}, timeout, interval).Should(BeTrue(), "KeycloakRealmRole should be deleted")
	})
	It("Should fail with invalid realm", func() {
		By("By creating a KeycloakRealmRole")
		role := &keycloakApi.KeycloakRealmRole{
			ObjectMeta: metav1.ObjectMeta{
				Name:      roleCR,
				Namespace: ns,
			},
			Spec: keycloakApi.KeycloakRealmRoleSpec{
				Name: "test-keycloak-realm-role-invalid-realm",
				RealmRef: common.RealmRef{
					Kind: keycloakApi.KeycloakRealmKind,
					Name: "invalid-realm",
				},
			},
		}
		Expect(k8sClient.Create(ctx, role)).Should(Succeed())
		Consistently(func() bool {
			createdRole := &keycloakApi.KeycloakRealmRole{}
			if err := k8sClient.Get(ctx, types.NamespacedName{Name: role.Name, Namespace: ns}, createdRole); err != nil {
				return false
			}

			return createdRole.Status.Value != helper.StatusOK
		}, timeout, interval).Should(BeTrue(), "KeycloakRealmRole should be in failed state")
	})
})

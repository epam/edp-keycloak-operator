package keycloakrealmrole

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
)

var _ = Describe("KeycloakRealmRole controller", Ordered, func() {
	const (
		roleCR = "test-keycloak-realm-role"
	)
	It("Should create KeycloakRealmRole", func() {
		By("Creating a KeycloakRealmRole")
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
		By("Getting KeycloakRealmRole")
		role := &keycloakApi.KeycloakRealmRole{}
		Expect(k8sClient.Get(ctx, types.NamespacedName{Namespace: ns, Name: roleCR}, role)).Should(Succeed())
		By("Updating KeycloakRealmRole")
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
		By("Getting KeycloakRealmRole")
		role := &keycloakApi.KeycloakRealmRole{}
		Expect(k8sClient.Get(ctx, types.NamespacedName{Namespace: ns, Name: roleCR}, role)).Should(Succeed())
		By("Deleting KeycloakRealmRole")
		Expect(k8sClient.Delete(ctx, role)).Should(Succeed())
		Eventually(func() bool {
			deletedRole := &keycloakApi.KeycloakRealmRole{}
			err := k8sClient.Get(ctx, types.NamespacedName{Name: role.Name, Namespace: ns}, deletedRole)

			return k8sErrors.IsNotFound(err)
		}, timeout, interval).Should(BeTrue(), "KeycloakRealmRole should be deleted")
	})

	It("Should create composite KeycloakRealmRole", func() {
		By("Creating realm role for composite role")
		_, err := keycloakApiClient.CreateRealmRole(ctx, getKeyCloakToken(), KeycloakRealmCR, gocloak.Role{
			Name: gocloak.StringP("role1"),
		})
		Expect(adapter.SkipAlreadyExistsErr(err)).ShouldNot(HaveOccurred())

		_, err = keycloakApiClient.CreateRealmRole(ctx, getKeyCloakToken(), KeycloakRealmCR, gocloak.Role{
			Name: gocloak.StringP("role2"),
		})
		Expect(adapter.SkipAlreadyExistsErr(err)).ShouldNot(HaveOccurred())

		By("Creating client for composite client role")
		_, err = keycloakApiClient.CreateClient(ctx, getKeyCloakToken(), KeycloakRealmCR, gocloak.Client{
			ClientID: gocloak.StringP("client1"),
		})
		Expect(adapter.SkipAlreadyExistsErr(err)).ShouldNot(HaveOccurred())

		By("Creating client role for composite role")
		cl, err := keycloakApiClient.GetClients(ctx, getKeyCloakToken(), KeycloakRealmCR, gocloak.GetClientsParams{
			ClientID: gocloak.StringP("client1"),
		})
		Expect(err).ShouldNot(HaveOccurred())
		Expect(cl).Should(HaveLen(1))

		_, err = keycloakApiClient.CreateClientRole(ctx, getKeyCloakToken(), KeycloakRealmCR, *cl[0].ID, gocloak.Role{
			Name: gocloak.StringP("client-role1"),
		})
		Expect(adapter.SkipAlreadyExistsErr(err)).ShouldNot(HaveOccurred())

		_, err = keycloakApiClient.CreateClientRole(ctx, getKeyCloakToken(), KeycloakRealmCR, *cl[0].ID, gocloak.Role{
			Name: gocloak.StringP("client-role2"),
		})
		Expect(adapter.SkipAlreadyExistsErr(err)).ShouldNot(HaveOccurred())

		By("Creating a KeycloakRealmRole")
		role := &keycloakApi.KeycloakRealmRole{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-keycloak-realm-composite-role",
				Namespace: ns,
			},
			Spec: keycloakApi.KeycloakRealmRoleSpec{
				Name: "test-keycloak-realm-composite-role",
				RealmRef: common.RealmRef{
					Kind: keycloakApi.KeycloakRealmKind,
					Name: KeycloakRealmCR,
				},
				Description: "test composite role",
				Attributes: map[string][]string{
					"test": {"test"},
				},
				Composite: true,
				Composites: []keycloakApi.Composite{
					{Name: "role1"},
					{Name: "role2"},
				},
				CompositesClientRoles: map[string][]keycloakApi.Composite{
					"client1": {
						{Name: "client-role1"},
						{Name: "client-role2"},
					},
				},
				IsDefault: true,
			},
		}
		Expect(k8sClient.Create(ctx, role)).Should(Succeed())
		Eventually(func() bool {
			createdRole := &keycloakApi.KeycloakRealmRole{}
			if err = k8sClient.Get(ctx, types.NamespacedName{Name: role.Name, Namespace: ns}, createdRole); err != nil {
				return false
			}

			return createdRole.Status.Value == helper.StatusOK
		}, timeout, interval).Should(BeTrue())

		By("Checking composite role")
		realmRole, err := keycloakApiClient.GetRealmRole(ctx, getKeyCloakToken(), KeycloakRealmCR, "test-keycloak-realm-composite-role")
		Expect(err).ShouldNot(HaveOccurred())

		roles, err := keycloakApiClient.GetCompositeRolesByRoleID(ctx, getKeyCloakToken(), KeycloakRealmCR, *realmRole.ID)
		Expect(err).ShouldNot(HaveOccurred())
		Expect(roles).Should(HaveLen(4))

		By("Updating composite role")
		role = &keycloakApi.KeycloakRealmRole{}
		Expect(k8sClient.Get(ctx, types.NamespacedName{Namespace: ns, Name: "test-keycloak-realm-composite-role"}, role)).Should(Succeed())

		role.Spec.Composites = []keycloakApi.Composite{
			{Name: "role1"},
		}
		role.Spec.CompositesClientRoles = map[string][]keycloakApi.Composite{
			"client1": {
				{Name: "client-role1"},
			},
		}

		Expect(k8sClient.Update(ctx, role)).Should(Succeed())
		Consistently(func() bool {
			updatedRole := &keycloakApi.KeycloakRealmRole{}
			if err = k8sClient.Get(ctx, types.NamespacedName{Name: role.Name, Namespace: ns}, updatedRole); err != nil {
				return false
			}

			return updatedRole.Status.Value == helper.StatusOK
		}, time.Second*3, time.Second).Should(BeTrue())

		By("Checking updated composite role")
		updatedRoles, err := keycloakApiClient.GetCompositeRolesByRoleID(ctx, getKeyCloakToken(), KeycloakRealmCR, *realmRole.ID)
		Expect(err).ShouldNot(HaveOccurred())
		Expect(updatedRoles).Should(HaveLen(2))
	})

	It("Should fail with invalid realm", func() {
		By("By creating a KeycloakRealmRole")
		role := &keycloakApi.KeycloakRealmRole{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-keycloak-realm-role-invalid-realm",
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

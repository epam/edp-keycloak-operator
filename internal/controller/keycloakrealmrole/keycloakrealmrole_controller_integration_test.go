package keycloakrealmrole

import (
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	k8sErrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/utils/ptr"

	"github.com/epam/edp-keycloak-operator/api/common"
	keycloakApi "github.com/epam/edp-keycloak-operator/api/v1"
	keycloakv2 "github.com/epam/edp-keycloak-operator/pkg/client/keycloakv2"
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

			return createdRole.Status.Value == common.StatusOK
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

			return updatedRole.Spec.Description == "updated description" && updatedRole.Status.Value == common.StatusOK
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
		_, err := keycloakApiClient.Roles.CreateRealmRole(ctx, KeycloakRealmCR, keycloakv2.RoleRepresentation{
			Name: ptr.To("role1"),
		})
		if err != nil && !keycloakv2.IsConflict(err) {
			Expect(err).ShouldNot(HaveOccurred())
		}

		_, err = keycloakApiClient.Roles.CreateRealmRole(ctx, KeycloakRealmCR, keycloakv2.RoleRepresentation{
			Name: ptr.To("role2"),
		})
		if err != nil && !keycloakv2.IsConflict(err) {
			Expect(err).ShouldNot(HaveOccurred())
		}

		By("Creating client for composite client role")
		_, err = keycloakApiClient.Clients.CreateClient(ctx, KeycloakRealmCR, keycloakv2.ClientRepresentation{
			ClientId: ptr.To("client1"),
		})
		if err != nil && !keycloakv2.IsConflict(err) {
			Expect(err).ShouldNot(HaveOccurred())
		}

		By("Creating client role for composite role")
		cl, _, err := keycloakApiClient.Clients.GetClients(ctx, KeycloakRealmCR, &keycloakv2.GetClientsParams{
			ClientId: ptr.To("client1"),
		})
		Expect(err).ShouldNot(HaveOccurred())
		Expect(cl).Should(HaveLen(1))

		clientUUID := *cl[0].Id

		_, err = keycloakApiClient.Clients.CreateClientRole(ctx, KeycloakRealmCR, clientUUID, keycloakv2.RoleRepresentation{
			Name: ptr.To("client-role1"),
		})
		if err != nil && !keycloakv2.IsConflict(err) {
			Expect(err).ShouldNot(HaveOccurred())
		}

		_, err = keycloakApiClient.Clients.CreateClientRole(ctx, KeycloakRealmCR, clientUUID, keycloakv2.RoleRepresentation{
			Name: ptr.To("client-role2"),
		})
		if err != nil && !keycloakv2.IsConflict(err) {
			Expect(err).ShouldNot(HaveOccurred())
		}

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

			return createdRole.Status.Value == common.StatusOK
		}, timeout, interval).Should(BeTrue())

		By("Checking composite role")
		roles, _, err := keycloakApiClient.Roles.GetRealmRoleComposites(ctx, KeycloakRealmCR, "test-keycloak-realm-composite-role")
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

			return updatedRole.Status.Value == common.StatusOK
		}, time.Second*3, time.Second).Should(BeTrue())

		By("Checking updated composite role")
		updatedRoles, _, err := keycloakApiClient.Roles.GetRealmRoleComposites(ctx, KeycloakRealmCR, "test-keycloak-realm-composite-role")
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

			return createdRole.Status.Value != common.StatusOK
		}, timeout, interval).Should(BeTrue(), "KeycloakRealmRole should be in failed state")
	})
})

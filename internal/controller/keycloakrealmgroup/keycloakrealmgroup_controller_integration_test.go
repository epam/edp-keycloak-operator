package keycloakrealmgroup

import (
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gstruct"

	k8sErrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	"github.com/epam/edp-keycloak-operator/api/common"
	keycloakApi "github.com/epam/edp-keycloak-operator/api/v1"
	keycloakv2 "github.com/epam/edp-keycloak-operator/pkg/client/keycloakv2"
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
		roleName := "test-group-role"
		_, err := keycloakApiClient.Roles.CreateRealmRole(ctx, KeycloakRealmCR, keycloakv2.RoleRepresentation{
			Name: &roleName,
		})
		if err != nil && !keycloakv2.IsConflict(err) {
			Expect(err).ShouldNot(HaveOccurred())
		}

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

		By("Verifying parent group was created in Keycloak with all parameters")
		Eventually(func(g Gomega) {
			parentGroupFromK8s := &keycloakApi.KeycloakRealmGroup{}
			err := k8sClient.Get(ctx, types.NamespacedName{Name: parentGroupCR, Namespace: ns}, parentGroupFromK8s)
			g.Expect(err).ShouldNot(HaveOccurred())

			groupRep, _, err := keycloakApiClient.Groups.GetGroup(ctx, KeycloakRealmCR, parentGroupFromK8s.Status.ID)
			g.Expect(err).ShouldNot(HaveOccurred())
			g.Expect(groupRep).ShouldNot(BeNil())
			g.Expect(groupRep.Name).ShouldNot(BeNil())
			g.Expect(*groupRep.Name).Should(Equal(parentGroup.Spec.Name))
			g.Expect(groupRep.Path).ShouldNot(BeNil())
			g.Expect(*groupRep.Path).Should(Equal(parentGroup.Spec.Path))
			g.Expect(groupRep.RealmRoles).ShouldNot(BeNil())
			g.Expect(*groupRep.RealmRoles).Should(ContainElement("test-group-role"))
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

		By("Verifying child group was created in Keycloak with parent relationship")
		Eventually(func(g Gomega) {
			parentGroupFromK8s := &keycloakApi.KeycloakRealmGroup{}
			err := k8sClient.Get(ctx, types.NamespacedName{Name: parentGroupCR, Namespace: ns}, parentGroupFromK8s)
			g.Expect(err).ShouldNot(HaveOccurred())

			childGroupFromK8s := &keycloakApi.KeycloakRealmGroup{}
			err = k8sClient.Get(ctx, types.NamespacedName{Name: childGroupCR, Namespace: ns}, childGroupFromK8s)
			g.Expect(err).ShouldNot(HaveOccurred())

			groupRep, _, err := keycloakApiClient.Groups.GetGroup(ctx, KeycloakRealmCR, childGroupFromK8s.Status.ID)
			g.Expect(err).ShouldNot(HaveOccurred())
			g.Expect(groupRep).ShouldNot(BeNil())
			g.Expect(groupRep.Name).ShouldNot(BeNil())
			g.Expect(*groupRep.Name).Should(Equal(childGroup.Spec.Name))
			g.Expect(groupRep.ParentId).ShouldNot(BeNil())
			g.Expect(*groupRep.ParentId).Should(Equal(parentGroupFromK8s.Status.ID))
			g.Expect(groupRep.Path).ShouldNot(BeNil())
			g.Expect(*groupRep.Path).Should(Equal("/parent-group/child-group"))
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

		gr, _, err := keycloakApiClient.Groups.GetGroups(ctx, KeycloakRealmCR, nil)
		Expect(err).ShouldNot(HaveOccurred())
		// Check if the parent group is in the list (top-level groups only).
		// childGroup and childGroup2 should not be in the list since they are children.
		Expect(gr).Should(ContainElement(HaveField("Name", PointTo(Equal(parentGroup.Spec.Name)))))
		Expect(gr).ShouldNot(ContainElement(HaveField("Name", PointTo(Equal(childGroup.Spec.Name)))))
		Expect(gr).ShouldNot(ContainElement(HaveField("Name", PointTo(Equal(childGroup2.Spec.Name)))))
	})
	It("Should create and update KeycloakRealmGroup with all parameters", func() {
		By("Creating a KeycloakRealmGroup with all parameters")
		group := &keycloakApi.KeycloakRealmGroup{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-group-all-params",
				Namespace: ns,
			},
			Spec: keycloakApi.KeycloakRealmGroupSpec{
				Name: "group-all-params",
				RealmRef: common.RealmRef{
					Kind: keycloakApi.KeycloakRealmKind,
					Name: KeycloakRealmCR,
				},
				Path: "/group-all-params",
				Attributes: map[string][]string{
					"attr1": {"value1", "value2"},
					"attr2": {"value3"},
				},
				RealmRoles: []string{"test-group-role"},
			},
		}
		Expect(k8sClient.Create(ctx, group)).Should(Succeed())
		Eventually(func(g Gomega) {
			createdGroup := &keycloakApi.KeycloakRealmGroup{}
			err := k8sClient.Get(ctx, types.NamespacedName{Name: group.Name, Namespace: ns}, createdGroup)
			g.Expect(err).ShouldNot(HaveOccurred())
			g.Expect(createdGroup.Status.Value).Should(Equal(common.StatusOK))
			g.Expect(createdGroup.Status.ID).ShouldNot(BeEmpty())
		}).WithTimeout(time.Second * 20).WithPolling(time.Second).Should(Succeed())

		By("Verifying group was created in Keycloak with all parameters")
		Eventually(func(g Gomega) {
			groupFromK8s := &keycloakApi.KeycloakRealmGroup{}
			err := k8sClient.Get(ctx, types.NamespacedName{Name: group.Name, Namespace: ns}, groupFromK8s)
			g.Expect(err).ShouldNot(HaveOccurred())

			groupRep, _, err := keycloakApiClient.Groups.GetGroup(ctx, KeycloakRealmCR, groupFromK8s.Status.ID)
			g.Expect(err).ShouldNot(HaveOccurred())
			g.Expect(groupRep).ShouldNot(BeNil())

			// Verify name
			g.Expect(groupRep.Name).ShouldNot(BeNil())
			g.Expect(*groupRep.Name).Should(Equal(group.Spec.Name))

			// Verify path
			g.Expect(groupRep.Path).ShouldNot(BeNil())
			g.Expect(*groupRep.Path).Should(Equal(group.Spec.Path))

			// Verify attributes
			g.Expect(groupRep.Attributes).ShouldNot(BeNil())
			g.Expect(*groupRep.Attributes).Should(Equal(group.Spec.Attributes))

			// Note: groupRep.Access is server-computed (current user's permissions on the group),
			// not the value sent in create/update. Keycloak ignores access on write and always
			// returns its own computed map on read, so we skip verifying it here.

			// Verify realm roles
			g.Expect(groupRep.RealmRoles).ShouldNot(BeNil())
			g.Expect(*groupRep.RealmRoles).Should(ContainElement("test-group-role"))
		}, time.Minute, time.Second*5).Should(Succeed())

		By("Creating additional role for update test")
		role2Name := "test-group-role-2"
		_, err := keycloakApiClient.Roles.CreateRealmRole(ctx, KeycloakRealmCR, keycloakv2.RoleRepresentation{
			Name: &role2Name,
		})
		if err != nil && !keycloakv2.IsConflict(err) {
			Expect(err).ShouldNot(HaveOccurred())
		}

		By("Updating KeycloakRealmGroup parameters")
		updatableGroup := &keycloakApi.KeycloakRealmGroup{}
		Expect(k8sClient.Get(ctx, types.NamespacedName{Name: group.Name, Namespace: ns}, updatableGroup)).Should(Succeed())
		updatableGroup.Spec.Attributes = map[string][]string{
			"updated-attr": {"updated-value"},
			"new-attr":     {"new-value"},
		}
		updatableGroup.Spec.RealmRoles = []string{"test-group-role-2"}
		Expect(k8sClient.Update(ctx, updatableGroup)).Should(Succeed())

		Eventually(func(g Gomega) {
			updatedGroup := &keycloakApi.KeycloakRealmGroup{}
			err := k8sClient.Get(ctx, types.NamespacedName{Name: group.Name, Namespace: ns}, updatedGroup)
			g.Expect(err).ShouldNot(HaveOccurred())
			g.Expect(updatedGroup.Status.Value).Should(Equal(common.StatusOK))
		}, time.Minute, time.Second*5).Should(Succeed())

		By("Verifying updates were applied in Keycloak")
		Eventually(func(g Gomega) {
			groupFromK8s := &keycloakApi.KeycloakRealmGroup{}
			err := k8sClient.Get(ctx, types.NamespacedName{Name: group.Name, Namespace: ns}, groupFromK8s)
			g.Expect(err).ShouldNot(HaveOccurred())

			groupRep, _, err := keycloakApiClient.Groups.GetGroup(ctx, KeycloakRealmCR, groupFromK8s.Status.ID)
			g.Expect(err).ShouldNot(HaveOccurred())
			g.Expect(groupRep).ShouldNot(BeNil())

			// Verify attributes were updated
			g.Expect(groupRep.Attributes).ShouldNot(BeNil())
			g.Expect(*groupRep.Attributes).Should(HaveKeyWithValue("updated-attr", []string{"updated-value"}))
			g.Expect(*groupRep.Attributes).Should(HaveKeyWithValue("new-attr", []string{"new-value"}))
			g.Expect(*groupRep.Attributes).ShouldNot(HaveKey("attr1"))
			g.Expect(*groupRep.Attributes).ShouldNot(HaveKey("attr2"))

			// Note: groupRep.Access is server-computed; skip verification (see comment above).

			// Verify realm roles were updated
			g.Expect(groupRep.RealmRoles).ShouldNot(BeNil())
			g.Expect(*groupRep.RealmRoles).Should(ContainElement("test-group-role-2"))
			g.Expect(*groupRep.RealmRoles).ShouldNot(ContainElement("test-group-role"))
		}, time.Minute, time.Second*5).Should(Succeed())

		By("Cleaning up")
		Expect(k8sClient.Delete(ctx, group)).Should(Succeed())
	})
	It("Should update KeycloakRealmGroup realm roles", func() {
		By("Creating additional roles for testing")
		role2Name := "test-group-role-2"
		_, err := keycloakApiClient.Roles.CreateRealmRole(ctx, KeycloakRealmCR, keycloakv2.RoleRepresentation{
			Name: &role2Name,
		})
		if err != nil && !keycloakv2.IsConflict(err) {
			Expect(err).ShouldNot(HaveOccurred())
		}

		role3Name := "test-group-role-3"
		_, err = keycloakApiClient.Roles.CreateRealmRole(ctx, KeycloakRealmCR, keycloakv2.RoleRepresentation{
			Name: &role3Name,
		})
		if err != nil && !keycloakv2.IsConflict(err) {
			Expect(err).ShouldNot(HaveOccurred())
		}

		By("Creating a KeycloakRealmGroup with initial roles")
		group := &keycloakApi.KeycloakRealmGroup{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-group-for-role-update",
				Namespace: ns,
			},
			Spec: keycloakApi.KeycloakRealmGroupSpec{
				Name: "group-for-role-update",
				RealmRef: common.RealmRef{
					Kind: keycloakApi.KeycloakRealmKind,
					Name: KeycloakRealmCR,
				},
				RealmRoles: []string{"test-group-role", "test-group-role-2"},
			},
		}
		Expect(k8sClient.Create(ctx, group)).Should(Succeed())
		Eventually(func(g Gomega) {
			createdGroup := &keycloakApi.KeycloakRealmGroup{}
			err := k8sClient.Get(ctx, types.NamespacedName{Name: group.Name, Namespace: ns}, createdGroup)
			g.Expect(err).ShouldNot(HaveOccurred())
			g.Expect(createdGroup.Status.Value).Should(Equal(common.StatusOK))
		}).WithTimeout(time.Second * 20).WithPolling(time.Second).Should(Succeed())

		By("Verifying initial roles in Keycloak")
		Eventually(func(g Gomega) {
			groupFromK8s := &keycloakApi.KeycloakRealmGroup{}
			err := k8sClient.Get(ctx, types.NamespacedName{Name: group.Name, Namespace: ns}, groupFromK8s)
			g.Expect(err).ShouldNot(HaveOccurred())

			groupRep, _, err := keycloakApiClient.Groups.GetGroup(ctx, KeycloakRealmCR, groupFromK8s.Status.ID)
			g.Expect(err).ShouldNot(HaveOccurred())
			g.Expect(groupRep.RealmRoles).ShouldNot(BeNil())
			g.Expect(*groupRep.RealmRoles).Should(ContainElement("test-group-role"))
			g.Expect(*groupRep.RealmRoles).Should(ContainElement("test-group-role-2"))
		}, time.Minute, time.Second*5).Should(Succeed())

		By("Updating KeycloakRealmGroup roles")
		updatableGroup := &keycloakApi.KeycloakRealmGroup{}
		Expect(k8sClient.Get(ctx, types.NamespacedName{Name: group.Name, Namespace: ns}, updatableGroup)).Should(Succeed())
		updatableGroup.Spec.RealmRoles = []string{"test-group-role-2", "test-group-role-3"}
		Expect(k8sClient.Update(ctx, updatableGroup)).Should(Succeed())

		Eventually(func(g Gomega) {
			updatedGroup := &keycloakApi.KeycloakRealmGroup{}
			err := k8sClient.Get(ctx, types.NamespacedName{Name: group.Name, Namespace: ns}, updatedGroup)
			g.Expect(err).ShouldNot(HaveOccurred())
			g.Expect(updatedGroup.Status.Value).Should(Equal(common.StatusOK))
		}, time.Minute, time.Second*5).Should(Succeed())

		By("Verifying role updates in Keycloak")
		Eventually(func(g Gomega) {
			groupFromK8s := &keycloakApi.KeycloakRealmGroup{}
			err := k8sClient.Get(ctx, types.NamespacedName{Name: group.Name, Namespace: ns}, groupFromK8s)
			g.Expect(err).ShouldNot(HaveOccurred())

			groupRep, _, err := keycloakApiClient.Groups.GetGroup(ctx, KeycloakRealmCR, groupFromK8s.Status.ID)
			g.Expect(err).ShouldNot(HaveOccurred())
			g.Expect(groupRep.RealmRoles).ShouldNot(BeNil())
			// Verify role-2 and role-3 are present
			g.Expect(*groupRep.RealmRoles).Should(ContainElement("test-group-role-2"))
			g.Expect(*groupRep.RealmRoles).Should(ContainElement("test-group-role-3"))
			// Verify test-group-role was removed
			g.Expect(*groupRep.RealmRoles).ShouldNot(ContainElement("test-group-role"))
		}, time.Minute, time.Second*5).Should(Succeed())

		By("Cleaning up")
		Expect(k8sClient.Delete(ctx, group)).Should(Succeed())
	})
	It("Should create and update KeycloakRealmGroup with client roles", func() {
		By("Creating a test client with roles")
		clientID := "test-client-for-group-roles"
		clientName := "Test Client for Group Roles"
		_, err := keycloakApiClient.Clients.CreateClient(ctx, KeycloakRealmCR, keycloakv2.ClientRepresentation{
			ClientId: &clientID,
			Name:     &clientName,
		})
		if err != nil && !keycloakv2.IsConflict(err) {
			Expect(err).ShouldNot(HaveOccurred())
		}

		By("Getting client UUID")
		var testClientUUID string
		Eventually(func(g Gomega) {
			clients, _, err := keycloakApiClient.Clients.GetClients(ctx, KeycloakRealmCR, &keycloakv2.GetClientsParams{
				ClientId: &clientID,
			})
			g.Expect(err).ShouldNot(HaveOccurred())
			g.Expect(clients).Should(HaveLen(1))
			g.Expect(clients[0].Id).ShouldNot(BeNil())
			testClientUUID = *clients[0].Id
		}, time.Second*10, time.Second).Should(Succeed())

		By("Creating client roles")
		role1Name := "group-client-role-1"
		role2Name := "group-client-role-2"
		role3Name := "group-client-role-3"
		_, err = keycloakApiClient.Clients.CreateClientRole(ctx, KeycloakRealmCR, testClientUUID, keycloakv2.RoleRepresentation{
			Name: &role1Name,
		})
		if err != nil && !keycloakv2.IsConflict(err) {
			Expect(err).ShouldNot(HaveOccurred())
		}
		_, err = keycloakApiClient.Clients.CreateClientRole(ctx, KeycloakRealmCR, testClientUUID, keycloakv2.RoleRepresentation{
			Name: &role2Name,
		})
		if err != nil && !keycloakv2.IsConflict(err) {
			Expect(err).ShouldNot(HaveOccurred())
		}
		_, err = keycloakApiClient.Clients.CreateClientRole(ctx, KeycloakRealmCR, testClientUUID, keycloakv2.RoleRepresentation{
			Name: &role3Name,
		})
		if err != nil && !keycloakv2.IsConflict(err) {
			Expect(err).ShouldNot(HaveOccurred())
		}

		By("Creating a KeycloakRealmGroup with client roles")
		group := &keycloakApi.KeycloakRealmGroup{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-group-with-client-roles",
				Namespace: ns,
			},
			Spec: keycloakApi.KeycloakRealmGroupSpec{
				Name: "group-with-client-roles",
				RealmRef: common.RealmRef{
					Kind: keycloakApi.KeycloakRealmKind,
					Name: KeycloakRealmCR,
				},
				ClientRoles: []keycloakApi.UserClientRole{
					{
						ClientID: clientID,
						Roles:    []string{"group-client-role-1", "group-client-role-2"},
					},
				},
			},
		}
		Expect(k8sClient.Create(ctx, group)).Should(Succeed())
		Eventually(func(g Gomega) {
			createdGroup := &keycloakApi.KeycloakRealmGroup{}
			err := k8sClient.Get(ctx, types.NamespacedName{Name: group.Name, Namespace: ns}, createdGroup)
			g.Expect(err).ShouldNot(HaveOccurred())
			g.Expect(createdGroup.Status.Value).Should(Equal(common.StatusOK))
			g.Expect(createdGroup.Status.ID).ShouldNot(BeEmpty())
		}).WithTimeout(time.Second * 20).WithPolling(time.Second).Should(Succeed())

		By("Verifying client roles were assigned in Keycloak")
		Eventually(func(g Gomega) {
			groupFromK8s := &keycloakApi.KeycloakRealmGroup{}
			err := k8sClient.Get(ctx, types.NamespacedName{Name: group.Name, Namespace: ns}, groupFromK8s)
			g.Expect(err).ShouldNot(HaveOccurred())

			clientRoles, _, err := keycloakApiClient.Groups.GetClientRoleMappings(ctx, KeycloakRealmCR, groupFromK8s.Status.ID, testClientUUID)
			g.Expect(err).ShouldNot(HaveOccurred())
			g.Expect(clientRoles).Should(HaveLen(2))

			roleNames := make([]string, 0)
			for _, role := range clientRoles {
				if role.Name != nil {
					roleNames = append(roleNames, *role.Name)
				}
			}
			g.Expect(roleNames).Should(ContainElement("group-client-role-1"))
			g.Expect(roleNames).Should(ContainElement("group-client-role-2"))
		}, time.Minute, time.Second*5).Should(Succeed())

		By("Updating KeycloakRealmGroup client roles")
		updatableGroup := &keycloakApi.KeycloakRealmGroup{}
		Expect(k8sClient.Get(ctx, types.NamespacedName{Name: group.Name, Namespace: ns}, updatableGroup)).Should(Succeed())
		updatableGroup.Spec.ClientRoles = []keycloakApi.UserClientRole{
			{
				ClientID: clientID,
				Roles:    []string{"group-client-role-2", "group-client-role-3"},
			},
		}
		Expect(k8sClient.Update(ctx, updatableGroup)).Should(Succeed())

		Eventually(func(g Gomega) {
			updatedGroup := &keycloakApi.KeycloakRealmGroup{}
			err := k8sClient.Get(ctx, types.NamespacedName{Name: group.Name, Namespace: ns}, updatedGroup)
			g.Expect(err).ShouldNot(HaveOccurred())
			g.Expect(updatedGroup.Status.Value).Should(Equal(common.StatusOK))
		}, time.Minute, time.Second*5).Should(Succeed())

		By("Verifying client role updates in Keycloak")
		Eventually(func(g Gomega) {
			groupFromK8s := &keycloakApi.KeycloakRealmGroup{}
			err := k8sClient.Get(ctx, types.NamespacedName{Name: group.Name, Namespace: ns}, groupFromK8s)
			g.Expect(err).ShouldNot(HaveOccurred())

			clientRoles, _, err := keycloakApiClient.Groups.GetClientRoleMappings(ctx, KeycloakRealmCR, groupFromK8s.Status.ID, testClientUUID)
			g.Expect(err).ShouldNot(HaveOccurred())
			g.Expect(clientRoles).Should(HaveLen(2))

			roleNames := make([]string, 0)
			for _, role := range clientRoles {
				if role.Name != nil {
					roleNames = append(roleNames, *role.Name)
				}
			}
			// Verify role-2 and role-3 are present
			g.Expect(roleNames).Should(ContainElement("group-client-role-2"))
			g.Expect(roleNames).Should(ContainElement("group-client-role-3"))
			// Verify role-1 was removed
			g.Expect(roleNames).ShouldNot(ContainElement("group-client-role-1"))
		}, time.Minute, time.Second*5).Should(Succeed())

		By("Cleaning up")
		Expect(k8sClient.Delete(ctx, group)).Should(Succeed())
		Eventually(func(g Gomega) {
			deletedGroup := &keycloakApi.KeycloakRealmGroup{}
			err := k8sClient.Get(ctx, types.NamespacedName{Name: group.Name, Namespace: ns}, deletedGroup)
			g.Expect(k8sErrors.IsNotFound(err)).Should(BeTrue())
		}, timeout, interval).Should(Succeed())

		_, err = keycloakApiClient.Clients.DeleteClient(ctx, KeycloakRealmCR, testClientUUID)
		Expect(err).ShouldNot(HaveOccurred())
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
			searchString := "test-group-preserve"
			groups, _, err := keycloakApiClient.Groups.GetGroups(ctx, KeycloakRealmCR, &keycloakv2.GetGroupsParams{
				Search: &searchString,
			})
			g.Expect(err).ShouldNot(HaveOccurred())
			g.Expect(groups).Should(HaveLen(1))
		}, time.Minute, time.Second*5).Should(Succeed())
	})
})

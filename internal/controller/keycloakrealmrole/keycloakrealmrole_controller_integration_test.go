package keycloakrealmrole

import (
	"fmt"
	"slices"
	"strings"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	k8sErrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/epam/edp-keycloak-operator/api/common"
	keycloakApi "github.com/epam/edp-keycloak-operator/api/v1"
	"github.com/epam/edp-keycloak-operator/pkg/client/keycloakapi"
	"github.com/epam/edp-keycloak-operator/pkg/testutils"
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
		_, err := keycloakApiClient.Roles.CreateRealmRole(ctx, KeycloakRealmCR, keycloakapi.RoleRepresentation{
			Name: ptr.To("role1"),
		})
		if err != nil && !keycloakapi.IsConflict(err) {
			Expect(err).ShouldNot(HaveOccurred())
		}

		_, err = keycloakApiClient.Roles.CreateRealmRole(ctx, KeycloakRealmCR, keycloakapi.RoleRepresentation{
			Name: ptr.To("role2"),
		})
		if err != nil && !keycloakapi.IsConflict(err) {
			Expect(err).ShouldNot(HaveOccurred())
		}

		By("Creating client for composite client role")
		_, err = keycloakApiClient.Clients.CreateClient(ctx, KeycloakRealmCR, keycloakapi.ClientRepresentation{
			ClientId: ptr.To("client1"),
		})
		if err != nil && !keycloakapi.IsConflict(err) {
			Expect(err).ShouldNot(HaveOccurred())
		}

		By("Creating client role for composite role")
		cl, _, err := keycloakApiClient.Clients.GetClients(ctx, KeycloakRealmCR, &keycloakapi.GetClientsParams{
			ClientId: ptr.To("client1"),
		})
		Expect(err).ShouldNot(HaveOccurred())
		Expect(cl).Should(HaveLen(1))

		clientUUID := *cl[0].Id

		_, err = keycloakApiClient.Clients.CreateClientRole(ctx, KeycloakRealmCR, clientUUID, keycloakapi.RoleRepresentation{
			Name: ptr.To("client-role1"),
		})
		if err != nil && !keycloakapi.IsConflict(err) {
			Expect(err).ShouldNot(HaveOccurred())
		}

		_, err = keycloakApiClient.Clients.CreateClientRole(ctx, KeycloakRealmCR, clientUUID, keycloakapi.RoleRepresentation{
			Name: ptr.To("client-role2"),
		})
		if err != nil && !keycloakapi.IsConflict(err) {
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

var _ = Describe("KeycloakRealmRole isDefault sync", func() {
	var (
		upperRealmCR   = testutils.RealmName("role-upper-realm")
		upperRealmName = "Upper-" + testutils.RealmName("role-upper")
		step           int
	)

	defaultRoleOf := func(g Gomega, realm string) string {
		kcRealm, _, err := keycloakApiClient.Realms.GetRealm(ctx, realm)
		g.Expect(err).ShouldNot(HaveOccurred())
		g.Expect(kcRealm).ShouldNot(BeNil())
		g.Expect(kcRealm.DefaultRole).ShouldNot(BeNil())

		return ptr.Deref(kcRealm.DefaultRole.Name, "")
	}

	isMember := func(g Gomega, realm, roleName string) bool {
		composites, _, err := keycloakApiClient.Roles.GetRealmRoleComposites(ctx, realm, defaultRoleOf(g, realm))
		g.Expect(err).ShouldNot(HaveOccurred())

		return slices.ContainsFunc(composites, func(r keycloakapi.RoleRepresentation) bool {
			return !ptr.Deref(r.ClientRole, false) && ptr.Deref(r.Name, "") == roleName
		})
	}

	newRole := func(crName, realmCR, roleName string, isDefault bool) *keycloakApi.KeycloakRealmRole {
		return &keycloakApi.KeycloakRealmRole{
			ObjectMeta: metav1.ObjectMeta{Name: crName, Namespace: ns},
			Spec: keycloakApi.KeycloakRealmRoleSpec{
				Name:      roleName,
				RealmRef:  common.RealmRef{Kind: keycloakApi.KeycloakRealmKind, Name: realmCR},
				IsDefault: isDefault,
			},
		}
	}

	waitStatusOK := func(crName string) {
		Eventually(func(g Gomega) {
			cr := &keycloakApi.KeycloakRealmRole{}
			g.Expect(k8sClient.Get(ctx, types.NamespacedName{Name: crName, Namespace: ns}, cr)).To(Succeed())
			g.Expect(cr.Status.Value).To(Equal(common.StatusOK))
		}, testutils.LongWait, interval).Should(Succeed())
	}

	waitGone := func(key types.NamespacedName, obj client.Object) {
		Eventually(func() bool {
			return k8sErrors.IsNotFound(k8sClient.Get(ctx, key, obj))
		}, testutils.LongWait, interval).Should(BeTrue())
	}

	// Description patch triggers a reconcile (generation change).
	patchSpec := func(crName string, mutate func(*keycloakApi.KeycloakRealmRole)) {
		cr := &keycloakApi.KeycloakRealmRole{}
		Expect(k8sClient.Get(ctx, types.NamespacedName{Name: crName, Namespace: ns}, cr)).To(Succeed())

		base := cr.DeepCopy()
		step++
		cr.Spec.Description = fmt.Sprintf("step %d", step)
		mutate(cr)

		Expect(k8sClient.Patch(ctx, cr, client.MergeFrom(base))).To(Succeed())
	}

	It("Should write the default role membership only on a difference", func() {
		crName := testutils.UniqueName("default-role")
		roleName := testutils.UniqueName("default-role")
		outsideRole := testutils.UniqueName("outside-role")

		var defaultRole string

		Eventually(func(g Gomega) {
			defaultRole = defaultRoleOf(g, KeycloakRealmCR)
		}, testutils.LongWait, interval).Should(Succeed())

		recorder := testutils.NewAdminEventRecorder(keycloakApiClient, KeycloakRealmCR).
			Scoped(defaultRole + "/composites")
		Expect(recorder.Enable(ctx)).To(Succeed())

		By("Adding a role that the operator does not own to the default role")
		_, err := keycloakApiClient.Roles.CreateRealmRole(ctx, KeycloakRealmCR,
			keycloakapi.RoleRepresentation{Name: ptr.To(outsideRole)})
		Expect(err).ShouldNot(HaveOccurred())
		DeferCleanup(func() {
			_, delErr := keycloakApiClient.Roles.DeleteRealmRole(ctx, KeycloakRealmCR, outsideRole)
			if delErr != nil {
				Expect(keycloakapi.IsNotFound(delErr)).To(BeTrue())
			}
		})

		outside, _, err := keycloakApiClient.Roles.GetRealmRole(ctx, KeycloakRealmCR, outsideRole)
		Expect(err).ShouldNot(HaveOccurred())
		_, err = keycloakApiClient.Roles.AddRealmRoleComposites(ctx, KeycloakRealmCR, defaultRole,
			[]keycloakapi.RoleRepresentation{{Id: outside.Id, Name: outside.Name}})
		Expect(err).ShouldNot(HaveOccurred())

		By("Creating a role with isDefault true")
		Expect(k8sClient.Create(ctx, newRole(crName, KeycloakRealmCR, roleName, true))).To(Succeed())
		DeferCleanup(func() {
			Expect(client.IgnoreNotFound(k8sClient.Delete(ctx, newRole(crName, KeycloakRealmCR, roleName, true)))).To(Succeed())
			waitGone(types.NamespacedName{Name: crName, Namespace: ns}, &keycloakApi.KeycloakRealmRole{})
		})
		waitStatusOK(crName)
		Eventually(func(g Gomega) {
			g.Expect(isMember(g, KeycloakRealmCR, roleName)).To(BeTrue())
		}, testutils.LongWait, interval).Should(Succeed())

		By("Reconciling a settled role writes nothing")
		const settledDescription = "settled"

		events, err := recorder.WritesDuring(ctx, testutils.Settle, func() {
			patchSpec(crName, func(r *keycloakApi.KeycloakRealmRole) { r.Spec.Description = settledDescription })
			Eventually(func(g Gomega) {
				kcRole, _, getErr := keycloakApiClient.Roles.GetRealmRole(ctx, KeycloakRealmCR, roleName)
				g.Expect(getErr).ShouldNot(HaveOccurred())
				g.Expect(ptr.Deref(kcRole.Description, "")).To(Equal(settledDescription))
			}, testutils.LongWait, interval).Should(Succeed())
		})
		Expect(err).ShouldNot(HaveOccurred())
		Expect(events).To(BeEmpty())

		By("Setting isDefault false removes the membership with one DELETE")
		events, err = recorder.WritesDuring(ctx, testutils.Settle, func() {
			patchSpec(crName, func(r *keycloakApi.KeycloakRealmRole) { r.Spec.IsDefault = false })
			Eventually(func(g Gomega) {
				g.Expect(isMember(g, KeycloakRealmCR, roleName)).To(BeFalse())
			}, testutils.LongWait, interval).Should(Succeed())
		})
		Expect(err).ShouldNot(HaveOccurred())
		Expect(events).To(HaveLen(1), "recorded: %s", events.GomegaString())
		Expect(ptr.Deref(events[0].OperationType, "")).To(Equal("DELETE"))

		By("Setting isDefault true adds the membership with one CREATE")
		events, err = recorder.WritesDuring(ctx, testutils.Settle, func() {
			patchSpec(crName, func(r *keycloakApi.KeycloakRealmRole) { r.Spec.IsDefault = true })
			Eventually(func(g Gomega) {
				g.Expect(isMember(g, KeycloakRealmCR, roleName)).To(BeTrue())
			}, testutils.LongWait, interval).Should(Succeed())
		})
		Expect(err).ShouldNot(HaveOccurred())
		Expect(events).To(HaveLen(1), "recorded: %s", events.GomegaString())
		Expect(ptr.Deref(events[0].OperationType, "")).To(Equal("CREATE"))

		By("Leaving the other members of the default role untouched")
		Eventually(func(g Gomega) {
			g.Expect(isMember(g, KeycloakRealmCR, outsideRole)).To(BeTrue())
		}, testutils.LongWait, interval).Should(Succeed())
	})

	It("Should resolve the default role of a realm with an upper-case name", func() {
		Expect(strings.ToLower(upperRealmName)).NotTo(Equal(upperRealmName))

		realmCR := &keycloakApi.KeycloakRealm{
			ObjectMeta: metav1.ObjectMeta{Name: upperRealmCR, Namespace: ns},
			Spec: keycloakApi.KeycloakRealmSpec{
				RealmName:   upperRealmName,
				KeycloakRef: common.KeycloakRef{Name: KeycloakCR},
			},
		}
		Expect(k8sClient.Create(ctx, realmCR)).To(Succeed())

		crName := testutils.UniqueName("default-role-upper")
		roleName := testutils.UniqueName("default-role-upper")
		roleCR := newRole(crName, upperRealmCR, roleName, true)

		DeferCleanup(func() {
			Expect(client.IgnoreNotFound(k8sClient.Delete(ctx, roleCR))).To(Succeed())
			waitGone(types.NamespacedName{Name: crName, Namespace: ns}, &keycloakApi.KeycloakRealmRole{})

			Expect(client.IgnoreNotFound(k8sClient.Delete(ctx, realmCR))).To(Succeed())
			waitGone(types.NamespacedName{Name: upperRealmCR, Namespace: ns}, &keycloakApi.KeycloakRealm{})
		})

		Eventually(func(g Gomega) {
			created := &keycloakApi.KeycloakRealm{}
			g.Expect(k8sClient.Get(ctx, types.NamespacedName{Name: upperRealmCR, Namespace: ns}, created)).To(Succeed())
			g.Expect(created.Status.Available).To(BeTrue())
		}, testutils.LongWait, interval).Should(Succeed())

		Expect(k8sClient.Create(ctx, roleCR)).To(Succeed())
		waitStatusOK(crName)
		Eventually(func(g Gomega) {
			g.Expect(defaultRoleOf(g, upperRealmName)).To(Equal("default-roles-" + strings.ToLower(upperRealmName)))
			g.Expect(isMember(g, upperRealmName, roleName)).To(BeTrue())
		}, testutils.LongWait, interval).Should(Succeed())
	})
})

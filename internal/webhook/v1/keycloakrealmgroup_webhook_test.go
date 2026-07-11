package v1

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/epam/edp-keycloak-operator/api/common"
	keycloakApi "github.com/epam/edp-keycloak-operator/api/v1"
	keycloakAlpha "github.com/epam/edp-keycloak-operator/api/v1alpha1"
)

func newRealmGroup(ns, name, groupName string, mutate ...func(*keycloakApi.KeycloakRealmGroup)) *keycloakApi.KeycloakRealmGroup {
	g := &keycloakApi.KeycloakRealmGroup{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: ns,
		},
		Spec: keycloakApi.KeycloakRealmGroupSpec{
			Name: groupName,
			RealmRef: common.RealmRef{
				Kind: keycloakApi.KeycloakRealmKind,
				Name: "test-realm",
			},
		},
	}

	for _, m := range mutate {
		m(g)
	}

	return g
}

func withParentGroup(name string) func(*keycloakApi.KeycloakRealmGroup) {
	return func(g *keycloakApi.KeycloakRealmGroup) {
		g.Spec.ParentGroup = &common.GroupRef{Name: name}
	}
}

func withClusterRealm(name string) func(*keycloakApi.KeycloakRealmGroup) {
	return func(g *keycloakApi.KeycloakRealmGroup) {
		g.Spec.RealmRef = common.RealmRef{
			Kind: keycloakAlpha.ClusterKeycloakRealmKind,
			Name: name,
		}
	}
}

var _ = Describe("KeycloakRealmGroup Webhook", func() {
	cleanup := func(groups ...*keycloakApi.KeycloakRealmGroup) {
		for _, g := range groups {
			Expect(client.IgnoreNotFound(k8sClient.Delete(ctx, g))).Should(Succeed())
		}
	}

	Context("When creating KeycloakRealmGroup", func() {
		It("Should deny a second top-level group with the same name for the same realm in the same namespace", func() {
			existing := newRealmGroup("ns1", "group-one", "developers")
			Expect(k8sClient.Create(ctx, existing)).Should(Succeed())
			defer cleanup(existing)

			duplicate := newRealmGroup("ns1", "group-two", "developers")

			err := k8sClient.Create(ctx, duplicate)
			Expect(err).Should(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring(
				`group name "developers" is already used by KeycloakRealmGroup ns1/group-one (top-level in realm KeycloakRealm/ns1/test-realm)`,
			))
		})

		It("Should allow the same group name for the same realm name in another namespace (namespaced realm kind)", func() {
			// KeycloakRealm refs resolve in the resource's own namespace, so the same
			// realm ref name in ns2 targets a different realm than in ns1.
			existing := newRealmGroup("ns1", "group-one", "developers")
			Expect(k8sClient.Create(ctx, existing)).Should(Succeed())
			defer cleanup(existing)

			other := newRealmGroup("ns2", "group-two", "developers")
			Expect(k8sClient.Create(ctx, other)).Should(Succeed())
			cleanup(other)
		})

		It("Should deny the same top-level group name across namespaces for the same ClusterKeycloakRealm", func() {
			existing := newRealmGroup("ns1", "cluster-group-one", "developers", withClusterRealm("shared-realm"))
			Expect(k8sClient.Create(ctx, existing)).Should(Succeed())
			defer cleanup(existing)

			duplicate := newRealmGroup("ns2", "cluster-group-two", "developers", withClusterRealm("shared-realm"))

			err := k8sClient.Create(ctx, duplicate)
			Expect(err).Should(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring(
				`group name "developers" is already used by KeycloakRealmGroup ns1/cluster-group-one (top-level in realm ClusterKeycloakRealm/shared-realm)`,
			))
		})

		It("Should allow the same group name under different parent groups", func() {
			parentA := newRealmGroup("ns1", "parent-a", "ParentA")
			parentB := newRealmGroup("ns1", "parent-b", "ParentB")
			rolesA := newRealmGroup("ns1", "roles-a", "Roles", withParentGroup("parent-a"))
			rolesB := newRealmGroup("ns1", "roles-b", "Roles", withParentGroup("parent-b"))

			Expect(k8sClient.Create(ctx, parentA)).Should(Succeed())
			Expect(k8sClient.Create(ctx, parentB)).Should(Succeed())
			Expect(k8sClient.Create(ctx, rolesA)).Should(Succeed())
			Expect(k8sClient.Create(ctx, rolesB)).Should(Succeed())
			cleanup(rolesB, rolesA, parentB, parentA)
		})

		It("Should allow the same group name for a top-level group and a child group", func() {
			parent := newRealmGroup("ns1", "parent-a", "ParentA")
			topLevel := newRealmGroup("ns1", "roles-top", "Roles")
			child := newRealmGroup("ns1", "roles-child", "Roles", withParentGroup("parent-a"))

			Expect(k8sClient.Create(ctx, parent)).Should(Succeed())
			Expect(k8sClient.Create(ctx, topLevel)).Should(Succeed())
			Expect(k8sClient.Create(ctx, child)).Should(Succeed())
			cleanup(child, topLevel, parent)
		})

		It("Should deny the same group name under the same parent group", func() {
			parent := newRealmGroup("ns1", "parent-a", "ParentA")
			existing := newRealmGroup("ns1", "roles-one", "Roles", withParentGroup("parent-a"))
			Expect(k8sClient.Create(ctx, parent)).Should(Succeed())
			Expect(k8sClient.Create(ctx, existing)).Should(Succeed())
			defer cleanup(existing, parent)

			duplicate := newRealmGroup("ns1", "roles-two", "Roles", withParentGroup("parent-a"))

			err := k8sClient.Create(ctx, duplicate)
			Expect(err).Should(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring(
				`group name "Roles" is already used by KeycloakRealmGroup ns1/roles-one (under parent group "parent-a" in realm KeycloakRealm/ns1/test-realm)`,
			))
		})

		It("Should allow the same group name and parent name in another namespace (parent refs are namespace-local)", func() {
			parentNs1 := newRealmGroup("ns1", "parent-a", "ParentA")
			childNs1 := newRealmGroup("ns1", "roles-one", "Roles", withParentGroup("parent-a"))
			Expect(k8sClient.Create(ctx, parentNs1)).Should(Succeed())
			Expect(k8sClient.Create(ctx, childNs1)).Should(Succeed())
			defer cleanup(childNs1, parentNs1)

			parentNs2 := newRealmGroup("ns2", "parent-a", "ParentA")
			childNs2 := newRealmGroup("ns2", "roles-one", "Roles", withParentGroup("parent-a"))
			Expect(k8sClient.Create(ctx, parentNs2)).Should(Succeed())
			Expect(k8sClient.Create(ctx, childNs2)).Should(Succeed())
			cleanup(childNs2, parentNs2)
		})

		It("Should allow the same group name for different realms in the same namespace", func() {
			existing := newRealmGroup("ns1", "group-one", "developers")
			Expect(k8sClient.Create(ctx, existing)).Should(Succeed())
			defer cleanup(existing)

			otherRealm := newRealmGroup("ns1", "group-two", "developers", func(g *keycloakApi.KeycloakRealmGroup) {
				g.Spec.RealmRef.Name = "another-realm"
			})
			Expect(k8sClient.Create(ctx, otherRealm)).Should(Succeed())
			cleanup(otherRealm)
		})
	})

	Context("When updating KeycloakRealmGroup", func() {
		It("Should deny renaming a group into a colliding name", func() {
			existing := newRealmGroup("ns1", "group-one", "developers")
			victim := newRealmGroup("ns1", "group-two", "testers")
			Expect(k8sClient.Create(ctx, existing)).Should(Succeed())
			Expect(k8sClient.Create(ctx, victim)).Should(Succeed())
			defer cleanup(victim, existing)

			victim.Spec.Name = "developers"

			err := k8sClient.Update(ctx, victim)
			Expect(err).Should(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring(`group name "developers" is already used by KeycloakRealmGroup ns1/group-one`))
		})

		It("Should deny re-parenting a group into a colliding slot", func() {
			parentA := newRealmGroup("ns1", "parent-a", "ParentA")
			parentB := newRealmGroup("ns1", "parent-b", "ParentB")
			rolesA := newRealmGroup("ns1", "roles-a", "Roles", withParentGroup("parent-a"))
			rolesB := newRealmGroup("ns1", "roles-b", "Roles", withParentGroup("parent-b"))
			Expect(k8sClient.Create(ctx, parentA)).Should(Succeed())
			Expect(k8sClient.Create(ctx, parentB)).Should(Succeed())
			Expect(k8sClient.Create(ctx, rolesA)).Should(Succeed())
			Expect(k8sClient.Create(ctx, rolesB)).Should(Succeed())
			defer cleanup(rolesB, rolesA, parentB, parentA)

			rolesB.Spec.ParentGroup = &common.GroupRef{Name: "parent-a"}

			err := k8sClient.Update(ctx, rolesB)
			Expect(err).Should(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring(`group name "Roles" is already used by KeycloakRealmGroup ns1/roles-a`))
		})

		It("Should allow a self-update that keeps the same identity", func() {
			existing := newRealmGroup("ns1", "group-one", "developers")
			Expect(k8sClient.Create(ctx, existing)).Should(Succeed())
			defer cleanup(existing)

			existing.Spec.Description = "updated description"
			Expect(k8sClient.Update(ctx, existing)).Should(Succeed())
		})

		It("Should allow identity-preserving updates on duplicates that predate the webhook", func() {
			// Duplicates created before the webhook existed must stay reconcilable and
			// deletable: finalizer or metadata updates keep the identity unchanged and
			// must not be blocked, even though the duplicate slot is occupied.
			existing := newRealmGroup("ns1", "group-one", "developers")
			Expect(k8sClient.Create(ctx, existing)).Should(Succeed())
			defer cleanup(existing)

			validator := NewKeycloakRealmGroupCustomValidator(k8sClient)

			preexistingDuplicate := newRealmGroup("ns1", "group-legacy", "developers")
			updated := preexistingDuplicate.DeepCopy()
			updated.Finalizers = nil
			updated.Labels = map[string]string{"updated": "true"}

			_, err := validator.ValidateUpdate(ctx, preexistingDuplicate, updated)
			Expect(err).ShouldNot(HaveOccurred(), "identity-preserving update on a pre-existing duplicate must be allowed")

			renamed := preexistingDuplicate.DeepCopy()
			renamed.Spec.Name = "testers"
			_, err = validator.ValidateUpdate(ctx, preexistingDuplicate, renamed)
			Expect(err).ShouldNot(HaveOccurred(), "renaming a duplicate into a free slot must be allowed")
		})
	})
})

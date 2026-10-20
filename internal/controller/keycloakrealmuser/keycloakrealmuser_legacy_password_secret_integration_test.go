package keycloakrealmuser

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/epam/edp-keycloak-operator/api/common"
	keycloakApi "github.com/epam/edp-keycloak-operator/api/v1"
	"github.com/epam/edp-keycloak-operator/pkg/client/keycloakapi"
)

// Built unstructured: a typed spec cannot hold the stub.
var _ = Describe("KeycloakRealmUser legacy passwordSecret stub", func() {
	It("Should drop the stub and sync roles when no password is configured", func() {
		const (
			crName, username = "legacy-stub", "legacy-stub-user"
			roleName         = "legacy-stub-role"
		)

		_, err := keycloakApiClient.Roles.CreateRealmRole(ctx, KeycloakRealmCR, keycloakapi.RoleRepresentation{
			Name: ptr.To(roleName),
		})
		if err != nil && !keycloakapi.IsConflict(err) {
			Expect(err).ShouldNot(HaveOccurred())
		}

		u := newUserCR(crName, username, map[string]any{
			"passwordSecret": map[string]any{"name": "", "key": ""},
			"roles":          []any{roleName},
		})
		Expect(k8sClient.Create(ctx, u)).Should(Succeed())
		DeferCleanup(func() {
			Expect(client.IgnoreNotFound(k8sClient.Delete(ctx, u))).Should(Succeed())
		})

		_, found, err := unstructured.NestedMap(u.Object, "spec", "passwordSecret")
		Expect(err).ShouldNot(HaveOccurred())
		Expect(found).Should(BeTrue(), "the API server must store the stub")

		cr := &keycloakApi.KeycloakRealmUser{}

		Eventually(func(g Gomega) {
			got := &unstructured.Unstructured{}
			got.SetGroupVersionKind(keycloakApi.GroupVersion.WithKind("KeycloakRealmUser"))
			g.Expect(k8sClient.Get(ctx, types.NamespacedName{Name: crName, Namespace: ns}, got)).Should(Succeed())

			_, found, err := unstructured.NestedMap(got.Object, "spec", "passwordSecret")
			g.Expect(err).ShouldNot(HaveOccurred())
			g.Expect(found).Should(BeFalse())

			g.Expect(k8sClient.Get(ctx, types.NamespacedName{Name: crName, Namespace: ns}, cr)).Should(Succeed())
			g.Expect(cr.Status.Value).Should(Equal(common.StatusOK))
		}, timeout, interval).Should(Succeed())

		Eventually(func(g Gomega) {
			foundUser, _, err := keycloakApiClient.Users.FindUserByUsername(ctx, KeycloakRealmCR, username)
			g.Expect(err).ShouldNot(HaveOccurred())
			g.Expect(foundUser).ShouldNot(BeNil())
			g.Expect(foundUser.Id).ShouldNot(BeNil())

			realmRoles, _, err := keycloakApiClient.Users.GetUserRealmRoleMappings(ctx, KeycloakRealmCR, *foundUser.Id)
			g.Expect(err).ShouldNot(HaveOccurred())

			realmRoleNames := make([]string, 0, len(realmRoles))
			for _, role := range realmRoles {
				if role.Name != nil {
					realmRoleNames = append(realmRoleNames, *role.Name)
				}
			}

			g.Expect(realmRoleNames).Should(ContainElement(roleName))
		}, timeout, interval).Should(Succeed())
	})
})

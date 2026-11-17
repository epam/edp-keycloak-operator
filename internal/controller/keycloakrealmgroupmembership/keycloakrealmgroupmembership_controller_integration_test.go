package keycloakrealmgroupmembership

import (
	"slices"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	k8sErrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/utils/ptr"

	"github.com/epam/edp-keycloak-operator/api/common"
	keycloakApi "github.com/epam/edp-keycloak-operator/api/v1"
	"github.com/epam/edp-keycloak-operator/pkg/client/keycloakapi"
	"github.com/epam/edp-keycloak-operator/pkg/objectmeta"
	"github.com/epam/edp-keycloak-operator/pkg/testutils"
)

var _ = Describe("KeycloakRealmGroupMembership controller", Ordered, func() {
	var (
		groupName string
		groupID   string
	)

	createUser := func(username, email string) string {
		resp, err := keycloakApiClient.Users.CreateUser(ctx, KeycloakRealmCR, keycloakapi.UserRepresentation{
			Username: ptr.To(username),
			Email:    ptr.To(email),
			Enabled:  ptr.To(true),
		})
		Expect(err).ShouldNot(HaveOccurred())

		return keycloakapi.GetResourceIDFromResponse(resp)
	}

	groupMembers := func(g Gomega) []string {
		members, _, err := keycloakApiClient.Groups.GetGroupMembers(ctx, KeycloakRealmCR, groupID, nil)
		g.Expect(err).ShouldNot(HaveOccurred())

		names := make([]string, 0, len(members))
		for _, m := range members {
			names = append(names, *m.Username)
		}

		slices.Sort(names)

		return names
	}

	getMembership := func(g Gomega, name string) *keycloakApi.KeycloakRealmGroupMembership {
		m := &keycloakApi.KeycloakRealmGroupMembership{}
		g.Expect(k8sClient.Get(ctx, types.NamespacedName{Name: name, Namespace: ns}, m)).Should(Succeed())

		return m
	}

	deleteMembership := func(name string) {
		Expect(k8sClient.Delete(ctx, getMembership(Default, name))).Should(Succeed())
		Eventually(func() bool {
			err := k8sClient.Get(ctx, types.NamespacedName{Name: name, Namespace: ns},
				&keycloakApi.KeycloakRealmGroupMembership{})

			return k8sErrors.IsNotFound(err)
		}, timeout, interval).Should(BeTrue())
	}

	BeforeAll(func() {
		groupName = testutils.UniqueName("membership-group")
		resp, err := keycloakApiClient.Groups.CreateGroup(ctx, KeycloakRealmCR, keycloakapi.GroupRepresentation{
			Name: ptr.To(groupName),
		})
		Expect(err).ShouldNot(HaveOccurred())

		groupID = keycloakapi.GetResourceIDFromResponse(resp)

		createUser("member-a", "member-a@example.com")
		createUser("member-b", "member-b@example.com")

		// The outsider is a member added by other means.
		_, err = keycloakApiClient.Users.AddUserToGroup(ctx, KeycloakRealmCR,
			createUser("outsider", "outsider@example.com"), groupID)
		Expect(err).ShouldNot(HaveOccurred())
	})

	It("Should add members without removing others and report pending members (addOnly)", func() {
		membership := &keycloakApi.KeycloakRealmGroupMembership{
			ObjectMeta: metav1.ObjectMeta{Name: "add-only", Namespace: ns},
			Spec: keycloakApi.KeycloakRealmGroupMembershipSpec{
				RealmRef: common.RealmRef{Kind: keycloakApi.KeycloakRealmKind, Name: KeycloakRealmCR},
				Group:    "/" + groupName,
				Members: []keycloakApi.GroupMember{
					{Username: "member-a"},
					{Email: "member-b@example.com"},
					{Username: "late-user"},
				},
			},
		}
		Expect(k8sClient.Create(ctx, membership)).Should(Succeed())

		Eventually(func(g Gomega) {
			m := getMembership(g, membership.Name)
			g.Expect(m.Status.Value).Should(Equal(common.StatusOK))
			g.Expect(m.Status.PendingMembers).Should(ConsistOf(keycloakApi.GroupMember{Username: "late-user"}))
			g.Expect(m.Status.ResolvedMembers).Should(HaveLen(2))
			g.Expect(groupMembers(g)).Should(Equal([]string{"member-a", "member-b", "outsider"}))
		}, timeout, interval).Should(Succeed())

		_, _, err := keycloakApiClient.Users.FindUserByUsername(ctx, KeycloakRealmCR, "late-user")
		Expect(keycloakapi.IsNotFound(err)).Should(BeTrue(), "the operator must not create users")
	})

	It("Should add a pending member once the user exists", func() {
		createUser("late-user", "late-user@example.com")

		Eventually(func(g Gomega) {
			m := getMembership(g, "add-only")
			g.Expect(m.Status.PendingMembers).Should(BeEmpty())
			g.Expect(m.Status.ResolvedMembers).Should(HaveLen(3))
			g.Expect(groupMembers(g)).Should(Equal([]string{"late-user", "member-a", "member-b", "outsider"}))
		}, timeout, interval).Should(Succeed())
	})

	It("Should remove unlisted members (full)", func() {
		m := getMembership(Default, "add-only")
		m.Spec.Strategy = keycloakApi.ReconciliationStrategyFull
		m.Spec.Members = []keycloakApi.GroupMember{{Username: "member-a"}, {Username: "late-user"}}
		Expect(k8sClient.Update(ctx, m)).Should(Succeed())

		Eventually(func(g Gomega) {
			g.Expect(groupMembers(g)).Should(Equal([]string{"late-user", "member-a"}))
		}, timeout, interval).Should(Succeed())
	})

	It("Should remove only the listed members on deletion", func() {
		_, err := keycloakApiClient.Users.AddUserToGroup(ctx, KeycloakRealmCR, mustFindUser("outsider"), groupID)
		Expect(err).ShouldNot(HaveOccurred())

		m := getMembership(Default, "add-only")
		m.Spec.Strategy = keycloakApi.ReconciliationStrategyAddOnly
		Expect(k8sClient.Update(ctx, m)).Should(Succeed())
		Eventually(func(g Gomega) {
			g.Expect(getMembership(g, "add-only").Status.Value).Should(Equal(common.StatusOK))
		}, timeout, interval).Should(Succeed())

		deleteMembership("add-only")

		Eventually(func(g Gomega) {
			g.Expect(groupMembers(g)).Should(Equal([]string{"outsider"}))
		}, timeout, interval).Should(Succeed())
	})

	It("Should keep members on deletion with the preserve annotation", func() {
		membership := &keycloakApi.KeycloakRealmGroupMembership{
			ObjectMeta: metav1.ObjectMeta{
				Name:        "preserve",
				Namespace:   ns,
				Annotations: map[string]string{objectmeta.PreserveResourcesOnDeletionAnnotation: "true"},
			},
			Spec: keycloakApi.KeycloakRealmGroupMembershipSpec{
				RealmRef: common.RealmRef{Kind: keycloakApi.KeycloakRealmKind, Name: KeycloakRealmCR},
				Group:    groupName,
				Members:  []keycloakApi.GroupMember{{Username: "member-b"}},
			},
		}
		Expect(k8sClient.Create(ctx, membership)).Should(Succeed())

		Eventually(func(g Gomega) {
			g.Expect(groupMembers(g)).Should(Equal([]string{"member-b", "outsider"}))
		}, timeout, interval).Should(Succeed())

		deleteMembership("preserve")

		Consistently(func(g Gomega) {
			g.Expect(groupMembers(g)).Should(Equal([]string{"member-b", "outsider"}))
		}, "2s", interval).Should(Succeed())
	})

	It("Should report a missing group", func() {
		membership := &keycloakApi.KeycloakRealmGroupMembership{
			ObjectMeta: metav1.ObjectMeta{Name: "missing-group", Namespace: ns},
			Spec: keycloakApi.KeycloakRealmGroupMembershipSpec{
				RealmRef: common.RealmRef{Kind: keycloakApi.KeycloakRealmKind, Name: KeycloakRealmCR},
				Group:    "/does-not-exist",
				Members:  []keycloakApi.GroupMember{{Username: "member-a"}},
			},
		}
		Expect(k8sClient.Create(ctx, membership)).Should(Succeed())

		Eventually(func(g Gomega) {
			m := getMembership(g, membership.Name)
			g.Expect(m.Status.Value).Should(ContainSubstring("create it with a KeycloakRealmGroup"))
		}, timeout, interval).Should(Succeed())

		deleteMembership(membership.Name)
	})

	It("Should reject a member with both username and email", func() {
		membership := &keycloakApi.KeycloakRealmGroupMembership{
			ObjectMeta: metav1.ObjectMeta{Name: "invalid", Namespace: ns},
			Spec: keycloakApi.KeycloakRealmGroupMembershipSpec{
				RealmRef: common.RealmRef{Kind: keycloakApi.KeycloakRealmKind, Name: KeycloakRealmCR},
				Group:    groupName,
				Members:  []keycloakApi.GroupMember{{Username: "a", Email: "a@example.com"}},
			},
		}
		err := k8sClient.Create(ctx, membership)
		Expect(err).Should(HaveOccurred())
		Expect(err.Error()).Should(ContainSubstring("exactly one of username or email must be set"))
	})
})

func mustFindUser(username string) string {
	u, _, err := keycloakApiClient.Users.FindUserByUsername(ctx, KeycloakRealmCR, username)
	Expect(err).ShouldNot(HaveOccurred())

	return *u.Id
}

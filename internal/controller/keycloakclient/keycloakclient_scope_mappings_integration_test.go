package keycloakclient

import (
	"slices"
	"strings"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	k8sErrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/epam/edp-keycloak-operator/api/common"
	keycloakApi "github.com/epam/edp-keycloak-operator/api/v1"
	"github.com/epam/edp-keycloak-operator/internal/controller/keycloakclient/chain"
	"github.com/epam/edp-keycloak-operator/pkg/client/keycloakapi"
	"github.com/epam/edp-keycloak-operator/pkg/testutils"
)

// Admin event log is the write oracle: a CLIENT UPDATE event proves a reconcile ran; scope-mapping events prove writes.
var _ = Describe("KeycloakClient scope mappings", Ordered, func() {
	const (
		crName       = "scope-mappings-client"
		helperClient = "scope-mappings-api"
		realmRole    = "scope-mappings-rr"
		clientRole   = "read"
		settle       = testutils.Settle
		longWait     = testutils.LongWait
	)

	key := types.NamespacedName{Name: crName, Namespace: ns}

	var (
		recorder   *testutils.AdminEventRecorder
		clientUUID string
		helperUUID string
	)

	scopeEvents := func(events testutils.AdminEvents, ops ...string) testutils.AdminEvents {
		out := make(testutils.AdminEvents, 0, len(events))

		for _, e := range events {
			if strings.Contains(ptr.Deref(e.ResourcePath, ""), "scope-mappings") &&
				slices.Contains(ops, ptr.Deref(e.OperationType, "")) {
				out = append(out, e)
			}
		}

		return out
	}

	clientUpdates := func(events testutils.AdminEvents) int {
		n := 0

		for _, e := range events {
			if ptr.Deref(e.OperationType, "") == "UPDATE" &&
				strings.HasSuffix(ptr.Deref(e.ResourcePath, ""), "clients/"+clientUUID) {
				n++
			}
		}

		return n
	}

	nudge := func() {
		Expect(testutils.Nudge(ctx, k8sClient, key, &keycloakApi.KeycloakClient{})).To(Succeed())
	}

	names := func(roles []keycloakapi.RoleRepresentation) []string {
		out := make([]string, 0, len(roles))
		for _, r := range roles {
			out = append(out, ptr.Deref(r.Name, ""))
		}

		return out
	}

	realmMappings := func(g Gomega) []string {
		roles, _, err := keycloakAdmin.Clients.GetRealmScopeMappings(ctx, KeycloakRealmCR, clientUUID)
		g.Expect(err).ShouldNot(HaveOccurred())

		return names(roles)
	}

	clientMappings := func(g Gomega) []string {
		roles, _, err := keycloakAdmin.Clients.GetClientScopeMappings(ctx, KeycloakRealmCR, clientUUID, helperUUID)
		g.Expect(err).ShouldNot(HaveOccurred())

		return names(roles)
	}

	patchSpec := func(mutate func(spec *keycloakApi.KeycloakClientSpec)) {
		cr := &keycloakApi.KeycloakClient{}
		Expect(k8sClient.Get(ctx, key, cr)).To(Succeed())

		original := cr.DeepCopy()
		mutate(&cr.Spec)
		Expect(k8sClient.Patch(ctx, cr, client.MergeFrom(original))).To(Succeed())
	}

	conditionTrue := func(g Gomega) {
		cr := &keycloakApi.KeycloakClient{}
		g.Expect(k8sClient.Get(ctx, key, cr)).To(Succeed())
		g.Expect(cr.Status.Value).To(Equal(common.StatusOK))

		cond := meta.FindStatusCondition(cr.Status.Conditions, chain.ConditionScopeMappingsSynced)
		g.Expect(cond).NotTo(BeNil())
		g.Expect(cond.Status).To(Equal(metav1.ConditionTrue))
		g.Expect(cond.Reason).To(Equal(chain.ReasonScopeMappingsSynced))
	}

	BeforeAll(func() {
		recorder = testutils.NewAdminEventRecorder(keycloakAdmin, KeycloakRealmCR)
		Expect(recorder.Enable(ctx)).To(Succeed())

		_, err := keycloakAdmin.Roles.CreateRealmRole(ctx, KeycloakRealmCR,
			keycloakapi.RoleRepresentation{Name: ptr.To(realmRole)})
		Expect(err).ShouldNot(HaveOccurred())

		_, err = keycloakAdmin.Clients.CreateClient(ctx, KeycloakRealmCR, keycloakapi.ClientRepresentation{
			ClientId: ptr.To(helperClient),
			Protocol: ptr.To("openid-connect"),
		})
		Expect(err).ShouldNot(HaveOccurred())

		helperUUID, err = keycloakAdmin.Clients.GetClientUUID(ctx, KeycloakRealmCR, helperClient)
		Expect(err).ShouldNot(HaveOccurred())

		_, err = keycloakAdmin.Clients.CreateClientRole(ctx, KeycloakRealmCR, helperUUID,
			keycloakapi.RoleRepresentation{Name: ptr.To(clientRole)})
		Expect(err).ShouldNot(HaveOccurred())
	})

	AfterAll(func() {
		cr := &keycloakApi.KeycloakClient{}
		if err := k8sClient.Get(ctx, key, cr); err == nil {
			Expect(k8sClient.Delete(ctx, cr)).To(Succeed())

			Eventually(func() bool {
				return k8sErrors.IsNotFound(k8sClient.Get(ctx, key, &keycloakApi.KeycloakClient{}))
			}, longWait, interval).Should(BeTrue())
		}

		if helperUUID != "" {
			_, _ = keycloakAdmin.Clients.DeleteClient(ctx, KeycloakRealmCR, helperUUID)
		}

		_, _ = keycloakAdmin.Roles.DeleteRealmRole(ctx, KeycloakRealmCR, realmRole)
	})

	It("Should write realm and client scope mappings", func() {
		cr := &keycloakApi.KeycloakClient{
			ObjectMeta: metav1.ObjectMeta{Name: crName, Namespace: ns},
			Spec: keycloakApi.KeycloakClientSpec{
				ClientId:         crName,
				RealmRef:         common.RealmRef{Name: KeycloakRealmCR, Kind: keycloakApi.KeycloakRealmKind},
				Public:           true,
				Enabled:          true,
				FullScopeAllowed: false,
				ScopeMappings: &keycloakApi.ScopeMappings{
					RealmRoles: ptr.To([]string{realmRole}),
					ClientRoles: []keycloakApi.UserClientRole{
						{ClientID: helperClient, Roles: ptr.To([]string{clientRole})},
					},
				},
			},
		}
		Expect(k8sClient.Create(ctx, cr)).To(Succeed())

		Eventually(func(g Gomega) {
			conditionTrue(g)

			var err error

			clientUUID, err = keycloakAdmin.Clients.GetClientUUID(ctx, KeycloakRealmCR, crName)
			g.Expect(err).ShouldNot(HaveOccurred())
			g.Expect(realmMappings(g)).To(Equal([]string{realmRole}))
			g.Expect(clientMappings(g)).To(Equal([]string{clientRole}))
		}, longWait, interval).Should(Succeed())

		recorder = recorder.Scoped(clientUUID)
	})

	It("Should not write on a settled reconcile", func() {
		events, err := recorder.WritesDuring(ctx, settle, nudge)
		Expect(err).ShouldNot(HaveOccurred())

		Expect(clientUpdates(events)).To(BeNumerically(">", 0),
			"no CLIENT UPDATE recorded: the reconcile did not reach Keycloak within %s", settle)
		Expect(scopeEvents(events, "CREATE", "DELETE")).To(BeEmpty())
	})

	It("Should remove a realm mapping added by hand", func() {
		role, _, err := keycloakAdmin.Roles.GetRealmRole(ctx, KeycloakRealmCR, "offline_access")
		Expect(err).ShouldNot(HaveOccurred())

		mappable, err := keycloakapi.RequireRealmRoleWithID(role, KeycloakRealmCR, "offline_access")
		Expect(err).ShouldNot(HaveOccurred())

		_, err = keycloakAdmin.Clients.AddRealmScopeMappings(ctx, KeycloakRealmCR, clientUUID,
			[]keycloakapi.RoleRepresentation{mappable})
		Expect(err).ShouldNot(HaveOccurred())

		events, err := recorder.WritesDuring(ctx, 0, func() {
			nudge()

			Eventually(func(g Gomega) {
				g.Expect(realmMappings(g)).To(Equal([]string{realmRole}))
			}, longWait, interval).Should(Succeed())
		})
		Expect(err).ShouldNot(HaveOccurred())
		Expect(scopeEvents(events, "DELETE")).To(HaveLen(1))
	})

	It("Should clear both facets with empty lists", func() {
		patchSpec(func(spec *keycloakApi.KeycloakClientSpec) {
			spec.ScopeMappings = &keycloakApi.ScopeMappings{
				RealmRoles:  ptr.To([]string{}),
				ClientRoles: []keycloakApi.UserClientRole{{ClientID: helperClient, Roles: ptr.To([]string{})}},
			}
		})

		Eventually(func(g Gomega) {
			g.Expect(realmMappings(g)).To(BeEmpty())
			g.Expect(clientMappings(g)).To(BeEmpty())
		}, longWait, interval).Should(Succeed())
	})

	It("Should leave mappings alone when the field is removed", func() {
		patchSpec(func(spec *keycloakApi.KeycloakClientSpec) { spec.ScopeMappings = nil })

		// The reconcile triggered by the patch must finish before the manual mapping exists.
		_, err := recorder.WritesDuring(ctx, settle, nudge)
		Expect(err).ShouldNot(HaveOccurred())

		role, _, err := keycloakAdmin.Roles.GetRealmRole(ctx, KeycloakRealmCR, "offline_access")
		Expect(err).ShouldNot(HaveOccurred())

		mappable, err := keycloakapi.RequireRealmRoleWithID(role, KeycloakRealmCR, "offline_access")
		Expect(err).ShouldNot(HaveOccurred())

		_, err = keycloakAdmin.Clients.AddRealmScopeMappings(ctx, KeycloakRealmCR, clientUUID,
			[]keycloakapi.RoleRepresentation{mappable})
		Expect(err).ShouldNot(HaveOccurred())

		events, err := recorder.WritesDuring(ctx, settle, nudge)
		Expect(err).ShouldNot(HaveOccurred())

		Expect(clientUpdates(events)).To(BeNumerically(">", 0),
			"no CLIENT UPDATE recorded: the reconcile did not reach Keycloak within %s", settle)
		Expect(scopeEvents(events, "CREATE", "DELETE")).To(BeEmpty())

		Eventually(func(g Gomega) {
			g.Expect(realmMappings(g)).To(Equal([]string{"offline_access"}))
		}, longWait, interval).Should(Succeed())
	})
})

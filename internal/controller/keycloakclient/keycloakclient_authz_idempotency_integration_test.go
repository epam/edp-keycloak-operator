package keycloakclient

import (
	"slices"
	"strings"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	corev1 "k8s.io/api/core/v1"
	k8sErrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/utils/ptr"

	"github.com/epam/edp-keycloak-operator/api/common"
	keycloakApi "github.com/epam/edp-keycloak-operator/api/v1"
	"github.com/epam/edp-keycloak-operator/pkg/client/keycloakapi"
	"github.com/epam/edp-keycloak-operator/pkg/secretref"
	"github.com/epam/edp-keycloak-operator/pkg/testutils"
)

// Idempotency suite for the KeycloakClient authorization block. Keycloak's admin event log is
// the write oracle.
//
// The handlers still update matching policies, permissions and resources on every reconcile, so
// a steady-state window holds UPDATE events. Only CREATE and DELETE are asserted absent: those
// are the operations that churn Keycloak IDs. Each is anchored by a spec that provokes it, since
// an absence assertion is evidence only while the log records that operation.
var _ = Describe("KeycloakClient authz idempotent reconcile", Ordered, func() {
	const (
		crName         = "authz-client"
		clientID       = "authz-client"
		secretName     = "authz-client-secret"
		secretKey      = "clientSecret"
		policyName     = "authz-policy"
		permissionName = "authz-permission"
		permissionDesc = "authz permission v1"
		settle         = testutils.Settle
		longWait       = testutils.LongWait
	)

	var (
		recorder   *testutils.AdminEventRecorder
		clientUUID string
	)

	// eventsMatching returns the events whose operation is one of ops and whose resourcePath
	// contains pathPart. An empty pathPart matches every event, including one with no path.
	// A non-empty pathPart drops pathless events: an anchor must name the object it proves.
	eventsMatching := func(events testutils.AdminEvents, pathPart string, ops ...string) testutils.AdminEvents {
		matched := make(testutils.AdminEvents, 0, len(events))

		for _, e := range events {
			if !strings.Contains(ptr.Deref(e.ResourcePath, ""), pathPart) {
				continue
			}

			if slices.Contains(ops, ptr.Deref(e.OperationType, "")) {
				matched = append(matched, e)
			}
		}

		return matched
	}

	nudge := func() {
		Expect(testutils.Nudge(ctx, k8sClient,
			types.NamespacedName{Name: crName, Namespace: ns}, &keycloakApi.KeycloakClient{})).To(Succeed())
	}

	waitReady := func() {
		Eventually(func(g Gomega) {
			cr := &keycloakApi.KeycloakClient{}
			g.Expect(k8sClient.Get(ctx, types.NamespacedName{Name: crName, Namespace: ns}, cr)).To(Succeed())
			g.Expect(cr.Status.Value).To(Equal(common.StatusOK))
		}, longWait, interval).Should(Succeed())
	}

	idsByName := func(entries []keycloakapi.AbstractPolicyRepresentation) map[string]string {
		ids := make(map[string]string, len(entries))
		for _, e := range entries {
			ids[ptr.Deref(e.Name, "")] = ptr.Deref(e.Id, "")
		}

		return ids
	}

	permissions := func() []keycloakapi.AbstractPolicyRepresentation {
		perms, _, err := keycloakAdmin.Authorization.GetPermissions(ctx, KeycloakRealmCR, clientUUID)
		Expect(err).ShouldNot(HaveOccurred())

		return perms
	}

	policies := func() []keycloakapi.AbstractPolicyRepresentation {
		pols, _, err := keycloakAdmin.Authorization.GetPolicies(ctx, KeycloakRealmCR, clientUUID)
		Expect(err).ShouldNot(HaveOccurred())

		return pols
	}

	permission := func(name string) keycloakapi.AbstractPolicyRepresentation {
		for _, p := range permissions() {
			if ptr.Deref(p.Name, "") == name {
				return p
			}
		}

		Fail("permission " + name + " is missing from Keycloak")

		return keycloakapi.AbstractPolicyRepresentation{}
	}

	BeforeAll(func() {
		recorder = testutils.NewAdminEventRecorder(keycloakAdmin, KeycloakRealmCR)
		Expect(recorder.Enable(ctx)).To(Succeed())

		Expect(k8sClient.Create(ctx, &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{Name: secretName, Namespace: ns},
			Data:       map[string][]byte{secretKey: []byte("authz-v1")},
		})).To(Succeed())
	})

	// Sibling Describes in this package share the namespace and the realm, and Ginkgo randomizes
	// top-level container order. Leave nothing behind.
	AfterAll(func() {
		cr := &keycloakApi.KeycloakClient{}
		if err := k8sClient.Get(ctx, types.NamespacedName{Name: crName, Namespace: ns}, cr); err == nil {
			Expect(k8sClient.Delete(ctx, cr)).To(Succeed())

			Eventually(func() bool {
				return k8sErrors.IsNotFound(k8sClient.Get(ctx,
					types.NamespacedName{Name: crName, Namespace: ns}, &keycloakApi.KeycloakClient{}))
			}, longWait, interval).Should(BeTrue())
		}

		secret := &corev1.Secret{}
		if err := k8sClient.Get(ctx, types.NamespacedName{Name: secretName, Namespace: ns}, secret); err == nil {
			Expect(k8sClient.Delete(ctx, secret)).To(Succeed())
		}
	})

	It("Should create the client with authorization scopes, resources, policies and permissions", func() {
		cr := &keycloakApi.KeycloakClient{
			ObjectMeta: metav1.ObjectMeta{Name: crName, Namespace: ns},
			Spec: keycloakApi.KeycloakClientSpec{
				ClientId:                     clientID,
				RealmRef:                     common.RealmRef{Name: KeycloakRealmCR, Kind: keycloakApi.KeycloakRealmKind},
				Secret:                       secretref.GenerateSecretRef(secretName, secretKey),
				Enabled:                      true,
				StandardFlowEnabled:          true,
				AuthorizationServicesEnabled: true,
				ClientAuthenticatorType:      "client-secret",
				// Mappers and service-account attributes are inert here: they emit no CREATE or
				// DELETE once converged.
				ProtocolMappers: &[]keycloakApi.ProtocolMapper{
					{
						Name:           "authz-pm-a",
						Protocol:       "openid-connect",
						ProtocolMapper: "oidc-hardcoded-claim-mapper",
						Config: map[string]string{
							"claim.name": "authz-a", "claim.value": "v1",
							"access.token.claim": "true", "id.token.claim": "true",
							"jsonType.label": "String",
						},
					},
				},
				ServiceAccount: &keycloakApi.ServiceAccount{
					Enabled:      true,
					AttributesV2: map[string][]string{"authz-attr": {"one"}},
				},
				Authorization: &keycloakApi.Authorization{
					Scopes: []string{"authz-scope"},
					Resources: []keycloakApi.Resource{{
						Name:        "authz-resource",
						DisplayName: "Authz Resource",
						Type:        "urn:authz:resource",
						Scopes:      []string{"authz-scope"},
					}},
					Policies: []keycloakApi.Policy{{
						Name:         policyName,
						Type:         keycloakApi.PolicyTypeClient,
						ClientPolicy: &keycloakApi.ClientPolicyData{Clients: []string{clientID}},
					}},
					Permissions: []keycloakApi.Permission{{
						Name:        permissionName,
						Type:        keycloakApi.PermissionTypeResource,
						Description: permissionDesc,
						Policies:    []string{policyName},
						Resources:   []string{"authz-resource"},
					}},
				},
			},
		}

		// The client UUID does not exist yet, so the window cannot be scoped in advance.
		events, err := recorder.WritesDuring(ctx, 0, func() {
			Expect(k8sClient.Create(ctx, cr)).To(Succeed())
			waitReady()
		})
		Expect(err).ShouldNot(HaveOccurred())

		clientUUID, err = keycloakAdmin.Clients.GetClientUUID(ctx, KeycloakRealmCR, clientID)
		Expect(err).ShouldNot(HaveOccurred())
		Expect(clientUUID).ShouldNot(BeEmpty())

		Expect(eventsMatching(events, clientUUID, "CREATE")).ShouldNot(BeEmpty(),
			"admin event log recorded no CREATE for a client it just created")

		Expect(idsByName(permissions())).To(HaveKey(permissionName))
		Expect(idsByName(policies())).To(HaveKey(policyName))

		// Sibling specs share the realm; report only this client's events from here on.
		recorder = recorder.Scoped(clientUUID)
	})

	// Must run before the drift and orphan specs: they mutate Keycloak, and a failure in an
	// Ordered suite skips every later spec.
	It("Should keep policy and permission IDs stable and not create or delete authorization objects on a forced reconcile", func() {
		permissionsBefore := idsByName(permissions())
		policiesBefore := idsByName(policies())

		events, err := recorder.WritesDuring(ctx, settle, nudge)
		Expect(err).ShouldNot(HaveOccurred())

		// Nudge returns before the reconcile lands and WritesDuring only sleeps, so an empty
		// window means either "no writes" or "nothing ran". The policy and permission handlers
		// write on every reconcile, so this UPDATE proves the reconcile reached Keycloak inside
		// the settle window and the assertions below are not vacuous.
		Expect(eventsMatching(events, "", "UPDATE")).ShouldNot(BeEmpty(),
			"no UPDATE recorded for a forced reconcile: the reconcile did not reach Keycloak within %s", settle)

		Expect(idsByName(permissions())).To(Equal(permissionsBefore),
			"permissions must be updated in place, not deleted and recreated")
		Expect(idsByName(policies())).To(Equal(policiesBefore),
			"policies must be updated in place, not deleted and recreated")

		// UPDATE is expected here; see the suite comment for why only CREATE and DELETE are
		// asserted absent.
		Expect(eventsMatching(events, "", "CREATE", "DELETE")).To(BeEmpty())
	})

	It("Should heal external drift on the permission", func() {
		drifted := permission(permissionName)
		permID := ptr.Deref(drifted.Id, "")
		Expect(permID).ShouldNot(BeEmpty())

		events, err := recorder.WritesDuring(ctx, 0, func() {
			_, updateErr := keycloakAdmin.Authorization.UpdatePermission(
				ctx, KeycloakRealmCR, clientUUID, keycloakApi.PermissionTypeResource, permID,
				keycloakapi.PolicyRepresentation{
					Id:               drifted.Id,
					Name:             drifted.Name,
					Type:             drifted.Type,
					Description:      ptr.To("drifted-by-hand"),
					DecisionStrategy: drifted.DecisionStrategy,
					Logic:            drifted.Logic,
					Policies:         drifted.Policies,
					Resources:        drifted.Resources,
				})
			Expect(updateErr).ShouldNot(HaveOccurred())

			nudge()

			Eventually(func(g Gomega) {
				g.Expect(ptr.Deref(permission(permissionName).Description, "")).To(Equal(permissionDesc))
			}, longWait, interval).Should(Succeed())
		})
		Expect(err).ShouldNot(HaveOccurred())
		Expect(events).ShouldNot(BeEmpty(), "admin event log recorded nothing for a known write")
	})

	It("Should delete a policy that is not in the spec", func() {
		// GetPolicies excludes permissions but must still return real orphan policies, so the
		// sweep can delete them.
		strayType := keycloakApi.PolicyTypeTime
		stray, _, err := keycloakAdmin.Authorization.CreatePolicy(
			ctx, KeycloakRealmCR, clientUUID, strayType,
			keycloakapi.PolicyRepresentation{
				Name:             ptr.To("stray-policy"),
				Type:             ptr.To(strayType),
				Logic:            ptr.To(keycloakapi.Logic("POSITIVE")),
				DecisionStrategy: ptr.To(keycloakapi.DecisionStrategy("UNANIMOUS")),
			})
		Expect(err).ShouldNot(HaveOccurred())
		Expect(stray).ShouldNot(BeNil())

		strayID := ptr.Deref(stray.Id, "")
		Expect(strayID).ShouldNot(BeEmpty())

		events, err := recorder.WritesDuring(ctx, 0, func() {
			nudge()

			Eventually(func(g Gomega) {
				g.Expect(idsByName(policies())).ShouldNot(HaveKey("stray-policy"))
			}, longWait, interval).Should(Succeed())
		})
		Expect(err).ShouldNot(HaveOccurred())

		Expect(eventsMatching(events, strayID, "DELETE")).ShouldNot(BeEmpty(),
			"admin event log recorded no DELETE for the orphan policy it just removed")
	})

})

package v1

import (
	"context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	k8sErrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"

	"github.com/epam/edp-keycloak-operator/api/common"
	keycloakApi "github.com/epam/edp-keycloak-operator/api/v1"
)

var _ = Describe("KeycloakClient Webhook", func() {
	const (
		testClientId     = "test-client"
		testRealmName    = "test-realm"
		testKeycloakName = "test-keycloak"
		testNamespace    = "ns1"
		testWebUrl       = "https://example.com"
	)

	var keycloakClient *keycloakApi.KeycloakClient

	AfterEach(func() {
		if keycloakClient != nil {
			Expect(client.IgnoreNotFound(k8sClient.Delete(ctx, keycloakClient))).
				Should(Succeed(), "failed to delete KeycloakClient")
			keycloakClient = nil
		}
	})

	Context("When creating KeycloakClient under Defaulting Webhook", func() {
		It("Should set default post.logout.redirect.uris attribute to +", func() {
			keycloakClient = &keycloakApi.KeycloakClient{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-client-default-attr",
					Namespace: testNamespace,
				},
				Spec: keycloakApi.KeycloakClientSpec{
					ClientId: testClientId,
					RealmRef: common.RealmRef{
						Kind: keycloakApi.KeycloakRealmKind,
						Name: testRealmName,
					},
					WebUrl: testWebUrl,
					// Attributes not set - should be defaulted
				},
			}

			Expect(k8sClient.Create(ctx, keycloakClient)).Should(Succeed())

			// Fetch the created resource to verify defaults
			createdClient := &keycloakApi.KeycloakClient{}
			Expect(k8sClient.Get(ctx, client.ObjectKeyFromObject(keycloakClient), createdClient)).
				Should(Succeed())

			// Verify default attribute was set
			Expect(createdClient.Spec.Attributes).Should(HaveKey(ClientAttributeLogoutRedirectUris))
			Expect(createdClient.Spec.Attributes[ClientAttributeLogoutRedirectUris]).Should(Equal(ClientAttributeLogoutRedirectUrisDefValue))
		})

		It("Should NOT override existing post.logout.redirect.uris attribute", func() {
			customValue := "https://custom-logout.com"
			keycloakClient = &keycloakApi.KeycloakClient{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-client-custom-attr",
					Namespace: testNamespace,
				},
				Spec: keycloakApi.KeycloakClientSpec{
					ClientId: testClientId,
					RealmRef: common.RealmRef{
						Kind: keycloakApi.KeycloakRealmKind,
						Name: testRealmName,
					},
					WebUrl: testWebUrl,
					Attributes: map[string]string{
						ClientAttributeLogoutRedirectUris: customValue,
					},
				},
			}

			Expect(k8sClient.Create(ctx, keycloakClient)).Should(Succeed())

			createdClient := &keycloakApi.KeycloakClient{}
			Expect(k8sClient.Get(ctx, client.ObjectKeyFromObject(keycloakClient), createdClient)).
				Should(Succeed())

			// Verify custom value was preserved
			Expect(createdClient.Spec.Attributes[ClientAttributeLogoutRedirectUris]).Should(Equal(customValue))
		})

		It("Should initialize WebOrigins from WebUrl when WebUrl is provided", func() {
			keycloakClient = &keycloakApi.KeycloakClient{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-client-weborigins",
					Namespace: testNamespace,
				},
				Spec: keycloakApi.KeycloakClientSpec{
					ClientId: testClientId,
					RealmRef: common.RealmRef{
						Kind: keycloakApi.KeycloakRealmKind,
						Name: testRealmName,
					},
					WebUrl: testWebUrl,
					// WebOrigins not set - should be defaulted from WebUrl
				},
			}

			Expect(k8sClient.Create(ctx, keycloakClient)).Should(Succeed())

			createdClient := &keycloakApi.KeycloakClient{}
			Expect(k8sClient.Get(ctx, client.ObjectKeyFromObject(keycloakClient), createdClient)).
				Should(Succeed())

			// Verify WebOrigins was initialized from WebUrl
			Expect(createdClient.Spec.WebOrigins).Should(Equal(ptr.To([]string{testWebUrl})))
		})

		It("Should NOT initialize WebOrigins when WebUrl is empty", func() {
			keycloakClient = &keycloakApi.KeycloakClient{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-client-no-weburl",
					Namespace: testNamespace,
				},
				Spec: keycloakApi.KeycloakClientSpec{
					ClientId: testClientId,
					RealmRef: common.RealmRef{
						Kind: keycloakApi.KeycloakRealmKind,
						Name: testRealmName,
					},
					// WebUrl is empty
					// WebOrigins not set
				},
			}

			Expect(k8sClient.Create(ctx, keycloakClient)).Should(Succeed())

			createdClient := &keycloakApi.KeycloakClient{}
			Expect(k8sClient.Get(ctx, client.ObjectKeyFromObject(keycloakClient), createdClient)).
				Should(Succeed())

			// Verify WebOrigins was NOT initialized (remains nil)
			Expect(createdClient.Spec.WebOrigins).Should(BeNil())
		})

		It("Should NOT override existing WebOrigins", func() {
			customOrigins := []string{"https://custom1.com", "https://custom2.com"}
			keycloakClient = &keycloakApi.KeycloakClient{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-client-custom-origins",
					Namespace: testNamespace,
				},
				Spec: keycloakApi.KeycloakClientSpec{
					ClientId: testClientId,
					RealmRef: common.RealmRef{
						Kind: keycloakApi.KeycloakRealmKind,
						Name: testRealmName,
					},
					WebUrl:     testWebUrl,
					WebOrigins: ptr.To(customOrigins),
				},
			}

			Expect(k8sClient.Create(ctx, keycloakClient)).Should(Succeed())

			createdClient := &keycloakApi.KeycloakClient{}
			Expect(k8sClient.Get(ctx, client.ObjectKeyFromObject(keycloakClient), createdClient)).
				Should(Succeed())

			// Verify custom WebOrigins were preserved
			Expect(createdClient.Spec.WebOrigins).Should(Equal(ptr.To(customOrigins)))
		})

		It("Should keep explicit empty WebOrigins and RedirectUris on create and across a full-object Update", func() {
			keycloakClient = &keycloakApi.KeycloakClient{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-client-empty-lists",
					Namespace: testNamespace,
				},
				Spec: keycloakApi.KeycloakClientSpec{
					ClientId: testClientId,
					RealmRef: common.RealmRef{
						Kind: keycloakApi.KeycloakRealmKind,
						Name: testRealmName,
					},
					WebUrl:       testWebUrl,
					WebOrigins:   ptr.To([]string{}),
					RedirectUris: ptr.To([]string{}),
				},
			}

			Expect(k8sClient.Create(ctx, keycloakClient)).Should(Succeed())

			createdClient := &keycloakApi.KeycloakClient{}
			Expect(k8sClient.Get(ctx, client.ObjectKeyFromObject(keycloakClient), createdClient)).
				Should(Succeed())

			Expect(createdClient.Spec.WebOrigins).Should(Equal(ptr.To([]string{})))
			Expect(createdClient.Spec.RedirectUris).Should(Equal(ptr.To([]string{})))

			createdClient.Spec.Secret = "$client-secret:secretKey"
			Expect(k8sClient.Update(ctx, createdClient)).Should(Succeed())

			updatedClient := &keycloakApi.KeycloakClient{}
			Expect(k8sClient.Get(ctx, client.ObjectKeyFromObject(keycloakClient), updatedClient)).
				Should(Succeed())

			Expect(updatedClient.Spec.Secret).Should(Equal("$client-secret:secretKey"))
			Expect(updatedClient.Spec.WebOrigins).Should(Equal(ptr.To([]string{})))
			Expect(updatedClient.Spec.RedirectUris).Should(Equal(ptr.To([]string{})))
		})

		It("Should migrate ClientRoles to ClientRolesV2", func() {
			roles := []string{"role1", "role2", "role3"}
			keycloakClient = &keycloakApi.KeycloakClient{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-client-migrate-roles",
					Namespace: testNamespace,
				},
				Spec: keycloakApi.KeycloakClientSpec{
					ClientId: testClientId,
					RealmRef: common.RealmRef{
						Kind: keycloakApi.KeycloakRealmKind,
						Name: testRealmName,
					},
					WebUrl:      testWebUrl,
					ClientRoles: roles,
					// ClientRolesV2 not set - should be populated from ClientRoles
				},
			}

			Expect(k8sClient.Create(ctx, keycloakClient)).Should(Succeed())

			createdClient := &keycloakApi.KeycloakClient{}
			Expect(k8sClient.Get(ctx, client.ObjectKeyFromObject(keycloakClient), createdClient)).
				Should(Succeed())

			// Verify ClientRolesV2 was populated
			Expect(createdClient.Spec.ClientRolesV2).Should(HaveLen(3))
			for i, role := range roles {
				Expect(createdClient.Spec.ClientRolesV2[i].Name).Should(Equal(role))
			}

			// Verify original ClientRoles is preserved for backward compatibility
			Expect(createdClient.Spec.ClientRoles).Should(Equal(roles))
		})

		It("Should NOT migrate when ClientRolesV2 already has values", func() {
			oldRoles := []string{"old-role1", "old-role2"}
			newRoles := []keycloakApi.ClientRole{
				{Name: "new-role1"},
				{Name: "new-role2"},
			}

			keycloakClient = &keycloakApi.KeycloakClient{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-client-no-migrate-roles",
					Namespace: testNamespace,
				},
				Spec: keycloakApi.KeycloakClientSpec{
					ClientId: testClientId,
					RealmRef: common.RealmRef{
						Kind: keycloakApi.KeycloakRealmKind,
						Name: testRealmName,
					},
					WebUrl:        testWebUrl,
					ClientRoles:   oldRoles,
					ClientRolesV2: newRoles,
				},
			}

			Expect(k8sClient.Create(ctx, keycloakClient)).Should(Succeed())

			createdClient := &keycloakApi.KeycloakClient{}
			Expect(k8sClient.Get(ctx, client.ObjectKeyFromObject(keycloakClient), createdClient)).
				Should(Succeed())

			// Verify ClientRolesV2 was NOT overwritten
			Expect(createdClient.Spec.ClientRolesV2).Should(HaveLen(2))
			Expect(createdClient.Spec.ClientRolesV2[0].Name).Should(Equal("new-role1"))
			Expect(createdClient.Spec.ClientRolesV2[1].Name).Should(Equal("new-role2"))
		})

		It("Should migrate ServiceAccount.Attributes to AttributesV2", func() {
			attributes := map[string]string{
				"attr1": "value1",
				"attr2": "value2",
			}

			keycloakClient = &keycloakApi.KeycloakClient{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-client-migrate-sa-attrs",
					Namespace: testNamespace,
				},
				Spec: keycloakApi.KeycloakClientSpec{
					ClientId: testClientId,
					RealmRef: common.RealmRef{
						Kind: keycloakApi.KeycloakRealmKind,
						Name: testRealmName,
					},
					WebUrl: testWebUrl,
					ServiceAccount: &keycloakApi.ServiceAccount{
						Enabled:    true,
						Attributes: attributes,
						// AttributesV2 not set - should be populated from Attributes
					},
				},
			}

			Expect(k8sClient.Create(ctx, keycloakClient)).Should(Succeed())

			createdClient := &keycloakApi.KeycloakClient{}
			Expect(k8sClient.Get(ctx, client.ObjectKeyFromObject(keycloakClient), createdClient)).
				Should(Succeed())

			// Verify AttributesV2 was populated
			Expect(createdClient.Spec.ServiceAccount.AttributesV2).Should(HaveLen(2))
			Expect(createdClient.Spec.ServiceAccount.AttributesV2["attr1"]).Should(Equal([]string{"value1"}))
			Expect(createdClient.Spec.ServiceAccount.AttributesV2["attr2"]).Should(Equal([]string{"value2"}))

			// Verify original Attributes is preserved for backward compatibility
			Expect(createdClient.Spec.ServiceAccount.Attributes).Should(Equal(attributes))
		})

		It("Should NOT migrate ServiceAccount attributes when AttributesV2 exists", func() {
			oldAttributes := map[string]string{
				"old-attr": "old-value",
			}
			newAttributesV2 := map[string][]string{
				"new-attr": {"new-value1", "new-value2"},
			}

			keycloakClient = &keycloakApi.KeycloakClient{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-client-no-migrate-sa-attrs",
					Namespace: testNamespace,
				},
				Spec: keycloakApi.KeycloakClientSpec{
					ClientId: testClientId,
					RealmRef: common.RealmRef{
						Kind: keycloakApi.KeycloakRealmKind,
						Name: testRealmName,
					},
					WebUrl: testWebUrl,
					ServiceAccount: &keycloakApi.ServiceAccount{
						Enabled:      true,
						Attributes:   oldAttributes,
						AttributesV2: newAttributesV2,
					},
				},
			}

			Expect(k8sClient.Create(ctx, keycloakClient)).Should(Succeed())

			createdClient := &keycloakApi.KeycloakClient{}
			Expect(k8sClient.Get(ctx, client.ObjectKeyFromObject(keycloakClient), createdClient)).
				Should(Succeed())

			// Verify AttributesV2 was NOT overwritten
			Expect(createdClient.Spec.ServiceAccount.AttributesV2).Should(Equal(newAttributesV2))
		})

		It("Should handle nil ServiceAccount gracefully", func() {
			keycloakClient = &keycloakApi.KeycloakClient{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-client-nil-sa",
					Namespace: testNamespace,
				},
				Spec: keycloakApi.KeycloakClientSpec{
					ClientId: testClientId,
					RealmRef: common.RealmRef{
						Kind: keycloakApi.KeycloakRealmKind,
						Name: testRealmName,
					},
					WebUrl: testWebUrl,
					// ServiceAccount is nil
				},
			}

			// Should not panic
			Expect(k8sClient.Create(ctx, keycloakClient)).Should(Succeed())

			createdClient := &keycloakApi.KeycloakClient{}
			Expect(k8sClient.Get(ctx, client.ObjectKeyFromObject(keycloakClient), createdClient)).
				Should(Succeed())

			Expect(createdClient.Spec.ServiceAccount).Should(BeNil())
		})

		It("Should handle all defaults together", func() {
			roles := []string{"role1", "role2"}
			saAttributes := map[string]string{"sa-attr": "sa-value"}

			keycloakClient = &keycloakApi.KeycloakClient{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-client-all-defaults",
					Namespace: testNamespace,
				},
				Spec: keycloakApi.KeycloakClientSpec{
					ClientId: testClientId,
					RealmRef: common.RealmRef{
						Kind: keycloakApi.KeycloakRealmKind,
						Name: testRealmName,
					},
					WebUrl:      testWebUrl,
					ClientRoles: roles,
					ServiceAccount: &keycloakApi.ServiceAccount{
						Enabled:    true,
						Attributes: saAttributes,
					},
					// No Attributes, WebOrigins, ClientRolesV2, or AttributesV2 set
				},
			}

			Expect(k8sClient.Create(ctx, keycloakClient)).Should(Succeed())

			createdClient := &keycloakApi.KeycloakClient{}
			Expect(k8sClient.Get(ctx, client.ObjectKeyFromObject(keycloakClient), createdClient)).
				Should(Succeed())

			// Verify all defaults were applied
			Expect(createdClient.Spec.Attributes[ClientAttributeLogoutRedirectUris]).Should(Equal(ClientAttributeLogoutRedirectUrisDefValue))
			Expect(createdClient.Spec.WebOrigins).Should(Equal(ptr.To([]string{testWebUrl})))
			Expect(createdClient.Spec.ClientRolesV2).Should(HaveLen(2))
			Expect(createdClient.Spec.ServiceAccount.AttributesV2).Should(HaveLen(1))
		})

		It("Should NOT modify client when all defaults already set", func() {
			attributes := map[string]string{ClientAttributeLogoutRedirectUris: ClientAttributeLogoutRedirectUrisDefValue}
			webOrigins := ptr.To([]string{testWebUrl})
			clientRolesV2 := []keycloakApi.ClientRole{{Name: "role1"}}
			saAttributesV2 := map[string][]string{"attr": {"value"}}

			keycloakClient = &keycloakApi.KeycloakClient{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-client-no-changes",
					Namespace: testNamespace,
				},
				Spec: keycloakApi.KeycloakClientSpec{
					ClientId: testClientId,
					RealmRef: common.RealmRef{
						Kind: keycloakApi.KeycloakRealmKind,
						Name: testRealmName,
					},
					WebUrl:        testWebUrl,
					Attributes:    attributes,
					WebOrigins:    webOrigins,
					ClientRolesV2: clientRolesV2,
					ServiceAccount: &keycloakApi.ServiceAccount{
						Enabled:      true,
						AttributesV2: saAttributesV2,
					},
				},
			}

			Expect(k8sClient.Create(ctx, keycloakClient)).Should(Succeed())

			createdClient := &keycloakApi.KeycloakClient{}
			Expect(k8sClient.Get(ctx, client.ObjectKeyFromObject(keycloakClient), createdClient)).
				Should(Succeed())

			// Verify nothing was changed
			Expect(createdClient.Spec.Attributes).Should(Equal(attributes))
			Expect(createdClient.Spec.WebOrigins).Should(Equal(webOrigins))
			Expect(createdClient.Spec.ClientRolesV2).Should(HaveLen(1))
			Expect(createdClient.Spec.ServiceAccount.AttributesV2).Should(Equal(saAttributesV2))
		})
	})

	Context("Deprecated spec.realmRoles (admission warning)", func() {
		clientWithRealmRoles := func(roles *[]keycloakApi.RealmRole) *keycloakApi.KeycloakClient {
			return &keycloakApi.KeycloakClient{
				Spec: keycloakApi.KeycloakClientSpec{
					RealmRoles: roles, //nolint:staticcheck // covers the deprecated realmRoles field
				},
			}
		}
		roles := &[]keycloakApi.RealmRole{{Name: "app-admin", Composite: "administrator"}}

		It("returns a warning from ValidateCreate when spec.realmRoles is set", func() {
			warnings, err := (&KeycloakClientCustomValidator{}).ValidateCreate(context.Background(), clientWithRealmRoles(roles))
			Expect(err).NotTo(HaveOccurred())
			Expect(warnings).To(Equal(admission.Warnings{"spec.realmRoles is deprecated; use KeycloakRealmRole"}))
		})

		It("returns a warning from ValidateUpdate when spec.realmRoles is set on the new object", func() {
			warnings, err := (&KeycloakClientCustomValidator{}).ValidateUpdate(
				context.Background(), clientWithRealmRoles(nil), clientWithRealmRoles(roles))
			Expect(err).NotTo(HaveOccurred())
			Expect(warnings).To(HaveLen(1))
		})

		It("returns no warning when spec.realmRoles is absent or empty", func() {
			for _, r := range []*[]keycloakApi.RealmRole{nil, {}} {
				warnings, err := (&KeycloakClientCustomValidator{}).ValidateCreate(context.Background(), clientWithRealmRoles(r))
				Expect(err).NotTo(HaveOccurred())
				Expect(warnings).To(BeEmpty())
			}
		})
	})
})

func newClient(name string, mutate func(*keycloakApi.KeycloakClientSpec)) *keycloakApi.KeycloakClient {
	c := &keycloakApi.KeycloakClient{
		ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: "ns1"},
		Spec: keycloakApi.KeycloakClientSpec{
			ClientId: name,
			RealmRef: common.RealmRef{Kind: keycloakApi.KeycloakRealmKind, Name: "test-realm"},
		},
	}
	mutate(&c.Spec)

	return c
}

var _ = Describe("KeycloakClient clientRoles admission", func() {
	It("Should reject an empty clientId in serviceAccount.clientRoles", func() {
		c := newClient("sa-empty-client-id", func(s *keycloakApi.KeycloakClientSpec) {
			s.ServiceAccount = &keycloakApi.ServiceAccount{
				Enabled:     true,
				ClientRoles: []keycloakApi.UserClientRole{{ClientID: "", Roles: ptr.To([]string{"manage-account"})}},
			}
		})

		err := k8sClient.Create(ctx, c)
		Expect(k8sErrors.IsInvalid(err)).To(BeTrue(), "got: %v", err)
		Expect(err.Error()).To(ContainSubstring("spec.serviceAccount.clientRoles[0].clientId"))
	})

	It("Should reject an empty clientId in scopeMappings.clientRoles", func() {
		c := newClient("scope-empty-client-id", func(s *keycloakApi.KeycloakClientSpec) {
			s.ScopeMappings = &keycloakApi.ScopeMappings{
				ClientRoles: []keycloakApi.UserClientRole{{Roles: ptr.To([]string{"x"})}},
			}
		})

		err := k8sClient.Create(ctx, c)
		Expect(k8sErrors.IsInvalid(err)).To(BeTrue(), "got: %v", err)
		Expect(err.Error()).To(ContainSubstring("spec.scopeMappings.clientRoles[0].clientId"))
	})

	It("Should accept a non-empty clientId", func() {
		c := newClient("roles-valid-client-id", func(s *keycloakApi.KeycloakClientSpec) {
			s.ServiceAccount = &keycloakApi.ServiceAccount{
				Enabled:     true,
				ClientRoles: []keycloakApi.UserClientRole{{ClientID: "account", Roles: ptr.To([]string{"manage-account"})}},
			}
		})

		Expect(k8sClient.Create(ctx, c)).To(Succeed())
		Expect(k8sClient.Delete(ctx, c)).To(Succeed())
	})
})

var _ = Describe("KeycloakClient clientScopePolicy admission", func() {
	newClientWithPolicies := func(name string, policies ...keycloakApi.Policy) *keycloakApi.KeycloakClient {
		return newClient(name, func(s *keycloakApi.KeycloakClientSpec) {
			s.Authorization = &keycloakApi.Authorization{Policies: policies}
		})
	}

	scopePolicy := func(scopes ...keycloakApi.ClientScopeDefinition) keycloakApi.Policy {
		return keycloakApi.Policy{
			Type:             keycloakApi.PolicyTypeClientScope,
			Name:             "p",
			DecisionStrategy: keycloakApi.PolicyDecisionStrategyUnanimous,
			Logic:            keycloakApi.PolicyLogicPositive,
			ClientScopePolicy: &keycloakApi.ClientScopePolicyData{
				ClientScopes: scopes,
			},
		}
	}

	create := func(c *keycloakApi.KeycloakClient) error {
		err := k8sClient.Create(ctx, c)

		DeferCleanup(func() {
			Expect(client.IgnoreNotFound(k8sClient.Delete(ctx, c))).To(Succeed())
		})

		return err
	}

	const policyPath = "spec.authorization.policies[0].clientScopePolicy.clientScopes"

	DescribeTable("Should reject a repeated client scope name",
		func(name string, scopes ...keycloakApi.ClientScopeDefinition) {
			err := create(newClientWithPolicies(name, scopePolicy(scopes...)))

			Expect(k8sErrors.IsInvalid(err)).To(BeTrue(), "got: %v", err)
			Expect(err.Error()).To(ContainSubstring(policyPath + "[1]"))
			Expect(err.Error()).To(ContainSubstring("Duplicate value"))
			Expect(err.Error()).NotTo(ContainSubstring("admission webhook"))
		},
		Entry("required then default", "csp-dup-1",
			keycloakApi.ClientScopeDefinition{Name: "a", Required: true},
			keycloakApi.ClientScopeDefinition{Name: "a"}),
		Entry("default then required", "csp-dup-2",
			keycloakApi.ClientScopeDefinition{Name: "a"},
			keycloakApi.ClientScopeDefinition{Name: "a", Required: true}),
		Entry("identical required", "csp-dup-3",
			keycloakApi.ClientScopeDefinition{Name: "a", Required: true},
			keycloakApi.ClientScopeDefinition{Name: "a", Required: true}),
		Entry("identical default", "csp-dup-4",
			keycloakApi.ClientScopeDefinition{Name: "a"},
			keycloakApi.ClientScopeDefinition{Name: "a"}),
	)

	It("Should report the first repeat of each name", func() {
		err := create(newClientWithPolicies("csp-dup-5", scopePolicy(
			keycloakApi.ClientScopeDefinition{Name: "a"},
			keycloakApi.ClientScopeDefinition{Name: "b"},
			keycloakApi.ClientScopeDefinition{Name: "a"},
			keycloakApi.ClientScopeDefinition{Name: "b"},
		)))

		Expect(k8sErrors.IsInvalid(err)).To(BeTrue(), "got: %v", err)
		Expect(err.Error()).To(ContainSubstring("clientScopes[2]"))
		Expect(err.Error()).To(ContainSubstring("clientScopes[3]"))
	})

	It("Should reject duplicates in a later policy", func() {
		valid := scopePolicy(keycloakApi.ClientScopeDefinition{Name: "a"})
		dup := scopePolicy(
			keycloakApi.ClientScopeDefinition{Name: "a"},
			keycloakApi.ClientScopeDefinition{Name: "a"},
		)
		dup.Name = "p2"

		err := create(newClientWithPolicies("csp-dup-6", valid, dup))

		Expect(k8sErrors.IsInvalid(err)).To(BeTrue(), "got: %v", err)
		Expect(err.Error()).To(ContainSubstring("policies[1].clientScopePolicy.clientScopes[1]"))
		Expect(err.Error()).To(ContainSubstring("Duplicate value"))
	})

	It("Should reject an update that adds a duplicate", func() {
		c := newClientWithPolicies("csp-dup-7", scopePolicy(
			keycloakApi.ClientScopeDefinition{Name: "a", Required: true},
			keycloakApi.ClientScopeDefinition{Name: "b"},
		))
		Expect(create(c)).To(Succeed())

		scopes := &c.Spec.Authorization.Policies[0].ClientScopePolicy.ClientScopes
		*scopes = append(*scopes, keycloakApi.ClientScopeDefinition{Name: "a"})

		err := k8sClient.Update(ctx, c)

		Expect(k8sErrors.IsInvalid(err)).To(BeTrue(), "got: %v", err)
		Expect(err.Error()).To(ContainSubstring(policyPath + "[2]"))
	})

	It("Should accept distinct names", func() {
		Expect(create(newClientWithPolicies("csp-ok-1", scopePolicy(
			keycloakApi.ClientScopeDefinition{Name: "a", Required: true},
			keycloakApi.ClientScopeDefinition{Name: "b"},
		)))).To(Succeed())
	})

	It("Should accept a single entry", func() {
		Expect(create(newClientWithPolicies("csp-ok-2", scopePolicy(
			keycloakApi.ClientScopeDefinition{Name: "a"},
		)))).To(Succeed())
	})
})

var _ = Describe("KeycloakClient role and group policy admission", func() {
	newClientWithPolicies := func(name string, policies ...keycloakApi.Policy) *keycloakApi.KeycloakClient {
		return newClient(name, func(s *keycloakApi.KeycloakClientSpec) {
			s.Authorization = &keycloakApi.Authorization{Policies: policies}
		})
	}

	rolePolicy := func(roles ...keycloakApi.RoleDefinition) keycloakApi.Policy {
		return keycloakApi.Policy{
			Type:             keycloakApi.PolicyTypeRole,
			Name:             "p",
			DecisionStrategy: keycloakApi.PolicyDecisionStrategyUnanimous,
			Logic:            keycloakApi.PolicyLogicPositive,
			RolePolicy:       &keycloakApi.RolePolicyData{Roles: roles},
		}
	}

	groupPolicy := func(groups ...keycloakApi.GroupDefinition) keycloakApi.Policy {
		return keycloakApi.Policy{
			Type:             keycloakApi.PolicyTypeGroup,
			Name:             "p",
			DecisionStrategy: keycloakApi.PolicyDecisionStrategyUnanimous,
			Logic:            keycloakApi.PolicyLogicPositive,
			GroupPolicy:      &keycloakApi.GroupPolicyData{Groups: groups},
		}
	}

	create := func(c *keycloakApi.KeycloakClient) error {
		err := k8sClient.Create(ctx, c)

		DeferCleanup(func() {
			Expect(client.IgnoreNotFound(k8sClient.Delete(ctx, c))).To(Succeed())
		})

		return err
	}

	const (
		rolesPath  = "spec.authorization.policies[0].rolePolicy.roles"
		groupsPath = "spec.authorization.policies[0].groupPolicy.groups"
	)

	DescribeTable("Should reject a repeated role name",
		func(name string, roles ...keycloakApi.RoleDefinition) {
			err := create(newClientWithPolicies(name, rolePolicy(roles...)))

			Expect(k8sErrors.IsInvalid(err)).To(BeTrue(), "got: %v", err)
			Expect(err.Error()).To(ContainSubstring(rolesPath + "[1]"))
			Expect(err.Error()).To(ContainSubstring("Duplicate value"))
			Expect(err.Error()).NotTo(ContainSubstring("admission webhook"))
		},
		Entry("required then default", "rgp-role-dup-1",
			keycloakApi.RoleDefinition{Name: "a", Required: true},
			keycloakApi.RoleDefinition{Name: "a"}),
		Entry("default then required", "rgp-role-dup-2",
			keycloakApi.RoleDefinition{Name: "a"},
			keycloakApi.RoleDefinition{Name: "a", Required: true}),
		Entry("identical", "rgp-role-dup-3",
			keycloakApi.RoleDefinition{Name: "a"},
			keycloakApi.RoleDefinition{Name: "a"}),
	)

	DescribeTable("Should reject a repeated group name",
		func(name string, groups ...keycloakApi.GroupDefinition) {
			err := create(newClientWithPolicies(name, groupPolicy(groups...)))

			Expect(k8sErrors.IsInvalid(err)).To(BeTrue(), "got: %v", err)
			Expect(err.Error()).To(ContainSubstring(groupsPath + "[1]"))
			Expect(err.Error()).To(ContainSubstring("Duplicate value"))
			Expect(err.Error()).NotTo(ContainSubstring("admission webhook"))
		},
		Entry("extendChildren then default", "rgp-group-dup-1",
			keycloakApi.GroupDefinition{Name: "g", ExtendChildren: true},
			keycloakApi.GroupDefinition{Name: "g"}),
		Entry("identical", "rgp-group-dup-2",
			keycloakApi.GroupDefinition{Name: "g"},
			keycloakApi.GroupDefinition{Name: "g"}),
	)

	It("Should report the first repeat of each role name", func() {
		err := create(newClientWithPolicies("rgp-role-dup-4", rolePolicy(
			keycloakApi.RoleDefinition{Name: "a"},
			keycloakApi.RoleDefinition{Name: "b"},
			keycloakApi.RoleDefinition{Name: "a"},
			keycloakApi.RoleDefinition{Name: "b"},
		)))

		Expect(k8sErrors.IsInvalid(err)).To(BeTrue(), "got: %v", err)
		Expect(err.Error()).To(ContainSubstring("roles[2]"))
		Expect(err.Error()).To(ContainSubstring("roles[3]"))
	})

	It("Should reject duplicate roles in a later policy", func() {
		valid := rolePolicy(keycloakApi.RoleDefinition{Name: "a"})
		dup := rolePolicy(
			keycloakApi.RoleDefinition{Name: "a"},
			keycloakApi.RoleDefinition{Name: "a"},
		)
		dup.Name = "p2"

		err := create(newClientWithPolicies("rgp-role-dup-5", valid, dup))

		Expect(k8sErrors.IsInvalid(err)).To(BeTrue(), "got: %v", err)
		Expect(err.Error()).To(ContainSubstring("policies[1].rolePolicy.roles[1]"))
		Expect(err.Error()).To(ContainSubstring("Duplicate value"))
	})

	It("Should reject an update that adds a duplicate role", func() {
		c := newClientWithPolicies("rgp-role-dup-6", rolePolicy(
			keycloakApi.RoleDefinition{Name: "a", Required: true},
			keycloakApi.RoleDefinition{Name: "b"},
		))
		Expect(create(c)).To(Succeed())

		roles := &c.Spec.Authorization.Policies[0].RolePolicy.Roles
		*roles = append(*roles, keycloakApi.RoleDefinition{Name: "a"})

		err := k8sClient.Update(ctx, c)

		Expect(k8sErrors.IsInvalid(err)).To(BeTrue(), "got: %v", err)
		Expect(err.Error()).To(ContainSubstring(rolesPath + "[2]"))
	})

	It("Should accept distinct role and group names", func() {
		roles := rolePolicy(
			keycloakApi.RoleDefinition{Name: "a", Required: true},
			keycloakApi.RoleDefinition{Name: "b"},
		)
		roles.Name = "p-role"

		groups := groupPolicy(
			keycloakApi.GroupDefinition{Name: "g1", ExtendChildren: true},
			keycloakApi.GroupDefinition{Name: "g2"},
		)
		groups.Name = "p-group"

		Expect(create(newClientWithPolicies("rgp-ok-1", roles, groups))).To(Succeed())
	})

	It("Should accept a single role and a single group", func() {
		roles := rolePolicy(keycloakApi.RoleDefinition{Name: "a"})
		roles.Name = "p-role"

		groups := groupPolicy(keycloakApi.GroupDefinition{Name: "g"})
		groups.Name = "p-group"

		Expect(create(newClientWithPolicies("rgp-ok-2", roles, groups))).To(Succeed())
	})
})

var _ = Describe("KeycloakClient authorization set admission", func() {
	policyWith := func(name, typ string, mutate func(*keycloakApi.Policy)) keycloakApi.Policy {
		p := keycloakApi.Policy{
			Type:             typ,
			Name:             name,
			DecisionStrategy: keycloakApi.PolicyDecisionStrategyUnanimous,
			Logic:            keycloakApi.PolicyLogicPositive,
		}
		mutate(&p)

		return p
	}

	permissionWith := func(typ string, mutate func(*keycloakApi.Permission)) keycloakApi.Permission {
		p := keycloakApi.Permission{
			Name:             "perm",
			Type:             typ,
			DecisionStrategy: keycloakApi.PolicyDecisionStrategyUnanimous,
			Logic:            keycloakApi.PolicyLogicPositive,
		}
		mutate(&p)

		return p
	}

	create := func(c *keycloakApi.KeycloakClient) error {
		err := k8sClient.Create(ctx, c)

		DeferCleanup(func() {
			Expect(client.IgnoreNotFound(k8sClient.Delete(ctx, c))).To(Succeed())
		})

		return err
	}

	DescribeTable("Should reject a repeated entry",
		func(name, path string, mutate func(*keycloakApi.Authorization)) {
			err := create(newClient(name, func(s *keycloakApi.KeycloakClientSpec) {
				s.Authorization = &keycloakApi.Authorization{}
				mutate(s.Authorization)
			}))

			Expect(k8sErrors.IsInvalid(err)).To(BeTrue(), "got: %v", err)
			Expect(err.Error()).To(ContainSubstring(path + "[1]"))
			Expect(err.Error()).To(ContainSubstring("Duplicate value"))
			Expect(err.Error()).NotTo(ContainSubstring("admission webhook"))
		},
		Entry("scopes", "az-set-dup-1", "spec.authorization.scopes",
			func(a *keycloakApi.Authorization) { a.Scopes = []string{"s", "s"} }),
		Entry("clientPolicy.clients", "az-set-dup-2", "spec.authorization.policies[0].clientPolicy.clients",
			func(a *keycloakApi.Authorization) {
				a.Policies = []keycloakApi.Policy{policyWith("p", keycloakApi.PolicyTypeClient, func(p *keycloakApi.Policy) {
					p.ClientPolicy = &keycloakApi.ClientPolicyData{Clients: []string{"c", "c"}}
				})}
			}),
		Entry("userPolicy.users", "az-set-dup-3", "spec.authorization.policies[0].userPolicy.users",
			func(a *keycloakApi.Authorization) {
				a.Policies = []keycloakApi.Policy{policyWith("p", keycloakApi.PolicyTypeUser, func(p *keycloakApi.Policy) {
					p.UserPolicy = &keycloakApi.UserPolicyData{Users: []string{"u", "u"}}
				})}
			}),
		Entry("aggregatedPolicy.policies", "az-set-dup-4", "spec.authorization.policies[0].aggregatedPolicy.policies",
			func(a *keycloakApi.Authorization) {
				a.Policies = []keycloakApi.Policy{policyWith("p", keycloakApi.PolicyTypeAggregate, func(p *keycloakApi.Policy) {
					p.AggregatedPolicy = &keycloakApi.AggregatedPolicyData{Policies: []string{"x", "x"}}
				})}
			}),
		Entry("permission policies", "az-set-dup-5", "spec.authorization.permissions[0].policies",
			func(a *keycloakApi.Authorization) {
				a.Permissions = []keycloakApi.Permission{permissionWith(keycloakApi.PermissionTypeScope, func(p *keycloakApi.Permission) {
					p.Policies = []string{"x", "x"}
				})}
			}),
		Entry("permission resources", "az-set-dup-6", "spec.authorization.permissions[0].resources",
			func(a *keycloakApi.Authorization) {
				a.Permissions = []keycloakApi.Permission{permissionWith(keycloakApi.PermissionTypeResource, func(p *keycloakApi.Permission) {
					p.Resources = []string{"r", "r"}
				})}
			}),
		Entry("permission scopes", "az-set-dup-7", "spec.authorization.permissions[0].scopes",
			func(a *keycloakApi.Authorization) {
				a.Permissions = []keycloakApi.Permission{permissionWith(keycloakApi.PermissionTypeScope, func(p *keycloakApi.Permission) {
					p.Scopes = []string{"s", "s"}
				})}
			}),
	)

	It("Should accept distinct entries and a null permission list", func() {
		c := newClient("az-set-ok-1", func(s *keycloakApi.KeycloakClientSpec) {
			s.Authorization = &keycloakApi.Authorization{
				Scopes: []string{"s1", "s2"},
				Policies: []keycloakApi.Policy{
					policyWith("p-client", keycloakApi.PolicyTypeClient, func(p *keycloakApi.Policy) {
						p.ClientPolicy = &keycloakApi.ClientPolicyData{Clients: []string{"c1", "c2"}}
					}),
					policyWith("p-user", keycloakApi.PolicyTypeUser, func(p *keycloakApi.Policy) {
						p.UserPolicy = &keycloakApi.UserPolicyData{Users: []string{"u1", "u2"}}
					}),
					policyWith("p-agg", keycloakApi.PolicyTypeAggregate, func(p *keycloakApi.Policy) {
						p.AggregatedPolicy = &keycloakApi.AggregatedPolicyData{Policies: []string{"p-client", "p-user"}}
					}),
				},
				Permissions: []keycloakApi.Permission{
					permissionWith(keycloakApi.PermissionTypeScope, func(p *keycloakApi.Permission) {
						p.Policies = []string{"p-client", "p-user"}
						p.Resources = []string{"r1", "r2"}
						p.Scopes = []string{"s1", "s2"}
					}),
					{
						Name:             "perm-nil",
						Type:             keycloakApi.PermissionTypeResource,
						DecisionStrategy: keycloakApi.PolicyDecisionStrategyUnanimous,
						Logic:            keycloakApi.PolicyLogicPositive,
					},
				},
			}
		})

		Expect(create(c)).To(Succeed())
	})

	It("Should reject an update that adds a duplicate scope", func() {
		c := newClient("az-set-dup-8", func(s *keycloakApi.KeycloakClientSpec) {
			s.Authorization = &keycloakApi.Authorization{Scopes: []string{"s1", "s2"}}
		})
		Expect(create(c)).To(Succeed())

		c.Spec.Authorization.Scopes = append(c.Spec.Authorization.Scopes, "s1")

		err := k8sClient.Update(ctx, c)

		Expect(k8sErrors.IsInvalid(err)).To(BeTrue(), "got: %v", err)
		Expect(err.Error()).To(ContainSubstring("spec.authorization.scopes[2]"))
		Expect(err.Error()).To(ContainSubstring("Duplicate value"))
	})
})

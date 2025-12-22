package v1

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

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
			Expect(createdClient.Spec.WebOrigins).Should(HaveLen(1))
			Expect(createdClient.Spec.WebOrigins[0]).Should(Equal(testWebUrl))
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

			// Verify WebOrigins was NOT initialized (remains nil or empty)
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
					WebOrigins: customOrigins,
				},
			}

			Expect(k8sClient.Create(ctx, keycloakClient)).Should(Succeed())

			createdClient := &keycloakApi.KeycloakClient{}
			Expect(k8sClient.Get(ctx, client.ObjectKeyFromObject(keycloakClient), createdClient)).
				Should(Succeed())

			// Verify custom WebOrigins were preserved
			Expect(createdClient.Spec.WebOrigins).Should(Equal(customOrigins))
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
			Expect(createdClient.Spec.WebOrigins).Should(Equal([]string{testWebUrl}))
			Expect(createdClient.Spec.ClientRolesV2).Should(HaveLen(2))
			Expect(createdClient.Spec.ServiceAccount.AttributesV2).Should(HaveLen(1))
		})

		It("Should NOT modify client when all defaults already set", func() {
			attributes := map[string]string{ClientAttributeLogoutRedirectUris: ClientAttributeLogoutRedirectUrisDefValue}
			webOrigins := []string{testWebUrl}
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
})

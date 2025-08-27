package keycloakclient

import (
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/Nerzal/gocloak/v12"
	corev1 "k8s.io/api/core/v1"
	k8sErrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	"github.com/epam/edp-keycloak-operator/api/common"
	keycloakApi "github.com/epam/edp-keycloak-operator/api/v1"
	"github.com/epam/edp-keycloak-operator/pkg/client/keycloak/adapter"
	"github.com/epam/edp-keycloak-operator/pkg/secretref"
)

var _ = Describe("KeycloakClient controller", Ordered, func() {
	var clientSecret *corev1.Secret
	It("Should create client secret", func() {
		clientSecret = &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-keycloak-client-secret",
				Namespace: ns,
			},
			Data: map[string][]byte{
				"secretKey": []byte("secretValue"),
			},
		}
		Expect(k8sClient.Create(ctx, clientSecret)).Should(Succeed(), "failed to create client secret")
	})

	It("Should create KeycloakClient with secret reference", func() {
		By("Checking feature flag ADMIN_FINE_GRAINED_AUTHZ")
		By("Creating a KeycloakClient to check feature flag")
		keycloakApiClient, err := controllerHelper.CreateKeycloakClientFromRealmRef(
			ctx,
			&keycloakApi.KeycloakClient{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-keycloak-client-to-check-feature-flag",
					Namespace: ns,
				},
				Spec: keycloakApi.KeycloakClientSpec{
					RealmRef: common.RealmRef{
						Name: KeycloakRealmCR,
						Kind: keycloakApi.KeycloakRealmKind,
					},
				},
			},
		)
		Expect(err).ShouldNot(HaveOccurred())

		featureFlagEnabled, err := keycloakApiClient.FeatureFlagEnabled(ctx, "ADMIN_FINE_GRAINED_AUTHZ")
		Expect(err).ShouldNot(HaveOccurred())
		Expect(featureFlagEnabled).Should(BeTrue(), "Feature flag ADMIN_FINE_GRAINED_AUTHZ should be enabled")

		By("Creating a KeycloakClient")
		keycloakClient := &keycloakApi.KeycloakClient{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-keycloak-client-with-secret-ref",
				Namespace: ns,
			},
			Spec: keycloakApi.KeycloakClientSpec{
				ClientId: "test-keycloak-client-with-secret-ref",
				RealmRef: common.RealmRef{
					Name: KeycloakRealmCR,
					Kind: keycloakApi.KeycloakRealmKind,
				},
				Secret:   secretref.GenerateSecretRef(clientSecret.Name, "secretKey"),
				Public:   true,
				WebUrl:   "https://test-keycloak-client-with-secret-ref",
				AdminUrl: "https://test-keycloak-client-admin",
				HomeUrl:  "/home/",
				Attributes: map[string]string{
					"post.logout.redirect.uris": "+",
				},
				ClientRoles:             []string{"administrator", "developer"},
				RedirectUris:            []string{"https://test-keycloak-client-with-secret-ref"},
				WebOrigins:              []string{"https://test-keycloak-client-with-secret-ref"},
				Description:             "test description",
				Enabled:                 true,
				FullScopeAllowed:        true,
				Name:                    "test name",
				StandardFlowEnabled:     true,
				ClientAuthenticatorType: "client-secret",
				AuthenticationFlowBindingOverrides: &keycloakApi.AuthenticationFlowBindingOverrides{
					Browser:     "browser",
					DirectGrant: "direct grant",
				},
				AdminFineGrainedPermissionsEnabled: true,
			},
		}

		Expect(k8sClient.Create(ctx, keycloakClient)).Should(Succeed())
		Eventually(func() bool {
			createdKeycloakClient := &keycloakApi.KeycloakClient{}
			err := k8sClient.Get(ctx, types.NamespacedName{Name: keycloakClient.Name, Namespace: ns}, createdKeycloakClient)
			if err != nil {
				return false
			}

			return createdKeycloakClient.Status.Value == common.StatusOK &&
				createdKeycloakClient.Status.ClientID != ""
		}, timeout, interval).Should(BeTrue(), "KeycloakClient should be created successfully")
	})
	It("Should delete KeycloakClient", func() {
		By("Getting KeycloakClient")
		keycloakClient := &keycloakApi.KeycloakClient{}
		Expect(k8sClient.Get(ctx, types.NamespacedName{Namespace: ns, Name: "test-keycloak-client-with-secret-ref"}, keycloakClient)).Should(Succeed())
		By("Deleting KeycloakClient")
		Expect(k8sClient.Delete(ctx, keycloakClient)).Should(Succeed())
		Eventually(func() bool {
			deletedKeycloakClient := &keycloakApi.KeycloakClient{}
			err := k8sClient.Get(ctx, types.NamespacedName{Name: keycloakClient.Name, Namespace: ns}, deletedKeycloakClient)

			return k8sErrors.IsNotFound(err)
		}, timeout, interval).Should(BeTrue(), "KeycloakClient should be deleted")
	})
	It("Should create KeycloakClient with client roles v2", func() {
		By("Creating a KeycloakClient")
		keycloakClient := &keycloakApi.KeycloakClient{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-keycloak-client-with-client-roles-v2",
				Namespace: ns,
			},
			Spec: keycloakApi.KeycloakClientSpec{
				ClientId: "test-keycloak-client-with-client-roles-v2",
				RealmRef: common.RealmRef{
					Name: KeycloakRealmCR,
					Kind: keycloakApi.KeycloakRealmKind,
				},
				Enabled: true,
				Public:  true,
				Secret:  secretref.GenerateSecretRef(clientSecret.Name, "secretKey"),
				ClientRolesV2: []keycloakApi.ClientRole{
					{
						Name:                  "roleA",
						Description:           "Role A",
						AssociatedClientRoles: []string{"roleB"},
					},
					{
						Name:        "roleB",
						Description: "Role B",
					},
				},
			},
		}
		Expect(k8sClient.Create(ctx, keycloakClient)).Should(Succeed())
		Eventually(func(g Gomega) {
			createdKeycloakClient := &keycloakApi.KeycloakClient{}
			err := k8sClient.Get(ctx, types.NamespacedName{Name: keycloakClient.Name, Namespace: ns}, createdKeycloakClient)
			g.Expect(err).ShouldNot(HaveOccurred())
			g.Expect(createdKeycloakClient.Status.Value).Should(Equal(common.StatusOK))
		}).WithTimeout(timeout).WithPolling(interval).Should(Succeed())

		By("Checking client roles")
		createdKeycloakClient := &keycloakApi.KeycloakClient{}
		Expect(k8sClient.Get(ctx, types.NamespacedName{Name: keycloakClient.Name, Namespace: ns}, createdKeycloakClient)).ShouldNot(HaveOccurred())
		Expect(createdKeycloakClient.Status.ClientID).Should(Not(BeEmpty()))

		roles, err := keycloakApiClient.GetClientRoles(ctx, getKeyCloakToken(), KeycloakRealmCR, createdKeycloakClient.Status.ClientID, gocloak.GetRoleParams{})
		Expect(err).ShouldNot(HaveOccurred())
		Expect(roles).Should(HaveLen(2))

		var indexOfRoleA int
		for i, role := range roles {
			if *role.Name == "roleA" {
				indexOfRoleA = i
			}
			Expect(*role.Name).Should(BeElementOf([]string{"roleA", "roleB"}))
			Expect(*role.Description).Should(BeElementOf([]string{"Role A", "Role B"}))
		}

		compositeRoles, err := keycloakApiClient.GetCompositeRolesByRoleID(ctx, getKeyCloakToken(), KeycloakRealmCR, *roles[indexOfRoleA].ID)
		Expect(err).ShouldNot(HaveOccurred())
		Expect(compositeRoles).Should(HaveLen(1))
		Expect(*compositeRoles[0].Name).Should(Equal("roleB"))

		By("Updating client roles")
		createdKeycloakClient.Spec.ClientRolesV2 = []keycloakApi.ClientRole{
			{
				Name:        "roleA",
				Description: "Role A updated",
			},
		}
		Expect(k8sClient.Update(ctx, createdKeycloakClient)).Should(Succeed())
		Consistently(func(g Gomega) {
			updatedKeycloakClient := &keycloakApi.KeycloakClient{}
			g.Expect(k8sClient.Get(ctx, types.NamespacedName{Name: keycloakClient.Name, Namespace: ns}, updatedKeycloakClient)).ShouldNot(HaveOccurred())
			g.Expect(updatedKeycloakClient.Status.Value).Should(Equal(common.StatusOK))
		}).WithTimeout(time.Second * 3).WithPolling(time.Second).Should(Succeed())

		By("Checking client roles")
		roles, err = keycloakApiClient.GetClientRoles(ctx, getKeyCloakToken(), KeycloakRealmCR, createdKeycloakClient.Status.ClientID, gocloak.GetRoleParams{})
		Expect(err).ShouldNot(HaveOccurred())
		Expect(roles).Should(HaveLen(1))
		Expect(*roles[0].Name).Should(Equal("roleA"))
		Expect(*roles[0].Description).Should(Equal("Role A updated"))

		compositeRoles, err = keycloakApiClient.GetCompositeRolesByRoleID(ctx, getKeyCloakToken(), KeycloakRealmCR, *roles[0].ID)
		Expect(err).ShouldNot(HaveOccurred())
		Expect(compositeRoles).Should(BeEmpty())
	})

	It("Should create KeycloakClient with empty secret", func() {
		By("Creating keycloak api client")
		client := gocloak.NewClient(keycloakURL)
		token, err := client.LoginAdmin(ctx, "admin", "admin", "master")
		Expect(err).ShouldNot(HaveOccurred())
		By("Creating group for service account")
		_, err = client.CreateGroup(ctx, token.AccessToken, KeycloakRealmCR, gocloak.Group{
			Name: gocloak.StringP("test-group"),
		})
		Expect(adapter.SkipAlreadyExistsErr(err)).ShouldNot(HaveOccurred())
		By("Creating a KeycloakClient")
		keycloakClient := &keycloakApi.KeycloakClient{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-keycloak-client-with-empty-secret",
				Namespace: ns,
			},
			Spec: keycloakApi.KeycloakClientSpec{
				ClientId: "test-keycloak-client-with-empty-secret",
				RealmRef: common.RealmRef{
					Name: KeycloakRealmCR,
					Kind: keycloakApi.KeycloakRealmKind,
				},
				AuthenticationFlowBindingOverrides: &keycloakApi.AuthenticationFlowBindingOverrides{
					Browser:     "browser",
					DirectGrant: "direct grant",
				},
				ServiceAccount: &keycloakApi.ServiceAccount{
					Enabled: true,
					Groups:  []string{"test-group"},
				},
			},
		}
		Expect(k8sClient.Create(ctx, keycloakClient)).Should(Succeed())
		Eventually(func() bool {
			createdKeycloakClient := &keycloakApi.KeycloakClient{}
			err := k8sClient.Get(ctx, types.NamespacedName{Name: keycloakClient.Name, Namespace: ns}, createdKeycloakClient)
			if err != nil {
				return false
			}

			return createdKeycloakClient.Status.Value == common.StatusOK &&
				createdKeycloakClient.Status.ClientID != ""
		}, timeout, interval).Should(BeTrue(), "KeycloakClient should be created successfully")
	})
	It("Should create KeycloakClient with direct secret name", func() {
		By("Creating a client secret")
		clientSecret := &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-keycloak-client-secret2",
				Namespace: ns,
			},
			Data: map[string][]byte{
				keycloakApi.ClientSecretKey: []byte("secretValue"),
			},
		}
		Expect(k8sClient.Create(ctx, clientSecret)).Should(Succeed())
		By("Creating a KeycloakClient")
		keycloakClient := &keycloakApi.KeycloakClient{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-keycloak-client-with-direct-secret-name",
				Namespace: ns,
			},
			Spec: keycloakApi.KeycloakClientSpec{
				ClientId: "test-keycloak-client-with-direct-secret-name",
				RealmRef: common.RealmRef{
					Name: KeycloakRealmCR,
					Kind: keycloakApi.KeycloakRealmKind,
				},
				Secret: clientSecret.Name,
				AuthenticationFlowBindingOverrides: &keycloakApi.AuthenticationFlowBindingOverrides{
					Browser:     "browser",
					DirectGrant: "direct grant",
				},
				ServiceAccount: &keycloakApi.ServiceAccount{
					Enabled: true,
				},
			},
		}
		Expect(k8sClient.Create(ctx, keycloakClient)).Should(Succeed())
		Eventually(func() bool {
			createdKeycloakClient := &keycloakApi.KeycloakClient{}
			err := k8sClient.Get(ctx, types.NamespacedName{Name: keycloakClient.Name, Namespace: ns}, createdKeycloakClient)
			if err != nil {
				return false
			}

			return createdKeycloakClient.Status.Value == common.StatusOK &&
				createdKeycloakClient.Status.ClientID != ""
		}, timeout, interval).Should(BeTrue(), "KeycloakClient should be created successfully")
	})
	It("Should create KeycloakClient with authorization settings", func() {
		By("Creating a client secret")
		clientSecret := &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-keycloak-client-secret3",
				Namespace: ns,
			},
			Data: map[string][]byte{
				keycloakApi.ClientSecretKey: []byte("secretValue"),
			},
		}
		Expect(k8sClient.Create(ctx, clientSecret)).Should(Succeed())
		By("Creating keycloak api client")
		client := gocloak.NewClient(keycloakURL)
		token, err := client.LoginAdmin(ctx, "admin", "admin", "master")
		Expect(err).ShouldNot(HaveOccurred())

		By("Creating group for policy")
		_, err = client.CreateGroup(ctx, token.AccessToken, KeycloakRealmCR, gocloak.Group{
			Name: gocloak.StringP("test-policy-group"),
		})
		Expect(adapter.SkipAlreadyExistsErr(err)).ShouldNot(HaveOccurred())

		By("Creating role for policy")
		_, err = client.CreateRealmRole(ctx, token.AccessToken, KeycloakRealmCR, gocloak.Role{
			Name: gocloak.StringP("test-policy-role"),
		})
		Expect(adapter.SkipAlreadyExistsErr(err)).ShouldNot(HaveOccurred())

		By("Creating user for policy")
		_, err = client.CreateUser(ctx, token.AccessToken, KeycloakRealmCR, gocloak.User{
			Username: gocloak.StringP("test-policy-user"),
			Enabled:  gocloak.BoolP(true),
		})
		Expect(adapter.SkipAlreadyExistsErr(err)).ShouldNot(HaveOccurred())

		By("Creating a KeycloakClient")
		keycloakClient := &keycloakApi.KeycloakClient{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-keycloak-client-with-authorization",
				Namespace: ns,
			},
			Spec: keycloakApi.KeycloakClientSpec{
				ClientId: "test-keycloak-client-with-authorization",
				RealmRef: common.RealmRef{
					Name: KeycloakRealmCR,
					Kind: keycloakApi.KeycloakRealmKind,
				},
				Secret:                       clientSecret.Name,
				DirectAccess:                 true,
				AuthorizationServicesEnabled: true,
				ServiceAccount: &keycloakApi.ServiceAccount{
					Enabled: true,
				},
				AuthenticationFlowBindingOverrides: &keycloakApi.AuthenticationFlowBindingOverrides{
					Browser:     "browser",
					DirectGrant: "direct grant",
				},
				Authorization: &keycloakApi.Authorization{
					Policies: []keycloakApi.Policy{
						{
							Name:             "client-policy",
							Type:             keycloakApi.PolicyTypeClient,
							DecisionStrategy: keycloakApi.PolicyDecisionStrategyUnanimous,
							Logic:            keycloakApi.PolicyLogicPositive,
							ClientPolicy: &keycloakApi.ClientPolicyData{
								Clients: []string{"account"},
							},
						},
						{
							Name:             "group-policy",
							Description:      "Group policy",
							Type:             keycloakApi.PolicyTypeGroup,
							DecisionStrategy: keycloakApi.PolicyDecisionStrategyUnanimous,
							Logic:            keycloakApi.PolicyLogicPositive,
							GroupPolicy: &keycloakApi.GroupPolicyData{
								Groups: []keycloakApi.GroupDefinition{
									{
										Name: "test-policy-group",
									},
								},
							},
						},
						{
							Name:             "role-policy",
							Description:      "Role policy",
							Type:             keycloakApi.PolicyTypeRole,
							DecisionStrategy: keycloakApi.PolicyDecisionStrategyUnanimous,
							Logic:            keycloakApi.PolicyLogicPositive,
							RolePolicy: &keycloakApi.RolePolicyData{
								Roles: []keycloakApi.RoleDefinition{
									{
										Name:     "test-policy-role",
										Required: true,
									},
								},
							},
						},
						{
							Name:             "time-policy",
							Description:      "Time policy",
							Type:             keycloakApi.PolicyTypeTime,
							DecisionStrategy: keycloakApi.PolicyDecisionStrategyUnanimous,
							Logic:            keycloakApi.PolicyLogicPositive,
							TimePolicy: &keycloakApi.TimePolicyData{
								NotBefore:    "2024-03-03 00:00:00",
								NotOnOrAfter: "2024-06-19 00:00:00",
							},
						},
						{
							Name:             "user-policy",
							Description:      "User policy",
							Type:             keycloakApi.PolicyTypeUser,
							DecisionStrategy: keycloakApi.PolicyDecisionStrategyUnanimous,
							Logic:            keycloakApi.PolicyLogicPositive,
							UserPolicy: &keycloakApi.UserPolicyData{
								Users: []string{"test-policy-user"},
							},
						},
					},
					Permissions: []keycloakApi.Permission{},
					Scopes:      []string{"test-scope"},
					Resources: []keycloakApi.Resource{
						{
							Name:               "test-resource",
							DisplayName:        "Test resource",
							Type:               "test",
							IconURI:            "https://example.com/icon.png",
							OwnerManagedAccess: true,
							URIs:               []string{"https://example.com"},
							Attributes:         map[string][]string{"test": {"test-value"}},
							Scopes:             []string{"test-scope"},
						},
					},
				},
			},
		}
		Expect(k8sClient.Create(ctx, keycloakClient)).Should(Succeed())
		Eventually(func() bool {
			createdKeycloakClient := &keycloakApi.KeycloakClient{}
			if err = k8sClient.Get(ctx, types.NamespacedName{Name: keycloakClient.Name, Namespace: ns}, createdKeycloakClient); err != nil {
				return false
			}

			return createdKeycloakClient.Status.Value == common.StatusOK &&
				createdKeycloakClient.Status.ClientID != ""
		}, timeout, interval).Should(BeTrue(), "KeycloakClient should be created successfully")

		By("Adding client permissions")

		By("Getting Client")
		clients, err := client.GetClients(ctx, token.AccessToken, KeycloakRealmCR, gocloak.GetClientsParams{
			ClientID: gocloak.StringP(keycloakClient.Spec.ClientId),
		})
		Expect(adapter.SkipAlreadyExistsErr(err)).ShouldNot(HaveOccurred())
		Expect(clients).Should(HaveLen(1))

		cl := clients[0]

		By("Creating scope for permission")
		_, err = client.CreateScope(ctx, token.AccessToken, KeycloakRealmCR, *cl.ID,
			gocloak.ScopeRepresentation{
				Name: gocloak.StringP("test-scope"),
			})
		Expect(adapter.SkipAlreadyExistsErr(err)).ShouldNot(HaveOccurred())

		By("Creating resource for permission")
		_, err = client.CreateResource(ctx, token.AccessToken, KeycloakRealmCR, *cl.ID,
			gocloak.ResourceRepresentation{
				Name:               gocloak.StringP("test-resource"),
				OwnerManagedAccess: gocloak.BoolP(false),
			})
		Expect(adapter.SkipAlreadyExistsErr(err)).ShouldNot(HaveOccurred())

		By("Getting KeycloakClient for update")
		clientToUpdate := &keycloakApi.KeycloakClient{}
		Expect(k8sClient.Get(ctx, types.NamespacedName{Namespace: ns, Name: keycloakClient.Name}, clientToUpdate)).Should(Succeed())

		clientToUpdate.Spec.Authorization.Permissions = []keycloakApi.Permission{
			{
				Name:             "scope-permission",
				Type:             keycloakApi.PermissionTypeScope,
				DecisionStrategy: keycloakApi.PolicyDecisionStrategyConsensus,
				Description:      "Scope permission",
				Logic:            keycloakApi.PolicyLogicNegative,
				Policies:         []string{"client-policy"},
				Scopes:           []string{"test-scope"},
			},
			{
				Name:             "resource-permission",
				Type:             keycloakApi.PermissionTypeResource,
				DecisionStrategy: keycloakApi.PolicyDecisionStrategyAffirmative,
				Description:      "Resource permission",
				Logic:            keycloakApi.PolicyLogicPositive,
				Policies:         []string{"client-policy"},
				Resources:        []string{"test-resource"},
			},
		}
		clientToUpdate.Spec.Authorization.Resources = []keycloakApi.Resource{}

		By("Waiting for the KeycloakClient will be processed at least once")
		time.Sleep(5 * time.Second)
		Eventually(func() bool {
			createdKeycloakClient := &keycloakApi.KeycloakClient{}
			err := k8sClient.Get(ctx, types.NamespacedName{Name: keycloakClient.Name, Namespace: ns}, createdKeycloakClient)
			if err != nil {
				return false
			}

			return createdKeycloakClient.Status.Value == common.StatusOK &&
				createdKeycloakClient.Status.ClientID != ""
		}, timeout, interval).Should(BeTrue(), "KeycloakClient should be updated successfully")
	})

	It("Should successfully delete KeycloakClient if ErrKeycloakRealmNotFound occurs", func() {
		By("By creating a KeycloakRealm")
		testRealm := &keycloakApi.KeycloakRealm{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-realm",
				Namespace: ns,
			},
			Spec: keycloakApi.KeycloakRealmSpec{
				RealmName: "test-realm",
				KeycloakRef: common.KeycloakRef{
					Kind: keycloakApi.KeycloakKind,
					Name: KeycloakCR,
				},
			},
		}
		Expect(k8sClient.Create(ctx, testRealm)).Should(Succeed())

		By("Creating a KeycloakClient")
		keycloakClient := &keycloakApi.KeycloakClient{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-keycloak-client",
				Namespace: ns,
			},
			Spec: keycloakApi.KeycloakClientSpec{
				RealmRef: common.RealmRef{Name: "test-realm"},
				ClientId: "test-client-id",
			},
		}
		Expect(k8sClient.Create(ctx, keycloakClient)).Should(Succeed())

		By("Waiting for KeycloakClient to be ready")
		Eventually(func() bool {
			var c keycloakApi.KeycloakClient
			err := k8sClient.Get(ctx, types.NamespacedName{Name: keycloakClient.Name, Namespace: ns}, &c)
			return err == nil && controllerutil.ContainsFinalizer(&c, keyCloakClientOperatorFinalizerName)
		}, timeout, interval).Should(BeTrue())

		By("Deleting KeycloakRealm")
		Expect(k8sClient.Delete(ctx, testRealm)).Should(Succeed())

		By("Deleting KeycloakClient")
		Expect(k8sClient.Delete(ctx, keycloakClient)).Should(Succeed())

		By("Waiting for KeycloakClient to be deleted")
		Eventually(func() bool {
			var c keycloakApi.KeycloakClient
			err := k8sClient.Get(ctx, types.NamespacedName{Name: keycloakClient.Name, Namespace: ns}, &c)
			return k8sErrors.IsNotFound(err)
		}, timeout, interval).Should(BeTrue())
	})
})

package keycloakclient

import (
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	corev1 "k8s.io/api/core/v1"
	k8sErrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/utils/ptr"

	"github.com/epam/edp-keycloak-operator/api/common"
	keycloakApi "github.com/epam/edp-keycloak-operator/api/v1"
	"github.com/epam/edp-keycloak-operator/internal/controller/keycloakclient/chain"
	"github.com/epam/edp-keycloak-operator/pkg/client/keycloakapi"
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

		featureFlagEnabled, err := keycloakAdmin.Server.FeatureFlagEnabled(ctx, "ADMIN_FINE_GRAINED_AUTHZ")
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
		Eventually(func(g Gomega) {
			createdKeycloakClient := &keycloakApi.KeycloakClient{}
			err := k8sClient.Get(ctx, types.NamespacedName{Name: keycloakClient.Name, Namespace: ns}, createdKeycloakClient)
			g.Expect(err).ShouldNot(HaveOccurred())

			// Check backward compatibility status field
			g.Expect(createdKeycloakClient.Status.Value).Should(Equal(common.StatusOK))
			g.Expect(createdKeycloakClient.Status.ClientID).ShouldNot(BeEmpty())

			// Check Ready condition
			readyCond := meta.FindStatusCondition(createdKeycloakClient.Status.Conditions, chain.ConditionReady)
			g.Expect(readyCond).ShouldNot(BeNil(), "Ready condition should be set")
			g.Expect(readyCond.Status).Should(Equal(metav1.ConditionTrue), "Ready condition should be True")
			g.Expect(readyCond.Reason).Should(Equal(chain.ReasonReconciliationSucceeded))

			// Check that key chain step conditions are set
			clientSyncedCond := meta.FindStatusCondition(createdKeycloakClient.Status.Conditions, chain.ConditionClientSynced)
			g.Expect(clientSyncedCond).ShouldNot(BeNil(), "ClientSynced condition should be set")
			g.Expect(clientSyncedCond.Status).Should(Equal(metav1.ConditionTrue), "ClientSynced condition should be True")
		}, timeout, interval).Should(Succeed(), "KeycloakClient should be created successfully")
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

			// Check Ready condition
			readyCond := meta.FindStatusCondition(createdKeycloakClient.Status.Conditions, chain.ConditionReady)
			g.Expect(readyCond).ShouldNot(BeNil(), "Ready condition should be set")
			g.Expect(readyCond.Status).Should(Equal(metav1.ConditionTrue), "Ready condition should be True")

			// Check ClientRolesSynced condition
			clientRolesCond := meta.FindStatusCondition(createdKeycloakClient.Status.Conditions, chain.ConditionClientRolesSynced)
			g.Expect(clientRolesCond).ShouldNot(BeNil(), "ClientRolesSynced condition should be set")
			g.Expect(clientRolesCond.Status).Should(Equal(metav1.ConditionTrue), "ClientRolesSynced condition should be True")
		}).WithTimeout(timeout).WithPolling(interval).Should(Succeed())

		By("Checking client roles")
		createdKeycloakClient := &keycloakApi.KeycloakClient{}
		Expect(k8sClient.Get(ctx, types.NamespacedName{Name: keycloakClient.Name, Namespace: ns}, createdKeycloakClient)).ShouldNot(HaveOccurred())
		Expect(createdKeycloakClient.Status.ClientID).Should(Not(BeEmpty()))

		// Get client UUID
		existingClient, _, err := keycloakAdmin.Clients.GetClientByClientID(ctx, KeycloakRealmCR, createdKeycloakClient.Spec.ClientId)
		Expect(err).ShouldNot(HaveOccurred())
		Expect(existingClient).ShouldNot(BeNil())
		Expect(existingClient.Id).ShouldNot(BeNil())

		clientUUID := *existingClient.Id

		roles, _, err := keycloakAdmin.Clients.GetClientRoles(ctx, KeycloakRealmCR, clientUUID, nil)
		Expect(err).ShouldNot(HaveOccurred())
		Expect(roles).Should(HaveLen(2))

		roleNames := make([]string, 0, len(roles))
		roleDescriptions := make([]string, 0, len(roles))

		for _, role := range roles {
			if role.Name != nil {
				roleNames = append(roleNames, *role.Name)
			}

			if role.Description != nil {
				roleDescriptions = append(roleDescriptions, *role.Description)
			}
		}

		Expect(roleNames).Should(ConsistOf("roleA", "roleB"))
		Expect(roleDescriptions).Should(ConsistOf("Role A", "Role B"))

		compositeRoles, _, err := keycloakAdmin.Clients.GetClientRoleComposites(ctx, KeycloakRealmCR, clientUUID, "roleA")
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
		roles, _, err = keycloakAdmin.Clients.GetClientRoles(ctx, KeycloakRealmCR, clientUUID, nil)
		Expect(err).ShouldNot(HaveOccurred())
		Expect(roles).Should(HaveLen(1))
		Expect(*roles[0].Name).Should(Equal("roleA"))
		Expect(*roles[0].Description).Should(Equal("Role A updated"))

		compositeRoles, _, err = keycloakAdmin.Clients.GetClientRoleComposites(ctx, KeycloakRealmCR, clientUUID, "roleA")
		Expect(err).ShouldNot(HaveOccurred())
		Expect(compositeRoles).Should(BeEmpty())
	})

	It("Should create KeycloakClient with empty secret", func() {
		By("Creating group for service account")
		_, err := keycloakAdmin.Groups.CreateGroup(ctx, KeycloakRealmCR, keycloakapi.GroupRepresentation{
			Name: ptr.To("test-group"),
		})
		Expect(keycloakapi.SkipConflict(err)).ShouldNot(HaveOccurred())
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
		Eventually(func(g Gomega) {
			createdKeycloakClient := &keycloakApi.KeycloakClient{}
			g.Expect(k8sClient.Get(ctx, types.NamespacedName{Name: keycloakClient.Name, Namespace: ns}, createdKeycloakClient)).Should(Succeed())
			g.Expect(createdKeycloakClient.Status.Value).Should(Equal(common.StatusOK))
			g.Expect(createdKeycloakClient.Status.ClientID).ShouldNot(BeEmpty())
		}, timeout, interval).Should(Succeed(), "KeycloakClient should be created successfully")
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
		Eventually(func(g Gomega) {
			createdKeycloakClient := &keycloakApi.KeycloakClient{}
			g.Expect(k8sClient.Get(ctx, types.NamespacedName{Name: keycloakClient.Name, Namespace: ns}, createdKeycloakClient)).Should(Succeed())
			g.Expect(createdKeycloakClient.Status.Value).Should(Equal(common.StatusOK))
			g.Expect(createdKeycloakClient.Status.ClientID).ShouldNot(BeEmpty())
		}, timeout, interval).Should(Succeed(), "KeycloakClient should be created successfully")
	})
	It("Should create KeycloakClient and verify all client fields in Keycloak", func() {
		By("Creating a KeycloakClient with all PutClient fields")
		keycloakClient := &keycloakApi.KeycloakClient{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-keycloak-client-verify-fields",
				Namespace: ns,
			},
			Spec: keycloakApi.KeycloakClientSpec{
				ClientId: "test-keycloak-client-verify-fields",
				RealmRef: common.RealmRef{
					Name: KeycloakRealmCR,
					Kind: keycloakApi.KeycloakRealmKind,
				},
				Secret:                  secretref.GenerateSecretRef(clientSecret.Name, "secretKey"),
				Public:                  true,
				WebUrl:                  "https://test-verify-fields.example.com",
				AdminUrl:                "https://test-verify-fields-admin.example.com",
				HomeUrl:                 "https://test-verify-fields-home.example.com",
				Protocol:                ptr.To("openid-connect"),
				Name:                    "Verify Fields Client",
				Description:             "Test all client fields",
				Enabled:                 true,
				FullScopeAllowed:        true,
				StandardFlowEnabled:     true,
				ImplicitFlowEnabled:     true,
				FrontChannelLogout:      true,
				ConsentRequired:         true,
				SurrogateAuthRequired:   true,
				ClientAuthenticatorType: "client-secret",
				Attributes: map[string]string{
					"post.logout.redirect.uris": "+",
				},
				RedirectUris: []string{"https://test-verify-fields.example.com/*"},
				WebOrigins:   []string{"https://test-verify-fields.example.com"},
				AuthenticationFlowBindingOverrides: &keycloakApi.AuthenticationFlowBindingOverrides{
					Browser:     "browser",
					DirectGrant: "direct grant",
				},
			},
		}
		Expect(k8sClient.Create(ctx, keycloakClient)).Should(Succeed())
		Eventually(func(g Gomega) {
			createdKeycloakClient := &keycloakApi.KeycloakClient{}
			err := k8sClient.Get(ctx, types.NamespacedName{Name: keycloakClient.Name, Namespace: ns}, createdKeycloakClient)
			g.Expect(err).ShouldNot(HaveOccurred())
			g.Expect(createdKeycloakClient.Status.Value).Should(Equal(common.StatusOK))
		}, timeout, interval).Should(Succeed())

		By("Verifying all client fields in Keycloak")
		kcClient, _, err := keycloakAdmin.Clients.GetClientByClientID(ctx, KeycloakRealmCR, keycloakClient.Spec.ClientId)
		Expect(err).ShouldNot(HaveOccurred())
		Expect(kcClient).ShouldNot(BeNil())

		Expect(kcClient.Name).ShouldNot(BeNil())
		Expect(*kcClient.Name).Should(Equal("Verify Fields Client"))

		Expect(kcClient.Description).ShouldNot(BeNil())
		Expect(*kcClient.Description).Should(Equal("Test all client fields"))

		Expect(kcClient.Enabled).ShouldNot(BeNil())
		Expect(*kcClient.Enabled).Should(BeTrue())

		Expect(kcClient.PublicClient).ShouldNot(BeNil())
		Expect(*kcClient.PublicClient).Should(BeTrue())

		Expect(kcClient.Protocol).ShouldNot(BeNil())
		Expect(*kcClient.Protocol).Should(Equal("openid-connect"))

		Expect(kcClient.FullScopeAllowed).ShouldNot(BeNil())
		Expect(*kcClient.FullScopeAllowed).Should(BeTrue())

		Expect(kcClient.StandardFlowEnabled).ShouldNot(BeNil())
		Expect(*kcClient.StandardFlowEnabled).Should(BeTrue())

		Expect(kcClient.ImplicitFlowEnabled).ShouldNot(BeNil())
		Expect(*kcClient.ImplicitFlowEnabled).Should(BeTrue())

		Expect(kcClient.FrontchannelLogout).ShouldNot(BeNil())
		Expect(*kcClient.FrontchannelLogout).Should(BeTrue())

		Expect(kcClient.ConsentRequired).ShouldNot(BeNil())
		Expect(*kcClient.ConsentRequired).Should(BeTrue())

		Expect(kcClient.SurrogateAuthRequired).ShouldNot(BeNil())
		Expect(*kcClient.SurrogateAuthRequired).Should(BeTrue())

		Expect(kcClient.ClientAuthenticatorType).ShouldNot(BeNil())
		Expect(*kcClient.ClientAuthenticatorType).Should(Equal("client-secret"))

		// HomeUrl maps to BaseUrl
		Expect(kcClient.BaseUrl).ShouldNot(BeNil())
		Expect(*kcClient.BaseUrl).Should(Equal("https://test-verify-fields-home.example.com"))

		// WebUrl maps to RootUrl
		Expect(kcClient.RootUrl).ShouldNot(BeNil())
		Expect(*kcClient.RootUrl).Should(Equal("https://test-verify-fields.example.com"))

		Expect(kcClient.AdminUrl).ShouldNot(BeNil())
		Expect(*kcClient.AdminUrl).Should(Equal("https://test-verify-fields-admin.example.com"))

		Expect(kcClient.Attributes).ShouldNot(BeNil())
		Expect((*kcClient.Attributes)["post.logout.redirect.uris"]).Should(Equal("+"))

		Expect(kcClient.RedirectUris).ShouldNot(BeNil())
		Expect(*kcClient.RedirectUris).Should(ContainElement("https://test-verify-fields.example.com/*"))

		Expect(kcClient.WebOrigins).ShouldNot(BeNil())
		Expect(*kcClient.WebOrigins).Should(ContainElement("https://test-verify-fields.example.com"))

		Expect(kcClient.AuthenticationFlowBindingOverrides).ShouldNot(BeNil())
		Expect(*kcClient.AuthenticationFlowBindingOverrides).Should(HaveLen(2))
	})

	It("Should create KeycloakClient with service account full config", func() {
		By("Creating group for service account")
		_, err := keycloakAdmin.Groups.CreateGroup(ctx, KeycloakRealmCR, keycloakapi.GroupRepresentation{
			Name: ptr.To("test-sa-group"),
		})
		Expect(keycloakapi.SkipConflict(err)).ShouldNot(HaveOccurred())

		By("Creating a KeycloakClient with full service account config")
		keycloakClient := &keycloakApi.KeycloakClient{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-keycloak-client-sa-full",
				Namespace: ns,
			},
			Spec: keycloakApi.KeycloakClientSpec{
				ClientId: "test-keycloak-client-sa-full",
				RealmRef: common.RealmRef{
					Name: KeycloakRealmCR,
					Kind: keycloakApi.KeycloakRealmKind,
				},
				Secret: secretref.GenerateSecretRef(clientSecret.Name, "secretKey"),
				ServiceAccount: &keycloakApi.ServiceAccount{
					Enabled:    true,
					RealmRoles: []string{"default-roles-" + KeycloakRealmCR},
					ClientRoles: []keycloakApi.UserClientRole{
						{
							ClientID: "account",
							Roles:    []string{"view-profile"},
						},
					},
					Groups: []string{"test-sa-group"},
					AttributesV2: map[string][]string{
						"test-attr": {"val1", "val2"},
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

			saCond := meta.FindStatusCondition(createdKeycloakClient.Status.Conditions, chain.ConditionServiceAccountSynced)
			g.Expect(saCond).ShouldNot(BeNil(), "ServiceAccountSynced condition should be set")
			g.Expect(saCond.Status).Should(Equal(metav1.ConditionTrue), "ServiceAccountSynced condition should be True")
		}, timeout, interval).Should(Succeed())

		By("Verifying service account in Keycloak")
		existingClient, _, err := keycloakAdmin.Clients.GetClientByClientID(ctx, KeycloakRealmCR, keycloakClient.Spec.ClientId)
		Expect(err).ShouldNot(HaveOccurred())
		Expect(existingClient).ShouldNot(BeNil())
		Expect(existingClient.Id).ShouldNot(BeNil())

		clientUUID := *existingClient.Id

		saUser, _, err := keycloakAdmin.Clients.GetServiceAccountUser(ctx, KeycloakRealmCR, clientUUID)
		Expect(err).ShouldNot(HaveOccurred())
		Expect(saUser).ShouldNot(BeNil())
		Expect(saUser.Id).ShouldNot(BeNil())

		saUserID := *saUser.Id

		By("Checking service account groups")
		groups, _, err := keycloakAdmin.Users.GetUserGroups(ctx, KeycloakRealmCR, saUserID)
		Expect(err).ShouldNot(HaveOccurred())

		groupNames := make([]string, 0, len(groups))
		for _, g := range groups {
			if g.Name != nil {
				groupNames = append(groupNames, *g.Name)
			}
		}

		Expect(groupNames).Should(ContainElement("test-sa-group"))

		By("Checking service account realm roles")
		realmRoles, _, err := keycloakAdmin.Users.GetUserRealmRoleMappings(ctx, KeycloakRealmCR, saUserID)
		Expect(err).ShouldNot(HaveOccurred())

		realmRoleNames := make([]string, 0, len(realmRoles))
		for _, r := range realmRoles {
			if r.Name != nil {
				realmRoleNames = append(realmRoleNames, *r.Name)
			}
		}

		Expect(realmRoleNames).Should(ContainElement("default-roles-" + KeycloakRealmCR))

		By("Checking service account client roles")
		accountClient, _, err := keycloakAdmin.Clients.GetClientByClientID(ctx, KeycloakRealmCR, "account")
		Expect(err).ShouldNot(HaveOccurred())
		Expect(accountClient).ShouldNot(BeNil())
		Expect(accountClient.Id).ShouldNot(BeNil())

		clientRoles, _, err := keycloakAdmin.Users.GetUserClientRoleMappings(ctx, KeycloakRealmCR, saUserID, *accountClient.Id)
		Expect(err).ShouldNot(HaveOccurred())

		clientRoleNames := make([]string, 0, len(clientRoles))
		for _, r := range clientRoles {
			if r.Name != nil {
				clientRoleNames = append(clientRoleNames, *r.Name)
			}
		}

		Expect(clientRoleNames).Should(ContainElement("view-profile"))

		By("Checking service account attributes")
		Expect(saUser.Attributes).ShouldNot(BeNil())
		Expect(*saUser.Attributes).Should(HaveKeyWithValue("test-attr", ConsistOf("val1", "val2")))
	})

	It("Should create KeycloakClient with protocol mappers", func() {
		By("Creating a KeycloakClient with protocol mappers")
		keycloakClient := &keycloakApi.KeycloakClient{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-keycloak-client-protocol-mappers",
				Namespace: ns,
			},
			Spec: keycloakApi.KeycloakClientSpec{
				ClientId: "test-keycloak-client-protocol-mappers",
				RealmRef: common.RealmRef{
					Name: KeycloakRealmCR,
					Kind: keycloakApi.KeycloakRealmKind,
				},
				Public: true,
				Secret: secretref.GenerateSecretRef(clientSecret.Name, "secretKey"),
				ProtocolMappers: &[]keycloakApi.ProtocolMapper{
					{
						Name:           "test-hardcoded-claim",
						Protocol:       "openid-connect",
						ProtocolMapper: "oidc-hardcoded-claim-mapper",
						Config: map[string]string{
							"claim.name":         "test-claim",
							"claim.value":        "test-value",
							"access.token.claim": "true",
							"id.token.claim":     "true",
						},
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

			pmCond := meta.FindStatusCondition(createdKeycloakClient.Status.Conditions, chain.ConditionProtocolMappersSynced)
			g.Expect(pmCond).ShouldNot(BeNil(), "ProtocolMappersSynced condition should be set")
			g.Expect(pmCond.Status).Should(Equal(metav1.ConditionTrue), "ProtocolMappersSynced condition should be True")
		}, timeout, interval).Should(Succeed())

		By("Verifying protocol mappers in Keycloak")
		existingClient, _, err := keycloakAdmin.Clients.GetClientByClientID(ctx, KeycloakRealmCR, keycloakClient.Spec.ClientId)
		Expect(err).ShouldNot(HaveOccurred())
		Expect(existingClient).ShouldNot(BeNil())
		Expect(existingClient.Id).ShouldNot(BeNil())

		mappers, _, err := keycloakAdmin.Clients.GetClientProtocolMappers(ctx, KeycloakRealmCR, *existingClient.Id)
		Expect(err).ShouldNot(HaveOccurred())

		var foundMapper *keycloakapi.ProtocolMapperRepresentation
		for i := range mappers {
			if mappers[i].Name != nil && *mappers[i].Name == "test-hardcoded-claim" {
				foundMapper = &mappers[i]

				break
			}
		}

		Expect(foundMapper).ShouldNot(BeNil(), "Protocol mapper 'test-hardcoded-claim' should exist")
		Expect(foundMapper.ProtocolMapper).ShouldNot(BeNil())
		Expect(*foundMapper.ProtocolMapper).Should(Equal("oidc-hardcoded-claim-mapper"))
		Expect(foundMapper.Protocol).ShouldNot(BeNil())
		Expect(*foundMapper.Protocol).Should(Equal("openid-connect"))
		Expect(foundMapper.Config).ShouldNot(BeNil())
		Expect((*foundMapper.Config)["claim.name"]).Should(Equal("test-claim"))
		Expect((*foundMapper.Config)["claim.value"]).Should(Equal("test-value"))
	})

	It("Should create KeycloakClient with client scopes", func() {
		By("Creating a KeycloakClient with default and optional client scopes")
		keycloakClient := &keycloakApi.KeycloakClient{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-keycloak-client-scopes",
				Namespace: ns,
			},
			Spec: keycloakApi.KeycloakClientSpec{
				ClientId: "test-keycloak-client-scopes",
				RealmRef: common.RealmRef{
					Name: KeycloakRealmCR,
					Kind: keycloakApi.KeycloakRealmKind,
				},
				Public:               true,
				Secret:               secretref.GenerateSecretRef(clientSecret.Name, "secretKey"),
				DefaultClientScopes:  []string{"email"},
				OptionalClientScopes: []string{"phone"},
			},
		}
		Expect(k8sClient.Create(ctx, keycloakClient)).Should(Succeed())
		Eventually(func(g Gomega) {
			createdKeycloakClient := &keycloakApi.KeycloakClient{}
			err := k8sClient.Get(ctx, types.NamespacedName{Name: keycloakClient.Name, Namespace: ns}, createdKeycloakClient)
			g.Expect(err).ShouldNot(HaveOccurred())
			g.Expect(createdKeycloakClient.Status.Value).Should(Equal(common.StatusOK))

			scopesCond := meta.FindStatusCondition(createdKeycloakClient.Status.Conditions, chain.ConditionClientScopesSynced)
			g.Expect(scopesCond).ShouldNot(BeNil(), "ClientScopesSynced condition should be set")
			g.Expect(scopesCond.Status).Should(Equal(metav1.ConditionTrue), "ClientScopesSynced condition should be True")
		}, timeout, interval).Should(Succeed())

		By("Verifying client scopes in Keycloak")
		existingClient, _, err := keycloakAdmin.Clients.GetClientByClientID(ctx, KeycloakRealmCR, keycloakClient.Spec.ClientId)
		Expect(err).ShouldNot(HaveOccurred())
		Expect(existingClient).ShouldNot(BeNil())
		Expect(existingClient.Id).ShouldNot(BeNil())

		clientUUID := *existingClient.Id

		By("Checking default client scopes")
		defaultScopes, _, err := keycloakAdmin.Clients.GetDefaultClientScopes(ctx, KeycloakRealmCR, clientUUID)
		Expect(err).ShouldNot(HaveOccurred())

		defaultScopeNames := make([]string, 0, len(defaultScopes))
		for _, s := range defaultScopes {
			if s.Name != nil {
				defaultScopeNames = append(defaultScopeNames, *s.Name)
			}
		}

		Expect(defaultScopeNames).Should(ContainElement("email"))

		By("Checking optional client scopes")
		optionalScopes, _, err := keycloakAdmin.Clients.GetOptionalClientScopes(ctx, KeycloakRealmCR, clientUUID)
		Expect(err).ShouldNot(HaveOccurred())

		optionalScopeNames := make([]string, 0, len(optionalScopes))
		for _, s := range optionalScopes {
			if s.Name != nil {
				optionalScopeNames = append(optionalScopeNames, *s.Name)
			}
		}

		Expect(optionalScopeNames).Should(ContainElement("phone"))
	})

	It("Should create KeycloakClient with realm roles", func() {
		By("Creating base realm role for composite")
		_, err := keycloakAdmin.Roles.CreateRealmRole(ctx, KeycloakRealmCR, keycloakapi.RoleRepresentation{
			Name: ptr.To("test-composite-base"),
		})
		Expect(keycloakapi.SkipConflict(err)).ShouldNot(HaveOccurred())

		By("Creating a KeycloakClient with realm roles")
		keycloakClient := &keycloakApi.KeycloakClient{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-keycloak-client-realm-roles",
				Namespace: ns,
			},
			Spec: keycloakApi.KeycloakClientSpec{
				ClientId: "test-keycloak-client-realm-roles",
				RealmRef: common.RealmRef{
					Name: KeycloakRealmCR,
					Kind: keycloakApi.KeycloakRealmKind,
				},
				Public: true,
				Secret: secretref.GenerateSecretRef(clientSecret.Name, "secretKey"),
				RealmRoles: &[]keycloakApi.RealmRole{
					{Name: "test-simple-role", Composite: ""},
					{Name: "test-composite-role", Composite: "test-composite-base"},
				},
			},
		}
		Expect(k8sClient.Create(ctx, keycloakClient)).Should(Succeed())
		Eventually(func(g Gomega) {
			createdKeycloakClient := &keycloakApi.KeycloakClient{}
			err := k8sClient.Get(ctx, types.NamespacedName{Name: keycloakClient.Name, Namespace: ns}, createdKeycloakClient)
			g.Expect(err).ShouldNot(HaveOccurred())
			g.Expect(createdKeycloakClient.Status.Value).Should(Equal(common.StatusOK))

			realmRolesCond := meta.FindStatusCondition(createdKeycloakClient.Status.Conditions, chain.ConditionRealmRolesSynced)
			g.Expect(realmRolesCond).ShouldNot(BeNil(), "RealmRolesSynced condition should be set")
			g.Expect(realmRolesCond.Status).Should(Equal(metav1.ConditionTrue), "RealmRolesSynced condition should be True")
		}, timeout, interval).Should(Succeed())

		By("Verifying simple realm role in Keycloak")
		simpleRole, _, err := keycloakAdmin.Roles.GetRealmRole(ctx, KeycloakRealmCR, "test-simple-role")
		Expect(err).ShouldNot(HaveOccurred())
		Expect(simpleRole).ShouldNot(BeNil())
		Expect(simpleRole.Name).ShouldNot(BeNil())
		Expect(*simpleRole.Name).Should(Equal("test-simple-role"))

		By("Verifying composite realm role in Keycloak")
		compositeRole, _, err := keycloakAdmin.Roles.GetRealmRole(ctx, KeycloakRealmCR, "test-composite-role")
		Expect(err).ShouldNot(HaveOccurred())
		Expect(compositeRole).ShouldNot(BeNil())
		Expect(compositeRole.Composite).ShouldNot(BeNil())
		Expect(*compositeRole.Composite).Should(BeTrue())

		composites, _, err := keycloakAdmin.Roles.GetRealmRoleComposites(ctx, KeycloakRealmCR, "test-composite-role")
		Expect(err).ShouldNot(HaveOccurred())
		Expect(composites).Should(HaveLen(1))
		Expect(*composites[0].Name).Should(Equal("test-composite-base"))
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

		By("Creating group for policy")
		_, err := keycloakAdmin.Groups.CreateGroup(ctx, KeycloakRealmCR, keycloakapi.GroupRepresentation{
			Name: ptr.To("test-policy-group"),
		})
		Expect(keycloakapi.SkipConflict(err)).ShouldNot(HaveOccurred())

		By("Creating role for policy")
		_, err = keycloakAdmin.Roles.CreateRealmRole(ctx, KeycloakRealmCR, keycloakapi.RoleRepresentation{
			Name: ptr.To("test-policy-role"),
		})
		Expect(keycloakapi.SkipConflict(err)).ShouldNot(HaveOccurred())

		By("Creating user for policy")
		_, err = keycloakAdmin.Users.CreateUser(ctx, KeycloakRealmCR, keycloakapi.UserRepresentation{
			Username: ptr.To("test-policy-user"),
			Enabled:  ptr.To(true),
		})
		Expect(keycloakapi.SkipConflict(err)).ShouldNot(HaveOccurred())

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
						{
							Name:             "aggregate-policy",
							Type:             keycloakApi.PolicyTypeAggregate,
							DecisionStrategy: keycloakApi.PolicyDecisionStrategyUnanimous,
							Logic:            keycloakApi.PolicyLogicPositive,
							AggregatedPolicy: &keycloakApi.AggregatedPolicyData{
								Policies: []string{"client-policy"},
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
		Eventually(func(g Gomega) {
			createdKeycloakClient := &keycloakApi.KeycloakClient{}
			g.Expect(k8sClient.Get(ctx, types.NamespacedName{Name: keycloakClient.Name, Namespace: ns}, createdKeycloakClient)).Should(Succeed())
			g.Expect(createdKeycloakClient.Status.Value).Should(Equal(common.StatusOK))
			g.Expect(createdKeycloakClient.Status.ClientID).ShouldNot(BeEmpty())
		}, timeout, interval).Should(Succeed(), "KeycloakClient should be created successfully")

		By("Verifying authorization scopes in Keycloak")
		authClient, _, err := keycloakAdmin.Clients.GetClientByClientID(ctx, KeycloakRealmCR, keycloakClient.Spec.ClientId)
		Expect(err).ShouldNot(HaveOccurred())
		Expect(authClient).ShouldNot(BeNil())
		Expect(authClient.Id).ShouldNot(BeNil())

		authClientUUID := *authClient.Id

		scopes, _, err := keycloakAdmin.Authorization.GetScopes(ctx, KeycloakRealmCR, authClientUUID)
		Expect(err).ShouldNot(HaveOccurred())

		scopeNames := make([]string, 0, len(scopes))
		for _, s := range scopes {
			if s.Name != nil {
				scopeNames = append(scopeNames, *s.Name)
			}
		}

		Expect(scopeNames).Should(ContainElement("test-scope"))

		By("Verifying authorization resources in Keycloak")
		resources, _, err := keycloakAdmin.Authorization.GetResources(ctx, KeycloakRealmCR, authClientUUID)
		Expect(err).ShouldNot(HaveOccurred())

		var testResource *keycloakapi.ResourceRepresentation
		for i := range resources {
			if resources[i].Name != nil && *resources[i].Name == "test-resource" {
				testResource = &resources[i]

				break
			}
		}

		Expect(testResource).ShouldNot(BeNil(), "Resource 'test-resource' should exist")
		Expect(testResource.DisplayName).ShouldNot(BeNil())
		Expect(*testResource.DisplayName).Should(Equal("Test resource"))
		Expect(testResource.Type).ShouldNot(BeNil())
		Expect(*testResource.Type).Should(Equal("test"))
		Expect(testResource.IconUri).ShouldNot(BeNil())
		Expect(*testResource.IconUri).Should(Equal("https://example.com/icon.png"))
		Expect(testResource.OwnerManagedAccess).ShouldNot(BeNil())
		Expect(*testResource.OwnerManagedAccess).Should(BeTrue())
		Expect(testResource.Uris).ShouldNot(BeNil())
		Expect(*testResource.Uris).Should(ContainElement("https://example.com"))

		By("Verifying authorization policies in Keycloak")
		policies, _, err := keycloakAdmin.Authorization.GetPolicies(ctx, KeycloakRealmCR, authClientUUID)
		Expect(err).ShouldNot(HaveOccurred())

		policyNames := make([]string, 0, len(policies))
		for _, p := range policies {
			if p.Name != nil {
				policyNames = append(policyNames, *p.Name)
			}
		}

		Expect(policyNames).Should(ContainElements("client-policy", "group-policy", "role-policy", "time-policy", "user-policy", "aggregate-policy"))

		By("Adding client permissions")

		By("Getting Client UUID")
		existingClient, _, err := keycloakAdmin.Clients.GetClientByClientID(ctx, KeycloakRealmCR, keycloakClient.Spec.ClientId)
		Expect(err).ShouldNot(HaveOccurred())
		Expect(existingClient).ShouldNot(BeNil())
		Expect(existingClient.Id).ShouldNot(BeNil())

		clientUUID := *existingClient.Id

		By("Creating scope for permission")
		_, err = keycloakAdmin.Authorization.CreateScope(ctx, KeycloakRealmCR, clientUUID,
			keycloakapi.ScopeRepresentation{
				Name: ptr.To("test-scope"),
			})
		Expect(keycloakapi.SkipConflict(err)).ShouldNot(HaveOccurred())

		By("Creating resource for permission")
		_, _, err = keycloakAdmin.Authorization.CreateResource(ctx, KeycloakRealmCR, clientUUID,
			keycloakapi.ResourceRepresentation{
				Name:               ptr.To("test-resource"),
				OwnerManagedAccess: ptr.To(false),
			})
		Expect(keycloakapi.SkipConflict(err)).ShouldNot(HaveOccurred())

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

		By("Updating KeycloakClient with permissions")
		Expect(k8sClient.Update(ctx, clientToUpdate)).Should(Succeed())

		Eventually(func(g Gomega) {
			updatedKeycloakClient := &keycloakApi.KeycloakClient{}
			g.Expect(k8sClient.Get(ctx, types.NamespacedName{Name: keycloakClient.Name, Namespace: ns}, updatedKeycloakClient)).Should(Succeed())
			g.Expect(updatedKeycloakClient.Status.Value).Should(Equal(common.StatusOK))
			g.Expect(updatedKeycloakClient.Status.ClientID).ShouldNot(BeEmpty())
		}, timeout, interval).Should(Succeed(), "KeycloakClient should be updated successfully")
	})

	It("Should create KeycloakClient with advancedSettings", func() {
		By("Creating a KeycloakClient with advancedSettings")
		keycloakClient := &keycloakApi.KeycloakClient{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-keycloak-client-access-token-lifespan",
				Namespace: ns,
			},
			Spec: keycloakApi.KeycloakClientSpec{
				ClientId: "test-keycloak-client-access-token-lifespan",
				RealmRef: common.RealmRef{
					Name: KeycloakRealmCR,
					Kind: keycloakApi.KeycloakRealmKind,
				},
				AdvancedSettings: &keycloakApi.KeycloakClientAdvancedSettings{
					AccessTokenLifespan: ptr.To(300),
				},
			},
		}
		Expect(k8sClient.Create(ctx, keycloakClient)).Should(Succeed())

		Eventually(func(g Gomega) {
			createdKeycloakClient := &keycloakApi.KeycloakClient{}
			g.Expect(k8sClient.Get(ctx, types.NamespacedName{Name: keycloakClient.Name, Namespace: ns}, createdKeycloakClient)).Should(Succeed())
			g.Expect(createdKeycloakClient.Status.Value).Should(Equal(common.StatusOK))
		}, timeout, interval).Should(Succeed())

		By("Verifying access.token.lifespan attribute in Keycloak")
		kcClient, _, err := keycloakAdmin.Clients.GetClientByClientID(ctx, KeycloakRealmCR, keycloakClient.Spec.ClientId)
		Expect(err).ShouldNot(HaveOccurred())
		Expect(kcClient.Attributes).ShouldNot(BeNil())
		Expect((*kcClient.Attributes)["access.token.lifespan"]).Should(Equal("300"))
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
		Eventually(func(g Gomega) {
			c := &keycloakApi.KeycloakClient{}
			err := k8sClient.Get(ctx, types.NamespacedName{Name: keycloakClient.Name, Namespace: ns}, c)
			g.Expect(err).ShouldNot(HaveOccurred())
			g.Expect(c.Status.Value).Should(Equal(common.StatusOK))
		}, timeout, interval).Should(Succeed())

		By("Deleting KeycloakClient")
		Expect(k8sClient.Delete(ctx, keycloakClient)).Should(Succeed())

		By("Waiting for KeycloakClient to be deleted")
		Eventually(func() bool {
			var c keycloakApi.KeycloakClient
			err := k8sClient.Get(ctx, types.NamespacedName{Name: keycloakClient.Name, Namespace: ns}, &c)
			return k8sErrors.IsNotFound(err)
		}, time.Minute, time.Second*5).Should(BeTrue())

		By("Deleting KeycloakRealm")
		Expect(k8sClient.Delete(ctx, testRealm)).Should(Succeed())
		Eventually(func() bool {
			var r keycloakApi.KeycloakRealm
			err := k8sClient.Get(ctx, types.NamespacedName{Name: testRealm.Name, Namespace: ns}, &r)
			return k8sErrors.IsNotFound(err)
		}, time.Minute, time.Second*5).Should(BeTrue())
	})
})

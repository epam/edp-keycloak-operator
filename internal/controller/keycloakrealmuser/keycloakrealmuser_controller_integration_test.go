package keycloakrealmuser

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
	"github.com/epam/edp-keycloak-operator/internal/controller/keycloakrealmuser/chain"
	keycloakv2 "github.com/epam/edp-keycloak-operator/pkg/client/keycloakv2"
	"github.com/epam/edp-keycloak-operator/pkg/objectmeta"
)

var _ = Describe("KeycloakRealmUser controller", Ordered, func() {
	const (
		userCR         = "test-keycloak-realm-user"
		userSecretName = "test-user-secret"
	)
	It("Should create KeycloakRealmUser", func() {
		By("Creating group for user")
		_, err := keycloakApiClient.Groups.CreateGroup(ctx, KeycloakRealmCR, keycloakv2.GroupRepresentation{
			Name: ptr.To("test-user-group"),
		})
		if err != nil && !keycloakv2.IsConflict(err) {
			Expect(err).ShouldNot(HaveOccurred())
		}

		By("Creating group and subgroup for user")
		_, err = keycloakApiClient.Groups.CreateGroup(ctx, KeycloakRealmCR, keycloakv2.GroupRepresentation{
			Name: ptr.To("test-user-group2"),
		})
		if err != nil && !keycloakv2.IsConflict(err) {
			Expect(err).ShouldNot(HaveOccurred())
		}

		gr, _, err := keycloakApiClient.Groups.GetGroups(ctx, KeycloakRealmCR, &keycloakv2.GetGroupsParams{
			Search: ptr.To("test-user-group2"),
		})
		Expect(err).ShouldNot(HaveOccurred())
		Expect(gr).Should(HaveLen(1))

		_, err = keycloakApiClient.Groups.CreateChildGroup(ctx, KeycloakRealmCR, *gr[0].Id, keycloakv2.GroupRepresentation{
			Name: ptr.To("test-user-group2-subgroup"),
		})
		if err != nil && !keycloakv2.IsConflict(err) {
			Expect(err).ShouldNot(HaveOccurred())
		}

		By("Creating a top-level group with the same name as the subgroup")
		_, err = keycloakApiClient.Groups.CreateGroup(ctx, KeycloakRealmCR, keycloakv2.GroupRepresentation{
			Name: ptr.To("test-user-group2-subgroup"),
		})
		if err != nil && !keycloakv2.IsConflict(err) {
			Expect(err).ShouldNot(HaveOccurred())
		}

		By("Creating role for user")
		_, err = keycloakApiClient.Roles.CreateRealmRole(ctx, KeycloakRealmCR, keycloakv2.RoleRepresentation{
			Name: ptr.To("test-user-role"),
		})
		if err != nil && !keycloakv2.IsConflict(err) {
			Expect(err).ShouldNot(HaveOccurred())
		}

		By("Creating Identity Provider")
		_, err = keycloakApiClient.IdentityProviders.CreateIdentityProvider(
			ctx,
			KeycloakRealmCR,
			keycloakv2.IdentityProviderRepresentation{
				Alias:                     ptr.To("test-idp"),
				DisplayName:               ptr.To("Test Identity Provider"),
				ProviderId:                ptr.To("oidc"),
				Enabled:                   ptr.To(true),
				TrustEmail:                ptr.To(true),
				StoreToken:                ptr.To(true),
				AddReadTokenRoleOnCreate:  ptr.To(false),
				LinkOnly:                  ptr.To(false),
				FirstBrokerLoginFlowAlias: ptr.To("first broker login"),
				Config: &map[string]string{
					"clientId":         "test-client-id",
					"clientSecret":     "test-client-secret",
					"issuer":           "https://example.com",
					"authorizationUrl": "https://example.com/auth",
					"tokenUrl":         "https://example.com/token",
					"userInfoUrl":      "https://example.com/userinfo",
				},
			},
		)
		if err != nil && !keycloakv2.IsConflict(err) {
			Expect(err).ShouldNot(HaveOccurred())
		}

		By("Creating a KeycloakRealmUser with realm roles and client roles")
		user := &keycloakApi.KeycloakRealmUser{
			ObjectMeta: metav1.ObjectMeta{
				Name:      userCR,
				Namespace: ns,
			},
			Spec: keycloakApi.KeycloakRealmUserSpec{
				RealmRef: common.RealmRef{
					Kind: keycloakApi.KeycloakRealmKind,
					Name: KeycloakRealmCR,
				},
				Username:      "test-user",
				Email:         "test-user@mail.com",
				FirstName:     "test-first-name",
				LastName:      "test-last-name",
				Enabled:       true,
				EmailVerified: true,
				RequiredUserActions: []string{
					"VERIFY_EMAIL",
				},
				Roles: []string{
					"offline_access",
					"uma_authorization",
				},
				ClientRoles: []keycloakApi.UserClientRole{
					{
						ClientID: "account",
						Roles: []string{
							"view-profile",
							"view-groups",
						},
					},
					{
						ClientID: "realm-management",
						Roles: []string{
							"create-client",
						},
					},
				},
				Groups: []string{
					"test-user-group",
					"/test-user-group2/test-user-group2-subgroup",
				},
				AttributesV2: map[string][]string{
					"attr1": {"test-value"},
				},
				KeepResource:      true,
				IdentityProviders: &[]string{"test-idp"},
			},
		}
		Expect(k8sClient.Create(ctx, user)).Should(Succeed())
		Eventually(func(g Gomega) {
			createdUser := &keycloakApi.KeycloakRealmUser{}
			err = k8sClient.Get(ctx, types.NamespacedName{Name: userCR, Namespace: ns}, createdUser)
			g.Expect(err).ShouldNot(HaveOccurred())
			g.Expect(createdUser.Status.Value).Should(Equal(common.StatusOK))
		}).WithTimeout(time.Second * 20).WithPolling(time.Second).Should(Succeed())

		// Verify that the roles were created in Keycloak
		Eventually(func(g Gomega) {
			foundUser, _, err := keycloakApiClient.Users.FindUserByUsername(ctx, KeycloakRealmCR, user.Spec.Username)
			g.Expect(err).ShouldNot(HaveOccurred())
			g.Expect(foundUser).ShouldNot(BeNil())
			g.Expect(foundUser.Id).ShouldNot(BeNil())

			// Get user realm role mappings
			realmRoles, _, err := keycloakApiClient.Users.GetUserRealmRoleMappings(ctx, KeycloakRealmCR, *foundUser.Id)
			g.Expect(err).ShouldNot(HaveOccurred())

			// Check realm roles
			realmRoleNames := make([]string, 0)
			for _, role := range realmRoles {
				if role.Name != nil {
					realmRoleNames = append(realmRoleNames, *role.Name)
				}
			}
			g.Expect(realmRoleNames).Should(ContainElement("offline_access"))
			g.Expect(realmRoleNames).Should(ContainElement("uma_authorization"))

			// Check client roles for account client
			accountClients, _, err := keycloakApiClient.Clients.GetClients(ctx, KeycloakRealmCR, &keycloakv2.GetClientsParams{ClientId: ptr.To("account")})
			g.Expect(err).ShouldNot(HaveOccurred())
			g.Expect(accountClients).ShouldNot(BeEmpty())

			accountClientRoles, _, err := keycloakApiClient.Users.GetUserClientRoleMappings(ctx, KeycloakRealmCR, *foundUser.Id, *accountClients[0].Id)
			g.Expect(err).ShouldNot(HaveOccurred())

			accountClientRoleNames := make([]string, 0)
			for _, role := range accountClientRoles {
				if role.Name != nil {
					accountClientRoleNames = append(accountClientRoleNames, *role.Name)
				}
			}
			g.Expect(accountClientRoleNames).Should(ContainElement("view-profile"))
			g.Expect(accountClientRoleNames).Should(ContainElement("view-groups"))

			// Check client roles for realm-management client
			realmMgmtClients, _, err := keycloakApiClient.Clients.GetClients(ctx, KeycloakRealmCR, &keycloakv2.GetClientsParams{ClientId: ptr.To("realm-management")})
			g.Expect(err).ShouldNot(HaveOccurred())
			g.Expect(realmMgmtClients).ShouldNot(BeEmpty())

			realmMgmtClientRoles, _, err := keycloakApiClient.Users.GetUserClientRoleMappings(ctx, KeycloakRealmCR, *foundUser.Id, *realmMgmtClients[0].Id)
			g.Expect(err).ShouldNot(HaveOccurred())

			realmManagementClientRoleNames := make([]string, 0)
			for _, role := range realmMgmtClientRoles {
				if role.Name != nil {
					realmManagementClientRoleNames = append(realmManagementClientRoleNames, *role.Name)
				}
			}
			g.Expect(realmManagementClientRoleNames).Should(ContainElement("create-client"))

			// Verify group membership - both plain name and slash-prefixed path groups
			userGroups, _, err := keycloakApiClient.Users.GetUserGroups(ctx, KeycloakRealmCR, *foundUser.Id)
			g.Expect(err).ShouldNot(HaveOccurred())

			groupNames := make([]string, 0)
			for _, grp := range userGroups {
				if grp.Path != nil {
					groupNames = append(groupNames, *grp.Path)
				}
			}

			g.Expect(groupNames).Should(ContainElement("/test-user-group"))
			// Verify the slash-prefixed path resolved to the child group, not the top-level
			// group with the same name.
			g.Expect(groupNames).Should(ContainElement("/test-user-group2/test-user-group2-subgroup"))
			g.Expect(groupNames).ShouldNot(ContainElement("/test-user-group2-subgroup"))
		}, time.Minute, time.Second*5).Should(Succeed())
	})
	It("Should create KeycloakRealmUser with password", func() {
		By("Creating a password secret")
		passwordSecret := &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      userSecretName,
				Namespace: ns,
			},
			Data: map[string][]byte{
				"password": []byte("test-password-123"),
			},
		}
		Expect(k8sClient.Create(ctx, passwordSecret)).Should(Succeed())

		By("Creating a KeycloakRealmUser with password secret")
		user := &keycloakApi.KeycloakRealmUser{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-keycloak-realm-user-with-password",
				Namespace: ns,
			},
			Spec: keycloakApi.KeycloakRealmUserSpec{
				RealmRef: common.RealmRef{
					Kind: keycloakApi.KeycloakRealmKind,
					Name: KeycloakRealmCR,
				},
				Username:  "test-user-with-password",
				Email:     "test-user-with-password@mail.com",
				FirstName: "test-first-name",
				LastName:  "test-last-name",
				Enabled:   true,
				PasswordSecret: &keycloakApi.PasswordSecret{
					Name:      passwordSecret.Name,
					Key:       "password",
					Temporary: false,
				},
				KeepResource: true,
			},
		}
		Expect(k8sClient.Create(ctx, user)).Should(Succeed())
		Eventually(func(g Gomega) {
			createdUser := &keycloakApi.KeycloakRealmUser{}
			err := k8sClient.Get(ctx, types.NamespacedName{Name: user.Name, Namespace: ns}, createdUser)
			g.Expect(err).ShouldNot(HaveOccurred())
			g.Expect(createdUser.Status.Value).Should(Equal(common.StatusOK))

			// Verify PasswordSynced condition is set correctly for non-temporary password
			passwordSyncedCondition := meta.FindStatusCondition(createdUser.Status.Conditions, chain.ConditionPasswordSynced)
			g.Expect(passwordSyncedCondition).ShouldNot(BeNil(), "PasswordSynced condition should exist")
			g.Expect(passwordSyncedCondition.Status).Should(Equal(metav1.ConditionTrue), "PasswordSynced condition should be True")
			g.Expect(passwordSyncedCondition.Reason).Should(Equal(chain.ReasonPasswordSetFromSecret), "Reason should be PasswordSetFromSecret")
			g.Expect(passwordSyncedCondition.Message).Should(ContainSubstring("Password synced from secret"))
			g.Expect(createdUser.Status.LastSyncedPasswordSecretVersion).ShouldNot(BeEmpty(), "LastSyncedPasswordSecretVersion should be set")
		}).WithTimeout(time.Second * 20).WithPolling(time.Second).Should(Succeed())

		By("Verifying user exists in Keycloak")
		Eventually(func(g Gomega) {
			foundUser, _, err := keycloakApiClient.Users.FindUserByUsername(ctx, KeycloakRealmCR, user.Spec.Username)
			g.Expect(err).ShouldNot(HaveOccurred())
			g.Expect(foundUser).ShouldNot(BeNil())
			g.Expect(*foundUser.Username).Should(Equal(user.Spec.Username))
			g.Expect(*foundUser.Email).Should(Equal(user.Spec.Email))
		}, time.Minute, time.Second*5).Should(Succeed())
	})
	It("Should create KeycloakRealmUser with temporary password", func() {
		By("Creating a KeycloakRealmUser with temporary password secret")
		user := &keycloakApi.KeycloakRealmUser{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-keycloak-realm-user-with-temp-password",
				Namespace: ns,
			},
			Spec: keycloakApi.KeycloakRealmUserSpec{
				RealmRef: common.RealmRef{
					Kind: keycloakApi.KeycloakRealmKind,
					Name: KeycloakRealmCR,
				},
				Username:  "test-user-with-temp-password",
				Email:     "test-user-with-temp-password@mail.com",
				FirstName: "test-first-name",
				LastName:  "test-last-name",
				Enabled:   true,
				PasswordSecret: &keycloakApi.PasswordSecret{
					Name:      userSecretName,
					Key:       "password",
					Temporary: true,
				},
				KeepResource: true,
			},
		}
		Expect(k8sClient.Create(ctx, user)).Should(Succeed())
		Eventually(func(g Gomega) {
			createdUser := &keycloakApi.KeycloakRealmUser{}
			err := k8sClient.Get(ctx, types.NamespacedName{Name: user.Name, Namespace: ns}, createdUser)
			g.Expect(err).ShouldNot(HaveOccurred())
			g.Expect(createdUser.Status.Value).Should(Equal(common.StatusOK))

			// Verify PasswordSynced condition is set correctly for temporary password
			passwordSyncedCondition := meta.FindStatusCondition(createdUser.Status.Conditions, chain.ConditionPasswordSynced)
			g.Expect(passwordSyncedCondition).ShouldNot(BeNil(), "PasswordSynced condition should exist")
			g.Expect(passwordSyncedCondition.Status).Should(Equal(metav1.ConditionTrue), "PasswordSynced condition should be True")
			g.Expect(passwordSyncedCondition.Reason).Should(Equal(chain.ReasonTemporaryPasswordSet), "Reason should be TemporaryPasswordSet")
			g.Expect(passwordSyncedCondition.Message).Should(ContainSubstring("Temporary password set from secret"))
			g.Expect(passwordSyncedCondition.Message).Should(ContainSubstring("will not reset"))
			g.Expect(createdUser.Status.LastSyncedPasswordSecretVersion).ShouldNot(BeEmpty(), "LastSyncedPasswordSecretVersion should be set")
		}).WithTimeout(time.Second * 20).WithPolling(time.Second).Should(Succeed())

		By("Verifying user exists in Keycloak with UPDATE_PASSWORD required action")
		Eventually(func(g Gomega) {
			foundUser, _, err := keycloakApiClient.Users.FindUserByUsername(ctx, KeycloakRealmCR, user.Spec.Username)
			g.Expect(err).ShouldNot(HaveOccurred())
			g.Expect(foundUser).ShouldNot(BeNil())
			g.Expect(*foundUser.Username).Should(Equal(user.Spec.Username))
			g.Expect(*foundUser.Email).Should(Equal(user.Spec.Email))
			// Temporary password should set UPDATE_PASSWORD required action
			g.Expect(foundUser.RequiredActions).ShouldNot(BeNil())
			g.Expect(*foundUser.RequiredActions).Should(ContainElement("UPDATE_PASSWORD"))
		}, time.Minute, time.Second*5).Should(Succeed())
	})
	It("Should update KeycloakRealmUser", func() {
		By("Getting KeycloakRealmUser")
		user := &keycloakApi.KeycloakRealmUser{}
		Expect(k8sClient.Get(ctx, types.NamespacedName{Namespace: ns, Name: userCR}, user)).Should(Succeed())

		By("Updating a parent KeycloakRealmUser")
		user.Spec.FirstName = "new-first-name"
		user.Spec.LastName = "new-last-name"

		Expect(k8sClient.Update(ctx, user)).Should(Succeed())
		Eventually(func(g Gomega) {
			updatedUser := &keycloakApi.KeycloakRealmUser{}
			err := k8sClient.Get(ctx, types.NamespacedName{Name: user.Name, Namespace: ns}, updatedUser)
			g.Expect(err).ShouldNot(HaveOccurred())
			g.Expect(updatedUser.Status.Value).Should(Equal(common.StatusOK))
		}, time.Minute, time.Second*5).Should(Succeed())

		Eventually(func(g Gomega) {
			foundUser, _, err := keycloakApiClient.Users.FindUserByUsername(ctx, KeycloakRealmCR, user.Spec.Username)
			g.Expect(err).ShouldNot(HaveOccurred())
			g.Expect(foundUser).ShouldNot(BeNil())
			g.Expect(*foundUser.FirstName).Should(Equal("new-first-name"))
			g.Expect(*foundUser.LastName).Should(Equal("new-last-name"))
		}, time.Minute, time.Second*5).Should(Succeed())
	})
	It("Should update KeycloakRealmUser roles", func() {
		By("Getting KeycloakRealmUser")
		user := &keycloakApi.KeycloakRealmUser{}
		Expect(k8sClient.Get(ctx, types.NamespacedName{Namespace: ns, Name: userCR}, user)).Should(Succeed())

		By("Updating KeycloakRealmUser roles")
		user.Spec.Roles = []string{
			"offline_access",
		}
		user.Spec.ClientRoles = []keycloakApi.UserClientRole{
			{
				ClientID: "account",
				Roles: []string{
					"view-profile",
				},
			},
		}

		Expect(k8sClient.Update(ctx, user)).Should(Succeed())
		Eventually(func(g Gomega) {
			updatedUser := &keycloakApi.KeycloakRealmUser{}
			err := k8sClient.Get(ctx, types.NamespacedName{Name: user.Name, Namespace: ns}, updatedUser)
			g.Expect(err).ShouldNot(HaveOccurred())
			g.Expect(updatedUser.Status.Value).Should(Equal(common.StatusOK))
		}, time.Minute, time.Second*5).Should(Succeed())

		// Verify that the roles were updated in Keycloak
		Eventually(func(g Gomega) {
			foundUser, _, err := keycloakApiClient.Users.FindUserByUsername(ctx, KeycloakRealmCR, user.Spec.Username)
			g.Expect(err).ShouldNot(HaveOccurred())
			g.Expect(foundUser).ShouldNot(BeNil())
			g.Expect(foundUser.Id).ShouldNot(BeNil())

			// Get user realm role mappings
			realmRoles, _, err := keycloakApiClient.Users.GetUserRealmRoleMappings(ctx, KeycloakRealmCR, *foundUser.Id)
			g.Expect(err).ShouldNot(HaveOccurred())

			// Check realm roles
			realmRoleNames := make([]string, 0)
			for _, role := range realmRoles {
				if role.Name != nil {
					realmRoleNames = append(realmRoleNames, *role.Name)
				}
			}
			g.Expect(realmRoleNames).Should(ContainElement("offline_access"))
			g.Expect(realmRoleNames).ShouldNot(ContainElement("uma_authorization"))

			// Check client roles for account client
			accountClients, _, err := keycloakApiClient.Clients.GetClients(ctx, KeycloakRealmCR, &keycloakv2.GetClientsParams{ClientId: ptr.To("account")})
			g.Expect(err).ShouldNot(HaveOccurred())
			g.Expect(accountClients).ShouldNot(BeEmpty())

			accountClientRoles, _, err := keycloakApiClient.Users.GetUserClientRoleMappings(ctx, KeycloakRealmCR, *foundUser.Id, *accountClients[0].Id)
			g.Expect(err).ShouldNot(HaveOccurred())

			clientRoleNames := make([]string, 0)
			for _, role := range accountClientRoles {
				if role.Name != nil {
					clientRoleNames = append(clientRoleNames, *role.Name)
				}
			}
			g.Expect(clientRoleNames).Should(ContainElement("view-profile"))
			g.Expect(clientRoleNames).ShouldNot(ContainElement("view-groups"))
		}, time.Minute, time.Second*5).Should(Succeed())
	})
	It("Should delete KeycloakRealmUser", func() {
		By("Getting KeycloakRealmUser")
		user := &keycloakApi.KeycloakRealmUser{}
		Expect(k8sClient.Get(ctx, types.NamespacedName{Namespace: ns, Name: userCR}, user)).Should(Succeed())
		By("Deleting KeycloakRealmUser")
		Expect(k8sClient.Delete(ctx, user)).Should(Succeed())
		Eventually(func(g Gomega) {
			deletedUser := &keycloakApi.KeycloakRealmUser{}
			err := k8sClient.Get(ctx, types.NamespacedName{Name: user.Name, Namespace: ns}, deletedUser)

			g.Expect(k8sErrors.IsNotFound(err)).Should(BeTrue())
		}, timeout, interval).Should(Succeed())
	})
	It("Should preserve user with annotation", func() {
		By("Creating a KeycloakRealmUser")
		user := &keycloakApi.KeycloakRealmUser{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-keycloak-realm-user-preserve",
				Namespace: ns,
				Annotations: map[string]string{
					objectmeta.PreserveResourcesOnDeletionAnnotation: "true",
				},
			},
			Spec: keycloakApi.KeycloakRealmUserSpec{
				RealmRef: common.RealmRef{
					Kind: keycloakApi.KeycloakRealmKind,
					Name: KeycloakRealmCR,
				},
				Username:     "test-user",
				KeepResource: true,
			},
		}
		Expect(k8sClient.Create(ctx, user)).Should(Succeed())
		Eventually(func(g Gomega) {
			createdUser := &keycloakApi.KeycloakRealmUser{}
			err := k8sClient.Get(ctx, types.NamespacedName{Name: user.Name, Namespace: ns}, createdUser)
			g.Expect(err).ShouldNot(HaveOccurred())
			g.Expect(createdUser.Status.Value).Should(Equal(common.StatusOK))
		}).WithTimeout(time.Second * 20).WithPolling(time.Second).Should(Succeed())

		By("Deleting KeycloakRealmUser")
		Expect(k8sClient.Delete(ctx, user)).Should(Succeed())
		Eventually(func(g Gomega) {
			foundUser, _, err := keycloakApiClient.Users.FindUserByUsername(ctx, KeycloakRealmCR, user.Spec.Username)
			g.Expect(err).ShouldNot(HaveOccurred())
			g.Expect(foundUser).ShouldNot(BeNil())
		}, time.Minute, time.Second*5).Should(Succeed())
	})
	It("Should fail to create KeycloakRealmUser with invalid password secret", func() {
		By("Creating a KeycloakRealmUser")
		user := &keycloakApi.KeycloakRealmUser{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-keycloak-realm-user-with-invalid-secret",
				Namespace: ns,
			},
			Spec: keycloakApi.KeycloakRealmUserSpec{
				RealmRef: common.RealmRef{
					Kind: keycloakApi.KeycloakRealmKind,
					Name: KeycloakRealmCR,
				},
				Username: "test-user-invalid-secret",
				PasswordSecret: &keycloakApi.PasswordSecret{
					Name: "invalid-secret",
					Key:  "password",
				},
			},
		}
		Expect(k8sClient.Create(ctx, user)).Should(Succeed())

		By("Waiting for KeycloakRealmUser to be processed")
		time.Sleep(time.Second * 3)

		By("Checking KeycloakRealmUser status")
		Consistently(func(g Gomega) {
			createdUser := &keycloakApi.KeycloakRealmUser{}
			err := k8sClient.Get(ctx, types.NamespacedName{Name: user.Name, Namespace: ns}, createdUser)
			g.Expect(err).ShouldNot(HaveOccurred())

			// Check Status.Value still contains error (backward compatibility)
			g.Expect(createdUser.Status.Value).Should(ContainSubstring("failed to get secret"))

			// Check PasswordSynced condition is False with correct reason
			passwordSyncedCondition := meta.FindStatusCondition(createdUser.Status.Conditions, chain.ConditionPasswordSynced)
			g.Expect(passwordSyncedCondition).ShouldNot(BeNil(), "PasswordSynced condition should be set on error")
			g.Expect(passwordSyncedCondition.Status).Should(Equal(metav1.ConditionFalse), "PasswordSynced should be False on error")
			g.Expect(passwordSyncedCondition.Reason).Should(Equal(chain.ReasonSecretNotFound), "Reason should be SecretNotFound")
			g.Expect(passwordSyncedCondition.Message).Should(ContainSubstring("Password secret"))
			g.Expect(passwordSyncedCondition.Message).Should(ContainSubstring("not found"))
			g.Expect(passwordSyncedCondition.ObservedGeneration).Should(Equal(createdUser.Generation))
		}, time.Second*3, time.Second).Should(Succeed())
	})
	It("Should fail to create KeycloakRealmUser with invalid password secret key", func() {
		By("Creating a KeycloakRealmUser")
		user := &keycloakApi.KeycloakRealmUser{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-keycloak-realm-user-with-invalid-secret-key",
				Namespace: ns,
			},
			Spec: keycloakApi.KeycloakRealmUserSpec{
				RealmRef: common.RealmRef{
					Kind: keycloakApi.KeycloakRealmKind,
					Name: KeycloakRealmCR,
				},
				Username: "test-user-invalid-secret-key",
				PasswordSecret: &keycloakApi.PasswordSecret{
					Name: userSecretName,
					Key:  "invalid-key",
				},
			},
		}
		Expect(k8sClient.Create(ctx, user)).Should(Succeed())

		By("Waiting for KeycloakRealmUser to be processed")
		time.Sleep(time.Second * 3)

		By("Checking KeycloakRealmUser status")
		Consistently(func(g Gomega) {
			createdUser := &keycloakApi.KeycloakRealmUser{}
			err := k8sClient.Get(ctx, types.NamespacedName{Name: user.Name, Namespace: ns}, createdUser)
			g.Expect(err).ShouldNot(HaveOccurred())

			// Check Status.Value still contains error (backward compatibility)
			g.Expect(createdUser.Status.Value).Should(ContainSubstring("key invalid-key not found in secret"))

			// Check PasswordSynced condition is False with correct reason
			passwordSyncedCondition := meta.FindStatusCondition(createdUser.Status.Conditions, chain.ConditionPasswordSynced)
			g.Expect(passwordSyncedCondition).ShouldNot(BeNil(), "PasswordSynced condition should be set on error")
			g.Expect(passwordSyncedCondition.Status).Should(Equal(metav1.ConditionFalse), "PasswordSynced should be False on error")
			g.Expect(passwordSyncedCondition.Reason).Should(Equal(chain.ReasonSecretKeyMissing), "Reason should be SecretKeyMissing")
			g.Expect(passwordSyncedCondition.Message).Should(ContainSubstring("Key"))
			g.Expect(passwordSyncedCondition.Message).Should(ContainSubstring("not found"))
			g.Expect(passwordSyncedCondition.ObservedGeneration).Should(Equal(createdUser.Generation))
		}, time.Second*3, time.Second).Should(Succeed())
	})
	It("Should fail to create KeycloakRealmUser with invalid role", func() {
		By("Creating a KeycloakRealmUser")
		user := &keycloakApi.KeycloakRealmUser{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-keycloak-realm-user-with-invalid-role",
				Namespace: ns,
			},
			Spec: keycloakApi.KeycloakRealmUserSpec{
				RealmRef: common.RealmRef{
					Kind: keycloakApi.KeycloakRealmKind,
					Name: KeycloakRealmCR,
				},
				Username: "test-user-invalid-role",
				Roles:    []string{"invalid-role"},
			},
		}
		Expect(k8sClient.Create(ctx, user)).Should(Succeed())

		By("Waiting for KeycloakRealmUser to be processed")
		time.Sleep(time.Second * 3)

		By("Checking KeycloakRealmUser status")
		Consistently(func(g Gomega) {
			createdUser := &keycloakApi.KeycloakRealmUser{}
			err := k8sClient.Get(ctx, types.NamespacedName{Name: user.Name, Namespace: ns}, createdUser)
			g.Expect(err).ShouldNot(HaveOccurred())
			g.Expect(createdUser.Status.Value).Should(ContainSubstring("unable to sync user realm roles"))
		}, time.Second*3, time.Second).Should(Succeed())
	})
	It("Should delete KeycloakRealmUser if user not found", func() {
		By("Creating a KeycloakRealmUser")
		user := &keycloakApi.KeycloakRealmUser{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-keycloak-realm-user-not-found",
				Namespace: ns,
			},
			Spec: keycloakApi.KeycloakRealmUserSpec{
				RealmRef: common.RealmRef{
					Kind: keycloakApi.KeycloakRealmKind,
					Name: KeycloakRealmCR,
				},
				Username:     "test-user-not-found",
				KeepResource: true,
			},
		}
		Expect(k8sClient.Create(ctx, user)).Should(Succeed())
		Eventually(func(g Gomega) {
			createdUser := &keycloakApi.KeycloakRealmUser{}
			err := k8sClient.Get(ctx, types.NamespacedName{Name: user.Name, Namespace: ns}, createdUser)
			g.Expect(err).ShouldNot(HaveOccurred())
			g.Expect(createdUser.Status.Value).Should(Equal(common.StatusOK))
		}).WithTimeout(time.Second * 20).WithPolling(time.Second).Should(Succeed())

		By("Manually deleting the user from Keycloak to simulate user not found scenario")
		foundUser, _, err := keycloakApiClient.Users.FindUserByUsername(ctx, KeycloakRealmCR, user.Spec.Username)
		Expect(err).ShouldNot(HaveOccurred())
		Expect(foundUser).ShouldNot(BeNil())
		Expect(foundUser.Id).ShouldNot(BeNil())

		_, err = keycloakApiClient.Users.DeleteUser(ctx, KeycloakRealmCR, *foundUser.Id)
		Expect(err).ShouldNot(HaveOccurred())

		By("Verifying user is deleted from Keycloak")
		Eventually(func(g Gomega) {
			_, _, err := keycloakApiClient.Users.FindUserByUsername(ctx, KeycloakRealmCR, user.Spec.Username)
			g.Expect(keycloakv2.IsNotFound(err)).Should(BeTrue())
		}, time.Minute, time.Second*5).Should(Succeed())

		By("Deleting KeycloakRealmUser CR - should succeed even though user doesn't exist in Keycloak")
		Expect(k8sClient.Delete(ctx, user)).Should(Succeed())
		Eventually(func(g Gomega) {
			deletedUser := &keycloakApi.KeycloakRealmUser{}
			err := k8sClient.Get(ctx, types.NamespacedName{Name: user.Name, Namespace: ns}, deletedUser)
			g.Expect(k8sErrors.IsNotFound(err)).Should(BeTrue())
		}, timeout, interval).Should(Succeed())
	})
})

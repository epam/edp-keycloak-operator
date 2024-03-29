package keycloakclient

import (
	"strings"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/Nerzal/gocloak/v12"
	corev1 "k8s.io/api/core/v1"
	k8sErrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	"github.com/epam/edp-keycloak-operator/api/common"
	keycloakApi "github.com/epam/edp-keycloak-operator/api/v1"
	"github.com/epam/edp-keycloak-operator/controllers/helper"
	"github.com/epam/edp-keycloak-operator/pkg/secretref"
)

var _ = Describe("KeycloakClient controller", Ordered, func() {
	It("Should create KeycloakClient with secret reference", func() {
		By("Creating a client secret")
		clientSecret := &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-keycloak-client-secret",
				Namespace: ns,
			},
			Data: map[string][]byte{
				"secretKey": []byte("secretValue"),
			},
		}
		Expect(k8sClient.Create(ctx, clientSecret)).Should(Succeed())
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
				Secret: secretref.GenerateSecretRef(clientSecret.Name, "secretKey"),
				Public: true,
				WebUrl: "https://test-keycloak-client-with-secret-ref",
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
			},
		}
		Expect(k8sClient.Create(ctx, keycloakClient)).Should(Succeed())
		Eventually(func() bool {
			createdKeycloakClient := &keycloakApi.KeycloakClient{}
			err := k8sClient.Get(ctx, types.NamespacedName{Name: keycloakClient.Name, Namespace: ns}, createdKeycloakClient)
			if err != nil {
				return false
			}

			return createdKeycloakClient.Status.Value == helper.StatusOK &&
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
	It("Should create KeycloakClient with empty secret", func() {
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
			},
		}
		Expect(k8sClient.Create(ctx, keycloakClient)).Should(Succeed())
		Eventually(func() bool {
			createdKeycloakClient := &keycloakApi.KeycloakClient{}
			err := k8sClient.Get(ctx, types.NamespacedName{Name: keycloakClient.Name, Namespace: ns}, createdKeycloakClient)
			if err != nil {
				return false
			}

			return createdKeycloakClient.Status.Value == helper.StatusOK &&
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
			},
		}
		Expect(k8sClient.Create(ctx, keycloakClient)).Should(Succeed())
		Eventually(func() bool {
			createdKeycloakClient := &keycloakApi.KeycloakClient{}
			err := k8sClient.Get(ctx, types.NamespacedName{Name: keycloakClient.Name, Namespace: ns}, createdKeycloakClient)
			if err != nil {
				return false
			}

			return createdKeycloakClient.Status.Value == helper.StatusOK &&
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
		By("Crating keycloak api client")
		client := gocloak.NewClient(keycloakURL)
		token, err := client.LoginAdmin(ctx, "admin", "admin", "master")
		Expect(err).ShouldNot(HaveOccurred())

		By("Creating group for policy")
		_, err = client.CreateGroup(ctx, token.AccessToken, KeycloakRealmCR, gocloak.Group{
			Name: gocloak.StringP("test-policy-group"),
		})
		Expect(skipAlreadyExistsErr(err)).ShouldNot(HaveOccurred())

		By("Creating role for policy")
		_, err = client.CreateRealmRole(ctx, token.AccessToken, KeycloakRealmCR, gocloak.Role{
			Name: gocloak.StringP("test-policy-role"),
		})
		Expect(skipAlreadyExistsErr(err)).ShouldNot(HaveOccurred())

		By("Creating user for policy")
		_, err = client.CreateUser(ctx, token.AccessToken, KeycloakRealmCR, gocloak.User{
			Username: gocloak.StringP("test-policy-user"),
			Enabled:  gocloak.BoolP(true),
		})
		Expect(skipAlreadyExistsErr(err)).ShouldNot(HaveOccurred())

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
				},
			},
		}
		Expect(k8sClient.Create(ctx, keycloakClient)).Should(Succeed())
		Eventually(func() bool {
			createdKeycloakClient := &keycloakApi.KeycloakClient{}
			if err = k8sClient.Get(ctx, types.NamespacedName{Name: keycloakClient.Name, Namespace: ns}, createdKeycloakClient); err != nil {
				return false
			}

			return createdKeycloakClient.Status.Value == helper.StatusOK &&
				createdKeycloakClient.Status.ClientID != ""
		}, timeout, interval).Should(BeTrue(), "KeycloakClient should be created successfully")

		By("Adding client permissions")

		By("Getting Client")
		clients, err := client.GetClients(ctx, token.AccessToken, KeycloakRealmCR, gocloak.GetClientsParams{
			ClientID: gocloak.StringP(keycloakClient.Spec.ClientId),
		})
		Expect(skipAlreadyExistsErr(err)).ShouldNot(HaveOccurred())
		Expect(len(clients)).Should(Equal(1))

		cl := clients[0]

		By("Creating scope for permission")
		_, err = client.CreateScope(ctx, token.AccessToken, KeycloakRealmCR, *cl.ID,
			gocloak.ScopeRepresentation{
				Name: gocloak.StringP("test-scope"),
			})
		Expect(skipAlreadyExistsErr(err)).ShouldNot(HaveOccurred())

		By("Creating resource for permission")
		_, err = client.CreateResource(ctx, token.AccessToken, KeycloakRealmCR, *cl.ID,
			gocloak.ResourceRepresentation{
				Name:               gocloak.StringP("test-resource"),
				OwnerManagedAccess: gocloak.BoolP(false),
			})
		Expect(skipAlreadyExistsErr(err)).ShouldNot(HaveOccurred())

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

		By("Waiting for the KeycloakClient will be processed at least once")
		time.Sleep(5 * time.Second)
		Eventually(func() bool {
			createdKeycloakClient := &keycloakApi.KeycloakClient{}
			err := k8sClient.Get(ctx, types.NamespacedName{Name: keycloakClient.Name, Namespace: ns}, createdKeycloakClient)
			if err != nil {
				return false
			}

			return createdKeycloakClient.Status.Value == helper.StatusOK &&
				createdKeycloakClient.Status.ClientID != ""
		}, timeout, interval).Should(BeTrue(), "KeycloakClient should be updated successfully")
	})
})

func skipAlreadyExistsErr(err error) error {
	if err == nil {
		return nil
	}

	if strings.Contains(err.Error(), "409 Conflict") {
		return nil
	}

	return err
}

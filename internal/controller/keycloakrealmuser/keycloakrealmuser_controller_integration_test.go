package keycloakrealmuser

import (
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
	"github.com/epam/edp-keycloak-operator/internal/controller/helper"
	"github.com/epam/edp-keycloak-operator/pkg/client/keycloak/adapter"
	"github.com/epam/edp-keycloak-operator/pkg/objectmeta"
)

var _ = Describe("KeycloakRealmUser controller", Ordered, func() {
	const (
		userCR         = "test-keycloak-realm-user"
		userSecretName = "test-user-secret"
	)
	It("Should create KeycloakRealmUser", func() {
		By("Creating group for user")
		_, err := keycloakApiClient.CreateGroup(ctx, getKeyCloakToken(), KeycloakRealmCR, gocloak.Group{
			Name: gocloak.StringP("test-user-group"),
		})
		Expect(adapter.SkipAlreadyExistsErr(err)).ShouldNot(HaveOccurred())

		By("Creating group and subgroup for user")
		_, err = keycloakApiClient.CreateGroup(ctx, getKeyCloakToken(), KeycloakRealmCR, gocloak.Group{
			Name: gocloak.StringP("test-user-group2"),
		})
		Expect(adapter.SkipAlreadyExistsErr(err)).ShouldNot(HaveOccurred())

		gr, err := keycloakApiClient.GetGroups(ctx, getKeyCloakToken(), KeycloakRealmCR, gocloak.GetGroupsParams{
			Search: gocloak.StringP("test-user-group2"),
		})
		Expect(adapter.SkipAlreadyExistsErr(err)).ShouldNot(HaveOccurred())
		Expect(gr).Should(HaveLen(1))

		_, err = keycloakApiClient.CreateChildGroup(ctx, getKeyCloakToken(), KeycloakRealmCR, *gr[0].ID, gocloak.Group{
			Name: gocloak.StringP("test-user-group2-subgroup"),
		})
		Expect(adapter.SkipAlreadyExistsErr(err)).ShouldNot(HaveOccurred())

		By("Creating role for user")
		_, err = keycloakApiClient.CreateRealmRole(ctx, getKeyCloakToken(), KeycloakRealmCR, gocloak.Role{
			Name: gocloak.StringP("test-user-role"),
		})
		Expect(adapter.SkipAlreadyExistsErr(err)).ShouldNot(HaveOccurred())

		By("Creating Secret for user password")
		secret := &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      userSecretName,
				Namespace: ns,
			},
			StringData: map[string]string{
				"password": "test-password",
			},
		}
		Expect(k8sClient.Create(ctx, secret)).Should(Succeed())

		By("Creating a KeycloakRealmUser")
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
					"UPDATE_PASSWORD",
				},
				Roles: []string{
					"test-user-role",
				},
				Groups: []string{
					"test-user-group",
					"test-user-group2-subgroup",
				},
				Attributes: map[string]string{
					"attr1": "test-value",
				},
				PasswordSecret: keycloakApi.PasswordSecret{
					Name: secret.Name,
					Key:  "password",
				},
				KeepResource: true,
			},
		}
		Expect(k8sClient.Create(ctx, user)).Should(Succeed())
		Eventually(func(g Gomega) {
			createdUser := &keycloakApi.KeycloakRealmUser{}
			err = k8sClient.Get(ctx, types.NamespacedName{Name: userCR, Namespace: ns}, createdUser)
			g.Expect(err).ShouldNot(HaveOccurred())
			g.Expect(createdUser.Status.Value).Should(Equal(helper.StatusOK))
		}).WithTimeout(time.Second * 20).WithPolling(time.Second).Should(Succeed())
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
			g.Expect(updatedUser.Status.Value).Should(Equal(helper.StatusOK))
		}, time.Minute, time.Second*5).Should(Succeed())

		Eventually(func(g Gomega) {
			users, err := keycloakApiClient.GetUsers(ctx, getKeyCloakToken(), KeycloakRealmCR, gocloak.GetUsersParams{
				Username: gocloak.StringP(user.Spec.Username),
				Exact:    gocloak.BoolP(true),
			})
			g.Expect(err).ShouldNot(HaveOccurred())
			g.Expect(users).Should(HaveLen(1))
			g.Expect(*users[0].FirstName).Should(Equal("new-first-name"))
			g.Expect(*users[0].LastName).Should(Equal("new-last-name"))
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
				Username: "test-user",
				PasswordSecret: keycloakApi.PasswordSecret{
					Name: userSecretName,
					Key:  "password",
				},
				KeepResource: true,
			},
		}
		Expect(k8sClient.Create(ctx, user)).Should(Succeed())
		Eventually(func(g Gomega) {
			createdUser := &keycloakApi.KeycloakRealmUser{}
			err := k8sClient.Get(ctx, types.NamespacedName{Name: user.Name, Namespace: ns}, createdUser)
			g.Expect(err).ShouldNot(HaveOccurred())
			g.Expect(createdUser.Status.Value).Should(Equal(helper.StatusOK))
		}).WithTimeout(time.Second * 20).WithPolling(time.Second).Should(Succeed())

		By("Deleting KeycloakRealmUser")
		Expect(k8sClient.Delete(ctx, user)).Should(Succeed())
		Eventually(func(g Gomega) {
			users, err := keycloakApiClient.GetUsers(ctx, getKeyCloakToken(), KeycloakRealmCR, gocloak.GetUsersParams{
				Username: gocloak.StringP(user.Spec.Username),
				Exact:    gocloak.BoolP(true),
			})
			g.Expect(err).ShouldNot(HaveOccurred())
			g.Expect(users).Should(HaveLen(1))
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
				PasswordSecret: keycloakApi.PasswordSecret{
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
			g.Expect(createdUser.Status.Value).Should(ContainSubstring("unable to get password"))
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
				PasswordSecret: keycloakApi.PasswordSecret{
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
			g.Expect(createdUser.Status.Value).Should(ContainSubstring("key invalid-key not found in secret"))
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
				PasswordSecret: keycloakApi.PasswordSecret{
					Name: userSecretName,
					Key:  "password",
				},
				Roles: []string{"invalid-role"},
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
			g.Expect(createdUser.Status.Value).Should(ContainSubstring("unable to sync realm user"))
		}, time.Second*3, time.Second).Should(Succeed())
	})
})

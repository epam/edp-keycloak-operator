package keycloakrealm

import (
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	k8sErrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/utils/ptr"

	"github.com/epam/edp-keycloak-operator/api/common"
	keycloakApi "github.com/epam/edp-keycloak-operator/api/v1"
)

var _ = Describe("KeycloakRealm externally-managed settings", Ordered, func() {
	const (
		crName    = "test-keycloak-realm-external"
		realmName = "test-realm-external-managed"
	)

	It("Should preserve externally-set realm settings not defined in the CR", func() {
		By("Creating a minimal KeycloakRealm without display name or organizations settings")
		keycloakRealm := &keycloakApi.KeycloakRealm{
			ObjectMeta: metav1.ObjectMeta{
				Name:      crName,
				Namespace: ns,
			},
			Spec: keycloakApi.KeycloakRealmSpec{
				RealmName: realmName,
				KeycloakRef: common.KeycloakRef{
					Name: keycloakCR,
					Kind: keycloakApi.KeycloakKind,
				},
			},
		}
		Expect(k8sClient.Create(ctx, keycloakRealm)).Should(Succeed())

		Eventually(func() bool {
			createdRealm := &keycloakApi.KeycloakRealm{}
			err := k8sClient.Get(ctx, types.NamespacedName{Name: crName, Namespace: ns}, createdRealm)
			Expect(err).ShouldNot(HaveOccurred())

			if !createdRealm.Status.Available {
				GinkgoWriter.Println("KeycloakRealm status error: ", createdRealm.Status.Value)
			}

			return createdRealm.Status.Available
		}, time.Minute, time.Second*5).Should(BeTrue())

		By("Setting display name and organizations flag directly in Keycloak (external tooling)")
		realm, _, err := keycloakApiClient.Realms.GetRealm(ctx, realmName)
		Expect(err).ShouldNot(HaveOccurred())

		realm.DisplayName = ptr.To("Externally Set")
		realm.DisplayNameHtml = ptr.To("<b>Externally Set</b>")
		realm.OrganizationsEnabled = ptr.To(true)
		_, err = keycloakApiClient.Realms.UpdateRealm(ctx, realmName, *realm)
		Expect(err).ShouldNot(HaveOccurred())

		By("Triggering reconciliation via an unrelated spec change")
		Expect(k8sClient.Get(ctx, types.NamespacedName{Name: crName, Namespace: ns}, keycloakRealm)).Should(Succeed())
		keycloakRealm.Spec.FrontendURL = "https://external-trigger.example.com"
		Expect(k8sClient.Update(ctx, keycloakRealm)).Should(Succeed())

		By("Verifying externally-set settings survive reconciliation")
		Eventually(func(g Gomega) {
			updatedRealm, _, err := keycloakApiClient.Realms.GetRealm(ctx, realmName)
			g.Expect(err).ShouldNot(HaveOccurred())
			g.Expect(updatedRealm).ShouldNot(BeNil())

			// Proves the reconcile loop processed the spec change.
			g.Expect(updatedRealm.Attributes).ShouldNot(BeNil())
			g.Expect((*updatedRealm.Attributes)["frontendUrl"]).Should(Equal("https://external-trigger.example.com"))

			// Fields absent from the CR must keep their externally-set values.
			g.Expect(updatedRealm.DisplayName).Should(Equal(ptr.To("Externally Set")))
			g.Expect(updatedRealm.DisplayNameHtml).Should(Equal(ptr.To("<b>Externally Set</b>")))
			g.Expect(updatedRealm.OrganizationsEnabled).Should(Equal(ptr.To(true)))
		}, time.Second*30, time.Second).Should(Succeed())
	})

	It("Should clean up the externally-managed test realm", func() {
		keycloakRealm := &keycloakApi.KeycloakRealm{}
		Expect(k8sClient.Get(ctx, types.NamespacedName{Name: crName, Namespace: ns}, keycloakRealm)).Should(Succeed())
		Expect(k8sClient.Delete(ctx, keycloakRealm)).Should(Succeed())
		Eventually(func() bool {
			deletedRealm := &keycloakApi.KeycloakRealm{}
			err := k8sClient.Get(ctx, types.NamespacedName{Name: crName, Namespace: ns}, deletedRealm)

			return k8sErrors.IsNotFound(err)
		}, timeout, interval).Should(BeTrue(), "KeycloakRealm should be deleted")
	})
})

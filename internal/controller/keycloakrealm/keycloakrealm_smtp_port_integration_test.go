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
	"github.com/epam/edp-keycloak-operator/pkg/testutils"
)

var _ = Describe("KeycloakRealm SMTP port", Ordered, func() {
	const crName = "test-keycloak-realm-smtp-port"

	realmName := testutils.RealmName("test-realm-smtp-port")

	newRealm := func(name string, port *int) *keycloakApi.KeycloakRealm {
		return &keycloakApi.KeycloakRealm{
			ObjectMeta: metav1.ObjectMeta{
				Name:      name,
				Namespace: ns,
			},
			Spec: keycloakApi.KeycloakRealmSpec{
				RealmName: realmName,
				KeycloakRef: common.KeycloakRef{
					Name: keycloakCR,
					Kind: keycloakApi.KeycloakKind,
				},
				Smtp: &common.SMTP{
					Template: common.EmailTemplate{From: "from@mail.com"},
					Connection: common.EmailConnection{
						Host:      "smtp-host",
						Port:      port,
						EnableSSL: true,
					},
				},
			},
		}
	}

	setPort := func(port *int) {
		Eventually(func(g Gomega) {
			realm := &keycloakApi.KeycloakRealm{}
			g.Expect(k8sClient.Get(ctx, types.NamespacedName{Name: crName, Namespace: ns}, realm)).Should(Succeed())

			realm.Spec.Smtp.Connection.Port = port
			g.Expect(k8sClient.Update(ctx, realm)).Should(Succeed())
		}, timeout, interval).Should(Succeed())
	}

	DescribeTable("Should reject a port outside 1..65535",
		func(port int) {
			err := k8sClient.Create(ctx, newRealm(crName+"-invalid", ptr.To(port)))

			Expect(k8sErrors.IsInvalid(err)).Should(BeTrue(), "expected an Invalid error, got: %v", err)
		},
		Entry("zero", 0),
		Entry("above the range", 65536),
	)

	It("Should leave the port unset in the CR and absent in Keycloak when omitted", func() {
		Expect(k8sClient.Create(ctx, newRealm(crName, nil))).Should(Succeed())

		created := &keycloakApi.KeycloakRealm{}
		Expect(k8sClient.Get(ctx, types.NamespacedName{Name: crName, Namespace: ns}, created)).Should(Succeed())
		Expect(created.Spec.Smtp.Connection.Port).Should(BeNil())

		Eventually(func() bool {
			current := &keycloakApi.KeycloakRealm{}
			Expect(k8sClient.Get(ctx, types.NamespacedName{Name: crName, Namespace: ns}, current)).Should(Succeed())

			return current.Status.Available
		}, time.Minute, time.Second*5).Should(BeTrue())

		Eventually(func(g Gomega) {
			realm, _, err := keycloakApiClient.Realms.GetRealm(ctx, realmName)
			g.Expect(err).ShouldNot(HaveOccurred())
			g.Expect(realm.SmtpServer).ShouldNot(BeNil())
			g.Expect(*realm.SmtpServer).Should(HaveKeyWithValue("ssl", "true"))
			g.Expect(*realm.SmtpServer).ShouldNot(HaveKey("port"))
		}, time.Second*30, time.Second).Should(Succeed())
	})

	It("Should send an explicit port and remove it again when it is dropped from spec", func() {
		setPort(ptr.To(2525))

		Eventually(func(g Gomega) {
			realm, _, err := keycloakApiClient.Realms.GetRealm(ctx, realmName)
			g.Expect(err).ShouldNot(HaveOccurred())
			g.Expect(realm.SmtpServer).ShouldNot(BeNil())
			g.Expect(*realm.SmtpServer).Should(HaveKeyWithValue("port", "2525"))
		}, time.Second*30, time.Second).Should(Succeed())

		setPort(nil)

		Eventually(func(g Gomega) {
			realm, _, err := keycloakApiClient.Realms.GetRealm(ctx, realmName)
			g.Expect(err).ShouldNot(HaveOccurred())
			g.Expect(realm.SmtpServer).ShouldNot(BeNil())
			g.Expect(*realm.SmtpServer).ShouldNot(HaveKey("port"))
		}, time.Second*30, time.Second).Should(Succeed())
	})

	It("Should clean up the SMTP port test realm", func() {
		realm := &keycloakApi.KeycloakRealm{}
		Expect(k8sClient.Get(ctx, types.NamespacedName{Name: crName, Namespace: ns}, realm)).Should(Succeed())
		Expect(k8sClient.Delete(ctx, realm)).Should(Succeed())
		Eventually(func() bool {
			deleted := &keycloakApi.KeycloakRealm{}
			err := k8sClient.Get(ctx, types.NamespacedName{Name: crName, Namespace: ns}, deleted)

			return k8sErrors.IsNotFound(err)
		}, timeout, interval).Should(BeTrue(), "KeycloakRealm should be deleted")
	})
})

package keycloakclient

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	corev1 "k8s.io/api/core/v1"
	k8sErrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/epam/edp-keycloak-operator/api/common"
	keycloakApi "github.com/epam/edp-keycloak-operator/api/v1"
	"github.com/epam/edp-keycloak-operator/internal/controller/keycloakclient/chain"
	"github.com/epam/edp-keycloak-operator/pkg/secretref"
)

// Confidential client without spec.secret: [] clears both lists in Keycloak; omitted keys fall back to webUrl.
var _ = Describe("KeycloakClient explicit empty webOrigins and redirectUris", Ordered, func() {
	const (
		crName     = "empty-uris-keycloak-client"
		secretName = "keycloak-client-" + crName + "-secret"
		webUrl     = "https://empty-uris.example.com"
	)

	key := types.NamespacedName{Name: crName, Namespace: ns}

	waitReady := func() *keycloakApi.KeycloakClient {
		got := &keycloakApi.KeycloakClient{}

		Eventually(func(g Gomega) {
			g.Expect(k8sClient.Get(ctx, key, got)).To(Succeed())
			g.Expect(got.Status.Value).To(Equal(common.StatusOK))

			ready := meta.FindStatusCondition(got.Status.Conditions, chain.ConditionReady)
			g.Expect(ready).NotTo(BeNil())
			g.Expect(ready.Status).To(Equal(metav1.ConditionTrue))
			g.Expect(ready.ObservedGeneration).To(Equal(got.Generation))
		}, timeout, interval).Should(Succeed())

		return got
	}

	keycloakLists := func(g Gomega) ([]string, []string) {
		kc, _, err := keycloakAdmin.Clients.GetClientByClientID(ctx, KeycloakRealmCR, crName)
		g.Expect(err).NotTo(HaveOccurred())
		g.Expect(kc).NotTo(BeNil())

		return ptr.Deref(kc.RedirectUris, nil), ptr.Deref(kc.WebOrigins, nil)
	}

	setLists := func(redirectUris, webOrigins *[]string) {
		cr := &keycloakApi.KeycloakClient{}
		Expect(k8sClient.Get(ctx, key, cr)).To(Succeed())

		original := cr.DeepCopy()
		cr.Spec.RedirectUris = redirectUris
		cr.Spec.WebOrigins = webOrigins

		Expect(k8sClient.Patch(ctx, cr, client.MergeFrom(original))).To(Succeed())
	}

	AfterAll(func() {
		cr := &keycloakApi.KeycloakClient{ObjectMeta: metav1.ObjectMeta{Name: crName, Namespace: ns}}
		Expect(client.IgnoreNotFound(k8sClient.Delete(ctx, cr))).To(Succeed())

		Eventually(func() bool {
			return k8sErrors.IsNotFound(k8sClient.Get(ctx, key, &keycloakApi.KeycloakClient{}))
		}, timeout, interval).Should(BeTrue())

		secret := &corev1.Secret{ObjectMeta: metav1.ObjectMeta{Name: secretName, Namespace: ns}}
		Expect(client.IgnoreNotFound(k8sClient.Delete(ctx, secret))).To(Succeed())
	})

	It("Should clear both lists for a confidential client created with [] and no secret", func() {
		cr := &keycloakApi.KeycloakClient{
			ObjectMeta: metav1.ObjectMeta{Name: crName, Namespace: ns},
			Spec: keycloakApi.KeycloakClientSpec{
				ClientId: crName,
				RealmRef: common.RealmRef{
					Name: KeycloakRealmCR,
					Kind: keycloakApi.KeycloakRealmKind,
				},
				WebUrl:       webUrl,
				RedirectUris: ptr.To([]string{}),
				WebOrigins:   ptr.To([]string{}),
			},
		}
		Expect(k8sClient.Create(ctx, cr)).To(Succeed())

		got := waitReady()
		Expect(secretref.HasSecretRef(got.Spec.Secret)).To(BeTrue())
		Expect(got.Spec.RedirectUris).To(Equal(ptr.To([]string{})))
		Expect(got.Spec.WebOrigins).To(Equal(ptr.To([]string{})))

		redirectUris, webOrigins := keycloakLists(Default)
		Expect(redirectUris).To(BeEmpty())
		Expect(webOrigins).To(BeEmpty())
	})

	It("Should apply an explicit list after an empty one", func() {
		setLists(ptr.To([]string{webUrl + "/cb"}), ptr.To([]string{webUrl}))
		waitReady()

		Eventually(func(g Gomega) {
			redirectUris, webOrigins := keycloakLists(g)
			g.Expect(redirectUris).To(ConsistOf(webUrl + "/cb"))
			g.Expect(webOrigins).To(ConsistOf(webUrl))
		}, timeout, interval).Should(Succeed())
	})

	It("Should clear both lists again when edited from a list to []", func() {
		setLists(ptr.To([]string{}), ptr.To([]string{}))
		waitReady()

		Eventually(func(g Gomega) {
			redirectUris, webOrigins := keycloakLists(g)
			g.Expect(redirectUris).To(BeEmpty())
			g.Expect(webOrigins).To(BeEmpty())
		}, timeout, interval).Should(Succeed())
	})

	It("Should fall back to webUrl when both keys are removed", func() {
		setLists(nil, nil)
		waitReady()

		Eventually(func(g Gomega) {
			redirectUris, webOrigins := keycloakLists(g)
			g.Expect(redirectUris).To(ConsistOf(webUrl + "/*"))
			g.Expect(webOrigins).To(ConsistOf(webUrl))
		}, timeout, interval).Should(Succeed())
	})
})

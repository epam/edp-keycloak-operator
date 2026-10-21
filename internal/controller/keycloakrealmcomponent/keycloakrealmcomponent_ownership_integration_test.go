package keycloakrealmcomponent

import (
	"errors"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	k8sErrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/utils/ptr"

	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/epam/edp-keycloak-operator/api/common"
	keycloakApi "github.com/epam/edp-keycloak-operator/api/v1"
	"github.com/epam/edp-keycloak-operator/pkg/client/keycloakapi"
	"github.com/epam/edp-keycloak-operator/pkg/objectmeta"
	"github.com/epam/edp-keycloak-operator/pkg/testutils"
)

// Ownership of a Keycloak component by one KeycloakRealmComponent CR.
var _ = Describe("KeycloakRealmComponent ownership", Ordered, func() {
	const (
		settle   = testutils.Settle
		longWait = testutils.LongWait

		sharedName = "ownership-shared"
		ownCR      = "ownership-own"
		dupCR      = "ownership-dup"
	)

	var (
		recorder *testutils.AdminEventRecorder
		ownID    string
		ownRec   *testutils.AdminEventRecorder
	)

	crKey := func(name string) types.NamespacedName {
		return types.NamespacedName{Name: name, Namespace: ns}
	}

	newCR := func(crName string) *keycloakApi.KeycloakRealmComponent {
		return &keycloakApi.KeycloakRealmComponent{
			ObjectMeta: metav1.ObjectMeta{Name: crName, Namespace: ns},
			Spec: keycloakApi.KeycloakComponentSpec{
				Name:         sharedName,
				ProviderID:   scopeProviderID,
				ProviderType: scopeProviderType,
				RealmRef:     common.RealmRef{Kind: keycloakApi.KeycloakRealmKind, Name: KeycloakRealmCR},
			},
		}
	}

	gone := func(crName string) bool {
		return k8sErrors.IsNotFound(k8sClient.Get(ctx, crKey(crName), &keycloakApi.KeycloakRealmComponent{}))
	}

	// Releases a CR without a Keycloak delete: preserve annotation, then finalizer removal.
	release := func(crName string) error {
		cr := &keycloakApi.KeycloakRealmComponent{}
		if err := k8sClient.Get(ctx, crKey(crName), cr); err != nil {
			return client.IgnoreNotFound(err)
		}

		patch := client.MergeFrom(cr.DeepCopy())
		cr.Annotations = map[string]string{objectmeta.PreserveResourcesOnDeletionAnnotation: "true"}

		if err := k8sClient.Patch(ctx, cr, patch); err != nil {
			return client.IgnoreNotFound(err)
		}

		if err := k8sClient.Delete(ctx, cr); err != nil {
			return client.IgnoreNotFound(err)
		}

		return nil
	}

	BeforeAll(func() {
		recorder = testutils.NewAdminEventRecorder(keycloakApiClient, KeycloakRealmCR)
		Expect(recorder.Enable(ctx)).To(Succeed())
	})

	AfterAll(func() {
		errs := make([]error, 0, 2)
		errs = append(errs, release(dupCR))

		own := &keycloakApi.KeycloakRealmComponent{ObjectMeta: metav1.ObjectMeta{Name: ownCR, Namespace: ns}}
		errs = append(errs, client.IgnoreNotFound(k8sClient.Delete(ctx, own)))

		Expect(errors.Join(errs...)).To(Succeed())

		for _, crName := range []string{dupCR, ownCR} {
			Eventually(func() bool { return gone(crName) }, longWait, interval).Should(BeTrue())
		}

		realm, _, err := keycloakApiClient.Realms.GetRealm(ctx, KeycloakRealmCR)
		Expect(err).ShouldNot(HaveOccurred())

		left, _, err := keycloakApiClient.RealmComponents.GetComponents(ctx, KeycloakRealmCR,
			&keycloakapi.GetComponentsParams{Name: ptr.To(sharedName), Parent: realm.Id})
		Expect(err).ShouldNot(HaveOccurred())

		for _, c := range left {
			_, err := keycloakApiClient.RealmComponents.DeleteComponent(ctx, KeycloakRealmCR, ptr.Deref(c.Id, ""))
			if err != nil && !keycloakapi.IsNotFound(err) {
				Expect(err).ShouldNot(HaveOccurred())
			}
		}
	})

	It("refuses a duplicate CR", func() {
		Expect(k8sClient.Create(ctx, newCR(ownCR))).To(Succeed())

		Eventually(func(g Gomega) {
			cr := &keycloakApi.KeycloakRealmComponent{}
			g.Expect(k8sClient.Get(ctx, crKey(ownCR), cr)).To(Succeed())
			g.Expect(cr.Status.Value).To(Equal(common.StatusOK))
			g.Expect(cr.Status.ID).ShouldNot(BeEmpty())

			ownID = cr.Status.ID
		}, longWait, interval).Should(Succeed())

		ownRec = recorder.Scoped("components/" + ownID)

		// Create after ownCR has status.id.
		Expect(ownRec.WritesDuring(ctx, settle, func() {
			Expect(k8sClient.Create(ctx, newCR(dupCR))).To(Succeed())

			Eventually(func(g Gomega) {
				cr := &keycloakApi.KeycloakRealmComponent{}
				g.Expect(k8sClient.Get(ctx, crKey(dupCR), cr)).To(Succeed())
				g.Expect(cr.Status.Value).To(ContainSubstring("already managed by KeycloakRealmComponent " + ns + "/" + ownCR))
				g.Expect(cr.Status.ID).To(BeEmpty())
			}, longWait, interval).Should(Succeed())
		})).To(BeEmpty())

		own := &keycloakApi.KeycloakRealmComponent{}
		Expect(k8sClient.Get(ctx, crKey(ownCR), own)).To(Succeed())
		Expect(own.Status.ID).To(Equal(ownID))
		Expect(own.Status.Value).To(Equal(common.StatusOK))
	})
})

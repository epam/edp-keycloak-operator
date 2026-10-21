package keycloakrealmcomponent

import (
	"context"
	"errors"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	k8sErrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/utils/ptr"

	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/epam/edp-keycloak-operator/api/common"
	keycloakApi "github.com/epam/edp-keycloak-operator/api/v1"
	"github.com/epam/edp-keycloak-operator/pkg/client/keycloakapi"
	"github.com/epam/edp-keycloak-operator/pkg/objectmeta"
	"github.com/epam/edp-keycloak-operator/pkg/testutils"
)

var _ = Describe("KeycloakRealmComponent ownership", Ordered, func() {
	const (
		settle   = testutils.Settle
		longWait = testutils.LongWait

		sharedName = "ownership-shared"
		ownCR      = "ownership-own"
		dupCR      = "ownership-dup"
		dup2CR     = "ownership-dup2"
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

	annotatePreserve := func(crName string) {
		Eventually(func(g Gomega) {
			cr := &keycloakApi.KeycloakRealmComponent{}
			err := k8sClient.Get(ctx, crKey(crName), cr)
			if k8sErrors.IsNotFound(err) {
				return
			}

			g.Expect(err).To(Succeed())

			patch := client.MergeFrom(cr.DeepCopy())

			if cr.Annotations == nil {
				cr.Annotations = map[string]string{}
			}

			cr.Annotations[objectmeta.PreserveResourcesOnDeletionAnnotation] = "true"
			g.Expect(k8sClient.Patch(ctx, cr, patch)).To(Succeed())
		}, longWait, interval).Should(Succeed())
	}

	// Releases a CR without a Keycloak delete: preserve annotation, then finalizer removal.
	release := func(crName string) error {
		annotatePreserve(crName)

		cr := &keycloakApi.KeycloakRealmComponent{ObjectMeta: metav1.ObjectMeta{Name: crName, Namespace: ns}}

		return client.IgnoreNotFound(k8sClient.Delete(ctx, cr))
	}

	stripFinalizers := func(crName string) error {
		cr := &keycloakApi.KeycloakRealmComponent{}
		if err := k8sClient.Get(ctx, crKey(crName), cr); err != nil {
			return client.IgnoreNotFound(err)
		}

		return client.IgnoreNotFound(k8sClient.Patch(ctx, cr, client.RawPatch(types.MergePatchType,
			[]byte(`{"metadata":{"finalizers":null}}`))))
	}

	// A blocked delete keeps the CR with its deletionTimestamp and finalizer and the component in Keycloak (status.id set).
	expectDeleteBlocked := func(crName string) {
		Expect(ownRec.WritesDuring(ctx, 0, func() {
			Consistently(func(g Gomega) {
				cr := &keycloakApi.KeycloakRealmComponent{}
				g.Expect(k8sClient.Get(ctx, crKey(crName), cr)).To(Succeed())
				g.Expect(cr.DeletionTimestamp).ShouldNot(BeNil())
				g.Expect(cr.Finalizers).ShouldNot(BeEmpty())

				rep, _, err := keycloakApiClient.RealmComponents.GetComponent(ctx, KeycloakRealmCR, ownID)
				g.Expect(err).ShouldNot(HaveOccurred())
				g.Expect(rep).ShouldNot(BeNil())
			}, 5*time.Second, interval).Should(Succeed())
		})).To(BeEmpty())
	}

	expectReleasedWithComponentKept := func(crName string) {
		annotatePreserve(crName)

		Eventually(func() bool { return gone(crName) }, longWait, interval).Should(BeTrue())

		rep, _, err := keycloakApiClient.RealmComponents.GetComponent(ctx, KeycloakRealmCR, ownID)
		Expect(err).ShouldNot(HaveOccurred())
		Expect(rep).ShouldNot(BeNil())
	}

	BeforeAll(func() {
		recorder = testutils.NewAdminEventRecorder(keycloakApiClient, KeycloakRealmCR)
		Expect(recorder.Enable(ctx)).To(Succeed())
	})

	AfterAll(func() {
		var errs []error

		errs = append(errs, release(dup2CR), release(dupCR))

		for _, crName := range []string{dup2CR, dupCR} {
			released := wait.PollUntilContextTimeout(ctx, interval, longWait, true,
				func(context.Context) (bool, error) { return gone(crName), nil }) == nil
			if !released {
				errs = append(errs, stripFinalizers(crName))
			}
		}

		own := &keycloakApi.KeycloakRealmComponent{ObjectMeta: metav1.ObjectMeta{Name: ownCR, Namespace: ns}}
		errs = append(errs, client.IgnoreNotFound(k8sClient.Delete(ctx, own)))

		Expect(errors.Join(errs...)).To(Succeed())

		for _, crName := range []string{dup2CR, dupCR, ownCR} {
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

	It("deletes the refused CR without touching the owner", func() {
		dup := &keycloakApi.KeycloakRealmComponent{}
		Expect(k8sClient.Get(ctx, crKey(dupCR), dup)).To(Succeed())
		Expect(dup.Status.ID).To(BeEmpty())
		Expect(dup.Annotations).NotTo(HaveKey(objectmeta.PreserveResourcesOnDeletionAnnotation))

		Expect(ownRec.WritesDuring(ctx, settle, func() {
			Expect(k8sClient.Delete(ctx, dup)).To(Succeed())
			Eventually(func() bool { return gone(dupCR) }, longWait, interval).Should(BeTrue())
		})).To(BeEmpty())

		rep, _, err := keycloakApiClient.RealmComponents.GetComponent(ctx, KeycloakRealmCR, ownID)
		Expect(err).ShouldNot(HaveOccurred())
		Expect(rep).ShouldNot(BeNil())

		own := &keycloakApi.KeycloakRealmComponent{}
		Expect(k8sClient.Get(ctx, crKey(ownCR), own)).To(Succeed())
		Expect(own.DeletionTimestamp).To(BeNil())
		Expect(own.Status.ID).To(Equal(ownID))
		Expect(own.Status.Value).To(Equal(common.StatusOK))
	})

	It("blocks deleting one CR of a shared-ID pair", func() {
		Expect(k8sClient.Create(ctx, newCR(dup2CR))).To(Succeed())

		Eventually(func(g Gomega) {
			cr := &keycloakApi.KeycloakRealmComponent{}
			g.Expect(k8sClient.Get(ctx, crKey(dup2CR), cr)).To(Succeed())
			g.Expect(cr.Status.Value).To(ContainSubstring("already managed by KeycloakRealmComponent " + ns + "/" + ownCR))
		}, longWait, interval).Should(Succeed())

		Eventually(func(g Gomega) {
			cr := &keycloakApi.KeycloakRealmComponent{}
			g.Expect(k8sClient.Get(ctx, crKey(dup2CR), cr)).To(Succeed())

			patch := client.MergeFrom(cr.DeepCopy())
			cr.Status.ID = ownID
			g.Expect(k8sClient.Status().Patch(ctx, cr, patch)).To(Succeed())
		}, longWait, interval).Should(Succeed())

		Eventually(func(g Gomega) {
			cr := &keycloakApi.KeycloakRealmComponent{}
			g.Expect(k8sClient.Get(ctx, crKey(dup2CR), cr)).To(Succeed())
			g.Expect(cr.Status.Value).To(Equal(common.StatusOK))
			g.Expect(cr.Status.ID).To(Equal(ownID))
		}, longWait, interval).Should(Succeed())

		dup2 := &keycloakApi.KeycloakRealmComponent{}
		Expect(k8sClient.Get(ctx, crKey(dup2CR), dup2)).To(Succeed())
		Expect(k8sClient.Delete(ctx, dup2)).To(Succeed())

		expectDeleteBlocked(dup2CR)
		expectReleasedWithComponentKept(dup2CR)
	})
})

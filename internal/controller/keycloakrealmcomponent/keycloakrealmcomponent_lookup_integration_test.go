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
	"github.com/epam/edp-keycloak-operator/pkg/testutils"
)

// Lookup identity of KeycloakRealmComponent: (name, parent, providerType).
var _ = Describe("KeycloakRealmComponent lookup identity", Ordered, func() {
	const (
		settle   = testutils.Settle
		longWait = testutils.LongWait

		pairName  = "lookup-same-name"
		keyCR     = "lookup-key"
		policyCR  = "lookup-policy"
		parent1CR = "lookup-parent-1"
		parent2CR = "lookup-parent-2"
		child1CR  = "lookup-child-1"
		child2CR  = "lookup-child-2"
		childName = "lookup-same-child"
	)

	var (
		recorder  *testutils.AdminEventRecorder
		realmID   string
		keyID     string
		policyID  string
		keyRec    *testutils.AdminEventRecorder
		policyRec *testutils.AdminEventRecorder
	)

	crKey := func(name string) types.NamespacedName {
		return types.NamespacedName{Name: name, Namespace: ns}
	}

	newCR := func(
		crName, name, providerID, providerType string,
		parentRef *keycloakApi.ParentComponent,
		config map[string][]string,
	) *keycloakApi.KeycloakRealmComponent {
		return &keycloakApi.KeycloakRealmComponent{
			ObjectMeta: metav1.ObjectMeta{Name: crName, Namespace: ns},
			Spec: keycloakApi.KeycloakComponentSpec{
				Name:         name,
				ProviderID:   providerID,
				ProviderType: providerType,
				RealmRef:     common.RealmRef{Kind: keycloakApi.KeycloakRealmKind, Name: KeycloakRealmCR},
				ParentRef:    parentRef,
				Config:       config,
			},
		}
	}

	waitID := func(crName string) string {
		var id string

		Eventually(func(g Gomega) {
			cr := &keycloakApi.KeycloakRealmComponent{}
			g.Expect(k8sClient.Get(ctx, crKey(crName), cr)).To(Succeed())
			g.Expect(cr.Status.Value).To(Equal(common.StatusOK))
			g.Expect(cr.Status.ID).ShouldNot(BeEmpty())

			id = cr.Status.ID
		}, longWait, interval).Should(Succeed())

		return id
	}

	nudge := func(crNames ...string) {
		for _, crName := range crNames {
			Eventually(func(g Gomega) {
				g.Expect(testutils.Nudge(ctx, k8sClient, crKey(crName), &keycloakApi.KeycloakRealmComponent{})).To(Succeed())
			}, longWait, interval).Should(Succeed())
		}
	}

	clearStatusID := func(crName string) {
		Eventually(func(g Gomega) {
			cr := &keycloakApi.KeycloakRealmComponent{}
			g.Expect(k8sClient.Get(ctx, crKey(crName), cr)).To(Succeed())

			patch := client.MergeFrom(cr.DeepCopy())
			cr.Status.ID = ""
			g.Expect(k8sClient.Status().Patch(ctx, cr, patch)).To(Succeed())
		}, longWait, interval).Should(Succeed())
	}

	idsUnderName := func(g Gomega, name string) []string {
		found, _, err := keycloakApiClient.RealmComponents.GetComponents(ctx, KeycloakRealmCR,
			&keycloakapi.GetComponentsParams{Name: ptr.To(name), Parent: ptr.To(realmID)})
		g.Expect(err).ShouldNot(HaveOccurred())

		ids := make([]string, 0, len(found))
		for _, c := range found {
			ids = append(ids, ptr.Deref(c.Id, ""))
		}

		return ids
	}

	kcComponent := func(g Gomega, id string) *keycloakapi.ComponentRepresentation {
		rep, _, err := keycloakApiClient.RealmComponents.GetComponent(ctx, KeycloakRealmCR, id)
		g.Expect(err).ShouldNot(HaveOccurred())
		g.Expect(rep).ShouldNot(BeNil())

		return rep
	}

	BeforeAll(func() {
		recorder = testutils.NewAdminEventRecorder(keycloakApiClient, KeycloakRealmCR)
		Expect(recorder.Enable(ctx)).To(Succeed())

		realm, _, err := keycloakApiClient.Realms.GetRealm(ctx, KeycloakRealmCR)
		Expect(err).ShouldNot(HaveOccurred())

		realmID = ptr.Deref(realm.Id, "")
		Expect(realmID).ShouldNot(BeEmpty())
	})

	AfterAll(func() {
		names := []string{child2CR, child1CR, parent2CR, parent1CR, policyCR, keyCR}

		errs := make([]error, 0, len(names))

		for _, crName := range names {
			cr := &keycloakApi.KeycloakRealmComponent{ObjectMeta: metav1.ObjectMeta{Name: crName, Namespace: ns}}
			errs = append(errs, client.IgnoreNotFound(k8sClient.Delete(ctx, cr)))
		}

		Expect(errors.Join(errs...)).To(Succeed())

		for _, crName := range names {
			Eventually(func() bool {
				return k8sErrors.IsNotFound(k8sClient.Get(ctx, crKey(crName), &keycloakApi.KeycloakRealmComponent{}))
			}, longWait, interval).Should(BeTrue())
		}
	})

	It("creates one component per providerType for the same name and parent", func() {
		Expect(k8sClient.Create(ctx,
			newCR(keyCR, pairName, keyProviderID, keyProviderType, nil, map[string][]string{"priority": {"0"}}),
		)).To(Succeed())
		keyID = waitID(keyCR)

		// Create after keyCR has status.id.
		Expect(k8sClient.Create(ctx, newCR(policyCR, pairName, scopeProviderID, scopeProviderType, nil, nil))).To(Succeed())
		policyID = waitID(policyCR)

		Expect(policyID).ShouldNot(Equal(keyID))

		Eventually(func(g Gomega) {
			key := kcComponent(g, keyID)
			g.Expect(ptr.Deref(key.ProviderId, "")).To(Equal(keyProviderID))
			g.Expect(ptr.Deref(key.ProviderType, "")).To(Equal(keyProviderType))
			g.Expect(ptr.Deref(key.ParentId, "")).To(Equal(realmID))

			policy := kcComponent(g, policyID)
			g.Expect(ptr.Deref(policy.ProviderId, "")).To(Equal(scopeProviderID))
			g.Expect(ptr.Deref(policy.ProviderType, "")).To(Equal(scopeProviderType))
		}, longWait, interval).Should(Succeed())

		Expect(idsUnderName(Default, pairName)).To(ConsistOf(keyID, policyID))

		keyRec = recorder.Scoped("components/" + keyID)
		policyRec = recorder.Scoped("components/" + policyID)
	})

	It("does not write to either component on forced reconciles", func() {
		Expect(keyRec.WritesDuring(ctx, settle, func() { nudge(keyCR, policyCR) })).To(BeEmpty())
		Expect(policyRec.WritesDuring(ctx, settle, func() { nudge(keyCR, policyCR) })).To(BeEmpty())
	})

	It("writes only to the edited component", func() {
		// Asserts the event recorder sees a known write.
		Expect(keyRec.WritesDuring(ctx, 0, func() {
			Eventually(func(g Gomega) {
				cr := &keycloakApi.KeycloakRealmComponent{}
				g.Expect(k8sClient.Get(ctx, crKey(keyCR), cr)).To(Succeed())
				cr.Spec.Config = map[string][]string{"priority": {"1"}}
				g.Expect(k8sClient.Update(ctx, cr)).To(Succeed())
			}, longWait, interval).Should(Succeed())

			Eventually(func(g Gomega) {
				g.Expect((*kcComponent(g, keyID).Config)["priority"]).To(ConsistOf("1"))
			}, longWait, interval).Should(Succeed())
		})).ShouldNot(BeEmpty(), "admin event log recorded nothing for a known write")

		Expect(policyRec.WritesDuring(ctx, settle, func() { nudge(policyCR) })).To(BeEmpty())

		Eventually(func(g Gomega) {
			g.Expect(ptr.Deref(kcComponent(g, policyID).ProviderType, "")).To(Equal(scopeProviderType))
		}, longWait, interval).Should(Succeed())
	})

	It("re-adopts its own component by name and providerType without status.id", func() {
		clearStatusID(policyCR)

		Expect(policyRec.WritesDuring(ctx, settle, func() {
			nudge(policyCR)

			Eventually(func(g Gomega) {
				cr := &keycloakApi.KeycloakRealmComponent{}
				g.Expect(k8sClient.Get(ctx, crKey(policyCR), cr)).To(Succeed())
				g.Expect(cr.Status.ID).To(Equal(policyID))
				g.Expect(idsUnderName(g, pairName)).To(ConsistOf(keyID, policyID))
			}, longWait, interval).Should(Succeed())
		})).To(BeEmpty())
	})

	It("deleting one CR keeps the other CR's component", func() {
		cr := &keycloakApi.KeycloakRealmComponent{}
		Expect(k8sClient.Get(ctx, crKey(policyCR), cr)).To(Succeed())
		Expect(k8sClient.Delete(ctx, cr)).To(Succeed())

		Eventually(func() bool {
			return k8sErrors.IsNotFound(k8sClient.Get(ctx, crKey(policyCR), &keycloakApi.KeycloakRealmComponent{}))
		}, longWait, interval).Should(BeTrue())

		Eventually(func(g Gomega) {
			_, _, err := keycloakApiClient.RealmComponents.GetComponent(ctx, KeycloakRealmCR, policyID)
			g.Expect(err).To(HaveOccurred())
			g.Expect(keycloakapi.IsNotFound(err)).To(BeTrue())
		}, longWait, interval).Should(Succeed())

		Eventually(func(g Gomega) {
			g.Expect(ptr.Deref(kcComponent(g, keyID).ProviderType, "")).To(Equal(keyProviderType))
		}, longWait, interval).Should(Succeed())
	})

	It("keeps same-name children under two parents apart", func() {
		parentRef := func(name string) *keycloakApi.ParentComponent {
			return &keycloakApi.ParentComponent{Kind: keycloakApi.KeycloakRealmComponentKind, Name: name}
		}

		Expect(k8sClient.Create(ctx, newCR(parent1CR, parent1CR, scopeProviderID, scopeProviderType, nil, nil))).To(Succeed())
		Expect(k8sClient.Create(ctx, newCR(parent2CR, parent2CR, scopeProviderID, scopeProviderType, nil, nil))).To(Succeed())

		parent1ID := waitID(parent1CR)
		parent2ID := waitID(parent2CR)

		Expect(k8sClient.Create(ctx,
			newCR(child1CR, childName, scopeProviderID, scopeProviderType, parentRef(parent1CR), nil))).To(Succeed())
		child1ID := waitID(child1CR)

		// Create after child1CR has status.id.
		Expect(k8sClient.Create(ctx,
			newCR(child2CR, childName, scopeProviderID, scopeProviderType, parentRef(parent2CR), nil))).To(Succeed())
		child2ID := waitID(child2CR)

		Expect(child2ID).ShouldNot(Equal(child1ID))

		Eventually(func(g Gomega) {
			g.Expect(ptr.Deref(kcComponent(g, child1ID).ParentId, "")).To(Equal(parent1ID))
			g.Expect(ptr.Deref(kcComponent(g, child2ID).ParentId, "")).To(Equal(parent2ID))
		}, longWait, interval).Should(Succeed())

		child1Rec := recorder.Scoped("components/" + child1ID)
		child2Rec := recorder.Scoped("components/" + child2ID)

		Expect(child1Rec.WritesDuring(ctx, settle, func() { nudge(child1CR, child2CR) })).To(BeEmpty())
		Expect(child2Rec.WritesDuring(ctx, settle, func() { nudge(child1CR, child2CR) })).To(BeEmpty())
	})
})

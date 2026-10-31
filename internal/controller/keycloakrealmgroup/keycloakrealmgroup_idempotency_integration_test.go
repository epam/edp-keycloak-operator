package keycloakrealmgroup

import (
	"maps"
	"slices"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	k8sErrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/epam/edp-keycloak-operator/api/common"
	keycloakApi "github.com/epam/edp-keycloak-operator/api/v1"
	"github.com/epam/edp-keycloak-operator/pkg/client/keycloakapi"
	"github.com/epam/edp-keycloak-operator/pkg/testutils"
)

func idemGroupKey(name string) types.NamespacedName {
	return types.NamespacedName{Name: name, Namespace: ns}
}

func idemGetGroupCR(g Gomega, name string) *keycloakApi.KeycloakRealmGroup {
	cr := &keycloakApi.KeycloakRealmGroup{}
	g.Expect(k8sClient.Get(ctx, idemGroupKey(name), cr)).To(Succeed())

	return cr
}

// Merge patch on an annotation. A typed Update erases an explicit empty spec.attributes.
func idemNudgeGroupCR(g Gomega, name string) {
	cr := idemGetGroupCR(g, name)
	original := cr.DeepCopy()

	if cr.Annotations == nil {
		cr.Annotations = map[string]string{}
	}

	cr.Annotations[testutils.NudgeAnnotation] = time.Now().Format(time.RFC3339Nano)
	g.Expect(k8sClient.Patch(ctx, cr, client.MergeFrom(original))).To(Succeed())
}

func idemPatchGroupSpec(g Gomega, name string, mutate func(*keycloakApi.KeycloakRealmGroupSpec)) {
	cr := idemGetGroupCR(g, name)
	original := cr.DeepCopy()

	mutate(&cr.Spec)
	g.Expect(k8sClient.Patch(ctx, cr, client.MergeFrom(original))).To(Succeed())
}

func idemWaitGroupReady(name string) string {
	var id string

	Eventually(func(g Gomega) {
		cr := idemGetGroupCR(g, name)
		g.Expect(cr.Status.Value).To(Equal(common.StatusOK))
		g.Expect(cr.Status.ID).ShouldNot(BeEmpty())

		id = cr.Status.ID
	}, testutils.LongWait, interval).Should(Succeed())

	return id
}

func idemRemoveGroupCR(name string) {
	err := k8sClient.Delete(ctx, &keycloakApi.KeycloakRealmGroup{
		ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: ns},
	})
	if k8sErrors.IsNotFound(err) {
		return
	}

	Expect(err).ShouldNot(HaveOccurred())

	Eventually(func() bool {
		return k8sErrors.IsNotFound(k8sClient.Get(ctx, idemGroupKey(name), &keycloakApi.KeycloakRealmGroup{}))
	}, testutils.LongWait, interval).Should(BeTrue())
}

func idemKeycloakGroup(g Gomega, id string) *keycloakapi.GroupRepresentation {
	rep, _, err := keycloakApiClient.Groups.GetGroup(ctx, KeycloakRealmCR, id)
	g.Expect(err).ShouldNot(HaveOccurred())
	g.Expect(rep).ShouldNot(BeNil())

	return rep
}

var _ = Describe("KeycloakRealmGroup idempotent reconcile", Ordered, func() {
	const (
		settle   = testutils.Settle
		longWait = testutils.LongWait
	)

	var (
		crName   = testutils.UniqueName("grp-idem")
		k2Values = []string{"z", "a", "m", "b", "q"}
		recorder *testutils.AdminEventRecorder
		groupID  string
	)

	nudge := func() { idemNudgeGroupCR(Default, crName) }

	BeforeAll(func() {
		base := testutils.NewAdminEventRecorder(keycloakApiClient, KeycloakRealmCR)
		Expect(base.Enable(ctx)).To(Succeed())

		Expect(k8sClient.Create(ctx, &keycloakApi.KeycloakRealmGroup{
			ObjectMeta: metav1.ObjectMeta{Name: crName, Namespace: ns},
			Spec: keycloakApi.KeycloakRealmGroupSpec{
				Name:        crName,
				Description: "d1",
				RealmRef:    common.RealmRef{Kind: keycloakApi.KeycloakRealmKind, Name: KeycloakRealmCR},
				Attributes: map[string][]string{
					"k1":    {"v1"},
					"k2":    slices.Clone(k2Values),
					"empty": {},
					"blank": {""},
				},
			},
		})).To(Succeed())
		DeferCleanup(func() { idemRemoveGroupCR(crName) })

		groupID = idemWaitGroupReady(crName)

		// Sibling specs share the realm; report only this group's events.
		recorder = base.Scoped("groups/" + groupID)

		Eventually(func(g Gomega) {
			rep := idemKeycloakGroup(g, groupID)
			attrs := ptr.Deref(rep.Attributes, nil)

			g.Expect(ptr.Deref(rep.Description, "")).To(Equal("d1"))
			g.Expect(attrs).To(HaveKeyWithValue("k1", ConsistOf("v1")))
			g.Expect(attrs).To(HaveKeyWithValue("k2", ConsistOf(k2Values)))
			g.Expect(attrs).To(HaveKeyWithValue("blank", ConsistOf("")))
			g.Expect(attrs).ShouldNot(HaveKey("empty"))
		}, longWait, interval).Should(Succeed())
	})

	It("Should not write to Keycloak on a forced reconcile", func() {
		Expect(recorder.WritesDuring(ctx, settle, nudge)).To(BeEmpty())
	})

	It("Should not write when only the order of attribute values changes in the spec", func() {
		Expect(recorder.WritesDuring(ctx, settle, func() {
			idemPatchGroupSpec(Default, crName, func(s *keycloakApi.KeycloakRealmGroupSpec) {
				slices.Reverse(s.Attributes["k2"])
			})
		})).To(BeEmpty())
	})

	It("Should repair out-of-band changes and then stay quiet", func() {
		// The drift write below lands in this window: the assertion proves the event log records writes.
		// The operator's repair write is not counted apart from it.
		Expect(recorder.WritesDuring(ctx, 0, func() {
			rep := idemKeycloakGroup(Default, groupID)
			attrs := maps.Clone(ptr.Deref(rep.Attributes, nil))
			attrs["extra"] = []string{"1"}
			rep.Description = ptr.To("drift")
			rep.Attributes = &attrs

			_, err := keycloakApiClient.Groups.UpdateGroup(ctx, KeycloakRealmCR, groupID, *rep)
			Expect(err).ShouldNot(HaveOccurred())

			nudge()

			Eventually(func(g Gomega) {
				healed := idemKeycloakGroup(g, groupID)
				g.Expect(ptr.Deref(healed.Description, "")).To(Equal("d1"))
				g.Expect(ptr.Deref(healed.Attributes, nil)).ShouldNot(HaveKey("extra"))
			}, longWait, interval).Should(Succeed())
		})).ShouldNot(BeEmpty(), "admin event log recorded nothing for a known write")

		Expect(recorder.WritesDuring(ctx, settle, nudge)).To(BeEmpty())
	})

	It("Should apply a spec edit that removes an attribute and then stay quiet", func() {
		Expect(recorder.WritesDuring(ctx, 0, func() {
			idemPatchGroupSpec(Default, crName, func(s *keycloakApi.KeycloakRealmGroupSpec) {
				s.Description = "d2"
				delete(s.Attributes, "k1")
			})

			Eventually(func(g Gomega) {
				rep := idemKeycloakGroup(g, groupID)
				attrs := ptr.Deref(rep.Attributes, nil)

				g.Expect(ptr.Deref(rep.Description, "")).To(Equal("d2"))
				g.Expect(attrs).ShouldNot(HaveKey("k1"))
				g.Expect(attrs).To(HaveKey("k2"))
			}, longWait, interval).Should(Succeed())
		})).ShouldNot(BeEmpty(), "a spec edit must reach Keycloak")

		Expect(recorder.WritesDuring(ctx, settle, nudge)).To(BeEmpty())
	})

	It("Should not write when the attributes block is removed from the spec", func() {
		Expect(recorder.WritesDuring(ctx, settle, func() {
			idemPatchGroupSpec(Default, crName, func(s *keycloakApi.KeycloakRealmGroupSpec) {
				s.Attributes = nil
			})
		})).To(BeEmpty())

		Expect(ptr.Deref(idemKeycloakGroup(Default, groupID).Attributes, nil)).To(HaveKey("k2"))
	})

	It("Should not write when status.id is cleared", func() {
		// Asserts the outcome only; unit tests cover the by-name path.
		Expect(recorder.WritesDuring(ctx, settle, func() {
			cr := idemGetGroupCR(Default, crName)
			original := cr.DeepCopy()

			cr.Status.ID = ""
			Expect(k8sClient.Status().Patch(ctx, cr, client.MergeFrom(original))).To(Succeed())
		})).To(BeEmpty())

		Eventually(func(g Gomega) {
			g.Expect(idemGetGroupCR(g, crName).Status.ID).To(Equal(groupID))
		}, longWait, interval).Should(Succeed())
	})
})

var _ = Describe("KeycloakRealmGroup adoption by name", Ordered, func() {
	const (
		settle   = testutils.Settle
		longWait = testutils.LongWait
	)

	var (
		crName    = testutils.UniqueName("grp-adopt")
		groupName = testutils.UniqueName("grp-adopt")
		recorder  *testutils.AdminEventRecorder
		groupID   string
	)

	BeforeAll(func() {
		base := testutils.NewAdminEventRecorder(keycloakApiClient, KeycloakRealmCR)
		Expect(base.Enable(ctx)).To(Succeed())

		resp, err := keycloakApiClient.Groups.CreateGroup(ctx, KeycloakRealmCR, keycloakapi.GroupRepresentation{
			Name:       ptr.To(groupName),
			Attributes: &map[string][]string{"stale": {"x"}},
		})
		Expect(err).ShouldNot(HaveOccurred())

		groupID = keycloakapi.GetResourceIDFromResponse(resp)
		Expect(groupID).ShouldNot(BeEmpty())

		// Runs after the CR removal; NotFound means the finalizer already deleted the group.
		DeferCleanup(func() {
			_, delErr := keycloakApiClient.Groups.DeleteGroup(ctx, KeycloakRealmCR, groupID)
			if delErr != nil && !keycloakapi.IsNotFound(delErr) {
				Expect(delErr).ShouldNot(HaveOccurred())
			}
		})
		DeferCleanup(func() { idemRemoveGroupCR(crName) })

		recorder = base.Scoped("groups/" + groupID)
	})

	It("Should adopt the group, replace its attributes with the empty spec map and then stay quiet", func() {
		// Unstructured keeps the explicit empty attributes map; a typed create drops it.
		cr := &unstructured.Unstructured{Object: map[string]any{
			"apiVersion": keycloakApi.GroupVersion.String(),
			"kind":       "KeycloakRealmGroup",
			"metadata":   map[string]any{"name": crName, "namespace": ns},
			"spec": map[string]any{
				"name":       groupName,
				"attributes": map[string]any{},
				"realmRef":   map[string]any{"kind": keycloakApi.KeycloakRealmKind, "name": KeycloakRealmCR},
			},
		}}

		Expect(recorder.WritesDuring(ctx, 0, func() {
			Expect(k8sClient.Create(ctx, cr)).To(Succeed())

			Eventually(func(g Gomega) {
				stored := idemGetGroupCR(g, crName)
				g.Expect(stored.Status.Value).To(Equal(common.StatusOK))
				g.Expect(stored.Status.ID).To(Equal(groupID))
				g.Expect(ptr.Deref(idemKeycloakGroup(g, groupID).Attributes, nil)).To(BeEmpty())
			}, longWait, interval).Should(Succeed())
		})).ShouldNot(BeEmpty(), "adopting a group with stale attributes must write once")

		Expect(recorder.WritesDuring(ctx, settle, func() { idemNudgeGroupCR(Default, crName) })).To(BeEmpty())
	})
})

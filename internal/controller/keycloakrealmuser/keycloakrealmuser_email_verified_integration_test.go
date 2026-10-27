package keycloakrealmuser

import (
	"fmt"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/epam/edp-keycloak-operator/api/common"
	keycloakApi "github.com/epam/edp-keycloak-operator/api/v1"
	"github.com/epam/edp-keycloak-operator/pkg/client/keycloakapi"
	"github.com/epam/edp-keycloak-operator/pkg/testutils"
)

var _ = Describe("KeycloakRealmUser spec.emailVerified", func() {
	var crdClient client.Client

	getCR := func(name string) *unstructured.Unstructured {
		got := &unstructured.Unstructured{}
		got.SetGroupVersionKind(keycloakApi.GroupVersion.WithKind("KeycloakRealmUser"))
		Expect(crdClient.Get(ctx, types.NamespacedName{Name: name, Namespace: ns}, got)).Should(Succeed())

		return got
	}

	specEmailVerified := func(name string) (value, present bool) {
		v, found, err := unstructured.NestedBool(getCR(name).Object, "spec", "emailVerified")
		Expect(err).ShouldNot(HaveOccurred())

		return v, found
	}

	keycloakUser := func(g Gomega, username string) *keycloakapi.UserRepresentation {
		u, _, err := keycloakApiClient.Users.FindUserByUsername(ctx, KeycloakRealmCR, username)
		g.Expect(err).ShouldNot(HaveOccurred())
		g.Expect(u).ShouldNot(BeNil())

		return u
	}

	keycloakEmailVerified := func(username string) bool {
		return ptr.Deref(keycloakUser(Default, username).EmailVerified, false)
	}

	setKeycloakEmailVerified := func(username string, emailVerified bool) {
		u := keycloakUser(Default, username)
		u.EmailVerified = ptr.To(emailVerified)
		_, err := keycloakApiClient.Users.UpdateUser(ctx, KeycloakRealmCR, *u.Id, *u)
		Expect(err).ShouldNot(HaveOccurred())
	}

	awaitReconciled := func(name string) {
		Eventually(func(g Gomega) {
			cr := &keycloakApi.KeycloakRealmUser{}
			g.Expect(k8sClient.Get(ctx, types.NamespacedName{Name: name, Namespace: ns}, cr)).Should(Succeed())
			g.Expect(cr.Status.Value).Should(Equal(common.StatusOK))
		}, timeout, interval).Should(Succeed())
	}

	// nudge: merge-patch spec.firstName, wait until Keycloak holds it.
	nudge := func(name, username, firstName string) {
		patch := fmt.Sprintf(`{"spec":{"firstName":%q}}`, firstName)
		Expect(crdClient.Patch(ctx, newUserCR(name, username, nil), client.RawPatch(types.MergePatchType, []byte(patch)))).
			Should(Succeed())

		Eventually(func(g Gomega) {
			g.Expect(ptr.Deref(keycloakUser(g, username).FirstName, "")).Should(Equal(firstName))
		}, timeout, interval).Should(Succeed())
	}

	BeforeEach(func() {
		scheme := runtime.NewScheme()
		Expect(keycloakApi.AddToScheme(scheme)).ShouldNot(HaveOccurred())

		var err error
		crdClient, err = client.New(cfg, client.Options{Scheme: scheme})
		Expect(err).ShouldNot(HaveOccurred())
	})

	It("Should not change emailVerified of an existing user across reconciles when spec.emailVerified is omitted", func() {
		crName := testutils.UniqueName("ev-omitted")

		Expect(crdClient.Create(ctx, newUserCR(crName, crName, nil))).Should(Succeed())
		awaitReconciled(crName)
		Expect(keycloakEmailVerified(crName)).Should(BeFalse())
		nudge(crName, crName, "n1")

		setKeycloakEmailVerified(crName, true)

		nudge(crName, crName, "n2")
		Expect(keycloakEmailVerified(crName)).Should(BeTrue())

		nudge(crName, crName, "n3")
		Expect(keycloakEmailVerified(crName)).Should(BeTrue())

		_, present := specEmailVerified(crName)
		Expect(present).Should(BeFalse(), "no CRD default: the stored spec keeps the field absent")
	})

	It("Should keep emailVerified of a verified user when spec.email changes and spec.emailVerified is omitted", func() {
		crName := testutils.UniqueName("ev-email")
		newEmail := crName + "-new@example.com"

		Expect(crdClient.Create(ctx, newUserCR(crName, crName, map[string]any{
			"email": crName + "@example.com",
		}))).Should(Succeed())
		awaitReconciled(crName)
		nudge(crName, crName, "n1")

		setKeycloakEmailVerified(crName, true)

		patch := fmt.Sprintf(`{"spec":{"email":%q}}`, newEmail)
		Expect(crdClient.Patch(ctx, newUserCR(crName, crName, nil), client.RawPatch(types.MergePatchType, []byte(patch)))).
			Should(Succeed())

		Eventually(func(g Gomega) {
			u := keycloakUser(g, crName)
			g.Expect(ptr.Deref(u.Email, "")).Should(Equal(newEmail))
			g.Expect(ptr.Deref(u.EmailVerified, false)).Should(BeTrue())
		}, timeout, interval).Should(Succeed())
	})

	It("Should keep an explicit false through the operator's own full-object update", func() {
		crName := testutils.UniqueName("ev-false-migr")

		// The deprecated attributes field drives migrateAttributes, which writes the whole
		// object back with client.Update.
		Expect(crdClient.Create(ctx, newUserCR(crName, crName, map[string]any{
			"emailVerified": false,
			"attributes":    map[string]any{"department": "IT"},
		}))).Should(Succeed())
		awaitReconciled(crName)

		Eventually(func(g Gomega) {
			_, found, err := unstructured.NestedMap(getCR(crName).Object, "spec", "attributesV2")
			g.Expect(err).ShouldNot(HaveOccurred())
			g.Expect(found).Should(BeTrue(), "the attribute migration must have written the object back")
		}, timeout, interval).Should(Succeed())

		value, present := specEmailVerified(crName)
		Expect(present).Should(BeTrue())
		Expect(value).Should(BeFalse(), "the migration write must not drop a stored false")
	})

	It("Should unverify a user verified in Keycloak when spec.emailVerified is false", func() {
		crName := testutils.UniqueName("ev-false")

		Expect(crdClient.Create(ctx, newUserCR(crName, crName, map[string]any{
			"emailVerified": false,
		}))).Should(Succeed())
		awaitReconciled(crName)
		nudge(crName, crName, "n1")

		setKeycloakEmailVerified(crName, true)

		nudge(crName, crName, "n2")
		Expect(keycloakEmailVerified(crName)).Should(BeFalse())

		nudge(crName, crName, "n3")
		Expect(keycloakEmailVerified(crName)).Should(BeFalse())
	})

	It("Should verify a user unverified in Keycloak when spec.emailVerified is true", func() {
		crName := testutils.UniqueName("ev-true")

		Expect(crdClient.Create(ctx, newUserCR(crName, crName, map[string]any{
			"emailVerified": true,
		}))).Should(Succeed())
		awaitReconciled(crName)
		Expect(keycloakEmailVerified(crName)).Should(BeTrue())
		nudge(crName, crName, "n1")

		setKeycloakEmailVerified(crName, false)

		nudge(crName, crName, "n2")
		Expect(keycloakEmailVerified(crName)).Should(BeTrue())

		nudge(crName, crName, "n3")
		Expect(keycloakEmailVerified(crName)).Should(BeTrue())
	})
})

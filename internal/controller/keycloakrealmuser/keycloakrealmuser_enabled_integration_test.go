package keycloakrealmuser

import (
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strings"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	k8sErrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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

func newUserCR(name, username string, spec map[string]any) *unstructured.Unstructured {
	base := map[string]any{
		"realmRef": map[string]any{
			"kind": keycloakApi.KeycloakRealmKind,
			"name": KeycloakRealmCR,
		},
		"username":     username,
		"keepResource": true,
	}
	for k, v := range spec {
		base[k] = v
	}

	u := &unstructured.Unstructured{}
	u.SetUnstructuredContent(map[string]any{
		"apiVersion": keycloakApi.GroupVersion.String(),
		"kind":       "KeycloakRealmUser",
		"metadata": map[string]any{
			"name":      name,
			"namespace": ns,
		},
		"spec": base,
	})

	return u
}

var _ = Describe("KeycloakRealmUser spec.enabled", func() {
	var crdClient client.Client

	getCR := func(name string) *unstructured.Unstructured {
		got := &unstructured.Unstructured{}
		got.SetGroupVersionKind(keycloakApi.GroupVersion.WithKind("KeycloakRealmUser"))
		Expect(crdClient.Get(ctx, types.NamespacedName{Name: name, Namespace: ns}, got)).Should(Succeed())

		return got
	}

	specEnabled := func(name string) (value, present bool) {
		v, found, err := unstructured.NestedBool(getCR(name).Object, "spec", "enabled")
		Expect(err).ShouldNot(HaveOccurred())

		return v, found
	}

	keycloakUser := func(g Gomega, username string) *keycloakapi.UserRepresentation {
		u, _, err := keycloakApiClient.Users.FindUserByUsername(ctx, KeycloakRealmCR, username)
		g.Expect(err).ShouldNot(HaveOccurred())
		g.Expect(u).ShouldNot(BeNil())

		return u
	}

	keycloakUserEnabled := func(username string) bool {
		return *keycloakUser(Default, username).Enabled
	}

	setKeycloakEnabled := func(username string, enabled bool) {
		u := keycloakUser(Default, username)
		u.Enabled = ptr.To(enabled)
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

	It("Should create an enabled user and store no value when spec.enabled is omitted", func() {
		crName := testutils.UniqueName("enabled-omitted")

		Expect(crdClient.Create(ctx, newUserCR(crName, crName, nil))).Should(Succeed())
		awaitReconciled(crName)

		_, present := specEnabled(crName)
		Expect(present).Should(BeFalse(), "no CRD default: the stored spec keeps the field absent")
		Expect(keycloakUserEnabled(crName)).Should(BeTrue())
	})

	It("Should create a disabled user when spec.enabled is false", func() {
		crName := testutils.UniqueName("enabled-false")

		Expect(crdClient.Create(ctx, newUserCR(crName, crName, map[string]any{
			"enabled": false,
		}))).Should(Succeed())
		awaitReconciled(crName)

		value, present := specEnabled(crName)
		Expect(present).Should(BeTrue())
		Expect(value).Should(BeFalse())
		Expect(keycloakUserEnabled(crName)).Should(BeFalse())
	})

	It("Should keep an explicit false through the operator's own full-object update", func() {
		crName := testutils.UniqueName("enabled-false-migr")

		// The deprecated attributes field drives migrateAttributes, which writes the whole
		// object back with client.Update.
		Expect(crdClient.Create(ctx, newUserCR(crName, crName, map[string]any{
			"enabled":    false,
			"attributes": map[string]any{"department": "IT"},
		}))).Should(Succeed())
		awaitReconciled(crName)

		Eventually(func(g Gomega) {
			_, found, err := unstructured.NestedMap(getCR(crName).Object, "spec", "attributesV2")
			g.Expect(err).ShouldNot(HaveOccurred())
			g.Expect(found).Should(BeTrue(), "the attribute migration must have written the object back")
		}, timeout, interval).Should(Succeed())

		value, present := specEnabled(crName)
		Expect(present).Should(BeTrue())
		Expect(value).Should(BeFalse(), "the migration write must not drop a stored false")

		Consistently(func() bool {
			return keycloakUserEnabled(crName)
		}, "3s", interval).Should(BeFalse())
	})

	It("Should not change enabled of an existing user across reconciles when spec.enabled is omitted", func() {
		crName := testutils.UniqueName("enabled-keep")

		Expect(crdClient.Create(ctx, newUserCR(crName, crName, nil))).Should(Succeed())
		awaitReconciled(crName)
		nudge(crName, crName, "n1")

		setKeycloakEnabled(crName, false)

		nudge(crName, crName, "n2")
		Expect(keycloakUserEnabled(crName)).Should(BeFalse())

		nudge(crName, crName, "n3")
		Expect(keycloakUserEnabled(crName)).Should(BeFalse())

		_, present := specEnabled(crName)
		Expect(present).Should(BeFalse())
	})

	It("Should re-enable a user disabled in Keycloak when spec.enabled is true", func() {
		crName := testutils.UniqueName("enabled-true")

		Expect(crdClient.Create(ctx, newUserCR(crName, crName, map[string]any{
			"enabled": true,
		}))).Should(Succeed())
		awaitReconciled(crName)

		setKeycloakEnabled(crName, false)
		nudge(crName, crName, "n1")

		Expect(keycloakUserEnabled(crName)).Should(BeTrue())
	})
})

var _ = Describe("KeycloakRealmUser spec.enabled and brute-force protection", func() {
	It("Should keep login failures and the temporary lockout when spec.enabled matches Keycloak", func() {
		realmName := testutils.RealmName("test-user-brute-force")
		userCR := testutils.UniqueName("bf-user")

		realm := &keycloakApi.KeycloakRealm{
			ObjectMeta: metav1.ObjectMeta{Name: realmName, Namespace: ns},
			Spec: keycloakApi.KeycloakRealmSpec{
				RealmName:   realmName,
				KeycloakRef: common.KeycloakRef{Kind: keycloakApi.KeycloakKind, Name: KeycloakCR},
				BruteForceDetection: &common.BruteForceDetection{
					BruteForceProtected:          ptr.To(true),
					PermanentLockout:             ptr.To(false),
					FailureFactor:                ptr.To(2),
					WaitIncrementSeconds:         ptr.To(900),
					MaxFailureWaitSeconds:        ptr.To(900),
					MinimumQuickLoginWaitSeconds: ptr.To(900),
				},
			},
		}

		Expect(k8sClient.Create(ctx, realm)).Should(Succeed())
		DeferCleanup(func() {
			Expect(client.IgnoreNotFound(k8sClient.Delete(ctx, realm))).Should(Succeed())
			Eventually(func() bool {
				return k8sErrors.IsNotFound(
					k8sClient.Get(ctx, types.NamespacedName{Name: realmName, Namespace: ns}, &keycloakApi.KeycloakRealm{}),
				)
			}, testutils.LongWait, interval).Should(BeTrue())
		})

		Eventually(func(g Gomega) {
			got := &keycloakApi.KeycloakRealm{}
			g.Expect(k8sClient.Get(ctx, types.NamespacedName{Name: realmName, Namespace: ns}, got)).Should(Succeed())
			g.Expect(got.Status.Available).Should(BeTrue())
		}, testutils.LongWait, interval).Should(Succeed())

		user := &keycloakApi.KeycloakRealmUser{
			ObjectMeta: metav1.ObjectMeta{Name: userCR, Namespace: ns},
			Spec: keycloakApi.KeycloakRealmUserSpec{
				RealmRef:     common.RealmRef{Kind: keycloakApi.KeycloakRealmKind, Name: realmName},
				Username:     userCR,
				Enabled:      ptr.To(true),
				KeepResource: true,
			},
		}

		Expect(k8sClient.Create(ctx, user)).Should(Succeed())
		DeferCleanup(func() {
			Expect(client.IgnoreNotFound(k8sClient.Delete(ctx, user))).Should(Succeed())
			Eventually(func() bool {
				return k8sErrors.IsNotFound(
					k8sClient.Get(ctx, types.NamespacedName{Name: userCR, Namespace: ns}, &keycloakApi.KeycloakRealmUser{}),
				)
			}, testutils.LongWait, interval).Should(BeTrue())
		})

		keycloakUser := func(g Gomega) *keycloakapi.UserRepresentation {
			u, _, err := keycloakApiClient.Users.FindUserByUsername(ctx, realmName, userCR)
			g.Expect(err).ShouldNot(HaveOccurred())
			g.Expect(u).ShouldNot(BeNil())

			return u
		}

		var userID string

		Eventually(func(g Gomega) {
			cr := &keycloakApi.KeycloakRealmUser{}
			g.Expect(k8sClient.Get(ctx, types.NamespacedName{Name: userCR, Namespace: ns}, cr)).Should(Succeed())
			g.Expect(cr.Status.Value).Should(Equal(common.StatusOK))

			userID = ptr.Deref(keycloakUser(g).Id, "")
			g.Expect(userID).ShouldNot(BeEmpty())
		}, testutils.LongWait, interval).Should(Succeed())

		_, err := keycloakApiClient.Users.SetUserPassword(ctx, realmName, userID, keycloakapi.CredentialRepresentation{
			Type:      ptr.To("password"),
			Value:     ptr.To("Passw0rd-1"),
			Temporary: ptr.To(false),
		})
		Expect(err).ShouldNot(HaveOccurred())

		tokenURL := fmt.Sprintf("%s/realms/%s/protocol/openid-connect/token", os.Getenv("TEST_KEYCLOAK_URL"), realmName)

		for range 2 {
			form := url.Values{
				"client_id":  {keycloakapi.DefaultAdminClientID},
				"grant_type": {"password"},
				"username":   {userCR},
				"password":   {"wrong-password"},
			}

			req, reqErr := http.NewRequestWithContext(ctx, http.MethodPost, tokenURL, strings.NewReader(form.Encode()))
			Expect(reqErr).ShouldNot(HaveOccurred())
			req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

			resp, postErr := http.DefaultClient.Do(req)
			Expect(postErr).ShouldNot(HaveOccurred())
			Expect(resp.Body.Close()).Should(Succeed())
			Expect(resp.StatusCode).Should(BeElementOf(http.StatusBadRequest, http.StatusUnauthorized))
		}

		assertLockedOut := func(g Gomega) {
			status, _, statusErr := keycloakApiClient.Events.GetBruteForceStatus(ctx, realmName, userID)
			g.Expect(statusErr).ShouldNot(HaveOccurred())
			g.Expect(status).Should(HaveKeyWithValue("numFailures", BeEquivalentTo(2)))
			g.Expect(status).Should(HaveKeyWithValue("disabled", BeTrue()))
		}

		Eventually(assertLockedOut, testutils.LongWait, interval).Should(Succeed())

		nudge := func(firstName string) {
			patch := fmt.Sprintf(`{"spec":{"firstName":%q}}`, firstName)
			Expect(k8sClient.Patch(ctx, user, client.RawPatch(types.MergePatchType, []byte(patch)))).Should(Succeed())

			Eventually(func(g Gomega) {
				g.Expect(ptr.Deref(keycloakUser(g).FirstName, "")).Should(Equal(firstName))
			}, timeout, interval).Should(Succeed())
		}

		nudge("n1")
		nudge("n2")

		Consistently(assertLockedOut, testutils.Settle, interval).Should(Succeed())
	})
})

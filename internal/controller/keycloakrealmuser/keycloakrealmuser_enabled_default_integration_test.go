package keycloakrealmuser

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/epam/edp-keycloak-operator/api/common"
	keycloakApi "github.com/epam/edp-keycloak-operator/api/v1"
)

// The CRD schema resolves spec.enabled, so these specs build the resource as an unstructured
// object. A typed KeycloakRealmUserSpec cannot express an absent field: Go sends the zero value
// and the API server never applies the default.
var _ = Describe("KeycloakRealmUser spec.enabled defaulting", Ordered, func() {
	const crdName = "keycloakrealmusers.v1.edp.epam.com"

	var crdClient client.Client

	newUserCR := func(name, username string, spec map[string]any) *unstructured.Unstructured {
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

	keycloakUserEnabled := func(username string) bool {
		u, _, err := keycloakApiClient.Users.FindUserByUsername(ctx, KeycloakRealmCR, username)
		Expect(err).ShouldNot(HaveOccurred())
		Expect(u).ShouldNot(BeNil())

		return *u.Enabled
	}

	awaitReconciled := func(name string) {
		Eventually(func(g Gomega) {
			cr := &keycloakApi.KeycloakRealmUser{}
			g.Expect(k8sClient.Get(ctx, types.NamespacedName{Name: name, Namespace: ns}, cr)).Should(Succeed())
			g.Expect(cr.Status.Value).Should(Equal(common.StatusOK))
		}, timeout, interval).Should(Succeed())
	}

	// setSchemaDefault writes or clears spec.enabled.default on the served CRD, so a spec can
	// observe the pre-fix schema and the post-fix schema in one run.
	setSchemaDefault := func(enabled *bool) {
		crd := &apiextensionsv1.CustomResourceDefinition{}
		Expect(crdClient.Get(ctx, types.NamespacedName{Name: crdName}, crd)).Should(Succeed())

		for i := range crd.Spec.Versions {
			props := crd.Spec.Versions[i].Schema.OpenAPIV3Schema.Properties["spec"].Properties["enabled"]
			if enabled == nil {
				props.Default = nil
			} else {
				props.Default = &apiextensionsv1.JSON{Raw: []byte("true")}
			}

			crd.Spec.Versions[i].Schema.OpenAPIV3Schema.Properties["spec"].Properties["enabled"] = props
		}

		Expect(crdClient.Update(ctx, crd)).Should(Succeed())
	}

	// A CRD schema update reaches the serving handler asynchronously. A dry-run create runs
	// defaulting without persisting anything, so it polls for the new schema without waking
	// the controller.
	awaitSchemaDefault := func(want *bool) {
		Eventually(func(g Gomega) {
			probe := newUserCR("schema-probe", "schema-probe-user", nil)
			g.Expect(crdClient.Create(ctx, probe, client.DryRunAll)).Should(Succeed())

			value, found, err := unstructured.NestedBool(probe.Object, "spec", "enabled")
			g.Expect(err).ShouldNot(HaveOccurred())

			if want == nil {
				g.Expect(found).Should(BeFalse())
				return
			}

			g.Expect(found).Should(BeTrue())
			g.Expect(value).Should(Equal(*want))
		}, timeout, interval).Should(Succeed())
	}

	BeforeAll(func() {
		scheme := runtime.NewScheme()
		Expect(keycloakApi.AddToScheme(scheme)).ShouldNot(HaveOccurred())
		Expect(apiextensionsv1.AddToScheme(scheme)).ShouldNot(HaveOccurred())

		var err error
		crdClient, err = client.New(cfg, client.Options{Scheme: scheme})
		Expect(err).ShouldNot(HaveOccurred())
	})

	It("Should create an enabled user when spec.enabled is omitted", func() {
		const crName, username = "enabled-omitted", "enabled-omitted-user"

		Expect(crdClient.Create(ctx, newUserCR(crName, username, nil))).Should(Succeed())
		awaitReconciled(crName)

		value, present := specEnabled(crName)
		Expect(present).Should(BeTrue(), "the API server must materialise the default into the stored spec")
		Expect(value).Should(BeTrue())

		Expect(keycloakUserEnabled(username)).Should(BeTrue())
	})

	It("Should create a disabled user when spec.enabled is false", func() {
		const crName, username = "enabled-explicit-false", "enabled-explicit-false-user"

		Expect(crdClient.Create(ctx, newUserCR(crName, username, map[string]any{
			"enabled": false,
		}))).Should(Succeed())
		awaitReconciled(crName)

		value, present := specEnabled(crName)
		Expect(present).Should(BeTrue())
		Expect(value).Should(BeFalse(), "the default must not overwrite an explicit false")

		Expect(keycloakUserEnabled(username)).Should(BeFalse())
	})

	It("Should keep an explicit false through the operator's own full-object update", func() {
		const crName, username = "enabled-false-migrated", "enabled-false-migrated-user"

		// The deprecated attributes field drives migrateAttributes, the one path that writes the
		// whole object back with client.Update. That write must not re-default an explicit false.
		Expect(crdClient.Create(ctx, newUserCR(crName, username, map[string]any{
			"enabled":    false,
			"attributes": map[string]any{"department": "IT"},
		}))).Should(Succeed())
		awaitReconciled(crName)

		Eventually(func(g Gomega) {
			_, found, err := unstructured.NestedMap(getCR(crName).Object, "spec", "attributesV2")
			g.Expect(err).ShouldNot(HaveOccurred())
			g.Expect(found).Should(BeTrue(), "the attribute migration must have written the object back")
		}, timeout, interval).Should(Succeed())

		value, _ := specEnabled(crName)
		Expect(value).Should(BeFalse(), "the migration write must not flip a disabled user on")

		Consistently(func() bool {
			return keycloakUserEnabled(username)
		}, "3s", interval).Should(BeFalse())
	})

	// Upgrading the CRD makes an existing resource that omitted spec.enabled reconcile to an
	// enabled user.
	// Assert against the API server, not the manager's client. The informer keeps serving the
	// schema its watch was established with until the object is written again; an operator that
	// restarts on upgrade never sees that stale view.
	It("Should enable a user that omitted spec.enabled once the CRD carries the default", func() {
		const crName, username = "enabled-upgrade", "enabled-upgrade-user"

		By("serving the pre-fix schema, which carries no default")
		setSchemaDefault(nil)
		DeferCleanup(func() {
			setSchemaDefault(ptr.To(true))
			awaitSchemaDefault(ptr.To(true))
		})
		awaitSchemaDefault(nil)

		By("creating a CR that omits spec.enabled")
		Expect(crdClient.Create(ctx, newUserCR(crName, username, nil))).Should(Succeed())
		awaitReconciled(crName)

		_, present := specEnabled(crName)
		Expect(present).Should(BeFalse(), "the pre-fix schema stores no value")

		By("upgrading the CRD to the post-fix schema")
		setSchemaDefault(ptr.To(true))
		awaitSchemaDefault(ptr.To(true))

		By("reading the untouched CR: defaulting applies on read, with no write to the resource")
		Eventually(func(g Gomega) {
			value, found := specEnabled(crName)
			g.Expect(found).Should(BeTrue())
			g.Expect(value).Should(BeTrue())
		}, timeout, interval).Should(Succeed())

		By("reconciling the resource as the operator does after it restarts on upgrade")
		touched := getCR(crName)
		touched.SetAnnotations(map[string]string{"test.edp.epam.com/resync": "1"})
		Expect(crdClient.Update(ctx, touched)).Should(Succeed())

		Eventually(func() bool {
			return keycloakUserEnabled(username)
		}, timeout, interval).Should(BeTrue(), "the release note: the user is switched on")
	})
})

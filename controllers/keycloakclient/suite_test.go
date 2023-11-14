package keycloakclient

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/rest"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"

	"github.com/epam/edp-keycloak-operator/api/common"
	keycloakApi "github.com/epam/edp-keycloak-operator/api/v1"
	"github.com/epam/edp-keycloak-operator/controllers/helper"
	"github.com/epam/edp-keycloak-operator/controllers/keycloak"
	"github.com/epam/edp-keycloak-operator/controllers/keycloakrealm"
)

var (
	cfg       *rest.Config
	k8sClient client.Client
	testEnv   *envtest.Environment
	ctx       context.Context
	cancel    context.CancelFunc
)

const (
	KeycloakCR      = "test-keycloak"
	KeycloakRealmCR = "test-keycloak-realm"
	ns              = "test-client"

	timeout  = time.Second * 10
	interval = time.Millisecond * 250
)

func TestKeycloakClient(t *testing.T) {
	RegisterFailHandler(Fail)

	if os.Getenv("TEST_KEYCLOAK_URL") == "" {
		t.Skip("TEST_KEYCLOAK_URL is not set")
	}

	RunSpecs(t, "KeycloakClient Controller Suite")
}

var _ = BeforeSuite(func() {
	logf.SetLogger(zap.New(zap.WriteTo(GinkgoWriter), zap.UseDevMode(true)))

	ctx, cancel = context.WithCancel(context.Background())
	ctx = ctrl.LoggerInto(ctx, logf.Log)

	By("bootstrapping test environment")
	testEnv = &envtest.Environment{
		CRDDirectoryPaths:     []string{filepath.Join("../..", "config", "crd", "bases")},
		ErrorIfCRDPathMissing: true,
	}

	var err error
	cfg, err = testEnv.Start()
	Expect(err).NotTo(HaveOccurred())
	Expect(cfg).NotTo(BeNil())

	scheme := runtime.NewScheme()
	Expect(keycloakApi.AddToScheme(scheme)).NotTo(HaveOccurred())
	Expect(corev1.AddToScheme(scheme)).NotTo(HaveOccurred())

	k8sClient, err = client.New(cfg, client.Options{Scheme: scheme})
	Expect(err).NotTo(HaveOccurred())
	Expect(k8sClient).NotTo(BeNil())

	k8sManager, err := ctrl.NewManager(cfg, ctrl.Options{
		Scheme:             scheme,
		MetricsBindAddress: "0",
	})
	Expect(err).ToNot(HaveOccurred())

	h := helper.MakeHelper(k8sManager.GetClient(), k8sManager.GetScheme(), "default")

	err = keycloak.NewReconcileKeycloak(k8sManager.GetClient(), k8sManager.GetScheme(), h).
		SetupWithManager(k8sManager, 0)
	Expect(err).ToNot(HaveOccurred())

	err = keycloakrealm.NewReconcileKeycloakRealm(k8sManager.GetClient(), k8sManager.GetScheme(), h).
		SetupWithManager(k8sManager, 0)
	Expect(err).ToNot(HaveOccurred())

	err = NewReconcileKeycloakClient(k8sManager.GetClient(), h, scheme).
		SetupWithManager(k8sManager, 0)
	Expect(err).ToNot(HaveOccurred())

	go func() {
		defer GinkgoRecover()
		err = k8sManager.Start(ctx)
		Expect(err).ToNot(HaveOccurred(), "failed to run manager")
	}()

	By("bootstrapping Keycloak and KeycloakRealm")
	namespace := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: ns,
		},
	}
	err = k8sClient.Create(ctx, namespace)
	Expect(err).To(Not(HaveOccurred()))
	By("By creating a Keycloak secret")
	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "keycloak-auth-secret",
			Namespace: ns,
		},
		Data: map[string][]byte{
			"username": []byte("admin"),
			"password": []byte("admin"),
		},
	}
	Expect(k8sClient.Create(ctx, secret)).Should(Succeed())
	By("By creating a Keycloak")
	keycloak := &keycloakApi.Keycloak{
		ObjectMeta: metav1.ObjectMeta{
			Name:      KeycloakCR,
			Namespace: ns,
		},
		Spec: keycloakApi.KeycloakSpec{
			Url:    os.Getenv("TEST_KEYCLOAK_URL"),
			Secret: secret.Name,
		},
	}
	Expect(k8sClient.Create(ctx, keycloak)).Should(Succeed())
	Eventually(func() bool {
		createdKeycloak := &keycloakApi.Keycloak{}
		err := k8sClient.Get(ctx, types.NamespacedName{Name: KeycloakCR, Namespace: ns}, createdKeycloak)
		Expect(err).ShouldNot(HaveOccurred())

		return createdKeycloak.Status.Connected
	}, time.Second*30, interval).Should(BeTrue())
	By("By creating a KeycloakRealm")
	keycloakRealm := &keycloakApi.KeycloakRealm{
		ObjectMeta: metav1.ObjectMeta{
			Name:      KeycloakRealmCR,
			Namespace: ns,
		},
		Spec: keycloakApi.KeycloakRealmSpec{
			RealmName: KeycloakRealmCR,
			KeycloakRef: common.KeycloakRef{
				Name: keycloak.Name,
				Kind: keycloakApi.KeycloakKind,
			},
		},
	}
	Expect(k8sClient.Create(ctx, keycloakRealm)).Should(Succeed())
	Eventually(func() bool {
		createdKeycloakRealm := &keycloakApi.KeycloakRealm{}
		err := k8sClient.Get(ctx, types.NamespacedName{Name: KeycloakRealmCR, Namespace: ns}, createdKeycloakRealm)
		Expect(err).ShouldNot(HaveOccurred())

		return createdKeycloakRealm.Status.Available
	}, timeout, interval).Should(BeTrue())
})

var _ = AfterSuite(func() {
	cancel()
	By("tearing down the test environment")
	err := testEnv.Stop()
	Expect(err).NotTo(HaveOccurred())
})

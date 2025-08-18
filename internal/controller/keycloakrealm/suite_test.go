package keycloakrealm

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
	metricsserver "sigs.k8s.io/controller-runtime/pkg/metrics/server"

	keycloakApi "github.com/epam/edp-keycloak-operator/api/v1"
	"github.com/epam/edp-keycloak-operator/internal/controller/helper"
	"github.com/epam/edp-keycloak-operator/internal/controller/keycloak"
	"github.com/epam/edp-keycloak-operator/internal/controller/keycloakclient"
	"github.com/epam/edp-keycloak-operator/pkg/testutils"
)

var (
	cfg         *rest.Config
	k8sClient   client.Client
	testEnv     *envtest.Environment
	ctx         context.Context
	cancel      context.CancelFunc
	keycloakURL string
)

const (
	keycloakCR = "test-keycloak"
	ns         = "default"

	timeout  = time.Second * 10
	interval = time.Millisecond * 250
)

func TestKeycloakRealm(t *testing.T) {
	RegisterFailHandler(Fail)

	if os.Getenv("TEST_KEYCLOAK_URL") == "" {
		t.Skip("TEST_KEYCLOAK_URL is not set")
	}

	RunSpecs(t, "KeycloakRealm Controller Suite")
}

var _ = BeforeSuite(func() {
	logf.SetLogger(zap.New(zap.WriteTo(GinkgoWriter), zap.UseDevMode(true)))

	ctx, cancel = context.WithCancel(context.Background())
	ctx = ctrl.LoggerInto(ctx, logf.Log)

	By("bootstrapping test environment")
	testEnv = &envtest.Environment{
		CRDDirectoryPaths:     []string{filepath.Join("..", "..", "..", "config", "crd", "bases")},
		ErrorIfCRDPathMissing: true,
		BinaryAssetsDirectory: testutils.GetFirstFoundEnvTestBinaryDir(),
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
		Scheme: scheme,
		Metrics: metricsserver.Options{
			BindAddress: "0",
		},
	})
	Expect(err).ToNot(HaveOccurred())

	h := helper.MakeHelper(k8sManager.GetClient(), k8sManager.GetScheme(), ns)

	err = keycloak.NewReconcileKeycloak(k8sManager.GetClient(), k8sManager.GetScheme(), h).
		SetupWithManager(k8sManager, time.Second)
	Expect(err).ToNot(HaveOccurred())

	err = NewReconcileKeycloakRealm(k8sManager.GetClient(), k8sManager.GetScheme(), h).
		SetupWithManager(k8sManager, time.Second)
	Expect(err).ToNot(HaveOccurred())

	err = keycloakclient.NewReconcileKeycloakClient(k8sManager.GetClient(), h).
		SetupWithManager(k8sManager, time.Second)

	keycloakURL = os.Getenv("TEST_KEYCLOAK_URL")

	go func() {
		defer GinkgoRecover()
		err = k8sManager.Start(ctx)
		Expect(err).ToNot(HaveOccurred(), "failed to run manager")
	}()

	By("bootstrapping Keycloak")

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
	By("By creating Keycloak")
	keycloak := &keycloakApi.Keycloak{
		ObjectMeta: metav1.ObjectMeta{
			Name:      keycloakCR,
			Namespace: ns,
		},
		Spec: keycloakApi.KeycloakSpec{
			Url:    keycloakURL,
			Secret: secret.Name,
		},
	}
	Expect(k8sClient.Create(ctx, keycloak)).Should(Succeed())
	Eventually(func() bool {
		createdKeycloak := &keycloakApi.Keycloak{}
		err := k8sClient.Get(ctx, types.NamespacedName{Name: keycloakCR, Namespace: ns}, createdKeycloak)
		Expect(err).ShouldNot(HaveOccurred())

		return createdKeycloak.Status.Connected
	}, time.Second*30, interval).Should(BeTrue())
})

var _ = AfterSuite(func() {
	cancel()
	By("tearing down the test environment")
	err := testEnv.Stop()
	Expect(err).NotTo(HaveOccurred())
})

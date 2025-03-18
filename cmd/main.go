package main

import (
	"crypto/tls"
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

	// Import all Kubernetes client auth plugins (e.g. Azure, GCP, OIDC, etc.)
	// to ensure that exec-entrypoint and run can make use of them.
	_ "k8s.io/client-go/plugin/pkg/client/auth"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/cache"
	"sigs.k8s.io/controller-runtime/pkg/healthz"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	"sigs.k8s.io/controller-runtime/pkg/metrics/filters"
	metricsserver "sigs.k8s.io/controller-runtime/pkg/metrics/server"
	ctrlwebhook "sigs.k8s.io/controller-runtime/pkg/webhook"

	buildInfo "github.com/epam/edp-common/pkg/config"

	keycloakApi "github.com/epam/edp-keycloak-operator/api/v1"
	keycloakApi1alpha1 "github.com/epam/edp-keycloak-operator/api/v1alpha1"
	"github.com/epam/edp-keycloak-operator/internal/controller/clusterkeycloak"
	"github.com/epam/edp-keycloak-operator/internal/controller/clusterkeycloakrealm"
	"github.com/epam/edp-keycloak-operator/internal/controller/helper"
	"github.com/epam/edp-keycloak-operator/internal/controller/keycloak"
	"github.com/epam/edp-keycloak-operator/internal/controller/keycloakauthflow"
	"github.com/epam/edp-keycloak-operator/internal/controller/keycloakclient"
	"github.com/epam/edp-keycloak-operator/internal/controller/keycloakclientscope"
	"github.com/epam/edp-keycloak-operator/internal/controller/keycloakrealm"
	"github.com/epam/edp-keycloak-operator/internal/controller/keycloakrealmcomponent"
	"github.com/epam/edp-keycloak-operator/internal/controller/keycloakrealmgroup"
	"github.com/epam/edp-keycloak-operator/internal/controller/keycloakrealmidentityprovider"
	"github.com/epam/edp-keycloak-operator/internal/controller/keycloakrealmrole"
	"github.com/epam/edp-keycloak-operator/internal/controller/keycloakrealmrolebatch"
	"github.com/epam/edp-keycloak-operator/internal/controller/keycloakrealmuser"
	"github.com/epam/edp-keycloak-operator/pkg/secretref"
	"github.com/epam/edp-keycloak-operator/pkg/util"
)

var (
	scheme   = runtime.NewScheme()
	setupLog = ctrl.Log.WithName("setup")
)

const (
	keycloakOperatorLock    = "edp-keycloak-operator-lock"
	successReconcileTimeout = "SUCCESS_RECONCILE_TIMEOUT"
	operatorNamespaceEnv    = "OPERATOR_NAMESPACE"
	managerPort             = 9443
)

func main() {
	var (
		metricsAddr          string
		probeAddr            string
		enableLeaderElection bool
		secureMetrics        bool
		enableHTTP2          bool
		tlsOpts              []func(*tls.Config)
	)

	flag.StringVar(&metricsAddr, "metrics-bind-address", "0", "The address the metrics endpoint binds to. "+
		"Use :8443 for HTTPS or :8080 for HTTP, or leave as 0 to disable the metrics service.")
	flag.StringVar(&probeAddr, "health-probe-bind-address", ":8081", "The address the probe endpoint binds to.")
	flag.BoolVar(&enableLeaderElection, "leader-elect", false,
		"Enable leader election for controller manager. "+
			"Enabling this will ensure there is only one active controller manager.")
	flag.BoolVar(&secureMetrics, "metrics-secure", true,
		"If set, the metrics endpoint is served securely via HTTPS. Use --metrics-secure=false to use HTTP instead.")
	flag.BoolVar(&enableHTTP2, "enable-http2", false,
		"If set, HTTP/2 will be enabled for the metrics and webhook servers")

	opts := zap.Options{
		Development: true,
	}

	opts.BindFlags(flag.CommandLine)
	flag.Parse()

	ctrl.SetLogger(zap.New(zap.UseFlagOptions(&opts)))

	v := buildInfo.Get()

	setupLog.Info("Starting the Keycloak Operator",
		"version", v.Version,
		"git-commit", v.GitCommit,
		"git-tag", v.GitTag,
		"build-date", v.BuildDate,
		"go-version", v.Go,
		"go-client", v.KubectlVersion,
		"platform", v.Platform,
	)
	// if the enable-http2 flag is false (the default), http/2 should be disabled
	// due to its vulnerabilities. More specifically, disabling http/2 will
	// prevent from being vulnerable to the HTTP/2 Stream Cancellation and
	// Rapid Reset CVEs. For more information see:
	// - https://github.com/advisories/GHSA-qppj-fm5r-hxr3
	// - https://github.com/advisories/GHSA-4374-p667-p6c8
	disableHTTP2 := func(c *tls.Config) {
		setupLog.Info("disabling http/2")

		c.NextProtos = []string{"http/1.1"}
	}

	if !enableHTTP2 {
		tlsOpts = append(tlsOpts, disableHTTP2)
	}

	webhookServer := ctrlwebhook.NewServer(ctrlwebhook.Options{
		TLSOpts: tlsOpts,
	})

	// Metrics endpoint is enabled in 'config/default/kustomization.yaml'. The Metrics options configure the server.
	// More info:
	// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.19.0/pkg/metrics/server
	// - https://book.kubebuilder.io/reference/metrics.html
	metricsServerOptions := metricsserver.Options{
		BindAddress:   metricsAddr,
		SecureServing: secureMetrics,
		// TLSOpts is used to allow configuring the TLS config used for the server. If certificates are
		// not provided, self-signed certificates will be generated by default. This option is not recommended for
		// production environments as self-signed certificates do not offer the same level of trust and security
		// as certificates issued by a trusted Certificate Authority (CA). The primary risk is potentially allowing
		// unauthorized access to sensitive metrics data. Consider replacing with CertDir, CertName, and KeyName
		// to provide certificates, ensuring the server communicates using trusted and secure certificates.
		TLSOpts: tlsOpts,
	}

	if secureMetrics {
		// FilterProvider is used to protect the metrics endpoint with authn/authz.
		// These configurations ensure that only authorized users and service accounts
		// can access the metrics endpoint. The RBAC are configured in 'config/rbac/kustomization.yaml'. More info:
		// https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.19.0/pkg/metrics/filters#WithAuthenticationAndAuthorization
		metricsServerOptions.FilterProvider = filters.WithAuthenticationAndAuthorization
	}

	utilruntime.Must(clientgoscheme.AddToScheme(scheme))
	utilruntime.Must(keycloakApi.AddToScheme(scheme))
	utilruntime.Must(keycloakApi1alpha1.AddToScheme(scheme))
	utilruntime.Must(corev1.AddToScheme(scheme))

	ns, err := util.GetWatchNamespace()
	if err != nil {
		setupLog.Error(err, "unable to get watch namespace")
		os.Exit(1)
	}

	cfg := ctrl.GetConfigOrDie()

	mgr, err := ctrl.NewManager(cfg, ctrl.Options{
		Scheme:                 scheme,
		Metrics:                metricsServerOptions,
		WebhookServer:          webhookServer,
		HealthProbeBindAddress: probeAddr,
		LeaderElection:         enableLeaderElection,
		LeaderElectionID:       keycloakOperatorLock,
		Cache: cache.Options{
			DefaultNamespaces: map[string]cache.Config{ns: cache.Config{}},
		},
	})
	if err != nil {
		setupLog.Error(err, "unable to start manager")
		os.Exit(1)
	}

	successReconcileTimeoutValue, err := getSuccessReconcileTimeout()
	if err != nil {
		setupLog.Error(err, "unable to parse reconcile timeout")
		os.Exit(1)
	}

	operatorNamespace, err := getOperatorNamespace()
	if err != nil {
		setupLog.Error(err, "unable to get operator namespace")
		os.Exit(1)
	}

	h := helper.MakeHelper(mgr.GetClient(), mgr.GetScheme(), operatorNamespace)

	keycloakCtrl := keycloak.NewReconcileKeycloak(mgr.GetClient(), mgr.GetScheme(), h)
	if err = keycloakCtrl.SetupWithManager(mgr, successReconcileTimeoutValue); err != nil {
		setupLog.Error(err, "unable to create keycloak controller")
		os.Exit(1)
	}

	keycloakClientCtrl := keycloakclient.NewReconcileKeycloakClient(mgr.GetClient(), h)
	if err = keycloakClientCtrl.SetupWithManager(mgr, successReconcileTimeoutValue); err != nil {
		setupLog.Error(err, "unable to create keycloak-client controller")
		os.Exit(1)
	}

	keycloakRealmCtrl := keycloakrealm.NewReconcileKeycloakRealm(mgr.GetClient(), mgr.GetScheme(), h)
	if err = keycloakRealmCtrl.SetupWithManager(mgr, successReconcileTimeoutValue); err != nil {
		setupLog.Error(err, "unable to create keycloak-realm controller")
		os.Exit(1)
	}

	krgCtrl := keycloakrealmgroup.NewReconcileKeycloakRealmGroup(mgr.GetClient(), h)
	if err = krgCtrl.SetupWithManager(mgr, successReconcileTimeoutValue); err != nil {
		setupLog.Error(err, "unable to create keycloak-realm-group controller")
		os.Exit(1)
	}

	krrCtrl := keycloakrealmrole.NewReconcileKeycloakRealmRole(mgr.GetClient(), h)
	if err = krrCtrl.SetupWithManager(mgr, successReconcileTimeoutValue); err != nil {
		setupLog.Error(err, "unable to create keycloak-realm-role controller")
		os.Exit(1)
	}

	krrbCtrl := keycloakrealmrolebatch.NewReconcileKeycloakRealmRoleBatch(mgr.GetClient(), h)
	if err = krrbCtrl.SetupWithManager(mgr, successReconcileTimeoutValue); err != nil {
		setupLog.Error(err, "unable to create keycloak-realm-role-batch controller")
		os.Exit(1)
	}

	kafCtrl := keycloakauthflow.NewReconcile(mgr.GetClient(), h)
	if err = kafCtrl.SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create keycloak-auth-flow controller")
		os.Exit(1)
	}

	kruCtrl := keycloakrealmuser.NewReconcile(mgr.GetClient(), h)
	if err = kruCtrl.SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create keycloak-realm-user controller")
		os.Exit(1)
	}

	if err = keycloakclientscope.NewReconcile(mgr.GetClient(), h).
		SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create keycloak-client-scope controller")
		os.Exit(1)
	}

	if err = keycloakrealmcomponent.NewReconcile(mgr.GetClient(), mgr.GetScheme(), h, secretref.NewSecretRef(mgr.GetClient())).
		SetupWithManager(mgr, successReconcileTimeoutValue); err != nil {
		setupLog.Error(err, "unable to create keycloak-realm-component controller")
		os.Exit(1)
	}

	if err = keycloakrealmidentityprovider.NewReconcile(mgr.GetClient(), h, secretref.NewSecretRef(mgr.GetClient())).
		SetupWithManager(mgr, successReconcileTimeoutValue); err != nil {
		setupLog.Error(err, "unable to create keycloak-realm-identity-provider controller")
		os.Exit(1)
	}

	if ns == "" {
		if err = clusterkeycloak.NewReconcile(mgr.GetClient(), mgr.GetScheme(), h, operatorNamespace).
			SetupWithManager(mgr); err != nil {
			setupLog.Error(err, "unable to create clusterkeycloak controller")
			os.Exit(1)
		}

		if err = clusterkeycloakrealm.NewClusterKeycloakRealmReconciler(mgr.GetClient(), mgr.GetScheme(), h, operatorNamespace).
			SetupWithManager(mgr); err != nil {
			setupLog.Error(err, "unable to create controller", "controller", "ClusterKeycloakRealm")
			os.Exit(1)
		}
	}

	//+kubebuilder:scaffold:builder

	if err := mgr.AddHealthzCheck("healthz", healthz.Ping); err != nil {
		setupLog.Error(err, "unable to set up health check")
		os.Exit(1)
	}

	if err := mgr.AddReadyzCheck("readyz", healthz.Ping); err != nil {
		setupLog.Error(err, "unable to set up ready check")
		os.Exit(1)
	}

	setupLog.Info("starting manager")

	if err := mgr.Start(ctrl.SetupSignalHandler()); err != nil {
		setupLog.Error(err, "problem running manager")
		os.Exit(1)
	}
}

func getSuccessReconcileTimeout() (time.Duration, error) {
	val, exists := os.LookupEnv(successReconcileTimeout)
	if !exists {
		return 0, nil
	}

	d, err := time.ParseDuration(val)
	if err != nil {
		return 0, fmt.Errorf("wrong reconcile timeout duration format: %w", err)
	}

	return d, nil
}

func getOperatorNamespace() (string, error) {
	ns, exists := os.LookupEnv(operatorNamespaceEnv)
	if !exists {
		return "", fmt.Errorf("environment variable %s is not set", operatorNamespaceEnv)
	}

	if ns = strings.TrimSpace(ns); ns == "" {
		return "", fmt.Errorf("environment variable %s is empty", operatorNamespaceEnv)
	}

	return ns, nil
}

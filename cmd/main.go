package main

import (
	"crypto/tls"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
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
	"sigs.k8s.io/controller-runtime/pkg/certwatcher"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/healthz"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	"sigs.k8s.io/controller-runtime/pkg/metrics/filters"
	metricsserver "sigs.k8s.io/controller-runtime/pkg/metrics/server"
	"sigs.k8s.io/controller-runtime/pkg/webhook"

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
	"github.com/epam/edp-keycloak-operator/internal/controller/keycloakorganization"
	"github.com/epam/edp-keycloak-operator/internal/controller/keycloakrealm"
	"github.com/epam/edp-keycloak-operator/internal/controller/keycloakrealmcomponent"
	"github.com/epam/edp-keycloak-operator/internal/controller/keycloakrealmgroup"
	"github.com/epam/edp-keycloak-operator/internal/controller/keycloakrealmidentityprovider"
	"github.com/epam/edp-keycloak-operator/internal/controller/keycloakrealmrole"
	"github.com/epam/edp-keycloak-operator/internal/controller/keycloakrealmrolebatch"
	"github.com/epam/edp-keycloak-operator/internal/controller/keycloakrealmuser"
	webhookv1 "github.com/epam/edp-keycloak-operator/internal/webhook/v1"
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
)

func init() {
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))
	utilruntime.Must(keycloakApi.AddToScheme(scheme))
	utilruntime.Must(keycloakApi1alpha1.AddToScheme(scheme))
	utilruntime.Must(corev1.AddToScheme(scheme))
	// +kubebuilder:scaffold:scheme
}

// nolint:gocyclo
func main() {
	var (
		metricsAddr                                      string
		metricsCertPath, metricsCertName, metricsCertKey string
		webhookCertPath, webhookCertName, webhookCertKey string
		enableLeaderElection                             bool
		probeAddr                                        string
		secureMetrics                                    bool
		enableHTTP2                                      bool
		tlsOpts                                          []func(*tls.Config)
	)

	flag.StringVar(&metricsAddr, "metrics-bind-address", "0", "The address the metrics endpoint binds to. "+
		"Use :8443 for HTTPS or :8080 for HTTP, or leave as 0 to disable the metrics service.")
	flag.StringVar(&probeAddr, "health-probe-bind-address", ":8081", "The address the probe endpoint binds to.")
	flag.BoolVar(&enableLeaderElection, "leader-elect", false,
		"Enable leader election for controller manager. "+
			"Enabling this will ensure there is only one active controller manager.")
	flag.BoolVar(&secureMetrics, "metrics-secure", true,
		"If set, the metrics endpoint is served securely via HTTPS. Use --metrics-secure=false to use HTTP instead.")
	flag.StringVar(&webhookCertPath, "webhook-cert-path", "", "The directory that contains the webhook certificate.")
	flag.StringVar(&webhookCertName, "webhook-cert-name", "tls.crt", "The name of the webhook certificate file.")
	flag.StringVar(&webhookCertKey, "webhook-cert-key", "tls.key", "The name of the webhook key file.")
	flag.StringVar(&metricsCertPath, "metrics-cert-path", "",
		"The directory that contains the metrics server certificate.")
	flag.StringVar(&metricsCertName, "metrics-cert-name", "tls.crt", "The name of the metrics server certificate file.")
	flag.StringVar(&metricsCertKey, "metrics-cert-key", "tls.key", "The name of the metrics server key file.")
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

	// Create watchers for metrics and webhooks certificates
	var metricsCertWatcher, webhookCertWatcher *certwatcher.CertWatcher

	// Initial webhook TLS options
	webhookTLSOpts := tlsOpts

	if len(webhookCertPath) > 0 {
		setupLog.Info("Initializing webhook certificate watcher using provided certificates",
			"webhook-cert-path", webhookCertPath, "webhook-cert-name", webhookCertName, "webhook-cert-key", webhookCertKey)

		var err error

		webhookCertWatcher, err = certwatcher.New(
			filepath.Join(webhookCertPath, webhookCertName),
			filepath.Join(webhookCertPath, webhookCertKey),
		)
		if err != nil {
			setupLog.Error(err, "Failed to initialize webhook certificate watcher")
			os.Exit(1)
		}

		webhookTLSOpts = append(webhookTLSOpts, func(config *tls.Config) {
			config.GetCertificate = webhookCertWatcher.GetCertificate
		})
	}

	webhookServer := webhook.NewServer(webhook.Options{
		TLSOpts: webhookTLSOpts,
	})

	// Metrics endpoint is enabled in 'config/default/kustomization.yaml'. The Metrics options configure the server.
	// More info:
	// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.21.0/pkg/metrics/server
	// - https://book.kubebuilder.io/reference/metrics.html
	metricsServerOptions := metricsserver.Options{
		BindAddress:   metricsAddr,
		SecureServing: secureMetrics,
		TLSOpts:       tlsOpts,
	}

	if secureMetrics {
		// FilterProvider is used to protect the metrics endpoint with authn/authz.
		// These configurations ensure that only authorized users and service accounts
		// can access the metrics endpoint. The RBAC are configured in 'config/rbac/kustomization.yaml'. More info:
		// https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.21.0/pkg/metrics/filters#WithAuthenticationAndAuthorization
		metricsServerOptions.FilterProvider = filters.WithAuthenticationAndAuthorization
	}

	// If the certificate is not specified, controller-runtime will automatically
	// generate self-signed certificates for the metrics server. While convenient for development and testing,
	// this setup is not recommended for production.
	//
	// TODO(user): If you enable certManager, uncomment the following lines:
	// - [METRICS-WITH-CERTS] at config/default/kustomization.yaml to generate and use certificates
	// managed by cert-manager for the metrics server.
	// - [PROMETHEUS-WITH-CERTS] at config/prometheus/kustomization.yaml for TLS certification.
	if len(metricsCertPath) > 0 {
		setupLog.Info("Initializing metrics certificate watcher using provided certificates",
			"metrics-cert-path", metricsCertPath, "metrics-cert-name", metricsCertName, "metrics-cert-key", metricsCertKey)

		var err error

		metricsCertWatcher, err = certwatcher.New(
			filepath.Join(metricsCertPath, metricsCertName),
			filepath.Join(metricsCertPath, metricsCertKey),
		)
		if err != nil {
			setupLog.Error(err, "to initialize metrics certificate watcher", "error", err)
			os.Exit(1)
		}

		metricsServerOptions.TLSOpts = append(metricsServerOptions.TLSOpts, func(config *tls.Config) {
			config.GetCertificate = metricsCertWatcher.GetCertificate
		})
	}

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
			DefaultNamespaces: map[string]cache.Config{ns: {}},
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

	h := helper.MakeHelper(mgr.GetClient(), mgr.GetScheme(), operatorNamespace, helper.EnableOwnerRef(enableOwnerRef()))

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

	if err = keycloakrealmcomponent.NewReconcile(
		mgr.GetClient(),
		mgr.GetScheme(),
		h,
		secretref.NewSecretRef(mgr.GetClient()),
	).
		SetupWithManager(mgr, successReconcileTimeoutValue); err != nil {
		setupLog.Error(err, "unable to create keycloak-realm-component controller")
		os.Exit(1)
	}

	if err = keycloakrealmidentityprovider.NewReconcile(mgr.GetClient(), h).
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

		if err = clusterkeycloakrealm.NewClusterKeycloakRealmReconciler(
			mgr.GetClient(),
			mgr.GetScheme(),
			h,
			operatorNamespace,
		).
			SetupWithManager(mgr); err != nil {
			setupLog.Error(err, "unable to create controller", "controller", "ClusterKeycloakRealm")
			os.Exit(1)
		}
	}

	organizationCtrl := keycloakorganization.NewReconcileOrganization(mgr.GetClient(), h)
	if err = organizationCtrl.SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create keycloak-organization controller")
		os.Exit(1)
	}

	if os.Getenv("ENABLE_WEBHOOKS") != "false" {
		// Setup k8s client without cache to enable reading from non-default namespaces.
		k8sClient, err := client.New(cfg, client.Options{Scheme: scheme})
		if err != nil {
			setupLog.Error(err, "unable to create k8s client for webhook setup")
			os.Exit(1)
		}

		if err := webhookv1.SetupKeycloakRealmWebhookWithManager(mgr, k8sClient); err != nil {
			setupLog.Error(err, "unable to create webhook", "webhook", "KeycloakRealm")
			os.Exit(1)
		}

		if err := webhookv1.SetupKeycloakClientWebhookWithManager(mgr); err != nil {
			setupLog.Error(err, "unable to create webhook", "webhook", "KeycloakClient")
			os.Exit(1)
		}
	}
	// +kubebuilder:scaffold:builder

	if metricsCertWatcher != nil {
		setupLog.Info("Adding metrics certificate watcher to manager")

		if err := mgr.Add(metricsCertWatcher); err != nil {
			setupLog.Error(err, "Unable to add metrics certificate watcher to manager")
			os.Exit(1)
		}
	}

	if webhookCertWatcher != nil {
		setupLog.Info("Adding webhook certificate watcher to manager")

		if err := mgr.Add(webhookCertWatcher); err != nil {
			setupLog.Error(err, "Unable to add webhook certificate watcher to manager")
			os.Exit(1)
		}
	}

	if err := mgr.AddHealthzCheck("healthz", healthz.Ping); err != nil {
		setupLog.Error(err, "unable to set up health check")
		os.Exit(1)
	}

	if err := mgr.AddReadyzCheck("readyz", healthz.Ping); err != nil {
		setupLog.Error(err, "unable to set up ready check")
		os.Exit(1)
	}

	setupLog.Info("Starting manager")

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

func enableOwnerRef() bool {
	val, exists := os.LookupEnv("ENABLE_OWNER_REF")
	if !exists {
		return false
	}

	b, err := strconv.ParseBool(val)
	if err != nil {
		setupLog.Error(err, "unable to parse ENABLE_OWNER_REF. Using default value false")
		return false
	}

	return b
}

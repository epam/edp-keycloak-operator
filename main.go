package main

import (
	"flag"
	"fmt"
	"os"
	"time"

	// Import all Kubernetes client auth plugins (e.g. Azure, GCP, OIDC, etc.)
	// to ensure that exec-entrypoint and run can make use of them.
	_ "k8s.io/client-go/plugin/pkg/client/auth"

	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/apiutil"
	"sigs.k8s.io/controller-runtime/pkg/healthz"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"

	buildInfo "github.com/epam/edp-common/pkg/config"

	keycloakApi "github.com/epam/edp-keycloak-operator/api/v1/v1"
	keycloakApi1alpha1 "github.com/epam/edp-keycloak-operator/api/v1/v1alpha1"
	"github.com/epam/edp-keycloak-operator/controllers/helper"
	"github.com/epam/edp-keycloak-operator/controllers/keycloak"
	"github.com/epam/edp-keycloak-operator/controllers/keycloakauthflow"
	"github.com/epam/edp-keycloak-operator/controllers/keycloakclient"
	"github.com/epam/edp-keycloak-operator/controllers/keycloakclientscope"
	"github.com/epam/edp-keycloak-operator/controllers/keycloakrealm"
	"github.com/epam/edp-keycloak-operator/controllers/keycloakrealmcomponent"
	"github.com/epam/edp-keycloak-operator/controllers/keycloakrealmgroup"
	"github.com/epam/edp-keycloak-operator/controllers/keycloakrealmidentityprovider"
	"github.com/epam/edp-keycloak-operator/controllers/keycloakrealmrole"
	"github.com/epam/edp-keycloak-operator/controllers/keycloakrealmrolebatch"
	"github.com/epam/edp-keycloak-operator/controllers/keycloakrealmuser"
	"github.com/epam/edp-keycloak-operator/pkg/util"
)

var (
	scheme   = runtime.NewScheme()
	setupLog = ctrl.Log.WithName("setup")
)

const (
	keycloakOperatorLock    = "edp-keycloak-operator-lock"
	successReconcileTimeout = "SUCCESS_RECONCILE_TIMEOUT"
	managerPort             = 9443
)

func main() {
	var (
		metricsAddr          string
		probeAddr            string
		enableLeaderElection bool
	)

	flag.StringVar(&metricsAddr, "metrics-bind-address", ":8090", "The address the metric endpoint binds to.")
	flag.StringVar(&probeAddr, "health-probe-bind-address", ":8091", "The address the probe endpoint binds to.")
	flag.BoolVar(&enableLeaderElection, "leader-elect", false,
		"Enable leader election for controller manager. "+
			"Enabling this will ensure there is only one active controller manager.")

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

	utilruntime.Must(clientgoscheme.AddToScheme(scheme))
	utilruntime.Must(keycloakApi.AddToScheme(scheme))
	utilruntime.Must(keycloakApi1alpha1.AddToScheme(scheme))

	ns, err := util.GetWatchNamespace()
	if err != nil {
		setupLog.Error(err, "unable to get watch namespace")
		os.Exit(1)
	}

	cfg := ctrl.GetConfigOrDie()

	mgr, err := ctrl.NewManager(cfg, ctrl.Options{
		Scheme:                 scheme,
		MetricsBindAddress:     metricsAddr,
		HealthProbeBindAddress: probeAddr,
		Port:                   managerPort,
		LeaderElection:         enableLeaderElection,
		LeaderElectionID:       keycloakOperatorLock,
		MapperProvider: func(c *rest.Config) (meta.RESTMapper, error) {
			return apiutil.NewDynamicRESTMapper(cfg)
		},
		Namespace: ns,
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

	ctrlLog := ctrl.Log.WithName("controllers")
	h := helper.MakeHelper(mgr.GetClient(), mgr.GetScheme(), ctrlLog)

	keycloakCtrl := keycloak.NewReconcileKeycloak(mgr.GetClient(), mgr.GetScheme(), ctrlLog, h)
	if err := keycloakCtrl.SetupWithManager(mgr, successReconcileTimeoutValue); err != nil {
		setupLog.Error(err, "unable to create keycloak controller")
		os.Exit(1)
	}

	keycloakClientCtrl := keycloakclient.NewReconcileKeycloakClient(mgr.GetClient(), ctrlLog, h)
	if err := keycloakClientCtrl.SetupWithManager(mgr, successReconcileTimeoutValue); err != nil {
		setupLog.Error(err, "unable to create keycloak-client controller")
		os.Exit(1)
	}

	keycloakRealmCtrl := keycloakrealm.NewReconcileKeycloakRealm(mgr.GetClient(), mgr.GetScheme(), ctrlLog, h)
	if err := keycloakRealmCtrl.SetupWithManager(mgr, successReconcileTimeoutValue); err != nil {
		setupLog.Error(err, "unable to create keycloak-realm controller")
		os.Exit(1)
	}

	krgCtrl := keycloakrealmgroup.NewReconcileKeycloakRealmGroup(mgr.GetClient(), ctrlLog, h)
	if err := krgCtrl.SetupWithManager(mgr, successReconcileTimeoutValue); err != nil {
		setupLog.Error(err, "unable to create keycloak-realm-group controller")
		os.Exit(1)
	}

	krrCtrl := keycloakrealmrole.NewReconcileKeycloakRealmRole(mgr.GetClient(), ctrlLog, h)
	if err := krrCtrl.SetupWithManager(mgr, successReconcileTimeoutValue); err != nil {
		setupLog.Error(err, "unable to create keycloak-realm-role controller")
		os.Exit(1)
	}

	krrbCtrl := keycloakrealmrolebatch.NewReconcileKeycloakRealmRoleBatch(mgr.GetClient(), ctrlLog, h)
	if err := krrbCtrl.SetupWithManager(mgr, successReconcileTimeoutValue); err != nil {
		setupLog.Error(err, "unable to create keycloak-realm-role-batch controller")
		os.Exit(1)
	}

	kafCtrl := keycloakauthflow.NewReconcile(mgr.GetClient(), ctrlLog, h)
	if err := kafCtrl.SetupWithManager(mgr, successReconcileTimeoutValue); err != nil {
		setupLog.Error(err, "unable to create keycloak-auth-flow controller")
		os.Exit(1)
	}

	kruCtrl := keycloakrealmuser.NewReconcile(mgr.GetClient(), ctrlLog, h)
	if err := kruCtrl.SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create keycloak-realm-user controller")
		os.Exit(1)
	}

	if err := keycloakclientscope.NewReconcile(mgr.GetClient(), ctrlLog, h).
		SetupWithManager(mgr, successReconcileTimeoutValue); err != nil {
		setupLog.Error(err, "unable to create keycloak-client-scope controller")
		os.Exit(1)
	}

	if err := keycloakrealmcomponent.NewReconcile(mgr.GetClient(), ctrlLog, h).
		SetupWithManager(mgr, successReconcileTimeoutValue); err != nil {
		setupLog.Error(err, "unable to create keycloak-realm-component controller")
		os.Exit(1)
	}

	if err := keycloakrealmidentityprovider.NewReconcile(mgr.GetClient(), ctrlLog, h).
		SetupWithManager(mgr, successReconcileTimeoutValue); err != nil {
		setupLog.Error(err, "unable to create keycloak-realm-identity-provider controller")
		os.Exit(1)
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

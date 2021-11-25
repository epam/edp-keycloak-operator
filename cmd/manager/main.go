package main

import (
	"flag"
	"os"
	"time"

	buildInfo "github.com/epam/edp-common/pkg/config"
	edpCompApi "github.com/epam/edp-component-operator/pkg/apis/v1/v1alpha1"
	keycloakApi "github.com/epam/edp-keycloak-operator/pkg/apis/v1/v1alpha1"
	"github.com/epam/edp-keycloak-operator/pkg/controller/helper"
	"github.com/epam/edp-keycloak-operator/pkg/controller/keycloak"
	"github.com/epam/edp-keycloak-operator/pkg/controller/keycloakauthflow"
	"github.com/epam/edp-keycloak-operator/pkg/controller/keycloakclient"
	"github.com/epam/edp-keycloak-operator/pkg/controller/keycloakclientscope"
	"github.com/epam/edp-keycloak-operator/pkg/controller/keycloakrealm"
	"github.com/epam/edp-keycloak-operator/pkg/controller/keycloakrealmcomponent"
	"github.com/epam/edp-keycloak-operator/pkg/controller/keycloakrealmgroup"
	"github.com/epam/edp-keycloak-operator/pkg/controller/keycloakrealmrole"
	"github.com/epam/edp-keycloak-operator/pkg/controller/keycloakrealmrolebatch"
	"github.com/epam/edp-keycloak-operator/pkg/controller/keycloakrealmuser"
	"github.com/epam/edp-keycloak-operator/pkg/util"
	"github.com/pkg/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/client-go/rest"

	// Import all Kubernetes client auth plugins (e.g. Azure, GCP, OIDC, etc.)
	// to ensure that exec-entrypoint and run can make use of them.
	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	_ "k8s.io/client-go/plugin/pkg/client/auth"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/healthz"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"

	//+kubebuilder:scaffold:imports
	"sigs.k8s.io/controller-runtime/pkg/client/apiutil"
)

var (
	scheme   = runtime.NewScheme()
	setupLog = ctrl.Log.WithName("setup")
)

const (
	keycloakOperatorLock    = "edp-keycloak-operator-lock"
	successReconcileTimeout = "SUCCESS_RECONCILE_TIMEOUT"
)

func init() {
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))
	utilruntime.Must(keycloakApi.AddToScheme(scheme))
	utilruntime.Must(edpCompApi.AddToScheme(scheme))
}

func main() {
	var (
		metricsAddr          string
		enableLeaderElection bool
		probeAddr            string
	)

	flag.StringVar(&metricsAddr, "metrics-bind-address", ":8080", "The address the metric endpoint binds to.")
	flag.StringVar(&probeAddr, "health-probe-bind-address", ":8081", "The address the probe endpoint binds to.")
	flag.BoolVar(&enableLeaderElection, "leader-elect", util.RunningInCluster(),
		"Enable leader election for controller manager. "+
			"Enabling this will ensure there is only one active controller manager.")

	mode, err := util.GetDebugMode()
	if err != nil {
		setupLog.Error(err, "unable to get debug mode value")
		os.Exit(1)
	}

	opts := zap.Options{
		Development: mode,
	}
	opts.BindFlags(flag.CommandLine)
	flag.Parse()

	v := buildInfo.Get()

	ctrl.SetLogger(zap.New(zap.UseFlagOptions(&opts)))

	setupLog.Info("Starting the Keycloak Operator",
		"version", v.Version,
		"git-commit", v.GitCommit,
		"git-tag", v.GitTag,
		"build-date", v.BuildDate,
		"go-version", v.Go,
		"go-client", v.KubectlVersion,
		"platform", v.Platform,
	)

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
		Port:                   9443,
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
		setupLog.Error(err, "unable to create controller", "controller", "keycloak")
		os.Exit(1)
	}

	keycloakClientCtrl := keycloakclient.NewReconcileKeycloakClient(mgr.GetClient(), ctrlLog, h)
	if err := keycloakClientCtrl.SetupWithManager(mgr, successReconcileTimeoutValue); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "keycloak-client")
		os.Exit(1)
	}

	keycloakRealmCtrl := keycloakrealm.NewReconcileKeycloakRealm(mgr.GetClient(), mgr.GetScheme(), ctrlLog, h)
	if err := keycloakRealmCtrl.SetupWithManager(mgr, successReconcileTimeoutValue); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "keycloak-realm")
		os.Exit(1)
	}

	krgCtrl := keycloakrealmgroup.NewReconcileKeycloakRealmGroup(mgr.GetClient(), ctrlLog, h)
	if err := krgCtrl.SetupWithManager(mgr, successReconcileTimeoutValue); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "keycloak-realm-group")
		os.Exit(1)
	}

	krrCtrl := keycloakrealmrole.NewReconcileKeycloakRealmRole(mgr.GetClient(), ctrlLog, h)
	if err := krrCtrl.SetupWithManager(mgr, successReconcileTimeoutValue); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "keycloak-realm-role")
		os.Exit(1)
	}

	krrbCtrl := keycloakrealmrolebatch.NewReconcileKeycloakRealmRoleBatch(mgr.GetClient(), ctrlLog, h)
	if err := krrbCtrl.SetupWithManager(mgr, successReconcileTimeoutValue); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "keycloak-realm-role-batch")
		os.Exit(1)
	}

	kafCtrl := keycloakauthflow.NewReconcile(mgr.GetClient(), ctrlLog, h)
	if err := kafCtrl.SetupWithManager(mgr, successReconcileTimeoutValue); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "keycloak-auth-flow")
		os.Exit(1)
	}

	kruCtrl := keycloakrealmuser.NewReconcile(mgr.GetClient(), ctrlLog, h)
	if err := kruCtrl.SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "keycloak-realm-user")
		os.Exit(1)
	}

	if err := keycloakclientscope.NewReconcile(mgr.GetClient(), ctrlLog, h).
		SetupWithManager(mgr, successReconcileTimeoutValue); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "keycloak-client-scope")
		os.Exit(1)
	}

	if err := keycloakrealmcomponent.NewReconcile(mgr.GetClient(), ctrlLog, h).
		SetupWithManager(mgr, successReconcileTimeoutValue); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "keycloak-realm-component")
		os.Exit(1)
	}

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
		return 0, errors.Wrap(err, "wrong reconcile timeout duration format")
	}

	return d, nil
}

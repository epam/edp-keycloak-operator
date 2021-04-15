package main

import (
	"flag"
	"github.com/epam/keycloak-operator/pkg/apis/v1/v1alpha1"
	"github.com/epam/keycloak-operator/pkg/client/keycloak/adapter"
	"github.com/epam/keycloak-operator/pkg/controller/helper"
	"github.com/epam/keycloak-operator/pkg/controller/keycloak"
	"github.com/epam/keycloak-operator/pkg/controller/keycloakclient"
	"github.com/epam/keycloak-operator/pkg/controller/keycloakrealm"
	"github.com/epam/keycloak-operator/pkg/controller/keycloakrealmgroup"
	"github.com/epam/keycloak-operator/pkg/controller/keycloakrealmrole"
	"github.com/epam/keycloak-operator/pkg/controller/keycloakrealmrolebatch"
	"github.com/epam/keycloak-operator/pkg/util"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/client-go/rest"
	"os"
	// Import all Kubernetes client auth plugins (e.g. Azure, GCP, OIDC, etc.)
	// to ensure that exec-entrypoint and run can make use of them.
	_ "k8s.io/client-go/plugin/pkg/client/auth"

	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
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

const keycloakOperatorLock = "edp-keycloak-operator-lock"

func init() {
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))

	utilruntime.Must(v1alpha1.AddToScheme(scheme))

	//+kubebuilder:scaffold:scheme
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

	ctrl.SetLogger(zap.New(zap.UseFlagOptions(&opts)))

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

	ctrlLog := ctrl.Log.WithName("controllers")

	if err = (&keycloak.ReconcileKeycloak{
		Client: mgr.GetClient(),
		Scheme: mgr.GetScheme(),
		Log:    ctrlLog.WithName("keycloak"),
		Factory: adapter.GoCloakAdapterFactory{
			Log: ctrl.Log.WithName("go-cloak-adapter-factory"),
		},
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "keycloak")
		os.Exit(1)
	}

	h := helper.MakeHelper(mgr.GetClient(), mgr.GetScheme())

	if err = (&keycloakclient.ReconcileKeycloakClient{
		Client: mgr.GetClient(),
		Helper: h,
		Log:    ctrlLog.WithName("keycloak-client"),
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "keycloak-client")
		os.Exit(1)
	}

	if err = (&keycloakrealm.ReconcileKeycloakRealm{
		Client: mgr.GetClient(),
		Scheme: mgr.GetScheme(),
		Factory: adapter.GoCloakAdapterFactory{
			Log: ctrl.Log.WithName("go-cloak-adapter-factory"),
		},
		Helper: h,
		Log:    ctrlLog.WithName("keycloak-realm"),
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "keycloak-realm")
		os.Exit(1)
	}

	if err = (&keycloakrealmgroup.ReconcileKeycloakRealmGroup{
		Client: mgr.GetClient(),
		Scheme: mgr.GetScheme(),
		Factory: adapter.GoCloakAdapterFactory{
			Log: ctrl.Log.WithName("go-cloak-adapter-factory"),
		},
		Helper: h,
		Log:    ctrlLog.WithName("keycloak-realm-group"),
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "keycloak-realm-group")
		os.Exit(1)
	}

	if err = (&keycloakrealmrole.ReconcileKeycloakRealmRole{
		Client: mgr.GetClient(),
		Scheme: mgr.GetScheme(),
		Factory: adapter.GoCloakAdapterFactory{
			Log: ctrl.Log.WithName("go-cloak-adapter-factory"),
		},
		Helper: h,
		Log:    ctrlLog.WithName("keycloak-realm-role"),
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "keycloak-realm-role")
		os.Exit(1)
	}

	if err = (&keycloakrealmrolebatch.ReconcileKeycloakRealmRoleBatch{
		Client: mgr.GetClient(),
		Scheme: mgr.GetScheme(),
		Helper: h,
		Log:    ctrlLog.WithName("keycloak-realm-role-batch"),
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "keycloak-realm-role-batch")
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

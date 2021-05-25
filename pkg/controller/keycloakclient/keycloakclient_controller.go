package keycloakclient

import (
	"context"

	v1v1alpha1 "github.com/epam/edp-keycloak-operator/pkg/apis/v1/v1alpha1"
	"github.com/epam/edp-keycloak-operator/pkg/client/keycloak/adapter"
	"github.com/epam/edp-keycloak-operator/pkg/controller/helper"
	"github.com/epam/edp-keycloak-operator/pkg/controller/keycloakclient/chain"
	pkgErrors "github.com/pkg/errors"
	"k8s.io/apimachinery/pkg/api/errors"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	logf "sigs.k8s.io/controller-runtime/pkg/runtime/log"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

var log = logf.Log.WithName("controller_keycloakclient")

const (
	Ok                                  = "OK"
	Fail                                = "FAIL"
	keyCloakClientOperatorFinalizerName = "keycloak.client.operator.finalizer.name"
)

/**
* USER ACTION REQUIRED: This is a scaffold file intended for the user to modify with their own Controller
* business logic.  Delete these comments after modifying this file.*
 */

// Add creates a new KeycloakClient Controller and adds it to the Manager. The Manager will set fields on the Controller
// and Start it when the Manager is Started.
func Add(mgr manager.Manager) error {
	c, err := controller.New("keycloakclient-controller", mgr, controller.Options{Reconciler: newReconciler(mgr)})
	if err != nil {
		return err
	}

	// Watch for changes to primary resource KeycloakClient
	return c.Watch(&source.Kind{Type: &v1v1alpha1.KeycloakClient{}}, &handler.EnqueueRequestForObject{},
		predicate.Funcs{
			UpdateFunc: helper.IsFailuresUpdated,
		})
}

// newReconciler returns a new reconcile.Reconciler
func newReconciler(mgr manager.Manager) reconcile.Reconciler {
	h := helper.MakeHelper(mgr.GetClient(), mgr.GetScheme())
	return &ReconcileKeycloakClient{
		client: mgr.GetClient(),
		helper: h,
		chain:  chain.Make(h, mgr.GetClient(), log, new(adapter.GoCloakAdapterFactory)),
	}
}

// blank assignment to verify that ReconcileKeycloakClient implements reconcile.Reconciler
var _ reconcile.Reconciler = &ReconcileKeycloakClient{}

// ReconcileKeycloakClient reconciles a KeycloakClient object
type ReconcileKeycloakClient struct {
	// This client, initialized using mgr.Client() above, is a split client
	// that reads objects from the cache and writes to the apiserver
	client client.Client
	helper *helper.Helper
	chain  chain.Element
}

// Reconcile reads that state of the cluster for a KeycloakClient object and makes changes based on the state read
// and what is in the KeycloakClient.Spec
// Note:
// The Controller will requeue the Request to be processed again if the returned error is non-nil or
// Result.Requeue is true, otherwise upon completion it will remove the work from the queue.
func (r *ReconcileKeycloakClient) Reconcile(request reconcile.Request) (result reconcile.Result, resultErr error) {
	reqLogger := log.WithValues("Request.Namespace", request.Namespace, "Request.Name", request.Name)
	reqLogger.Info("Reconciling KeycloakClient")

	// Fetch the KeycloakClient instance
	var instance v1v1alpha1.KeycloakClient
	if err := r.client.Get(context.TODO(), request.NamespacedName, &instance); err != nil {
		if errors.IsNotFound(err) {
			// Request object not found, could have been deleted after reconcile request.
			// Owned objects are automatically garbage collected. For additional cleanup logic use finalizers.
			// Return and don't requeue
			return
		}
		// Error reading the object - requeue the request.
		resultErr = err
		return
	}

	if err := r.tryReconcile(&instance); err != nil {
		instance.Status.Value = Fail
		result.RequeueAfter = r.helper.SetFailureCount(&instance)
		log.Error(err, "an error has occurred while handling keycloak client", "name",
			request.Name)
	} else {
		helper.SetSuccessStatus(&instance)
	}

	if err := r.helper.UpdateStatus(&instance); err != nil {
		resultErr = pkgErrors.Wrap(err, "unable to update status")
	}

	return
}

func (r *ReconcileKeycloakClient) tryReconcile(keycloakClient *v1v1alpha1.KeycloakClient) error {
	if err := r.chain.Serve(keycloakClient); err != nil {
		return pkgErrors.Wrap(err, "error during kc chain")
	}

	if _, err := r.helper.TryToDelete(keycloakClient, makeTerminator(keycloakClient.Status.ClientID,
		keycloakClient.Spec.TargetRealm, r.chain.GetState().AdapterClient),
		keyCloakClientOperatorFinalizerName); err != nil {
		return pkgErrors.Wrap(err, "unable to delete kc client")
	}

	return nil
}

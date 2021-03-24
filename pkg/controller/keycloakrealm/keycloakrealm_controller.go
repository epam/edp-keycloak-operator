package keycloakrealm

import (
	"context"

	v1v1alpha1 "github.com/epmd-edp/keycloak-operator/pkg/apis/v1/v1alpha1"
	"github.com/epmd-edp/keycloak-operator/pkg/client/keycloak"
	"github.com/epmd-edp/keycloak-operator/pkg/client/keycloak/adapter"
	"github.com/epmd-edp/keycloak-operator/pkg/controller/helper"
	"github.com/epmd-edp/keycloak-operator/pkg/controller/keycloakrealm/chain"
	rHand "github.com/epmd-edp/keycloak-operator/pkg/controller/keycloakrealm/chain/handler"
	"github.com/pkg/errors"
	k8sErrors "k8s.io/apimachinery/pkg/api/errors"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	logf "sigs.k8s.io/controller-runtime/pkg/runtime/log"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

const (
	keyCloakRealmOperatorFinalizerName = "keycloak.realm.operator.finalizer.name"
)

var log = logf.Log.WithName("controller_keycloakrealm")

/**
* USER ACTION REQUIRED: This is a scaffold file intended for the user to modify with their own Controller
* business logic.  Delete these comments after modifying this file.*
 */

// Add creates a new KeycloakRealm Controller and adds it to the Manager. The Manager will set fields on the Controller
// and Start it when the Manager is Started.
func Add(mgr manager.Manager) error {
	return add(mgr, newReconciler(mgr))
}

// newReconciler returns a new reconcile.Reconciler
func newReconciler(mgr manager.Manager) reconcile.Reconciler {
	return &ReconcileKeycloakRealm{
		client:  mgr.GetClient(),
		factory: new(adapter.GoCloakAdapterFactory),
		handler: chain.CreateDefChain(mgr.GetClient(), mgr.GetScheme()),
		helper:  helper.MakeHelper(mgr.GetClient(), mgr.GetScheme()),
	}
}

// add adds a new Controller to mgr with r as the reconcile.Reconciler
func add(mgr manager.Manager, r reconcile.Reconciler) error {
	// Create a new controller
	c, err := controller.New("keycloakrealm-controller", mgr, controller.Options{Reconciler: r})
	if err != nil {
		return err
	}

	// Watch for changes to primary resource KeycloakRealm
	return c.Watch(&source.Kind{Type: &v1v1alpha1.KeycloakRealm{}}, &handler.EnqueueRequestForObject{})
}

// blank assignment to verify that ReconcileKeycloakRealm implements reconcile.Reconciler
var _ reconcile.Reconciler = &ReconcileKeycloakRealm{}

// ReconcileKeycloakRealm reconciles a KeycloakRealm object
type ReconcileKeycloakRealm struct {
	// This client, initialized using mgr.Client() above, is a split client
	// that reads objects from the cache and writes to the apiserver
	client  client.Client
	factory keycloak.ClientFactory
	handler rHand.RealmHandler
	helper  *helper.Helper
}

// Reconcile reads that state of the cluster for a KeycloakRealm object and makes changes based on the state read
// and what is in the KeycloakRealm.Spec
// The Controller will requeue the Request to be processed again if the returned error is non-nil or
// Result.Requeue is true, otherwise upon completion it will remove the work from the queue.
func (r *ReconcileKeycloakRealm) Reconcile(request reconcile.Request) (result reconcile.Result, resultErr error) {
	reqLogger := log.WithValues("Request.Namespace", request.Namespace, "Request.Name", request.Name)
	reqLogger.Info("Reconciling KeycloakRealm")

	// Fetch the KeycloakRealm instance
	instance := &v1v1alpha1.KeycloakRealm{}
	err := r.client.Get(context.TODO(), request.NamespacedName, instance)
	if err != nil {
		if k8sErrors.IsNotFound(err) {
			// Request object not found, could have been deleted after reconcile request.
			// Owned objects are automatically garbage collected. For additional cleanup logic use finalizers.
			// Return and don't requeue
			return
		}
		// Error reading the object - requeue the request.
		resultErr = err
		return
	}

	defer func() {
		instance.Status.Available = true
		if resultErr != nil {
			result.RequeueAfter = r.helper.SetFailureCount(&instance.Status)
			instance.Status.Available = false
		}
		if err := r.helper.UpdateStatus(instance); err != nil {
			resultErr = err
		}
	}()

	if err := r.tryReconcile(instance); err != nil {
		resultErr = errors.Wrap(err, "error during tryReconcile")
		return
	}

	return
}

func (r *ReconcileKeycloakRealm) tryReconcile(realm *v1v1alpha1.KeycloakRealm) error {
	kClient, err := r.helper.CreateKeycloakClient(realm, r.factory)
	if err != nil {
		return err
	}

	deleted, err := r.helper.TryToDelete(realm, makeTerminator(realm.Spec.RealmName, kClient),
		keyCloakRealmOperatorFinalizerName)
	if err != nil {
		return errors.Wrap(err, "error during realm deletion")
	}
	if deleted {
		return nil
	}

	if err := r.handler.ServeRequest(realm, kClient); err != nil {
		return errors.Wrap(err, "error during realm chain")
	}

	return nil
}

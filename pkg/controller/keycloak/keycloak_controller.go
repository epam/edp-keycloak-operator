package keycloak

import (
	"context"
	"fmt"
	"gopkg.in/nerzal/gocloak.v2"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"keycloak-operator/pkg/adapter/keycloak"
	v1v1alpha1 "keycloak-operator/pkg/apis/v1/v1alpha1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	logf "sigs.k8s.io/controller-runtime/pkg/runtime/log"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

var log = logf.Log.WithName("controller_keycloak")

/**
* USER ACTION REQUIRED: This is a scaffold file intended for the user to modify with their own Controller
* business logic.  Delete these comments after modifying this file.*
 */

// Add creates a new Keycloak Controller and adds it to the Manager. The Manager will set fields on the Controller
// and Start it when the Manager is Started.
func Add(mgr manager.Manager) error {
	return add(mgr, newReconciler(mgr))
}

// newReconciler returns a new reconcile.Reconciler
func newReconciler(mgr manager.Manager) reconcile.Reconciler {
	adapter := keycloak.GoCloakAdapter{
		ClientSup: func(url string) gocloak.GoCloak {
			return gocloak.NewClient(url)
		},
	}
	return &ReconcileKeycloak{
		client:  mgr.GetClient(),
		scheme:  mgr.GetScheme(),
		adapter: adapter}
}

// add adds a new Controller to mgr with r as the reconcile.Reconciler
func add(mgr manager.Manager, r reconcile.Reconciler) error {
	// Create a new controller
	c, err := controller.New("keycloak-controller", mgr, controller.Options{Reconciler: r})
	if err != nil {
		return err
	}

	// Watch for changes to primary resource Keycloak
	return c.Watch(&source.Kind{Type: &v1v1alpha1.Keycloak{}}, &handler.EnqueueRequestForObject{})
}

// blank assignment to verify that ReconcileKeycloak implements reconcile.Reconciler
var _ reconcile.Reconciler = &ReconcileKeycloak{}

// ReconcileKeycloak reconciles a Keycloak object
type ReconcileKeycloak struct {
	// This client, initialized using mgr.Client() above, is a split client
	// that reads objects from the cache and writes to the apiserver
	client  client.Client
	scheme  *runtime.Scheme
	adapter keycloak.IGoCloakAdapter
}

// Reconcile reads that state of the cluster for a Keycloak object and makes changes based on the state read
// and what is in the Keycloak.Spec
// Note:
// The Controller will requeue the Request to be processed again if the returned error is non-nil or
// Result.Requeue is true, otherwise upon completion it will remove the work from the queue.
func (r *ReconcileKeycloak) Reconcile(request reconcile.Request) (reconcile.Result, error) {
	reqLogger := log.WithValues("Request.Namespace", request.Namespace, "Request.Name", request.Name)
	reqLogger.Info("Reconciling Keycloak")

	// Fetch the Keycloak instance
	instance := &v1v1alpha1.Keycloak{}
	err := r.client.Get(context.TODO(), request.NamespacedName, instance)
	if err != nil {
		if errors.IsNotFound(err) {
			// Request object not found, could have been deleted after reconcile request.
			// Owned objects are automatically garbage collected. For additional cleanup logic use finalizers.
			// Return and don't requeue
			return reconcile.Result{}, nil
		}
		// Error reading the object - requeue the request.
		return reconcile.Result{}, err
	}
	err = r.updateConnectionStatusToKeycloak(instance)
	if err != nil {
		return reconcile.Result{}, err
	}
	con, err := r.isStatusConnected(request)
	if err != nil {
		return reconcile.Result{}, err
	}
	if con {
		err = r.putMainRealm(instance)
	}
	reqLogger.Info("Reconciling Keycloak has been finished")
	return reconcile.Result{}, err
}

func (r *ReconcileKeycloak) updateConnectionStatusToKeycloak(instance *v1v1alpha1.Keycloak) error {
	reqLogger := log.WithValues("instance", instance)
	reqLogger.Info("Start updating connection status to Keycloak")
	_, err := r.adapter.GetConnection(*instance)
	instance.Status.Connected = err == nil
	err = r.client.Update(context.TODO(), instance)
	reqLogger.Info("Status has been updated", "status", instance.Status)
	return err
}

func (r *ReconcileKeycloak) putMainRealm(instance *v1v1alpha1.Keycloak) error {
	reqLog := log.WithValues("instance", instance)
	reqLog.Info("Start put main realm into k8s")
	nsn := types.NamespacedName{
		Name:      "main",
		Namespace: instance.Namespace,
	}
	realmCr := &v1v1alpha1.KeycloakRealm{}
	err := r.client.Get(context.TODO(), nsn, realmCr)
	reqLog.Info("Realm has been retrieved from k8s", "realmCr", realmCr)
	if errors.IsNotFound(err) {
		return r.createMainRealm(instance)
	}
	return err
}

func (r *ReconcileKeycloak) createMainRealm(instance *v1v1alpha1.Keycloak) error {
	reqLog := log.WithValues("instance", instance)
	reqLog.Info("Start creation of main Keycloak Realm CR")
	realmCr := &v1v1alpha1.KeycloakRealm{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "main",
			Namespace: instance.Namespace,
		},
		Spec: v1v1alpha1.KeycloakRealmSpec{
			RealmName: fmt.Sprintf("%s.%s", instance.Namespace, "main"),
		},
	}
	err := controllerutil.SetControllerReference(instance, realmCr, r.scheme)

	if err != nil {
		return err
	}
	err = r.client.Create(context.TODO(), realmCr)
	reqLog.Info("Keycloak Realm CR has been created", "keycloak realm", realmCr)

	return err
}

func (r *ReconcileKeycloak) isStatusConnected(request reconcile.Request) (bool, error) {
	log.Info("Check is status of CR is connected", "request", request)
	instance := &v1v1alpha1.Keycloak{}
	err := r.client.Get(context.TODO(), request.NamespacedName, instance)
	if err != nil {
		return false, err
	}
	log.Info("Retrieved the actual cr for Keycloak", "keycloak cr", instance)
	return instance.Status.Connected, nil
}

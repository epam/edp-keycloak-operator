package keycloakclient

import (
	"context"
	v1v1alpha1 "github.com/epmd-edp/keycloak-operator/pkg/apis/v1/v1alpha1"
	"github.com/epmd-edp/keycloak-operator/pkg/client/keycloak"
	"github.com/epmd-edp/keycloak-operator/pkg/client/keycloak/adapter"
	"github.com/epmd-edp/keycloak-operator/pkg/client/keycloak/dto"
	"github.com/epmd-edp/keycloak-operator/pkg/controller/helper"
	"k8s.io/apimachinery/pkg/types"

	coreV1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	logf "sigs.k8s.io/controller-runtime/pkg/runtime/log"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

var log = logf.Log.WithName("controller_keycloakclient")

/**
* USER ACTION REQUIRED: This is a scaffold file intended for the user to modify with their own Controller
* business logic.  Delete these comments after modifying this file.*
 */

// Add creates a new KeycloakClient Controller and adds it to the Manager. The Manager will set fields on the Controller
// and Start it when the Manager is Started.
func Add(mgr manager.Manager) error {
	return add(mgr, newReconciler(mgr))
}

// newReconciler returns a new reconcile.Reconciler
func newReconciler(mgr manager.Manager) reconcile.Reconciler {
	return &ReconcileKeycloakClient{
		client:  mgr.GetClient(),
		scheme:  mgr.GetScheme(),
		factory: new(adapter.GoCloakAdapterFactory),
	}
}

// add adds a new Controller to mgr with r as the reconcile.Reconciler
func add(mgr manager.Manager, r reconcile.Reconciler) error {
	// Create a new controller
	c, err := controller.New("keycloakclient-controller", mgr, controller.Options{Reconciler: r})
	if err != nil {
		return err
	}

	// Watch for changes to primary resource KeycloakClient
	return c.Watch(&source.Kind{Type: &v1v1alpha1.KeycloakClient{}}, &handler.EnqueueRequestForObject{})
}

// blank assignment to verify that ReconcileKeycloakClient implements reconcile.Reconciler
var _ reconcile.Reconciler = &ReconcileKeycloakClient{}

// ReconcileKeycloakClient reconciles a KeycloakClient object
type ReconcileKeycloakClient struct {
	// This client, initialized using mgr.Client() above, is a split client
	// that reads objects from the cache and writes to the apiserver
	client  client.Client
	scheme  *runtime.Scheme
	factory keycloak.ClientFactory
}

// Reconcile reads that state of the cluster for a KeycloakClient object and makes changes based on the state read
// and what is in the KeycloakClient.Spec
// Note:
// The Controller will requeue the Request to be processed again if the returned error is non-nil or
// Result.Requeue is true, otherwise upon completion it will remove the work from the queue.
func (r *ReconcileKeycloakClient) Reconcile(request reconcile.Request) (reconcile.Result, error) {
	reqLogger := log.WithValues("Request.Namespace", request.Namespace, "Request.Name", request.Name)
	reqLogger.Info("Reconciling KeycloakClient")

	// Fetch the KeycloakClient instance
	instance := &v1v1alpha1.KeycloakClient{}
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

	err = r.putKeycloakClient(instance)

	if err != nil {
		return reconcile.Result{}, err
	}

	return reconcile.Result{}, nil
}

func (r *ReconcileKeycloakClient) putKeycloakClient(keycloakClient *v1v1alpha1.KeycloakClient) error {
	reqLog := log.WithValues("keycloak client cr", keycloakClient)
	reqLog.Info("Start put keycloak client...")

	realm, err := helper.GetOwnerKeycloakRealm(r.client, keycloakClient.ObjectMeta)
	if err != nil {
		return err
	}
	keycloakCr, err := helper.GetOwnerKeycloak(r.client, realm.ObjectMeta)
	if err != nil {
		return nil
	}
	clientSecret := &coreV1.Secret{}
	err = r.client.Get(context.TODO(), types.NamespacedName{
		Name:      keycloakClient.Spec.Secret,
		Namespace: keycloakClient.Namespace,
	}, clientSecret)
	if err != nil {
		return err
	}
	clientId := string(clientSecret.Data["clientId"])
	clientSecretVal := string(clientSecret.Data["clientSecret"])
	clientDto := dto.ConvertSpecToClient(keycloakClient.Spec, clientId, clientSecretVal)

	keycloakSecret := &coreV1.Secret{}
	err = r.client.Get(context.TODO(), types.NamespacedName{
		Name:      keycloakCr.Spec.Secret,
		Namespace: keycloakCr.Namespace,
	}, keycloakSecret)
	if err != nil {
		return err
	}
	usr := string(keycloakSecret.Data["username"])
	pwd := string(keycloakSecret.Data["password"])

	keyDto := dto.ConvertSpecToKeycloak(keycloakCr.Spec, usr, pwd)
	kClient, err := r.factory.New(keyDto)

	if err != nil {
		return err
	}

	exist, err := kClient.ExistClient(clientDto)

	if err != nil {
		return err
	}

	if *exist {
		reqLog.Info("Client already exists")
		return nil
	}

	err = kClient.CreateClient(clientDto)
	if err != nil {
		return err
	}

	reqLog.Info("End put keycloak client")
	return nil
}

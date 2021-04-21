package keycloakrealm

import (
	"context"
	keycloakApi "github.com/epam/edp-keycloak-operator/pkg/apis/v1/v1alpha1"
	"github.com/epam/edp-keycloak-operator/pkg/client/keycloak"
	"github.com/epam/edp-keycloak-operator/pkg/client/keycloak/adapter"
	"github.com/epam/edp-keycloak-operator/pkg/controller/helper"
	"github.com/epam/edp-keycloak-operator/pkg/controller/keycloakrealm/chain"
	"github.com/go-logr/logr"
	"github.com/pkg/errors"
	k8sErrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

const keyCloakRealmOperatorFinalizerName = "keycloak.realm.operator.finalizer.name"

func NewReconcileKeycloakRealm(client client.Client, scheme *runtime.Scheme, log logr.Logger, helper *helper.Helper) *ReconcileKeycloakRealm {
	return &ReconcileKeycloakRealm{
		client: client,
		scheme: scheme,
		factory: adapter.GoCloakAdapterFactory{
			Log: ctrl.Log.WithName("go-cloak-adapter-factory"),
		},
		helper: helper,
		log:    log.WithName("keycloak-realm"),
	}
}

// ReconcileKeycloakRealm reconciles a KeycloakRealm object
type ReconcileKeycloakRealm struct {
	client  client.Client
	scheme  *runtime.Scheme
	factory keycloak.ClientFactory
	helper  *helper.Helper
	log     logr.Logger
}

func (r *ReconcileKeycloakRealm) SetupWithManager(mgr ctrl.Manager) error {
	pred := predicate.Funcs{
		UpdateFunc: helper.IsFailuresUpdated,
	}
	return ctrl.NewControllerManagedBy(mgr).
		For(&keycloakApi.KeycloakRealm{}, builder.WithPredicates(pred)).
		Complete(r)
}

func (r *ReconcileKeycloakRealm) Reconcile(ctx context.Context, request reconcile.Request) (result reconcile.Result, resultErr error) {
	log := r.log.WithValues("Request.Namespace", request.Namespace, "Request.Name", request.Name)
	log.Info("Reconciling KeycloakRealm")

	instance := &keycloakApi.KeycloakRealm{}
	if err := r.client.Get(ctx, request.NamespacedName, instance); err != nil {
		if k8sErrors.IsNotFound(err) {
			// Request object not found, could have been deleted after reconcile request.
			// Owned objects are automatically garbage collected. For additional cleanup logic use finalizers.
			// Return and don't requeue
			return
		}
		resultErr = err
		return
	}

	if err := r.tryReconcile(instance); err != nil {
		instance.Status.Available = false
		result.RequeueAfter = r.helper.SetFailureCount(instance)
		log.Error(err, "an error has occurred while handling keycloak realm", "name",
			request.Name)
	} else {
		instance.Status.Available = true
		instance.Status.FailureCount = 0
	}

	if err := r.helper.UpdateStatus(instance); err != nil {
		resultErr = errors.Wrap(err, "unable to update status")
	}

	return
}

func (r *ReconcileKeycloakRealm) tryReconcile(realm *keycloakApi.KeycloakRealm) error {
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

	if err := chain.CreateDefChain(r.client, r.scheme).ServeRequest(realm, kClient); err != nil {
		return errors.Wrap(err, "error during realm chain")
	}

	return nil
}

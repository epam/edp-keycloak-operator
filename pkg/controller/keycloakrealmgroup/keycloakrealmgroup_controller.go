package keycloakrealmgroup

import (
	"context"

	"github.com/epam/edp-keycloak-operator/pkg/apis/v1/v1alpha1"
	"github.com/epam/edp-keycloak-operator/pkg/client/keycloak"
	"github.com/epam/edp-keycloak-operator/pkg/client/keycloak/adapter"
	"github.com/epam/edp-keycloak-operator/pkg/controller/helper"
	"github.com/pkg/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	logf "sigs.k8s.io/controller-runtime/pkg/runtime/log"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

var log = logf.Log.WithName("controller_keycloakrealmgroup")

const (
	keyCloakRealmGroupOperatorFinalizerName = "keycloak.realmgroup.operator.finalizer.name"
)

type ReconcileKeycloakRealmGroup struct {
	client  client.Client
	scheme  *runtime.Scheme
	factory keycloak.ClientFactory
	helper  *helper.Helper
}

// newReconciler returns a new reconcile.Reconciler
func newReconciler(mgr manager.Manager) reconcile.Reconciler {
	return &ReconcileKeycloakRealmGroup{
		client:  mgr.GetClient(),
		scheme:  mgr.GetScheme(),
		factory: new(adapter.GoCloakAdapterFactory),
		helper:  helper.MakeHelper(mgr.GetClient(), mgr.GetScheme()),
	}
}

func Add(mgr manager.Manager) error {
	c, err := controller.New("keycloakrealmgroup-controller", mgr, controller.Options{
		Reconciler: newReconciler(mgr)})
	if err != nil {
		return err
	}

	return c.Watch(&source.Kind{Type: &v1alpha1.KeycloakRealmGroup{}}, &handler.EnqueueRequestForObject{},
		predicate.Funcs{
			UpdateFunc: helper.IsFailuresUpdated,
		})
}

func (r *ReconcileKeycloakRealmGroup) Reconcile(request reconcile.Request) (result reconcile.Result, resultErr error) {
	reqLogger := log.WithValues("Request.Namespace", request.Namespace, "Request.Name", request.Name)
	reqLogger.Info("Reconciling KeycloakRealmGroup")

	var instance v1alpha1.KeycloakRealmGroup
	if err := r.client.Get(context.TODO(), request.NamespacedName, &instance); err != nil {
		resultErr = errors.Wrap(err, "unable to get keycloak realm group from k8s")
		return
	}

	if err := r.tryReconcile(&instance); err != nil {
		instance.Status.Value = err.Error()
		result.RequeueAfter = r.helper.SetFailureCount(&instance)
		log.Error(err, "an error has occurred while handling keycloak realm group", "name",
			request.Name)
	} else {
		helper.SetSuccessStatus(&instance)
	}

	if err := r.helper.UpdateStatus(&instance); err != nil {
		resultErr = errors.Wrap(err, "unable to update status")
	}

	return
}

func (r *ReconcileKeycloakRealmGroup) tryReconcile(keycloakRealmGroup *v1alpha1.KeycloakRealmGroup) error {
	realm, err := r.helper.GetOrCreateRealmOwnerRef(keycloakRealmGroup, keycloakRealmGroup.ObjectMeta)
	if err != nil {
		return errors.Wrap(err, "unable to get realm owner ref")
	}

	kClient, err := r.helper.CreateKeycloakClient(realm, r.factory)
	if err != nil {
		return errors.Wrap(err, "unable to create keycloak client")
	}

	id, err := kClient.SyncRealmGroup(realm.Spec.RealmName, &keycloakRealmGroup.Spec)
	if err != nil {
		return errors.Wrap(err, "unable to sync realm role")
	}
	keycloakRealmGroup.Status.ID = id

	if _, err := r.helper.TryToDelete(keycloakRealmGroup,
		makeTerminator(kClient, realm.Spec.RealmName, keycloakRealmGroup.Spec.Name),
		keyCloakRealmGroupOperatorFinalizerName); err != nil {
		return errors.Wrap(err, "unable to tryToDelete realm role")
	}

	return nil
}

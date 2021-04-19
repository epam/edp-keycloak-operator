package keycloakrealmrole

import (
	"context"

	"github.com/epam/edp-keycloak-operator/pkg/apis/v1/v1alpha1"
	"github.com/epam/edp-keycloak-operator/pkg/client/keycloak"
	"github.com/epam/edp-keycloak-operator/pkg/client/keycloak/adapter"
	"github.com/epam/edp-keycloak-operator/pkg/client/keycloak/dto"
	"github.com/epam/edp-keycloak-operator/pkg/controller/helper"
	"github.com/go-logr/logr"
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

const (
	keyCloakRealmRoleOperatorFinalizerName = "keycloak.realmrole.operator.finalizer.name"
)

type ReconcileKeycloakRealmRole struct {
	client  client.Client
	scheme  *runtime.Scheme
	factory keycloak.ClientFactory
	helper  *helper.Helper
	logger  logr.Logger
}

// newReconciler returns a new reconcile.Reconciler
func newReconciler(mgr manager.Manager) reconcile.Reconciler {
	return &ReconcileKeycloakRealmRole{
		client:  mgr.GetClient(),
		scheme:  mgr.GetScheme(),
		factory: new(adapter.GoCloakAdapterFactory),
		helper:  helper.MakeHelper(mgr.GetClient(), mgr.GetScheme()),
		logger:  logf.Log.WithName("controller_keycloakrealmrole"),
	}
}

func Add(mgr manager.Manager) error {
	c, err := controller.New("keycloakrealmrole-controller", mgr, controller.Options{
		Reconciler: newReconciler(mgr)})
	if err != nil {
		return err
	}

	return c.Watch(&source.Kind{Type: &v1alpha1.KeycloakRealmRole{}}, &handler.EnqueueRequestForObject{},
		predicate.Funcs{
			UpdateFunc: helper.IsFailuresUpdated,
		})
}

func (r *ReconcileKeycloakRealmRole) Reconcile(request reconcile.Request) (result reconcile.Result, resultErr error) {
	reqLogger := r.logger.WithValues("Request.Namespace", request.Namespace, "Request.Name", request.Name)
	reqLogger.Info("Reconciling KeycloakRealmRole")

	var instance v1alpha1.KeycloakRealmRole
	if err := r.client.Get(context.TODO(), request.NamespacedName, &instance); err != nil {
		resultErr = errors.Wrap(err, "unable to get keycloak realm role from k8s")
		return
	}

	if err := r.tryReconcile(&instance); err != nil {
		instance.Status.Value = err.Error()
		result.RequeueAfter = r.helper.SetFailureCount(&instance)
		r.logger.Error(err, "an error has occurred while handling keycloak realm role", "name",
			request.Name)
	} else {
		helper.SetSuccessStatus(&instance)
	}

	if err := r.helper.UpdateStatus(&instance); err != nil {
		resultErr = err
	}

	return
}

func (r *ReconcileKeycloakRealmRole) tryReconcile(keycloakRealmRole *v1alpha1.KeycloakRealmRole) error {
	realm, err := r.helper.GetOrCreateRealmOwnerRef(keycloakRealmRole, keycloakRealmRole.ObjectMeta)
	if err != nil {
		return errors.Wrap(err, "unable to get realm owner ref")
	}

	kClient, err := r.helper.CreateKeycloakClient(realm, r.factory)
	if err != nil {
		return errors.Wrap(err, "unable to create keycloak client")
	}

	if err := r.putRole(realm, keycloakRealmRole, kClient); err != nil {
		return errors.Wrap(err, "unable to put role")
	}

	if _, err := r.helper.TryToDelete(keycloakRealmRole,
		makeTerminator(realm.Spec.RealmName, keycloakRealmRole.Spec.Name, kClient, r.logger),
		keyCloakRealmRoleOperatorFinalizerName); err != nil {
		return errors.Wrap(err, "unable to tryToDelete realm role")
	}

	return nil
}

func (r *ReconcileKeycloakRealmRole) putRole(
	keycloakRealm *v1alpha1.KeycloakRealm, keycloakRealmRole *v1alpha1.KeycloakRealmRole,
	kClient keycloak.Client) error {

	reqLog := r.logger.WithValues("keycloak role cr", keycloakRealmRole)
	reqLog.Info("Start put keycloak cr role...")

	realm := dto.ConvertSpecToRealm(keycloakRealm.Spec)
	role := dto.ConvertSpecToRole(&keycloakRealmRole.Spec)

	if err := kClient.SyncRealmRole(realm, role); err != nil {
		return errors.Wrap(err, "unable to sync realm role CR")
	}

	if role.ID != nil {
		keycloakRealmRole.Status.ID = *role.ID
	}
	reqLog.Info("Done putting keycloak cr role...")

	return nil
}

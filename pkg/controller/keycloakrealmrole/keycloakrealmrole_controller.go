package keycloakrealmrole

import (
	"context"

	"github.com/epmd-edp/keycloak-operator/pkg/apis/v1/v1alpha1"
	"github.com/epmd-edp/keycloak-operator/pkg/client/keycloak"
	"github.com/epmd-edp/keycloak-operator/pkg/client/keycloak/adapter"
	"github.com/epmd-edp/keycloak-operator/pkg/client/keycloak/dto"
	"github.com/epmd-edp/keycloak-operator/pkg/controller/helper"
	"github.com/pkg/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	logf "sigs.k8s.io/controller-runtime/pkg/runtime/log"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

var log = logf.Log.WithName("controller_keycloakrealmrole")

const (
	keyCloakRealmRoleOperatorFinalizerName = "keycloak.realmrole.operator.finalizer.name"
)

type ReconcileKeycloakRealmRole struct {
	client  client.Client
	scheme  *runtime.Scheme
	factory keycloak.ClientFactory
	helper  *helper.Helper
}

// newReconciler returns a new reconcile.Reconciler
func newReconciler(mgr manager.Manager) reconcile.Reconciler {
	return &ReconcileKeycloakRealmRole{
		client:  mgr.GetClient(),
		scheme:  mgr.GetScheme(),
		factory: new(adapter.GoCloakAdapterFactory),
		helper:  helper.MakeHelper(mgr.GetClient(), mgr.GetScheme()),
	}
}

func Add(mgr manager.Manager) error {
	c, err := controller.New("keycloakrealmrole-controller", mgr, controller.Options{
		Reconciler: newReconciler(mgr)})
	if err != nil {
		return err
	}

	// Watch for changes to primary resource KeycloakRealm
	return c.Watch(&source.Kind{Type: &v1alpha1.KeycloakRealmRole{}}, &handler.EnqueueRequestForObject{})
}

func (r *ReconcileKeycloakRealmRole) Reconcile(request reconcile.Request) (result reconcile.Result, resultErr error) {
	reqLogger := log.WithValues("Request.Namespace", request.Namespace, "Request.Name", request.Name)
	reqLogger.Info("Reconciling KeycloakRealmRole")

	var instance v1alpha1.KeycloakRealmRole
	if err := r.client.Get(context.TODO(), request.NamespacedName, &instance); err != nil {
		resultErr = errors.Wrap(err, "unable to get keycloak realm role from k8s")
		return
	}

	defer func() {
		if resultErr != nil {
			instance.Status.Value = resultErr.Error()
		}

		if err := r.updateStatus(&instance); err != nil {
			resultErr = err
		}
	}()

	if err := r.tryReconcile(&instance); err != nil {
		resultErr = err
		return
	}

	return
}

func (r *ReconcileKeycloakRealmRole) updateStatus(kc *v1alpha1.KeycloakRealmRole) error {
	if err := r.client.Status().Update(context.TODO(), kc); err != nil {
		return errors.Wrap(err, "unable to update realm role status")
	}

	return nil
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

	if err := r.tryToDelete(keycloakRealmRole, realm, kClient); err != nil {
		return errors.Wrap(err, "unable to tryToDelete realm role")
	}

	return nil
}

func (r *ReconcileKeycloakRealmRole) tryToDelete(
	keycloakRealmRole *v1alpha1.KeycloakRealmRole, keycloakReam *v1alpha1.KeycloakRealm,
	kClient keycloak.Client) error {

	if keycloakRealmRole.GetDeletionTimestamp().IsZero() {
		if !helper.ContainsString(keycloakRealmRole.ObjectMeta.Finalizers, keyCloakRealmRoleOperatorFinalizerName) {
			keycloakRealmRole.ObjectMeta.Finalizers = append(keycloakRealmRole.ObjectMeta.Finalizers,
				keyCloakRealmRoleOperatorFinalizerName)
			if err := r.client.Update(context.TODO(), keycloakRealmRole); err != nil {
				return errors.Wrap(err, "unable to update kk client cr")
			}
		}

		return nil
	}

	reqLog := log.WithValues("keycloak realm role cr", keycloakRealmRole)
	reqLog.Info("Start deleting keycloak client...")

	if err := kClient.DeleteRealmRole(keycloakReam.Spec.RealmName, keycloakRealmRole.Spec.Name); err != nil {
		return errors.Wrap(err, "unable to delete realm role")
	}

	reqLog.Info("realm role deletion done")

	keycloakRealmRole.ObjectMeta.Finalizers = helper.RemoveString(keycloakRealmRole.ObjectMeta.Finalizers,
		keyCloakRealmRoleOperatorFinalizerName)
	if err := r.client.Update(context.TODO(), keycloakRealmRole); err != nil {
		return errors.Wrap(err, "unable to update realm role cr")
	}

	return nil
}

func (r *ReconcileKeycloakRealmRole) putRole(
	keycloakRealm *v1alpha1.KeycloakRealm, keycloakRealmRole *v1alpha1.KeycloakRealmRole,
	kClient keycloak.Client) error {

	reqLog := log.WithValues("keycloak role cr", keycloakRealmRole)
	reqLog.Info("Start put keycloak cr role...")

	realm := dto.ConvertSpecToRealm(keycloakRealm.Spec)
	role := dto.ConvertSpecToRole(&keycloakRealmRole.Spec)

	if err := kClient.SyncRealmRole(&realm, &role); err != nil {
		return errors.Wrap(err, "unable to sync realm role CR")
	}

	if role.ID != nil {
		keycloakRealmRole.Status.ID = *role.ID
	}
	reqLog.Info("Done putting keycloak cr role...")

	return nil
}

package keycloakrealmgroup

import (
	"context"
	"github.com/go-logr/logr"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"

	"github.com/epmd-edp/keycloak-operator/pkg/apis/v1/v1alpha1"
	keycloakApi "github.com/epmd-edp/keycloak-operator/pkg/apis/v1/v1alpha1"
	"github.com/epmd-edp/keycloak-operator/pkg/client/keycloak"
	"github.com/epmd-edp/keycloak-operator/pkg/controller/helper"
	"github.com/pkg/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

const keyCloakRealmGroupOperatorFinalizerName = "keycloak.realmgroup.operator.finalizer.name"

type ReconcileKeycloakRealmGroup struct {
	Client  client.Client
	Scheme  *runtime.Scheme
	Factory keycloak.ClientFactory
	Helper  *helper.Helper
	Log     logr.Logger
}

func (r *ReconcileKeycloakRealmGroup) SetupWithManager(mgr ctrl.Manager) error {
	pred := predicate.Funcs{
		UpdateFunc: helper.IsFailuresUpdated,
	}
	return ctrl.NewControllerManagedBy(mgr).
		For(&keycloakApi.KeycloakRealmGroup{}, builder.WithPredicates(pred)).
		Complete(r)
}

func (r *ReconcileKeycloakRealmGroup) Reconcile(ctx context.Context, request reconcile.Request) (result reconcile.Result, resultErr error) {
	log := r.Log.WithValues("Request.Namespace", request.Namespace, "Request.Name", request.Name)
	log.Info("Reconciling KeycloakRealmGroup")

	var instance keycloakApi.KeycloakRealmGroup
	if err := r.Client.Get(ctx, request.NamespacedName, &instance); err != nil {
		resultErr = errors.Wrap(err, "unable to get keycloak realm group from k8s")
		return
	}

	if err := r.tryReconcile(&instance); err != nil {
		instance.Status.Value = err.Error()
		result.RequeueAfter = r.Helper.SetFailureCount(&instance)
		log.Error(err, "an error has occurred while handling keycloak realm group", "name",
			request.Name)
	} else {
		helper.SetSuccessStatus(&instance)
	}

	if err := r.Helper.UpdateStatus(&instance); err != nil {
		resultErr = errors.Wrap(err, "unable to update status")
	}

	return
}

func (r *ReconcileKeycloakRealmGroup) tryReconcile(keycloakRealmGroup *v1alpha1.KeycloakRealmGroup) error {
	realm, err := r.Helper.GetOrCreateRealmOwnerRef(keycloakRealmGroup, keycloakRealmGroup.ObjectMeta)
	if err != nil {
		return errors.Wrap(err, "unable to get realm owner ref")
	}

	kClient, err := r.Helper.CreateKeycloakClient(realm, r.Factory)
	if err != nil {
		return errors.Wrap(err, "unable to create keycloak client")
	}

	id, err := kClient.SyncRealmGroup(realm.Spec.RealmName, &keycloakRealmGroup.Spec)
	if err != nil {
		return errors.Wrap(err, "unable to sync realm role")
	}
	keycloakRealmGroup.Status.ID = id

	if _, err := r.Helper.TryToDelete(keycloakRealmGroup,
		makeTerminator(kClient, realm.Spec.RealmName, keycloakRealmGroup.Spec.Name),
		keyCloakRealmGroupOperatorFinalizerName); err != nil {
		return errors.Wrap(err, "unable to tryToDelete realm role")
	}

	return nil
}

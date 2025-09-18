package keycloakrealmrole

import (
	"context"
	"github.com/epam/edp-keycloak-operator/pkg/client/keycloak/adapter"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"

	"github.com/epam/edp-keycloak-operator/pkg/apis/v1/v1alpha1"
	keycloakApi "github.com/epam/edp-keycloak-operator/pkg/apis/v1/v1alpha1"
	"github.com/epam/edp-keycloak-operator/pkg/client/keycloak"
	"github.com/epam/edp-keycloak-operator/pkg/client/keycloak/dto"
	"github.com/epam/edp-keycloak-operator/pkg/controller/helper"
	"github.com/go-logr/logr"
	"github.com/pkg/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

const keyCloakRealmRoleOperatorFinalizerName = "keycloak.realmrole.operator.finalizer.name"

func NewReconcileKeycloakRealmRole(client client.Client, scheme *runtime.Scheme, log logr.Logger, helper *helper.Helper) *ReconcileKeycloakRealmRole {
	return &ReconcileKeycloakRealmRole{
		client: client,
		scheme: scheme,
		factory: adapter.GoCloakAdapterFactory{
			Log: ctrl.Log.WithName("go-cloak-adapter-factory"),
		},
		helper: helper,
		log:    log.WithName("keycloak-realm-role"),
	}
}

type ReconcileKeycloakRealmRole struct {
	client  client.Client
	scheme  *runtime.Scheme
	factory keycloak.ClientFactory
	helper  *helper.Helper
	log     logr.Logger
}

func (r *ReconcileKeycloakRealmRole) SetupWithManager(mgr ctrl.Manager) error {
	pred := predicate.Funcs{
		UpdateFunc: helper.IsFailuresUpdated,
	}
	return ctrl.NewControllerManagedBy(mgr).
		For(&keycloakApi.KeycloakRealmRole{}, builder.WithPredicates(pred)).
		Complete(r)
}

func (r *ReconcileKeycloakRealmRole) Reconcile(ctx context.Context, request reconcile.Request) (result reconcile.Result, resultErr error) {
	log := r.log.WithValues("Request.Namespace", request.Namespace, "Request.Name", request.Name)
	log.Info("Reconciling KeycloakRealmRole")

	var instance keycloakApi.KeycloakRealmRole
	if err := r.client.Get(ctx, request.NamespacedName, &instance); err != nil {
		resultErr = errors.Wrap(err, "unable to get keycloak realm role from k8s")
		return
	}

	if err := r.tryReconcile(&instance); err != nil {
		instance.Status.Value = err.Error()
		result.RequeueAfter = r.helper.SetFailureCount(&instance)
		log.Error(err, "an error has occurred while handling keycloak realm role", "name",
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
		makeTerminator(realm.Spec.RealmName, keycloakRealmRole.Spec.Name, kClient),
		keyCloakRealmRoleOperatorFinalizerName); err != nil {
		return errors.Wrap(err, "unable to tryToDelete realm role")
	}

	return nil
}

func (r *ReconcileKeycloakRealmRole) putRole(
	keycloakRealm *v1alpha1.KeycloakRealm, keycloakRealmRole *v1alpha1.KeycloakRealmRole,
	kClient keycloak.Client) error {

	log := r.log.WithValues("keycloak role cr", keycloakRealmRole)
	log.Info("Start put keycloak cr role...")

	role := dto.ConvertSpecToRole(&keycloakRealmRole.Spec)

	if err := kClient.SyncRealmRole(keycloakRealm.Spec.RealmName, role); err != nil {
		return errors.Wrap(err, "unable to sync realm role CR")
	}

	if role.ID != nil {
		keycloakRealmRole.Status.ID = *role.ID
	}
	log.Info("Done putting keycloak cr role...")

	return nil
}

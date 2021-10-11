package keycloakrealmgroup

import (
	"context"
	"time"

	k8sErrors "k8s.io/apimachinery/pkg/api/errors"

	"github.com/epam/edp-keycloak-operator/pkg/apis/v1/v1alpha1"
	keycloakApi "github.com/epam/edp-keycloak-operator/pkg/apis/v1/v1alpha1"
	"github.com/epam/edp-keycloak-operator/pkg/client/keycloak"
	"github.com/epam/edp-keycloak-operator/pkg/controller/helper"
	"github.com/go-logr/logr"
	"github.com/pkg/errors"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

const keyCloakRealmGroupOperatorFinalizerName = "keycloak.realmgroup.operator.finalizer.name"

type Helper interface {
	SetFailureCount(fc helper.FailureCountable) time.Duration
	UpdateStatus(obj client.Object) error
	GetOrCreateRealmOwnerRef(object helper.RealmChild, objectMeta v1.ObjectMeta) (*v1alpha1.KeycloakRealm, error)
	TryToDelete(ctx context.Context, obj helper.Deletable, terminator helper.Terminator, finalizer string) (isDeleted bool, resultErr error)
	CreateKeycloakClientForRealm(ctx context.Context, realm *v1alpha1.KeycloakRealm) (keycloak.Client, error)
}

func NewReconcileKeycloakRealmGroup(client client.Client, scheme *runtime.Scheme, log logr.Logger,
	helper Helper) *ReconcileKeycloakRealmGroup {
	return &ReconcileKeycloakRealmGroup{
		client: client,
		scheme: scheme,
		helper: helper,
		log:    log.WithName("keycloak-realm-group"),
	}
}

type ReconcileKeycloakRealmGroup struct {
	client client.Client
	scheme *runtime.Scheme
	helper Helper
	log    logr.Logger
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
	log := r.log.WithValues("Request.Namespace", request.Namespace, "Request.Name", request.Name)
	log.Info("Reconciling KeycloakRealmGroup")

	var instance keycloakApi.KeycloakRealmGroup
	if err := r.client.Get(ctx, request.NamespacedName, &instance); err != nil {
		if k8sErrors.IsNotFound(err) {
			return
		}

		resultErr = errors.Wrap(err, "unable to get keycloak realm group from k8s")
		return
	}

	if err := r.tryReconcile(ctx, &instance); err != nil {
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

func (r *ReconcileKeycloakRealmGroup) tryReconcile(ctx context.Context, keycloakRealmGroup *v1alpha1.KeycloakRealmGroup) error {
	realm, err := r.helper.GetOrCreateRealmOwnerRef(keycloakRealmGroup, keycloakRealmGroup.ObjectMeta)
	if err != nil {
		return errors.Wrap(err, "unable to get realm owner ref")
	}

	kClient, err := r.helper.CreateKeycloakClientForRealm(ctx, realm)
	if err != nil {
		return errors.Wrap(err, "unable to create keycloak client")
	}

	id, err := kClient.SyncRealmGroup(realm.Spec.RealmName, &keycloakRealmGroup.Spec)
	if err != nil {
		return errors.Wrap(err, "unable to sync realm role")
	}
	keycloakRealmGroup.Status.ID = id

	if _, err := r.helper.TryToDelete(ctx, keycloakRealmGroup,
		makeTerminator(kClient, realm.Spec.RealmName, keycloakRealmGroup.Spec.Name,
			r.log.WithName("realm-group-term")),
		keyCloakRealmGroupOperatorFinalizerName); err != nil {
		return errors.Wrap(err, "unable to tryToDelete realm role")
	}

	return nil
}

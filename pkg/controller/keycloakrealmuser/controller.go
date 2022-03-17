package keycloakrealmuser

import (
	"context"
	"reflect"
	"time"

	"github.com/epam/edp-keycloak-operator/pkg/apis/v1/v1alpha1"
	keycloakApi "github.com/epam/edp-keycloak-operator/pkg/apis/v1/v1alpha1"
	"github.com/epam/edp-keycloak-operator/pkg/client/keycloak"
	"github.com/epam/edp-keycloak-operator/pkg/client/keycloak/adapter"
	"github.com/epam/edp-keycloak-operator/pkg/controller/helper"
	"github.com/go-logr/logr"
	"github.com/pkg/errors"
	k8sErrors "k8s.io/apimachinery/pkg/api/errors"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

const finalizer = "keycloak.realmuser.operator.finalizer.name"

type Helper interface {
	SetFailureCount(fc helper.FailureCountable) time.Duration
	UpdateStatus(obj client.Object) error
	CreateKeycloakClientForRealm(ctx context.Context, realm *v1alpha1.KeycloakRealm) (keycloak.Client, error)
	GetOrCreateRealmOwnerRef(object helper.RealmChild, objectMeta v1.ObjectMeta) (*v1alpha1.KeycloakRealm, error)
	TryToDelete(ctx context.Context, obj helper.Deletable, terminator helper.Terminator, finalizer string) (isDeleted bool, resultErr error)
}

type Reconcile struct {
	client client.Client
	helper Helper
	log    logr.Logger
}

func NewReconcile(client client.Client, log logr.Logger, helper Helper) *Reconcile {
	return &Reconcile{
		client: client,
		helper: helper,
		log:    log.WithName("keycloak-realm-user"),
	}
}

func (r *Reconcile) SetupWithManager(mgr ctrl.Manager) error {
	pred := predicate.Funcs{
		UpdateFunc: isSpecUpdated,
		DeleteFunc: func(deleteEvent event.DeleteEvent) bool {
			return false
		},
	}
	return ctrl.NewControllerManagedBy(mgr).
		For(&keycloakApi.KeycloakRealmUser{}, builder.WithPredicates(pred)).
		Complete(r)
}

func isSpecUpdated(e event.UpdateEvent) bool {
	oo := e.ObjectOld.(*keycloakApi.KeycloakRealmUser)
	no := e.ObjectNew.(*keycloakApi.KeycloakRealmUser)

	return !reflect.DeepEqual(oo.Spec, no.Spec) ||
		(oo.GetDeletionTimestamp().IsZero() && !no.GetDeletionTimestamp().IsZero())
}

func (r *Reconcile) Reconcile(ctx context.Context, request reconcile.Request) (result reconcile.Result,
	resultErr error) {
	log := r.log.WithValues("Request.Namespace", request.Namespace, "Request.Name", request.Name)
	log.Info("Reconciling KeycloakRealmUser")

	var instance keycloakApi.KeycloakRealmUser
	if err := r.client.Get(ctx, request.NamespacedName, &instance); err != nil {
		if k8sErrors.IsNotFound(err) {
			return
		}

		resultErr = errors.Wrap(err, "unable to get keycloak realm user from k8s")
		return
	}

	if err := r.tryReconcile(ctx, &instance); err != nil {
		instance.Status.Value = err.Error()
		result.RequeueAfter = r.helper.SetFailureCount(&instance)
		log.Error(err, "an error has occurred while handling keycloak auth flow", "name",
			request.Name)
	} else {
		helper.SetSuccessStatus(&instance)
	}

	if err := r.helper.UpdateStatus(&instance); err != nil {
		resultErr = err
	}

	log.Info("Reconciling KeycloakRealmUser done.")
	return
}

func (r *Reconcile) tryReconcile(ctx context.Context, instance *keycloakApi.KeycloakRealmUser) error {
	realm, err := r.helper.GetOrCreateRealmOwnerRef(instance, instance.ObjectMeta)
	if err != nil {
		return errors.Wrap(err, "unable to get realm owner ref")
	}

	kClient, err := r.helper.CreateKeycloakClientForRealm(ctx, realm)
	if err != nil {
		return errors.Wrap(err, "unable to create keycloak client")
	}

	if err := kClient.SyncRealmUser(ctx, realm.Spec.RealmName, &adapter.KeycloakUser{
		Username:            instance.Spec.Username,
		Groups:              instance.Spec.Groups,
		Roles:               instance.Spec.Roles,
		RequiredUserActions: instance.Spec.RequiredUserActions,
		LastName:            instance.Spec.LastName,
		FirstName:           instance.Spec.FirstName,
		EmailVerified:       instance.Spec.EmailVerified,
		Enabled:             instance.Spec.Enabled,
		Email:               instance.Spec.Email,
		Attributes:          instance.Spec.Attributes,
		Password:            instance.Spec.Password,
	}, instance.GetReconciliationStrategy() == v1alpha1.ReconciliationStrategyAddOnly); err != nil {
		return errors.Wrap(err, "unable to sync realm user")
	}

	if instance.Spec.KeepResource {
		if _, err := r.helper.TryToDelete(ctx, instance,
			makeTerminator(realm.Spec.RealmName, instance.Spec.Username, kClient, r.log), finalizer); err != nil {
			return errors.Wrap(err, "unable to set finalizers")
		}
	} else {
		if err := r.client.Delete(ctx, instance); err != nil {
			return errors.Wrap(err, "unable to delete instance of keycloak realm user")
		}
	}

	return nil
}

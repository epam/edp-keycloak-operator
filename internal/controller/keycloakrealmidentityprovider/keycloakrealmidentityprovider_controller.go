package keycloakrealmidentityprovider

import (
	"context"
	"fmt"
	"reflect"
	"time"

	"github.com/Nerzal/gocloak/v12"
	"github.com/pkg/errors"
	k8sErrors "k8s.io/apimachinery/pkg/api/errors"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	keycloakApi "github.com/epam/edp-keycloak-operator/api/v1"
	"github.com/epam/edp-keycloak-operator/internal/controller/helper"
	"github.com/epam/edp-keycloak-operator/internal/controller/keycloakrealmidentityprovider/chain"
	"github.com/epam/edp-keycloak-operator/pkg/client/keycloak"
	"github.com/epam/edp-keycloak-operator/pkg/objectmeta"
)

const finalizerName = "keycloak.realmidp.operator.finalizer.name"

type Helper interface {
	SetFailureCount(fc helper.FailureCountable) time.Duration
	TryRemoveFinalizer(ctx context.Context, obj client.Object, finalizer string) error
	TryToDelete(ctx context.Context, obj client.Object, terminator helper.Terminator, finalizer string) (isDeleted bool, resultErr error)
	SetRealmOwnerRef(ctx context.Context, object helper.ObjectWithRealmRef) error
	GetKeycloakRealmFromRef(ctx context.Context, object helper.ObjectWithRealmRef, kcClient keycloak.Client) (*gocloak.RealmRepresentation, error)
	CreateKeycloakClientFromRealmRef(ctx context.Context, object helper.ObjectWithRealmRef) (keycloak.Client, error)
}

type Reconcile struct {
	client                  client.Client
	helper                  Helper
	successReconcileTimeout time.Duration
}

func NewReconcile(k8sClient client.Client, controllerHelper Helper) *Reconcile {
	return &Reconcile{
		client: k8sClient,
		helper: controllerHelper,
	}
}

func (r *Reconcile) SetupWithManager(mgr ctrl.Manager, successReconcileTimeout time.Duration) error {
	r.successReconcileTimeout = successReconcileTimeout

	pred := predicate.Funcs{
		UpdateFunc: isSpecUpdated,
	}

	err := ctrl.NewControllerManagedBy(mgr).
		For(&keycloakApi.KeycloakRealmIdentityProvider{}, builder.WithPredicates(pred)).
		Complete(r)
	if err != nil {
		return fmt.Errorf("failed to setup KeycloakRealmIdentityProvider controller: %w", err)
	}

	return nil
}

func isSpecUpdated(e event.UpdateEvent) bool {
	oo, ok := e.ObjectOld.(*keycloakApi.KeycloakRealmIdentityProvider)
	if !ok {
		return false
	}

	no, ok := e.ObjectNew.(*keycloakApi.KeycloakRealmIdentityProvider)
	if !ok {
		return false
	}

	return !reflect.DeepEqual(oo.Spec, no.Spec) ||
		(oo.GetDeletionTimestamp().IsZero() && !no.GetDeletionTimestamp().IsZero())
}

// +kubebuilder:rbac:groups=v1.edp.epam.com,namespace=placeholder,resources=keycloakrealmidentityproviders,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=v1.edp.epam.com,namespace=placeholder,resources=keycloakrealmidentityproviders/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=v1.edp.epam.com,namespace=placeholder,resources=keycloakrealmidentityproviders/finalizers,verbs=update

// Reconcile is a loop for reconciling KeycloakRealmIdentityProvider object.
func (r *Reconcile) Reconcile(ctx context.Context, request reconcile.Request) (result reconcile.Result, resultErr error) {
	log := ctrl.LoggerFrom(ctx)
	log.Info("Reconciling KeycloakRealmIdentityProvider")

	var instance keycloakApi.KeycloakRealmIdentityProvider
	if err := r.client.Get(ctx, request.NamespacedName, &instance); err != nil {
		if k8sErrors.IsNotFound(err) {
			log.Info("instance not found")

			return result, resultErr
		}

		resultErr = errors.Wrap(err, "unable to get keycloak realm idp from k8s")

		return result, resultErr
	}

	if err := r.tryReconcile(ctx, &instance); err != nil {
		if errors.Is(err, helper.ErrKeycloakIsNotAvailable) {
			return ctrl.Result{
				RequeueAfter: helper.RequeueOnKeycloakNotAvailablePeriod,
			}, nil
		}

		instance.Status.Value = err.Error()
		result.RequeueAfter = r.helper.SetFailureCount(&instance)

		log.Error(err, "an error has occurred while handling keycloak realm idp", "name", request.Name)
	} else {
		helper.SetSuccessStatus(&instance)

		result.RequeueAfter = r.successReconcileTimeout
	}

	if err := r.client.Status().Update(ctx, &instance); err != nil {
		resultErr = errors.Wrap(err, "unable to update status")
	}

	return result, resultErr
}

func (r *Reconcile) tryReconcile(ctx context.Context, keycloakRealmIDP *keycloakApi.KeycloakRealmIdentityProvider) error {
	err := r.helper.SetRealmOwnerRef(ctx, keycloakRealmIDP)
	if err != nil {
		return fmt.Errorf("unable to set realm owner ref: %w", err)
	}

	kClient, err := r.helper.CreateKeycloakClientFromRealmRef(ctx, keycloakRealmIDP)
	if err != nil {
		// if the realm is already deleted try to delete finalizer
		if errors.Is(err, helper.ErrKeycloakRealmNotFound) {
			if removeErr := r.helper.TryRemoveFinalizer(ctx, keycloakRealmIDP, finalizerName); removeErr != nil {
				return fmt.Errorf("unable to remove finalizer: %w", removeErr)
			}

			return nil
		}

		return fmt.Errorf("unable to create keycloak client from realm ref: %w", err)
	}

	realm, err := r.helper.GetKeycloakRealmFromRef(ctx, keycloakRealmIDP, kClient)
	if err != nil {
		return fmt.Errorf("unable to get keycloak realm from ref: %w", err)
	}

	term := makeTerminator(
		gocloak.PString(realm.Realm),
		keycloakRealmIDP.Spec.Alias,
		kClient,
		objectmeta.PreserveResourcesOnDeletion(keycloakRealmIDP),
	)

	deleted, err := r.helper.TryToDelete(ctx, keycloakRealmIDP, term, finalizerName)
	if err != nil {
		return fmt.Errorf("unable to delete realm idp: %w", err)
	}

	if deleted {
		return nil
	}

	if err = chain.MakeChain(kClient, r.client).Serve(ctx, keycloakRealmIDP, *realm.Realm); err != nil {
		return fmt.Errorf("unable to serve keycloak idp: %w", err)
	}

	return nil
}

package keycloakrealmgroup

import (
	"context"
	"fmt"
	"time"

	"github.com/Nerzal/gocloak/v12"
	"github.com/pkg/errors"
	k8sErrors "k8s.io/apimachinery/pkg/api/errors"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	"github.com/epam/edp-keycloak-operator/api/common"
	keycloakApi "github.com/epam/edp-keycloak-operator/api/v1"
	"github.com/epam/edp-keycloak-operator/controllers/helper"
	"github.com/epam/edp-keycloak-operator/pkg/client/keycloak"
	"github.com/epam/edp-keycloak-operator/pkg/objectmeta"
)

const keyCloakRealmGroupOperatorFinalizerName = "keycloak.realmgroup.operator.finalizer.name"

type Helper interface {
	SetFailureCount(fc helper.FailureCountable) time.Duration
	TryToDelete(ctx context.Context, obj client.Object, terminator helper.Terminator, finalizer string) (isDeleted bool, resultErr error)
	GetKeycloakRealmFromRef(ctx context.Context, object helper.ObjectWithRealmRef, kcClient keycloak.Client) (*gocloak.RealmRepresentation, error)
	CreateKeycloakClientFromRealmRef(ctx context.Context, object helper.ObjectWithRealmRef) (keycloak.Client, error)
}

func NewReconcileKeycloakRealmGroup(client client.Client,
	helper Helper) *ReconcileKeycloakRealmGroup {
	return &ReconcileKeycloakRealmGroup{
		client: client,
		helper: helper,
	}
}

type ReconcileKeycloakRealmGroup struct {
	client                  client.Client
	helper                  Helper
	successReconcileTimeout time.Duration
}

func (r *ReconcileKeycloakRealmGroup) SetupWithManager(mgr ctrl.Manager, successReconcileTimeout time.Duration) error {
	r.successReconcileTimeout = successReconcileTimeout

	pred := predicate.Funcs{
		UpdateFunc: helper.IsFailuresUpdated,
	}

	err := ctrl.NewControllerManagedBy(mgr).
		For(&keycloakApi.KeycloakRealmGroup{}, builder.WithPredicates(pred)).
		Complete(r)
	if err != nil {
		return fmt.Errorf("failed to setup KeycloakRealmGroup controller: %w", err)
	}

	return nil
}

//+kubebuilder:rbac:groups=v1.edp.epam.com,namespace=placeholder,resources=keycloakrealmgroups,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=v1.edp.epam.com,namespace=placeholder,resources=keycloakrealmgroups/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=v1.edp.epam.com,namespace=placeholder,resources=keycloakrealmgroups/finalizers,verbs=update

// Reconcile is a loop for reconciling KeycloakRealmGroup object.
func (r *ReconcileKeycloakRealmGroup) Reconcile(ctx context.Context, request reconcile.Request) (result reconcile.Result, resultErr error) {
	log := ctrl.LoggerFrom(ctx)
	log.Info("Reconciling KeycloakRealmGroup")

	var instance keycloakApi.KeycloakRealmGroup
	if err := r.client.Get(ctx, request.NamespacedName, &instance); err != nil {
		if k8sErrors.IsNotFound(err) {
			return
		}

		resultErr = errors.Wrap(err, "unable to get keycloak realm group from k8s")

		return
	}

	if updated, err := r.applyDefaults(ctx, &instance); err != nil {
		resultErr = fmt.Errorf("unable to apply default values: %w", err)
		return
	} else if updated {
		return
	}

	if err := r.tryReconcile(ctx, &instance); err != nil {
		if errors.Is(err, helper.ErrKeycloakIsNotAvailable) {
			return ctrl.Result{
				RequeueAfter: helper.RequeueOnKeycloakNotAvailablePeriod,
			}, nil
		}

		instance.Status.Value = err.Error()
		result.RequeueAfter = r.helper.SetFailureCount(&instance)

		log.Error(err, "an error has occurred while handling keycloak realm group", "name", request.Name)
	} else {
		helper.SetSuccessStatus(&instance)
		result.RequeueAfter = r.successReconcileTimeout
	}

	if err := r.client.Status().Update(ctx, &instance); err != nil {
		resultErr = errors.Wrap(err, "unable to update status")
	}

	log.Info("Reconciling done")

	return
}

func (r *ReconcileKeycloakRealmGroup) tryReconcile(ctx context.Context, keycloakRealmGroup *keycloakApi.KeycloakRealmGroup) error {
	kClient, err := r.helper.CreateKeycloakClientFromRealmRef(ctx, keycloakRealmGroup)
	if err != nil {
		return fmt.Errorf("unable to create keycloak client from realm ref: %w", err)
	}

	realm, err := r.helper.GetKeycloakRealmFromRef(ctx, keycloakRealmGroup, kClient)
	if err != nil {
		return fmt.Errorf("unable to get keycloak realm from ref: %w", err)
	}

	id, err := kClient.SyncRealmGroup(gocloak.PString(realm.Realm), &keycloakRealmGroup.Spec)
	if err != nil {
		return fmt.Errorf("unable to sync realm group: %w", err)
	}

	keycloakRealmGroup.Status.ID = id

	if _, err := r.helper.TryToDelete(
		ctx,
		keycloakRealmGroup,
		makeTerminator(
			kClient,
			gocloak.PString(realm.Realm),
			keycloakRealmGroup.Spec.Name,
			objectmeta.PreserveResourcesOnDeletion(keycloakRealmGroup),
		),
		keyCloakRealmGroupOperatorFinalizerName,
	); err != nil {
		return fmt.Errorf("failed to delete keycloak realm group: %w", err)
	}

	return nil
}

func (r *ReconcileKeycloakRealmGroup) applyDefaults(ctx context.Context, instance *keycloakApi.KeycloakRealmGroup) (bool, error) {
	if instance.Spec.RealmRef.Name == "" {
		instance.Spec.RealmRef = common.RealmRef{
			Kind: keycloakApi.KeycloakRealmKind,
			Name: instance.Spec.Realm,
		}

		if err := r.client.Update(ctx, instance); err != nil {
			return false, fmt.Errorf("failed to update default values: %w", err)
		}

		return true, nil
	}

	return false, nil
}

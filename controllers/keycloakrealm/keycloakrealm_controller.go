package keycloakrealm

import (
	"context"
	"fmt"
	"time"

	"github.com/pkg/errors"
	k8sErrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	"github.com/epam/edp-keycloak-operator/api/common"
	keycloakApi "github.com/epam/edp-keycloak-operator/api/v1"
	"github.com/epam/edp-keycloak-operator/controllers/helper"
	"github.com/epam/edp-keycloak-operator/controllers/keycloakrealm/chain"
	"github.com/epam/edp-keycloak-operator/controllers/keycloakrealm/chain/handler"
	"github.com/epam/edp-keycloak-operator/pkg/client/keycloak"
	"github.com/epam/edp-keycloak-operator/pkg/objectmeta"
)

const keyCloakRealmOperatorFinalizerName = "keycloak.realm.operator.finalizer.name"

type Helper interface {
	SetFailureCount(fc helper.FailureCountable) time.Duration
	TryToDelete(ctx context.Context, obj client.Object, terminator helper.Terminator, finalizer string) (isDeleted bool, resultErr error)
	CreateKeycloakClientFromRealm(ctx context.Context, realm *keycloakApi.KeycloakRealm) (keycloak.Client, error)
	InvalidateKeycloakClientTokenSecret(ctx context.Context, namespace, rootKeycloakName string) error
}

func NewReconcileKeycloakRealm(client client.Client, scheme *runtime.Scheme, helper Helper) *ReconcileKeycloakRealm {
	return &ReconcileKeycloakRealm{
		client: client,
		helper: helper,
		chain:  chain.CreateDefChain(client, scheme, helper),
	}
}

// ReconcileKeycloakRealm reconciles a KeycloakRealm object.
type ReconcileKeycloakRealm struct {
	client                  client.Client
	helper                  Helper
	chain                   handler.RealmHandler
	successReconcileTimeout time.Duration
}

func (r *ReconcileKeycloakRealm) SetupWithManager(mgr ctrl.Manager, successReconcileTimeout time.Duration) error {
	r.successReconcileTimeout = successReconcileTimeout
	pred := predicate.Funcs{
		UpdateFunc: helper.IsFailuresUpdated,
	}

	err := ctrl.NewControllerManagedBy(mgr).
		For(&keycloakApi.KeycloakRealm{}, builder.WithPredicates(pred)).
		Complete(r)
	if err != nil {
		return fmt.Errorf("failed to setup KeycloakRealm controller: %w", err)
	}

	return nil
}

//+kubebuilder:rbac:groups=v1.edp.epam.com,namespace=placeholder,resources=keycloakrealms,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=v1.edp.epam.com,namespace=placeholder,resources=keycloakrealms/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=v1.edp.epam.com,namespace=placeholder,resources=keycloakrealms/finalizers,verbs=update
//+kubebuilder:rbac:groups="",namespace=placeholder,resources=secrets,verbs=get;list;watch;create;update;patch;delete

// Reconcile is a loop for reconciling KeycloakRealm object.
func (r *ReconcileKeycloakRealm) Reconcile(ctx context.Context, request reconcile.Request) (result reconcile.Result, resultErr error) {
	log := ctrl.LoggerFrom(ctx)
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

	if updated, err := r.applyDefaults(ctx, instance); err != nil {
		resultErr = fmt.Errorf("unable to apply default values: %w", err)
		return
	} else if updated {
		return
	}

	if err := r.tryReconcile(ctx, instance); err != nil {
		if errors.Is(err, helper.ErrKeycloakIsNotAvailable) {
			return ctrl.Result{
				RequeueAfter: helper.RequeueOnKeycloakNotAvailablePeriod,
			}, nil
		}

		instance.Status.Available = false
		instance.Status.Value = err.Error()
		result.RequeueAfter = r.helper.SetFailureCount(instance)

		log.Error(err, "an error has occurred while handling keycloak realm", "name", request.Name)
	} else {
		instance.Status.Available = true
		instance.Status.Value = helper.StatusOK
		instance.Status.FailureCount = 0
		result.RequeueAfter = r.successReconcileTimeout
	}

	if err := r.client.Status().Update(ctx, instance); err != nil {
		resultErr = errors.Wrap(err, "unable to update status")
	}

	return
}

func (r *ReconcileKeycloakRealm) tryReconcile(ctx context.Context, realm *keycloakApi.KeycloakRealm) error {
	kClient, err := r.helper.CreateKeycloakClientFromRealm(ctx, realm)
	if err != nil {
		return fmt.Errorf("failed to create keycloak client for realm: %w", err)
	}

	deleted, err := r.helper.TryToDelete(
		ctx,
		realm,
		makeTerminator(realm.Spec.RealmName, kClient, objectmeta.PreserveResourcesOnDeletion(realm)),
		keyCloakRealmOperatorFinalizerName,
	)
	if err != nil {
		return fmt.Errorf("failed to delete realm: %w", err)
	}

	if deleted {
		return nil
	}

	if err := r.chain.ServeRequest(ctx, realm, kClient); err != nil {
		return errors.Wrap(err, "error during realm chain")
	}

	return nil
}

func (r *ReconcileKeycloakRealm) applyDefaults(ctx context.Context, instance *keycloakApi.KeycloakRealm) (bool, error) {
	if instance.Spec.KeycloakRef.Name == "" {
		instance.Spec.KeycloakRef = common.KeycloakRef{
			Kind: keycloakApi.KeycloakKind,
			Name: instance.Spec.KeycloakOwner,
		}

		if err := r.client.Update(ctx, instance); err != nil {
			return false, fmt.Errorf("failed to update default values: %w", err)
		}

		return true, nil
	}

	return false, nil
}

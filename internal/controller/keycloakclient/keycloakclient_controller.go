package keycloakclient

import (
	"context"
	"errors"
	"fmt"
	"time"

	k8sErrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	keycloakApi "github.com/epam/edp-keycloak-operator/api/v1"
	"github.com/epam/edp-keycloak-operator/internal/controller/helper"
	"github.com/epam/edp-keycloak-operator/internal/controller/keycloakclient/chain"
	"github.com/epam/edp-keycloak-operator/pkg/client/keycloakapi"
)

type Helper interface {
	SetFailureCount(fc helper.FailureCountable) time.Duration
	SetRealmOwnerRef(ctx context.Context, object helper.ObjectWithRealmRef) error
	CreateKeycloakClientFromRealmRef(ctx context.Context, object helper.ObjectWithRealmRef) (*keycloakapi.KeycloakClient, error)
	GetRealmNameFromRef(ctx context.Context, object helper.ObjectWithRealmRef) (string, error)
}

const keyCloakClientOperatorFinalizerName = "keycloak.client.operator.finalizer.name"

func NewReconcileKeycloakClient(k8sClient client.Client, controllerHelper Helper) *ReconcileKeycloakClient {
	return &ReconcileKeycloakClient{
		client: k8sClient,
		helper: controllerHelper,
	}
}

// ReconcileKeycloakClient reconciles a KeycloakClient object.
type ReconcileKeycloakClient struct {
	client                  client.Client
	helper                  Helper
	successReconcileTimeout time.Duration
}

func (r *ReconcileKeycloakClient) SetupWithManager(mgr ctrl.Manager, successReconcileTimeout time.Duration) error {
	r.successReconcileTimeout = successReconcileTimeout

	pred := predicate.Funcs{
		UpdateFunc: helper.IsFailuresUpdated,
	}

	err := ctrl.NewControllerManagedBy(mgr).
		For(&keycloakApi.KeycloakClient{}, builder.WithPredicates(pred)).
		Complete(r)
	if err != nil {
		return fmt.Errorf("failed to setup KeycloakClient controller: %w", err)
	}

	return nil
}

// +kubebuilder:rbac:groups=v1.edp.epam.com,namespace=placeholder,resources=keycloakclients,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=v1.edp.epam.com,namespace=placeholder,resources=keycloakclients/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=v1.edp.epam.com,namespace=placeholder,resources=keycloakclients/finalizers,verbs=update

// Reconcile is a loop for reconciling KeycloakClient object.
func (r *ReconcileKeycloakClient) Reconcile(ctx context.Context, request reconcile.Request) (result reconcile.Result, resultErr error) {
	log := ctrl.LoggerFrom(ctx)
	log.Info("Reconciling KeycloakClient")

	instance, kClient, realmName, err := r.initializeReconciliation(ctx, request)
	if err != nil {
		return reconcile.Result{}, err
	}

	if instance == nil {
		return reconcile.Result{}, nil
	}

	if instance.GetDeletionTimestamp() != nil {
		return r.handleDeletion(ctx, instance, kClient, realmName)
	}

	return r.handleReconciliation(ctx, instance, kClient, realmName)
}

func (r *ReconcileKeycloakClient) initializeReconciliation(ctx context.Context, request reconcile.Request) (*keycloakApi.KeycloakClient, *keycloakapi.KeycloakClient, string, error) {
	instance := &keycloakApi.KeycloakClient{}
	if err := r.client.Get(ctx, request.NamespacedName, instance); err != nil {
		if k8sErrors.IsNotFound(err) {
			return nil, nil, "", nil
		}

		return nil, nil, "", fmt.Errorf("failed to get KeycloakClient: %w", err)
	}

	if err := r.helper.SetRealmOwnerRef(ctx, instance); err != nil {
		return nil, nil, "", fmt.Errorf("unable to set realm owner ref: %w", err)
	}

	kClient, err := r.helper.CreateKeycloakClientFromRealmRef(ctx, instance)
	if err != nil {
		if errors.Is(err, helper.ErrKeycloakRealmNotFound) && instance.GetDeletionTimestamp() != nil {
			stop, removeErr := helper.RemoveFinalizersOnRealmNotFound(ctx, r.client, instance, keyCloakClientOperatorFinalizerName)
			if removeErr != nil {
				return nil, nil, "", removeErr
			}

			if stop {
				return nil, nil, "", nil
			}
		}

		return nil, nil, "", fmt.Errorf("unable to create keycloak client from realm ref: %w", err)
	}

	realmName, err := r.helper.GetRealmNameFromRef(ctx, instance)
	if err != nil {
		return nil, nil, "", fmt.Errorf("unable to get realm name from ref: %w", err)
	}

	return instance, kClient, realmName, nil
}

func (r *ReconcileKeycloakClient) handleDeletion(ctx context.Context, instance *keycloakApi.KeycloakClient, kClient *keycloakapi.KeycloakClient, realmName string) (reconcile.Result, error) {
	if controllerutil.ContainsFinalizer(instance, keyCloakClientOperatorFinalizerName) {
		if err := chain.NewRemoveClient(kClient).Serve(ctx, instance, realmName); err != nil {
			return ctrl.Result{}, fmt.Errorf("failed to remove keycloak client: %w", err)
		}

		controllerutil.RemoveFinalizer(instance, keyCloakClientOperatorFinalizerName)

		if err := r.client.Update(ctx, instance); err != nil {
			return ctrl.Result{}, fmt.Errorf("failed to update keycloak client after finalizer removal: %w", err)
		}
	}

	return ctrl.Result{}, nil
}

func (r *ReconcileKeycloakClient) handleReconciliation(ctx context.Context, instance *keycloakApi.KeycloakClient, kClient *keycloakapi.KeycloakClient, realmName string) (reconcile.Result, error) {
	log := ctrl.LoggerFrom(ctx)

	if controllerutil.AddFinalizer(instance, keyCloakClientOperatorFinalizerName) {
		if err := r.client.Update(ctx, instance); err != nil {
			return ctrl.Result{}, fmt.Errorf("failed to add finalizer to keycloak client: %w", err)
		}
	}

	var resultErr error

	if err := chain.MakeChain(kClient, r.client).Serve(ctx, instance, realmName); err != nil {
		if errors.Is(err, helper.ErrKeycloakIsNotAvailable) {
			return ctrl.Result{RequeueAfter: helper.RequeueOnKeycloakNotAvailablePeriod}, nil
		}

		log.Error(err, "an error has occurred while handling keycloak client", "name", instance.Name)

		meta.SetStatusCondition(&instance.Status.Conditions, metav1.Condition{
			Type:               chain.ConditionReady,
			Status:             metav1.ConditionFalse,
			Reason:             chain.ReasonKeycloakAPIError,
			Message:            fmt.Sprintf("Reconciliation failed: %s", err.Error()),
			ObservedGeneration: instance.Generation,
		})

		instance.Status.Value = err.Error()
		resultErr = fmt.Errorf("keycloak client chain processing failed: %w", err)
	} else {
		meta.SetStatusCondition(&instance.Status.Conditions, metav1.Condition{
			Type:               chain.ConditionReady,
			Status:             metav1.ConditionTrue,
			Reason:             chain.ReasonReconciliationSucceeded,
			Message:            "KeycloakClient reconciliation completed successfully",
			ObservedGeneration: instance.Generation,
		})

		helper.SetSuccessStatus(instance)
	}

	if err := r.client.Status().Update(ctx, instance); err != nil {
		return reconcile.Result{}, fmt.Errorf("unable to update status: %w", err)
	}

	if resultErr != nil {
		return reconcile.Result{RequeueAfter: r.helper.SetFailureCount(instance)}, resultErr
	}

	return reconcile.Result{RequeueAfter: r.successReconcileTimeout}, nil
}

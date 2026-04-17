package keycloakauthflow

import (
	"context"
	"errors"
	"fmt"
	"time"

	"k8s.io/apimachinery/pkg/api/equality"
	k8sErrors "k8s.io/apimachinery/pkg/api/errors"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	"github.com/epam/edp-keycloak-operator/api/common"
	keycloakApi "github.com/epam/edp-keycloak-operator/api/v1"
	"github.com/epam/edp-keycloak-operator/internal/controller/helper"
	"github.com/epam/edp-keycloak-operator/internal/controller/keycloakauthflow/chain"
	keycloakapi "github.com/epam/edp-keycloak-operator/pkg/client/keycloakapi"
)

const successRequeueTime = time.Minute * 10

// Deprecated: legacyFinalizerName is the old finalizer used before migration to common.FinalizerName.
// Kept to ensure existing resources carrying the old finalizer can be deleted cleanly.
const legacyFinalizerName = "keycloak.authflow.operator.finalizer.name"

// Helper is the subset of controller helper methods used by this reconciler.
type Helper interface {
	SetRealmOwnerRef(ctx context.Context, object helper.ObjectWithRealmRef) error
	GetRealmNameFromRef(ctx context.Context, object helper.ObjectWithRealmRef) (string, error)
	CreateKeycloakeycloakAPIClientFromRealmRef(ctx context.Context, object helper.ObjectWithRealmRef) (*keycloakapi.APIClient, error)
}

func NewReconcile(k8sClient client.Client, controllerHelper Helper) *Reconcile {
	return &Reconcile{
		client: k8sClient,
		helper: controllerHelper,
	}
}

// Reconcile reconciles a KeycloakAuthFlow object.
type Reconcile struct {
	client client.Client
	helper Helper
}

func (r *Reconcile) SetupWithManager(mgr ctrl.Manager) error {
	if err := ctrl.NewControllerManagedBy(mgr).
		For(&keycloakApi.KeycloakAuthFlow{}).
		Complete(r); err != nil {
		return fmt.Errorf("failed to setup KeycloakAuthFlow controller: %w", err)
	}

	return nil
}

// +kubebuilder:rbac:groups=v1.edp.epam.com,namespace=placeholder,resources=keycloakauthflows,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=v1.edp.epam.com,namespace=placeholder,resources=keycloakauthflows/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=v1.edp.epam.com,namespace=placeholder,resources=keycloakauthflows/finalizers,verbs=update

// Reconcile is a loop for reconciling KeycloakAuthFlow objects.
func (r *Reconcile) Reconcile(ctx context.Context, request reconcile.Request) (result reconcile.Result, resultErr error) {
	log := ctrl.LoggerFrom(ctx)
	log.Info("Reconciling KeycloakAuthFlow")

	instance, keycloakAPIClient, realmName, err := r.initializeReconciliation(ctx, request)
	if err != nil {
		if errors.Is(err, helper.ErrKeycloakIsNotAvailable) {
			return ctrl.Result{RequeueAfter: helper.RequeueOnKeycloakNotAvailablePeriod}, nil
		}

		return reconcile.Result{}, err
	}

	if instance == nil {
		return reconcile.Result{}, nil
	}

	if instance.GetDeletionTimestamp() != nil {
		return r.handleDeletion(ctx, instance, keycloakAPIClient, realmName)
	}

	return r.handleReconciliation(ctx, instance, keycloakAPIClient, realmName)
}

func (r *Reconcile) initializeReconciliation(ctx context.Context, request reconcile.Request) (*keycloakApi.KeycloakAuthFlow, *keycloakapi.APIClient, string, error) {
	instance := &keycloakApi.KeycloakAuthFlow{}
	if err := r.client.Get(ctx, request.NamespacedName, instance); err != nil {
		if k8sErrors.IsNotFound(err) {
			return nil, nil, "", nil
		}

		return nil, nil, "", fmt.Errorf("failed to get KeycloakAuthFlow: %w", err)
	}

	if err := r.helper.SetRealmOwnerRef(ctx, instance); err != nil {
		return nil, nil, "", fmt.Errorf("unable to set realm owner ref: %w", err)
	}

	keycloakAPIClient, err := r.helper.CreateKeycloakeycloakAPIClientFromRealmRef(ctx, instance)
	if err != nil {
		if errors.Is(err, helper.ErrKeycloakRealmNotFound) && instance.GetDeletionTimestamp() != nil {
			stop, removeErr := helper.RemoveFinalizersOnRealmNotFound(ctx, r.client, instance, common.FinalizerName, legacyFinalizerName)
			if removeErr != nil {
				return nil, nil, "", removeErr
			}

			if stop {
				return nil, nil, "", nil
			}
		}

		return nil, nil, "", fmt.Errorf("failed to create Keycloak client: %w", err)
	}

	realmName, err := r.helper.GetRealmNameFromRef(ctx, instance)
	if err != nil {
		return nil, nil, "", fmt.Errorf("unable to get realm name from ref: %w", err)
	}

	return instance, keycloakAPIClient, realmName, nil
}

func (r *Reconcile) handleDeletion(ctx context.Context, instance *keycloakApi.KeycloakAuthFlow, keycloakAPIClient *keycloakapi.APIClient, realmName string) (reconcile.Result, error) {
	if controllerutil.ContainsFinalizer(instance, common.FinalizerName) || controllerutil.ContainsFinalizer(instance, legacyFinalizerName) {
		if err := chain.NewRemoveAuthFlow(keycloakAPIClient, r.client).Serve(ctx, instance, realmName); err != nil {
			return ctrl.Result{}, fmt.Errorf("failed to remove auth flow: %w", err)
		}

		controllerutil.RemoveFinalizer(instance, common.FinalizerName)
		controllerutil.RemoveFinalizer(instance, legacyFinalizerName)

		if err := r.client.Update(ctx, instance); err != nil {
			return ctrl.Result{}, fmt.Errorf("failed to update KeycloakAuthFlow after finalizer removal: %w", err)
		}
	}

	return ctrl.Result{}, nil
}

func (r *Reconcile) handleReconciliation(ctx context.Context, instance *keycloakApi.KeycloakAuthFlow, keycloakAPIClient *keycloakapi.APIClient, realmName string) (reconcile.Result, error) {
	log := ctrl.LoggerFrom(ctx)

	if controllerutil.AddFinalizer(instance, common.FinalizerName) {
		if err := r.client.Update(ctx, instance); err != nil {
			return ctrl.Result{}, fmt.Errorf("failed to add finalizer to KeycloakAuthFlow: %w", err)
		}
	}

	oldStatus := instance.Status

	if err := chain.MakeChain(keycloakAPIClient).Serve(ctx, instance, realmName); err != nil {
		log.Error(err, "An error has occurred while handling KeycloakAuthFlow")

		resultErr := fmt.Errorf("auth flow chain processing failed: %w", err)
		instance.Status.Value = resultErr.Error()

		if statusErr := r.updateKeycloakAuthFlowStatus(ctx, instance, oldStatus); statusErr != nil {
			return reconcile.Result{}, fmt.Errorf("failed to update KeycloakAuthFlow status (%s): %w", resultErr, statusErr)
		}

		return reconcile.Result{}, resultErr
	}

	instance.Status.Value = common.StatusOK

	if err := r.updateKeycloakAuthFlowStatus(ctx, instance, oldStatus); err != nil {
		return reconcile.Result{}, err
	}

	log.Info("Reconciling KeycloakAuthFlow done")

	return reconcile.Result{RequeueAfter: successRequeueTime}, nil
}

func (r *Reconcile) updateKeycloakAuthFlowStatus(ctx context.Context, instance *keycloakApi.KeycloakAuthFlow, oldStatus keycloakApi.KeycloakAuthFlowStatus) error {
	if equality.Semantic.DeepEqual(&instance.Status, &oldStatus) {
		return nil
	}

	if err := r.client.Status().Update(ctx, instance); err != nil {
		return fmt.Errorf("failed to update KeycloakAuthFlow status: %w", err)
	}

	return nil
}

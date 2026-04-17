package keycloakrealmidentityprovider

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
	"github.com/epam/edp-keycloak-operator/internal/controller/keycloakrealmidentityprovider/chain"
	keycloakapi "github.com/epam/edp-keycloak-operator/pkg/client/keycloakapi"
)

const successRequeueTime = time.Minute * 10

// Deprecated: legacyFinalizerName is the old finalizer used before migration to common.FinalizerName.
// Kept to ensure existing resources carrying the old finalizer can be deleted cleanly.
const legacyFinalizerName = "keycloak.realmidp.operator.finalizer.name"

type IdentityProviderReconcilerCtrlHelper interface {
	SetRealmOwnerRef(ctx context.Context, object helper.ObjectWithRealmRef) error
	GetRealmNameFromRef(ctx context.Context, object helper.ObjectWithRealmRef) (string, error)
	CreateKeycloakeycloakAPIClientFromRealmRef(ctx context.Context, object helper.ObjectWithRealmRef) (*keycloakapi.APIClient, error)
}

type IdentityProviderReconciler struct {
	client client.Client
	helper IdentityProviderReconcilerCtrlHelper
}

func NewIdentityProviderReconciler(k8sClient client.Client, controllerHelper IdentityProviderReconcilerCtrlHelper) *IdentityProviderReconciler {
	return &IdentityProviderReconciler{
		client: k8sClient,
		helper: controllerHelper,
	}
}

func (r *IdentityProviderReconciler) SetupWithManager(mgr ctrl.Manager) error {
	if err := ctrl.NewControllerManagedBy(mgr).
		For(&keycloakApi.KeycloakRealmIdentityProvider{}).
		Complete(r); err != nil {
		return fmt.Errorf("failed to setup KeycloakRealmIdentityProvider controller: %w", err)
	}

	return nil
}

// +kubebuilder:rbac:groups=v1.edp.epam.com,namespace=placeholder,resources=keycloakrealmidentityproviders,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=v1.edp.epam.com,namespace=placeholder,resources=keycloakrealmidentityproviders/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=v1.edp.epam.com,namespace=placeholder,resources=keycloakrealmidentityproviders/finalizers,verbs=update

// Reconcile is a loop for reconciling KeycloakRealmIdentityProvider object.
func (r *IdentityProviderReconciler) Reconcile(ctx context.Context, request reconcile.Request) (result reconcile.Result, resultErr error) {
	log := ctrl.LoggerFrom(ctx)
	log.Info("Reconciling KeycloakRealmIdentityProvider")

	instance, kClient, realmName, err := r.initializeReconciliation(ctx, request)
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
		return r.handleDeletion(ctx, instance, kClient, realmName)
	}

	return r.handleReconciliation(ctx, instance, kClient, realmName)
}

func (r *IdentityProviderReconciler) initializeReconciliation(ctx context.Context, request reconcile.Request) (*keycloakApi.KeycloakRealmIdentityProvider, *keycloakapi.APIClient, string, error) {
	instance := &keycloakApi.KeycloakRealmIdentityProvider{}
	if err := r.client.Get(ctx, request.NamespacedName, instance); err != nil {
		if k8sErrors.IsNotFound(err) {
			return nil, nil, "", nil
		}

		return nil, nil, "", fmt.Errorf("failed to get KeycloakRealmIdentityProvider: %w", err)
	}

	if err := r.helper.SetRealmOwnerRef(ctx, instance); err != nil {
		return nil, nil, "", fmt.Errorf("unable to set realm owner ref: %w", err)
	}

	kClient, err := r.helper.CreateKeycloakeycloakAPIClientFromRealmRef(ctx, instance)
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

	return instance, kClient, realmName, nil
}

func (r *IdentityProviderReconciler) handleDeletion(ctx context.Context, instance *keycloakApi.KeycloakRealmIdentityProvider, kClient *keycloakapi.APIClient, realmName string) (reconcile.Result, error) {
	if controllerutil.ContainsFinalizer(instance, common.FinalizerName) || controllerutil.ContainsFinalizer(instance, legacyFinalizerName) {
		if err := chain.NewRemoveIDP(kClient).Serve(ctx, instance, realmName); err != nil {
			return ctrl.Result{}, fmt.Errorf("failed to remove identity provider: %w", err)
		}

		controllerutil.RemoveFinalizer(instance, common.FinalizerName)
		controllerutil.RemoveFinalizer(instance, legacyFinalizerName)

		if err := r.client.Update(ctx, instance); err != nil {
			return ctrl.Result{}, fmt.Errorf("failed to update KeycloakRealmIdentityProvider after finalizer removal: %w", err)
		}
	}

	return ctrl.Result{}, nil
}

func (r *IdentityProviderReconciler) handleReconciliation(ctx context.Context, instance *keycloakApi.KeycloakRealmIdentityProvider, kClient *keycloakapi.APIClient, realmName string) (reconcile.Result, error) {
	log := ctrl.LoggerFrom(ctx)

	if controllerutil.AddFinalizer(instance, common.FinalizerName) {
		if err := r.client.Update(ctx, instance); err != nil {
			return ctrl.Result{}, fmt.Errorf("failed to add finalizer to KeycloakRealmIdentityProvider: %w", err)
		}
	}

	oldStatus := instance.Status

	if err := chain.MakeChain(kClient, r.client).Serve(ctx, instance, realmName); err != nil {
		log.Error(err, "An error has occurred while handling KeycloakRealmIdentityProvider")

		resultErr := fmt.Errorf("identity provider chain processing failed: %w", err)
		instance.Status.Value = resultErr.Error()

		if statusErr := r.updateStatus(ctx, instance, oldStatus); statusErr != nil {
			return reconcile.Result{}, fmt.Errorf("failed to update KeycloakRealmIdentityProvider status: %w", statusErr)
		}

		return reconcile.Result{}, resultErr
	}

	instance.Status.Value = common.StatusOK

	if err := r.updateStatus(ctx, instance, oldStatus); err != nil {
		return reconcile.Result{}, err
	}

	return reconcile.Result{
		RequeueAfter: successRequeueTime,
	}, nil
}

func (r *IdentityProviderReconciler) updateStatus(ctx context.Context, instance *keycloakApi.KeycloakRealmIdentityProvider, oldStatus keycloakApi.KeycloakRealmIdentityProviderStatus) error {
	if equality.Semantic.DeepEqual(&instance.Status, &oldStatus) {
		return nil
	}

	if err := r.client.Status().Update(ctx, instance); err != nil {
		return fmt.Errorf("failed to update KeycloakRealmIdentityProvider status: %w", err)
	}

	return nil
}

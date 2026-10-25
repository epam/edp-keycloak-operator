package keycloakclientscope

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
	"github.com/epam/edp-keycloak-operator/internal/controller/keycloakclientscope/chain"
	"github.com/epam/edp-keycloak-operator/pkg/client/keycloakapi"
)

const successRequeueTime = time.Minute * 10

// Deprecated: legacyFinalizerName is the old finalizer used before migration to common.FinalizerName.
// Kept to ensure existing resources carrying the old finalizer can be deleted cleanly.
const legacyFinalizerName = "keycloak.clientscope.operator.finalizer.name"

type Helper interface {
	SetRealmOwnerRef(ctx context.Context, object helper.ObjectWithRealmRef) error
	GetRealmNameFromRef(ctx context.Context, object helper.ObjectWithRealmRef) (string, error)
	CreateKeycloakClientFromRealmRef(ctx context.Context, object helper.ObjectWithRealmRef) (*keycloakapi.KeycloakClient, error)
}

func NewReconcile(k8sClient client.Client, controllerHelper Helper) *Reconcile {
	return &Reconcile{
		client: k8sClient,
		helper: controllerHelper,
	}
}

// Reconcile reconciles a KeycloakClientScope object.
type Reconcile struct {
	client client.Client
	helper Helper
}

func (r *Reconcile) SetupWithManager(mgr ctrl.Manager) error {
	if err := ctrl.NewControllerManagedBy(mgr).
		For(&keycloakApi.KeycloakClientScope{}).
		Complete(r); err != nil {
		return fmt.Errorf("failed to setup KeycloakClientScope controller: %w", err)
	}

	return nil
}

// +kubebuilder:rbac:groups=v1.edp.epam.com,namespace=placeholder,resources=keycloakclientscopes,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=v1.edp.epam.com,namespace=placeholder,resources=keycloakclientscopes/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=v1.edp.epam.com,namespace=placeholder,resources=keycloakclientscopes/finalizers,verbs=update
// +kubebuilder:rbac:groups="",namespace=placeholder,resources=secrets,verbs=get;list;watch

// Reconcile is a loop for reconciling KeycloakClientScope object.
func (r *Reconcile) Reconcile(ctx context.Context, request reconcile.Request) (result reconcile.Result, resultErr error) {
	log := ctrl.LoggerFrom(ctx)
	log.Info("Reconciling KeycloakClientScope")

	instance := &keycloakApi.KeycloakClientScope{}
	if err := r.client.Get(ctx, request.NamespacedName, instance); err != nil {
		if k8sErrors.IsNotFound(err) {
			return reconcile.Result{}, nil
		}

		return reconcile.Result{}, fmt.Errorf("failed to get KeycloakClientScope: %w", err)
	}

	kClient, realmName, stop, err := r.initializeReconciliation(ctx, instance)
	if err != nil {
		if errors.Is(err, helper.ErrKeycloakIsNotAvailable) {
			return ctrl.Result{RequeueAfter: helper.RequeueOnKeycloakNotAvailablePeriod}, nil
		}

		return reconcile.Result{}, r.updateErrorStatus(ctx, instance, err)
	}

	if stop {
		return reconcile.Result{}, nil
	}

	if instance.GetDeletionTimestamp() != nil {
		return r.handleDeletion(ctx, instance, kClient, realmName)
	}

	return r.handleReconciliation(ctx, instance, kClient, realmName)
}

// initializeReconciliation returns stop=true when the finalizers were removed because the realm is gone.
func (r *Reconcile) initializeReconciliation(ctx context.Context, instance *keycloakApi.KeycloakClientScope) (*keycloakapi.KeycloakClient, string, bool, error) {
	if err := r.helper.SetRealmOwnerRef(ctx, instance); err != nil {
		return nil, "", false, fmt.Errorf("unable to set realm owner ref: %w", err)
	}

	kClient, err := r.helper.CreateKeycloakClientFromRealmRef(ctx, instance)
	if err != nil {
		if errors.Is(err, helper.ErrKeycloakRealmNotFound) && instance.GetDeletionTimestamp() != nil {
			stop, removeErr := helper.RemoveFinalizersOnRealmNotFound(ctx, r.client, instance, common.FinalizerName, legacyFinalizerName)
			if removeErr != nil {
				return nil, "", false, removeErr
			}

			if stop {
				return nil, "", true, nil
			}
		}

		return nil, "", false, fmt.Errorf("failed to create Keycloak client: %w", err)
	}

	realmName, err := r.helper.GetRealmNameFromRef(ctx, instance)
	if err != nil {
		return nil, "", false, fmt.Errorf("unable to get realm name from ref: %w", err)
	}

	return kClient, realmName, false, nil
}

func (r *Reconcile) handleDeletion(ctx context.Context, instance *keycloakApi.KeycloakClientScope, kClient *keycloakapi.KeycloakClient, realmName string) (reconcile.Result, error) {
	if controllerutil.ContainsFinalizer(instance, common.FinalizerName) || controllerutil.ContainsFinalizer(instance, legacyFinalizerName) {
		if err := chain.NewRemoveScope(kClient).Serve(ctx, instance, realmName); err != nil {
			return ctrl.Result{}, r.updateErrorStatus(ctx, instance, fmt.Errorf("failed to remove client scope: %w", err))
		}

		original := instance.DeepCopy()
		controllerutil.RemoveFinalizer(instance, common.FinalizerName)
		controllerutil.RemoveFinalizer(instance, legacyFinalizerName)

		if err := helper.PatchObject(ctx, r.client, original, instance); err != nil {
			return ctrl.Result{}, r.updateErrorStatus(ctx, instance, fmt.Errorf("failed to update KeycloakClientScope after finalizer removal: %w", err))
		}
	}

	return ctrl.Result{}, nil
}

func (r *Reconcile) handleReconciliation(ctx context.Context, instance *keycloakApi.KeycloakClientScope, kClient *keycloakapi.KeycloakClient, realmName string) (reconcile.Result, error) {
	log := ctrl.LoggerFrom(ctx)

	original := instance.DeepCopy()
	if controllerutil.AddFinalizer(instance, common.FinalizerName) {
		if err := helper.PatchObject(ctx, r.client, original, instance); err != nil {
			return ctrl.Result{}, r.updateErrorStatus(ctx, instance, fmt.Errorf("failed to add finalizer to KeycloakClientScope: %w", err))
		}
	}

	oldStatus := instance.Status

	if err := chain.MakeChain(kClient).Serve(ctx, instance, realmName); err != nil {
		log.Error(err, "An error has occurred while handling KeycloakClientScope")

		resultErr := fmt.Errorf("client scope chain processing failed: %w", err)
		instance.Status.Value = resultErr.Error()

		if statusErr := r.updateClientScopeStatus(ctx, instance, oldStatus); statusErr != nil {
			return reconcile.Result{}, fmt.Errorf("failed to update KeycloakClientScope status: %w", statusErr)
		}

		return reconcile.Result{}, resultErr
	}

	instance.Status.Value = common.StatusOK
	instance.Status.ObservedGeneration = instance.Generation

	if err := r.updateClientScopeStatus(ctx, instance, oldStatus); err != nil {
		return reconcile.Result{}, err
	}

	return reconcile.Result{
		RequeueAfter: successRequeueTime,
	}, nil
}

func (r *Reconcile) updateClientScopeStatus(ctx context.Context, instance *keycloakApi.KeycloakClientScope, oldStatus keycloakApi.KeycloakClientScopeStatus) error {
	if equality.Semantic.DeepEqual(&instance.Status, &oldStatus) {
		return nil
	}

	if err := r.client.Status().Update(ctx, instance); err != nil {
		return fmt.Errorf("failed to update KeycloakClientScope status: %w", err)
	}

	return nil
}

// updateErrorStatus writes err to status.value and returns err, or the status write error.
func (r *Reconcile) updateErrorStatus(ctx context.Context, instance *keycloakApi.KeycloakClientScope, err error) error {
	ctrl.LoggerFrom(ctx).Error(err, "An error has occurred while handling KeycloakClientScope")

	oldStatus := instance.Status
	instance.Status.Value = err.Error()

	if statusErr := r.updateClientScopeStatus(ctx, instance, oldStatus); statusErr != nil {
		return fmt.Errorf("failed to update KeycloakClientScope status: %w", statusErr)
	}

	return err
}

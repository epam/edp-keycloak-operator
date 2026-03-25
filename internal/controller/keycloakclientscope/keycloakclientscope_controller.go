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
	keycloakv2 "github.com/epam/edp-keycloak-operator/pkg/client/keycloakv2"
)

const successRequeueTime = time.Minute * 10

// Deprecated: legacyFinalizerName is the old finalizer used before migration to common.FinalizerName.
// Kept to ensure existing resources carrying the old finalizer can be deleted cleanly.
const legacyFinalizerName = "keycloak.clientscope.operator.finalizer.name"

type Helper interface {
	SetRealmOwnerRef(ctx context.Context, object helper.ObjectWithRealmRef) error
	GetRealmNameFromRef(ctx context.Context, object helper.ObjectWithRealmRef) (string, error)
	CreateKeycloakClientV2FromRealmRef(ctx context.Context, object helper.ObjectWithRealmRef) (*keycloakv2.KeycloakClient, error)
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

	instance, kClientV2, realmName, err := r.initializeReconciliation(ctx, request)
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
		return r.handleDeletion(ctx, instance, kClientV2, realmName)
	}

	return r.handleReconciliation(ctx, instance, kClientV2, realmName)
}

func (r *Reconcile) initializeReconciliation(ctx context.Context, request reconcile.Request) (*keycloakApi.KeycloakClientScope, *keycloakv2.KeycloakClient, string, error) {
	log := ctrl.LoggerFrom(ctx)

	instance := &keycloakApi.KeycloakClientScope{}
	if err := r.client.Get(ctx, request.NamespacedName, instance); err != nil {
		if k8sErrors.IsNotFound(err) {
			return nil, nil, "", nil
		}

		return nil, nil, "", fmt.Errorf("failed to get KeycloakClientScope: %w", err)
	}

	if err := r.helper.SetRealmOwnerRef(ctx, instance); err != nil {
		return nil, nil, "", fmt.Errorf("unable to set realm owner ref: %w", err)
	}

	kClientV2, err := r.helper.CreateKeycloakClientV2FromRealmRef(ctx, instance)
	if err != nil {
		if errors.Is(err, helper.ErrKeycloakRealmNotFound) {
			if instance.GetDeletionTimestamp() != nil {
				log.Info("Keycloak realm not found, removing finalizers")

				removed := controllerutil.RemoveFinalizer(instance, common.FinalizerName)
				removed = controllerutil.RemoveFinalizer(instance, legacyFinalizerName) || removed

				if removed {
					if updateErr := r.client.Update(ctx, instance); updateErr != nil {
						return nil, nil, "", fmt.Errorf("failed to remove finalizers: %w", updateErr)
					}
				}

				log.Info("Finalizers removed")

				return nil, nil, "", nil
			}
		}

		return nil, nil, "", fmt.Errorf("failed to create Keycloak client: %w", err)
	}

	realmName, err := r.helper.GetRealmNameFromRef(ctx, instance)
	if err != nil {
		return nil, nil, "", fmt.Errorf("unable to get realm name from ref: %w", err)
	}

	return instance, kClientV2, realmName, nil
}

func (r *Reconcile) handleDeletion(ctx context.Context, instance *keycloakApi.KeycloakClientScope, kClientV2 *keycloakv2.KeycloakClient, realmName string) (reconcile.Result, error) {
	if controllerutil.ContainsFinalizer(instance, common.FinalizerName) || controllerutil.ContainsFinalizer(instance, legacyFinalizerName) {
		if err := chain.NewRemoveScope(kClientV2).Serve(ctx, instance, realmName); err != nil {
			return ctrl.Result{}, fmt.Errorf("failed to remove client scope: %w", err)
		}

		controllerutil.RemoveFinalizer(instance, common.FinalizerName)
		controllerutil.RemoveFinalizer(instance, legacyFinalizerName)

		if err := r.client.Update(ctx, instance); err != nil {
			return ctrl.Result{}, fmt.Errorf("failed to update KeycloakClientScope after finalizer removal: %w", err)
		}
	}

	return ctrl.Result{}, nil
}

func (r *Reconcile) handleReconciliation(ctx context.Context, instance *keycloakApi.KeycloakClientScope, kClientV2 *keycloakv2.KeycloakClient, realmName string) (reconcile.Result, error) {
	log := ctrl.LoggerFrom(ctx)

	if controllerutil.AddFinalizer(instance, common.FinalizerName) {
		if err := r.client.Update(ctx, instance); err != nil {
			return ctrl.Result{}, fmt.Errorf("failed to add finalizer to KeycloakClientScope: %w", err)
		}
	}

	oldStatus := instance.Status

	if err := chain.MakeChain(kClientV2).Serve(ctx, instance, realmName); err != nil {
		log.Error(err, "An error has occurred while handling KeycloakClientScope")

		resultErr := fmt.Errorf("client scope chain processing failed: %w", err)
		instance.Status.Value = resultErr.Error()

		if statusErr := r.updateClientScopeStatus(ctx, instance, oldStatus); statusErr != nil {
			return reconcile.Result{}, fmt.Errorf("failed to update KeycloakClientScope status: %w", statusErr)
		}

		return reconcile.Result{}, resultErr
	}

	instance.Status.Value = common.StatusOK

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

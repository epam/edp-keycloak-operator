package keycloakrealmidentityprovider

import (
	"context"
	"fmt"
	"time"

	"k8s.io/apimachinery/pkg/api/equality"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	"github.com/epam/edp-keycloak-operator/api/common"
	keycloakApi "github.com/epam/edp-keycloak-operator/api/v1"
	"github.com/epam/edp-keycloak-operator/internal/controller/helper"
	"github.com/epam/edp-keycloak-operator/internal/controller/keycloakrealmidentityprovider/chain"
	"github.com/epam/edp-keycloak-operator/pkg/client/keycloakapi"
	"github.com/epam/edp-keycloak-operator/pkg/secretref"
)

const successRequeueTime = time.Minute * 10

const kind = "KeycloakRealmIdentityProvider"

// Deprecated: legacyFinalizerName is the old finalizer used before migration to common.FinalizerName.
// Kept to ensure existing resources carrying the old finalizer can be deleted cleanly.
const legacyFinalizerName = "keycloak.realmidp.operator.finalizer.name"

type IdentityProviderReconcilerCtrlHelper interface {
	SetRealmOwnerRef(ctx context.Context, object helper.ObjectWithRealmRef) error
	GetRealmNameFromRef(ctx context.Context, object helper.ObjectWithRealmRef) (string, error)
	CreateKeycloakClientFromRealmRef(ctx context.Context, object helper.ObjectWithRealmRef) (*keycloakapi.KeycloakClient, error)
}

type IdentityProviderReconciler struct {
	client    client.Client
	helper    IdentityProviderReconcilerCtrlHelper
	secretRef *secretref.SecretRef
}

func NewIdentityProviderReconciler(
	k8sClient client.Client,
	controllerHelper IdentityProviderReconcilerCtrlHelper,
	secretRef *secretref.SecretRef,
) *IdentityProviderReconciler {
	return &IdentityProviderReconciler{
		client:    k8sClient,
		helper:    controllerHelper,
		secretRef: secretRef,
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
func (r *IdentityProviderReconciler) Reconcile(ctx context.Context, request reconcile.Request) (reconcile.Result, error) {
	return helper.ReconcileRealmRef(ctx, r.client, request, &keycloakApi.KeycloakRealmIdentityProvider{}, helper.RealmRefReconcile[*keycloakApi.KeycloakRealmIdentityProvider]{
		Kind:                 kind,
		Helper:               r.helper,
		Finalizers:           []string{common.FinalizerName, legacyFinalizerName},
		Remove:               r.removeFromKeycloak,
		HandleReconciliation: r.handleReconciliation,
	})
}

func (r *IdentityProviderReconciler) removeFromKeycloak(ctx context.Context, instance *keycloakApi.KeycloakRealmIdentityProvider, kClient *keycloakapi.KeycloakClient, realmName string) error {
	if err := chain.NewRemoveIDP(kClient).Serve(ctx, instance, realmName); err != nil {
		return fmt.Errorf("failed to remove identity provider: %w", err)
	}

	return nil
}

func (r *IdentityProviderReconciler) handleReconciliation(ctx context.Context, instance *keycloakApi.KeycloakRealmIdentityProvider, kClient *keycloakapi.KeycloakClient, realmName string) (reconcile.Result, error) {
	log := ctrl.LoggerFrom(ctx)

	original := instance.DeepCopy()
	if controllerutil.AddFinalizer(instance, common.FinalizerName) {
		if err := helper.PatchObject(ctx, r.client, original, instance); err != nil {
			return ctrl.Result{}, helper.UpdateErrorStatus(ctx, r.client, kind, instance, fmt.Errorf("failed to add finalizer to KeycloakRealmIdentityProvider: %w", err))
		}
	}

	oldStatus := instance.Status

	if err := chain.MakeChain(kClient, r.client, r.secretRef).Serve(ctx, instance, realmName); err != nil {
		log.Error(err, "An error has occurred while handling KeycloakRealmIdentityProvider")

		resultErr := fmt.Errorf("identity provider chain processing failed: %w", err)
		instance.Status.Value = resultErr.Error()

		if statusErr := r.updateStatus(ctx, instance, oldStatus); statusErr != nil {
			return reconcile.Result{}, fmt.Errorf("failed to update KeycloakRealmIdentityProvider status: %w", statusErr)
		}

		return reconcile.Result{}, resultErr
	}

	instance.Status.Value = common.StatusOK
	instance.Status.ObservedGeneration = instance.Generation

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

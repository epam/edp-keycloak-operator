package keycloakclientscope

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
	"github.com/epam/edp-keycloak-operator/internal/controller/keycloakclientscope/chain"
	"github.com/epam/edp-keycloak-operator/pkg/client/keycloakapi"
)

const successRequeueTime = time.Minute * 10

const kind = "KeycloakClientScope"

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
func (r *Reconcile) Reconcile(ctx context.Context, request reconcile.Request) (reconcile.Result, error) {
	return helper.ReconcileRealmRef(ctx, r.client, request, &keycloakApi.KeycloakClientScope{}, helper.RealmRefReconcile[*keycloakApi.KeycloakClientScope]{
		Kind:                 kind,
		Helper:               r.helper,
		Finalizers:           []string{common.FinalizerName, legacyFinalizerName},
		Remove:               r.removeFromKeycloak,
		HandleReconciliation: r.handleReconciliation,
	})
}

func (r *Reconcile) removeFromKeycloak(ctx context.Context, instance *keycloakApi.KeycloakClientScope, kClient *keycloakapi.KeycloakClient, realmName string) error {
	if err := chain.NewRemoveScope(kClient).Serve(ctx, instance, realmName); err != nil {
		return fmt.Errorf("failed to remove client scope: %w", err)
	}

	return nil
}

func (r *Reconcile) handleReconciliation(ctx context.Context, instance *keycloakApi.KeycloakClientScope, kClient *keycloakapi.KeycloakClient, realmName string) (reconcile.Result, error) {
	log := ctrl.LoggerFrom(ctx)

	original := instance.DeepCopy()
	if controllerutil.AddFinalizer(instance, common.FinalizerName) {
		if err := helper.PatchObject(ctx, r.client, original, instance); err != nil {
			return ctrl.Result{}, helper.UpdateErrorStatus(ctx, r.client, kind, instance, fmt.Errorf("failed to add finalizer to KeycloakClientScope: %w", err))
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

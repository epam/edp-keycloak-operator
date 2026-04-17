package keycloakrealmrole

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"time"

	k8sErrors "k8s.io/apimachinery/pkg/api/errors"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	keycloakApi "github.com/epam/edp-keycloak-operator/api/v1"
	"github.com/epam/edp-keycloak-operator/internal/controller/helper"
	"github.com/epam/edp-keycloak-operator/internal/controller/keycloakrealmrole/chain"
	keycloakapi "github.com/epam/edp-keycloak-operator/pkg/client/keycloakapi"
)

const keyCloakRealmRoleOperatorFinalizerName = "keycloak.realmrole.operator.finalizer.name"

type Helper interface {
	SetFailureCount(fc helper.FailureCountable) time.Duration
	TryRemoveFinalizer(ctx context.Context, obj client.Object, finalizer string) error
	SetRealmOwnerRef(ctx context.Context, object helper.ObjectWithRealmRef) error
	GetRealmNameFromRef(ctx context.Context, object helper.ObjectWithRealmRef) (string, error)
	CreateKeycloakeycloakAPIClientFromRealmRef(ctx context.Context, object helper.ObjectWithRealmRef) (*keycloakapi.APIClient, error)
}

func NewReconcileKeycloakRealmRole(k8sClient client.Client, controllerHelper Helper) *ReconcileKeycloakRealmRole {
	return &ReconcileKeycloakRealmRole{
		client: k8sClient,
		helper: controllerHelper,
	}
}

type ReconcileKeycloakRealmRole struct {
	client                  client.Client
	helper                  Helper
	successReconcileTimeout time.Duration
}

func (r *ReconcileKeycloakRealmRole) SetupWithManager(mgr ctrl.Manager, successReconcileTimeout time.Duration) error {
	r.successReconcileTimeout = successReconcileTimeout

	pred := predicate.Funcs{
		UpdateFunc: isSpecUpdated,
	}

	err := ctrl.NewControllerManagedBy(mgr).
		For(&keycloakApi.KeycloakRealmRole{}, builder.WithPredicates(pred)).
		Complete(r)
	if err != nil {
		return fmt.Errorf("failed to setup KeycloakRealmRole controller: %w", err)
	}

	return nil
}

func isSpecUpdated(e event.UpdateEvent) bool {
	oo, ok := e.ObjectOld.(*keycloakApi.KeycloakRealmRole)
	if !ok {
		return false
	}

	no, ok := e.ObjectNew.(*keycloakApi.KeycloakRealmRole)
	if !ok {
		return false
	}

	return !reflect.DeepEqual(oo.Spec, no.Spec) ||
		(oo.GetDeletionTimestamp().IsZero() && !no.GetDeletionTimestamp().IsZero())
}

// +kubebuilder:rbac:groups=v1.edp.epam.com,namespace=placeholder,resources=keycloakrealmroles,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=v1.edp.epam.com,namespace=placeholder,resources=keycloakrealmroles/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=v1.edp.epam.com,namespace=placeholder,resources=keycloakrealmroles/finalizers,verbs=update

// Reconcile is a loop for reconciling KeycloakRealmRole object.
func (r *ReconcileKeycloakRealmRole) Reconcile(ctx context.Context, request reconcile.Request) (result reconcile.Result, resultErr error) {
	log := ctrl.LoggerFrom(ctx)
	log.Info("Reconciling KeycloakRealmRole")

	var instance keycloakApi.KeycloakRealmRole
	if err := r.client.Get(ctx, request.NamespacedName, &instance); err != nil {
		if k8sErrors.IsNotFound(err) {
			return result, resultErr
		}

		resultErr = fmt.Errorf("unable to get keycloak realm role from k8s: %w", err)

		return result, resultErr
	}

	defer func() {
		if err := r.client.Status().Update(ctx, &instance); err != nil {
			resultErr = err
		}
	}()

	roleID, err := r.tryReconcile(ctx, &instance)
	if err != nil {
		if errors.Is(err, helper.ErrKeycloakIsNotAvailable) {
			return ctrl.Result{
				RequeueAfter: helper.RequeueOnKeycloakNotAvailablePeriod,
			}, nil
		}

		instance.Status.Value = err.Error()
		result.RequeueAfter = r.helper.SetFailureCount(&instance)

		log.Error(err, "an error has occurred while handling keycloak realm role", "name",
			request.Name)

		return result, resultErr
	}

	helper.SetSuccessStatus(&instance)
	instance.Status.ID = roleID
	result.RequeueAfter = r.successReconcileTimeout

	log.Info("Reconciling done")

	return result, resultErr
}

func (r *ReconcileKeycloakRealmRole) tryReconcile(ctx context.Context, keycloakRealmRole *keycloakApi.KeycloakRealmRole) (string, error) {
	err := r.helper.SetRealmOwnerRef(ctx, keycloakRealmRole)
	if err != nil {
		return "", fmt.Errorf("unable to set realm owner ref: %w", err)
	}

	kClient, err := r.helper.CreateKeycloakeycloakAPIClientFromRealmRef(ctx, keycloakRealmRole)
	if err != nil {
		if errors.Is(err, helper.ErrKeycloakRealmNotFound) {
			if removeErr := r.helper.TryRemoveFinalizer(ctx, keycloakRealmRole, keyCloakRealmRoleOperatorFinalizerName); removeErr != nil {
				return "", fmt.Errorf("unable to remove finalizer: %w", removeErr)
			}

			return "", nil
		}

		return "", fmt.Errorf("unable to create keycloak client from realm ref: %w", err)
	}

	realmName, err := r.helper.GetRealmNameFromRef(ctx, keycloakRealmRole)
	if err != nil {
		return "", fmt.Errorf("unable to get realm name from ref: %w", err)
	}

	if keycloakRealmRole.GetDeletionTimestamp() != nil {
		if controllerutil.ContainsFinalizer(keycloakRealmRole, keyCloakRealmRoleOperatorFinalizerName) {
			if err := chain.NewRemoveRole(kClient).ServeRequest(ctx, keycloakRealmRole, realmName); err != nil {
				return "", fmt.Errorf("failed to remove role: %w", err)
			}

			controllerutil.RemoveFinalizer(keycloakRealmRole, keyCloakRealmRoleOperatorFinalizerName)

			if err := r.client.Update(ctx, keycloakRealmRole); err != nil {
				return "", fmt.Errorf("failed to remove finalizer: %w", err)
			}
		}

		return "", nil
	}

	if controllerutil.AddFinalizer(keycloakRealmRole, keyCloakRealmRoleOperatorFinalizerName) {
		if err := r.client.Update(ctx, keycloakRealmRole); err != nil {
			return "", fmt.Errorf("failed to add finalizer: %w", err)
		}
	}

	roleCtx := &chain.RoleContext{}
	if err := chain.MakeChain(kClient).Serve(ctx, keycloakRealmRole, realmName, roleCtx); err != nil {
		return "", fmt.Errorf("error during realm role chain: %w", err)
	}

	return roleCtx.RoleID, nil
}

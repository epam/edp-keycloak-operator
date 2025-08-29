package keycloakrealmrole

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"time"

	"github.com/Nerzal/gocloak/v12"
	k8sErrors "k8s.io/apimachinery/pkg/api/errors"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	keycloakApi "github.com/epam/edp-keycloak-operator/api/v1"
	"github.com/epam/edp-keycloak-operator/internal/controller/helper"
	"github.com/epam/edp-keycloak-operator/pkg/client/keycloak"
	"github.com/epam/edp-keycloak-operator/pkg/client/keycloak/dto"
	"github.com/epam/edp-keycloak-operator/pkg/objectmeta"
)

const keyCloakRealmRoleOperatorFinalizerName = "keycloak.realmrole.operator.finalizer.name"

type Helper interface {
	SetFailureCount(fc helper.FailureCountable) time.Duration
	TryRemoveFinalizer(ctx context.Context, obj client.Object, finalizer string) error
	TryToDelete(ctx context.Context, obj client.Object, terminator helper.Terminator, finalizer string) (isDeleted bool, resultErr error)
	SetRealmOwnerRef(ctx context.Context, object helper.ObjectWithRealmRef) error
	GetKeycloakRealmFromRef(ctx context.Context, object helper.ObjectWithRealmRef, kcClient keycloak.Client) (*gocloak.RealmRepresentation, error)
	CreateKeycloakClientFromRealmRef(ctx context.Context, object helper.ObjectWithRealmRef) (keycloak.Client, error)
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

	kClient, err := r.helper.CreateKeycloakClientFromRealmRef(ctx, keycloakRealmRole)
	if err != nil {
		// if the realm is already deleted try to delete finalizer
		if errors.Is(err, helper.ErrKeycloakRealmNotFound) {
			if removeErr := r.helper.TryRemoveFinalizer(ctx, keycloakRealmRole, keyCloakRealmRoleOperatorFinalizerName); removeErr != nil {
				return "", fmt.Errorf("unable to remove finalizer: %w", removeErr)
			}

			return "", nil
		}

		return "", fmt.Errorf("unable to create keycloak client from realm ref: %w", err)
	}

	realm, err := r.helper.GetKeycloakRealmFromRef(ctx, keycloakRealmRole, kClient)
	if err != nil {
		return "", fmt.Errorf("unable to get keycloak realm from ref: %w", err)
	}

	roleID, err := r.putRole(ctx, gocloak.PString(realm.Realm), keycloakRealmRole, kClient)
	if err != nil {
		return "", fmt.Errorf("unable to put role: %w", err)
	}

	if _, err := r.helper.TryToDelete(
		ctx,
		keycloakRealmRole,
		makeTerminator(
			gocloak.PString(realm.Realm),
			keycloakRealmRole.Spec.Name,
			kClient,
			objectmeta.PreserveResourcesOnDeletion(keycloakRealmRole),
		),
		keyCloakRealmRoleOperatorFinalizerName,
	); err != nil {
		return "", fmt.Errorf("unable to tryToDelete realm role: %w", err)
	}

	return roleID, nil
}

func (r *ReconcileKeycloakRealmRole) putRole(
	ctx context.Context,
	realmName string,
	keycloakRealmRole *keycloakApi.KeycloakRealmRole,
	kClient keycloak.Client,
) (string, error) {
	log := ctrl.LoggerFrom(ctx)
	log.Info("Start creating realm role")

	role := dto.ConvertSpecToRole(keycloakRealmRole)

	if err := kClient.SyncRealmRole(ctx, realmName, role); err != nil {
		return "", fmt.Errorf("unable to sync realm role CR: %w", err)
	}

	var roleID string

	if role.ID != nil {
		roleID = *role.ID
	}

	log.Info("Realm role has been created")

	return roleID, nil
}

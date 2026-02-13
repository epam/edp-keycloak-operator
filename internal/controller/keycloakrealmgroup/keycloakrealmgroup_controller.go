package keycloakrealmgroup

import (
	"context"
	"errors"
	"fmt"
	"time"

	k8sErrors "k8s.io/apimachinery/pkg/api/errors"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	keycloakApi "github.com/epam/edp-keycloak-operator/api/v1"
	"github.com/epam/edp-keycloak-operator/internal/controller/helper"
	"github.com/epam/edp-keycloak-operator/internal/controller/keycloakrealmgroup/chain"
	keycloakv2 "github.com/epam/edp-keycloak-operator/pkg/client/keycloakv2"
	"github.com/epam/edp-keycloak-operator/pkg/objectmeta"
)

const keyCloakRealmGroupOperatorFinalizerName = "keycloak.realmgroup.operator.finalizer.name"

type Helper interface {
	SetFailureCount(fc helper.FailureCountable) time.Duration
	TryRemoveFinalizer(ctx context.Context, obj client.Object, finalizer string) error
	TryToDelete(ctx context.Context, obj client.Object, terminator helper.Terminator, finalizer string) (isDeleted bool, resultErr error)
	SetRealmOwnerRef(ctx context.Context, object helper.ObjectWithRealmRef) error
	CreateKeycloakClientV2FromRealmRef(ctx context.Context, object helper.ObjectWithRealmRef) (*keycloakv2.KeycloakClient, error)
	GetRealmNameFromRef(ctx context.Context, object helper.ObjectWithRealmRef) (string, error)
}

func NewReconcileKeycloakRealmGroup(
	k8sClient client.Client,
	controllerHelper Helper,
) *ReconcileKeycloakRealmGroup {
	return &ReconcileKeycloakRealmGroup{
		client: k8sClient,
		helper: controllerHelper,
	}
}

type ReconcileKeycloakRealmGroup struct {
	client                  client.Client
	helper                  Helper
	successReconcileTimeout time.Duration
}

func (r *ReconcileKeycloakRealmGroup) SetupWithManager(mgr ctrl.Manager, successReconcileTimeout time.Duration) error {
	r.successReconcileTimeout = successReconcileTimeout

	pred := predicate.Funcs{
		UpdateFunc: helper.IsFailuresUpdated,
	}

	err := ctrl.NewControllerManagedBy(mgr).
		For(&keycloakApi.KeycloakRealmGroup{}, builder.WithPredicates(pred)).
		Complete(r)
	if err != nil {
		return fmt.Errorf("failed to setup KeycloakRealmGroup controller: %w", err)
	}

	return nil
}

// +kubebuilder:rbac:groups=v1.edp.epam.com,namespace=placeholder,resources=keycloakrealmgroups,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=v1.edp.epam.com,namespace=placeholder,resources=keycloakrealmgroups/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=v1.edp.epam.com,namespace=placeholder,resources=keycloakrealmgroups/finalizers,verbs=update

// Reconcile is a loop for reconciling KeycloakRealmGroup object.
func (r *ReconcileKeycloakRealmGroup) Reconcile(ctx context.Context, request reconcile.Request) (result reconcile.Result, resultErr error) {
	log := ctrl.LoggerFrom(ctx)
	log.Info("Reconciling KeycloakRealmGroup")

	var instance keycloakApi.KeycloakRealmGroup
	if err := r.client.Get(ctx, request.NamespacedName, &instance); err != nil {
		if k8sErrors.IsNotFound(err) {
			return result, resultErr
		}

		resultErr = fmt.Errorf("unable to get keycloak realm group from k8s: %w", err)

		return result, resultErr
	}

	if err := r.tryReconcile(ctx, &instance); err != nil {
		if errors.Is(err, helper.ErrKeycloakIsNotAvailable) {
			return ctrl.Result{
				RequeueAfter: helper.RequeueOnKeycloakNotAvailablePeriod,
			}, nil
		}

		instance.Status.Value = err.Error()
		result.RequeueAfter = r.helper.SetFailureCount(&instance)

		log.Error(err, "an error has occurred while handling keycloak realm group", "name", request.Name)
	} else {
		helper.SetSuccessStatus(&instance)

		result.RequeueAfter = r.successReconcileTimeout
	}

	if err := r.client.Status().Update(ctx, &instance); err != nil {
		resultErr = fmt.Errorf("unable to update status: %w", err)
	}

	log.Info("Reconciling done")

	return result, resultErr
}

func (r *ReconcileKeycloakRealmGroup) tryReconcile(ctx context.Context, keycloakRealmGroup *keycloakApi.KeycloakRealmGroup) error {
	// TODO: Move this validation to a validating webhook when webhook is configured.
	// Validate that SubGroups and ParentGroup are not used together.
	if len(keycloakRealmGroup.Spec.SubGroups) > 0 && keycloakRealmGroup.Spec.ParentGroup != nil {
		return fmt.Errorf("cannot use both SubGroups (deprecated) and ParentGroup fields - migrate to ParentGroup approach")
	}

	err := r.helper.SetRealmOwnerRef(ctx, keycloakRealmGroup)
	if err != nil {
		return fmt.Errorf("unable to set realm owner ref: %w", err)
	}

	kClientV2, err := r.helper.CreateKeycloakClientV2FromRealmRef(ctx, keycloakRealmGroup)
	if err != nil {
		if errors.Is(err, helper.ErrKeycloakRealmNotFound) {
			if removeErr := r.helper.TryRemoveFinalizer(ctx, keycloakRealmGroup, keyCloakRealmGroupOperatorFinalizerName); removeErr != nil {
				return fmt.Errorf("unable to remove finalizer: %w", removeErr)
			}

			return nil
		}

		return fmt.Errorf("unable to create keycloak v2 client from realm ref: %w", err)
	}

	realmName, err := r.helper.GetRealmNameFromRef(ctx, keycloakRealmGroup)
	if err != nil {
		return fmt.Errorf("unable to get realm name from ref: %w", err)
	}

	deleted, err := r.helper.TryToDelete(
		ctx,
		keycloakRealmGroup,
		makeTerminator(
			kClientV2,
			realmName,
			keycloakRealmGroup.Status.ID,
			keycloakRealmGroup.Spec.Name,
			objectmeta.PreserveResourcesOnDeletion(keycloakRealmGroup),
		),
		keyCloakRealmGroupOperatorFinalizerName,
	)
	if err != nil {
		return fmt.Errorf("failed to delete keycloak realm group: %w", err)
	}

	if deleted {
		return nil
	}

	parentGroupID, err := r.resolveParentGroupID(ctx, keycloakRealmGroup)
	if err != nil {
		return err
	}

	groupCtx := &chain.GroupContext{
		RealmName:     realmName,
		ParentGroupID: parentGroupID,
	}

	if err := chain.MakeChain().Serve(ctx, keycloakRealmGroup, kClientV2, groupCtx); err != nil {
		return fmt.Errorf("error during realm group chain: %w", err)
	}

	keycloakRealmGroup.Status.ID = groupCtx.GroupID

	return nil
}

// resolveParentGroupID resolves the parent group's Keycloak ID from the ParentGroup reference.
// Returns empty string if no parent is specified.
// Returns error if parent CR doesn't exist or is not ready yet (no ID in status).
func (r *ReconcileKeycloakRealmGroup) resolveParentGroupID(
	ctx context.Context,
	keycloakRealmGroup *keycloakApi.KeycloakRealmGroup,
) (string, error) {
	// No parent group specified
	if keycloakRealmGroup.Spec.ParentGroup == nil {
		return "", nil
	}

	log := ctrl.LoggerFrom(ctx)

	// Fetch the parent KeycloakRealmGroup CR
	parentGroupCR := &keycloakApi.KeycloakRealmGroup{}
	if err := r.client.Get(ctx, client.ObjectKey{
		Name:      keycloakRealmGroup.Spec.ParentGroup.Name,
		Namespace: keycloakRealmGroup.Namespace,
	}, parentGroupCR); err != nil {
		return "", fmt.Errorf("unable to get parent KeycloakRealmGroup %s: %w",
			keycloakRealmGroup.Spec.ParentGroup.Name, err)
	}

	// Check if parent group has been reconciled and has an ID
	if parentGroupCR.Status.ID == "" {
		log.Info("Parent group not ready yet, skipping reconciliation",
			"parentGroup", keycloakRealmGroup.Spec.ParentGroup.Name,
			"childGroup", keycloakRealmGroup.Name)

		return "", fmt.Errorf("parent group %s is not ready yet (no ID in status)",
			keycloakRealmGroup.Spec.ParentGroup.Name)
	}

	return parentGroupCR.Status.ID, nil
}

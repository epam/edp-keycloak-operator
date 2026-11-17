package keycloakrealmgroupmembership

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
	"github.com/epam/edp-keycloak-operator/internal/controller/keycloakrealmgroupmembership/chain"
	"github.com/epam/edp-keycloak-operator/pkg/client/keycloakapi"
)

const (
	finalizerName = "keycloak.realmgroupmembership.operator.finalizer.name"

	// pendingMembersRequeuePeriod is the longest delay before pending members are looked up again.
	pendingMembersRequeuePeriod = time.Minute
)

type Helper interface {
	SetFailureCount(fc helper.FailureCountable) time.Duration
	TryRemoveFinalizer(ctx context.Context, obj client.Object, finalizer string) error
	SetRealmOwnerRef(ctx context.Context, object helper.ObjectWithRealmRef) error
	GetRealmNameFromRef(ctx context.Context, object helper.ObjectWithRealmRef) (string, error)
	CreateKeycloakClientFromRealmRef(ctx context.Context, object helper.ObjectWithRealmRef) (*keycloakapi.KeycloakClient, error)
}

func NewReconcileKeycloakRealmGroupMembership(
	k8sClient client.Client,
	controllerHelper Helper,
) *ReconcileKeycloakRealmGroupMembership {
	return &ReconcileKeycloakRealmGroupMembership{
		client:               k8sClient,
		helper:               controllerHelper,
		pendingRequeuePeriod: pendingMembersRequeuePeriod,
	}
}

type ReconcileKeycloakRealmGroupMembership struct {
	client                  client.Client
	helper                  Helper
	successReconcileTimeout time.Duration
	pendingRequeuePeriod    time.Duration
}

func (r *ReconcileKeycloakRealmGroupMembership) SetupWithManager(mgr ctrl.Manager, successReconcileTimeout time.Duration) error {
	r.successReconcileTimeout = successReconcileTimeout

	pred := predicate.Funcs{
		UpdateFunc: isSpecUpdated,
	}

	err := ctrl.NewControllerManagedBy(mgr).
		For(&keycloakApi.KeycloakRealmGroupMembership{}, builder.WithPredicates(pred)).
		Complete(r)
	if err != nil {
		return fmt.Errorf("failed to setup KeycloakRealmGroupMembership controller: %w", err)
	}

	return nil
}

func isSpecUpdated(e event.UpdateEvent) bool {
	oo, ok := e.ObjectOld.(*keycloakApi.KeycloakRealmGroupMembership)
	if !ok {
		return false
	}

	no, ok := e.ObjectNew.(*keycloakApi.KeycloakRealmGroupMembership)
	if !ok {
		return false
	}

	return !reflect.DeepEqual(oo.Spec, no.Spec) ||
		(oo.GetDeletionTimestamp().IsZero() && !no.GetDeletionTimestamp().IsZero())
}

// +kubebuilder:rbac:groups=v1.edp.epam.com,namespace=placeholder,resources=keycloakrealmgroupmemberships,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=v1.edp.epam.com,namespace=placeholder,resources=keycloakrealmgroupmemberships/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=v1.edp.epam.com,namespace=placeholder,resources=keycloakrealmgroupmemberships/finalizers,verbs=update

// Reconcile is a loop for reconciling KeycloakRealmGroupMembership object.
func (r *ReconcileKeycloakRealmGroupMembership) Reconcile(
	ctx context.Context,
	request reconcile.Request,
) (result reconcile.Result, resultErr error) {
	log := ctrl.LoggerFrom(ctx)
	log.Info("Reconciling KeycloakRealmGroupMembership")

	var instance keycloakApi.KeycloakRealmGroupMembership
	if err := r.client.Get(ctx, request.NamespacedName, &instance); err != nil {
		if k8sErrors.IsNotFound(err) {
			return result, resultErr
		}

		resultErr = fmt.Errorf("unable to get keycloak realm group membership from k8s: %w", err)

		return result, resultErr
	}

	membershipCtx, deleted, err := r.tryReconcile(ctx, &instance)
	if deleted {
		return ctrl.Result{}, nil
	}

	defer func() {
		if err := r.client.Status().Update(ctx, &instance); err != nil {
			resultErr = err
		}
	}()

	if err != nil {
		if errors.Is(err, helper.ErrKeycloakIsNotAvailable) {
			return ctrl.Result{
				RequeueAfter: helper.RequeueOnKeycloakNotAvailablePeriod,
			}, nil
		}

		instance.Status.Value = err.Error()
		result.RequeueAfter = r.helper.SetFailureCount(&instance)

		log.Error(err, "an error has occurred while handling keycloak realm group membership", "name",
			request.Name)

		return result, resultErr
	}

	helper.SetSuccessStatus(&instance)
	setMembersStatus(&instance, membershipCtx)

	result.RequeueAfter = r.successReconcileTimeout
	if len(instance.Status.PendingMembers) > 0 &&
		(result.RequeueAfter == 0 || result.RequeueAfter > r.pendingRequeuePeriod) {
		result.RequeueAfter = r.pendingRequeuePeriod
	}

	log.Info("Reconciling done")

	return result, resultErr
}

// tryReconcile returns the chain context, and deleted when the object is deleting and the operator has no cleanup left.
func (r *ReconcileKeycloakRealmGroupMembership) tryReconcile(
	ctx context.Context,
	membership *keycloakApi.KeycloakRealmGroupMembership,
) (membershipCtx *chain.MembershipContext, deleted bool, err error) {
	err = r.helper.SetRealmOwnerRef(ctx, membership)
	if err != nil {
		return nil, false, fmt.Errorf("unable to set realm owner ref: %w", err)
	}

	kClient, err := r.helper.CreateKeycloakClientFromRealmRef(ctx, membership)
	if err != nil {
		if errors.Is(err, helper.ErrKeycloakRealmNotFound) && membership.GetDeletionTimestamp() != nil {
			if removeErr := r.helper.TryRemoveFinalizer(ctx, membership, finalizerName); removeErr != nil {
				return nil, false, fmt.Errorf("unable to remove finalizer: %w", removeErr)
			}

			return nil, true, nil
		}

		return nil, false, fmt.Errorf("unable to create keycloak client from realm ref: %w", err)
	}

	realmName, err := r.helper.GetRealmNameFromRef(ctx, membership)
	if err != nil {
		return nil, false, fmt.Errorf("unable to get realm name from ref: %w", err)
	}

	if membership.GetDeletionTimestamp() != nil {
		if controllerutil.ContainsFinalizer(membership, finalizerName) {
			if err := chain.NewRemoveMembers(kClient).ServeRequest(ctx, membership, realmName); err != nil {
				return nil, false, fmt.Errorf("failed to remove group members: %w", err)
			}

			original := membership.DeepCopy()
			controllerutil.RemoveFinalizer(membership, finalizerName)

			if err := helper.PatchObject(ctx, r.client, original, membership); err != nil {
				return nil, false, fmt.Errorf("failed to remove finalizer: %w", err)
			}
		}

		return nil, true, nil
	}

	original := membership.DeepCopy()
	if controllerutil.AddFinalizer(membership, finalizerName) {
		if err := helper.PatchObject(ctx, r.client, original, membership); err != nil {
			return nil, false, fmt.Errorf("failed to add finalizer: %w", err)
		}
	}

	membershipCtx = &chain.MembershipContext{}
	if err := chain.MakeChain(kClient).Serve(ctx, membership, realmName, membershipCtx); err != nil {
		return nil, false, fmt.Errorf("error during realm group membership chain: %w", err)
	}

	return membershipCtx, false, nil
}

func setMembersStatus(membership *keycloakApi.KeycloakRealmGroupMembership, membershipCtx *chain.MembershipContext) {
	resolved := make([]keycloakApi.GroupMember, 0, len(membershipCtx.Resolved))
	for _, m := range membershipCtx.Resolved {
		resolved = append(resolved, m.Member)
	}

	membership.Status.ResolvedMembers = resolved
	membership.Status.PendingMembers = membershipCtx.Pending
}

package keycloakrealmrolebatch

import (
	"context"
	"fmt"
	"time"

	"github.com/Nerzal/gocloak/v12"
	"github.com/pkg/errors"
	k8sErrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	"github.com/epam/edp-keycloak-operator/api/common"
	keycloakApi "github.com/epam/edp-keycloak-operator/api/v1"
	"github.com/epam/edp-keycloak-operator/controllers/helper"
	"github.com/epam/edp-keycloak-operator/pkg/objectmeta"
)

const keyCloakRealmRoleBatchOperatorFinalizerName = "keycloak.realmrolebatch.operator.finalizer.name"

type Helper interface {
	TryToDelete(ctx context.Context, obj client.Object, terminator helper.Terminator, finalizer string) (isDeleted bool, resultErr error)
	SetRealmOwnerRef(ctx context.Context, object helper.ObjectWithRealmRef) error
	SetFailureCount(fc helper.FailureCountable) time.Duration
}

func NewReconcileKeycloakRealmRoleBatch(client client.Client, helper Helper) *ReconcileKeycloakRealmRoleBatch {
	return &ReconcileKeycloakRealmRoleBatch{
		client: client,
		helper: helper,
	}
}

type ReconcileKeycloakRealmRoleBatch struct {
	client                  client.Client
	helper                  Helper
	successReconcileTimeout time.Duration
}

func (r *ReconcileKeycloakRealmRoleBatch) SetupWithManager(mgr ctrl.Manager, successReconcileTimeout time.Duration) error {
	r.successReconcileTimeout = successReconcileTimeout

	pred := predicate.Funcs{
		UpdateFunc: helper.IsFailuresUpdated,
	}

	err := ctrl.NewControllerManagedBy(mgr).
		For(&keycloakApi.KeycloakRealmRoleBatch{}, builder.WithPredicates(pred)).
		Complete(r)
	if err != nil {
		return fmt.Errorf("failed to setup KeycloakRealmRoleBatch controller: %w", err)
	}

	return nil
}

//+kubebuilder:rbac:groups=v1.edp.epam.com,namespace=placeholder,resources=keycloakrealmrolebatches,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=v1.edp.epam.com,namespace=placeholder,resources=keycloakrealmrolebatches/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=v1.edp.epam.com,namespace=placeholder,resources=keycloakrealmrolebatches/finalizers,verbs=update

// Reconcile is a loop for reconciling KeycloakRealmRoleBatch object.
func (r *ReconcileKeycloakRealmRoleBatch) Reconcile(ctx context.Context, request reconcile.Request) (result reconcile.Result, resultErr error) {
	log := ctrl.LoggerFrom(ctx)
	log.Info("Reconciling KeycloakRealmRoleBatch")

	var instance keycloakApi.KeycloakRealmRoleBatch
	if err := r.client.Get(ctx, request.NamespacedName, &instance); err != nil {
		if k8sErrors.IsNotFound(err) {
			return
		}

		resultErr = errors.Wrap(err, "unable to get keycloak realm role batch from k8s")

		return
	}

	if updated, err := r.applyDefaults(ctx, &instance); err != nil {
		resultErr = fmt.Errorf("unable to apply default values: %w", err)
		return
	} else if updated {
		return
	}

	if err := r.tryReconcile(ctx, &instance); err != nil {
		instance.Status.Value = err.Error()
		result.RequeueAfter = r.helper.SetFailureCount(&instance)

		log.Error(err, "an error has occurred while handling keycloak realm role batch")
	} else {
		helper.SetSuccessStatus(&instance)

		result.RequeueAfter = r.successReconcileTimeout
	}

	instanceDeleted := !controllerutil.ContainsFinalizer(&instance, keyCloakRealmRoleBatchOperatorFinalizerName) &&
		instance.GetDeletionTimestamp() != nil

	if !instanceDeleted {
		if err := r.client.Status().Update(ctx, &instance); err != nil {
			resultErr = err
		}
	}

	log.Info("Reconciling done")

	return
}

func (r *ReconcileKeycloakRealmRoleBatch) isOwner(batch *keycloakApi.KeycloakRealmRoleBatch, role *keycloakApi.KeycloakRealmRole) bool {
	for _, owner := range role.GetOwnerReferences() {
		if owner.Kind == batch.Kind && owner.Name == batch.Name && owner.UID == batch.UID {
			return true
		}
	}

	return false
}

func (r *ReconcileKeycloakRealmRoleBatch) removeRoles(ctx context.Context, batch *keycloakApi.KeycloakRealmRoleBatch) error {
	var (
		namespaceRoles keycloakApi.KeycloakRealmRoleList
		specRoles      = make(map[string]struct{})
	)

	if err := r.client.List(ctx, &namespaceRoles); err != nil {
		return errors.Wrap(err, "unable to get keycloak realm roles")
	}

	for _, r := range batch.Spec.Roles {
		specRoles[batch.FormattedRoleName(r.Name)] = struct{}{}
	}

	for i := range namespaceRoles.Items {
		if _, ok := specRoles[namespaceRoles.Items[i].Name]; !ok && r.isOwner(batch, &namespaceRoles.Items[i]) {
			if err := r.client.Delete(ctx, &namespaceRoles.Items[i]); err != nil {
				return errors.Wrap(err, "unable to delete keycloak realm role")
			}
		}
	}

	return nil
}

func (r *ReconcileKeycloakRealmRoleBatch) putRoles(
	ctx context.Context,
	batch *keycloakApi.KeycloakRealmRoleBatch,
) (roles []keycloakApi.KeycloakRealmRole, resultErr error) {
	log := ctrl.LoggerFrom(ctx)
	log.Info("Start putting keycloak cr role batch")

	for _, role := range batch.Spec.Roles {
		roleName := batch.FormattedRoleName(role.Name)

		var crRole keycloakApi.KeycloakRealmRole

		err := r.client.Get(ctx, types.NamespacedName{Namespace: batch.Namespace, Name: roleName}, &crRole)
		if err != nil && !k8sErrors.IsNotFound(err) {
			return nil, errors.Wrap(err, "unable to check batch role")
		} else if err == nil {
			if r.isOwner(batch, &crRole) {
				log.Info("Role already created")

				roles = append(roles, crRole)

				continue
			}

			return nil, errors.New("one of batch role already exists")
		}

		newRole := keycloakApi.KeycloakRealmRole{
			ObjectMeta: metav1.ObjectMeta{Name: roleName,
				Namespace: batch.Namespace,
				OwnerReferences: []metav1.OwnerReference{
					{Name: batch.Name, Kind: batch.Kind, BlockOwnerDeletion: gocloak.BoolP(true), UID: batch.UID,
						APIVersion: batch.APIVersion},
				}},
			Spec: keycloakApi.KeycloakRealmRoleSpec{
				Name:        role.Name,
				RealmRef:    batch.GetRealmRef(),
				Composite:   role.Composite,
				Composites:  role.Composites,
				Description: role.Description,
				Attributes:  role.Attributes,
				IsDefault:   role.IsDefault,
			}}
		if err := r.client.Create(ctx, &newRole); err != nil {
			return nil, errors.Wrap(err, "unable to create child role from batch")
		}

		roles = append(roles, newRole)
	}

	log.Info("Realm role batch put successfully")

	return
}

func (r *ReconcileKeycloakRealmRoleBatch) tryReconcile(ctx context.Context, batch *keycloakApi.KeycloakRealmRoleBatch) error {
	err := r.helper.SetRealmOwnerRef(ctx, batch)
	if err != nil {
		return fmt.Errorf("unable to set realm owner ref: %w", err)
	}

	createdRoles, err := r.putRoles(ctx, batch)
	if err != nil {
		return errors.Wrap(err, "unable to put roles batch")
	}

	if err := r.removeRoles(ctx, batch); err != nil {
		return errors.Wrap(err, "unable to delete roles")
	}

	if _, err := r.helper.TryToDelete(
		ctx,
		batch,
		makeTerminator(r.client, createdRoles, objectmeta.PreserveResourcesOnDeletion(batch)),
		keyCloakRealmRoleBatchOperatorFinalizerName,
	); err != nil {
		return fmt.Errorf("unable to delete keycloak realm role batch: %w", err)
	}

	return nil
}

func (r *ReconcileKeycloakRealmRoleBatch) applyDefaults(ctx context.Context, instance *keycloakApi.KeycloakRealmRoleBatch) (bool, error) {
	if instance.Spec.RealmRef.Name == "" {
		instance.Spec.RealmRef = common.RealmRef{
			Kind: keycloakApi.KeycloakRealmKind,
			Name: instance.Spec.Realm,
		}

		if err := r.client.Update(ctx, instance); err != nil {
			return false, fmt.Errorf("failed to update default values: %w", err)
		}

		return true, nil
	}

	return false, nil
}

package keycloakrealmrolebatch

import (
	"context"
	"fmt"
	"github.com/Nerzal/gocloak/v8"
	"github.com/epam/keycloak-operator/pkg/apis/v1/v1alpha1"
	keycloakApi "github.com/epam/keycloak-operator/pkg/apis/v1/v1alpha1"
	"github.com/epam/keycloak-operator/pkg/controller/helper"
	"github.com/go-logr/logr"
	"github.com/pkg/errors"
	k8sErrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

const keyCloakRealmRoleBatchOperatorFinalizerName = "keycloak.realmrolebatch.operator.finalizer.name"

type ReconcileKeycloakRealmRoleBatch struct {
	Client client.Client
	Scheme *runtime.Scheme
	Helper *helper.Helper
	Log    logr.Logger
}

func (r *ReconcileKeycloakRealmRoleBatch) SetupWithManager(mgr ctrl.Manager) error {
	pred := predicate.Funcs{
		UpdateFunc: helper.IsFailuresUpdated,
	}
	return ctrl.NewControllerManagedBy(mgr).
		For(&keycloakApi.KeycloakRealmRoleBatch{}, builder.WithPredicates(pred)).
		Complete(r)
}

func (r *ReconcileKeycloakRealmRoleBatch) Reconcile(ctx context.Context, request reconcile.Request) (result reconcile.Result,
	resultErr error) {
	log := r.Log.WithValues("Request.Namespace", request.Namespace, "Request.Name", request.Name)
	log.Info("Reconciling KeycloakRealmRoleBatch")

	var instance v1alpha1.KeycloakRealmRoleBatch
	if err := r.Client.Get(ctx, request.NamespacedName, &instance); err != nil {
		resultErr = errors.Wrap(err, "unable to get keycloak realm role batch from k8s")
		return
	}

	if err := r.tryReconcile(ctx, &instance); err != nil {
		instance.Status.Value = err.Error()
		result.RequeueAfter = r.Helper.SetFailureCount(&instance)
		r.Log.Error(err, "an error has occurred while handling keycloak realm role batch", "name",
			request.Name)
	} else {
		helper.SetSuccessStatus(&instance)
	}

	if err := r.Helper.UpdateStatus(&instance); err != nil {
		resultErr = err
	}

	return
}

func (r *ReconcileKeycloakRealmRoleBatch) isOwner(batch *v1alpha1.KeycloakRealmRoleBatch,
	role v1alpha1.KeycloakRealmRole) bool {

	for _, owner := range role.GetOwnerReferences() {
		if owner.Kind == batch.Kind && owner.Name == batch.Name && owner.UID == batch.UID {
			return true
		}
	}

	return false
}

func (r *ReconcileKeycloakRealmRoleBatch) putRoles(ctx context.Context, batch *v1alpha1.KeycloakRealmRoleBatch,
	realm *v1alpha1.KeycloakRealm) (roles []v1alpha1.KeycloakRealmRole, resultErr error) {
	log := r.Log.WithValues("keycloak role batch cr", batch)
	log.Info("Start putting keycloak cr role batch...")

	for _, role := range batch.Spec.Roles {
		roleName := fmt.Sprintf("%s-%s", batch.Name, role.Name)

		var crRole v1alpha1.KeycloakRealmRole
		err := r.Client.Get(ctx, types.NamespacedName{Namespace: batch.Namespace, Name: roleName},
			&crRole)

		if err != nil && !k8sErrors.IsNotFound(err) {
			return nil, errors.Wrap(err, "unable to check batch role")
		} else if err == nil {
			if r.isOwner(batch, crRole) {
				log.Info("role already created")
				roles = append(roles, crRole)
				continue
			}

			return nil, errors.New("one of batch role already exists")
		}

		newRole := v1alpha1.KeycloakRealmRole{
			ObjectMeta: metav1.ObjectMeta{Name: roleName,
				Namespace: batch.Namespace,
				OwnerReferences: []metav1.OwnerReference{
					{Name: realm.Name, Kind: realm.Kind, BlockOwnerDeletion: gocloak.BoolP(true), UID: realm.UID,
						APIVersion: realm.APIVersion},
					{Name: batch.Name, Kind: batch.Kind, BlockOwnerDeletion: gocloak.BoolP(true), UID: batch.UID,
						APIVersion: batch.APIVersion},
				}},
			Spec: v1alpha1.KeycloakRealmRoleSpec{
				Name:        role.Name,
				Realm:       realm.Name,
				Composite:   role.Composite,
				Composites:  role.Composites,
				Description: role.Description,
				Attributes:  role.Attributes,
			}}
		if err := r.Client.Create(ctx, &newRole); err != nil {
			return nil, errors.Wrap(err, "unable to create child role from batch")
		}
		roles = append(roles, newRole)
	}

	log.Info("Done putting keycloak cr role batch...")

	return
}

func (r *ReconcileKeycloakRealmRoleBatch) tryReconcile(ctx context.Context, batch *v1alpha1.KeycloakRealmRoleBatch) error {
	realm, err := r.Helper.GetOrCreateRealmOwnerRef(batch, batch.ObjectMeta)
	if err != nil {
		return errors.Wrap(err, "unable to get realm owner ref")
	}

	createdRoles, err := r.putRoles(ctx, batch, realm)
	if err != nil {
		return errors.Wrap(err, "unable to put roles batch")
	}

	if _, err := r.Helper.TryToDelete(batch,
		makeTerminator(r.Client, createdRoles), keyCloakRealmRoleBatchOperatorFinalizerName); err != nil {
		return errors.Wrap(err, "unable to remove child entity")
	}

	return nil
}

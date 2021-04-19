package keycloakrealmrolebatch

import (
	"context"
	"fmt"

	"github.com/Nerzal/gocloak/v8"
	"github.com/epam/edp-keycloak-operator/pkg/apis/v1/v1alpha1"
	"github.com/epam/edp-keycloak-operator/pkg/controller/helper"
	"github.com/go-logr/logr"
	"github.com/pkg/errors"
	k8sErrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	logf "sigs.k8s.io/controller-runtime/pkg/runtime/log"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

const (
	keyCloakRealmRoleBatchOperatorFinalizerName = "keycloak.realmrolebatch.operator.finalizer.name"
)

type ReconcileKeycloakRealmRoleBatch struct {
	client client.Client
	scheme *runtime.Scheme
	helper *helper.Helper
	logger logr.Logger
}

func Add(mgr manager.Manager) error {
	c, err := controller.New("keycloakrealmrolebatch-controller", mgr, controller.Options{
		Reconciler: newReconciler(mgr)})
	if err != nil {
		return err
	}

	// Watch for changes to primary resource KeycloakRealm
	return c.Watch(&source.Kind{Type: &v1alpha1.KeycloakRealmRoleBatch{}}, &handler.EnqueueRequestForObject{},
		predicate.Funcs{
			UpdateFunc: helper.IsFailuresUpdated,
		})
}

// newReconciler returns a new reconcile.Reconciler
func newReconciler(mgr manager.Manager) reconcile.Reconciler {
	return &ReconcileKeycloakRealmRoleBatch{
		client: mgr.GetClient(),
		scheme: mgr.GetScheme(),
		helper: helper.MakeHelper(mgr.GetClient(), mgr.GetScheme()),
		logger: logf.Log.WithName("controller_keycloakrealmrolebatch"),
	}
}

func (r *ReconcileKeycloakRealmRoleBatch) Reconcile(request reconcile.Request) (result reconcile.Result,
	resultErr error) {

	reqLogger := r.logger.WithValues("Request.Namespace", request.Namespace, "Request.Name", request.Name)
	reqLogger.Info("Reconciling KeycloakRealmRoleBatch")

	var instance v1alpha1.KeycloakRealmRoleBatch
	if err := r.client.Get(context.TODO(), request.NamespacedName, &instance); err != nil {
		resultErr = errors.Wrap(err, "unable to get keycloak realm role batch from k8s")
		return
	}

	if err := r.tryReconcile(&instance); err != nil {
		instance.Status.Value = err.Error()
		result.RequeueAfter = r.helper.SetFailureCount(&instance)
		r.logger.Error(err, "an error has occurred while handling keycloak realm role batch", "name",
			request.Name)
	} else {
		helper.SetSuccessStatus(&instance)
	}

	if err := r.helper.UpdateStatus(&instance); err != nil {
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

func (r *ReconcileKeycloakRealmRoleBatch) putRoles(batch *v1alpha1.KeycloakRealmRoleBatch,
	realm *v1alpha1.KeycloakRealm) (roles []v1alpha1.KeycloakRealmRole, resultErr error) {

	reqLog := r.logger.WithValues("keycloak role batch cr", batch)
	reqLog.Info("Start putting keycloak cr role batch...")

	for _, role := range batch.Spec.Roles {
		roleName := fmt.Sprintf("%s-%s", batch.Name, role.Name)

		var crRole v1alpha1.KeycloakRealmRole
		err := r.client.Get(context.TODO(), types.NamespacedName{Namespace: batch.Namespace, Name: roleName},
			&crRole)

		if err != nil && !k8sErrors.IsNotFound(err) {
			return nil, errors.Wrap(err, "unable to check batch role")
		} else if err == nil {
			if r.isOwner(batch, crRole) {
				reqLog.Info("role already created")
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
		if err := r.client.Create(context.TODO(), &newRole); err != nil {
			return nil, errors.Wrap(err, "unable to create child role from batch")
		}
		roles = append(roles, newRole)
	}

	reqLog.Info("Done putting keycloak cr role batch...")

	return
}

func (r *ReconcileKeycloakRealmRoleBatch) tryReconcile(batch *v1alpha1.KeycloakRealmRoleBatch) error {
	realm, err := r.helper.GetOrCreateRealmOwnerRef(batch, batch.ObjectMeta)
	if err != nil {
		return errors.Wrap(err, "unable to get realm owner ref")
	}

	createdRoles, err := r.putRoles(batch, realm)
	if err != nil {
		return errors.Wrap(err, "unable to put roles batch")
	}

	if _, err := r.helper.TryToDelete(batch,
		makeTerminator(r.client, createdRoles), keyCloakRealmRoleBatchOperatorFinalizerName); err != nil {
		return errors.Wrap(err, "unable to remove child entity")
	}

	return nil
}

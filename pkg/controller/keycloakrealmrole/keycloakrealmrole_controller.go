package keycloakrealmrole

import (
	"context"
	"reflect"
	"time"

	"github.com/go-logr/logr"
	"github.com/pkg/errors"
	k8sErrors "k8s.io/apimachinery/pkg/api/errors"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	keycloakApi "github.com/epam/edp-keycloak-operator/pkg/apis/v1/v1"
	"github.com/epam/edp-keycloak-operator/pkg/client/keycloak"
	"github.com/epam/edp-keycloak-operator/pkg/client/keycloak/adapter"
	"github.com/epam/edp-keycloak-operator/pkg/client/keycloak/dto"
	"github.com/epam/edp-keycloak-operator/pkg/controller/helper"
)

const keyCloakRealmRoleOperatorFinalizerName = "keycloak.realmrole.operator.finalizer.name"

type Helper interface {
	SetFailureCount(fc helper.FailureCountable) time.Duration
	UpdateStatus(obj client.Object) error
	TryToDelete(ctx context.Context, obj helper.Deletable, terminator helper.Terminator, finalizer string) (isDeleted bool, resultErr error)
	GetOrCreateRealmOwnerRef(object helper.RealmChild, objectMeta v1.ObjectMeta) (*keycloakApi.KeycloakRealm, error)
	CreateKeycloakClientForRealm(ctx context.Context, realm *keycloakApi.KeycloakRealm) (keycloak.Client, error)
}

func NewReconcileKeycloakRealmRole(client client.Client, log logr.Logger, helper Helper) *ReconcileKeycloakRealmRole {
	return &ReconcileKeycloakRealmRole{
		client: client,
		helper: helper,
		log:    log.WithName("keycloak-realm-role"),
	}
}

type ReconcileKeycloakRealmRole struct {
	client                  client.Client
	helper                  Helper
	log                     logr.Logger
	successReconcileTimeout time.Duration
}

func (r *ReconcileKeycloakRealmRole) SetupWithManager(mgr ctrl.Manager, successReconcileTimeout time.Duration) error {
	r.successReconcileTimeout = successReconcileTimeout

	pred := predicate.Funcs{
		UpdateFunc: isSpecUpdated,
	}

	return ctrl.NewControllerManagedBy(mgr).
		For(&keycloakApi.KeycloakRealmRole{}, builder.WithPredicates(pred)).
		Complete(r)
}

func isSpecUpdated(e event.UpdateEvent) bool {
	oo := e.ObjectOld.(*keycloakApi.KeycloakRealmRole)
	no := e.ObjectNew.(*keycloakApi.KeycloakRealmRole)

	return !reflect.DeepEqual(oo.Spec, no.Spec) ||
		(oo.GetDeletionTimestamp().IsZero() && !no.GetDeletionTimestamp().IsZero())
}

func (r *ReconcileKeycloakRealmRole) Reconcile(ctx context.Context, request reconcile.Request) (result reconcile.Result, resultErr error) {
	log := r.log.WithValues("Request.Namespace", request.Namespace, "Request.Name", request.Name)
	log.Info("Reconciling KeycloakRealmRole")

	var instance keycloakApi.KeycloakRealmRole
	if err := r.client.Get(ctx, request.NamespacedName, &instance); err != nil {
		if k8sErrors.IsNotFound(err) {
			return
		}

		resultErr = errors.Wrap(err, "unable to get keycloak realm role from k8s")
		return
	}

	if instance.Status.Value == keycloakApi.StatusDuplicated {
		log.Info("Role is duplicated, exit.")
		return
	}

	defer func() {
		if err := r.helper.UpdateStatus(&instance); err != nil {
			resultErr = err
		}
	}()

	roleID, err := r.tryReconcile(ctx, &instance)
	if err != nil {
		if adapter.IsErrDuplicated(err) {
			instance.Status.Value = keycloakApi.StatusDuplicated
			log.Info("Role is duplicated", "name", instance.Name)
			return
		}

		instance.Status.Value = err.Error()
		result.RequeueAfter = r.helper.SetFailureCount(&instance)
		log.Error(err, "an error has occurred while handling keycloak realm role", "name",
			request.Name)

		return
	}

	helper.SetSuccessStatus(&instance)
	instance.Status.ID = roleID
	result.RequeueAfter = r.successReconcileTimeout
	log.Info("Reconciling done")

	return
}

func (r *ReconcileKeycloakRealmRole) tryReconcile(ctx context.Context, keycloakRealmRole *keycloakApi.KeycloakRealmRole) (string, error) {
	realm, err := r.helper.GetOrCreateRealmOwnerRef(keycloakRealmRole, keycloakRealmRole.ObjectMeta)
	if err != nil {
		return "", errors.Wrap(err, "unable to get realm owner ref")
	}

	kClient, err := r.helper.CreateKeycloakClientForRealm(ctx, realm)
	if err != nil {
		return "", errors.Wrap(err, "unable to create keycloak client")
	}

	roleID, err := r.putRole(realm, keycloakRealmRole, kClient)
	if err != nil {
		return "", errors.Wrap(err, "unable to put role")
	}

	if _, err := r.helper.TryToDelete(ctx, keycloakRealmRole,
		makeTerminator(realm.Spec.RealmName, keycloakRealmRole.Spec.Name, kClient, r.log.WithName("realm-role-term")),
		keyCloakRealmRoleOperatorFinalizerName); err != nil {
		return "", errors.Wrap(err, "unable to tryToDelete realm role")
	}

	return roleID, nil
}

func (r *ReconcileKeycloakRealmRole) putRole(
	keycloakRealm *keycloakApi.KeycloakRealm, keycloakRealmRole *keycloakApi.KeycloakRealmRole,
	kClient keycloak.Client) (string, error) {

	log := r.log.WithValues("keycloak role cr", keycloakRealmRole)
	log.Info("Start put keycloak cr role...")

	role := dto.ConvertSpecToRole(keycloakRealmRole)

	if err := kClient.SyncRealmRole(keycloakRealm.Spec.RealmName, role); err != nil {
		return "", errors.Wrap(err, "unable to sync realm role CR")
	}

	var roleID string

	if role.ID != nil {
		roleID = *role.ID
	}

	log.Info("Done putting keycloak cr role...")

	return roleID, nil
}

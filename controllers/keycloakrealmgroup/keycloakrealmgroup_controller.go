package keycloakrealmgroup

import (
	"context"
	"fmt"
	"time"

	"github.com/go-logr/logr"
	"github.com/pkg/errors"
	k8sErrors "k8s.io/apimachinery/pkg/api/errors"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	keycloakApi "github.com/epam/edp-keycloak-operator/api/v1/v1"
	"github.com/epam/edp-keycloak-operator/controllers/helper"
	"github.com/epam/edp-keycloak-operator/pkg/client/keycloak"
)

const keyCloakRealmGroupOperatorFinalizerName = "keycloak.realmgroup.operator.finalizer.name"

type Helper interface {
	SetFailureCount(fc helper.FailureCountable) time.Duration
	UpdateStatus(obj client.Object) error
	GetOrCreateRealmOwnerRef(object helper.RealmChild, objectMeta *v1.ObjectMeta) (*keycloakApi.KeycloakRealm, error)
	TryToDelete(ctx context.Context, obj helper.Deletable, terminator helper.Terminator, finalizer string) (isDeleted bool, resultErr error)
	CreateKeycloakClientForRealm(ctx context.Context, realm *keycloakApi.KeycloakRealm) (keycloak.Client, error)
}

func NewReconcileKeycloakRealmGroup(client client.Client, log logr.Logger,
	helper Helper) *ReconcileKeycloakRealmGroup {
	return &ReconcileKeycloakRealmGroup{
		client: client,
		helper: helper,
		log:    log.WithName("keycloak-realm-group"),
	}
}

type ReconcileKeycloakRealmGroup struct {
	client                  client.Client
	helper                  Helper
	log                     logr.Logger
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

//+kubebuilder:rbac:groups=v1.edp.epam.com,namespace=placeholder,resources=keycloakrealmgroups,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=v1.edp.epam.com,namespace=placeholder,resources=keycloakrealmgroups/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=v1.edp.epam.com,namespace=placeholder,resources=keycloakrealmgroups/finalizers,verbs=update

// Reconcile is a loop for reconciling KeycloakRealmGroup object.
func (r *ReconcileKeycloakRealmGroup) Reconcile(ctx context.Context, request reconcile.Request) (result reconcile.Result, resultErr error) {
	log := r.log.WithValues("Request.Namespace", request.Namespace, "Request.Name", request.Name)
	log.Info("Reconciling KeycloakRealmGroup")

	var instance keycloakApi.KeycloakRealmGroup
	if err := r.client.Get(ctx, request.NamespacedName, &instance); err != nil {
		if k8sErrors.IsNotFound(err) {
			return
		}

		resultErr = errors.Wrap(err, "unable to get keycloak realm group from k8s")

		return
	}

	if err := r.tryReconcile(ctx, &instance); err != nil {
		instance.Status.Value = err.Error()
		result.RequeueAfter = r.helper.SetFailureCount(&instance)

		log.Error(err, "an error has occurred while handling keycloak realm group", "name", request.Name)
	} else {
		helper.SetSuccessStatus(&instance)
		result.RequeueAfter = r.successReconcileTimeout
	}

	if err := r.helper.UpdateStatus(&instance); err != nil {
		resultErr = errors.Wrap(err, "unable to update status")
	}

	log.Info("Reconciling done")

	return
}

func (r *ReconcileKeycloakRealmGroup) tryReconcile(ctx context.Context, keycloakRealmGroup *keycloakApi.KeycloakRealmGroup) error {
	realm, err := r.helper.GetOrCreateRealmOwnerRef(keycloakRealmGroup, &keycloakRealmGroup.ObjectMeta)
	if err != nil {
		return errors.Wrap(err, "unable to get realm owner ref")
	}

	kClient, err := r.helper.CreateKeycloakClientForRealm(ctx, realm)
	if err != nil {
		return errors.Wrap(err, "unable to create keycloak client")
	}

	id, err := kClient.SyncRealmGroup(realm.Spec.RealmName, &keycloakRealmGroup.Spec)
	if err != nil {
		return errors.Wrap(err, "unable to sync realm role")
	}

	keycloakRealmGroup.Status.ID = id

	if _, err := r.helper.TryToDelete(ctx, keycloakRealmGroup,
		makeTerminator(kClient, realm.Spec.RealmName, keycloakRealmGroup.Spec.Name,
			r.log.WithName("realm-group-term")),
		keyCloakRealmGroupOperatorFinalizerName); err != nil {
		return errors.Wrap(err, "unable to tryToDelete realm role")
	}

	return nil
}

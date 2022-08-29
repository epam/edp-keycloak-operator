package keycloakrealm

import (
	"context"
	"fmt"
	"time"

	"github.com/go-logr/logr"
	"github.com/pkg/errors"
	k8sErrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	keycloakApi "github.com/epam/edp-keycloak-operator/pkg/apis/v1/v1"
	"github.com/epam/edp-keycloak-operator/pkg/client/keycloak"
	"github.com/epam/edp-keycloak-operator/pkg/controller/helper"
	"github.com/epam/edp-keycloak-operator/pkg/controller/keycloakrealm/chain"
	"github.com/epam/edp-keycloak-operator/pkg/controller/keycloakrealm/chain/handler"
)

const keyCloakRealmOperatorFinalizerName = "keycloak.realm.operator.finalizer.name"

type Helper interface {
	SetFailureCount(fc helper.FailureCountable) time.Duration
	UpdateStatus(obj client.Object) error
	TryToDelete(ctx context.Context, obj helper.Deletable, terminator helper.Terminator, finalizer string) (isDeleted bool, resultErr error)
	CreateKeycloakClientForRealm(ctx context.Context, realm *keycloakApi.KeycloakRealm) (keycloak.Client, error)
	InvalidateKeycloakClientTokenSecret(ctx context.Context, namespace, rootKeycloakName string) error
}

func NewReconcileKeycloakRealm(client client.Client, scheme *runtime.Scheme, log logr.Logger, helper Helper) *ReconcileKeycloakRealm {
	return &ReconcileKeycloakRealm{
		client: client,
		helper: helper,
		log:    log.WithName("keycloak-realm"),
		chain:  chain.CreateDefChain(client, scheme, helper),
	}
}

// ReconcileKeycloakRealm reconciles a KeycloakRealm object.
type ReconcileKeycloakRealm struct {
	client                  client.Client
	helper                  Helper
	log                     logr.Logger
	chain                   handler.RealmHandler
	successReconcileTimeout time.Duration
}

func (r *ReconcileKeycloakRealm) SetupWithManager(mgr ctrl.Manager, successReconcileTimeout time.Duration) error {
	r.successReconcileTimeout = successReconcileTimeout
	pred := predicate.Funcs{
		UpdateFunc: helper.IsFailuresUpdated,
	}

	err := ctrl.NewControllerManagedBy(mgr).
		For(&keycloakApi.KeycloakRealm{}, builder.WithPredicates(pred)).
		Complete(r)
	if err != nil {
		return fmt.Errorf("failed to setup KeycloakRealm controller: %w", err)
	}

	return nil
}

func (r *ReconcileKeycloakRealm) Reconcile(ctx context.Context, request reconcile.Request) (result reconcile.Result, resultErr error) {
	log := r.log.WithValues("Request.Namespace", request.Namespace, "Request.Name", request.Name)
	log.Info("Reconciling KeycloakRealm")

	instance := &keycloakApi.KeycloakRealm{}
	if err := r.client.Get(ctx, request.NamespacedName, instance); err != nil {
		if k8sErrors.IsNotFound(err) {
			// Request object not found, could have been deleted after reconcile request.
			// Owned objects are automatically garbage collected. For additional cleanup logic use finalizers.
			// Return and don't requeue
			return
		}

		resultErr = err

		return
	}

	if err := r.tryReconcile(ctx, instance); err != nil {
		instance.Status.Available = false
		instance.Status.Value = err.Error()
		result.RequeueAfter = r.helper.SetFailureCount(instance)

		log.Error(err, "an error has occurred while handling keycloak realm", "name", request.Name)
	} else {
		instance.Status.Available = true
		instance.Status.Value = helper.StatusOK
		instance.Status.FailureCount = 0
		result.RequeueAfter = r.successReconcileTimeout
	}

	if err := r.helper.UpdateStatus(instance); err != nil {
		resultErr = errors.Wrap(err, "unable to update status")
	}

	return
}

func (r *ReconcileKeycloakRealm) tryReconcile(ctx context.Context, realm *keycloakApi.KeycloakRealm) error {
	kClient, err := r.helper.CreateKeycloakClientForRealm(ctx, realm)
	if err != nil {
		return fmt.Errorf("failed to create keycloak client for realm: %w", err)
	}

	deleted, err := r.helper.TryToDelete(ctx, realm,
		makeTerminator(realm.Spec.RealmName, kClient, r.log.WithName("realm-group-term")),
		keyCloakRealmOperatorFinalizerName)
	if err != nil {
		return errors.Wrap(err, "error during realm deletion")
	}

	if deleted {
		return nil
	}

	if err := r.chain.ServeRequest(ctx, realm, kClient); err != nil {
		return errors.Wrap(err, "error during realm chain")
	}

	return nil
}

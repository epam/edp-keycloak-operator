package keycloak

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/go-logr/logr"
	pkgErrors "github.com/pkg/errors"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	keycloakApi "github.com/epam/edp-keycloak-operator/api/v1"
	"github.com/epam/edp-keycloak-operator/controllers/helper"
	"github.com/epam/edp-keycloak-operator/pkg/client/keycloak"
	"github.com/epam/edp-keycloak-operator/pkg/client/keycloak/adapter"
)

const (
	keycloakCRLogKey = "keycloak cr"
)

type Helper interface {
	CreateKeycloakClientFromTokenSecret(ctx context.Context, kc *keycloakApi.Keycloak) (keycloak.Client, error)
	CreateKeycloakClientFromLoginPassword(ctx context.Context, kc *keycloakApi.Keycloak) (keycloak.Client, error)
	TokenSecretLock() *sync.Mutex
}

func NewReconcileKeycloak(client client.Client, scheme *runtime.Scheme, log logr.Logger, helper Helper) *ReconcileKeycloak {
	return &ReconcileKeycloak{
		client: client,
		scheme: scheme,
		log:    log.WithName("keycloak"),
		helper: helper,
	}
}

// ReconcileKeycloak reconciles a Keycloak object.
type ReconcileKeycloak struct {
	client                  client.Client
	scheme                  *runtime.Scheme
	log                     logr.Logger
	helper                  Helper
	successReconcileTimeout time.Duration
}

func (r *ReconcileKeycloak) SetupWithManager(mgr ctrl.Manager, successReconcileTimeout time.Duration) error {
	r.successReconcileTimeout = successReconcileTimeout

	pred := predicate.Funcs{
		DeleteFunc: func(e event.DeleteEvent) bool {
			return false
		},
	}

	err := ctrl.NewControllerManagedBy(mgr).
		For(&keycloakApi.Keycloak{}, builder.WithPredicates(pred)).
		Complete(r)
	if err != nil {
		return fmt.Errorf("failed to setup Keycloak controller: %w", err)
	}

	return nil
}

//+kubebuilder:rbac:groups=v1.edp.epam.com,namespace=placeholder,resources=keycloaks,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=v1.edp.epam.com,namespace=placeholder,resources=keycloaks/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=v1.edp.epam.com,namespace=placeholder,resources=keycloaks/finalizers,verbs=update

// Reconcile is a loop for reconciling Keycloak object.
func (r *ReconcileKeycloak) Reconcile(ctx context.Context, request reconcile.Request) (reconcile.Result, error) {
	log := r.log.WithValues("Request.Namespace", request.Namespace, "Request.Name", request.Name)
	log.Info("Reconciling Keycloak")

	instance := &keycloakApi.Keycloak{}
	if err := r.client.Get(ctx, request.NamespacedName, instance); err != nil {
		if errors.IsNotFound(err) {
			log.Info("instance not found")
			return reconcile.Result{}, nil
		}

		log.Error(err, "unable to get keycloak cr")

		return reconcile.Result{RequeueAfter: helper.DefaultRequeueTime}, nil
	}

	if err := r.tryToReconcile(ctx, instance, request); err != nil {
		log.Error(err, "error during reconcilation")
		return reconcile.Result{RequeueAfter: helper.DefaultRequeueTime}, nil
	}

	log.Info("Reconciling Keycloak has been finished")

	return reconcile.Result{
		RequeueAfter: r.successReconcileTimeout,
	}, nil
}

func (r *ReconcileKeycloak) tryToReconcile(ctx context.Context, instance *keycloakApi.Keycloak,
	request reconcile.Request) error {
	if err := r.updateConnectionStatusToKeycloak(ctx, instance); err != nil {
		return pkgErrors.Wrap(err, "unable to update connection status to keycloak")
	}

	con, err := r.isStatusConnected(ctx, request)
	if err != nil {
		return pkgErrors.Wrap(err, "unable to check connection status")
	}

	if !con {
		return pkgErrors.New("Keycloak CR status is not connected")
	}

	return nil
}

func (r *ReconcileKeycloak) updateConnectionStatusToKeycloak(ctx context.Context, instance *keycloakApi.Keycloak) error {
	log := r.log.WithValues(keycloakCRLogKey, instance)
	log.Info("Start updating connection status to Keycloak")

	connected, err := r.isInstanceConnected(ctx, instance, log)
	if err != nil {
		return pkgErrors.Wrap(err, "error during kc checking connection")
	}

	instance.Status.Connected = connected

	err = r.client.Status().Update(ctx, instance)
	if err != nil {
		log.Error(err, "unable to update keycloak cr status")

		err := r.client.Update(ctx, instance)
		if err != nil {
			log.Info(fmt.Sprintf("Couldn't update status for Keycloak %s", instance.Name))
			return fmt.Errorf("failed to update status for keycloak: %w", err)
		}
	}

	log.Info("Status has been updated", "status", instance.Status)

	return nil
}

func (r *ReconcileKeycloak) isInstanceConnected(ctx context.Context, instance *keycloakApi.Keycloak,
	logger logr.Logger) (bool, error) {
	r.helper.TokenSecretLock().Lock()
	defer r.helper.TokenSecretLock().Unlock()

	_, err := r.helper.CreateKeycloakClientFromTokenSecret(ctx, instance)
	if err == nil {
		return true, nil
	}

	if !errors.IsNotFound(err) && !adapter.IsErrTokenExpired(err) {
		return false, fmt.Errorf("failed to create keycloak client from token: %w", err)
	}

	_, err = r.helper.CreateKeycloakClientFromLoginPassword(ctx, instance)
	if err != nil {
		logger.Error(err, "error during the creation of connection")
	}

	return err == nil, nil
}

func (r *ReconcileKeycloak) isStatusConnected(ctx context.Context, request reconcile.Request) (bool, error) {
	r.log.Info("Check is status of CR is connected", "request", request)

	instance := &keycloakApi.Keycloak{}

	err := r.client.Get(ctx, request.NamespacedName, instance)
	if err != nil {
		return false, pkgErrors.Wrapf(err, "unable to get keycloak instance, request: %+v", request)
	}

	r.log.Info("Retrieved the actual cr for Keycloak", keycloakCRLogKey, instance)

	return instance.Status.Connected, nil
}

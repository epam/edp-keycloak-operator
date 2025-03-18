package keycloak

import (
	"context"
	"fmt"
	"time"

	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	keycloakApi "github.com/epam/edp-keycloak-operator/api/v1"
	"github.com/epam/edp-keycloak-operator/internal/controller/helper"
	"github.com/epam/edp-keycloak-operator/pkg/client/keycloak"
)

type Helper interface {
	CreateKeycloakClientFomAuthData(ctx context.Context, authData *helper.KeycloakAuthData) (keycloak.Client, error)
}

func NewReconcileKeycloak(client client.Client, scheme *runtime.Scheme, helper Helper) *ReconcileKeycloak {
	return &ReconcileKeycloak{
		client: client,
		scheme: scheme,
		helper: helper,
	}
}

// ReconcileKeycloak reconciles a Keycloak object.
type ReconcileKeycloak struct {
	client                  client.Client
	scheme                  *runtime.Scheme
	helper                  Helper
	successReconcileTimeout time.Duration
}

const connectionRetryPeriod = time.Second * 10

//+kubebuilder:rbac:groups=v1.edp.epam.com,namespace=placeholder,resources=keycloaks,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=v1.edp.epam.com,namespace=placeholder,resources=keycloaks/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=v1.edp.epam.com,namespace=placeholder,resources=keycloaks/finalizers,verbs=update
//+kubebuilder:rbac:groups=v1,namespace=placeholder,resources=configmap,verbs=get;list;watch

// Reconcile is a loop for reconciling Keycloak object.
func (r *ReconcileKeycloak) Reconcile(ctx context.Context, req reconcile.Request) (reconcile.Result, error) {
	log := ctrl.LoggerFrom(ctx)
	log.Info("Reconciling Keycloak")

	instance := &keycloakApi.Keycloak{}
	if err := r.client.Get(ctx, req.NamespacedName, instance); err != nil {
		if errors.IsNotFound(err) {
			log.Info("Instance not found")

			return reconcile.Result{}, nil
		}

		return ctrl.Result{}, fmt.Errorf("unable to get keycloak instance: %w", err)
	}

	if err := r.updateConnectionStatusToKeycloak(ctx, instance); err != nil {
		return reconcile.Result{}, err
	}

	if !instance.Status.Connected {
		log.Info("Keycloak is not connected, will retry")
		return reconcile.Result{RequeueAfter: connectionRetryPeriod}, nil
	}

	log.Info("Reconciling Keycloak has been finished")

	return reconcile.Result{}, nil
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

func (r *ReconcileKeycloak) updateConnectionStatusToKeycloak(ctx context.Context, instance *keycloakApi.Keycloak) error {
	log := ctrl.LoggerFrom(ctx)
	log.Info("Start updating connection status to Keycloak")

	err := r.createClient(ctx, instance)
	if err != nil {
		log.Error(err, "Unable to connect to Keycloak")
	}

	connected := err == nil

	if instance.Status.Connected == connected {
		log.Info("Connection status hasn't been changed", "status", instance.Status.Connected)

		return nil
	}

	log.Info("Connection status has been changed", "from", instance.Status.Connected, "to", connected)

	instance.Status.Connected = connected

	err = r.client.Status().Update(ctx, instance)
	if err != nil {
		return fmt.Errorf("failed to update status: %w", err)
	}

	log.Info("Status has been updated", "status", instance.Status)

	return nil
}

func (r *ReconcileKeycloak) createClient(ctx context.Context, instance *keycloakApi.Keycloak) error {
	auth, err := helper.MakeKeycloakAuthDataFromKeycloak(ctx, instance, r.client)
	if err != nil {
		return fmt.Errorf("failed to make Keycloak auth data: %w", err)
	}

	_, err = r.helper.CreateKeycloakClientFomAuthData(ctx, auth)
	if err != nil {
		return fmt.Errorf("failed to create Keycloak client: %w", err)
	}

	return nil
}

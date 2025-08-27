package clusterkeycloak

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

	keycloakApi "github.com/epam/edp-keycloak-operator/api/v1alpha1"
	"github.com/epam/edp-keycloak-operator/internal/controller/helper"
	"github.com/epam/edp-keycloak-operator/pkg/client/keycloak"
)

type keycloakClientProvider interface {
	CreateKeycloakClientFomAuthData(ctx context.Context, authData *helper.KeycloakAuthData) (keycloak.Client, error)
}

func NewReconcile(
	k8sClient client.Client,
	scheme *runtime.Scheme,
	controllerHelper keycloakClientProvider,
	operatorNamespace string,
) *Reconciler {
	return &Reconciler{
		client:            k8sClient,
		scheme:            scheme,
		helper:            controllerHelper,
		operatorNamespace: operatorNamespace,
	}
}

// Reconciler reconciles a Keycloak object.
type Reconciler struct {
	client            client.Client
	scheme            *runtime.Scheme
	helper            keycloakClientProvider
	operatorNamespace string
}

const (
	failedConnectionRetryPeriod  = time.Second * 10
	successConnectionRetryPeriod = time.Minute * 30
)

// +kubebuilder:rbac:groups=v1.edp.epam.com,resources=clusterkeycloaks,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=v1.edp.epam.com,resources=clusterkeycloaks/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=v1.edp.epam.com,resources=clusterkeycloaks/finalizers,verbs=update
// +kubebuilder:rbac:groups=v1,resources=configmap,verbs=get;list;watch

// Reconcile is a loop for reconciling ClusterKeycloak object.
func (r *Reconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := ctrl.LoggerFrom(ctx)
	log.Info("Reconciling ClusterKeycloak")

	clusterKeycloak := &keycloakApi.ClusterKeycloak{}
	if err := r.client.Get(ctx, req.NamespacedName, clusterKeycloak); err != nil {
		if errors.IsNotFound(err) {
			log.Info("Instance not found")

			return reconcile.Result{}, nil
		}

		return ctrl.Result{}, fmt.Errorf("unable to get cluster keycloak: %w", err)
	}

	if err := r.updateConnectionStatusToKeycloak(ctx, clusterKeycloak); err != nil {
		return reconcile.Result{}, err
	}

	if !clusterKeycloak.Status.Connected {
		log.Info("ClusterKeycloak is not connected, will retry")
		return reconcile.Result{RequeueAfter: failedConnectionRetryPeriod}, nil
	}

	log.Info("Reconciling ClusterKeycloak has been finished")

	return reconcile.Result{
		RequeueAfter: successConnectionRetryPeriod,
	}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *Reconciler) SetupWithManager(mgr ctrl.Manager) error {
	pred := predicate.Funcs{
		DeleteFunc: func(e event.DeleteEvent) bool {
			return false
		},
	}

	err := ctrl.NewControllerManagedBy(mgr).
		For(&keycloakApi.ClusterKeycloak{}, builder.WithPredicates(pred)).
		Complete(r)

	if err != nil {
		return fmt.Errorf("failed to setup ClusterKeycloak controller: %w", err)
	}

	return nil
}

func (r *Reconciler) updateConnectionStatusToKeycloak(ctx context.Context, instance *keycloakApi.ClusterKeycloak) error {
	log := ctrl.LoggerFrom(ctx)
	log.Info("Start updating connection status to ClusterKeycloak")

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

func (r *Reconciler) createClient(ctx context.Context, instance *keycloakApi.ClusterKeycloak) error {
	auth, err := helper.MakeKeycloakAuthDataFromClusterKeycloak(ctx, instance, r.operatorNamespace, r.client)
	if err != nil {
		return fmt.Errorf("failed to make Keycloak auth data: %w", err)
	}

	_, err = r.helper.CreateKeycloakClientFomAuthData(ctx, auth)
	if err != nil {
		return fmt.Errorf("failed to create Keycloak client: %w", err)
	}

	return nil
}

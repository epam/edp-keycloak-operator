package clusterkeycloak

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	keycloakApi "github.com/epam/edp-keycloak-operator/api/v1alpha1"
	"github.com/epam/edp-keycloak-operator/controllers/helper"
)

type Helper interface {
	TokenSecretLock() *sync.Mutex
}

func NewReconcile(client client.Client, scheme *runtime.Scheme, log logr.Logger, helper Helper) *ClusterKeycloakReconciler {
	return &ClusterKeycloakReconciler{
		client: client,
		scheme: scheme,
		log:    log.WithName("clusterkeycloak"),
		helper: helper,
	}
}

// ReconcileKeycloak reconciles a Keycloak object.
type ClusterKeycloakReconciler struct {
	client                  client.Client
	scheme                  *runtime.Scheme
	log                     logr.Logger
	helper                  Helper
	successReconcileTimeout time.Duration
}

//+kubebuilder:rbac:groups=v1.edp.epam.com,resources=clusterkeycloaks,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=v1.edp.epam.com,resources=clusterkeycloaks/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=v1.edp.epam.com,resources=clusterkeycloaks/finalizers,verbs=update

// Reconcile is a loop for reconciling ClusterKeycloak object.
func (r *ClusterKeycloakReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := r.log.WithValues("Request.Namespace", req.Namespace, "Request.Name", req.Name)
	log.Info("Reconciling Keycloak")

	instance := &keycloakApi.ClusterKeycloak{}
	if err := r.client.Get(ctx, req.NamespacedName, instance); err != nil {
		if errors.IsNotFound(err) {
			log.Info("instance not found")
			return reconcile.Result{}, nil
		}

		log.Error(err, "unable to get clusterkeycloak cr")

		return reconcile.Result{RequeueAfter: helper.DefaultRequeueTime}, nil
	}

	log.Info("Reconciling ClusterKeycloak has been finished")

	return reconcile.Result{
		RequeueAfter: r.successReconcileTimeout,
	}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *ClusterKeycloakReconciler) SetupWithManager(mgr ctrl.Manager, successReconcileTimeout time.Duration) error {
	r.successReconcileTimeout = successReconcileTimeout

	err := ctrl.NewControllerManagedBy(mgr).
		For(&keycloakApi.ClusterKeycloak{}).
		Complete(r)

	if err != nil {
		return fmt.Errorf("failed to setup Keycloak controller: %w", err)
	}

	return nil
}

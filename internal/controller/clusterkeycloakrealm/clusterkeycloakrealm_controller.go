package clusterkeycloakrealm

import (
	"context"
	"errors"
	"fmt"
	"time"

	k8sErrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	"github.com/epam/edp-keycloak-operator/api/common"
	keycloakAlpha "github.com/epam/edp-keycloak-operator/api/v1alpha1"
	"github.com/epam/edp-keycloak-operator/internal/controller/clusterkeycloakrealm/chain"
	"github.com/epam/edp-keycloak-operator/internal/controller/helper"
	"github.com/epam/edp-keycloak-operator/pkg/client/keycloak"
	"github.com/epam/edp-keycloak-operator/pkg/objectmeta"
)

type Helper interface {
	SetFailureCount(fc helper.FailureCountable) time.Duration
	TryToDelete(ctx context.Context, obj client.Object, terminator helper.Terminator, finalizer string) (isDeleted bool, resultErr error)
	CreateKeycloakClientFromClusterRealm(ctx context.Context, realm *keycloakAlpha.ClusterKeycloakRealm) (keycloak.Client, error)
	SetKeycloakOwnerRef(ctx context.Context, object helper.ObjectWithKeycloakRef) error
	InvalidateKeycloakClientTokenSecret(ctx context.Context, namespace, rootKeycloakName string) error
}

// ClusterKeycloakRealmReconciler reconciles a ClusterKeycloakRealm object.
type ClusterKeycloakRealmReconciler struct {
	client            client.Client
	scheme            *runtime.Scheme
	helper            Helper
	operatorNamespace string
}

func NewClusterKeycloakRealmReconciler(client client.Client, scheme *runtime.Scheme, helper Helper, operatorNamespace string) *ClusterKeycloakRealmReconciler {
	return &ClusterKeycloakRealmReconciler{client: client, scheme: scheme, helper: helper, operatorNamespace: operatorNamespace}
}

const (
	keyCloakRealmOperatorFinalizerName = "keycloak.realm.operator.finalizer.name"
	successConnectionRetryPeriod       = time.Minute * 30
)

//+kubebuilder:rbac:groups=v1.edp.epam.com,resources=clusterkeycloakrealms,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=v1.edp.epam.com,resources=clusterkeycloakrealms/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=v1.edp.epam.com,resources=clusterkeycloakrealms/finalizers,verbs=update

// Reconcile is loop for reconciling ClusterKeycloakRealm object.
func (r *ClusterKeycloakRealmReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := ctrl.LoggerFrom(ctx)
	log.Info("Reconciling ClusterKeycloakRealm")

	clusterRealm := &keycloakAlpha.ClusterKeycloakRealm{}
	if err := r.client.Get(ctx, req.NamespacedName, clusterRealm); err != nil {
		if k8sErrors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}

		return ctrl.Result{}, fmt.Errorf("unable to get cluster realm: %w", err)
	}

	if err := r.helper.SetKeycloakOwnerRef(ctx, clusterRealm); err != nil {
		return ctrl.Result{}, fmt.Errorf("unable to set keycloak owner ref: %w", err)
	}

	kClient, err := r.helper.CreateKeycloakClientFromClusterRealm(ctx, clusterRealm)
	if err != nil {
		if errors.Is(err, helper.ErrKeycloakIsNotAvailable) {
			return ctrl.Result{
				RequeueAfter: helper.RequeueOnKeycloakNotAvailablePeriod,
			}, nil
		}

		return ctrl.Result{}, fmt.Errorf("failed to create keycloak client for realm: %w", err)
	}

	if deleted, err := r.helper.TryToDelete(
		ctx,
		clusterRealm,
		makeTerminator(clusterRealm.Spec.RealmName, kClient, objectmeta.PreserveResourcesOnDeletion(clusterRealm)),
		keyCloakRealmOperatorFinalizerName,
	); err != nil {
		return ctrl.Result{}, fmt.Errorf("failed to delete realm %w", err)
	} else if deleted {
		return reconcile.Result{}, nil
	}

	if err := chain.MakeChain(r.client, r.operatorNamespace).ServeRequest(ctx, clusterRealm, kClient); err != nil {
		clusterRealm.Status.Available = false
		clusterRealm.Status.Value = err.Error()
		requeue := r.helper.SetFailureCount(clusterRealm)

		if updateErr := r.client.Status().Update(ctx, clusterRealm); updateErr != nil {
			return ctrl.Result{}, fmt.Errorf("unable to update cluster realm status: %w", updateErr)
		}

		return ctrl.Result{
			RequeueAfter: requeue,
		}, fmt.Errorf("error during ClusterRealm chain: %w", err)
	}

	if err := r.updateSuccessStatus(ctx, clusterRealm); err != nil {
		return ctrl.Result{}, err
	}

	return ctrl.Result{
		RequeueAfter: successConnectionRetryPeriod,
	}, nil
}

func (r *ClusterKeycloakRealmReconciler) updateSuccessStatus(ctx context.Context, clusterRealm *keycloakAlpha.ClusterKeycloakRealm) error {
	if clusterRealm.Status.Available {
		return nil
	}

	clusterRealm.Status.Available = true
	clusterRealm.Status.Value = common.StatusOK
	clusterRealm.Status.FailureCount = 0

	if err := r.client.Status().Update(ctx, clusterRealm); err != nil {
		return fmt.Errorf("unable to update cluster realm status: %w", err)
	}

	return nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *ClusterKeycloakRealmReconciler) SetupWithManager(mgr ctrl.Manager) error {
	err := ctrl.NewControllerManagedBy(mgr).
		For(&keycloakAlpha.ClusterKeycloakRealm{}).
		Complete(r)

	if err != nil {
		return fmt.Errorf("unable to create ClusterKeycloakRealm controller: %w", err)
	}

	return nil
}

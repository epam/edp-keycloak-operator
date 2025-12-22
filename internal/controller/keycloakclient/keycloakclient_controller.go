package keycloakclient

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/Nerzal/gocloak/v12"
	k8sErrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	keycloakApi "github.com/epam/edp-keycloak-operator/api/v1"
	"github.com/epam/edp-keycloak-operator/internal/controller/helper"
	"github.com/epam/edp-keycloak-operator/internal/controller/keycloakclient/chain"
	"github.com/epam/edp-keycloak-operator/pkg/client/keycloak"
	"github.com/epam/edp-keycloak-operator/pkg/objectmeta"
)

type Helper interface {
	SetFailureCount(fc helper.FailureCountable) time.Duration
	TryRemoveFinalizer(ctx context.Context, obj client.Object, finalizer string) error
	TryToDelete(ctx context.Context, obj client.Object, terminator helper.Terminator, finalizer string) (isDeleted bool, resultErr error)
	SetRealmOwnerRef(ctx context.Context, object helper.ObjectWithRealmRef) error
	CreateKeycloakClientFromRealmRef(ctx context.Context, object helper.ObjectWithRealmRef) (keycloak.Client, error)
	GetKeycloakRealmFromRef(ctx context.Context, object helper.ObjectWithRealmRef, kcClient keycloak.Client) (*gocloak.RealmRepresentation, error)
}

const (
	keyCloakClientOperatorFinalizerName = "keycloak.client.operator.finalizer.name"
)

func NewReconcileKeycloakClient(k8sClient client.Client, controllerHelper Helper) *ReconcileKeycloakClient {
	return &ReconcileKeycloakClient{
		client: k8sClient,
		helper: controllerHelper,
	}
}

// ReconcileKeycloakClient reconciles a KeycloakClient object.
type ReconcileKeycloakClient struct {
	client                  client.Client
	helper                  Helper
	successReconcileTimeout time.Duration
}

func (r *ReconcileKeycloakClient) SetupWithManager(mgr ctrl.Manager, successReconcileTimeout time.Duration) error {
	r.successReconcileTimeout = successReconcileTimeout

	pred := predicate.Funcs{
		UpdateFunc: helper.IsFailuresUpdated,
	}

	err := ctrl.NewControllerManagedBy(mgr).
		For(&keycloakApi.KeycloakClient{}, builder.WithPredicates(pred)).
		Complete(r)
	if err != nil {
		return fmt.Errorf("failed to setup KeycloakClient controller: %w", err)
	}

	return nil
}

// +kubebuilder:rbac:groups=v1.edp.epam.com,namespace=placeholder,resources=keycloakclients,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=v1.edp.epam.com,namespace=placeholder,resources=keycloakclients/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=v1.edp.epam.com,namespace=placeholder,resources=keycloakclients/finalizers,verbs=update

// Reconcile is a loop for reconciling KeycloakClient object.
func (r *ReconcileKeycloakClient) Reconcile(ctx context.Context, request reconcile.Request) (result reconcile.Result, resultErr error) {
	log := ctrl.LoggerFrom(ctx)
	log.Info("Reconciling KeycloakClient")

	var instance keycloakApi.KeycloakClient
	if err := r.client.Get(ctx, request.NamespacedName, &instance); err != nil {
		if k8sErrors.IsNotFound(err) {
			return result, resultErr
		}

		resultErr = err

		return result, resultErr
	}

	if err := r.tryReconcile(ctx, &instance); err != nil {
		if errors.Is(err, helper.ErrKeycloakIsNotAvailable) {
			return ctrl.Result{
				RequeueAfter: helper.RequeueOnKeycloakNotAvailablePeriod,
			}, nil
		}

		// Set Ready condition to False
		meta.SetStatusCondition(&instance.Status.Conditions, metav1.Condition{
			Type:               chain.ConditionReady,
			Status:             metav1.ConditionFalse,
			Reason:             chain.ReasonKeycloakAPIError,
			Message:            fmt.Sprintf("Reconciliation failed: %s", err.Error()),
			ObservedGeneration: instance.Generation,
		})

		// Backward compatibility: set Value field
		instance.Status.Value = err.Error()
		result.RequeueAfter = r.helper.SetFailureCount(&instance)

		log.Error(err, "an error has occurred while handling keycloak client", "name", request.Name)
	} else {
		// Set Ready condition to True
		meta.SetStatusCondition(&instance.Status.Conditions, metav1.Condition{
			Type:               chain.ConditionReady,
			Status:             metav1.ConditionTrue,
			Reason:             chain.ReasonReconciliationSucceeded,
			Message:            "KeycloakClient reconciliation completed successfully",
			ObservedGeneration: instance.Generation,
		})

		// Backward compatibility: set Value field
		helper.SetSuccessStatus(&instance)

		result.RequeueAfter = r.successReconcileTimeout
	}

	// Final status update for Ready condition and Value field
	if err := r.client.Status().Update(ctx, &instance); err != nil {
		resultErr = fmt.Errorf("unable to update status: %w", err)
	}

	return result, resultErr
}

func (r *ReconcileKeycloakClient) tryReconcile(ctx context.Context, keycloakClient *keycloakApi.KeycloakClient) error {
	err := r.helper.SetRealmOwnerRef(ctx, keycloakClient)
	if err != nil {
		return fmt.Errorf("unable to set realm owner ref: %w", err)
	}

	kClient, err := r.helper.CreateKeycloakClientFromRealmRef(ctx, keycloakClient)
	if err != nil {
		// if the realm is already deleted try to delete finalizer
		if errors.Is(err, helper.ErrKeycloakRealmNotFound) {
			if removeErr := r.helper.TryRemoveFinalizer(ctx, keycloakClient, keyCloakClientOperatorFinalizerName); removeErr != nil {
				return fmt.Errorf("unable to remove finalizer: %w", removeErr)
			}

			return nil
		}

		return fmt.Errorf("unable to create keycloak client from realm ref: %w", err)
	}

	realm, err := r.getKeycloakRealm(ctx, keycloakClient, kClient)
	if err != nil {
		return fmt.Errorf("unable to get keycloak realm: %w", err)
	}

	deleted, err := r.helper.TryToDelete(
		ctx,
		keycloakClient,
		makeTerminator(keycloakClient.Status.ClientID, realm, kClient, objectmeta.PreserveResourcesOnDeletion(keycloakClient)),
		keyCloakClientOperatorFinalizerName,
	)
	if err != nil {
		return fmt.Errorf("deleting keycloak client: %w", err)
	}

	if deleted {
		return nil
	}

	if err = chain.MakeChain(kClient, r.client).Serve(ctx, keycloakClient, realm); err != nil {
		return fmt.Errorf("unable to serve keycloak client: %w", err)
	}

	return nil
}

func (r *ReconcileKeycloakClient) getKeycloakRealm(
	ctx context.Context,
	keycloakClient *keycloakApi.KeycloakClient,
	adapterClient keycloak.Client,
) (string, error) {
	realm, err := r.helper.GetKeycloakRealmFromRef(ctx, keycloakClient, adapterClient)
	if err != nil {
		return "", fmt.Errorf("unable to get keycloak realm from ref: %w", err)
	}

	return gocloak.PString(realm.Realm), nil
}

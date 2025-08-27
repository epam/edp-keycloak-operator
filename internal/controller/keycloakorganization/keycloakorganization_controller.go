package keycloakorganization

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/Nerzal/gocloak/v12"
	"k8s.io/apimachinery/pkg/api/equality"
	k8sErrors "k8s.io/apimachinery/pkg/api/errors"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	"github.com/epam/edp-keycloak-operator/api/common"
	keycloakApi "github.com/epam/edp-keycloak-operator/api/v1alpha1"
	"github.com/epam/edp-keycloak-operator/internal/controller/helper"
	"github.com/epam/edp-keycloak-operator/internal/controller/keycloakorganization/chain"
	"github.com/epam/edp-keycloak-operator/pkg/client/keycloak"
)

type KeycloakProvider interface {
	GetKeycloakRealmFromRef(
		ctx context.Context,
		object helper.ObjectWithRealmRef,
		kcClient keycloak.Client,
	) (*gocloak.RealmRepresentation, error)
	CreateKeycloakClientFromRealmRef(
		ctx context.Context,
		object helper.ObjectWithRealmRef,
	) (keycloak.Client, error)
}

const (
	defaultRequeueTime = time.Second * 30
	successRequeueTime = time.Minute * 10
)

func NewReconcileOrganization(k8sClient client.Client, keycloakProvider KeycloakProvider) *ReconcileOrganization {
	return &ReconcileOrganization{
		client:           k8sClient,
		keycloakProvider: keycloakProvider,
	}
}

// ReconcileOrganization reconciles an Organization object.
type ReconcileOrganization struct {
	client           client.Client
	keycloakProvider KeycloakProvider
}

func (r *ReconcileOrganization) SetupWithManager(mgr ctrl.Manager) error {
	if err := ctrl.NewControllerManagedBy(mgr).
		For(&keycloakApi.KeycloakOrganization{}).
		Complete(r); err != nil {
		return fmt.Errorf("failed to setup KeycloakOrganization controller: %w", err)
	}

	return nil
}

// +kubebuilder:rbac:groups=v1.edp.epam.com,namespace=placeholder,resources=keycloakorganizations,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=v1.edp.epam.com,namespace=placeholder,resources=keycloakorganizations/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=v1.edp.epam.com,namespace=placeholder,resources=keycloakorganizations/finalizers,verbs=update
// +kubebuilder:rbac:groups="",namespace=placeholder,resources=secrets,verbs=get;list;watch

// Reconcile is a loop for reconciling Organization object.
func (r *ReconcileOrganization) Reconcile(ctx context.Context, request reconcile.Request) (result reconcile.Result, resultErr error) {
	log := ctrl.LoggerFrom(ctx)
	log.Info("Reconciling Organization")

	organization, keycloakApiClient, realm, err := r.initializeReconciliation(ctx, request)
	if err != nil {
		return reconcile.Result{}, err
	}

	if organization == nil {
		return reconcile.Result{}, nil
	}

	if organization.GetDeletionTimestamp() != nil {
		return r.handleDeletion(ctx, organization, keycloakApiClient, realm)
	}

	return r.handleReconciliation(ctx, organization, keycloakApiClient, realm)
}

func (r *ReconcileOrganization) initializeReconciliation(ctx context.Context, request reconcile.Request) (*keycloakApi.KeycloakOrganization, keycloak.Client, *gocloak.RealmRepresentation, error) {
	log := ctrl.LoggerFrom(ctx)

	organization := &keycloakApi.KeycloakOrganization{}
	if err := r.client.Get(ctx, request.NamespacedName, organization); err != nil {
		if k8sErrors.IsNotFound(err) {
			return nil, nil, nil, nil
		}

		return nil, nil, nil, fmt.Errorf("failed to get KeycloakOrganization: %w", err)
	}

	keycloakApiClient, err := r.keycloakProvider.CreateKeycloakClientFromRealmRef(ctx, organization)
	if err != nil {
		if errors.Is(err, helper.ErrKeycloakRealmNotFound) {
			if organization.GetDeletionTimestamp() != nil {
				log.Info("Keycloak realm not found, removing finalizer")

				if controllerutil.RemoveFinalizer(organization, common.FinalizerName) {
					if updateErr := r.client.Update(ctx, organization); updateErr != nil {
						return nil, nil, nil, fmt.Errorf("failed to remove finalizer: %w", updateErr)
					}
				}

				log.Info("Finalizer removed")

				return nil, nil, nil, nil
			}
		}

		return nil, nil, nil, fmt.Errorf("failed to create Keycloak client: %w", err)
	}

	realm, err := r.keycloakProvider.GetKeycloakRealmFromRef(ctx, organization, keycloakApiClient)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("unable to get keycloak realm from ref: %w", err)
	}

	return organization, keycloakApiClient, realm, nil
}

func (r *ReconcileOrganization) handleDeletion(ctx context.Context, organization *keycloakApi.KeycloakOrganization, keycloakApiClient keycloak.Client, realm *gocloak.RealmRepresentation) (reconcile.Result, error) {
	if controllerutil.ContainsFinalizer(organization, common.FinalizerName) {
		if err := chain.NewRemoveOrganization(keycloakApiClient).ServeRequest(ctx, organization, realm); err != nil {
			return ctrl.Result{}, fmt.Errorf("failed to remove organization: %w", err)
		}

		controllerutil.RemoveFinalizer(organization, common.FinalizerName)

		if err := r.client.Update(ctx, organization); err != nil {
			return ctrl.Result{}, fmt.Errorf("failed to update organization after finalizer removal: %w", err)
		}
	}

	return ctrl.Result{}, nil
}

func (r *ReconcileOrganization) handleReconciliation(ctx context.Context, organization *keycloakApi.KeycloakOrganization, keycloakApiClient keycloak.Client, realm *gocloak.RealmRepresentation) (reconcile.Result, error) {
	log := ctrl.LoggerFrom(ctx)

	if controllerutil.AddFinalizer(organization, common.FinalizerName) {
		if err := r.client.Update(ctx, organization); err != nil {
			return ctrl.Result{}, fmt.Errorf("failed to add finalizer to organization: %w", err)
		}
	}

	oldStatus := organization.Status.DeepCopy()

	if err := chain.MakeChain(keycloakApiClient, r.client).Serve(ctx, organization, realm); err != nil {
		log.Error(err, "An error has occurred while handling Organization")

		organization.Status.Value = err.Error()

		if statusErr := r.updateOrganizationStatus(ctx, organization, *oldStatus); statusErr != nil {
			return reconcile.Result{}, fmt.Errorf("failed to update organization status: %w", statusErr)
		}

		return reconcile.Result{}, fmt.Errorf("organization chain processing failed: %w", err)
	}

	organization.Status.SetOK()

	if err := r.updateOrganizationStatus(ctx, organization, *oldStatus); err != nil {
		return reconcile.Result{}, err
	}

	return reconcile.Result{
		RequeueAfter: successRequeueTime,
	}, nil
}

func (r *ReconcileOrganization) updateOrganizationStatus(ctx context.Context, organization *keycloakApi.KeycloakOrganization, oldStatus keycloakApi.KeycloakOrganizationStatus) error {
	if equality.Semantic.DeepEqual(&organization.Status, &oldStatus) {
		return nil
	}

	if err := r.client.Status().Update(ctx, organization); err != nil {
		return fmt.Errorf("failed to update Organization status: %w", err)
	}

	return nil
}

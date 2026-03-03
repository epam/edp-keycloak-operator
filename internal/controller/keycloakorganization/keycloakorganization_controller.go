package keycloakorganization

import (
	"context"
	"errors"
	"fmt"
	"time"

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
	keycloakv2 "github.com/epam/edp-keycloak-operator/pkg/client/keycloakv2"
)

type Helper interface {
	CreateKeycloakClientV2FromRealmRef(
		ctx context.Context,
		object helper.ObjectWithRealmRef,
	) (*keycloakv2.KeycloakClient, error)
	GetRealmNameFromRef(
		ctx context.Context,
		object helper.ObjectWithRealmRef,
	) (string, error)
}

const successRequeueTime = time.Minute * 10

func NewReconcileOrganization(k8sClient client.Client, controllerHelper Helper) *ReconcileOrganization {
	return &ReconcileOrganization{
		client: k8sClient,
		helper: controllerHelper,
	}
}

// ReconcileOrganization reconciles an Organization object.
type ReconcileOrganization struct {
	client client.Client
	helper Helper
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

	organization, kClientV2, realmName, err := r.initializeReconciliation(ctx, request)
	if err != nil {
		return reconcile.Result{}, err
	}

	if organization == nil {
		return reconcile.Result{}, nil
	}

	if organization.GetDeletionTimestamp() != nil {
		return r.handleDeletion(ctx, organization, kClientV2, realmName)
	}

	return r.handleReconciliation(ctx, organization, kClientV2, realmName)
}

func (r *ReconcileOrganization) initializeReconciliation(ctx context.Context, request reconcile.Request) (*keycloakApi.KeycloakOrganization, *keycloakv2.KeycloakClient, string, error) {
	log := ctrl.LoggerFrom(ctx)

	organization := &keycloakApi.KeycloakOrganization{}
	if err := r.client.Get(ctx, request.NamespacedName, organization); err != nil {
		if k8sErrors.IsNotFound(err) {
			return nil, nil, "", nil
		}

		return nil, nil, "", fmt.Errorf("failed to get KeycloakOrganization: %w", err)
	}

	kClientV2, err := r.helper.CreateKeycloakClientV2FromRealmRef(ctx, organization)
	if err != nil {
		if errors.Is(err, helper.ErrKeycloakRealmNotFound) {
			if organization.GetDeletionTimestamp() != nil {
				log.Info("Keycloak realm not found, removing finalizer")

				if controllerutil.RemoveFinalizer(organization, common.FinalizerName) {
					if updateErr := r.client.Update(ctx, organization); updateErr != nil {
						return nil, nil, "", fmt.Errorf("failed to remove finalizer: %w", updateErr)
					}
				}

				log.Info("Finalizer removed")

				return nil, nil, "", nil
			}
		}

		return nil, nil, "", fmt.Errorf("failed to create Keycloak client: %w", err)
	}

	realmName, err := r.helper.GetRealmNameFromRef(ctx, organization)
	if err != nil {
		return nil, nil, "", fmt.Errorf("unable to get realm name from ref: %w", err)
	}

	return organization, kClientV2, realmName, nil
}

func (r *ReconcileOrganization) handleDeletion(ctx context.Context, organization *keycloakApi.KeycloakOrganization, kClientV2 *keycloakv2.KeycloakClient, realmName string) (reconcile.Result, error) {
	if controllerutil.ContainsFinalizer(organization, common.FinalizerName) {
		if err := chain.NewRemoveOrganization(kClientV2).ServeRequest(ctx, organization, realmName); err != nil {
			return ctrl.Result{}, fmt.Errorf("failed to remove organization: %w", err)
		}

		controllerutil.RemoveFinalizer(organization, common.FinalizerName)

		if err := r.client.Update(ctx, organization); err != nil {
			return ctrl.Result{}, fmt.Errorf("failed to update organization after finalizer removal: %w", err)
		}
	}

	return ctrl.Result{}, nil
}

func (r *ReconcileOrganization) handleReconciliation(ctx context.Context, organization *keycloakApi.KeycloakOrganization, kClientV2 *keycloakv2.KeycloakClient, realmName string) (reconcile.Result, error) {
	log := ctrl.LoggerFrom(ctx)

	if controllerutil.AddFinalizer(organization, common.FinalizerName) {
		if err := r.client.Update(ctx, organization); err != nil {
			return ctrl.Result{}, fmt.Errorf("failed to add finalizer to organization: %w", err)
		}
	}

	oldStatus := organization.Status.DeepCopy()

	if err := chain.MakeChain(kClientV2).Serve(ctx, organization, realmName); err != nil {
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

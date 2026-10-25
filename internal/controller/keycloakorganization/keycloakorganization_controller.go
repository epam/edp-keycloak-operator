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
	"github.com/epam/edp-keycloak-operator/pkg/client/keycloakapi"
)

type Helper interface {
	CreateKeycloakClientFromRealmRef(
		ctx context.Context,
		object helper.ObjectWithRealmRef,
	) (*keycloakapi.KeycloakClient, error)
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

	organization := &keycloakApi.KeycloakOrganization{}
	if err := r.client.Get(ctx, request.NamespacedName, organization); err != nil {
		if k8sErrors.IsNotFound(err) {
			return reconcile.Result{}, nil
		}

		return reconcile.Result{}, fmt.Errorf("failed to get KeycloakOrganization: %w", err)
	}

	kClient, realmName, stop, err := r.initializeReconciliation(ctx, organization)
	if err != nil {
		if errors.Is(err, helper.ErrKeycloakIsNotAvailable) {
			return reconcile.Result{RequeueAfter: helper.RequeueOnKeycloakNotAvailablePeriod}, nil
		}

		return reconcile.Result{}, r.updateErrorStatus(ctx, organization, err)
	}

	if stop {
		return reconcile.Result{}, nil
	}

	if organization.GetDeletionTimestamp() != nil {
		return r.handleDeletion(ctx, organization, kClient, realmName)
	}

	oldStatus := organization.Status.DeepCopy()

	if updated, err := r.applyDefaults(ctx, organization); err != nil {
		log.Error(err, "An error has occurred while applying Organization defaults")

		organization.Status.Value = err.Error()

		if statusErr := r.updateOrganizationStatus(ctx, organization, *oldStatus); statusErr != nil {
			return reconcile.Result{}, fmt.Errorf("failed to update organization status: %w", statusErr)
		}

		return reconcile.Result{}, err
	} else if updated {
		return reconcile.Result{}, nil
	}

	return r.handleReconciliation(ctx, organization, kClient, realmName)
}

func (r *ReconcileOrganization) applyDefaults(
	ctx context.Context,
	organization *keycloakApi.KeycloakOrganization,
) (bool, error) {
	if !migrateDomains(organization) {
		return false, nil
	}

	if err := r.client.Update(ctx, organization); err != nil {
		return false, fmt.Errorf("failed to migrate domains to domainsV2: %w", err)
	}

	if len(organization.Spec.DomainsV2) == 0 {
		return false, errors.New(
			"spec.domainsV2 was not persisted: upgrade the KeycloakOrganization CRD, Helm does not upgrade CRDs")
	}

	return true, nil
}

// migrateDomains copies domains to domainsV2 when domainsV2 is empty. domains stays unchanged.
//
// Deprecated: exists only for the deprecated domains field.
func migrateDomains(organization *keycloakApi.KeycloakOrganization) bool {
	//nolint:staticcheck // migrates the deprecated domains field to domainsV2
	if len(organization.Spec.DomainsV2) > 0 || len(organization.Spec.Domains) == 0 {
		return false
	}

	//nolint:staticcheck // migrates the deprecated domains field to domainsV2
	for _, name := range organization.Spec.Domains {
		organization.Spec.DomainsV2 = append(organization.Spec.DomainsV2, keycloakApi.OrgDomain{Name: name})
	}

	return true
}

// initializeReconciliation returns stop=true when the finalizer was removed because the realm is gone.
func (r *ReconcileOrganization) initializeReconciliation(ctx context.Context, organization *keycloakApi.KeycloakOrganization) (*keycloakapi.KeycloakClient, string, bool, error) {
	log := ctrl.LoggerFrom(ctx)

	kClient, err := r.helper.CreateKeycloakClientFromRealmRef(ctx, organization)
	if err != nil {
		if errors.Is(err, helper.ErrKeycloakRealmNotFound) {
			if organization.GetDeletionTimestamp() != nil {
				log.Info("Keycloak realm not found, removing finalizer")

				original := organization.DeepCopy()
				if controllerutil.RemoveFinalizer(organization, common.FinalizerName) {
					if updateErr := helper.PatchObject(ctx, r.client, original, organization); updateErr != nil {
						return nil, "", false, fmt.Errorf("failed to remove finalizer: %w", updateErr)
					}
				}

				log.Info("Finalizer removed")

				return nil, "", true, nil
			}
		}

		return nil, "", false, fmt.Errorf("failed to create Keycloak client: %w", err)
	}

	realmName, err := r.helper.GetRealmNameFromRef(ctx, organization)
	if err != nil {
		return nil, "", false, fmt.Errorf("unable to get realm name from ref: %w", err)
	}

	return kClient, realmName, false, nil
}

func (r *ReconcileOrganization) handleDeletion(ctx context.Context, organization *keycloakApi.KeycloakOrganization, kClient *keycloakapi.KeycloakClient, realmName string) (reconcile.Result, error) {
	if controllerutil.ContainsFinalizer(organization, common.FinalizerName) {
		if err := chain.NewRemoveOrganization(kClient).ServeRequest(ctx, organization, realmName); err != nil {
			return ctrl.Result{}, r.updateErrorStatus(ctx, organization, fmt.Errorf("failed to remove organization: %w", err))
		}

		original := organization.DeepCopy()
		controllerutil.RemoveFinalizer(organization, common.FinalizerName)

		if err := helper.PatchObject(ctx, r.client, original, organization); err != nil {
			return ctrl.Result{}, r.updateErrorStatus(ctx, organization, fmt.Errorf("failed to update organization after finalizer removal: %w", err))
		}
	}

	return ctrl.Result{}, nil
}

func (r *ReconcileOrganization) handleReconciliation(ctx context.Context, organization *keycloakApi.KeycloakOrganization, kClient *keycloakapi.KeycloakClient, realmName string) (reconcile.Result, error) {
	log := ctrl.LoggerFrom(ctx)

	original := organization.DeepCopy()
	if controllerutil.AddFinalizer(organization, common.FinalizerName) {
		if err := helper.PatchObject(ctx, r.client, original, organization); err != nil {
			return ctrl.Result{}, r.updateErrorStatus(ctx, organization, fmt.Errorf("failed to add finalizer to organization: %w", err))
		}
	}

	oldStatus := organization.Status.DeepCopy()

	if err := chain.MakeChain(kClient).Serve(ctx, organization, realmName); err != nil {
		log.Error(err, "An error has occurred while handling Organization")

		organization.Status.Value = err.Error()

		if statusErr := r.updateOrganizationStatus(ctx, organization, *oldStatus); statusErr != nil {
			return reconcile.Result{}, fmt.Errorf("failed to update organization status: %w", statusErr)
		}

		return reconcile.Result{}, fmt.Errorf("organization chain processing failed: %w", err)
	}

	organization.Status.SetOK()
	organization.Status.ObservedGeneration = organization.Generation

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

// updateErrorStatus writes err to status.value and returns err, or the status write error.
func (r *ReconcileOrganization) updateErrorStatus(ctx context.Context, organization *keycloakApi.KeycloakOrganization, err error) error {
	ctrl.LoggerFrom(ctx).Error(err, "An error has occurred while handling Organization")

	oldStatus := organization.Status
	organization.Status.Value = err.Error()

	if statusErr := r.updateOrganizationStatus(ctx, organization, oldStatus); statusErr != nil {
		return fmt.Errorf("failed to update organization status: %w", statusErr)
	}

	return err
}

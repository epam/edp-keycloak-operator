package keycloakrealmcomponent

import (
	"context"
	"errors"
	"fmt"
	"time"

	"k8s.io/apimachinery/pkg/api/equality"
	k8sErrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/utils/ptr"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/apiutil"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	"github.com/epam/edp-keycloak-operator/api/common"
	keycloakApi "github.com/epam/edp-keycloak-operator/api/v1"
	"github.com/epam/edp-keycloak-operator/internal/controller/helper"
	"github.com/epam/edp-keycloak-operator/internal/controller/keycloakrealmcomponent/chain"
	keycloakv2 "github.com/epam/edp-keycloak-operator/pkg/client/keycloakv2"
)

const (
	successRequeueTime = time.Minute * 10

	// legacyFinalizerName is the old finalizer used before migration to common.FinalizerName.
	// Kept to ensure existing resources carrying the old finalizer can be deleted cleanly.
	legacyFinalizerName = "keycloak.realmcomponent.operator.finalizer.name"
)

type RealmComponentHelper interface {
	SetRealmOwnerRef(ctx context.Context, object helper.ObjectWithRealmRef) error
	GetRealmNameFromRef(ctx context.Context, object helper.ObjectWithRealmRef) (string, error)
	CreateKeycloakClientV2FromRealmRef(ctx context.Context, object helper.ObjectWithRealmRef) (*keycloakv2.KeycloakClient, error)
}

type RealmComponentReconciler struct {
	client          client.Client
	helper          RealmComponentHelper
	secretRefClient chain.SecretRefClient
	scheme          *runtime.Scheme
}

func NewRealmComponentReconciler(
	k8sClient client.Client,
	scheme *runtime.Scheme,
	controllerHelper RealmComponentHelper,
	secretRefClient chain.SecretRefClient,
) *RealmComponentReconciler {
	return &RealmComponentReconciler{
		client:          k8sClient,
		scheme:          scheme,
		helper:          controllerHelper,
		secretRefClient: secretRefClient,
	}
}

func (r *RealmComponentReconciler) SetupWithManager(mgr ctrl.Manager) error {
	if err := ctrl.NewControllerManagedBy(mgr).
		For(&keycloakApi.KeycloakRealmComponent{}).
		Complete(r); err != nil {
		return fmt.Errorf("failed to setup KeycloakRealmComponent controller: %w", err)
	}

	return nil
}

// +kubebuilder:rbac:groups=v1.edp.epam.com,namespace=placeholder,resources=keycloakrealmcomponents,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=v1.edp.epam.com,namespace=placeholder,resources=keycloakrealmcomponents/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=v1.edp.epam.com,namespace=placeholder,resources=keycloakrealmcomponents/finalizers,verbs=update

// Reconcile is a loop for reconciling KeycloakRealmComponent object.
func (r *RealmComponentReconciler) Reconcile(ctx context.Context, request reconcile.Request) (reconcile.Result, error) {
	log := ctrl.LoggerFrom(ctx)
	log.Info("Reconciling KeycloakRealmComponent")

	instance, kClientV2, realmName, err := r.initializeReconciliation(ctx, request)
	if err != nil {
		if errors.Is(err, helper.ErrKeycloakIsNotAvailable) {
			return ctrl.Result{RequeueAfter: helper.RequeueOnKeycloakNotAvailablePeriod}, nil
		}

		return reconcile.Result{}, err
	}

	if instance == nil {
		return reconcile.Result{}, nil
	}

	if instance.GetDeletionTimestamp() != nil {
		return r.handleDeletion(ctx, instance, kClientV2, realmName)
	}

	return r.handleReconciliation(ctx, instance, kClientV2, realmName)
}

func (r *RealmComponentReconciler) initializeReconciliation(
	ctx context.Context,
	request reconcile.Request,
) (*keycloakApi.KeycloakRealmComponent, *keycloakv2.KeycloakClient, string, error) {
	instance := &keycloakApi.KeycloakRealmComponent{}
	if err := r.client.Get(ctx, request.NamespacedName, instance); err != nil {
		if k8sErrors.IsNotFound(err) {
			return nil, nil, "", nil
		}

		return nil, nil, "", fmt.Errorf("failed to get KeycloakRealmComponent: %w", err)
	}

	if err := r.helper.SetRealmOwnerRef(ctx, instance); err != nil {
		return nil, nil, "", fmt.Errorf("unable to set realm owner ref: %w", err)
	}

	if instance.GetDeletionTimestamp() == nil {
		if err := r.setComponentOwnerReference(ctx, instance); err != nil {
			return nil, nil, "", fmt.Errorf("unable to set component owner reference: %w", err)
		}
	}

	kClientV2, err := r.helper.CreateKeycloakClientV2FromRealmRef(ctx, instance)
	if err != nil {
		if errors.Is(err, helper.ErrKeycloakRealmNotFound) && instance.GetDeletionTimestamp() != nil {
			stop, removeErr := helper.RemoveFinalizersOnRealmNotFound(ctx, r.client, instance, common.FinalizerName, legacyFinalizerName)
			if removeErr != nil {
				return nil, nil, "", removeErr
			}

			if stop {
				return nil, nil, "", nil
			}
		}

		return nil, nil, "", fmt.Errorf("failed to create Keycloak client: %w", err)
	}

	realmName, err := r.helper.GetRealmNameFromRef(ctx, instance)
	if err != nil {
		return nil, nil, "", fmt.Errorf("unable to get realm name from ref: %w", err)
	}

	return instance, kClientV2, realmName, nil
}

func (r *RealmComponentReconciler) handleDeletion(
	ctx context.Context,
	instance *keycloakApi.KeycloakRealmComponent,
	kClientV2 *keycloakv2.KeycloakClient,
	realmName string,
) (reconcile.Result, error) {
	if controllerutil.ContainsFinalizer(instance, common.FinalizerName) ||
		controllerutil.ContainsFinalizer(instance, legacyFinalizerName) {
		if err := chain.NewRemoveComponent(kClientV2).Serve(ctx, instance, realmName); err != nil {
			return ctrl.Result{}, fmt.Errorf("failed to remove realm component: %w", err)
		}

		controllerutil.RemoveFinalizer(instance, common.FinalizerName)
		controllerutil.RemoveFinalizer(instance, legacyFinalizerName)

		if err := r.client.Update(ctx, instance); err != nil {
			return ctrl.Result{}, fmt.Errorf("failed to update KeycloakRealmComponent after finalizer removal: %w", err)
		}
	}

	return ctrl.Result{}, nil
}

func (r *RealmComponentReconciler) handleReconciliation(
	ctx context.Context,
	instance *keycloakApi.KeycloakRealmComponent,
	kClientV2 *keycloakv2.KeycloakClient,
	realmName string,
) (reconcile.Result, error) {
	log := ctrl.LoggerFrom(ctx)

	if controllerutil.AddFinalizer(instance, common.FinalizerName) {
		if err := r.client.Update(ctx, instance); err != nil {
			return ctrl.Result{}, fmt.Errorf("failed to add finalizer to KeycloakRealmComponent: %w", err)
		}
	}

	oldStatus := instance.Status

	if err := chain.MakeChain(r.client, kClientV2, r.secretRefClient).Serve(ctx, instance, realmName); err != nil {
		log.Error(err, "An error has occurred while handling KeycloakRealmComponent")

		resultErr := fmt.Errorf("realm component chain processing failed: %w", err)
		instance.Status.Value = resultErr.Error()

		if statusErr := r.updateStatus(ctx, instance, oldStatus); statusErr != nil {
			return reconcile.Result{}, fmt.Errorf("failed to update KeycloakRealmComponent status: %w", statusErr)
		}

		return reconcile.Result{}, resultErr
	}

	instance.Status.Value = common.StatusOK

	if err := r.updateStatus(ctx, instance, oldStatus); err != nil {
		return reconcile.Result{}, err
	}

	return reconcile.Result{RequeueAfter: successRequeueTime}, nil
}

func (r *RealmComponentReconciler) updateStatus(
	ctx context.Context,
	instance *keycloakApi.KeycloakRealmComponent,
	oldStatus keycloakApi.KeycloakComponentStatus,
) error {
	if equality.Semantic.DeepEqual(&instance.Status, &oldStatus) {
		return nil
	}

	if err := r.client.Status().Update(ctx, instance); err != nil {
		return fmt.Errorf("failed to update KeycloakRealmComponent status: %w", err)
	}

	return nil
}

// setComponentOwnerReference sets the owner reference for the component.
// In case the component has a parent component, we need to set owner reference to it
// to trigger the deletion of the child KeycloakRealmComponent.
// In the keycloak API side child component is automatically deleted,
// so we need to do the same with the KeycloakRealmComponent resource.
func (r *RealmComponentReconciler) setComponentOwnerReference(
	ctx context.Context,
	component *keycloakApi.KeycloakRealmComponent,
) error {
	if component.Spec.ParentRef == nil || component.Spec.ParentRef.Kind != keycloakApi.KeycloakRealmComponentKind {
		return nil
	}

	for _, ref := range component.GetOwnerReferences() {
		if ref.Kind == keycloakApi.KeycloakRealmComponentKind {
			return nil
		}
	}

	parentComponent := &keycloakApi.KeycloakRealmComponent{}
	if err := r.client.Get(ctx, types.NamespacedName{
		Name:      component.Spec.ParentRef.Name,
		Namespace: component.GetNamespace(),
	}, parentComponent); err != nil {
		return fmt.Errorf("unable to get parent component: %w", err)
	}

	gvk, err := apiutil.GVKForObject(parentComponent, r.scheme)
	if err != nil {
		return fmt.Errorf("unable to get gvk for parent component: %w", err)
	}

	ref := metav1.OwnerReference{
		APIVersion:         gvk.GroupVersion().String(),
		Kind:               gvk.Kind,
		Name:               parentComponent.GetName(),
		UID:                parentComponent.GetUID(),
		BlockOwnerDeletion: ptr.To(true),
		Controller:         ptr.To(false),
	}
	component.SetOwnerReferences(append(component.GetOwnerReferences(), ref))

	if err := r.client.Update(ctx, component); err != nil {
		return fmt.Errorf("failed to set owner reference %s: %w", parentComponent.Name, err)
	}

	return nil
}

package keycloakrealmcomponent

import (
	"context"
	"errors"
	"fmt"
	"time"

	"k8s.io/apimachinery/pkg/api/equality"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/utils/ptr"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/apiutil"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	"github.com/epam/edp-keycloak-operator/api/common"
	keycloakApi "github.com/epam/edp-keycloak-operator/api/v1"
	"github.com/epam/edp-keycloak-operator/internal/controller/helper"
	"github.com/epam/edp-keycloak-operator/internal/controller/keycloakrealmcomponent/chain"
	"github.com/epam/edp-keycloak-operator/pkg/client/keycloakapi"
)

const (
	kind = keycloakApi.KeycloakRealmComponentKind

	successRequeueTime = time.Minute * 10

	// parentNotReadyRequeueTime is the fallback when a parent is expected to appear; the watch on
	// the parent's status.id requeues the child earlier.
	parentNotReadyRequeueTime = time.Second * 30

	// legacyFinalizerName is the old finalizer used before migration to common.FinalizerName.
	// Kept to ensure existing resources carrying the old finalizer can be deleted cleanly.
	legacyFinalizerName = "keycloak.realmcomponent.operator.finalizer.name"

	// parentComponentIndexField indexes KeycloakRealmComponents by the parent CR name
	// in spec.parentRef; set only for parentRef.kind KeycloakRealmComponent.
	parentComponentIndexField = "spec.parentRef.name"
)

type RealmComponentHelper interface {
	SetRealmOwnerRef(ctx context.Context, object helper.ObjectWithRealmRef) error
	GetRealmNameFromRef(ctx context.Context, object helper.ObjectWithRealmRef) (string, error)
	CreateKeycloakClientFromRealmRef(ctx context.Context, object helper.ObjectWithRealmRef) (*keycloakapi.KeycloakClient, error)
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
	if err := mgr.GetFieldIndexer().IndexField(
		context.Background(),
		&keycloakApi.KeycloakRealmComponent{},
		parentComponentIndexField,
		indexParentComponent,
	); err != nil {
		return fmt.Errorf("failed to index KeycloakRealmComponent by %s: %w", parentComponentIndexField, err)
	}

	if err := ctrl.NewControllerManagedBy(mgr).
		For(&keycloakApi.KeycloakRealmComponent{}).
		Watches(
			&keycloakApi.KeycloakRealmComponent{},
			handler.EnqueueRequestsFromMapFunc(r.childComponentRequests),
			builder.WithPredicates(parentIDChangedPredicate()),
		).
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
	return helper.ReconcileRealmRef(ctx, r.client, request, &keycloakApi.KeycloakRealmComponent{}, helper.RealmRefReconcile[*keycloakApi.KeycloakRealmComponent]{
		Kind:                 kind,
		Helper:               r.helper,
		Finalizers:           []string{common.FinalizerName, legacyFinalizerName},
		Remove:               r.removeFromKeycloak,
		HandleReconciliation: r.handleReconciliation,
	})
}

func (r *RealmComponentReconciler) removeFromKeycloak(ctx context.Context, instance *keycloakApi.KeycloakRealmComponent, kClient *keycloakapi.KeycloakClient, realmName string) error {
	if err := chain.NewRemoveComponent(r.client, kClient).Serve(ctx, instance, realmName); err != nil {
		return fmt.Errorf("failed to remove realm component: %w", err)
	}

	return nil
}

func (r *RealmComponentReconciler) handleReconciliation(
	ctx context.Context,
	instance *keycloakApi.KeycloakRealmComponent,
	kClient *keycloakapi.KeycloakClient,
	realmName string,
) (reconcile.Result, error) {
	log := ctrl.LoggerFrom(ctx)

	original := instance.DeepCopy()
	if controllerutil.AddFinalizer(instance, common.FinalizerName) {
		if err := helper.PatchObject(ctx, r.client, original, instance); err != nil {
			return ctrl.Result{}, helper.UpdateErrorStatus(ctx, r.client, kind, instance, fmt.Errorf("failed to add finalizer to KeycloakRealmComponent: %w", err))
		}
	}

	oldStatus := instance.Status

	parent, err := chain.ResolveParent(ctx, r.client, kClient, instance, realmName)
	if err != nil {
		resultErr := fmt.Errorf("unable to resolve parent: %w", err)
		instance.Status.Value = resultErr.Error()

		if statusErr := r.updateStatus(ctx, instance, oldStatus); statusErr != nil {
			return reconcile.Result{}, fmt.Errorf("failed to update KeycloakRealmComponent status: %w", statusErr)
		}

		if errors.Is(err, chain.ErrParentNotReady) {
			log.Info("Waiting for parent component", "reason", err.Error())

			return reconcile.Result{RequeueAfter: parentNotReadyRequeueTime}, nil
		}

		return reconcile.Result{}, resultErr
	}

	// Runs before the chain: the patch decodes the server object into instance, status included.
	if err := r.syncParentOwnerReference(ctx, instance, parent.CR); err != nil {
		return reconcile.Result{}, helper.UpdateErrorStatus(ctx, r.client, kind, instance, err)
	}

	if err := chain.MakeChain(r.client, kClient, r.secretRefClient, parent.ID).Serve(ctx, instance, realmName); err != nil {
		resultErr := fmt.Errorf("realm component chain processing failed: %w", err)
		instance.Status.Value = resultErr.Error()

		if statusErr := r.updateStatus(ctx, instance, oldStatus); statusErr != nil {
			return reconcile.Result{}, fmt.Errorf("failed to update KeycloakRealmComponent status: %w", statusErr)
		}

		return reconcile.Result{}, resultErr
	}

	instance.Status.Value = common.StatusOK
	instance.Status.ObservedGeneration = instance.Generation

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

// syncParentOwnerReference makes the KeycloakRealmComponent owner references of component
// exactly one reference to parent, or none when parent is nil. Other owner references are kept.
// Kubernetes garbage-collects the child CR with its parent, as Keycloak deletes child components.
func (r *RealmComponentReconciler) syncParentOwnerReference(
	ctx context.Context,
	component *keycloakApi.KeycloakRealmComponent,
	parent *keycloakApi.KeycloakRealmComponent,
) error {
	gvk, err := apiutil.GVKForObject(component, r.scheme)
	if err != nil {
		return fmt.Errorf("unable to get gvk for KeycloakRealmComponent: %w", err)
	}

	apiVersion := gvk.GroupVersion().String()

	var parentRef *metav1.OwnerReference

	if parent != nil {
		parentRef = &metav1.OwnerReference{
			APIVersion:         apiVersion,
			Kind:               gvk.Kind,
			Name:               parent.GetName(),
			UID:                parent.GetUID(),
			BlockOwnerDeletion: ptr.To(true),
			Controller:         ptr.To(false),
		}
	}

	current := component.GetOwnerReferences()

	var desired []metav1.OwnerReference

	for _, ref := range current {
		if ref.APIVersion != apiVersion || ref.Kind != gvk.Kind {
			desired = append(desired, ref)

			continue
		}

		// The existing parent reference keeps its position.
		if parentRef != nil && ref.UID == parentRef.UID && ref.Name == parentRef.Name {
			desired = append(desired, *parentRef)
			parentRef = nil
		}
	}

	if parentRef != nil {
		desired = append(desired, *parentRef)
	}

	if equality.Semantic.DeepEqual(current, desired) {
		return nil
	}

	original := component.DeepCopy()
	component.SetOwnerReferences(desired)

	if err := helper.PatchObject(ctx, r.client, original, component); err != nil {
		return fmt.Errorf("failed to sync parent component owner reference: %w", err)
	}

	return nil
}

// indexParentComponent returns the parent CR name for parentRef.kind KeycloakRealmComponent.
func indexParentComponent(obj client.Object) []string {
	component, ok := obj.(*keycloakApi.KeycloakRealmComponent)
	if !ok || component.Spec.ParentRef == nil ||
		component.Spec.ParentRef.Kind != keycloakApi.KeycloakRealmComponentKind {
		return nil
	}

	return []string{component.Spec.ParentRef.Name}
}

// childComponentRequests enqueues the KeycloakRealmComponents whose parentRef names obj.
func (r *RealmComponentReconciler) childComponentRequests(ctx context.Context, obj client.Object) []reconcile.Request {
	children := &keycloakApi.KeycloakRealmComponentList{}
	if err := r.client.List(ctx, children,
		client.InNamespace(obj.GetNamespace()),
		client.MatchingFields{parentComponentIndexField: obj.GetName()},
	); err != nil {
		ctrl.LoggerFrom(ctx).Error(err, "Unable to list child KeycloakRealmComponents", "parent", obj.GetName())

		return nil
	}

	requests := make([]reconcile.Request, 0, len(children.Items))
	for i := range children.Items {
		requests = append(requests, reconcile.Request{NamespacedName: types.NamespacedName{
			Namespace: children.Items[i].Namespace,
			Name:      children.Items[i].Name,
		}})
	}

	return requests
}

// parentIDChangedPredicate passes update events that change status.id.
func parentIDChangedPredicate() predicate.Funcs {
	return predicate.Funcs{
		CreateFunc:  func(event.CreateEvent) bool { return false },
		DeleteFunc:  func(event.DeleteEvent) bool { return false },
		GenericFunc: func(event.GenericEvent) bool { return false },
		UpdateFunc: func(e event.UpdateEvent) bool {
			oldComponent, okOld := e.ObjectOld.(*keycloakApi.KeycloakRealmComponent)
			newComponent, okNew := e.ObjectNew.(*keycloakApi.KeycloakRealmComponent)

			return okOld && okNew && oldComponent.Status.ID != newComponent.Status.ID
		},
	}
}

package keycloakrealmcomponent

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"time"

	"github.com/Nerzal/gocloak/v12"
	k8sErrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/utils/pointer"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/apiutil"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	"github.com/epam/edp-keycloak-operator/api/common"
	keycloakApi "github.com/epam/edp-keycloak-operator/api/v1"
	"github.com/epam/edp-keycloak-operator/controllers/helper"
	"github.com/epam/edp-keycloak-operator/pkg/client/keycloak"
	"github.com/epam/edp-keycloak-operator/pkg/client/keycloak/adapter"
	"github.com/epam/edp-keycloak-operator/pkg/objectmeta"
)

const finalizerName = "keycloak.realmcomponent.operator.finalizer.name"

type Helper interface {
	SetFailureCount(fc helper.FailureCountable) time.Duration
	TryToDelete(ctx context.Context, obj client.Object, terminator helper.Terminator, finalizer string) (isDeleted bool, resultErr error)
	SetRealmOwnerRef(ctx context.Context, object helper.ObjectWithRealmRef) error
	GetKeycloakRealmFromRef(ctx context.Context, object helper.ObjectWithRealmRef, kcClient keycloak.Client) (*gocloak.RealmRepresentation, error)
	CreateKeycloakClientFromRealmRef(ctx context.Context, object helper.ObjectWithRealmRef) (keycloak.Client, error)
}

type Reconcile struct {
	client                  client.Client
	helper                  Helper
	successReconcileTimeout time.Duration
	scheme                  *runtime.Scheme
}

func NewReconcile(client client.Client, scheme *runtime.Scheme, helper Helper) *Reconcile {
	return &Reconcile{
		client: client,
		scheme: scheme,
		helper: helper,
	}
}

func (r *Reconcile) SetupWithManager(mgr ctrl.Manager, successReconcileTimeout time.Duration) error {
	r.successReconcileTimeout = successReconcileTimeout

	pred := predicate.Funcs{
		UpdateFunc: isSpecUpdated,
	}

	err := ctrl.NewControllerManagedBy(mgr).
		For(&keycloakApi.KeycloakRealmComponent{}, builder.WithPredicates(pred)).
		Complete(r)
	if err != nil {
		return fmt.Errorf("failed to setup keycloakRealmComponent controller: %w", err)
	}

	return nil
}

func isSpecUpdated(e event.UpdateEvent) bool {
	oo, ok := e.ObjectOld.(*keycloakApi.KeycloakRealmComponent)
	if !ok {
		return false
	}

	no, ok := e.ObjectNew.(*keycloakApi.KeycloakRealmComponent)
	if !ok {
		return false
	}

	return !reflect.DeepEqual(oo.Spec, no.Spec) ||
		(oo.GetDeletionTimestamp().IsZero() && !no.GetDeletionTimestamp().IsZero())
}

//+kubebuilder:rbac:groups=v1.edp.epam.com,namespace=placeholder,resources=keycloakrealmcomponents,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=v1.edp.epam.com,namespace=placeholder,resources=keycloakrealmcomponents/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=v1.edp.epam.com,namespace=placeholder,resources=keycloakrealmcomponents/finalizers,verbs=update

// Reconcile is a loop for reconciling KeycloakRealmComponent object.
// nolint:cyclop
// nolint:funlen
func (r *Reconcile) Reconcile(ctx context.Context, request reconcile.Request) (reconcile.Result, error) {
	log := ctrl.LoggerFrom(ctx)
	log.Info("Reconciling KeycloakRealmComponent")

	keycloakRealmComponent := &keycloakApi.KeycloakRealmComponent{}
	if err := r.client.Get(ctx, request.NamespacedName, keycloakRealmComponent); err != nil {
		if k8sErrors.IsNotFound(err) {
			return reconcile.Result{}, nil
		}

		return ctrl.Result{}, fmt.Errorf("unable to get KeycloakRealmComponent: %w", err)
	}

	if updated, err := r.applyDefaults(ctx, keycloakRealmComponent); err != nil {
		return ctrl.Result{}, fmt.Errorf("unable to apply defaults: %w", err)
	} else if updated {
		return ctrl.Result{}, nil
	}

	err := r.helper.SetRealmOwnerRef(ctx, keycloakRealmComponent)
	if err != nil {
		return ctrl.Result{}, fmt.Errorf("unable to get realm owner ref: %w", err)
	}

	if err = r.setComponentOwnerReference(ctx, keycloakRealmComponent); err != nil {
		return reconcile.Result{}, err
	}

	kClient, err := r.helper.CreateKeycloakClientFromRealmRef(ctx, keycloakRealmComponent)
	if err != nil {
		if errors.Is(err, helper.ErrKeycloakIsNotAvailable) {
			return ctrl.Result{
				RequeueAfter: helper.RequeueOnKeycloakNotAvailablePeriod,
			}, nil
		}

		return ctrl.Result{}, fmt.Errorf("unable to create keycloak client: %w", err)
	}

	realm, err := r.helper.GetKeycloakRealmFromRef(ctx, keycloakRealmComponent, kClient)
	if err != nil {
		return ctrl.Result{}, fmt.Errorf("unable to get realm: %w", err)
	}

	term := makeTerminator(
		gocloak.PString(realm.Realm),
		keycloakRealmComponent.Spec.Name,
		kClient,
		objectmeta.PreserveResourcesOnDeletion(keycloakRealmComponent),
	)

	if deleted, err := r.helper.TryToDelete(ctx, keycloakRealmComponent, term, finalizerName); err != nil {
		return ctrl.Result{}, fmt.Errorf("unable to tryToDelete realm component %w", err)
	} else if deleted {
		return reconcile.Result{}, nil
	}

	if err := r.tryReconcile(ctx, keycloakRealmComponent, gocloak.PString(realm.Realm), kClient); err != nil {
		keycloakRealmComponent.Status.Value = err.Error()
		requeueAfter := r.helper.SetFailureCount(keycloakRealmComponent)

		if statusErr := r.client.Status().Update(ctx, keycloakRealmComponent); statusErr != nil {
			return ctrl.Result{}, fmt.Errorf("unable to update KeycloakRealmComponent status: %w", statusErr)
		}

		return ctrl.Result{
			RequeueAfter: requeueAfter,
		}, fmt.Errorf("unable to reconcile KeycloakRealmComponent: %w", err)
	}

	helper.SetSuccessStatus(keycloakRealmComponent)

	if err := r.client.Status().Update(ctx, keycloakRealmComponent); err != nil {
		return ctrl.Result{}, fmt.Errorf("unable to update KeycloakRealmComponent status: %w", err)
	}

	return reconcile.Result{}, nil
}

func (r *Reconcile) tryReconcile(
	ctx context.Context,
	keycloakRealmComponent *keycloakApi.KeycloakRealmComponent,
	realmName string,
	kClient keycloak.Client,
) error {
	keycloakComponent, err := r.createKeycloakComponent(ctx, keycloakRealmComponent, realmName, kClient)
	if err != nil {
		return fmt.Errorf("unable to create keycloak component: %w", err)
	}

	cmp, err := kClient.GetComponent(ctx, realmName, keycloakRealmComponent.Spec.Name)
	if err != nil {
		if !adapter.IsErrNotFound(err) {
			return fmt.Errorf("unable to get component, unexpected error: %w", err)
		}

		if err := kClient.CreateComponent(ctx, realmName, keycloakComponent); err != nil {
			return fmt.Errorf("unable to create component %w", err)
		}

		return nil
	}

	keycloakComponent.ID = cmp.ID

	if err := kClient.UpdateComponent(ctx, realmName, keycloakComponent); err != nil {
		return fmt.Errorf("unable to update component: %w", err)
	}

	return nil
}

func (r *Reconcile) createKeycloakComponent(
	ctx context.Context,
	component *keycloakApi.KeycloakRealmComponent,
	kcRealmName string,
	kClient keycloak.Client,
) (*adapter.Component, error) {
	ksComponent := &adapter.Component{
		Name:         component.Spec.Name,
		Config:       component.Spec.Config,
		ProviderID:   component.Spec.ProviderID,
		ProviderType: component.Spec.ProviderType,
	}

	parenID, err := r.getParentID(ctx, component, kcRealmName, kClient)
	if err != nil {
		return nil, fmt.Errorf("unable to get parent id: %w", err)
	}

	if parenID != "" {
		ksComponent.ParentID = parenID
	}

	return ksComponent, nil
}

func (r *Reconcile) getParentID(
	ctx context.Context,
	component *keycloakApi.KeycloakRealmComponent,
	kcRealmName string,
	kClient keycloak.Client,
) (string, error) {
	if component.Spec.ParentRef == nil {
		return "", nil
	}

	if component.Spec.ParentRef.Kind == keycloakApi.KeycloakRealmKind {
		parentRealm := &keycloakApi.KeycloakRealm{}
		if err := r.client.Get(ctx, types.NamespacedName{Name: component.Spec.ParentRef.Name, Namespace: component.GetNamespace()}, parentRealm); err != nil {
			return "", fmt.Errorf("unable to get parent kcRealmName: %w", err)
		}

		kcParentRealm, err := kClient.GetRealm(ctx, parentRealm.Spec.RealmName)
		if err != nil {
			return "", fmt.Errorf("unable to get parent kcRealmName: %w", err)
		}

		if kcParentRealm.ID == nil || *kcParentRealm.ID == "" {
			return "", fmt.Errorf("kcRealmName id is empty")
		}

		return *kcParentRealm.ID, nil
	}

	if component.Spec.ParentRef.Kind == keycloakApi.KeycloakRealmComponentKind {
		parentComponent := &keycloakApi.KeycloakRealmComponent{}
		if err := r.client.Get(ctx, types.NamespacedName{Name: component.Spec.ParentRef.Name, Namespace: component.GetNamespace()}, parentComponent); err != nil {
			return "", fmt.Errorf("unable to get parent component: %w", err)
		}

		kcParentComponent, err := kClient.GetComponent(ctx, kcRealmName, parentComponent.Spec.Name)
		if err != nil {
			return "", fmt.Errorf("unable to get parent component: %w", err)
		}

		return kcParentComponent.ID, nil
	}

	return "", fmt.Errorf("parent kind %s is not supported", component.Spec.ParentRef.Kind)
}

// setComponentOwnerReference sets the owner reference for the component.
// In case the component has a parent component, we need to set owner reference to it
// to trigger the deletion of the child KeycloakRealmComponent.
// In the keycloak API side child component is automatically deleted,
// so we need to do the same with the KeycloakRealmComponent resource.
func (r *Reconcile) setComponentOwnerReference(
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
	if err := r.client.Get(ctx, types.NamespacedName{Name: component.Spec.ParentRef.Name, Namespace: component.GetNamespace()}, parentComponent); err != nil {
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
		BlockOwnerDeletion: pointer.Bool(true),
		Controller:         pointer.Bool(true),
	}
	component.SetOwnerReferences([]v1.OwnerReference{ref})

	if err := r.client.Update(ctx, component); err != nil {
		return fmt.Errorf("failed to set owner reference %s: %w", parentComponent.Name, err)
	}

	return nil
}

func (r *Reconcile) applyDefaults(ctx context.Context, instance *keycloakApi.KeycloakRealmComponent) (bool, error) {
	if instance.Spec.RealmRef.Name == "" {
		instance.Spec.RealmRef = common.RealmRef{
			Kind: keycloakApi.KeycloakRealmKind,
			Name: instance.Spec.Realm,
		}

		if err := r.client.Update(ctx, instance); err != nil {
			return false, fmt.Errorf("failed to update default values: %w", err)
		}

		return true, nil
	}

	return false, nil
}

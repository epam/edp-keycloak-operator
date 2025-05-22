package keycloakclientscope

import (
	"context"
	"fmt"
	"maps"
	"reflect"
	"time"

	"github.com/Nerzal/gocloak/v12"
	"github.com/pkg/errors"
	k8sErrors "k8s.io/apimachinery/pkg/api/errors"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	keycloakApi "github.com/epam/edp-keycloak-operator/api/v1"
	"github.com/epam/edp-keycloak-operator/internal/controller/helper"
	"github.com/epam/edp-keycloak-operator/pkg/client/keycloak"
	"github.com/epam/edp-keycloak-operator/pkg/client/keycloak/adapter"
	"github.com/epam/edp-keycloak-operator/pkg/objectmeta"
)

const finalizerName = "keycloak.clientscope.operator.finalizer.name"

type Helper interface {
	SetFailureCount(fc helper.FailureCountable) time.Duration
	TryRemoveFinalizer(ctx context.Context, obj client.Object, finalizer string) error
	TryToDelete(ctx context.Context, obj client.Object, terminator helper.Terminator, finalizer string) (isDeleted bool, resultErr error)
	SetRealmOwnerRef(ctx context.Context, object helper.ObjectWithRealmRef) error
	GetKeycloakRealmFromRef(ctx context.Context, object helper.ObjectWithRealmRef, kcClient keycloak.Client) (*gocloak.RealmRepresentation, error)
	CreateKeycloakClientFromRealmRef(ctx context.Context, object helper.ObjectWithRealmRef) (keycloak.Client, error)
}

type Reconcile struct {
	client client.Client
	helper Helper
}

func NewReconcile(client client.Client, helper Helper) *Reconcile {
	return &Reconcile{
		client: client,
		helper: helper,
	}
}

func (r *Reconcile) SetupWithManager(mgr ctrl.Manager) error {
	pred := predicate.Funcs{
		UpdateFunc: isSpecUpdated,
	}

	err := ctrl.NewControllerManagedBy(mgr).
		For(&keycloakApi.KeycloakClientScope{}, builder.WithPredicates(pred)).
		Complete(r)
	if err != nil {
		return fmt.Errorf("failed to setup KeycloakClientScope controller: %w", err)
	}

	return nil
}

func isSpecUpdated(e event.UpdateEvent) bool {
	oo, ok := e.ObjectOld.(*keycloakApi.KeycloakClientScope)
	if !ok {
		return false
	}

	no, ok := e.ObjectNew.(*keycloakApi.KeycloakClientScope)
	if !ok {
		return false
	}

	return !reflect.DeepEqual(oo.Spec, no.Spec) ||
		(oo.GetDeletionTimestamp().IsZero() && !no.GetDeletionTimestamp().IsZero())
}

//+kubebuilder:rbac:groups=v1.edp.epam.com,namespace=placeholder,resources=keycloakclientscopes,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=v1.edp.epam.com,namespace=placeholder,resources=keycloakclientscopes/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=v1.edp.epam.com,namespace=placeholder,resources=keycloakclientscopes/finalizers,verbs=update

// Reconcile is a loop for reconciling KeycloakClientScope object.
func (r *Reconcile) Reconcile(ctx context.Context, request reconcile.Request) (reconcile.Result, error) {
	log := ctrl.LoggerFrom(ctx)
	log.Info("Reconciling KeycloakClientScope")

	scope := &keycloakApi.KeycloakClientScope{}
	if err := r.client.Get(ctx, request.NamespacedName, scope); err != nil {
		if k8sErrors.IsNotFound(err) {
			return reconcile.Result{}, nil
		}

		return reconcile.Result{}, fmt.Errorf("unable to get keycloak client scope from k8s: %w", err)
	}

	oldStatus := scope.Status

	scopeID, err := r.tryReconcile(ctx, scope)
	if err != nil {
		if errors.Is(err, helper.ErrKeycloakIsNotAvailable) {
			return helper.RequeueOnKeycloakNotAvailable, nil
		}

		scope.Status.Value = err.Error()

		if statusErr := r.updateKeycloakClientScopeStatus(ctx, scope, oldStatus); statusErr != nil {
			return reconcile.Result{}, statusErr
		}

		return reconcile.Result{}, err
	}

	scope.Status.Value = helper.StatusOK
	scope.Status.ID = scopeID

	if statusErr := r.updateKeycloakClientScopeStatus(ctx, scope, oldStatus); statusErr != nil {
		return reconcile.Result{}, statusErr
	}

	log.Info("Reconciling KeycloakClientScope done")

	return reconcile.Result{}, nil
}

func (r *Reconcile) tryReconcile(ctx context.Context, instance *keycloakApi.KeycloakClientScope) (string, error) {
	err := r.helper.SetRealmOwnerRef(ctx, instance)
	if err != nil {
		return "", fmt.Errorf("unable to set realm owner ref: %w", err)
	}

	cl, err := r.helper.CreateKeycloakClientFromRealmRef(ctx, instance)
	if err != nil {
		// if the realm is already deleted try to delete finalizer
		if errors.Is(err, helper.ErrKeycloakRealmNotFound) {
			if removeErr := r.helper.TryRemoveFinalizer(ctx, instance, finalizerName); removeErr != nil {
				return "", fmt.Errorf("unable to remove finalizer: %w", removeErr)
			}

			return "", nil
		}

		return "", fmt.Errorf("unable to create keycloak client from realm ref: %w", err)
	}

	realm, err := r.helper.GetKeycloakRealmFromRef(ctx, instance, cl)
	if err != nil {
		return "", fmt.Errorf("unable to get keycloak realm from ref: %w", err)
	}

	scopeID, err := syncClientScope(ctx, instance, gocloak.PString(realm.Realm), cl)
	if err != nil {
		return "", errors.Wrap(err, "unable to sync client scope")
	}

	if _, err = r.helper.TryToDelete(ctx, instance,
		makeTerminator(
			cl,
			gocloak.PString(realm.Realm),
			instance.Status.ID,
			objectmeta.PreserveResourcesOnDeletion(instance),
		),
		finalizerName,
	); err != nil {
		return "", fmt.Errorf("unable to delete client scope: %w", err)
	}

	return scopeID, nil
}

func (r *Reconcile) updateKeycloakClientScopeStatus(
	ctx context.Context,
	scope *keycloakApi.KeycloakClientScope,
	oldStatus keycloakApi.KeycloakClientScopeStatus,
) error {
	if scope.Status == oldStatus {
		return nil
	}

	if err := r.client.Status().Update(ctx, scope); err != nil {
		return fmt.Errorf("failed to update KeycloakClientScope status: %w", err)
	}

	return nil
}

func syncClientScope(ctx context.Context, instance *keycloakApi.KeycloakClientScope, realmName string, cl keycloak.Client) (string, error) {
	clientScope, err := cl.GetClientScope(instance.Spec.Name, realmName)
	if err != nil && !adapter.IsErrNotFound(err) {
		return "", errors.Wrap(err, "unable to get client scope")
	}

	cScope := adapter.ClientScope{
		Name:            instance.Spec.Name,
		Attributes:      instance.Spec.Attributes,
		Protocol:        instance.Spec.Protocol,
		ProtocolMappers: convertProtocolMappers(instance.Spec.ProtocolMappers),
		Description:     instance.Spec.Description,
		Default:         instance.Spec.Default,
	}

	if err == nil {
		if instance.Status.ID == "" {
			instance.Status.ID = clientScope.ID
		}

		if err = cl.UpdateClientScope(ctx, realmName, instance.Status.ID, &cScope); err != nil {
			return "", errors.Wrap(err, "unable to update client scope")
		}

		return instance.Status.ID, nil
	}

	id, err := cl.CreateClientScope(ctx, realmName, &cScope)
	if err != nil {
		return "", errors.Wrap(err, "unable to create client scope")
	}

	instance.Status.ID = id

	return instance.Status.ID, nil
}

func convertProtocolMappers(mappers []keycloakApi.ProtocolMapper) []adapter.ProtocolMapper {
	aMappers := make([]adapter.ProtocolMapper, 0, len(mappers))

	for _, m := range mappers {
		pm := adapter.ProtocolMapper{
			Name:           m.Name,
			Config:         make(map[string]string, len(m.Config)),
			ProtocolMapper: m.ProtocolMapper,
			Protocol:       m.Protocol,
		}

		maps.Copy(pm.Config, m.Config)

		aMappers = append(aMappers, pm)
	}

	return aMappers
}

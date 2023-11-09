package keycloakauthflow

import (
	"context"
	"fmt"
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

	"github.com/epam/edp-keycloak-operator/api/common"
	keycloakApi "github.com/epam/edp-keycloak-operator/api/v1"
	"github.com/epam/edp-keycloak-operator/controllers/helper"
	"github.com/epam/edp-keycloak-operator/pkg/client/keycloak"
	"github.com/epam/edp-keycloak-operator/pkg/client/keycloak/adapter"
	"github.com/epam/edp-keycloak-operator/pkg/objectmeta"
)

const finalizerName = "keycloak.authflow.operator.finalizer.name"

type Helper interface {
	SetFailureCount(fc helper.FailureCountable) time.Duration
	TryToDelete(ctx context.Context, obj client.Object, terminator helper.Terminator, finalizer string) (isDeleted bool, resultErr error)
	CreateKeycloakClientFromRealmRef(ctx context.Context, object helper.ObjectWithRealmRef) (keycloak.Client, error)
	SetRealmOwnerRef(ctx context.Context, object helper.ObjectWithRealmRef) error
	GetKeycloakRealmFromRef(ctx context.Context, object helper.ObjectWithRealmRef, kcClient keycloak.Client) (*gocloak.RealmRepresentation, error)
}

type Reconcile struct {
	client                  client.Client
	helper                  Helper
	successReconcileTimeout time.Duration
}

func NewReconcile(client client.Client, helper Helper) *Reconcile {
	return &Reconcile{
		client: client,
		helper: helper,
	}
}

func (r *Reconcile) SetupWithManager(mgr ctrl.Manager, successReconcileTimeout time.Duration) error {
	r.successReconcileTimeout = successReconcileTimeout

	pred := predicate.Funcs{
		UpdateFunc: isSpecUpdated,
	}

	err := ctrl.NewControllerManagedBy(mgr).
		For(&keycloakApi.KeycloakAuthFlow{}, builder.WithPredicates(pred)).
		Complete(r)
	if err != nil {
		return fmt.Errorf("failed to setup keycloakAuthFlow controller: %w", err)
	}

	return nil
}

func isSpecUpdated(e event.UpdateEvent) bool {
	oo, ok := e.ObjectOld.(*keycloakApi.KeycloakAuthFlow)
	if !ok {
		return false
	}

	no, ok := e.ObjectNew.(*keycloakApi.KeycloakAuthFlow)
	if !ok {
		return false
	}

	return !reflect.DeepEqual(oo.Spec, no.Spec) ||
		(oo.GetDeletionTimestamp().IsZero() && !no.GetDeletionTimestamp().IsZero())
}

//+kubebuilder:rbac:groups=v1.edp.epam.com,namespace=placeholder,resources=keycloakauthflows,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=v1.edp.epam.com,namespace=placeholder,resources=keycloakauthflows/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=v1.edp.epam.com,namespace=placeholder,resources=keycloakauthflows/finalizers,verbs=update

// Reconcile is a loop for reconciling KeycloakAuthFlow object.
func (r *Reconcile) Reconcile(ctx context.Context, request reconcile.Request) (result reconcile.Result, resultErr error) {
	log := ctrl.LoggerFrom(ctx)
	log.Info("Reconciling KeycloakAuthFlow")

	var instance keycloakApi.KeycloakAuthFlow
	if err := r.client.Get(ctx, request.NamespacedName, &instance); err != nil {
		if k8sErrors.IsNotFound(err) {
			return
		}

		resultErr = errors.Wrap(err, "unable to get keycloak auth flow from k8s")

		return
	}

	if updated, err := r.applyDefaults(ctx, &instance); err != nil {
		resultErr = fmt.Errorf("unable to apply default values: %w", err)
		return
	} else if updated {
		return
	}

	if err := r.tryReconcile(ctx, &instance); err != nil {
		if errors.Is(err, helper.ErrKeycloakIsNotAvailable) {
			return ctrl.Result{
				RequeueAfter: helper.RequeueOnKeycloakNotAvailablePeriod,
			}, nil
		}

		instance.Status.Value = err.Error()
		result.RequeueAfter = r.helper.SetFailureCount(&instance)

		log.Error(err, "an error has occurred while handling keycloak auth flow", "name", request.Name)
	} else {
		result.RequeueAfter = r.successReconcileTimeout
		helper.SetSuccessStatus(&instance)
	}

	if err := r.client.Status().Update(ctx, &instance); err != nil {
		resultErr = err
	}

	log.Info("Reconciling KeycloakAuthFlow done.")

	return
}

func (r *Reconcile) tryReconcile(ctx context.Context, instance *keycloakApi.KeycloakAuthFlow) error {
	if err := r.helper.SetRealmOwnerRef(ctx, instance); err != nil {
		return fmt.Errorf("unable to set realm owner ref: %w", err)
	}

	kClient, err := r.helper.CreateKeycloakClientFromRealmRef(ctx, instance)
	if err != nil {
		return fmt.Errorf("unable to create keycloak client from realm ref: %w", err)
	}

	realm, err := r.helper.GetKeycloakRealmFromRef(ctx, instance, kClient)
	if err != nil {
		return fmt.Errorf("unable to get realm from ref: %w", err)
	}

	keycloakAuthFlow := authFlowSpecToAdapterAuthFlow(&instance.Spec)

	deleted, err := r.helper.TryToDelete(
		ctx,
		instance,
		makeTerminator(
			gocloak.PString(realm.Realm),
			instance.GetRealmRef().Name,
			keycloakAuthFlow,
			r.client,
			kClient,
			objectmeta.PreserveResourcesOnDeletion(instance),
		),
		finalizerName,
	)
	if err != nil {
		return fmt.Errorf("unable to delete auth flow: %w", err)
	}

	if deleted {
		return nil
	}

	if err := kClient.SyncAuthFlow(gocloak.PString(realm.Realm), keycloakAuthFlow); err != nil {
		return fmt.Errorf("unable to sync auth flow: %w", err)
	}

	return nil
}

func (r *Reconcile) applyDefaults(ctx context.Context, instance *keycloakApi.KeycloakAuthFlow) (bool, error) {
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

func authFlowSpecToAdapterAuthFlow(spec *keycloakApi.KeycloakAuthFlowSpec) *adapter.KeycloakAuthFlow {
	flow := adapter.KeycloakAuthFlow{
		Alias:                    spec.Alias,
		Description:              spec.Description,
		BuiltIn:                  spec.BuiltIn,
		ProviderID:               spec.ProviderID,
		TopLevel:                 spec.TopLevel,
		AuthenticationExecutions: make([]adapter.AuthenticationExecution, 0, len(spec.AuthenticationExecutions)),
		ParentName:               spec.ParentName,
		ChildType:                spec.ChildType,
	}

	for _, ae := range spec.AuthenticationExecutions {
		exec := adapter.AuthenticationExecution{
			Authenticator:    ae.Authenticator,
			Requirement:      ae.Requirement,
			Priority:         ae.Priority,
			AutheticatorFlow: ae.AuthenticatorFlow,
			Alias:            ae.Alias,
		}

		if ae.AuthenticatorConfig != nil {
			exec.AuthenticatorConfig = &adapter.AuthenticatorConfig{
				Alias:  ae.AuthenticatorConfig.Alias,
				Config: ae.AuthenticatorConfig.Config,
			}
		}

		flow.AuthenticationExecutions = append(flow.AuthenticationExecutions, exec)
	}

	return &flow
}

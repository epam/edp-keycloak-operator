package keycloakauthflow

import (
	"context"
	"fmt"
	"reflect"
	"time"

	"github.com/go-logr/logr"
	"github.com/pkg/errors"
	k8sErrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	keycloakApi "github.com/epam/edp-keycloak-operator/api/v1"
	"github.com/epam/edp-keycloak-operator/controllers/helper"
	"github.com/epam/edp-keycloak-operator/pkg/client/keycloak"
	"github.com/epam/edp-keycloak-operator/pkg/client/keycloak/adapter"
)

const finalizerName = "keycloak.authflow.operator.finalizer.name"

type Helper interface {
	SetFailureCount(fc helper.FailureCountable) time.Duration
	UpdateStatus(obj client.Object) error
	TryToDelete(ctx context.Context, obj helper.Deletable, terminator helper.Terminator,
		finalizer string) (isDeleted bool, resultErr error)
	CreateKeycloakClientForRealm(ctx context.Context, realm *keycloakApi.KeycloakRealm) (keycloak.Client, error)
	GetOrCreateRealmOwnerRef(object helper.RealmChild, objectMeta *metav1.ObjectMeta) (*keycloakApi.KeycloakRealm, error)
}

type Reconcile struct {
	client                  client.Client
	helper                  Helper
	log                     logr.Logger
	successReconcileTimeout time.Duration
}

func NewReconcile(client client.Client, log logr.Logger, helper Helper) *Reconcile {
	return &Reconcile{
		client: client,
		helper: helper,
		log:    log.WithName("keycloak-auth-flow"),
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
func (r *Reconcile) Reconcile(ctx context.Context, request reconcile.Request) (result reconcile.Result,
	resultErr error) {
	log := r.log.WithValues("Request.Namespace", request.Namespace, "Request.Name", request.Name)
	log.Info("Reconciling KeycloakAuthFlow")

	var instance keycloakApi.KeycloakAuthFlow
	if err := r.client.Get(ctx, request.NamespacedName, &instance); err != nil {
		if k8sErrors.IsNotFound(err) {
			return
		}

		resultErr = errors.Wrap(err, "unable to get keycloak auth flow from k8s")

		return
	}

	if err := r.tryReconcile(ctx, &instance); err != nil {
		instance.Status.Value = err.Error()
		result.RequeueAfter = r.helper.SetFailureCount(&instance)

		log.Error(err, "an error has occurred while handling keycloak auth flow", "name", request.Name)
	} else {
		result.RequeueAfter = r.successReconcileTimeout
		helper.SetSuccessStatus(&instance)
	}

	if err := r.helper.UpdateStatus(&instance); err != nil {
		resultErr = err
	}

	log.Info("Reconciling KeycloakAuthFlow done.")

	return
}

func (r *Reconcile) tryReconcile(ctx context.Context, instance *keycloakApi.KeycloakAuthFlow) error {
	realm, err := r.helper.GetOrCreateRealmOwnerRef(instance, &instance.ObjectMeta)
	if err != nil {
		return errors.Wrap(err, "unable to get realm owner ref")
	}

	kClient, err := r.helper.CreateKeycloakClientForRealm(ctx, realm)
	if err != nil {
		return errors.Wrap(err, "unable to create keycloak client")
	}

	keycloakAuthFlow := authFlowSpecToAdapterAuthFlow(&instance.Spec)

	deleted, err := r.helper.TryToDelete(ctx, instance,
		makeTerminator(realm, keycloakAuthFlow, r.client, kClient,
			r.log.WithName("auth-flow-term")), finalizerName)
	if err != nil {
		return errors.Wrap(err, "unable to tryToDelete auth flow")
	}

	if deleted {
		return nil
	}

	if err := kClient.SyncAuthFlow(realm.Spec.RealmName, keycloakAuthFlow); err != nil {
		return errors.Wrap(err, "unable to sync auth flow")
	}

	return nil
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

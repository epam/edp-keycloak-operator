package keycloakclientscope

import (
	"context"
	"fmt"
	"reflect"
	"time"

	"github.com/go-logr/logr"
	"github.com/pkg/errors"
	k8sErrors "k8s.io/apimachinery/pkg/api/errors"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	keycloakApi "github.com/epam/edp-keycloak-operator/api/v1/v1"
	"github.com/epam/edp-keycloak-operator/controllers/helper"
	"github.com/epam/edp-keycloak-operator/pkg/client/keycloak"
	"github.com/epam/edp-keycloak-operator/pkg/client/keycloak/adapter"
)

const finalizerName = "keycloak.clientscope.operator.finalizer.name"

type Helper interface {
	SetFailureCount(fc helper.FailureCountable) time.Duration
	UpdateStatus(obj client.Object) error
	CreateKeycloakClientForRealm(ctx context.Context, realm *keycloakApi.KeycloakRealm) (keycloak.Client, error)
	GetOrCreateRealmOwnerRef(object helper.RealmChild, objectMeta *v1.ObjectMeta) (*keycloakApi.KeycloakRealm, error)
	TryToDelete(ctx context.Context, obj helper.Deletable, terminator helper.Terminator, finalizer string) (isDeleted bool, resultErr error)
}

type Reconcile struct {
	client                  client.Client
	log                     logr.Logger
	helper                  Helper
	successReconcileTimeout time.Duration
}

func NewReconcile(client client.Client, log logr.Logger, helper Helper) *Reconcile {
	return &Reconcile{
		client: client,
		helper: helper,
		log:    log.WithName("keycloak-client-scope"),
	}
}

func (r *Reconcile) SetupWithManager(mgr ctrl.Manager, successReconcileTimeout time.Duration) error {
	r.successReconcileTimeout = successReconcileTimeout

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
func (r *Reconcile) Reconcile(ctx context.Context, request reconcile.Request) (result reconcile.Result,
	resultErr error) {
	log := r.log.WithValues("Request.Namespace", request.Namespace, "Request.Name", request.Name)
	log.Info("Reconciling KeycloakClientScope")

	var instance keycloakApi.KeycloakClientScope
	if err := r.client.Get(ctx, request.NamespacedName, &instance); err != nil {
		if k8sErrors.IsNotFound(err) {
			log.Info("instance not found")
			return
		}

		resultErr = errors.Wrap(err, "unable to get keycloak realm user from k8s")

		return
	}

	scopeID, err := r.tryReconcile(ctx, &instance)
	if err != nil {
		instance.Status.Value = err.Error()
		result.RequeueAfter = r.helper.SetFailureCount(&instance)

		log.Error(err, "an error has occurred while handling keycloak client scope", "name", request.Name)
	} else {
		helper.SetSuccessStatus(&instance)
		instance.Status.ID = scopeID
		result.RequeueAfter = r.successReconcileTimeout
	}

	if err := r.helper.UpdateStatus(&instance); err != nil {
		resultErr = err
	}

	log.Info("Reconciling KeycloakClientScope done.")

	return
}

func (r *Reconcile) tryReconcile(ctx context.Context, instance *keycloakApi.KeycloakClientScope) (string, error) {
	realm, err := r.helper.GetOrCreateRealmOwnerRef(instance, &instance.ObjectMeta)
	if err != nil {
		return "", errors.Wrap(err, "unable to get realm owner ref")
	}

	cl, err := r.helper.CreateKeycloakClientForRealm(ctx, realm)
	if err != nil {
		return "", errors.Wrap(err, "unable to create keycloak client")
	}

	scopeID, err := syncClientScope(ctx, instance, realm, cl)
	if err != nil {
		return "", errors.Wrap(err, "unable to sync client scope")
	}

	if _, err := r.helper.TryToDelete(ctx, instance,
		makeTerminator(cl, realm.Spec.RealmName, instance.Status.ID, r.log.WithName("client-scope-term")),
		finalizerName); err != nil {
		return "", errors.Wrap(err, "error during TryToDelete")
	}

	return scopeID, nil
}

func syncClientScope(ctx context.Context, instance *keycloakApi.KeycloakClientScope, realm *keycloakApi.KeycloakRealm, cl keycloak.Client) (string, error) {
	clientScope, err := cl.GetClientScope(instance.Spec.Name, realm.Spec.RealmName)
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

		if err = cl.UpdateClientScope(ctx, realm.Spec.RealmName, instance.Status.ID, &cScope); err != nil {
			return "", errors.Wrap(err, "unable to update client scope")
		}

		return instance.Status.ID, nil
	}

	id, err := cl.CreateClientScope(ctx, realm.Spec.RealmName, &cScope)
	if err != nil {
		return "", errors.Wrap(err, "unable to create client scope")
	}

	instance.Status.ID = id

	return instance.Status.ID, nil
}

func convertProtocolMappers(mappers []keycloakApi.ProtocolMapper) []adapter.ProtocolMapper {
	aMappers := make([]adapter.ProtocolMapper, 0, len(mappers))
	for _, m := range mappers {
		aMappers = append(aMappers, adapter.ProtocolMapper{
			Name:           m.Name,
			Config:         m.Config,
			ProtocolMapper: m.ProtocolMapper,
			Protocol:       m.Protocol,
		})
	}

	return aMappers
}

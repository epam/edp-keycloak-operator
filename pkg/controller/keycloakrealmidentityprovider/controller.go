package keycloakrealmidentityprovider

import (
	"context"
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

	keycloakApi "github.com/epam/edp-keycloak-operator/pkg/apis/v1/v1"
	"github.com/epam/edp-keycloak-operator/pkg/client/keycloak"
	"github.com/epam/edp-keycloak-operator/pkg/client/keycloak/adapter"
	"github.com/epam/edp-keycloak-operator/pkg/controller/helper"
)

const finalizerName = "keycloak.realmidp.operator.finalizer.name"

type Helper interface {
	SetFailureCount(fc helper.FailureCountable) time.Duration
	UpdateStatus(obj client.Object) error
	GetOrCreateRealmOwnerRef(object helper.RealmChild, objectMeta *v1.ObjectMeta) (*keycloakApi.KeycloakRealm, error)
	CreateKeycloakClientForRealm(ctx context.Context, realm *keycloakApi.KeycloakRealm) (keycloak.Client, error)
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
		log:    log.WithName("keycloak-realm-identity-provider"),
	}
}

func (r *Reconcile) SetupWithManager(mgr ctrl.Manager, successReconcileTimeout time.Duration) error {
	r.successReconcileTimeout = successReconcileTimeout

	pred := predicate.Funcs{
		UpdateFunc: isSpecUpdated,
	}

	return ctrl.NewControllerManagedBy(mgr).
		For(&keycloakApi.KeycloakRealmIdentityProvider{}, builder.WithPredicates(pred)).
		Complete(r)
}

func isSpecUpdated(e event.UpdateEvent) bool {
	oo, ok := e.ObjectOld.(*keycloakApi.KeycloakRealmIdentityProvider)
	if !ok {
		return false
	}

	no, ok := e.ObjectNew.(*keycloakApi.KeycloakRealmIdentityProvider)
	if !ok {
		return false
	}

	return !reflect.DeepEqual(oo.Spec, no.Spec) ||
		(oo.GetDeletionTimestamp().IsZero() && !no.GetDeletionTimestamp().IsZero())
}

func (r *Reconcile) Reconcile(ctx context.Context, request reconcile.Request) (result reconcile.Result, resultErr error) {
	log := r.log.WithValues("Request.Namespace", request.Namespace, "Request.Name", request.Name)
	log.Info("Reconciling KeycloakRealmIdentityProvider")

	var instance keycloakApi.KeycloakRealmIdentityProvider
	if err := r.client.Get(ctx, request.NamespacedName, &instance); err != nil {
		if k8sErrors.IsNotFound(err) {
			log.Info("instance not found")

			return
		}

		resultErr = errors.Wrap(err, "unable to get keycloak realm idp from k8s")

		return
	}

	if err := r.tryReconcile(ctx, &instance); err != nil {
		instance.Status.Value = err.Error()
		result.RequeueAfter = r.helper.SetFailureCount(&instance)

		log.Error(err, "an error has occurred while handling keycloak realm idp", "name", request.Name)
	} else {
		helper.SetSuccessStatus(&instance)
		result.RequeueAfter = r.successReconcileTimeout
	}

	if err := r.helper.UpdateStatus(&instance); err != nil {
		resultErr = errors.Wrap(err, "unable to update status")
	}

	return
}

func (r *Reconcile) tryReconcile(ctx context.Context, keycloakRealmIDP *keycloakApi.KeycloakRealmIdentityProvider) error {
	realm, err := r.helper.GetOrCreateRealmOwnerRef(keycloakRealmIDP, &keycloakRealmIDP.ObjectMeta)
	if err != nil {
		return errors.Wrap(err, "unable to get realm owner ref")
	}

	kClient, err := r.helper.CreateKeycloakClientForRealm(ctx, realm)
	if err != nil {
		return errors.Wrap(err, "unable to create keycloak client")
	}

	keycloakIDP := createKeycloakIDPFromSpec(&keycloakRealmIDP.Spec)

	providerExists, err := kClient.IdentityProviderExists(ctx, realm.Spec.RealmName, keycloakRealmIDP.Spec.Alias)
	if err != nil {
		return err
	}

	if providerExists {
		if err = kClient.UpdateIdentityProvider(ctx, realm.Spec.RealmName, keycloakIDP); err != nil {
			return errors.Wrap(err, "unable to update idp")
		}
	} else {
		if err = kClient.CreateIdentityProvider(ctx, realm.Spec.RealmName, keycloakIDP); err != nil {
			return errors.Wrap(err, "unable to create idp")
		}
	}

	if err := syncIDPMappers(ctx, &keycloakRealmIDP.Spec, kClient, realm.Spec.RealmName); err != nil {
		return errors.Wrap(err, "unable to sync idp mappers")
	}

	term := makeTerminator(realm.Spec.RealmName, keycloakRealmIDP.Spec.Alias, kClient, r.log.WithName("realm-idp-term"))
	if _, err := r.helper.TryToDelete(ctx, keycloakRealmIDP, term, finalizerName); err != nil {
		return errors.Wrap(err, "unable to delete realm idp")
	}

	return nil
}

func syncIDPMappers(ctx context.Context, idpSpec *keycloakApi.KeycloakRealmIdentityProviderSpec,
	kClient keycloak.Client, targetRealm string) error {
	if len(idpSpec.Mappers) == 0 {
		return nil
	}

	mappers, err := kClient.GetIDPMappers(ctx, targetRealm, idpSpec.Alias)
	if err != nil {
		return errors.Wrap(err, "unable to get idp mappers")
	}

	for _, m := range mappers {
		if err := kClient.DeleteIDPMapper(ctx, targetRealm, idpSpec.Alias, m.ID); err != nil {
			return errors.Wrap(err, "unable to delete idp mapper")
		}
	}

	for _, m := range idpSpec.Mappers {
		if m.IdentityProviderAlias == "" {
			m.IdentityProviderAlias = idpSpec.Alias
		}

		if _, err := kClient.CreateIDPMapper(ctx, targetRealm, idpSpec.Alias,
			createKeycloakIDPMapperFromSpec(&m)); err != nil {
			return errors.Wrap(err, "unable to create idp mapper")
		}
	}

	return nil
}

func createKeycloakIDPMapperFromSpec(spec *keycloakApi.IdentityProviderMapper) *adapter.IdentityProviderMapper {
	return &adapter.IdentityProviderMapper{
		IdentityProviderMapper: spec.IdentityProviderMapper,
		Name:                   spec.Name,
		Config:                 spec.Config,
		IdentityProviderAlias:  spec.IdentityProviderAlias,
	}
}

func createKeycloakIDPFromSpec(spec *keycloakApi.KeycloakRealmIdentityProviderSpec) *adapter.IdentityProvider {
	return &adapter.IdentityProvider{
		Config:                    spec.Config,
		ProviderID:                spec.ProviderID,
		Alias:                     spec.Alias,
		Enabled:                   spec.Enabled,
		AddReadTokenRoleOnCreate:  spec.AddReadTokenRoleOnCreate,
		AuthenticateByDefault:     spec.AuthenticateByDefault,
		DisplayName:               spec.DisplayName,
		FirstBrokerLoginFlowAlias: spec.FirstBrokerLoginFlowAlias,
		LinkOnly:                  spec.LinkOnly,
		StoreToken:                spec.StoreToken,
		TrustEmail:                spec.TrustEmail,
	}
}

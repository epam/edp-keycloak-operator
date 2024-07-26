package keycloakrealmidentityprovider

import (
	"context"
	"fmt"
	"reflect"
	"time"

	"github.com/Nerzal/gocloak/v12"
	"github.com/pkg/errors"
	"golang.org/x/exp/maps"
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

const finalizerName = "keycloak.realmidp.operator.finalizer.name"

type Helper interface {
	SetFailureCount(fc helper.FailureCountable) time.Duration
	TryToDelete(ctx context.Context, obj client.Object, terminator helper.Terminator, finalizer string) (isDeleted bool, resultErr error)
	GetKeycloakRealmFromRef(ctx context.Context, object helper.ObjectWithRealmRef, kcClient keycloak.Client) (*gocloak.RealmRepresentation, error)
	CreateKeycloakClientFromRealmRef(ctx context.Context, object helper.ObjectWithRealmRef) (keycloak.Client, error)
}

type RefClient interface {
	MapConfigSecretsRefs(ctx context.Context, config map[string]string, namespace string) error
}

type Reconcile struct {
	client                  client.Client
	helper                  Helper
	secretRefClient         RefClient
	successReconcileTimeout time.Duration
}

func NewReconcile(client client.Client, helper Helper, secretRefClient RefClient) *Reconcile {
	return &Reconcile{
		client:          client,
		helper:          helper,
		secretRefClient: secretRefClient,
	}
}

func (r *Reconcile) SetupWithManager(mgr ctrl.Manager, successReconcileTimeout time.Duration) error {
	r.successReconcileTimeout = successReconcileTimeout

	pred := predicate.Funcs{
		UpdateFunc: isSpecUpdated,
	}

	err := ctrl.NewControllerManagedBy(mgr).
		For(&keycloakApi.KeycloakRealmIdentityProvider{}, builder.WithPredicates(pred)).
		Complete(r)
	if err != nil {
		return fmt.Errorf("failed to setup KeycloakRealmIdentityProvider controller: %w", err)
	}

	return nil
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

//+kubebuilder:rbac:groups=v1.edp.epam.com,namespace=placeholder,resources=keycloakrealmidentityproviders,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=v1.edp.epam.com,namespace=placeholder,resources=keycloakrealmidentityproviders/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=v1.edp.epam.com,namespace=placeholder,resources=keycloakrealmidentityproviders/finalizers,verbs=update

// Reconcile is a loop for reconciling KeycloakRealmIdentityProvider object.
func (r *Reconcile) Reconcile(ctx context.Context, request reconcile.Request) (result reconcile.Result, resultErr error) {
	log := ctrl.LoggerFrom(ctx)
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

		log.Error(err, "an error has occurred while handling keycloak realm idp", "name", request.Name)
	} else {
		helper.SetSuccessStatus(&instance)
		result.RequeueAfter = r.successReconcileTimeout
	}

	if err := r.client.Status().Update(ctx, &instance); err != nil {
		resultErr = errors.Wrap(err, "unable to update status")
	}

	return
}

func (r *Reconcile) tryReconcile(ctx context.Context, keycloakRealmIDP *keycloakApi.KeycloakRealmIdentityProvider) error {
	kClient, err := r.helper.CreateKeycloakClientFromRealmRef(ctx, keycloakRealmIDP)
	if err != nil {
		return fmt.Errorf("unable to create keycloak client from realm ref: %w", err)
	}

	realm, err := r.helper.GetKeycloakRealmFromRef(ctx, keycloakRealmIDP, kClient)
	if err != nil {
		return fmt.Errorf("unable to get keycloak realm from ref: %w", err)
	}

	keycloakIDP := createKeycloakIDPFromSpec(&keycloakRealmIDP.Spec)

	if err = r.secretRefClient.MapConfigSecretsRefs(ctx, keycloakIDP.Config, keycloakRealmIDP.Namespace); err != nil {
		return fmt.Errorf("unable to map config secrets: %w", err)
	}

	providerExists, err := kClient.IdentityProviderExists(ctx, gocloak.PString(realm.Realm), keycloakRealmIDP.Spec.Alias)
	if err != nil {
		return fmt.Errorf("failed to check if the identity provider exists: %w", err)
	}

	if providerExists {
		if err = kClient.UpdateIdentityProvider(ctx, gocloak.PString(realm.Realm), keycloakIDP); err != nil {
			return errors.Wrap(err, "unable to update idp")
		}
	} else {
		if err = kClient.CreateIdentityProvider(ctx, gocloak.PString(realm.Realm), keycloakIDP); err != nil {
			return errors.Wrap(err, "unable to create idp")
		}
	}

	if err := syncIDPMappers(ctx, &keycloakRealmIDP.Spec, kClient, gocloak.PString(realm.Realm)); err != nil {
		return errors.Wrap(err, "unable to sync idp mappers")
	}

	term := makeTerminator(
		gocloak.PString(realm.Realm),
		keycloakRealmIDP.Spec.Alias,
		kClient,
		objectmeta.PreserveResourcesOnDeletion(keycloakRealmIDP),
	)
	if _, err := r.helper.TryToDelete(ctx, keycloakRealmIDP, term, finalizerName); err != nil {
		return errors.Wrap(err, "unable to delete realm idp")
	}

	return nil
}

func (r *Reconcile) applyDefaults(ctx context.Context, instance *keycloakApi.KeycloakRealmIdentityProvider) (bool, error) {
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
	m := &adapter.IdentityProviderMapper{
		IdentityProviderMapper: spec.IdentityProviderMapper,
		Name:                   spec.Name,
		Config:                 make(map[string]string, len(spec.Config)),
		IdentityProviderAlias:  spec.IdentityProviderAlias,
	}

	maps.Copy(m.Config, spec.Config)

	return m
}

func createKeycloakIDPFromSpec(spec *keycloakApi.KeycloakRealmIdentityProviderSpec) *adapter.IdentityProvider {
	p := &adapter.IdentityProvider{
		Config:                    make(map[string]string, len(spec.Config)),
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

	maps.Copy(p.Config, spec.Config)

	return p
}

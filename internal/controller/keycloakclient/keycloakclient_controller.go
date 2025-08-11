package keycloakclient

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/Nerzal/gocloak/v12"
	pkgErrors "github.com/pkg/errors"
	k8sErrors "k8s.io/apimachinery/pkg/api/errors"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	keycloakApi "github.com/epam/edp-keycloak-operator/api/v1"
	"github.com/epam/edp-keycloak-operator/internal/controller/helper"
	"github.com/epam/edp-keycloak-operator/internal/controller/keycloakclient/chain"
	"github.com/epam/edp-keycloak-operator/pkg/client/keycloak"
	"github.com/epam/edp-keycloak-operator/pkg/objectmeta"
)

type Helper interface {
	SetFailureCount(fc helper.FailureCountable) time.Duration
	TryRemoveFinalizer(ctx context.Context, obj client.Object, finalizer string) error
	TryToDelete(ctx context.Context, obj client.Object, terminator helper.Terminator, finalizer string) (isDeleted bool, resultErr error)
	SetRealmOwnerRef(ctx context.Context, object helper.ObjectWithRealmRef) error
	CreateKeycloakClientFromRealmRef(ctx context.Context, object helper.ObjectWithRealmRef) (keycloak.Client, error)
	GetKeycloakRealmFromRef(ctx context.Context, object helper.ObjectWithRealmRef, kcClient keycloak.Client) (*gocloak.RealmRepresentation, error)
}

const (
	keyCloakClientOperatorFinalizerName       = "keycloak.client.operator.finalizer.name"
	clientAttributeLogoutRedirectUris         = "post.logout.redirect.uris"
	clientAttributeLogoutRedirectUrisDefValue = "+"
)

func NewReconcileKeycloakClient(client client.Client, helper Helper) *ReconcileKeycloakClient {
	return &ReconcileKeycloakClient{
		client: client,
		helper: helper,
	}
}

// ReconcileKeycloakClient reconciles a KeycloakClient object.
type ReconcileKeycloakClient struct {
	client                  client.Client
	helper                  Helper
	successReconcileTimeout time.Duration
}

func (r *ReconcileKeycloakClient) SetupWithManager(mgr ctrl.Manager, successReconcileTimeout time.Duration) error {
	r.successReconcileTimeout = successReconcileTimeout

	pred := predicate.Funcs{
		UpdateFunc: helper.IsFailuresUpdated,
	}

	err := ctrl.NewControllerManagedBy(mgr).
		For(&keycloakApi.KeycloakClient{}, builder.WithPredicates(pred)).
		Complete(r)
	if err != nil {
		return fmt.Errorf("failed to setup KeycloakClient controller: %w", err)
	}

	return nil
}

//+kubebuilder:rbac:groups=v1.edp.epam.com,namespace=placeholder,resources=keycloakclients,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=v1.edp.epam.com,namespace=placeholder,resources=keycloakclients/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=v1.edp.epam.com,namespace=placeholder,resources=keycloakclients/finalizers,verbs=update

// Reconcile is a loop for reconciling KeycloakClient object.
func (r *ReconcileKeycloakClient) Reconcile(ctx context.Context, request reconcile.Request) (result reconcile.Result, resultErr error) {
	log := ctrl.LoggerFrom(ctx)
	log.Info("Reconciling KeycloakClient")

	var instance keycloakApi.KeycloakClient
	if err := r.client.Get(ctx, request.NamespacedName, &instance); err != nil {
		if k8sErrors.IsNotFound(err) {
			return
		}

		resultErr = err

		return
	}

	if updated, err := r.applyDefaults(ctx, &instance); err != nil {
		return reconcile.Result{}, err
	} else if updated {
		return reconcile.Result{}, nil
	}

	if err := r.tryReconcile(ctx, &instance); err != nil {
		if errors.Is(err, helper.ErrKeycloakIsNotAvailable) {
			return ctrl.Result{
				RequeueAfter: helper.RequeueOnKeycloakNotAvailablePeriod,
			}, nil
		}

		instance.Status.Value = err.Error()
		result.RequeueAfter = r.helper.SetFailureCount(&instance)

		log.Error(err, "an error has occurred while handling keycloak client", "name", request.Name)
	} else {
		helper.SetSuccessStatus(&instance)

		result.RequeueAfter = r.successReconcileTimeout
	}

	if err := r.client.Status().Update(ctx, &instance); err != nil {
		resultErr = pkgErrors.Wrap(err, "unable to update status")
	}

	return
}

func (r *ReconcileKeycloakClient) tryReconcile(ctx context.Context, keycloakClient *keycloakApi.KeycloakClient) error {
	err := r.helper.SetRealmOwnerRef(ctx, keycloakClient)
	if err != nil {
		return fmt.Errorf("unable to set realm owner ref: %w", err)
	}

	kClient, err := r.helper.CreateKeycloakClientFromRealmRef(ctx, keycloakClient)
	if err != nil {
		// if the realm is already deleted try to delete finalizer
		if errors.Is(err, helper.ErrKeycloakRealmNotFound) {
			if removeErr := r.helper.TryRemoveFinalizer(ctx, keycloakClient, keyCloakClientOperatorFinalizerName); removeErr != nil {
				return fmt.Errorf("unable to remove finalizer: %w", removeErr)
			}

			return nil
		}

		return fmt.Errorf("unable to create keycloak client from realm ref: %w", err)
	}

	realm, err := r.getKeycloakRealm(ctx, keycloakClient, kClient)
	if err != nil {
		return fmt.Errorf("unable to get keycloak realm: %w", err)
	}

	deleted, err := r.helper.TryToDelete(
		ctx,
		keycloakClient,
		makeTerminator(keycloakClient.Status.ClientID, realm, kClient, objectmeta.PreserveResourcesOnDeletion(keycloakClient)),
		keyCloakClientOperatorFinalizerName,
	)
	if err != nil {
		return fmt.Errorf("deleting keycloak client: %w", err)
	}

	if deleted {
		return nil
	}

	if err = chain.MakeChain(kClient, r.client).Serve(ctx, keycloakClient, realm); err != nil {
		return fmt.Errorf("unable to serve keycloak client: %w", err)
	}

	return nil
}

// applyDefaults applies default values to KeycloakClient.
func (r *ReconcileKeycloakClient) applyDefaults(ctx context.Context, keycloakClient *keycloakApi.KeycloakClient) (bool, error) {
	if keycloakClient.Spec.Attributes == nil {
		keycloakClient.Spec.Attributes = make(map[string]string)
	}

	updated := false

	if _, ok := keycloakClient.Spec.Attributes[clientAttributeLogoutRedirectUris]; !ok {
		// set default value for logout redirect uris to "+" is required for correct logout from keycloak
		keycloakClient.Spec.Attributes[clientAttributeLogoutRedirectUris] = clientAttributeLogoutRedirectUrisDefValue
		updated = true
	}

	if keycloakClient.Spec.WebOrigins == nil {
		keycloakClient.Spec.WebOrigins = []string{
			keycloakClient.Spec.WebUrl,
		}

		updated = true
	}

	// Migrate ClientRoles to ClientRolesV2 if needed
	if migrated := r.migrateClientRoles(keycloakClient); migrated {
		updated = true
	}

	if updated {
		if err := r.client.Update(ctx, keycloakClient); err != nil {
			return false, fmt.Errorf("failed to update keycloak client default values: %w", err)
		}

		return true, nil
	}

	return false, nil
}

// migrateClientRoles migrates ClientRoles to ClientRolesV2 format.
// This function converts the old string-based client roles to the new ClientRole struct format.
// It only performs migration if ClientRolesV2 is empty and ClientRoles is not empty.
func (r *ReconcileKeycloakClient) migrateClientRoles(keycloakClient *keycloakApi.KeycloakClient) bool {
	// Only migrate if ClientRolesV2 is empty and ClientRoles is not empty
	if len(keycloakClient.Spec.ClientRolesV2) == 0 && len(keycloakClient.Spec.ClientRoles) > 0 {
		// Convert string-based roles to ClientRole structs
		for _, roleName := range keycloakClient.Spec.ClientRoles {
			clientRole := keycloakApi.ClientRole{
				Name: roleName,
				// Composite field is left empty as it wasn't available in the old format
			}
			keycloakClient.Spec.ClientRolesV2 = append(keycloakClient.Spec.ClientRolesV2, clientRole)
		}

		// Keep the original ClientRoles field for backward compatibility
		// keycloakClient.Spec.ClientRoles remains unchanged

		return true
	}

	return false
}

func (r *ReconcileKeycloakClient) getKeycloakRealm(
	ctx context.Context,
	keycloakClient *keycloakApi.KeycloakClient,
	adapterClient keycloak.Client,
) (string, error) {
	realm, err := r.helper.GetKeycloakRealmFromRef(ctx, keycloakClient, adapterClient)
	if err != nil {
		return "", fmt.Errorf("unable to get keycloak realm from ref: %w", err)
	}

	return gocloak.PString(realm.Realm), nil
}

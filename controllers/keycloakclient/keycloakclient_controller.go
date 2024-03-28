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

	"github.com/epam/edp-keycloak-operator/api/common"
	keycloakApi "github.com/epam/edp-keycloak-operator/api/v1"
	"github.com/epam/edp-keycloak-operator/controllers/helper"
	"github.com/epam/edp-keycloak-operator/controllers/keycloakclient/chain"
	"github.com/epam/edp-keycloak-operator/pkg/client/keycloak"
	"github.com/epam/edp-keycloak-operator/pkg/objectmeta"
)

type Helper interface {
	SetFailureCount(fc helper.FailureCountable) time.Duration
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
		return fmt.Errorf("unable to create keycloak client from realm ref: %w", err)
	}

	realm, err := r.getKeycloakRealm(ctx, keycloakClient, kClient)
	if err != nil {
		return fmt.Errorf("unable to get keycloak realm: %w", err)
	}

	if err := chain.MakeChain(kClient, r.client).Serve(ctx, keycloakClient, realm); err != nil {
		return fmt.Errorf("unable to serve keycloak client: %w", err)
	}

	if _, err := r.helper.TryToDelete(
		ctx,
		keycloakClient,
		makeTerminator(keycloakClient.Status.ClientID, realm, kClient, objectmeta.PreserveResourcesOnDeletion(keycloakClient)),
		keyCloakClientOperatorFinalizerName,
	); err != nil {
		return pkgErrors.Wrap(err, "unable to delete kc client")
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

	if keycloakClient.Spec.RealmRef.Name == "" {
		realmName, err := r.getKeycloakCRName(ctx, keycloakClient.Spec.TargetRealm, keycloakClient.Namespace)
		if err != nil {
			return false, fmt.Errorf("unable to get keycloak cr name: %w", err)
		}

		keycloakClient.Spec.RealmRef = common.RealmRef{
			Kind: keycloakApi.KeycloakRealmKind,
			Name: realmName,
		}
		updated = true
	}

	if keycloakClient.Spec.WebOrigins == nil {
		keycloakClient.Spec.WebOrigins = []string{
			keycloakClient.Spec.WebUrl,
		}

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

func (r *ReconcileKeycloakClient) getKeycloakCRName(ctx context.Context, targetRealm, namespace string) (string, error) {
	realmList := &keycloakApi.KeycloakRealmList{}

	if err := r.client.List(ctx, realmList, client.InNamespace(namespace)); err != nil {
		return "", fmt.Errorf("unable to get realms: %w", err)
	}

	for i := 0; i < len(realmList.Items); i++ {
		if realmList.Items[i].Spec.RealmName == targetRealm {
			return realmList.Items[i].Name, nil
		}
	}

	// Add this for backward compatibility because in old versions KeycloakRealm CR name was hardcoded to "main".
	// We can remove this in the future release as RealmRef will be set for all KeycloakClient CRs.
	for i := 0; i < len(realmList.Items); i++ {
		if realmList.Items[i].Name == "main" {
			return realmList.Items[i].Name, nil
		}
	}

	return "", fmt.Errorf("realm %s not found", targetRealm)
}

func (r *ReconcileKeycloakClient) getKeycloakRealm(
	ctx context.Context,
	keycloakClient *keycloakApi.KeycloakClient,
	adapterClient keycloak.Client,
) (string, error) {
	if keycloakClient.Spec.TargetRealm == "" {
		realm, err := r.helper.GetKeycloakRealmFromRef(ctx, keycloakClient, adapterClient)
		if err != nil {
			return "", fmt.Errorf("unable to get keycloak realm from ref: %w", err)
		}

		return gocloak.PString(realm.Realm), nil
	}

	// If TargetRealm is set, use it instead of RealmRef. This is for backward compatibility.
	return keycloakClient.Spec.TargetRealm, nil
}

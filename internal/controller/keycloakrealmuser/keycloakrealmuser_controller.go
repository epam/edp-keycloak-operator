package keycloakrealmuser

import (
	"context"
	"fmt"
	"time"

	"github.com/Nerzal/gocloak/v12"
	"github.com/pkg/errors"
	coreV1 "k8s.io/api/core/v1"
	k8sErrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	"github.com/epam/edp-keycloak-operator/api/common"
	keycloakApi "github.com/epam/edp-keycloak-operator/api/v1"
	"github.com/epam/edp-keycloak-operator/internal/controller/helper"
	"github.com/epam/edp-keycloak-operator/pkg/client/keycloak"
	"github.com/epam/edp-keycloak-operator/pkg/client/keycloak/adapter"
	"github.com/epam/edp-keycloak-operator/pkg/objectmeta"
)

const finalizerName = "keycloak.realmuser.operator.finalizer.name"

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
		DeleteFunc: func(deleteEvent event.DeleteEvent) bool {
			return false
		},
	}

	err := ctrl.NewControllerManagedBy(mgr).
		For(&keycloakApi.KeycloakRealmUser{}, builder.WithPredicates(pred)).
		Complete(r)
	if err != nil {
		return fmt.Errorf("failed to setup KeycloakRealmUser controller: %w", err)
	}

	return nil
}

//+kubebuilder:rbac:groups=v1.edp.epam.com,namespace=placeholder,resources=keycloakrealmusers,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=v1.edp.epam.com,namespace=placeholder,resources=keycloakrealmusers/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=v1.edp.epam.com,namespace=placeholder,resources=keycloakrealmusers/finalizers,verbs=update

// Reconcile is a loop for reconciling KeycloakRealmUser object.
func (r *Reconcile) Reconcile(ctx context.Context, request reconcile.Request) (ctrl.Result, error) {
	log := ctrl.LoggerFrom(ctx)
	log.Info("Reconciling KeycloakRealmUser")

	var instance keycloakApi.KeycloakRealmUser
	if err := r.client.Get(ctx, request.NamespacedName, &instance); err != nil {
		if k8sErrors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}

		return ctrl.Result{}, errors.Wrap(err, "unable to get keycloak realm user from k8s")
	}

	oldStatus := instance.Status

	if updated, err := r.applyDefaults(ctx, &instance); err != nil {
		return reconcile.Result{}, err
	} else if updated {
		return reconcile.Result{}, nil
	}

	if err := r.tryReconcile(ctx, &instance); err != nil {
		log.Error(err, "An error has occurred while handling KeycloakRealmUser")

		if errors.Is(err, helper.ErrKeycloakIsNotAvailable) {
			return helper.RequeueOnKeycloakNotAvailable, nil
		}

		instance.Status.Value = err.Error()

		if statusErr := r.updateKeycloakRealmUserStatus(ctx, &instance, oldStatus); statusErr != nil {
			return ctrl.Result{}, statusErr
		}

		return ctrl.Result{}, err
	}

	instance.Status.Value = common.StatusOK
	if statusErr := r.updateKeycloakRealmUserStatus(ctx, &instance, oldStatus); statusErr != nil {
		return ctrl.Result{}, statusErr
	}

	log.Info("Reconciling KeycloakRealmUser done")

	return ctrl.Result{}, nil
}

func (r *Reconcile) applyDefaults(ctx context.Context, instance *keycloakApi.KeycloakRealmUser) (bool, error) {
	updated := false

	if migrate := r.migrateAttributes(instance); migrate {
		updated = true
	}

	if updated {
		if err := r.client.Update(ctx, instance); err != nil {
			return false, fmt.Errorf("failed to update keycloak client default values: %w", err)
		}

		return true, nil
	}

	return updated, nil
}

func convertClientRoles(apiClientRoles []keycloakApi.UserClientRole) map[string][]string {
	if apiClientRoles == nil {
		return nil
	}

	clientRolesMap := make(map[string][]string)
	for _, apiRole := range apiClientRoles {
		clientRolesMap[apiRole.ClientID] = apiRole.Roles
	}

	return clientRolesMap
}

func (r *Reconcile) tryReconcile(ctx context.Context, instance *keycloakApi.KeycloakRealmUser) error {
	err := r.helper.SetRealmOwnerRef(ctx, instance)
	if err != nil {
		return fmt.Errorf("unable to set realm owner ref: %w", err)
	}

	kClient, err := r.helper.CreateKeycloakClientFromRealmRef(ctx, instance)
	if err != nil {
		// if the realm is already deleted try to delete finalizer
		if errors.Is(err, helper.ErrKeycloakRealmNotFound) {
			if removeErr := r.helper.TryRemoveFinalizer(ctx, instance, finalizerName); removeErr != nil {
				return fmt.Errorf("unable to remove finalizer: %w", removeErr)
			}

			return nil
		}

		return fmt.Errorf("unable to create keycloak client from ref: %w", err)
	}

	realm, err := r.helper.GetKeycloakRealmFromRef(ctx, instance, kClient)
	if err != nil {
		return fmt.Errorf("unable to get keycloak realm from ref: %w", err)
	}

	if instance.Spec.KeepResource {
		deleted, err := r.helper.TryToDelete(ctx, instance,
			makeTerminator(
				gocloak.PString(realm.Realm),
				instance.Spec.Username,
				kClient,
				objectmeta.PreserveResourcesOnDeletion(instance),
			),
			finalizerName,
		)
		if err != nil {
			return fmt.Errorf("failed to delete keycloak realm user: %w", err)
		}

		if deleted {
			return nil
		}
	}

	password, getPasswordErr := r.getPassword(ctx, instance)
	if getPasswordErr != nil {
		return fmt.Errorf("unable to get password: %w", getPasswordErr)
	}

	userSpec := instance.Spec.DeepCopy()

	if err := kClient.SyncRealmUser(ctx, gocloak.PString(realm.Realm), &adapter.KeycloakUser{
		Username:            userSpec.Username,
		Groups:              userSpec.Groups,
		Roles:               userSpec.Roles,
		ClientRoles:         convertClientRoles(userSpec.ClientRoles),
		RequiredUserActions: userSpec.RequiredUserActions,
		LastName:            userSpec.LastName,
		FirstName:           userSpec.FirstName,
		EmailVerified:       userSpec.EmailVerified,
		Enabled:             userSpec.Enabled,
		Email:               userSpec.Email,
		Attributes:          userSpec.AttributesV2,
		Password:            password,
		IdentityProviders:   userSpec.IdentityProviders,
	}, instance.GetReconciliationStrategy() == keycloakApi.ReconciliationStrategyAddOnly); err != nil {
		return errors.Wrap(err, "unable to sync realm user")
	}

	if !instance.Spec.KeepResource {
		if err := r.client.Delete(ctx, instance); err != nil {
			return errors.Wrap(err, "unable to delete instance of keycloak realm user")
		}
	}

	return nil
}

func (r *Reconcile) getPassword(ctx context.Context, instance *keycloakApi.KeycloakRealmUser) (string, error) {
	log := ctrl.LoggerFrom(ctx)

	if instance.Spec.PasswordSecret.Name != "" && instance.Spec.PasswordSecret.Key != "" {
		secret := &coreV1.Secret{}
		if err := r.client.Get(ctx, types.NamespacedName{Name: instance.Spec.PasswordSecret.Name, Namespace: instance.Namespace}, secret); err != nil {
			if k8sErrors.IsNotFound(err) {
				return "", errors.Wrapf(err, "secret %s not found", instance.Spec.PasswordSecret.Name)
			}

			return "", errors.Wrapf(err, "unable to get secret %s", instance.Spec.PasswordSecret.Name)
		}

		passwordBytes, ok := secret.Data[instance.Spec.PasswordSecret.Key]
		if !ok {
			return "", errors.Errorf("key %s not found in secret %s", instance.Spec.PasswordSecret.Key, instance.Spec.PasswordSecret.Name)
		}

		log.Info("Using password from secret", "secret", instance.Spec.PasswordSecret.Name)

		return string(passwordBytes), nil
	}

	log.Info("Using password from instance Spec.password")

	return instance.Spec.Password, nil
}

func (r *Reconcile) updateKeycloakRealmUserStatus(
	ctx context.Context,
	user *keycloakApi.KeycloakRealmUser,
	oldStatus keycloakApi.KeycloakRealmUserStatus,
) error {
	if user.Status == oldStatus {
		return nil
	}

	if err := r.client.Status().Update(ctx, user); err != nil {
		return fmt.Errorf("failed to update KeycloakRealmUser status: %w", err)
	}

	return nil
}

// migrateAttributes migrates Attributes to AttributesV2 format.
// This function converts the old string-based attributes to the new []string format.
// It only performs migration if AttributesV2 is empty and Attributes is not empty.
func (r *Reconcile) migrateAttributes(keycloakRealmUser *keycloakApi.KeycloakRealmUser) bool {
	// Only migrate if AttributesV2 is empty and Attributes is not empty
	if len(keycloakRealmUser.Spec.AttributesV2) == 0 && len(keycloakRealmUser.Spec.Attributes) > 0 {
		keycloakRealmUser.Spec.AttributesV2 = make(map[string][]string, len(keycloakRealmUser.Spec.Attributes))

		// Convert string bases attributes to []string
		for attr, value := range keycloakRealmUser.Spec.Attributes {
			keycloakRealmUser.Spec.AttributesV2[attr] = []string{value}
		}

		// Keep the original Attributes field for backward compatibility
		// keycloakRealmUser.Spec.Attributes remains unchanged

		return true
	}

	return false
}

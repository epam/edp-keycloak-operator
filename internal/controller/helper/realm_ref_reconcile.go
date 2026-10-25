package helper

import (
	"context"
	"errors"
	"fmt"
	"slices"

	k8sErrors "k8s.io/apimachinery/pkg/api/errors"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	"github.com/epam/edp-keycloak-operator/pkg/client/keycloakapi"
)

// RealmRefObject is a realm-scoped CR whose status.value carries the last error.
type RealmRefObject interface {
	ObjectWithRealmRef
	StatusValue
}

// RealmRefHelper is the subset of Helper that ReconcileRealmRef calls.
type RealmRefHelper interface {
	SetRealmOwnerRef(ctx context.Context, object ObjectWithRealmRef) error
	CreateKeycloakClientFromRealmRef(ctx context.Context, object ObjectWithRealmRef) (*keycloakapi.KeycloakClient, error)
	GetRealmNameFromRef(ctx context.Context, object ObjectWithRealmRef) (string, error)
}

// RealmRefReconcile configures ReconcileRealmRef for one kind.
type RealmRefReconcile[T RealmRefObject] struct {
	// Kind is the type name used in log, Get error and status error texts.
	Kind   string
	Helper RealmRefHelper
	// Finalizers are checked and removed on deletion, and removed when the realm is gone.
	Finalizers []string
	// Remove deletes the object from Keycloak and wraps its own errors.
	Remove func(ctx context.Context, obj T, kClient *keycloakapi.KeycloakClient, realmName string) error
	// HandleReconciliation runs for objects that are not being deleted.
	HandleReconciliation func(ctx context.Context, obj T, kClient *keycloakapi.KeycloakClient, realmName string) (reconcile.Result, error)
}

// ReconcileRealmRef fetches obj, builds the Keycloak client and realm name from its realmRef,
// handles deletion, and passes live objects to cfg.HandleReconciliation.
// A CR Get error is returned without a status write.
// Other errors before cfg.HandleReconciliation are written to status.value and returned.
// ErrKeycloakIsNotAvailable requeues after RequeueOnKeycloakNotAvailablePeriod without a status write.
func ReconcileRealmRef[T RealmRefObject](
	ctx context.Context,
	k8sClient client.Client,
	request reconcile.Request,
	obj T,
	cfg RealmRefReconcile[T],
) (reconcile.Result, error) {
	ctrl.LoggerFrom(ctx).Info("Reconciling " + cfg.Kind)

	if err := k8sClient.Get(ctx, request.NamespacedName, obj); err != nil {
		if k8sErrors.IsNotFound(err) {
			return reconcile.Result{}, nil
		}

		return reconcile.Result{}, fmt.Errorf("failed to get %s: %w", cfg.Kind, err)
	}

	kClient, realmName, stop, err := initRealmRef(ctx, k8sClient, obj, cfg)
	if err != nil {
		if errors.Is(err, ErrKeycloakIsNotAvailable) {
			return RequeueOnKeycloakNotAvailable, nil
		}

		return reconcile.Result{}, UpdateErrorStatus(ctx, k8sClient, cfg.Kind, obj, err)
	}

	if stop {
		return reconcile.Result{}, nil
	}

	if obj.GetDeletionTimestamp() != nil {
		return reconcile.Result{}, deleteRealmRef(ctx, k8sClient, obj, kClient, realmName, cfg)
	}

	return cfg.HandleReconciliation(ctx, obj, kClient, realmName)
}

// UpdateErrorStatus writes err to status.value and returns err.
// On a failed status write it returns an error wrapping both err and the write error.
// An unchanged status.value is not written again.
func UpdateErrorStatus(ctx context.Context, k8sClient client.Client, kind string, obj RealmRefObject, err error) error {
	ctrl.LoggerFrom(ctx).Error(err, "An error has occurred while handling "+kind)

	if obj.GetStatus() == err.Error() {
		return err
	}

	obj.SetStatus(err.Error())

	if statusErr := k8sClient.Status().Update(ctx, obj); statusErr != nil {
		return fmt.Errorf("failed to update %s status (%w): %w", kind, err, statusErr)
	}

	return err
}

// initRealmRef returns stop=true when the finalizers were removed because the realm is gone.
func initRealmRef[T RealmRefObject](
	ctx context.Context,
	k8sClient client.Client,
	obj T,
	cfg RealmRefReconcile[T],
) (*keycloakapi.KeycloakClient, string, bool, error) {
	if err := cfg.Helper.SetRealmOwnerRef(ctx, obj); err != nil {
		return nil, "", false, fmt.Errorf("unable to set realm owner ref: %w", err)
	}

	kClient, err := cfg.Helper.CreateKeycloakClientFromRealmRef(ctx, obj)
	if err != nil {
		if errors.Is(err, ErrKeycloakRealmNotFound) && obj.GetDeletionTimestamp() != nil {
			stop, removeErr := RemoveFinalizersOnRealmNotFound(ctx, k8sClient, obj, cfg.Finalizers...)
			if removeErr != nil {
				return nil, "", false, removeErr
			}

			if stop {
				return nil, "", true, nil
			}
		}

		return nil, "", false, fmt.Errorf("failed to create Keycloak client: %w", err)
	}

	realmName, err := cfg.Helper.GetRealmNameFromRef(ctx, obj)
	if err != nil {
		return nil, "", false, fmt.Errorf("unable to get realm name from ref: %w", err)
	}

	return kClient, realmName, false, nil
}

func deleteRealmRef[T RealmRefObject](
	ctx context.Context,
	k8sClient client.Client,
	obj T,
	kClient *keycloakapi.KeycloakClient,
	realmName string,
	cfg RealmRefReconcile[T],
) error {
	hasFinalizer := slices.ContainsFunc(cfg.Finalizers, func(f string) bool {
		return controllerutil.ContainsFinalizer(obj, f)
	})

	if !hasFinalizer {
		return nil
	}

	if err := cfg.Remove(ctx, obj, kClient, realmName); err != nil {
		return UpdateErrorStatus(ctx, k8sClient, cfg.Kind, obj, err)
	}

	original := deepCopyObject(obj)

	for _, finalizer := range cfg.Finalizers {
		controllerutil.RemoveFinalizer(obj, finalizer)
	}

	if err := PatchObject(ctx, k8sClient, original, obj); err != nil {
		return UpdateErrorStatus(ctx, k8sClient, cfg.Kind, obj, fmt.Errorf("failed to update %s after finalizer removal: %w", cfg.Kind, err))
	}

	return nil
}

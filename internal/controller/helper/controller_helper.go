package helper

import (
	"context"
	"fmt"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	"github.com/epam/edp-keycloak-operator/api/common"
	keycloakApi "github.com/epam/edp-keycloak-operator/api/v1"
	keycloakAlpha "github.com/epam/edp-keycloak-operator/api/v1alpha1"
	keycloakClient "github.com/epam/edp-keycloak-operator/pkg/client/keycloakapi"
)

const (
	RequeueOnKeycloakNotAvailablePeriod = time.Minute
)

var (
	RequeueOnKeycloakNotAvailable = ctrl.Result{
		RequeueAfter: RequeueOnKeycloakNotAvailablePeriod,
	}
)

type Terminator interface {
	DeleteResource(ctx context.Context) error
}

type ObjectWithRealmRef interface {
	common.HasRealmRef
	client.Object
}

type ObjectWithKeycloakRef interface {
	common.HasKeycloakRef
	client.Object
}

// ControllerHelper interface defines methods for working with keycloak client and owner references.
type ControllerHelper interface {
	SetKeycloakOwnerRef(ctx context.Context, object ObjectWithKeycloakRef) error
	SetRealmOwnerRef(ctx context.Context, object ObjectWithRealmRef) error
	SetFailureCount(fc FailureCountable) time.Duration
	TryToDelete(ctx context.Context, obj client.Object, terminator Terminator, finalizer string) (isDeleted bool, resultErr error)
	TryRemoveFinalizer(ctx context.Context, obj client.Object, finalizer string) error
	CreateKeycloakClientFromKeycloak(ctx context.Context, kc *keycloakApi.Keycloak) (*keycloakClient.KeycloakClient, error)
	CreateKeycloakClientFromClusterKeycloak(ctx context.Context, clusterKeycloak *keycloakAlpha.ClusterKeycloak) (*keycloakClient.KeycloakClient, error)
	CreateKeycloakClientFromRealmRef(ctx context.Context, object ObjectWithRealmRef) (*keycloakClient.KeycloakClient, error)
	CreateKeycloakClientFromRealm(ctx context.Context, realm *keycloakApi.KeycloakRealm) (*keycloakClient.KeycloakClient, error)
	CreateKeycloakClientFromClusterRealm(ctx context.Context, realm *keycloakAlpha.ClusterKeycloakRealm) (*keycloakClient.KeycloakClient, error)
	GetRealmNameFromRef(ctx context.Context, object ObjectWithRealmRef) (string, error)
}

type Helper struct {
	client            client.Client
	scheme            *runtime.Scheme
	operatorNamespace string
	// enableOwnerRef is a flag to enable legacy owner reference to Keycloak and KeycloakRealm for operator objects.
	// This is needed for backward compatibility with the old version of the operator.
	enableOwnerRef bool
}

func MakeHelper(k8sClient client.Client, scheme *runtime.Scheme, operatorNamespace string, options ...func(*Helper)) *Helper {
	helper := &Helper{
		client:            k8sClient,
		scheme:            scheme,
		operatorNamespace: operatorNamespace,
		enableOwnerRef:    false,
	}

	for _, option := range options {
		option(helper)
	}

	return helper
}

// EnableOwnerRef is an option to set the enableOwnerRef field in Helper.
func EnableOwnerRef(setOwnerRef bool) func(*Helper) {
	return func(h *Helper) {
		h.enableOwnerRef = setOwnerRef
	}
}

// SetKeycloakOwnerRef sets owner reference for object.
//
//nolint:dupl,cyclop
func (h *Helper) SetKeycloakOwnerRef(ctx context.Context, object ObjectWithKeycloakRef) error {
	if !h.enableOwnerRef {
		return nil
	}

	if metav1.GetControllerOf(object) != nil {
		return nil
	}

	kind := object.GetKeycloakRef().Kind
	name := object.GetKeycloakRef().Name

	switch kind {
	case keycloakApi.KeycloakKind:
		kc := &keycloakApi.Keycloak{}
		if err := h.client.Get(ctx, types.NamespacedName{
			Namespace: object.GetNamespace(),
			Name:      name,
		}, kc); err != nil {
			return fmt.Errorf("failed to get Keycloak: %w", err)
		}

		if err := controllerutil.SetControllerReference(kc, object, h.scheme); err != nil {
			return fmt.Errorf("failed to set controller reference for %s: %w", object.GetName(), err)
		}

		if err := h.client.Update(ctx, object); err != nil {
			return fmt.Errorf("failed to update keycloak owner reference %s: %w", kc.GetName(), err)
		}

		return nil

	case keycloakAlpha.ClusterKeycloakKind:
		clusterKc := &keycloakAlpha.ClusterKeycloak{}
		if err := h.client.Get(ctx, types.NamespacedName{
			Name: name,
		}, clusterKc); err != nil {
			return fmt.Errorf("failed to get ClusterKeycloak: %w", err)
		}

		if err := controllerutil.SetControllerReference(clusterKc, object, h.scheme); err != nil {
			return fmt.Errorf("failed to set controller reference for %s: %w", object.GetName(), err)
		}

		if err := h.client.Update(ctx, object); err != nil {
			return fmt.Errorf("failed to update keycloak owner reference %s: %w", clusterKc.GetName(), err)
		}

		return nil

	default:
		return fmt.Errorf("unknown keycloak kind: %s", kind)
	}
}

// SetRealmOwnerRef sets owner reference for object.
//
//nolint:dupl,cyclop
func (h *Helper) SetRealmOwnerRef(ctx context.Context, object ObjectWithRealmRef) error {
	if !h.enableOwnerRef {
		return nil
	}

	if metav1.GetControllerOf(object) != nil {
		return nil
	}

	kind := object.GetRealmRef().Kind
	name := object.GetRealmRef().Name

	switch kind {
	case keycloakApi.KeycloakRealmKind:
		realm := &keycloakApi.KeycloakRealm{}
		if err := h.client.Get(ctx, types.NamespacedName{
			Namespace: object.GetNamespace(),
			Name:      name,
		}, realm); err != nil {
			return fmt.Errorf("failed to get KeycloakRealm: %w", err)
		}

		if err := controllerutil.SetControllerReference(realm, object, h.scheme); err != nil {
			return fmt.Errorf("failed to set controller reference for %s: %w", object.GetName(), err)
		}

		if err := h.client.Update(ctx, object); err != nil {
			return fmt.Errorf("failed to update realm owner reference %s: %w", realm.GetName(), err)
		}

		return nil

	case keycloakAlpha.ClusterKeycloakRealmKind:
		clusterRealm := &keycloakAlpha.ClusterKeycloakRealm{}
		if err := h.client.Get(ctx, types.NamespacedName{
			Name: name,
		}, clusterRealm); err != nil {
			return fmt.Errorf("failed to get ClusterKeycloakRealm: %w", err)
		}

		if err := controllerutil.SetControllerReference(clusterRealm, object, h.scheme); err != nil {
			return fmt.Errorf("unable to set controller reference for %s: %w", object.GetName(), err)
		}

		if err := h.client.Update(ctx, object); err != nil {
			return fmt.Errorf("failed to update realm owner reference %s: %w", clusterRealm.GetName(), err)
		}

		return nil

	default:
		return fmt.Errorf("unknown realm kind: %s", kind)
	}
}

func (h *Helper) TryRemoveFinalizer(ctx context.Context, obj client.Object, finalizer string) error {
	if !obj.GetDeletionTimestamp().IsZero() {
		if controllerutil.RemoveFinalizer(obj, finalizer) {
			if err := h.client.Update(ctx, obj); err != nil {
				return fmt.Errorf("unable to update instance: %w", err)
			}
		}
	}

	return nil
}

// RemoveFinalizersOnRealmNotFound removes the given finalizers from obj and persists the update
// when the realm is gone and the object is being deleted.
// Returns true when cleanup was performed and reconciliation should stop.
func RemoveFinalizersOnRealmNotFound(ctx context.Context, k8sClient client.Client, obj client.Object, finalizers ...string) (bool, error) {
	log := ctrl.LoggerFrom(ctx)
	log.Info("Keycloak realm not found, removing finalizers")

	removed := false
	for _, f := range finalizers {
		removed = controllerutil.RemoveFinalizer(obj, f) || removed
	}

	if removed {
		if err := k8sClient.Update(ctx, obj); err != nil {
			return false, fmt.Errorf("failed to remove finalizers: %w", err)
		}
	}

	log.Info("Finalizers removed")

	return true, nil
}

func (h *Helper) TryToDelete(ctx context.Context, obj client.Object, terminator Terminator, finalizer string) (isDeleted bool, resultErr error) {
	logger := ctrl.LoggerFrom(ctx)

	if obj.GetDeletionTimestamp().IsZero() {
		logger.Info("instance timestamp is zero")

		if controllerutil.AddFinalizer(obj, finalizer) {
			logger.Info("Adding finalizer to instance")

			if err := h.client.Update(ctx, obj); err != nil {
				return false, fmt.Errorf("unable to update deletable object: %w", err)
			}
		}

		logger.Info("processing finalizers done, exit.")

		return false, nil
	}

	logger.Info("terminator deleting resource")

	if err := terminator.DeleteResource(ctx); err != nil {
		return false, fmt.Errorf("error during keycloak resource deletion: %w", err)
	}

	logger.Info("terminator removing finalizers")

	if controllerutil.RemoveFinalizer(obj, finalizer) {
		if err := h.client.Update(ctx, obj); err != nil {
			return false, fmt.Errorf("unable to update instance: %w", err)
		}
	}

	logger.Info("terminator deleting instance done, exit")

	return true, nil
}

// GetRealmNameFromRef resolves the Keycloak realm name from a RealmRef without calling the Keycloak API.
// It reads the realm name directly from the CR spec.
func (h *Helper) GetRealmNameFromRef(ctx context.Context, object ObjectWithRealmRef) (string, error) {
	kind := object.GetRealmRef().Kind
	name := object.GetRealmRef().Name

	switch kind {
	case keycloakApi.KeycloakRealmKind:
		realm := &keycloakApi.KeycloakRealm{}
		if err := h.client.Get(ctx, types.NamespacedName{
			Namespace: object.GetNamespace(),
			Name:      name,
		}, realm); err != nil {
			return "", fmt.Errorf("failed to get KeycloakRealm: %w", err)
		}

		return realm.Spec.RealmName, nil

	case keycloakAlpha.ClusterKeycloakRealmKind:
		clusterRealm := &keycloakAlpha.ClusterKeycloakRealm{}
		if err := h.client.Get(ctx, types.NamespacedName{Name: name}, clusterRealm); err != nil {
			return "", fmt.Errorf("failed to get ClusterKeycloakRealm: %w", err)
		}

		return clusterRealm.Spec.RealmName, nil

	default:
		return "", fmt.Errorf("unknown realm kind: %s", kind)
	}
}

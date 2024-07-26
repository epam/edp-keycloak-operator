package helper

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/Nerzal/gocloak/v12"
	"github.com/go-logr/logr"
	"github.com/go-resty/resty/v2"
	"github.com/pkg/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	"github.com/epam/edp-keycloak-operator/api/common"
	keycloakApi "github.com/epam/edp-keycloak-operator/api/v1"
	keycloakAlpha "github.com/epam/edp-keycloak-operator/api/v1alpha1"
	"github.com/epam/edp-keycloak-operator/pkg/client/keycloak"
	"github.com/epam/edp-keycloak-operator/pkg/client/keycloak/adapter"
)

const (
	StatusOK                            = "OK"
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

type adapterBuilder func(
	ctx context.Context,
	conf adapter.GoCloakConfig,
	adminType string,
	log logr.Logger,
	restyClient *resty.Client,
) (keycloak.Client, error)

// ControllerHelper interface defines methods for working with keycloak client and owner references.
//
//go:generate mockery --name ControllerHelper --filename helper_mock.go
type ControllerHelper interface {
	SetFailureCount(fc FailureCountable) time.Duration
	TryToDelete(ctx context.Context, obj client.Object, terminator Terminator, finalizer string) (isDeleted bool, resultErr error)
	GetKeycloakRealmFromRef(ctx context.Context, object ObjectWithRealmRef, kcClient keycloak.Client) (*gocloak.RealmRepresentation, error)
	CreateKeycloakClientFromRealmRef(ctx context.Context, object ObjectWithRealmRef) (keycloak.Client, error)
	CreateKeycloakClientFromRealm(ctx context.Context, realm *keycloakApi.KeycloakRealm) (keycloak.Client, error)
	CreateKeycloakClientFromClusterRealm(ctx context.Context, realm *keycloakAlpha.ClusterKeycloakRealm) (keycloak.Client, error)
	CreateKeycloakClient(ctx context.Context, url, user, password, adminType, caCert string, insecureSkipVerify bool) (keycloak.Client, error)
	CreateKeycloakClientFomAuthData(ctx context.Context, authData *KeycloakAuthData) (keycloak.Client, error)
	InvalidateKeycloakClientTokenSecret(ctx context.Context, namespace, rootKeycloakName string) error
}

type Helper struct {
	client            client.Client
	scheme            *runtime.Scheme
	restyClient       *resty.Client
	adapterBuilder    adapterBuilder
	tokenSecretLock   *sync.Mutex
	operatorNamespace string
}

func MakeHelper(client client.Client, scheme *runtime.Scheme, operatorNamespace string) *Helper {
	return &Helper{
		tokenSecretLock:   new(sync.Mutex),
		client:            client,
		scheme:            scheme,
		operatorNamespace: operatorNamespace,
		adapterBuilder: func(
			ctx context.Context,
			conf adapter.GoCloakConfig,
			adminType string,
			log logr.Logger,
			restyClient *resty.Client,
		) (keycloak.Client, error) {
			if adminType == keycloakApi.KeycloakAdminTypeServiceAccount {
				goKeycloakAdapter, err := adapter.MakeFromServiceAccount(ctx, conf, "master", log, restyClient)
				if err != nil {
					return nil, fmt.Errorf("failed to make go keycloak adapter from seviceaccount: %w", err)
				}

				return goKeycloakAdapter, nil
			}

			goKeycloakAdapter, err := adapter.Make(ctx, conf, log, restyClient)
			if err != nil {
				return nil, fmt.Errorf("failed to make go keycloak adapter: %w", err)
			}

			return goKeycloakAdapter, nil
		},
	}
}

func (h *Helper) TryToDelete(ctx context.Context, obj client.Object, terminator Terminator, finalizer string) (isDeleted bool, resultErr error) {
	logger := ctrl.LoggerFrom(ctx)

	if obj.GetDeletionTimestamp().IsZero() {
		logger.Info("instance timestamp is zero")

		if controllerutil.AddFinalizer(obj, finalizer) {
			logger.Info("Adding finalizer to instance")

			if err := h.client.Update(ctx, obj); err != nil {
				return false, errors.Wrap(err, "unable to update deletable object")
			}
		}

		logger.Info("processing finalizers done, exit.")

		return false, nil
	}

	logger.Info("terminator deleting resource")

	if err := terminator.DeleteResource(ctx); err != nil {
		return false, errors.Wrap(err, "error during keycloak resource deletion")
	}

	logger.Info("terminator removing finalizers")

	if controllerutil.RemoveFinalizer(obj, finalizer) {
		if err := h.client.Update(ctx, obj); err != nil {
			return false, errors.Wrap(err, "unable to update instance")
		}
	}

	logger.Info("terminator deleting instance done, exit")

	return true, nil
}

func (h *Helper) GetKeycloakRealmFromRef(ctx context.Context, object ObjectWithRealmRef, kcClient keycloak.Client) (*gocloak.RealmRepresentation, error) {
	kind := object.GetRealmRef().Kind
	name := object.GetRealmRef().Name

	switch kind {
	case keycloakApi.KeycloakRealmKind:
		realm := &keycloakApi.KeycloakRealm{}
		if err := h.client.Get(ctx, types.NamespacedName{
			Namespace: object.GetNamespace(),
			Name:      name,
		}, realm); err != nil {
			return nil, fmt.Errorf("failed to get KeycloakRealm: %w", err)
		}

		kcRealm, err := kcClient.GetRealm(ctx, realm.Spec.RealmName)
		if err != nil {
			return nil, fmt.Errorf("failed to get realm: %w", err)
		}

		return kcRealm, nil

	case keycloakAlpha.ClusterKeycloakRealmKind:
		clusterRealm := &keycloakAlpha.ClusterKeycloakRealm{}
		if err := h.client.Get(ctx, types.NamespacedName{
			Name: name,
		}, clusterRealm); err != nil {
			return nil, fmt.Errorf("failed to get ClusterKeycloakRealm: %w", err)
		}

		kcRealm, err := kcClient.GetRealm(ctx, clusterRealm.Spec.RealmName)
		if err != nil {
			return nil, fmt.Errorf("failed to get realm: %w", err)
		}

		return kcRealm, nil

	default:
		return nil, fmt.Errorf("unknown realm kind: %s", kind)
	}
}

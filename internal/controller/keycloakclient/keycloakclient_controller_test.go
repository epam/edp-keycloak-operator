package keycloakclient

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/go-logr/logr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	k8sErrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/client/interceptor"

	"github.com/epam/edp-keycloak-operator/api/common"
	keycloakApi "github.com/epam/edp-keycloak-operator/api/v1"
	"github.com/epam/edp-keycloak-operator/internal/controller/helper"
	helpermock "github.com/epam/edp-keycloak-operator/internal/controller/helper/mocks"
	"github.com/epam/edp-keycloak-operator/internal/controller/keycloakclient/chain"
	"github.com/epam/edp-keycloak-operator/pkg/client/keycloakapi"
	keycloakapimocks "github.com/epam/edp-keycloak-operator/pkg/client/keycloakapi/mocks"
	"github.com/epam/edp-keycloak-operator/pkg/destination"
	"github.com/epam/edp-keycloak-operator/pkg/secretref"
)

// Chain failure persists the incremented failureCount and returns a nil error with RequeueAfter from SetFailureCount.
func TestReconcile_ChainFailureRequeuesAfterBackoff(t *testing.T) {
	t.Parallel()

	scheme := runtime.NewScheme()
	require.NoError(t, corev1.AddToScheme(scheme))
	require.NoError(t, keycloakApi.AddToScheme(scheme))

	key := client.ObjectKey{Namespace: "default", Name: "client"}
	kc := &keycloakApi.KeycloakClient{
		ObjectMeta: metav1.ObjectMeta{Namespace: key.Namespace, Name: key.Name},
		Spec: keycloakApi.KeycloakClientSpec{
			ClientId: "client",
			Secret:   secretref.GenerateSecretRef("missing-secret", "key"),
		},
		Status: keycloakApi.KeycloakClientStatus{FailureCount: 2},
	}
	cl := fake.NewClientBuilder().
		WithScheme(scheme).
		WithStatusSubresource(&keycloakApi.KeycloakClient{}).
		WithObjects(kc).
		Build()

	secretRef, err := secretref.NewSecretRef(cl, destination.AllowAll())
	require.NoError(t, err)

	const backoff = 30 * time.Second

	h := helpermock.NewMockControllerHelper(t)
	h.EXPECT().SetRealmOwnerRef(mock.Anything, mock.Anything).Return(nil)
	h.EXPECT().CreateKeycloakClientFromRealmRef(mock.Anything, mock.Anything).Return(&keycloakapi.KeycloakClient{}, nil)
	h.EXPECT().GetRealmNameFromRef(mock.Anything, mock.Anything).Return("realm", nil)
	h.EXPECT().SetFailureCount(mock.Anything).RunAndReturn(func(fc helper.FailureCountable) time.Duration {
		fc.SetFailureCount(fc.GetFailureCount() + 1)

		return backoff
	})

	ctx := ctrl.LoggerInto(context.Background(), logr.Discard())
	r := NewReconcileKeycloakClient(cl, h, secretRef)

	res, err := r.Reconcile(ctx, ctrl.Request{NamespacedName: key})
	require.NoError(t, err)
	assert.Equal(t, backoff, res.RequeueAfter)

	got := &keycloakApi.KeycloakClient{}
	require.NoError(t, cl.Get(ctx, key, got))
	assert.Equal(t, int64(3), got.Status.FailureCount)
	assert.Contains(t, got.Status.Value, "missing-secret")

	ready := meta.FindStatusCondition(got.Status.Conditions, chain.ConditionReady)
	require.NotNil(t, ready)
	assert.Equal(t, metav1.ConditionFalse, ready.Status)
	assert.Contains(t, ready.Message, "missing-secret")
}

func TestReconcile_PreChainErrorWritesStatus(t *testing.T) {
	t.Parallel()

	const (
		backoff = 30 * time.Second
		oldFC   = int64(2)
	)

	scheme := runtime.NewScheme()
	require.NoError(t, corev1.AddToScheme(scheme))
	require.NoError(t, keycloakApi.AddToScheme(scheme))

	key := client.ObjectKey{Namespace: "default", Name: "client"}

	newClient := func() *keycloakApi.KeycloakClient {
		return &keycloakApi.KeycloakClient{
			ObjectMeta: metav1.ObjectMeta{Namespace: key.Namespace, Name: key.Name, Generation: 2},
			Spec:       keycloakApi.KeycloakClientSpec{ClientId: "client"},
			Status:     keycloakApi.KeycloakClientStatus{FailureCount: oldFC},
		}
	}

	deleting := func(kc *keycloakApi.KeycloakClient) {
		now := metav1.Now()
		kc.DeletionTimestamp = &now
		kc.Finalizers = []string{keyCloakClientOperatorFinalizerName}
	}

	okStatus := func(kc *keycloakApi.KeycloakClient) {
		kc.Status.FailureCount = 0
		kc.Status.Value = "OK"
		kc.Status.Conditions = []metav1.Condition{{
			Type:               chain.ConditionReady,
			Status:             metav1.ConditionTrue,
			Reason:             chain.ReasonReconciliationSucceeded,
			LastTransitionTime: metav1.Now(),
		}}
	}

	failPatch := func(context.Context, client.WithWatch, client.Object, client.Patch, ...client.PatchOption) error {
		return k8sErrors.NewInternalError(errors.New("patch denied"))
	}

	tests := []struct {
		name         string
		mutate       func(kc *keycloakApi.KeycloakClient)
		setup        func(h *helpermock.MockControllerHelper)
		setupT       func(t *testing.T, h *helpermock.MockControllerHelper)
		patch        func(context.Context, client.WithWatch, client.Object, client.Patch, ...client.PatchOption) error
		statusFails  bool
		wantValue    string
		wantCount    int64
		wantRequeue  time.Duration
		wantErrMsg   string
		wantWrites   int
		wantGone     bool
		wantNoStatus bool
	}{
		{
			name: "owner ref error",
			setup: func(h *helpermock.MockControllerHelper) {
				h.EXPECT().SetRealmOwnerRef(mock.Anything, mock.Anything).Return(errors.New("realm get failed"))
			},
			wantValue:   "unable to set realm owner ref: realm get failed",
			wantCount:   oldFC + 1,
			wantRequeue: backoff,
			wantWrites:  1,
		},
		{
			name: "client creation error",
			setup: func(h *helpermock.MockControllerHelper) {
				h.EXPECT().SetRealmOwnerRef(mock.Anything, mock.Anything).Return(nil)
				h.EXPECT().CreateKeycloakClientFromRealmRef(mock.Anything, mock.Anything).
					Return(nil, errors.New("unable to get realm"))
			},
			wantValue:   "unable to create keycloak client from realm ref: unable to get realm",
			wantCount:   oldFC + 1,
			wantRequeue: backoff,
			wantWrites:  1,
		},
		{
			name: "realm name error",
			setup: func(h *helpermock.MockControllerHelper) {
				h.EXPECT().SetRealmOwnerRef(mock.Anything, mock.Anything).Return(nil)
				h.EXPECT().CreateKeycloakClientFromRealmRef(mock.Anything, mock.Anything).
					Return(&keycloakapi.KeycloakClient{}, nil)
				h.EXPECT().GetRealmNameFromRef(mock.Anything, mock.Anything).Return("", errors.New("no realm name"))
			},
			wantValue:   "unable to get realm name from ref: no realm name",
			wantCount:   oldFC + 1,
			wantRequeue: backoff,
			wantWrites:  1,
		},
		{
			name: "keycloak not available requeues without status write",
			setup: func(h *helpermock.MockControllerHelper) {
				h.EXPECT().SetRealmOwnerRef(mock.Anything, mock.Anything).Return(nil)
				h.EXPECT().CreateKeycloakClientFromRealmRef(mock.Anything, mock.Anything).
					Return(nil, fmt.Errorf("x: %w", helper.ErrKeycloakIsNotAvailable))
			},
			wantRequeue:  helper.RequeueOnKeycloakNotAvailablePeriod,
			wantNoStatus: true,
		},
		{
			name: "finalizer add error",
			setup: func(h *helpermock.MockControllerHelper) {
				h.EXPECT().SetRealmOwnerRef(mock.Anything, mock.Anything).Return(nil)
				h.EXPECT().CreateKeycloakClientFromRealmRef(mock.Anything, mock.Anything).
					Return(&keycloakapi.KeycloakClient{}, nil)
				h.EXPECT().GetRealmNameFromRef(mock.Anything, mock.Anything).Return("realm", nil)
			},
			patch:       failPatch,
			wantValue:   "failed to add finalizer to keycloak client",
			wantCount:   oldFC + 1,
			wantRequeue: backoff,
			wantWrites:  1,
		},
		{
			name:   "deleting, realm gone, finalizer removal error",
			mutate: deleting,
			setup: func(h *helpermock.MockControllerHelper) {
				h.EXPECT().SetRealmOwnerRef(mock.Anything, mock.Anything).Return(nil)
				h.EXPECT().CreateKeycloakClientFromRealmRef(mock.Anything, mock.Anything).
					Return(nil, helper.ErrKeycloakRealmNotFound)
			},
			patch:       failPatch,
			wantValue:   "failed to remove finalizers",
			wantCount:   oldFC + 1,
			wantRequeue: backoff,
			wantWrites:  1,
		},
		{
			name:   "deleting, realm gone, finalizer removed stops without status write",
			mutate: deleting,
			setup: func(h *helpermock.MockControllerHelper) {
				h.EXPECT().SetRealmOwnerRef(mock.Anything, mock.Anything).Return(nil)
				h.EXPECT().CreateKeycloakClientFromRealmRef(mock.Anything, mock.Anything).
					Return(nil, helper.ErrKeycloakRealmNotFound)
			},
			wantGone: true,
		},
		{
			name: "deleting, keycloak client removal error",
			mutate: func(kc *keycloakApi.KeycloakClient) {
				deleting(kc)
				kc.Status.ClientID = "client-uuid"
			},
			setupT: func(t *testing.T, h *helpermock.MockControllerHelper) {
				clients := keycloakapimocks.NewMockClientsClient(t)
				clients.EXPECT().DeleteClient(mock.Anything, "realm", "client-uuid").
					Return(nil, errors.New("delete denied"))

				h.EXPECT().SetRealmOwnerRef(mock.Anything, mock.Anything).Return(nil)
				h.EXPECT().CreateKeycloakClientFromRealmRef(mock.Anything, mock.Anything).
					Return(&keycloakapi.KeycloakClient{Clients: clients}, nil)
				h.EXPECT().GetRealmNameFromRef(mock.Anything, mock.Anything).Return("realm", nil)
			},
			wantValue:   "failed to remove keycloak client",
			wantCount:   oldFC + 1,
			wantRequeue: backoff,
			wantWrites:  1,
		},
		{
			name:   "deleting, finalizer removal after client removal error",
			mutate: deleting,
			setup: func(h *helpermock.MockControllerHelper) {
				h.EXPECT().SetRealmOwnerRef(mock.Anything, mock.Anything).Return(nil)
				h.EXPECT().CreateKeycloakClientFromRealmRef(mock.Anything, mock.Anything).
					Return(&keycloakapi.KeycloakClient{}, nil)
				h.EXPECT().GetRealmNameFromRef(mock.Anything, mock.Anything).Return("realm", nil)
			},
			patch:       failPatch,
			wantValue:   "failed to update keycloak client after finalizer removal",
			wantCount:   oldFC + 1,
			wantRequeue: backoff,
			wantWrites:  1,
		},
		{
			name:   "healthy client re-pointed to a missing realm shows the error",
			mutate: okStatus,
			setup: func(h *helpermock.MockControllerHelper) {
				h.EXPECT().SetRealmOwnerRef(mock.Anything, mock.Anything).Return(nil)
				h.EXPECT().CreateKeycloakClientFromRealmRef(mock.Anything, mock.Anything).
					Return(nil, errors.New("unable to get realm: not found"))
			},
			wantValue:   "unable to create keycloak client from realm ref: unable to get realm: not found",
			wantCount:   1,
			wantRequeue: backoff,
			wantWrites:  1,
		},
		{
			name: "status write conflict returns the status error and stores nothing",
			setup: func(h *helpermock.MockControllerHelper) {
				h.EXPECT().SetRealmOwnerRef(mock.Anything, mock.Anything).Return(errors.New("realm get failed"))
			},
			statusFails:  true,
			wantErrMsg:   "unable to update status",
			wantWrites:   1,
			wantNoStatus: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			kc := newClient()
			if tt.mutate != nil {
				tt.mutate(kc)
			}

			writes := 0

			cl := fake.NewClientBuilder().
				WithScheme(scheme).
				WithStatusSubresource(&keycloakApi.KeycloakClient{}).
				WithObjects(kc).
				WithInterceptorFuncs(interceptor.Funcs{
					Patch: tt.patch,
					SubResourceUpdate: func(
						ctx context.Context, c client.Client, subResourceName string, obj client.Object, opts ...client.SubResourceUpdateOption,
					) error {
						writes++

						if tt.statusFails {
							return k8sErrors.NewConflict(keycloakApi.GroupVersion.WithResource("keycloakclients").GroupResource(), obj.GetName(), errors.New("conflict"))
						}

						return c.SubResource(subResourceName).Update(ctx, obj, opts...)
					},
				}).
				Build()

			h := helpermock.NewMockControllerHelper(t)

			if tt.setupT != nil {
				tt.setupT(t, h)
			} else {
				tt.setup(h)
			}

			if tt.wantWrites > 0 {
				h.EXPECT().SetFailureCount(mock.Anything).RunAndReturn(func(fc helper.FailureCountable) time.Duration {
					fc.SetFailureCount(fc.GetFailureCount() + 1)

					return backoff
				})
			}

			secretRef, err := secretref.NewSecretRef(cl, destination.AllowAll())
			require.NoError(t, err)

			ctx := ctrl.LoggerInto(context.Background(), logr.Discard())
			r := NewReconcileKeycloakClient(cl, h, secretRef)

			res, err := r.Reconcile(ctx, ctrl.Request{NamespacedName: key})

			if tt.wantErrMsg != "" {
				require.ErrorContains(t, err, tt.wantErrMsg)
			} else {
				require.NoError(t, err)
			}

			assert.Equal(t, tt.wantRequeue, res.RequeueAfter)
			assert.Equal(t, tt.wantWrites, writes)

			got := &keycloakApi.KeycloakClient{}
			getErr := cl.Get(ctx, key, got)

			if tt.wantGone {
				require.True(t, k8sErrors.IsNotFound(getErr))

				return
			}

			require.NoError(t, getErr)

			if tt.wantNoStatus {
				assert.Equal(t, kc.Status.Value, got.Status.Value)
				assert.Equal(t, kc.Status.FailureCount, got.Status.FailureCount)

				return
			}

			assert.Contains(t, got.Status.Value, tt.wantValue)
			assert.Equal(t, tt.wantCount, got.Status.FailureCount)

			ready := meta.FindStatusCondition(got.Status.Conditions, chain.ConditionReady)
			require.NotNil(t, ready)
			assert.Equal(t, metav1.ConditionFalse, ready.Status)
			assert.Equal(t, chain.ReasonKeycloakAPIError, ready.Reason)
			assert.Equal(t, kc.Generation, ready.ObservedGeneration)
			assert.Contains(t, ready.Message, tt.wantValue)
		})
	}
}

func TestReconcile_DeletingRealmGoneWithOwnerRefEnabled(t *testing.T) {
	t.Parallel()

	scheme := runtime.NewScheme()
	require.NoError(t, corev1.AddToScheme(scheme))
	require.NoError(t, keycloakApi.AddToScheme(scheme))

	now := metav1.Now()
	key := client.ObjectKey{Namespace: "default", Name: "client"}
	kc := &keycloakApi.KeycloakClient{
		ObjectMeta: metav1.ObjectMeta{
			Namespace:         key.Namespace,
			Name:              key.Name,
			DeletionTimestamp: &now,
			Finalizers:        []string{keyCloakClientOperatorFinalizerName},
		},
		Spec: keycloakApi.KeycloakClientSpec{
			ClientId: "client",
			RealmRef: common.RealmRef{Kind: keycloakApi.KeycloakRealmKind, Name: "missing-realm"},
		},
	}
	cl := fake.NewClientBuilder().
		WithScheme(scheme).
		WithStatusSubresource(&keycloakApi.KeycloakClient{}).
		WithObjects(kc).
		Build()

	h, err := helper.MakeHelper(cl, scheme, "default", destination.AllowAll(), helper.EnableOwnerRef(true))
	require.NoError(t, err)

	secretRef, err := secretref.NewSecretRef(cl, destination.AllowAll())
	require.NoError(t, err)

	ctx := ctrl.LoggerInto(context.Background(), logr.Discard())
	r := NewReconcileKeycloakClient(cl, h, secretRef)

	res, err := r.Reconcile(ctx, ctrl.Request{NamespacedName: key})
	require.NoError(t, err)
	assert.Zero(t, res)

	err = cl.Get(ctx, key, &keycloakApi.KeycloakClient{})
	require.True(t, k8sErrors.IsNotFound(err), "client must be released, got: %v", err)
}

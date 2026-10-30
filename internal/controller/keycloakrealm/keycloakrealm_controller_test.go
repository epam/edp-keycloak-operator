package keycloakrealm

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
	k8sErrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/utils/ptr"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/client/interceptor"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	"github.com/epam/edp-keycloak-operator/api/common"
	keycloakApi "github.com/epam/edp-keycloak-operator/api/v1"
	"github.com/epam/edp-keycloak-operator/internal/controller/helper"
	helpermock "github.com/epam/edp-keycloak-operator/internal/controller/helper/mocks"
	handlermocks "github.com/epam/edp-keycloak-operator/internal/controller/keycloakrealm/chain/handler/mocks"
	"github.com/epam/edp-keycloak-operator/pkg/client/keycloakapi"
)

type setupFunc func(t *testing.T, h *helpermock.MockControllerHelper, chainMock *handlermocks.MockRealmHandler, cl client.Client)

func TestReconcileKeycloakRealm_Reconcile(t *testing.T) {
	t.Parallel()

	const (
		backoff        = 30 * time.Second
		successTimeout = time.Minute
	)

	foreignFinalizer := "test/foreign"
	key := client.ObjectKey{Name: "realm", Namespace: "default"}

	scheme := runtime.NewScheme()
	require.NoError(t, keycloakApi.AddToScheme(scheme))

	expectClient := func(h *helpermock.MockControllerHelper) {
		h.EXPECT().SetKeycloakOwnerRef(mock.Anything, mock.Anything).Return(nil)
		h.EXPECT().CreateKeycloakClientFromRealm(mock.Anything, mock.Anything).Return(&keycloakapi.KeycloakClient{}, nil)
	}

	expectBackoff := func(h *helpermock.MockControllerHelper) {
		h.EXPECT().SetFailureCount(mock.Anything).RunAndReturn(func(fc helper.FailureCountable) time.Duration {
			fc.SetFailureCount(fc.GetFailureCount() + 1)

			return backoff
		})
	}

	tests := []struct {
		name        string
		finalizers  []string
		deleting    bool
		generation  int64
		status      keycloakApi.KeycloakRealmStatus
		setup       setupFunc
		statusFails bool

		wantErr         string
		wantRequeue     time.Duration
		wantWrites      int
		wantGone        bool
		wantStatus      *keycloakApi.KeycloakRealmStatus
		wantValue       string
		wantObservedGen int64
	}{
		{
			name:       "deleting, only operator finalizer, finalizer removed",
			finalizers: []string{keyCloakRealmOperatorFinalizerName},
			deleting:   true,
			generation: 2,
			status:     keycloakApi.KeycloakRealmStatus{Available: true, Value: common.StatusOK, ObservedGeneration: 1},
			setup: func(t *testing.T, h *helpermock.MockControllerHelper, _ *handlermocks.MockRealmHandler, cl client.Client) {
				expectClient(h)
				h.EXPECT().TryToDelete(mock.Anything, mock.Anything, mock.Anything, keyCloakRealmOperatorFinalizerName).
					RunAndReturn(func(ctx context.Context, obj client.Object, _ helper.Terminator, finalizer string) (bool, error) {
						original, ok := obj.DeepCopyObject().(client.Object)
						require.True(t, ok)

						controllerutil.RemoveFinalizer(obj, finalizer)

						return true, helper.PatchObject(ctx, cl, original, obj)
					})
			},
			wantGone: true,
		},
		{
			name:       "deleting, foreign finalizer remains, status untouched",
			finalizers: []string{keyCloakRealmOperatorFinalizerName, foreignFinalizer},
			deleting:   true,
			generation: 2,
			status:     keycloakApi.KeycloakRealmStatus{Available: true, Value: common.StatusOK, ObservedGeneration: 1},
			setup: func(t *testing.T, h *helpermock.MockControllerHelper, _ *handlermocks.MockRealmHandler, cl client.Client) {
				expectClient(h)
				h.EXPECT().TryToDelete(mock.Anything, mock.Anything, mock.Anything, keyCloakRealmOperatorFinalizerName).
					RunAndReturn(func(ctx context.Context, obj client.Object, _ helper.Terminator, finalizer string) (bool, error) {
						original, ok := obj.DeepCopyObject().(client.Object)
						require.True(t, ok)

						controllerutil.RemoveFinalizer(obj, finalizer)

						return true, helper.PatchObject(ctx, cl, original, obj)
					})
			},
			wantStatus: &keycloakApi.KeycloakRealmStatus{Available: true, Value: common.StatusOK, ObservedGeneration: 1},
		},
		{
			name:       "deleting, keycloak delete fails, error status written",
			finalizers: []string{keyCloakRealmOperatorFinalizerName},
			deleting:   true,
			generation: 2,
			status:     keycloakApi.KeycloakRealmStatus{Available: true, Value: common.StatusOK, ObservedGeneration: 1},
			setup: func(_ *testing.T, h *helpermock.MockControllerHelper, _ *handlermocks.MockRealmHandler, _ client.Client) {
				expectClient(h)
				h.EXPECT().TryToDelete(mock.Anything, mock.Anything, mock.Anything, keyCloakRealmOperatorFinalizerName).
					Return(false, errors.New("delete failed"))
				expectBackoff(h)
			},
			wantRequeue:     backoff,
			wantWrites:      1,
			wantValue:       "delete failed",
			wantObservedGen: 1,
		},
		{
			name:       "deleting, finalizer patch fails after in-memory removal, error status written",
			finalizers: []string{keyCloakRealmOperatorFinalizerName},
			deleting:   true,
			generation: 2,
			status:     keycloakApi.KeycloakRealmStatus{Available: true, Value: common.StatusOK, ObservedGeneration: 1},
			setup: func(_ *testing.T, h *helpermock.MockControllerHelper, _ *handlermocks.MockRealmHandler, _ client.Client) {
				expectClient(h)
				h.EXPECT().TryToDelete(mock.Anything, mock.Anything, mock.Anything, keyCloakRealmOperatorFinalizerName).
					RunAndReturn(func(_ context.Context, obj client.Object, _ helper.Terminator, finalizer string) (bool, error) {
						controllerutil.RemoveFinalizer(obj, finalizer)

						return false, errors.New("finalizer patch failed")
					})
				expectBackoff(h)
			},
			wantRequeue:     backoff,
			wantWrites:      1,
			wantValue:       "finalizer patch failed",
			wantObservedGen: 1,
		},
		{
			name:       "live, finalizer add fails, error status written",
			finalizers: []string{keyCloakRealmOperatorFinalizerName},
			generation: 3,
			status:     keycloakApi.KeycloakRealmStatus{Available: true, Value: common.StatusOK, ObservedGeneration: 3},
			setup: func(_ *testing.T, h *helpermock.MockControllerHelper, _ *handlermocks.MockRealmHandler, _ client.Client) {
				expectClient(h)
				h.EXPECT().TryToDelete(mock.Anything, mock.Anything, mock.Anything, keyCloakRealmOperatorFinalizerName).
					Return(false, errors.New("failed to add finalizer: patch refused"))
				expectBackoff(h)
			},
			wantRequeue:     backoff,
			wantWrites:      1,
			wantValue:       "failed to add finalizer: patch refused",
			wantObservedGen: 3,
		},
		{
			name:       "keycloak unavailable, requeue without status write",
			finalizers: []string{keyCloakRealmOperatorFinalizerName},
			deleting:   true,
			generation: 2,
			status:     keycloakApi.KeycloakRealmStatus{Available: true, Value: common.StatusOK, ObservedGeneration: 1},
			setup: func(_ *testing.T, h *helpermock.MockControllerHelper, _ *handlermocks.MockRealmHandler, _ client.Client) {
				h.EXPECT().SetKeycloakOwnerRef(mock.Anything, mock.Anything).Return(nil)
				h.EXPECT().CreateKeycloakClientFromRealm(mock.Anything, mock.Anything).
					Return(nil, fmt.Errorf("x: %w", helper.ErrKeycloakIsNotAvailable))
			},
			wantRequeue: helper.RequeueOnKeycloakNotAvailablePeriod,
			wantStatus:  &keycloakApi.KeycloakRealmStatus{Available: true, Value: common.StatusOK, ObservedGeneration: 1},
		},
		{
			name:       "live, chain ok, status differs",
			finalizers: []string{keyCloakRealmOperatorFinalizerName},
			generation: 3,
			status:     keycloakApi.KeycloakRealmStatus{Available: true, Value: common.StatusOK, ObservedGeneration: 2},
			setup: func(_ *testing.T, h *helpermock.MockControllerHelper, chainMock *handlermocks.MockRealmHandler, _ client.Client) {
				expectClient(h)
				h.EXPECT().TryToDelete(mock.Anything, mock.Anything, mock.Anything, keyCloakRealmOperatorFinalizerName).
					Return(false, nil)
				chainMock.EXPECT().ServeRequest(mock.Anything, mock.Anything, mock.Anything).Return(nil)
			},
			wantRequeue:     successTimeout,
			wantWrites:      1,
			wantValue:       common.StatusOK,
			wantObservedGen: 3,
		},
		{
			name:       "live, chain ok, status already current",
			finalizers: []string{keyCloakRealmOperatorFinalizerName},
			generation: 3,
			status:     keycloakApi.KeycloakRealmStatus{Available: true, Value: common.StatusOK, ObservedGeneration: 3},
			setup: func(_ *testing.T, h *helpermock.MockControllerHelper, chainMock *handlermocks.MockRealmHandler, _ client.Client) {
				expectClient(h)
				h.EXPECT().TryToDelete(mock.Anything, mock.Anything, mock.Anything, keyCloakRealmOperatorFinalizerName).
					Return(false, nil)
				chainMock.EXPECT().ServeRequest(mock.Anything, mock.Anything, mock.Anything).Return(nil)
			},
			wantRequeue: successTimeout,
			wantStatus:  &keycloakApi.KeycloakRealmStatus{Available: true, Value: common.StatusOK, ObservedGeneration: 3},
		},
		{
			name:       "live, chain fails, error status written",
			finalizers: []string{keyCloakRealmOperatorFinalizerName},
			generation: 3,
			status:     keycloakApi.KeycloakRealmStatus{Available: true, Value: common.StatusOK, ObservedGeneration: 3},
			setup: func(_ *testing.T, h *helpermock.MockControllerHelper, chainMock *handlermocks.MockRealmHandler, _ client.Client) {
				expectClient(h)
				h.EXPECT().TryToDelete(mock.Anything, mock.Anything, mock.Anything, keyCloakRealmOperatorFinalizerName).
					Return(false, nil)
				chainMock.EXPECT().ServeRequest(mock.Anything, mock.Anything, mock.Anything).Return(errors.New("chain failed"))
				expectBackoff(h)
			},
			wantRequeue:     backoff,
			wantWrites:      1,
			wantValue:       "error during realm chain: chain failed",
			wantObservedGen: 3,
		},
		{
			name:       "live, status write conflict returns the status error",
			finalizers: []string{keyCloakRealmOperatorFinalizerName},
			generation: 3,
			status:     keycloakApi.KeycloakRealmStatus{Available: true, Value: common.StatusOK, ObservedGeneration: 2},
			setup: func(_ *testing.T, h *helpermock.MockControllerHelper, chainMock *handlermocks.MockRealmHandler, _ client.Client) {
				expectClient(h)
				h.EXPECT().TryToDelete(mock.Anything, mock.Anything, mock.Anything, keyCloakRealmOperatorFinalizerName).
					Return(false, nil)
				chainMock.EXPECT().ServeRequest(mock.Anything, mock.Anything, mock.Anything).Return(nil)
			},
			statusFails: true,
			wantErr:     "unable to update status",
			wantRequeue: successTimeout,
			wantWrites:  1,
			wantStatus:  &keycloakApi.KeycloakRealmStatus{Available: true, Value: common.StatusOK, ObservedGeneration: 2},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			realm := &keycloakApi.KeycloakRealm{
				ObjectMeta: metav1.ObjectMeta{
					Name:       key.Name,
					Namespace:  key.Namespace,
					Generation: tt.generation,
					Finalizers: tt.finalizers,
				},
				Spec:   keycloakApi.KeycloakRealmSpec{RealmName: "realm"},
				Status: tt.status,
			}

			if tt.deleting {
				realm.DeletionTimestamp = ptr.To(metav1.Now())
			}

			writes := 0

			cl := fake.NewClientBuilder().
				WithScheme(scheme).
				WithStatusSubresource(&keycloakApi.KeycloakRealm{}).
				WithObjects(realm).
				WithInterceptorFuncs(interceptor.Funcs{
					SubResourceUpdate: func(
						ctx context.Context, c client.Client, subResourceName string, obj client.Object, opts ...client.SubResourceUpdateOption,
					) error {
						writes++

						if tt.statusFails {
							return k8sErrors.NewConflict(keycloakApi.GroupVersion.WithResource("keycloakrealms").GroupResource(), obj.GetName(), errors.New("conflict"))
						}

						return c.SubResource(subResourceName).Update(ctx, obj, opts...)
					},
				}).
				Build()

			h := helpermock.NewMockControllerHelper(t)
			chainMock := handlermocks.NewMockRealmHandler(t)

			tt.setup(t, h, chainMock, cl)

			r := &ReconcileKeycloakRealm{client: cl, helper: h, chain: chainMock, successReconcileTimeout: successTimeout}
			reqCtx := ctrl.LoggerInto(t.Context(), logr.Discard())

			res, err := r.Reconcile(reqCtx, ctrl.Request{NamespacedName: key})

			if tt.wantErr != "" {
				require.ErrorContains(t, err, tt.wantErr)
			} else {
				require.NoError(t, err)
			}

			assert.Equal(t, tt.wantRequeue, res.RequeueAfter)
			assert.Equal(t, tt.wantWrites, writes)

			got := &keycloakApi.KeycloakRealm{}
			getErr := cl.Get(reqCtx, key, got)

			if tt.wantGone {
				require.True(t, k8sErrors.IsNotFound(getErr), "expected NotFound, got %v", getErr)

				return
			}

			require.NoError(t, getErr)

			if tt.wantStatus != nil {
				assert.Equal(t, *tt.wantStatus, got.Status)

				return
			}

			assert.Equal(t, tt.wantValue, got.Status.Value)
			assert.Equal(t, tt.wantObservedGen, got.Status.ObservedGeneration)
		})
	}
}

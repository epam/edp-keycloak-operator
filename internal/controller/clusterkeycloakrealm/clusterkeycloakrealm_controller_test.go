package clusterkeycloakrealm

import (
	"context"
	"errors"
	"fmt"
	"net/http"
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
	"sigs.k8s.io/controller-runtime/pkg/event"

	keycloakAlpha "github.com/epam/edp-keycloak-operator/api/v1alpha1"
	"github.com/epam/edp-keycloak-operator/internal/controller/helper"
	helpermock "github.com/epam/edp-keycloak-operator/internal/controller/helper/mocks"
	"github.com/epam/edp-keycloak-operator/pkg/client/keycloakapi"
	v2mocks "github.com/epam/edp-keycloak-operator/pkg/client/keycloakapi/mocks"
	"github.com/epam/edp-keycloak-operator/pkg/destination"
)

// Chain failure returns a nil error with RequeueAfter from SetFailureCount.
func TestReconcile_ChainFailureRequeuesAfterBackoff(t *testing.T) {
	t.Parallel()

	scheme := runtime.NewScheme()
	require.NoError(t, keycloakAlpha.AddToScheme(scheme))

	realm := &keycloakAlpha.ClusterKeycloakRealm{
		ObjectMeta: metav1.ObjectMeta{Name: "realm"},
		Spec:       keycloakAlpha.ClusterKeycloakRealmSpec{RealmName: "realm", ClusterKeycloakRef: "kc"},
		Status:     keycloakAlpha.ClusterKeycloakRealmStatus{Available: true, FailureCount: 2},
	}
	cl := fake.NewClientBuilder().
		WithScheme(scheme).
		WithStatusSubresource(&keycloakAlpha.ClusterKeycloakRealm{}).
		WithObjects(realm).
		Build()

	realmClient := v2mocks.NewMockRealmClient(t)
	realmClient.EXPECT().GetRealm(mock.Anything, "realm").
		Return(nil, nil, &keycloakapi.ApiError{Code: http.StatusNotFound})
	realmClient.EXPECT().CreateRealm(mock.Anything, mock.Anything).
		Return(nil, &keycloakapi.ApiError{Code: http.StatusForbidden, Message: "HTTP 403 Forbidden"})

	const backoff = 30 * time.Second

	h := helpermock.NewMockControllerHelper(t)
	h.EXPECT().SetKeycloakOwnerRef(mock.Anything, mock.Anything).Return(nil)
	h.EXPECT().CreateKeycloakClientFromClusterRealm(mock.Anything, mock.Anything).
		Return(&keycloakapi.KeycloakClient{Realms: realmClient}, nil)
	h.EXPECT().TryToDelete(mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(false, nil)
	h.EXPECT().SetFailureCount(mock.Anything).RunAndReturn(func(fc helper.FailureCountable) time.Duration {
		fc.SetFailureCount(fc.GetFailureCount() + 1)

		return backoff
	})

	ctx := ctrl.LoggerInto(context.Background(), logr.Discard())
	r := NewClusterKeycloakRealmReconciler(cl, scheme, h, "default", destination.AllowAll())

	res, err := r.Reconcile(ctx, ctrl.Request{NamespacedName: client.ObjectKey{Name: "realm"}})
	require.NoError(t, err)
	assert.Equal(t, backoff, res.RequeueAfter)

	got := &keycloakAlpha.ClusterKeycloakRealm{}
	require.NoError(t, cl.Get(ctx, client.ObjectKey{Name: "realm"}, got))
	assert.False(t, got.Status.Available)
	assert.Equal(t, int64(3), got.Status.FailureCount)
	assert.Contains(t, got.Status.Value, "403")
}

// SetupWithManager filters ClusterKeycloakRealm updates with helper.IsFailuresUpdated.
func TestIsFailuresUpdated_ClusterKeycloakRealm(t *testing.T) {
	t.Parallel()

	base := func() *keycloakAlpha.ClusterKeycloakRealm {
		return &keycloakAlpha.ClusterKeycloakRealm{
			ObjectMeta: metav1.ObjectMeta{Name: "realm", Generation: 1},
			Spec:       keycloakAlpha.ClusterKeycloakRealmSpec{RealmName: "realm"},
			Status:     keycloakAlpha.ClusterKeycloakRealmStatus{FailureCount: 2},
		}
	}

	tests := []struct {
		name   string
		mutate func(*keycloakAlpha.ClusterKeycloakRealm)
		want   bool
	}{
		{
			name:   "failure status write is filtered",
			mutate: func(r *keycloakAlpha.ClusterKeycloakRealm) { r.Status.FailureCount = 3 },
			want:   false,
		},
		{
			name: "spec change is reconciled",
			mutate: func(r *keycloakAlpha.ClusterKeycloakRealm) {
				r.Generation = 2
				r.Spec.DisplayName = ptr.To("new")
			},
			want: true,
		},
		{
			name: "deletion is reconciled",
			mutate: func(r *keycloakAlpha.ClusterKeycloakRealm) {
				r.DeletionTimestamp = &metav1.Time{Time: time.Now()}
			},
			want: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			updated := base()
			tt.mutate(updated)

			assert.Equal(t, tt.want, helper.IsFailuresUpdated(event.UpdateEvent{ObjectOld: base(), ObjectNew: updated}))
		})
	}
}

func TestReconcile_PreChainErrorWritesStatus(t *testing.T) {
	t.Parallel()

	const (
		backoff = 30 * time.Second
		oldFC   = int64(2)
	)

	scheme := runtime.NewScheme()
	require.NoError(t, keycloakAlpha.AddToScheme(scheme))

	key := client.ObjectKey{Name: "realm"}

	tests := []struct {
		name         string
		setup        func(h *helpermock.MockControllerHelper)
		statusFails  bool
		deleting     bool
		wantValue    string
		wantRequeue  time.Duration
		wantErrMsg   string
		wantWrites   int
		wantNoStatus bool
	}{
		{
			name: "owner ref error",
			setup: func(h *helpermock.MockControllerHelper) {
				h.EXPECT().SetKeycloakOwnerRef(mock.Anything, mock.Anything).Return(errors.New("clusterkeycloak missing"))
			},
			wantValue:   "unable to set keycloak owner ref: clusterkeycloak missing",
			wantRequeue: backoff,
			wantWrites:  1,
		},
		{
			name: "client creation error",
			setup: func(h *helpermock.MockControllerHelper) {
				h.EXPECT().SetKeycloakOwnerRef(mock.Anything, mock.Anything).Return(nil)
				h.EXPECT().CreateKeycloakClientFromClusterRealm(mock.Anything, mock.Anything).
					Return(nil, errors.New("login failed"))
			},
			wantValue:   "failed to create keycloak v2 client for realm: login failed",
			wantRequeue: backoff,
			wantWrites:  1,
		},
		{
			name: "finalizer add error",
			setup: func(h *helpermock.MockControllerHelper) {
				h.EXPECT().SetKeycloakOwnerRef(mock.Anything, mock.Anything).Return(nil)
				h.EXPECT().CreateKeycloakClientFromClusterRealm(mock.Anything, mock.Anything).
					Return(&keycloakapi.KeycloakClient{}, nil)
				h.EXPECT().TryToDelete(mock.Anything, mock.Anything, mock.Anything, mock.Anything).
					Return(false, errors.New("failed to add finalizer: patch refused"))
			},
			wantValue:   "failed to add finalizer: patch refused",
			wantRequeue: backoff,
			wantWrites:  1,
		},
		{
			name: "try to delete error on a deleting realm",
			setup: func(h *helpermock.MockControllerHelper) {
				h.EXPECT().SetKeycloakOwnerRef(mock.Anything, mock.Anything).Return(nil)
				h.EXPECT().CreateKeycloakClientFromClusterRealm(mock.Anything, mock.Anything).
					Return(&keycloakapi.KeycloakClient{}, nil)
				h.EXPECT().TryToDelete(mock.Anything, mock.Anything, mock.Anything, mock.Anything).
					Return(false, errors.New("error during keycloak resource deletion: realm delete failed"))
			},
			deleting:    true,
			wantValue:   "error during keycloak resource deletion: realm delete failed",
			wantRequeue: backoff,
			wantWrites:  1,
		},
		{
			name: "keycloak not available requeues without status write",
			setup: func(h *helpermock.MockControllerHelper) {
				h.EXPECT().SetKeycloakOwnerRef(mock.Anything, mock.Anything).Return(nil)
				h.EXPECT().CreateKeycloakClientFromClusterRealm(mock.Anything, mock.Anything).
					Return(nil, fmt.Errorf("x: %w", helper.ErrKeycloakIsNotAvailable))
			},
			wantRequeue:  helper.RequeueOnKeycloakNotAvailablePeriod,
			wantNoStatus: true,
		},
		{
			name: "status write conflict returns the status error and stores nothing",
			setup: func(h *helpermock.MockControllerHelper) {
				h.EXPECT().SetKeycloakOwnerRef(mock.Anything, mock.Anything).Return(errors.New("patch failed"))
			},
			statusFails:  true,
			wantErrMsg:   "unable to update cluster realm status",
			wantWrites:   1,
			wantNoStatus: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			realm := &keycloakAlpha.ClusterKeycloakRealm{
				ObjectMeta: metav1.ObjectMeta{Name: key.Name},
				Spec:       keycloakAlpha.ClusterKeycloakRealmSpec{RealmName: "realm", ClusterKeycloakRef: "kc"},
				Status:     keycloakAlpha.ClusterKeycloakRealmStatus{Available: true, FailureCount: oldFC, Value: "OK"},
			}

			if tt.deleting {
				realm.Finalizers = []string{keyCloakRealmOperatorFinalizerName}
				realm.DeletionTimestamp = ptr.To(metav1.Now())
			}

			writes := 0

			cl := fake.NewClientBuilder().
				WithScheme(scheme).
				WithStatusSubresource(&keycloakAlpha.ClusterKeycloakRealm{}).
				WithObjects(realm).
				WithInterceptorFuncs(interceptor.Funcs{
					SubResourceUpdate: func(
						ctx context.Context, c client.Client, subResourceName string, obj client.Object, opts ...client.SubResourceUpdateOption,
					) error {
						writes++

						if tt.statusFails {
							return k8sErrors.NewConflict(keycloakAlpha.GroupVersion.WithResource("clusterkeycloakrealms").GroupResource(), obj.GetName(), errors.New("conflict"))
						}

						return c.SubResource(subResourceName).Update(ctx, obj, opts...)
					},
				}).
				Build()

			h := helpermock.NewMockControllerHelper(t)
			tt.setup(h)

			if tt.wantWrites > 0 {
				h.EXPECT().SetFailureCount(mock.Anything).RunAndReturn(func(fc helper.FailureCountable) time.Duration {
					fc.SetFailureCount(fc.GetFailureCount() + 1)

					return backoff
				})
			}

			ctx := ctrl.LoggerInto(context.Background(), logr.Discard())
			r := NewClusterKeycloakRealmReconciler(cl, scheme, h, "default", destination.AllowAll())

			res, err := r.Reconcile(ctx, ctrl.Request{NamespacedName: key})

			if tt.wantErrMsg != "" {
				require.ErrorContains(t, err, tt.wantErrMsg)
			} else {
				require.NoError(t, err)
			}

			assert.Equal(t, tt.wantRequeue, res.RequeueAfter)
			assert.Equal(t, tt.wantWrites, writes)

			got := &keycloakAlpha.ClusterKeycloakRealm{}
			require.NoError(t, cl.Get(ctx, key, got))

			if tt.wantNoStatus {
				assert.Equal(t, realm.Status, got.Status)

				return
			}

			assert.False(t, got.Status.Available)
			assert.Equal(t, tt.wantValue, got.Status.Value)
			assert.Equal(t, oldFC+1, got.Status.FailureCount)
		})
	}
}

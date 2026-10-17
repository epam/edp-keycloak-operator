package clusterkeycloakrealm

import (
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/go-logr/logr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/utils/ptr"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
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

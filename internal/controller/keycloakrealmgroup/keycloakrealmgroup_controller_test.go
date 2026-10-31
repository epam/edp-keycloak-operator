package keycloakrealmgroup

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
	"github.com/epam/edp-keycloak-operator/pkg/client/keycloakapi"
	v2mocks "github.com/epam/edp-keycloak-operator/pkg/client/keycloakapi/mocks"
)

func TestReconcileKeycloakRealmGroup_Reconcile(t *testing.T) {
	t.Parallel()

	const backoff = 30 * time.Second

	foreignFinalizer := "test/foreign"
	key := client.ObjectKey{Name: "group", Namespace: "default"}
	prevStatus := keycloakApi.KeycloakRealmGroupStatus{Value: "previous", ID: "group-id", FailureCount: 2}

	scheme := runtime.NewScheme()
	require.NoError(t, keycloakApi.AddToScheme(scheme))

	expectBackoff := func(h *helpermock.MockControllerHelper) {
		h.EXPECT().SetFailureCount(mock.Anything).RunAndReturn(func(fc helper.FailureCountable) time.Duration {
			fc.SetFailureCount(fc.GetFailureCount() + 1)

			return backoff
		})
	}

	expectClient := func(h *helpermock.MockControllerHelper, kClient *keycloakapi.KeycloakClient) {
		h.EXPECT().SetRealmOwnerRef(mock.Anything, mock.Anything).Return(nil)
		h.EXPECT().CreateKeycloakClientFromRealmRef(mock.Anything, mock.Anything).Return(kClient, nil)
		h.EXPECT().GetRealmNameFromRef(mock.Anything, mock.Anything).Return("realm", nil)
	}

	removeFinalizer := func(t *testing.T, cl client.Client, obj client.Object, finalizer string) error {
		t.Helper()

		original, ok := obj.DeepCopyObject().(client.Object)
		require.True(t, ok)

		controllerutil.RemoveFinalizer(obj, finalizer)

		return helper.PatchObject(t.Context(), cl, original, obj)
	}

	tests := []struct {
		name       string
		finalizers []string
		deleting   bool
		invalid    bool
		setup      func(t *testing.T, h *helpermock.MockControllerHelper, cl client.Client)

		wantRequeue   time.Duration
		wantWrites    int
		wantGone      bool
		wantUnchanged bool
		wantStatus    *keycloakApi.KeycloakRealmGroupStatus
		wantValue     string
	}{
		{
			name:       "deleting, only operator finalizer, finalizer removed",
			finalizers: []string{keyCloakRealmGroupOperatorFinalizerName},
			deleting:   true,
			setup: func(t *testing.T, h *helpermock.MockControllerHelper, cl client.Client) {
				expectClient(h, &keycloakapi.KeycloakClient{})
				h.EXPECT().TryToDelete(mock.Anything, mock.Anything, mock.Anything, keyCloakRealmGroupOperatorFinalizerName).
					RunAndReturn(func(_ context.Context, obj client.Object, _ helper.Terminator, finalizer string) (bool, error) {
						return true, removeFinalizer(t, cl, obj, finalizer)
					})
			},
			wantGone: true,
		},
		{
			name:       "deleting, foreign finalizer remains, status untouched",
			finalizers: []string{keyCloakRealmGroupOperatorFinalizerName, foreignFinalizer},
			deleting:   true,
			setup: func(t *testing.T, h *helpermock.MockControllerHelper, cl client.Client) {
				expectClient(h, &keycloakapi.KeycloakClient{})
				h.EXPECT().TryToDelete(mock.Anything, mock.Anything, mock.Anything, keyCloakRealmGroupOperatorFinalizerName).
					RunAndReturn(func(_ context.Context, obj client.Object, _ helper.Terminator, finalizer string) (bool, error) {
						return true, removeFinalizer(t, cl, obj, finalizer)
					})
			},
			wantUnchanged: true,
		},
		{
			name:       "deleting, realm gone, finalizer removed",
			finalizers: []string{keyCloakRealmGroupOperatorFinalizerName},
			deleting:   true,
			setup: func(t *testing.T, h *helpermock.MockControllerHelper, cl client.Client) {
				h.EXPECT().SetRealmOwnerRef(mock.Anything, mock.Anything).Return(nil)
				h.EXPECT().CreateKeycloakClientFromRealmRef(mock.Anything, mock.Anything).
					Return(nil, fmt.Errorf("x: %w", helper.ErrKeycloakRealmNotFound))
				h.EXPECT().TryRemoveFinalizer(mock.Anything, mock.Anything, keyCloakRealmGroupOperatorFinalizerName).
					RunAndReturn(func(_ context.Context, obj client.Object, finalizer string) error {
						return removeFinalizer(t, cl, obj, finalizer)
					})
			},
			wantGone: true,
		},
		{
			name:       "deleting, realm gone, finalizer removal fails, error status written",
			finalizers: []string{keyCloakRealmGroupOperatorFinalizerName},
			deleting:   true,
			setup: func(_ *testing.T, h *helpermock.MockControllerHelper, _ client.Client) {
				h.EXPECT().SetRealmOwnerRef(mock.Anything, mock.Anything).Return(nil)
				h.EXPECT().CreateKeycloakClientFromRealmRef(mock.Anything, mock.Anything).
					Return(nil, fmt.Errorf("x: %w", helper.ErrKeycloakRealmNotFound))
				h.EXPECT().TryRemoveFinalizer(mock.Anything, mock.Anything, keyCloakRealmGroupOperatorFinalizerName).
					Return(errors.New("patch failed"))
				expectBackoff(h)
			},
			wantRequeue: backoff,
			wantWrites:  1,
			wantValue:   "unable to remove finalizer: patch failed",
		},
		{
			name:       "live, realm gone, error status written",
			finalizers: []string{keyCloakRealmGroupOperatorFinalizerName},
			setup: func(_ *testing.T, h *helpermock.MockControllerHelper, _ client.Client) {
				h.EXPECT().SetRealmOwnerRef(mock.Anything, mock.Anything).Return(nil)
				h.EXPECT().CreateKeycloakClientFromRealmRef(mock.Anything, mock.Anything).
					Return(nil, fmt.Errorf("x: %w", helper.ErrKeycloakRealmNotFound))
				expectBackoff(h)
			},
			wantRequeue: backoff,
			wantWrites:  1,
			wantValue:   "unable to create keycloak v2 client from realm ref: x: " + helper.ErrKeycloakRealmNotFound.Error(),
		},
		{
			name:       "deleting, keycloak delete fails, error status written",
			finalizers: []string{keyCloakRealmGroupOperatorFinalizerName},
			deleting:   true,
			setup: func(_ *testing.T, h *helpermock.MockControllerHelper, _ client.Client) {
				expectClient(h, &keycloakapi.KeycloakClient{})
				h.EXPECT().TryToDelete(mock.Anything, mock.Anything, mock.Anything, keyCloakRealmGroupOperatorFinalizerName).
					Return(false, errors.New("delete failed"))
				expectBackoff(h)
			},
			wantRequeue: backoff,
			wantWrites:  1,
			wantValue:   "delete failed",
		},
		{
			name:       "keycloak unavailable, requeue without status write",
			finalizers: []string{keyCloakRealmGroupOperatorFinalizerName},
			deleting:   true,
			setup: func(_ *testing.T, h *helpermock.MockControllerHelper, _ client.Client) {
				h.EXPECT().SetRealmOwnerRef(mock.Anything, mock.Anything).Return(nil)
				h.EXPECT().CreateKeycloakClientFromRealmRef(mock.Anything, mock.Anything).
					Return(nil, fmt.Errorf("x: %w", helper.ErrKeycloakIsNotAvailable))
			},
			wantRequeue:   helper.RequeueOnKeycloakNotAvailablePeriod,
			wantUnchanged: true,
		},
		{
			name:       "deleting, invalid spec, error status written",
			finalizers: []string{keyCloakRealmGroupOperatorFinalizerName},
			deleting:   true,
			invalid:    true,
			setup: func(_ *testing.T, h *helpermock.MockControllerHelper, _ client.Client) {
				expectBackoff(h)
			},
			wantRequeue: backoff,
			wantWrites:  1,
			wantValue:   "cannot use both SubGroups (deprecated) and ParentGroup fields - migrate to ParentGroup approach",
		},
		{
			name:       "deleting, finalizer patch fails after in-memory removal, error status written",
			finalizers: []string{keyCloakRealmGroupOperatorFinalizerName},
			deleting:   true,
			setup: func(_ *testing.T, h *helpermock.MockControllerHelper, _ client.Client) {
				expectClient(h, &keycloakapi.KeycloakClient{})
				h.EXPECT().TryToDelete(mock.Anything, mock.Anything, mock.Anything, keyCloakRealmGroupOperatorFinalizerName).
					RunAndReturn(func(_ context.Context, obj client.Object, _ helper.Terminator, finalizer string) (bool, error) {
						controllerutil.RemoveFinalizer(obj, finalizer)

						return false, errors.New("finalizer patch failed")
					})
				expectBackoff(h)
			},
			wantRequeue: backoff,
			wantWrites:  1,
			wantValue:   "finalizer patch failed",
		},
		{
			name:       "live, finalizer add fails, error status written",
			finalizers: []string{keyCloakRealmGroupOperatorFinalizerName},
			setup: func(_ *testing.T, h *helpermock.MockControllerHelper, _ client.Client) {
				expectClient(h, &keycloakapi.KeycloakClient{})
				h.EXPECT().TryToDelete(mock.Anything, mock.Anything, mock.Anything, keyCloakRealmGroupOperatorFinalizerName).
					Return(false, errors.New("failed to add finalizer: patch refused"))
				expectBackoff(h)
			},
			wantRequeue: backoff,
			wantWrites:  1,
			wantValue:   "failed to add finalizer: patch refused",
		},
		{
			name:       "live, chain ok, success status written",
			finalizers: []string{keyCloakRealmGroupOperatorFinalizerName},
			setup: func(t *testing.T, h *helpermock.MockControllerHelper, _ client.Client) {
				groups := v2mocks.NewMockGroupsClient(t)
				groups.EXPECT().GetGroup(mock.Anything, "realm", "group-id").
					Return(&keycloakapi.GroupRepresentation{Id: ptr.To("group-id")}, nil, nil)
				groups.EXPECT().UpdateGroup(mock.Anything, "realm", "group-id", mock.Anything).Return(nil, nil)

				expectClient(h, &keycloakapi.KeycloakClient{Groups: groups})
				h.EXPECT().TryToDelete(mock.Anything, mock.Anything, mock.Anything, keyCloakRealmGroupOperatorFinalizerName).
					Return(false, nil)
			},
			wantWrites: 1,
			wantStatus: &keycloakApi.KeycloakRealmGroupStatus{Value: common.StatusOK, ID: "group-id"},
		},
		{
			name:       "live, owner ref error, error status written",
			finalizers: []string{keyCloakRealmGroupOperatorFinalizerName},
			setup: func(_ *testing.T, h *helpermock.MockControllerHelper, _ client.Client) {
				h.EXPECT().SetRealmOwnerRef(mock.Anything, mock.Anything).Return(errors.New("realm missing"))
				expectBackoff(h)
			},
			wantRequeue: backoff,
			wantWrites:  1,
			wantValue:   "unable to set realm owner ref: realm missing",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			group := &keycloakApi.KeycloakRealmGroup{
				ObjectMeta: metav1.ObjectMeta{
					Name:       key.Name,
					Namespace:  key.Namespace,
					Finalizers: tt.finalizers,
				},
				Spec: keycloakApi.KeycloakRealmGroupSpec{
					Name:     "group",
					RealmRef: common.RealmRef{Kind: "KeycloakRealm", Name: "realm"},
				},
				Status: prevStatus,
			}

			if tt.deleting {
				group.DeletionTimestamp = ptr.To(metav1.Now())
			}

			if tt.invalid {
				group.Spec.SubGroups = []string{"child"}
				group.Spec.ParentGroup = &common.GroupRef{Name: "parent"}
			}

			writes := 0

			cl := fake.NewClientBuilder().
				WithScheme(scheme).
				WithStatusSubresource(&keycloakApi.KeycloakRealmGroup{}).
				WithObjects(group).
				WithInterceptorFuncs(interceptor.Funcs{
					SubResourceUpdate: func(
						ctx context.Context, c client.Client, subResourceName string, obj client.Object, opts ...client.SubResourceUpdateOption,
					) error {
						writes++

						return c.SubResource(subResourceName).Update(ctx, obj, opts...)
					},
				}).
				Build()

			h := helpermock.NewMockControllerHelper(t)

			tt.setup(t, h, cl)

			r := NewReconcileKeycloakRealmGroup(cl, h)
			reqCtx := ctrl.LoggerInto(t.Context(), logr.Discard())

			res, err := r.Reconcile(reqCtx, ctrl.Request{NamespacedName: key})
			require.NoError(t, err)

			assert.Equal(t, tt.wantRequeue, res.RequeueAfter)
			assert.Equal(t, tt.wantWrites, writes)

			got := &keycloakApi.KeycloakRealmGroup{}
			getErr := cl.Get(reqCtx, key, got)

			if tt.wantGone {
				require.True(t, k8sErrors.IsNotFound(getErr), "expected NotFound, got %v", getErr)

				return
			}

			require.NoError(t, getErr)

			if tt.wantUnchanged {
				assert.Equal(t, prevStatus, got.Status)

				return
			}

			if tt.wantStatus != nil {
				assert.Equal(t, *tt.wantStatus, got.Status)

				return
			}

			assert.Equal(t, tt.wantValue, got.Status.Value)
			assert.Equal(t, prevStatus.FailureCount+1, got.Status.FailureCount)
			assert.Equal(t, prevStatus.ID, got.Status.ID)
		})
	}
}

package keycloakrealmrole

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
	"github.com/epam/edp-keycloak-operator/pkg/objectmeta"
)

func TestReconcileKeycloakRealmRole_Reconcile(t *testing.T) {
	t.Parallel()

	const backoff = 30 * time.Second

	foreignFinalizer := "test/foreign"
	key := client.ObjectKey{Name: "role", Namespace: "default"}
	prevStatus := keycloakApi.KeycloakRealmRoleStatus{Value: "previous", ID: "role-id", FailureCount: 2}

	scheme := runtime.NewScheme()
	require.NoError(t, keycloakApi.AddToScheme(scheme))

	expectBackoff := func(h *helpermock.MockControllerHelper) {
		h.EXPECT().SetFailureCount(mock.Anything).RunAndReturn(func(fc helper.FailureCountable) time.Duration {
			fc.SetFailureCount(fc.GetFailureCount() + 1)

			return backoff
		})
	}

	expectClient := func(h *helpermock.MockControllerHelper, roles *v2mocks.MockRolesClient) {
		h.EXPECT().SetRealmOwnerRef(mock.Anything, mock.Anything).Return(nil)
		h.EXPECT().CreateKeycloakClientFromRealmRef(mock.Anything, mock.Anything).
			Return(&keycloakapi.KeycloakClient{Roles: roles}, nil)
		h.EXPECT().GetRealmNameFromRef(mock.Anything, mock.Anything).Return("realm", nil)
	}

	expectRealmGone := func(h *helpermock.MockControllerHelper) {
		h.EXPECT().SetRealmOwnerRef(mock.Anything, mock.Anything).Return(nil)
		h.EXPECT().CreateKeycloakClientFromRealmRef(mock.Anything, mock.Anything).
			Return(nil, fmt.Errorf("x: %w", helper.ErrKeycloakRealmNotFound))
	}

	tests := []struct {
		name       string
		finalizers []string
		deleting   bool
		preserve   bool
		patchFails bool
		setup      func(t *testing.T, h *helpermock.MockControllerHelper, rolesClient *v2mocks.MockRolesClient, cl client.Client)

		wantRequeue   time.Duration
		wantWrites    int
		wantGone      bool
		wantUnchanged bool
		wantStatus    *keycloakApi.KeycloakRealmRoleStatus
		wantValue     string
	}{
		{
			name:       "deleting, only operator finalizer, role removed",
			finalizers: []string{keyCloakRealmRoleOperatorFinalizerName},
			deleting:   true,
			setup: func(_ *testing.T, h *helpermock.MockControllerHelper, rolesClient *v2mocks.MockRolesClient, _ client.Client) {
				expectClient(h, rolesClient)
				rolesClient.EXPECT().DeleteRealmRole(mock.Anything, "realm", "role").Return(nil, nil)
			},
			wantGone: true,
		},
		{
			name:       "deleting with preserve annotation, keycloak untouched",
			finalizers: []string{keyCloakRealmRoleOperatorFinalizerName},
			deleting:   true,
			preserve:   true,
			setup: func(_ *testing.T, h *helpermock.MockControllerHelper, rolesClient *v2mocks.MockRolesClient, _ client.Client) {
				expectClient(h, rolesClient)
			},
			wantGone: true,
		},
		{
			name:       "deleting, foreign finalizer remains, status and id kept",
			finalizers: []string{keyCloakRealmRoleOperatorFinalizerName, foreignFinalizer},
			deleting:   true,
			setup: func(_ *testing.T, h *helpermock.MockControllerHelper, rolesClient *v2mocks.MockRolesClient, _ client.Client) {
				expectClient(h, rolesClient)
				rolesClient.EXPECT().DeleteRealmRole(mock.Anything, "realm", "role").Return(nil, nil)
			},
			wantUnchanged: true,
		},
		{
			name:       "deleting, only foreign finalizer, status and id kept",
			finalizers: []string{foreignFinalizer},
			deleting:   true,
			setup: func(_ *testing.T, h *helpermock.MockControllerHelper, rolesClient *v2mocks.MockRolesClient, _ client.Client) {
				expectClient(h, rolesClient)
			},
			wantUnchanged: true,
		},
		{
			name:       "deleting, realm gone, finalizer removed",
			finalizers: []string{keyCloakRealmRoleOperatorFinalizerName},
			deleting:   true,
			setup: func(t *testing.T, h *helpermock.MockControllerHelper, _ *v2mocks.MockRolesClient, cl client.Client) {
				expectRealmGone(h)
				h.EXPECT().TryRemoveFinalizer(mock.Anything, mock.Anything, keyCloakRealmRoleOperatorFinalizerName).
					RunAndReturn(func(ctx context.Context, obj client.Object, finalizer string) error {
						original, ok := obj.DeepCopyObject().(client.Object)
						require.True(t, ok)

						controllerutil.RemoveFinalizer(obj, finalizer)

						return helper.PatchObject(ctx, cl, original, obj)
					})
			},
			wantGone: true,
		},
		{
			name:       "live, realm gone, error status written",
			finalizers: []string{keyCloakRealmRoleOperatorFinalizerName},
			setup: func(_ *testing.T, h *helpermock.MockControllerHelper, _ *v2mocks.MockRolesClient, _ client.Client) {
				expectRealmGone(h)
				expectBackoff(h)
			},
			wantRequeue: backoff,
			wantWrites:  1,
			wantValue:   "unable to create keycloak client from realm ref: x: " + helper.ErrKeycloakRealmNotFound.Error(),
		},
		{
			name:       "deleting, keycloak delete fails, error status written",
			finalizers: []string{keyCloakRealmRoleOperatorFinalizerName},
			deleting:   true,
			setup: func(_ *testing.T, h *helpermock.MockControllerHelper, rolesClient *v2mocks.MockRolesClient, _ client.Client) {
				expectClient(h, rolesClient)
				rolesClient.EXPECT().DeleteRealmRole(mock.Anything, "realm", "role").Return(nil, errors.New("delete failed"))
				expectBackoff(h)
			},
			wantRequeue: backoff,
			wantWrites:  1,
			wantValue:   "failed to remove role: failed to delete realm role role: delete failed",
		},
		{
			name:       "deleting, finalizer patch fails, error status written",
			finalizers: []string{keyCloakRealmRoleOperatorFinalizerName},
			deleting:   true,
			patchFails: true,
			setup: func(_ *testing.T, h *helpermock.MockControllerHelper, rolesClient *v2mocks.MockRolesClient, _ client.Client) {
				expectClient(h, rolesClient)
				rolesClient.EXPECT().DeleteRealmRole(mock.Anything, "realm", "role").Return(nil, nil)
				expectBackoff(h)
			},
			wantRequeue: backoff,
			wantWrites:  1,
			wantValue:   "failed to remove finalizer: unable to patch object: ",
		},
		{
			name:       "keycloak unavailable, requeue with unchanged status",
			finalizers: []string{keyCloakRealmRoleOperatorFinalizerName},
			deleting:   true,
			setup: func(_ *testing.T, h *helpermock.MockControllerHelper, _ *v2mocks.MockRolesClient, _ client.Client) {
				h.EXPECT().SetRealmOwnerRef(mock.Anything, mock.Anything).Return(nil)
				h.EXPECT().CreateKeycloakClientFromRealmRef(mock.Anything, mock.Anything).
					Return(nil, fmt.Errorf("x: %w", helper.ErrKeycloakIsNotAvailable))
			},
			wantRequeue:   helper.RequeueOnKeycloakNotAvailablePeriod,
			wantWrites:    1,
			wantUnchanged: true,
		},
		{
			name:       "live, chain ok, success status and id written",
			finalizers: []string{keyCloakRealmRoleOperatorFinalizerName},
			setup: func(_ *testing.T, h *helpermock.MockControllerHelper, rolesClient *v2mocks.MockRolesClient, _ client.Client) {
				expectClient(h, rolesClient)
				rolesClient.EXPECT().GetRealmRole(mock.Anything, "realm", "role").
					Return(&keycloakapi.RoleRepresentation{Id: ptr.To("new-role-id"), Name: ptr.To("role")}, nil, nil)
				rolesClient.EXPECT().UpdateRealmRole(mock.Anything, "realm", "role", mock.Anything).Return(nil, nil)
			},
			wantWrites: 1,
			wantStatus: &keycloakApi.KeycloakRealmRoleStatus{Value: common.StatusOK, ID: "new-role-id"},
		},
		{
			name:       "live, owner ref error, error status written",
			finalizers: []string{keyCloakRealmRoleOperatorFinalizerName},
			setup: func(_ *testing.T, h *helpermock.MockControllerHelper, _ *v2mocks.MockRolesClient, _ client.Client) {
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

			role := &keycloakApi.KeycloakRealmRole{
				ObjectMeta: metav1.ObjectMeta{
					Name:       key.Name,
					Namespace:  key.Namespace,
					Finalizers: tt.finalizers,
				},
				Spec: keycloakApi.KeycloakRealmRoleSpec{
					Name:     "role",
					RealmRef: common.RealmRef{Kind: "KeycloakRealm", Name: "realm"},
				},
				Status: prevStatus,
			}

			if tt.deleting {
				role.DeletionTimestamp = ptr.To(metav1.Now())
			}

			if tt.preserve {
				role.Annotations = map[string]string{objectmeta.PreserveResourcesOnDeletionAnnotation: "true"}
			}

			writes := 0

			cl := fake.NewClientBuilder().
				WithScheme(scheme).
				WithStatusSubresource(&keycloakApi.KeycloakRealmRole{}).
				WithObjects(role).
				WithInterceptorFuncs(interceptor.Funcs{
					SubResourceUpdate: func(
						ctx context.Context, c client.Client, subResourceName string, obj client.Object, opts ...client.SubResourceUpdateOption,
					) error {
						writes++

						return c.SubResource(subResourceName).Update(ctx, obj, opts...)
					},
					Patch: func(
						ctx context.Context, c client.WithWatch, obj client.Object, patch client.Patch, opts ...client.PatchOption,
					) error {
						if tt.patchFails {
							return k8sErrors.NewConflict(keycloakApi.GroupVersion.WithResource("keycloakrealmroles").GroupResource(), obj.GetName(), errors.New("conflict"))
						}

						return c.Patch(ctx, obj, patch, opts...)
					},
				}).
				Build()

			rolesClient := v2mocks.NewMockRolesClient(t)
			h := helpermock.NewMockControllerHelper(t)

			tt.setup(t, h, rolesClient, cl)

			r := NewReconcileKeycloakRealmRole(cl, h)
			reqCtx := ctrl.LoggerInto(t.Context(), logr.Discard())

			res, err := r.Reconcile(reqCtx, ctrl.Request{NamespacedName: key})
			require.NoError(t, err)

			assert.Equal(t, tt.wantRequeue, res.RequeueAfter)
			assert.Equal(t, tt.wantWrites, writes)

			got := &keycloakApi.KeycloakRealmRole{}
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

			assert.Contains(t, got.Status.Value, tt.wantValue)
			assert.Equal(t, prevStatus.FailureCount+1, got.Status.FailureCount)
			assert.Equal(t, prevStatus.ID, got.Status.ID)
		})
	}
}

package keycloakrealmidentityprovider

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
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/client/interceptor"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	"github.com/epam/edp-keycloak-operator/api/common"
	keycloakApi "github.com/epam/edp-keycloak-operator/api/v1"
	"github.com/epam/edp-keycloak-operator/internal/controller/helper"
	helpermock "github.com/epam/edp-keycloak-operator/internal/controller/helper/mocks"
	"github.com/epam/edp-keycloak-operator/pkg/client/keycloakapi"
	keycloakapimocks "github.com/epam/edp-keycloak-operator/pkg/client/keycloakapi/mocks"
)

func TestReconcile_PreChainErrorWritesStatus(t *testing.T) {
	t.Parallel()

	scheme := runtime.NewScheme()
	require.NoError(t, keycloakApi.AddToScheme(scheme))

	key := client.ObjectKey{Namespace: "default", Name: "obj"}

	failPatch := func(context.Context, client.WithWatch, client.Object, client.Patch, ...client.PatchOption) error {
		return k8sErrors.NewInternalError(errors.New("patch denied"))
	}

	realmGone := func(h *helpermock.MockControllerHelper) {
		h.EXPECT().SetRealmOwnerRef(mock.Anything, mock.Anything).Return(nil)
		h.EXPECT().CreateKeycloakClientFromRealmRef(mock.Anything, mock.Anything).
			Return(nil, helper.ErrKeycloakRealmNotFound)
	}

	clientOK := func(h *helpermock.MockControllerHelper) {
		h.EXPECT().SetRealmOwnerRef(mock.Anything, mock.Anything).Return(nil)
		h.EXPECT().CreateKeycloakClientFromRealmRef(mock.Anything, mock.Anything).
			Return(&keycloakapi.KeycloakClient{}, nil)
	}

	tests := []struct {
		name        string
		deleting    bool
		setup       func(h *helpermock.MockControllerHelper)
		patch       func(context.Context, client.WithWatch, client.Object, client.Patch, ...client.PatchOption) error
		statusFails bool
		runs        int
		wantValue   string
		wantErrMsgs []string
		wantRequeue time.Duration
		wantWrites  int
		wantGone    bool
	}{
		{
			name: "owner ref error",
			setup: func(h *helpermock.MockControllerHelper) {
				h.EXPECT().SetRealmOwnerRef(mock.Anything, mock.Anything).Return(errors.New("realm get failed"))
			},
			wantValue:   "unable to set realm owner ref: realm get failed",
			wantErrMsgs: []string{"unable to set realm owner ref: realm get failed"},
			wantWrites:  1,
		},
		{
			name: "client creation error",
			setup: func(h *helpermock.MockControllerHelper) {
				h.EXPECT().SetRealmOwnerRef(mock.Anything, mock.Anything).Return(nil)
				h.EXPECT().CreateKeycloakClientFromRealmRef(mock.Anything, mock.Anything).
					Return(nil, errors.New("unable to get realm"))
			},
			wantValue:   "failed to create Keycloak client: unable to get realm",
			wantErrMsgs: []string{"failed to create Keycloak client: unable to get realm"},
			wantWrites:  1,
		},
		{
			name: "realm name error",
			setup: func(h *helpermock.MockControllerHelper) {
				clientOK(h)
				h.EXPECT().GetRealmNameFromRef(mock.Anything, mock.Anything).Return("", errors.New("no realm name"))
			},
			wantValue:   "unable to get realm name from ref: no realm name",
			wantErrMsgs: []string{"unable to get realm name from ref: no realm name"},
			wantWrites:  1,
		},
		{
			name: "keycloak not available requeues without status write",
			setup: func(h *helpermock.MockControllerHelper) {
				h.EXPECT().SetRealmOwnerRef(mock.Anything, mock.Anything).Return(nil)
				h.EXPECT().CreateKeycloakClientFromRealmRef(mock.Anything, mock.Anything).
					Return(nil, fmt.Errorf("x: %w", helper.ErrKeycloakIsNotAvailable))
			},
			wantValue:   "OK",
			wantRequeue: helper.RequeueOnKeycloakNotAvailablePeriod,
		},
		{
			name: "finalizer add error",
			setup: func(h *helpermock.MockControllerHelper) {
				clientOK(h)
				h.EXPECT().GetRealmNameFromRef(mock.Anything, mock.Anything).Return("realm", nil)
			},
			patch:       failPatch,
			wantValue:   "failed to add finalizer to",
			wantErrMsgs: []string{"failed to add finalizer to"},
			wantWrites:  1,
		},
		{
			name:        "deleting, realm gone, finalizer removal error",
			deleting:    true,
			setup:       realmGone,
			patch:       failPatch,
			wantValue:   "failed to remove finalizer",
			wantErrMsgs: []string{"failed to remove finalizer"},
			wantWrites:  1,
		},
		{
			name:     "deleting, realm gone, finalizer removed stops without status write",
			deleting: true,
			setup:    realmGone,
			wantGone: true,
		},
		{
			name: "same error twice writes once",
			setup: func(h *helpermock.MockControllerHelper) {
				h.EXPECT().SetRealmOwnerRef(mock.Anything, mock.Anything).Return(nil)
				h.EXPECT().CreateKeycloakClientFromRealmRef(mock.Anything, mock.Anything).
					Return(nil, errors.New("unable to get realm"))
			},
			runs:        2,
			wantValue:   "failed to create Keycloak client: unable to get realm",
			wantErrMsgs: []string{"failed to create Keycloak client: unable to get realm"},
			wantWrites:  1,
		},
		{
			name: "status write conflict returns the status error and stores nothing",
			setup: func(h *helpermock.MockControllerHelper) {
				h.EXPECT().SetRealmOwnerRef(mock.Anything, mock.Anything).Return(nil)
				h.EXPECT().CreateKeycloakClientFromRealmRef(mock.Anything, mock.Anything).
					Return(nil, errors.New("unable to get realm"))
			},
			statusFails: true,
			wantValue:   "OK",
			wantErrMsgs: []string{"failed to update KeycloakRealmIdentityProvider status"},
			wantWrites:  1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			obj := &keycloakApi.KeycloakRealmIdentityProvider{
				ObjectMeta: metav1.ObjectMeta{Namespace: key.Namespace, Name: key.Name, Generation: 2},
				Status:     keycloakApi.KeycloakRealmIdentityProviderStatus{Value: common.StatusOK},
			}

			if tt.deleting {
				now := metav1.Now()
				obj.DeletionTimestamp = &now
				obj.Finalizers = []string{common.FinalizerName}
			}

			writes := 0

			cl := fake.NewClientBuilder().
				WithScheme(scheme).
				WithStatusSubresource(&keycloakApi.KeycloakRealmIdentityProvider{}).
				WithObjects(obj).
				WithInterceptorFuncs(interceptor.Funcs{
					Patch: tt.patch,
					SubResourceUpdate: func(
						ctx context.Context, c client.Client, subResourceName string, o client.Object, opts ...client.SubResourceUpdateOption,
					) error {
						writes++

						if tt.statusFails {
							return k8sErrors.NewConflict(keycloakApi.GroupVersion.WithResource("keycloakrealmidentityproviders").GroupResource(), o.GetName(), errors.New("conflict"))
						}

						return c.SubResource(subResourceName).Update(ctx, o, opts...)
					},
				}).
				Build()

			h := helpermock.NewMockControllerHelper(t)
			tt.setup(h)

			ctx := ctrl.LoggerInto(context.Background(), logr.Discard())
			r := NewIdentityProviderReconciler(cl, h, nil)

			runs := max(tt.runs, 1)

			var (
				res reconcile.Result
				err error
			)

			for range runs {
				res, err = r.Reconcile(ctx, ctrl.Request{NamespacedName: key})

				if len(tt.wantErrMsgs) == 0 {
					require.NoError(t, err)
				}

				for _, msg := range tt.wantErrMsgs {
					require.ErrorContains(t, err, msg)
				}
			}

			assert.Equal(t, tt.wantRequeue, res.RequeueAfter)
			assert.Equal(t, tt.wantWrites, writes)

			got := &keycloakApi.KeycloakRealmIdentityProvider{}
			getErr := cl.Get(ctx, key, got)

			if tt.wantGone {
				require.True(t, k8sErrors.IsNotFound(getErr))

				return
			}

			require.NoError(t, getErr)
			assert.Contains(t, got.Status.Value, tt.wantValue)
			assert.Zero(t, got.Status.FailureCount)
		})
	}
}

func TestReconcile_KeycloakDeleteErrorWritesStatus(t *testing.T) {
	t.Parallel()

	scheme := runtime.NewScheme()
	require.NoError(t, keycloakApi.AddToScheme(scheme))

	key := client.ObjectKey{Namespace: "default", Name: "obj"}
	now := metav1.Now()

	obj := &keycloakApi.KeycloakRealmIdentityProvider{
		ObjectMeta: metav1.ObjectMeta{
			Namespace:         key.Namespace,
			Name:              key.Name,
			Generation:        2,
			DeletionTimestamp: &now,
			Finalizers:        []string{common.FinalizerName},
		},
		Status: keycloakApi.KeycloakRealmIdentityProviderStatus{Value: common.StatusOK},
	}
	obj.Spec.Alias = "idp"

	cl := fake.NewClientBuilder().
		WithScheme(scheme).
		WithStatusSubresource(&keycloakApi.KeycloakRealmIdentityProvider{}).
		WithObjects(obj).
		Build()

	idps := keycloakapimocks.NewMockIdentityProvidersClient(t)
	idps.EXPECT().DeleteIdentityProvider(mock.Anything, "realm", "idp").Return(nil, errors.New("delete refused"))

	kc := &keycloakapi.KeycloakClient{IdentityProviders: idps}

	h := helpermock.NewMockControllerHelper(t)
	h.EXPECT().SetRealmOwnerRef(mock.Anything, mock.Anything).Return(nil)
	h.EXPECT().CreateKeycloakClientFromRealmRef(mock.Anything, mock.Anything).Return(kc, nil)
	h.EXPECT().GetRealmNameFromRef(mock.Anything, mock.Anything).Return("realm", nil)

	ctx := ctrl.LoggerInto(context.Background(), logr.Discard())
	r := NewIdentityProviderReconciler(cl, h, nil)

	_, err := r.Reconcile(ctx, ctrl.Request{NamespacedName: key})
	require.ErrorContains(t, err, "failed to remove identity provider: failed to delete identity provider: delete refused")

	got := &keycloakApi.KeycloakRealmIdentityProvider{}
	require.NoError(t, cl.Get(ctx, key, got))
	assert.Equal(t, "failed to remove identity provider: failed to delete identity provider: delete refused", got.Status.Value)
	assert.Contains(t, got.Finalizers, common.FinalizerName)
}

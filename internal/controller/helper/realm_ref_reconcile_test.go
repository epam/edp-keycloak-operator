package helper

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/go-logr/logr"
	"github.com/stretchr/testify/assert"
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
	"github.com/epam/edp-keycloak-operator/pkg/client/keycloakapi"
)

const (
	testKind         = "KeycloakClientScope"
	testLegacyFinal  = "legacy.finalizer"
	testForeignFinal = "foreign.finalizer"
	testFakeRealm    = "realm"
)

type fakeRealmRefHelper struct {
	ownerRefErr error
	clientErr   error
	realmErr    error
	kClient     *keycloakapi.KeycloakClient
}

func (f fakeRealmRefHelper) SetRealmOwnerRef(context.Context, ObjectWithRealmRef) error {
	return f.ownerRefErr
}

func (f fakeRealmRefHelper) CreateKeycloakClientFromRealmRef(context.Context, ObjectWithRealmRef) (*keycloakapi.KeycloakClient, error) {
	if f.clientErr != nil {
		return nil, f.clientErr
	}

	return f.kClient, nil
}

func (f fakeRealmRefHelper) GetRealmNameFromRef(context.Context, ObjectWithRealmRef) (string, error) {
	if f.realmErr != nil {
		return "", f.realmErr
	}

	return testFakeRealm, nil
}

func TestReconcileRealmRef(t *testing.T) {
	t.Parallel()

	scheme := runtime.NewScheme()
	require.NoError(t, keycloakApi.AddToScheme(scheme))

	key := client.ObjectKey{Namespace: "default", Name: "scope"}
	failPatch := func(context.Context, client.WithWatch, client.Object, client.Patch, ...client.PatchOption) error {
		return k8sErrors.NewInternalError(errors.New("patch denied"))
	}

	errNoRealm := errors.New("unable to get realm")

	tests := []struct {
		name         string
		wantErrIs    error
		noObject     bool
		finalizers   []string
		deleting     bool
		helper       fakeRealmRefHelper
		secondHelper *fakeRealmRefHelper
		checkArgs    bool
		getErr       error
		removeErr    error
		patch        func(context.Context, client.WithWatch, client.Object, client.Patch, ...client.PatchOption) error
		statusFails  bool
		runs         int
		wantErrMsgs  []string
		wantRequeue  bool
		wantValue    string
		wantWrites   int
		wantRemove   int
		wantHandle   int
		wantGone     bool
		wantFinalize []string
	}{
		{
			name:     "not found stops silently",
			noObject: true,
		},
		{
			name:        "get error is returned without status write",
			getErr:      errors.New("api down"),
			wantErrMsgs: []string{"failed to get " + testKind, "api down"},
			wantValue:   common.StatusOK,
		},
		{
			name:        "keycloak not available requeues without status write",
			helper:      fakeRealmRefHelper{clientErr: fmt.Errorf("x: %w", ErrKeycloakIsNotAvailable)},
			wantRequeue: true,
			wantValue:   common.StatusOK,
		},
		{
			name:        "owner ref error is written",
			helper:      fakeRealmRefHelper{ownerRefErr: errors.New("realm get failed")},
			wantErrMsgs: []string{"unable to set realm owner ref: realm get failed"},
			wantValue:   "unable to set realm owner ref: realm get failed",
			wantWrites:  1,
		},
		{
			name:        "client creation error is written",
			helper:      fakeRealmRefHelper{clientErr: errors.New("unable to get realm")},
			wantErrMsgs: []string{"failed to create Keycloak client: unable to get realm"},
			wantValue:   "failed to create Keycloak client: unable to get realm",
			wantWrites:  1,
		},
		{
			name:        "realm name error is written",
			helper:      fakeRealmRefHelper{realmErr: errors.New("no realm name")},
			wantErrMsgs: []string{"unable to get realm name from ref: no realm name"},
			wantValue:   "unable to get realm name from ref: no realm name",
			wantWrites:  1,
		},
		{
			name:        "same error twice writes once",
			helper:      fakeRealmRefHelper{clientErr: errors.New("unable to get realm")},
			runs:        2,
			wantErrMsgs: []string{"failed to create Keycloak client: unable to get realm"},
			wantValue:   "failed to create Keycloak client: unable to get realm",
			wantWrites:  1,
		},
		{
			name:         "changed error on the second run is written again",
			helper:       fakeRealmRefHelper{ownerRefErr: errors.New("realm get failed")},
			secondHelper: &fakeRealmRefHelper{clientErr: errors.New("unable to get realm")},
			runs:         2,
			wantErrMsgs:  []string{"failed to create Keycloak client: unable to get realm"},
			wantValue:    "failed to create Keycloak client: unable to get realm",
			wantWrites:   2,
		},
		{
			name:        "status write conflict carries both errors and stores nothing",
			helper:      fakeRealmRefHelper{clientErr: errNoRealm},
			statusFails: true,
			wantErrIs:   errNoRealm,
			wantErrMsgs: []string{"failed to update " + testKind + " status (", "unable to get realm", "conflict"},
			wantValue:   common.StatusOK,
			wantWrites:  1,
		},
		{
			name:         "realm gone on a live object is written and finalizers kept",
			finalizers:   []string{common.FinalizerName, testLegacyFinal},
			helper:       fakeRealmRefHelper{clientErr: ErrKeycloakRealmNotFound},
			wantErrMsgs:  []string{"failed to create Keycloak client", ErrKeycloakRealmNotFound.Error()},
			wantValue:    "failed to create Keycloak client",
			wantWrites:   1,
			wantFinalize: []string{common.FinalizerName, testLegacyFinal},
		},
		{
			name:       "realm gone while deleting removes finalizers and stops",
			deleting:   true,
			finalizers: []string{common.FinalizerName, testLegacyFinal},
			helper:     fakeRealmRefHelper{clientErr: ErrKeycloakRealmNotFound},
			wantGone:   true,
		},
		{
			name:        "realm gone while deleting, finalizer removal fails",
			deleting:    true,
			finalizers:  []string{common.FinalizerName},
			helper:      fakeRealmRefHelper{clientErr: ErrKeycloakRealmNotFound},
			patch:       failPatch,
			wantErrMsgs: []string{"failed to remove finalizers"},
			wantValue:   "failed to remove finalizers",
			wantWrites:  1,
		},
		{
			name:       "deleting removes from keycloak and drops all finalizers",
			deleting:   true,
			finalizers: []string{common.FinalizerName, testLegacyFinal},
			wantRemove: 1,
			wantGone:   true,
			checkArgs:  true,
		},
		{
			name:         "deleting, remove error is written and finalizer kept",
			deleting:     true,
			finalizers:   []string{common.FinalizerName},
			removeErr:    errors.New("failed to remove client scope: x"),
			wantErrMsgs:  []string{"failed to remove client scope: x"},
			wantValue:    "failed to remove client scope: x",
			wantWrites:   1,
			wantRemove:   1,
			wantFinalize: []string{common.FinalizerName},
		},
		{
			name:         "deleting, finalizer removal patch fails",
			deleting:     true,
			finalizers:   []string{common.FinalizerName},
			patch:        failPatch,
			wantErrMsgs:  []string{"failed to update " + testKind + " after finalizer removal"},
			wantValue:    "failed to update " + testKind + " after finalizer removal",
			wantWrites:   1,
			wantRemove:   1,
			wantFinalize: []string{common.FinalizerName},
		},
		{
			name:         "deleting without an operator finalizer does nothing",
			deleting:     true,
			finalizers:   []string{testForeignFinal},
			wantValue:    common.StatusOK,
			wantFinalize: []string{testForeignFinal},
		},
		{
			name:       "live object goes to reconciliation",
			wantHandle: 1,
			wantValue:  common.StatusOK,
			checkArgs:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			builder := fake.NewClientBuilder().
				WithScheme(scheme).
				WithStatusSubresource(&keycloakApi.KeycloakClientScope{})

			if !tt.noObject {
				obj := &keycloakApi.KeycloakClientScope{
					ObjectMeta: metav1.ObjectMeta{Namespace: key.Namespace, Name: key.Name, Finalizers: tt.finalizers},
					Status:     keycloakApi.KeycloakClientScopeStatus{Value: common.StatusOK},
				}

				if tt.deleting {
					now := metav1.Now()
					obj.DeletionTimestamp = &now
				}

				builder = builder.WithObjects(obj)
			}

			writes := 0

			cl := builder.WithInterceptorFuncs(interceptor.Funcs{
				Get: func(ctx context.Context, c client.WithWatch, k client.ObjectKey, o client.Object, opts ...client.GetOption) error {
					if tt.getErr != nil {
						return tt.getErr
					}

					return c.Get(ctx, k, o, opts...)
				},
				Patch: tt.patch,
				SubResourceUpdate: func(
					ctx context.Context, c client.Client, subResourceName string, o client.Object, opts ...client.SubResourceUpdateOption,
				) error {
					writes++

					if tt.statusFails {
						return k8sErrors.NewConflict(keycloakApi.GroupVersion.WithResource("keycloakclientscopes").GroupResource(), o.GetName(), errors.New("conflict"))
					}

					return c.SubResource(subResourceName).Update(ctx, o, opts...)
				},
			}).Build()

			type call struct {
				obj       *keycloakApi.KeycloakClientScope
				kClient   *keycloakapi.KeycloakClient
				realmName string
			}

			var removeCall, handleCall call

			removed, handled := 0, 0
			kClient := &keycloakapi.KeycloakClient{}
			tt.helper.kClient = kClient

			if tt.secondHelper != nil {
				tt.secondHelper.kClient = kClient
			}

			cfg := RealmRefReconcile[*keycloakApi.KeycloakClientScope]{
				Kind:       testKind,
				Helper:     tt.helper,
				Finalizers: []string{common.FinalizerName, testLegacyFinal},
				Remove: func(_ context.Context, o *keycloakApi.KeycloakClientScope, kc *keycloakapi.KeycloakClient, realm string) error {
					removed++
					removeCall = call{o, kc, realm}

					return tt.removeErr
				},
				HandleReconciliation: func(_ context.Context, o *keycloakApi.KeycloakClientScope, kc *keycloakapi.KeycloakClient, realm string) (reconcile.Result, error) {
					handled++
					handleCall = call{o, kc, realm}

					return reconcile.Result{RequeueAfter: 7}, nil
				},
			}

			ctx := ctrl.LoggerInto(context.Background(), logr.Discard())

			var (
				res reconcile.Result
				err error
			)

			reconciled := &keycloakApi.KeycloakClientScope{}

			for i := range max(tt.runs, 1) {
				if i > 0 && tt.secondHelper != nil {
					cfg.Helper = *tt.secondHelper
				}

				res, err = ReconcileRealmRef(ctx, cl, reconcile.Request{NamespacedName: key}, reconciled, cfg)
			}

			if len(tt.wantErrMsgs) == 0 {
				require.NoError(t, err)
			}

			for _, msg := range tt.wantErrMsgs {
				require.ErrorContains(t, err, msg)
			}

			if tt.wantErrIs != nil {
				require.ErrorIs(t, err, tt.wantErrIs)
			}

			assert.Equal(t, tt.wantWrites, writes)
			assert.Equal(t, tt.wantRemove, removed)
			assert.Equal(t, tt.wantHandle, handled)

			if tt.checkArgs {
				if tt.wantRemove > 0 {
					assert.Equal(t, testFakeRealm, removeCall.realmName)
					assert.Same(t, kClient, removeCall.kClient)
					assert.Same(t, reconciled, removeCall.obj)
				}

				if tt.wantHandle > 0 {
					assert.Equal(t, testFakeRealm, handleCall.realmName)
					assert.Same(t, kClient, handleCall.kClient)
					assert.Same(t, reconciled, handleCall.obj)
				}
			}

			switch {
			case tt.wantRequeue:
				assert.Equal(t, RequeueOnKeycloakNotAvailablePeriod, res.RequeueAfter)
			case tt.wantHandle > 0:
				assert.EqualValues(t, 7, res.RequeueAfter)
			default:
				assert.Zero(t, res.RequeueAfter)
			}

			if tt.noObject || tt.getErr != nil {
				return
			}

			got := &keycloakApi.KeycloakClientScope{}
			getErr := cl.Get(ctx, key, got)

			if tt.wantGone {
				require.True(t, k8sErrors.IsNotFound(getErr))

				return
			}

			require.NoError(t, getErr)
			assert.Contains(t, got.Status.Value, tt.wantValue)

			if tt.wantFinalize != nil {
				assert.ElementsMatch(t, tt.wantFinalize, got.Finalizers)
			}
		})
	}
}

func TestUpdateErrorStatus_EqualValueIsNotWritten(t *testing.T) {
	t.Parallel()

	scheme := runtime.NewScheme()
	require.NoError(t, keycloakApi.AddToScheme(scheme))

	obj := &keycloakApi.KeycloakClientScope{
		ObjectMeta: metav1.ObjectMeta{Namespace: "default", Name: "scope"},
		Status:     keycloakApi.KeycloakClientScopeStatus{Value: "boom"},
	}

	writes := 0
	cl := fake.NewClientBuilder().
		WithScheme(scheme).
		WithStatusSubresource(obj).
		WithObjects(obj).
		WithInterceptorFuncs(interceptor.Funcs{
			SubResourceUpdate: func(
				ctx context.Context, c client.Client, subResourceName string, o client.Object, opts ...client.SubResourceUpdateOption,
			) error {
				writes++

				return c.SubResource(subResourceName).Update(ctx, o, opts...)
			},
		}).
		Build()

	boom := errors.New("boom")
	ctx := ctrl.LoggerInto(context.Background(), logr.Discard())

	err := UpdateErrorStatus(ctx, cl, testKind, obj, boom)

	require.ErrorIs(t, err, boom)
	assert.Zero(t, writes)
}

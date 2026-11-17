package keycloakrealmgroupmembership

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
	"sigs.k8s.io/controller-runtime/pkg/event"

	"github.com/epam/edp-keycloak-operator/api/common"
	keycloakApi "github.com/epam/edp-keycloak-operator/api/v1"
	"github.com/epam/edp-keycloak-operator/internal/controller/helper"
	helpermock "github.com/epam/edp-keycloak-operator/internal/controller/helper/mocks"
	"github.com/epam/edp-keycloak-operator/pkg/client/keycloakapi"
	v2mocks "github.com/epam/edp-keycloak-operator/pkg/client/keycloakapi/mocks"
	"github.com/epam/edp-keycloak-operator/pkg/objectmeta"
)

func TestReconcileKeycloakRealmGroupMembership_Reconcile(t *testing.T) {
	t.Parallel()

	const (
		backoff         = 30 * time.Second
		successInterval = time.Hour
		groupID         = "group-id"
	)

	jane := keycloakApi.GroupMember{Username: "jane"}
	john := keycloakApi.GroupMember{Email: "john@example.com"}

	foreignFinalizer := "test/foreign"
	key := client.ObjectKey{Name: "membership", Namespace: "default"}
	prevStatus := keycloakApi.KeycloakRealmGroupMembershipStatus{Value: "previous", FailureCount: 2}

	scheme := runtime.NewScheme()
	require.NoError(t, keycloakApi.AddToScheme(scheme))

	expectBackoff := func(h *helpermock.MockControllerHelper) {
		h.EXPECT().SetFailureCount(mock.Anything).RunAndReturn(func(fc helper.FailureCountable) time.Duration {
			fc.SetFailureCount(fc.GetFailureCount() + 1)

			return backoff
		})
	}

	expectClient := func(h *helpermock.MockControllerHelper, users *v2mocks.MockUsersClient, groups *v2mocks.MockGroupsClient) {
		h.EXPECT().SetRealmOwnerRef(mock.Anything, mock.Anything).Return(nil)
		h.EXPECT().CreateKeycloakClientFromRealmRef(mock.Anything, mock.Anything).
			Return(&keycloakapi.KeycloakClient{Users: users, Groups: groups}, nil)
		h.EXPECT().GetRealmNameFromRef(mock.Anything, mock.Anything).Return("realm", nil)
	}

	expectRealmGone := func(h *helpermock.MockControllerHelper) {
		h.EXPECT().SetRealmOwnerRef(mock.Anything, mock.Anything).Return(nil)
		h.EXPECT().CreateKeycloakClientFromRealmRef(mock.Anything, mock.Anything).
			Return(nil, fmt.Errorf("x: %w", helper.ErrKeycloakRealmNotFound))
	}

	expectGroup := func(groups *v2mocks.MockGroupsClient) {
		groups.EXPECT().GetGroupByPath(mock.Anything, "realm", "/team").
			Return(&keycloakapi.GroupRepresentation{Id: ptr.To(groupID)}, nil, nil)
	}

	expectJane := func(users *v2mocks.MockUsersClient) {
		users.EXPECT().FindUserByUsername(mock.Anything, "realm", "jane").
			Return(&keycloakapi.UserRepresentation{Id: ptr.To("jane-id")}, nil, nil)
	}

	type mocks struct {
		h      *helpermock.MockControllerHelper
		users  *v2mocks.MockUsersClient
		groups *v2mocks.MockGroupsClient
	}

	tests := []struct {
		name       string
		finalizers []string
		deleting   bool
		preserve   bool
		patchFails bool
		setup      func(t *testing.T, m mocks, cl client.Client)

		wantRequeue   time.Duration
		wantWrites    int
		wantGone      bool
		wantUnchanged bool
		wantStatus    *keycloakApi.KeycloakRealmGroupMembershipStatus
		wantValue     string
	}{
		{
			name:       "live, all members resolved, success status and interval",
			finalizers: []string{finalizerName},
			setup: func(_ *testing.T, m mocks, _ client.Client) {
				expectClient(m.h, m.users, m.groups)
				expectGroup(m.groups)
				expectJane(m.users)
				m.users.EXPECT().FindUserByEmail(mock.Anything, "realm", "john@example.com").
					Return(&keycloakapi.UserRepresentation{Id: ptr.To("john-id")}, nil, nil)
				m.groups.EXPECT().GetGroupMembers(mock.Anything, "realm", groupID, mock.Anything).Return(nil, nil, nil)
				m.users.EXPECT().AddUserToGroup(mock.Anything, "realm", "jane-id", groupID).Return(nil, nil)
				m.users.EXPECT().AddUserToGroup(mock.Anything, "realm", "john-id", groupID).Return(nil, nil)
			},
			wantRequeue: successInterval,
			wantWrites:  1,
			wantStatus: &keycloakApi.KeycloakRealmGroupMembershipStatus{
				Value:           common.StatusOK,
				ResolvedMembers: []keycloakApi.GroupMember{jane, john},
			},
		},
		{
			name: "live, pending member, finalizer added and requeue capped",
			setup: func(_ *testing.T, m mocks, _ client.Client) {
				expectClient(m.h, m.users, m.groups)
				expectGroup(m.groups)
				expectJane(m.users)
				m.users.EXPECT().FindUserByEmail(mock.Anything, "realm", "john@example.com").
					Return(nil, nil, keycloakapi.ErrNotFound)
				m.groups.EXPECT().GetGroupMembers(mock.Anything, "realm", groupID, mock.Anything).
					Return([]keycloakapi.UserRepresentation{{Id: ptr.To("jane-id")}}, nil, nil)
			},
			wantRequeue: pendingMembersRequeuePeriod,
			wantWrites:  1,
			wantStatus: &keycloakApi.KeycloakRealmGroupMembershipStatus{
				Value:           common.StatusOK,
				ResolvedMembers: []keycloakApi.GroupMember{jane},
				PendingMembers:  []keycloakApi.GroupMember{john},
			},
		},
		{
			name:       "live, group missing, error status written",
			finalizers: []string{finalizerName},
			setup: func(_ *testing.T, m mocks, _ client.Client) {
				expectClient(m.h, m.users, m.groups)
				m.groups.EXPECT().GetGroupByPath(mock.Anything, "realm", "/team").Return(nil, nil, keycloakapi.ErrNotFound)
				expectBackoff(m.h)
			},
			wantRequeue: backoff,
			wantWrites:  1,
			wantValue:   `group "/team" not found, create it with a KeycloakRealmGroup`,
		},
		{
			name:       "deleting, listed members removed",
			finalizers: []string{finalizerName},
			deleting:   true,
			setup: func(_ *testing.T, m mocks, _ client.Client) {
				expectClient(m.h, m.users, m.groups)
				expectGroup(m.groups)
				expectJane(m.users)
				m.users.EXPECT().FindUserByEmail(mock.Anything, "realm", "john@example.com").
					Return(nil, nil, keycloakapi.ErrNotFound)
				m.users.EXPECT().RemoveUserFromGroup(mock.Anything, "realm", "jane-id", groupID).Return(nil, nil)
			},
			wantGone: true,
		},
		{
			name:       "deleting with preserve annotation, keycloak untouched",
			finalizers: []string{finalizerName},
			deleting:   true,
			preserve:   true,
			setup: func(_ *testing.T, m mocks, _ client.Client) {
				expectClient(m.h, m.users, m.groups)
			},
			wantGone: true,
		},
		{
			name:       "deleting, only foreign finalizer, status kept",
			finalizers: []string{foreignFinalizer},
			deleting:   true,
			setup: func(_ *testing.T, m mocks, _ client.Client) {
				expectClient(m.h, m.users, m.groups)
			},
			wantUnchanged: true,
		},
		{
			name:       "deleting, realm gone, finalizer removed",
			finalizers: []string{finalizerName},
			deleting:   true,
			setup: func(t *testing.T, m mocks, cl client.Client) {
				expectRealmGone(m.h)
				m.h.EXPECT().TryRemoveFinalizer(mock.Anything, mock.Anything, finalizerName).
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
			name:       "deleting, keycloak remove fails, error status written",
			finalizers: []string{finalizerName},
			deleting:   true,
			setup: func(_ *testing.T, m mocks, _ client.Client) {
				expectClient(m.h, m.users, m.groups)
				expectGroup(m.groups)
				expectJane(m.users)
				m.users.EXPECT().RemoveUserFromGroup(mock.Anything, "realm", "jane-id", groupID).
					Return(nil, errors.New("remove failed"))
				expectBackoff(m.h)
			},
			wantRequeue: backoff,
			wantWrites:  1,
			wantValue:   `failed to remove group members: unable to remove user with username "jane" from group: remove failed`,
		},
		{
			name:       "deleting, finalizer patch fails, error status written",
			finalizers: []string{finalizerName},
			deleting:   true,
			preserve:   true,
			patchFails: true,
			setup: func(_ *testing.T, m mocks, _ client.Client) {
				expectClient(m.h, m.users, m.groups)
				expectBackoff(m.h)
			},
			wantRequeue: backoff,
			wantWrites:  1,
			wantValue:   "failed to remove finalizer: unable to patch object: ",
		},
		{
			name:       "keycloak unavailable, requeue with unchanged status",
			finalizers: []string{finalizerName},
			setup: func(_ *testing.T, m mocks, _ client.Client) {
				m.h.EXPECT().SetRealmOwnerRef(mock.Anything, mock.Anything).Return(nil)
				m.h.EXPECT().CreateKeycloakClientFromRealmRef(mock.Anything, mock.Anything).
					Return(nil, fmt.Errorf("x: %w", helper.ErrKeycloakIsNotAvailable))
			},
			wantRequeue:   helper.RequeueOnKeycloakNotAvailablePeriod,
			wantWrites:    1,
			wantUnchanged: true,
		},
		{
			name:       "live, realm gone, error status written",
			finalizers: []string{finalizerName},
			setup: func(_ *testing.T, m mocks, _ client.Client) {
				expectRealmGone(m.h)
				expectBackoff(m.h)
			},
			wantRequeue: backoff,
			wantWrites:  1,
			wantValue:   "unable to create keycloak client from realm ref: x: " + helper.ErrKeycloakRealmNotFound.Error(),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			membership := &keycloakApi.KeycloakRealmGroupMembership{
				ObjectMeta: metav1.ObjectMeta{
					Name:       key.Name,
					Namespace:  key.Namespace,
					Finalizers: tt.finalizers,
				},
				Spec: keycloakApi.KeycloakRealmGroupMembershipSpec{
					RealmRef: common.RealmRef{Kind: keycloakApi.KeycloakRealmKind, Name: "realm"},
					Group:    "/team",
					Members:  []keycloakApi.GroupMember{jane, john},
				},
				Status: prevStatus,
			}

			if tt.deleting {
				membership.DeletionTimestamp = ptr.To(metav1.Now())
			}

			if tt.preserve {
				membership.Annotations = map[string]string{objectmeta.PreserveResourcesOnDeletionAnnotation: "true"}
			}

			writes := 0

			cl := fake.NewClientBuilder().
				WithScheme(scheme).
				WithStatusSubresource(&keycloakApi.KeycloakRealmGroupMembership{}).
				WithObjects(membership).
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
							return k8sErrors.NewConflict(
								keycloakApi.GroupVersion.WithResource("keycloakrealmgroupmemberships").GroupResource(),
								obj.GetName(), errors.New("conflict"),
							)
						}

						return c.Patch(ctx, obj, patch, opts...)
					},
				}).
				Build()

			m := mocks{
				h:      helpermock.NewMockControllerHelper(t),
				users:  v2mocks.NewMockUsersClient(t),
				groups: v2mocks.NewMockGroupsClient(t),
			}

			tt.setup(t, m, cl)

			r := NewReconcileKeycloakRealmGroupMembership(cl, m.h)
			r.successReconcileTimeout = successInterval
			reqCtx := ctrl.LoggerInto(t.Context(), logr.Discard())

			res, err := r.Reconcile(reqCtx, ctrl.Request{NamespacedName: key})
			require.NoError(t, err)

			assert.Equal(t, tt.wantRequeue, res.RequeueAfter)
			assert.Equal(t, tt.wantWrites, writes)

			got := &keycloakApi.KeycloakRealmGroupMembership{}
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
				assert.True(t, controllerutil.ContainsFinalizer(got, finalizerName))

				return
			}

			assert.Contains(t, got.Status.Value, tt.wantValue)
			assert.Equal(t, prevStatus.FailureCount+1, got.Status.FailureCount)
		})
	}
}

func TestIsSpecUpdated(t *testing.T) {
	t.Parallel()

	old := &keycloakApi.KeycloakRealmGroupMembership{
		Spec: keycloakApi.KeycloakRealmGroupMembershipSpec{Group: "a"},
	}

	changed := old.DeepCopy()
	changed.Spec.Group = "b"

	statusOnly := old.DeepCopy()
	statusOnly.Status.Value = common.StatusOK

	deleting := old.DeepCopy()
	deleting.DeletionTimestamp = ptr.To(metav1.Now())

	assert.True(t, isSpecUpdated(event.UpdateEvent{ObjectOld: old, ObjectNew: changed}))
	assert.False(t, isSpecUpdated(event.UpdateEvent{ObjectOld: old, ObjectNew: statusOnly}))
	assert.True(t, isSpecUpdated(event.UpdateEvent{ObjectOld: old, ObjectNew: deleting}))
}

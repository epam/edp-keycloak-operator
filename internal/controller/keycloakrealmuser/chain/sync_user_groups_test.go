package chain

import (
	"context"
	"errors"
	"testing"

	"github.com/Nerzal/gocloak/v12"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"

	keycloakApi "github.com/epam/edp-keycloak-operator/api/v1"
	keycloakv2 "github.com/epam/edp-keycloak-operator/pkg/client/keycloakv2"
	v2mocks "github.com/epam/edp-keycloak-operator/pkg/client/keycloakv2/mocks"
)

func TestNewSyncUserGroups(t *testing.T) {
	h := NewSyncUserGroups(&keycloakv2.KeycloakClient{})
	require.NotNil(t, h)
}

func TestSyncUserGroups_Serve(t *testing.T) {
	realm := &gocloak.RealmRepresentation{Realm: ptr.To("test-realm")}

	tests := []struct {
		name      string
		user      *keycloakApi.KeycloakRealmUser
		userCtx   *UserContext
		mockSetup func(u *v2mocks.MockUsersClient, g *v2mocks.MockGroupsClient)
		wantErr   require.ErrorAssertionFunc
	}{
		{
			name: "success - name-style group added when not yet assigned",
			user: &keycloakApi.KeycloakRealmUser{
				ObjectMeta: metav1.ObjectMeta{Name: "u", Namespace: "default"},
				Spec:       keycloakApi.KeycloakRealmUserSpec{Groups: []string{"developers"}},
			},
			userCtx: &UserContext{UserID: "user-1"},
			mockSetup: func(u *v2mocks.MockUsersClient, g *v2mocks.MockGroupsClient) {
				u.EXPECT().GetUserGroups(context.Background(), "test-realm", "user-1").
					Return(nil, nil, nil)
				g.EXPECT().FindGroupByName(context.Background(), "test-realm", "developers").
					Return(&keycloakv2.GroupRepresentation{Id: ptr.To("grp-1"), Name: ptr.To("developers")}, nil, nil)
				u.EXPECT().AddUserToGroup(context.Background(), "test-realm", "user-1", "grp-1").
					Return(nil, nil)
			},
			wantErr: require.NoError,
		},
		{
			name: "success - path-style group added when not yet assigned",
			user: &keycloakApi.KeycloakRealmUser{
				ObjectMeta: metav1.ObjectMeta{Name: "u", Namespace: "default"},
				Spec:       keycloakApi.KeycloakRealmUserSpec{Groups: []string{"/system/data_admin"}},
			},
			userCtx: &UserContext{UserID: "user-2"},
			mockSetup: func(u *v2mocks.MockUsersClient, g *v2mocks.MockGroupsClient) {
				u.EXPECT().GetUserGroups(context.Background(), "test-realm", "user-2").
					Return(nil, nil, nil)
				g.EXPECT().GetGroupByPath(context.Background(), "test-realm", "/system/data_admin").
					Return(&keycloakv2.GroupRepresentation{
						Id:   ptr.To("grp-nested"),
						Path: ptr.To("/system/data_admin"),
					}, nil, nil)
				u.EXPECT().AddUserToGroup(context.Background(), "test-realm", "user-2", "grp-nested").
					Return(nil, nil)
			},
			wantErr: require.NoError,
		},
		{
			name: "success - group already assigned, no add called",
			user: &keycloakApi.KeycloakRealmUser{
				ObjectMeta: metav1.ObjectMeta{Name: "u", Namespace: "default"},
				Spec:       keycloakApi.KeycloakRealmUserSpec{Groups: []string{"developers"}},
			},
			userCtx: &UserContext{UserID: "user-3"},
			mockSetup: func(u *v2mocks.MockUsersClient, g *v2mocks.MockGroupsClient) {
				u.EXPECT().GetUserGroups(context.Background(), "test-realm", "user-3").
					Return([]keycloakv2.GroupRepresentation{
						{Id: ptr.To("grp-1"), Name: ptr.To("developers")},
					}, nil, nil)
				g.EXPECT().FindGroupByName(context.Background(), "test-realm", "developers").
					Return(&keycloakv2.GroupRepresentation{Id: ptr.To("grp-1"), Name: ptr.To("developers")}, nil, nil)
			},
			wantErr: require.NoError,
		},
		{
			name: "success - path-style: correct nested group selected over root with same name",
			user: &keycloakApi.KeycloakRealmUser{
				ObjectMeta: metav1.ObjectMeta{Name: "u", Namespace: "default"},
				Spec:       keycloakApi.KeycloakRealmUserSpec{Groups: []string{"/system/data_admin"}},
			},
			userCtx: &UserContext{UserID: "user-4"},
			mockSetup: func(u *v2mocks.MockUsersClient, g *v2mocks.MockGroupsClient) {
				u.EXPECT().GetUserGroups(context.Background(), "test-realm", "user-4").
					Return([]keycloakv2.GroupRepresentation{
						{Id: ptr.To("grp-root"), Name: ptr.To("data_admin"), Path: ptr.To("/data_admin")},
					}, nil, nil)
				g.EXPECT().GetGroupByPath(context.Background(), "test-realm", "/system/data_admin").
					Return(&keycloakv2.GroupRepresentation{
						Id:   ptr.To("grp-nested"),
						Path: ptr.To("/system/data_admin"),
					}, nil, nil)
				u.EXPECT().AddUserToGroup(context.Background(), "test-realm", "user-4", "grp-nested").
					Return(nil, nil)
				u.EXPECT().RemoveUserFromGroup(context.Background(), "test-realm", "user-4", "grp-root").
					Return(nil, nil)
			},
			wantErr: require.NoError,
		},
		{
			name: "success - add-only: extra group not removed",
			user: &keycloakApi.KeycloakRealmUser{
				ObjectMeta: metav1.ObjectMeta{Name: "u", Namespace: "default"},
				Spec: keycloakApi.KeycloakRealmUserSpec{
					Groups:                 []string{"developers"},
					ReconciliationStrategy: keycloakApi.ReconciliationStrategyAddOnly,
				},
			},
			userCtx: &UserContext{UserID: "user-5"},
			mockSetup: func(u *v2mocks.MockUsersClient, g *v2mocks.MockGroupsClient) {
				u.EXPECT().GetUserGroups(context.Background(), "test-realm", "user-5").
					Return([]keycloakv2.GroupRepresentation{
						{Id: ptr.To("grp-1"), Name: ptr.To("developers")},
						{Id: ptr.To("grp-extra"), Name: ptr.To("ops")},
					}, nil, nil)
				g.EXPECT().FindGroupByName(context.Background(), "test-realm", "developers").
					Return(&keycloakv2.GroupRepresentation{Id: ptr.To("grp-1"), Name: ptr.To("developers")}, nil, nil)
			},
			wantErr: require.NoError,
		},
		{
			name: "success - add-only with empty groups skips API calls",
			user: &keycloakApi.KeycloakRealmUser{
				ObjectMeta: metav1.ObjectMeta{Name: "u", Namespace: "default"},
				Spec: keycloakApi.KeycloakRealmUserSpec{
					Groups:                 []string{},
					ReconciliationStrategy: keycloakApi.ReconciliationStrategyAddOnly,
				},
			},
			userCtx:   &UserContext{UserID: "user-9"},
			mockSetup: func(_ *v2mocks.MockUsersClient, _ *v2mocks.MockGroupsClient) {},
			wantErr:   require.NoError,
		},
		{
			name: "error - FindGroupByName returns nil (group not found)",
			user: &keycloakApi.KeycloakRealmUser{
				ObjectMeta: metav1.ObjectMeta{Name: "u", Namespace: "default"},
				Spec:       keycloakApi.KeycloakRealmUserSpec{Groups: []string{"missing-group"}},
			},
			userCtx: &UserContext{UserID: "user-6"},
			mockSetup: func(u *v2mocks.MockUsersClient, g *v2mocks.MockGroupsClient) {
				u.EXPECT().GetUserGroups(context.Background(), "test-realm", "user-6").
					Return(nil, nil, nil)
				g.EXPECT().FindGroupByName(context.Background(), "test-realm", "missing-group").
					Return(nil, nil, nil)
			},
			wantErr: func(t require.TestingT, err error, _ ...any) {
				require.Error(t, err)
				assert.Contains(t, err.Error(), "group not found by name")
			},
		},
		{
			name: "error - GetGroupByPath returns error",
			user: &keycloakApi.KeycloakRealmUser{
				ObjectMeta: metav1.ObjectMeta{Name: "u", Namespace: "default"},
				Spec:       keycloakApi.KeycloakRealmUserSpec{Groups: []string{"/bad/path"}},
			},
			userCtx: &UserContext{UserID: "user-7"},
			mockSetup: func(u *v2mocks.MockUsersClient, g *v2mocks.MockGroupsClient) {
				u.EXPECT().GetUserGroups(context.Background(), "test-realm", "user-7").
					Return(nil, nil, nil)
				g.EXPECT().GetGroupByPath(context.Background(), "test-realm", "/bad/path").
					Return(nil, nil, errors.New("keycloak 404"))
			},
			wantErr: func(t require.TestingT, err error, _ ...any) {
				require.Error(t, err)
				assert.Contains(t, err.Error(), "unable to get group by path")
			},
		},
		{
			name: "error - GetUserGroups fails",
			user: &keycloakApi.KeycloakRealmUser{
				ObjectMeta: metav1.ObjectMeta{Name: "u", Namespace: "default"},
				Spec:       keycloakApi.KeycloakRealmUserSpec{Groups: []string{"developers"}},
			},
			userCtx: &UserContext{UserID: "user-8"},
			mockSetup: func(u *v2mocks.MockUsersClient, g *v2mocks.MockGroupsClient) {
				u.EXPECT().GetUserGroups(context.Background(), "test-realm", "user-8").
					Return(nil, nil, errors.New("connection refused"))
			},
			wantErr: func(t require.TestingT, err error, _ ...any) {
				require.Error(t, err)
				assert.Contains(t, err.Error(), "unable to get user groups")
			},
		},
		{
			name: "error - AddUserToGroup fails",
			user: &keycloakApi.KeycloakRealmUser{
				ObjectMeta: metav1.ObjectMeta{Name: "u", Namespace: "default"},
				Spec:       keycloakApi.KeycloakRealmUserSpec{Groups: []string{"developers"}},
			},
			userCtx: &UserContext{UserID: "user-10"},
			mockSetup: func(u *v2mocks.MockUsersClient, g *v2mocks.MockGroupsClient) {
				u.EXPECT().GetUserGroups(context.Background(), "test-realm", "user-10").
					Return(nil, nil, nil)
				g.EXPECT().FindGroupByName(context.Background(), "test-realm", "developers").
					Return(&keycloakv2.GroupRepresentation{Id: ptr.To("grp-1"), Name: ptr.To("developers")}, nil, nil)
				u.EXPECT().AddUserToGroup(context.Background(), "test-realm", "user-10", "grp-1").
					Return(nil, errors.New("keycloak error"))
			},
			wantErr: func(t require.TestingT, err error, _ ...any) {
				require.Error(t, err)
				assert.Contains(t, err.Error(), "unable to add user to group")
			},
		},
		{
			name: "error - RemoveUserFromGroup fails",
			user: &keycloakApi.KeycloakRealmUser{
				ObjectMeta: metav1.ObjectMeta{Name: "u", Namespace: "default"},
				Spec:       keycloakApi.KeycloakRealmUserSpec{Groups: []string{"developers"}},
			},
			userCtx: &UserContext{UserID: "user-11"},
			mockSetup: func(u *v2mocks.MockUsersClient, g *v2mocks.MockGroupsClient) {
				u.EXPECT().GetUserGroups(context.Background(), "test-realm", "user-11").
					Return([]keycloakv2.GroupRepresentation{
						{Id: ptr.To("grp-1"), Name: ptr.To("developers")},
						{Id: ptr.To("grp-extra"), Name: ptr.To("ops")},
					}, nil, nil)
				g.EXPECT().FindGroupByName(context.Background(), "test-realm", "developers").
					Return(&keycloakv2.GroupRepresentation{Id: ptr.To("grp-1"), Name: ptr.To("developers")}, nil, nil)
				u.EXPECT().RemoveUserFromGroup(context.Background(), "test-realm", "user-11", "grp-extra").
					Return(nil, errors.New("keycloak error"))
			},
			wantErr: func(t require.TestingT, err error, _ ...any) {
				require.Error(t, err)
				assert.Contains(t, err.Error(), "unable to remove user from group")
			},
		},
		{
			name: "error - GetGroupByPath returns nil (group not found by path)",
			user: &keycloakApi.KeycloakRealmUser{
				ObjectMeta: metav1.ObjectMeta{Name: "u", Namespace: "default"},
				Spec:       keycloakApi.KeycloakRealmUserSpec{Groups: []string{"/missing/path"}},
			},
			userCtx: &UserContext{UserID: "user-12"},
			mockSetup: func(u *v2mocks.MockUsersClient, g *v2mocks.MockGroupsClient) {
				u.EXPECT().GetUserGroups(context.Background(), "test-realm", "user-12").
					Return(nil, nil, nil)
				g.EXPECT().GetGroupByPath(context.Background(), "test-realm", "/missing/path").
					Return(nil, nil, nil)
			},
			wantErr: func(t require.TestingT, err error, _ ...any) {
				require.Error(t, err)
				assert.Contains(t, err.Error(), "group not found by path")
			},
		},
		{
			name: "success - full strategy removes extra groups",
			user: &keycloakApi.KeycloakRealmUser{
				ObjectMeta: metav1.ObjectMeta{Name: "u", Namespace: "default"},
				Spec:       keycloakApi.KeycloakRealmUserSpec{Groups: []string{"developers"}},
			},
			userCtx: &UserContext{UserID: "user-13"},
			mockSetup: func(u *v2mocks.MockUsersClient, g *v2mocks.MockGroupsClient) {
				u.EXPECT().GetUserGroups(context.Background(), "test-realm", "user-13").
					Return([]keycloakv2.GroupRepresentation{
						{Id: ptr.To("grp-1"), Name: ptr.To("developers")},
						{Id: ptr.To("grp-2"), Name: ptr.To("ops")},
					}, nil, nil)
				g.EXPECT().FindGroupByName(context.Background(), "test-realm", "developers").
					Return(&keycloakv2.GroupRepresentation{Id: ptr.To("grp-1"), Name: ptr.To("developers")}, nil, nil)
				u.EXPECT().RemoveUserFromGroup(context.Background(), "test-realm", "user-13", "grp-2").
					Return(nil, nil)
			},
			wantErr: require.NoError,
		},
		{
			name: "success - empty groups with full strategy removes all",
			user: &keycloakApi.KeycloakRealmUser{
				ObjectMeta: metav1.ObjectMeta{Name: "u", Namespace: "default"},
				Spec:       keycloakApi.KeycloakRealmUserSpec{Groups: []string{}},
			},
			userCtx: &UserContext{UserID: "user-14"},
			mockSetup: func(u *v2mocks.MockUsersClient, _ *v2mocks.MockGroupsClient) {
				u.EXPECT().GetUserGroups(context.Background(), "test-realm", "user-14").
					Return([]keycloakv2.GroupRepresentation{
						{Id: ptr.To("grp-1"), Name: ptr.To("developers")},
						{Id: ptr.To("grp-2"), Name: ptr.To("ops")},
					}, nil, nil)
				u.EXPECT().RemoveUserFromGroup(context.Background(), "test-realm", "user-14", "grp-1").
					Return(nil, nil)
				u.EXPECT().RemoveUserFromGroup(context.Background(), "test-realm", "user-14", "grp-2").
					Return(nil, nil)
			},
			wantErr: require.NoError,
		},
		{
			name: "success - multiple groups: mixed name-style and path-style",
			user: &keycloakApi.KeycloakRealmUser{
				ObjectMeta: metav1.ObjectMeta{Name: "u", Namespace: "default"},
				Spec:       keycloakApi.KeycloakRealmUserSpec{Groups: []string{"developers", "/system/admins"}},
			},
			userCtx: &UserContext{UserID: "user-15"},
			mockSetup: func(u *v2mocks.MockUsersClient, g *v2mocks.MockGroupsClient) {
				u.EXPECT().GetUserGroups(context.Background(), "test-realm", "user-15").
					Return(nil, nil, nil)
				g.EXPECT().FindGroupByName(context.Background(), "test-realm", "developers").
					Return(&keycloakv2.GroupRepresentation{Id: ptr.To("grp-1"), Name: ptr.To("developers")}, nil, nil)
				g.EXPECT().GetGroupByPath(context.Background(), "test-realm", "/system/admins").
					Return(&keycloakv2.GroupRepresentation{
						Id:   ptr.To("grp-2"),
						Path: ptr.To("/system/admins"),
					}, nil, nil)
				u.EXPECT().AddUserToGroup(context.Background(), "test-realm", "user-15", "grp-1").
					Return(nil, nil)
				u.EXPECT().AddUserToGroup(context.Background(), "test-realm", "user-15", "grp-2").
					Return(nil, nil)
			},
			wantErr: require.NoError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockUsers := v2mocks.NewMockUsersClient(t)
			mockGroups := v2mocks.NewMockGroupsClient(t)
			tt.mockSetup(mockUsers, mockGroups)

			h := NewSyncUserGroups(&keycloakv2.KeycloakClient{
				Users:  mockUsers,
				Groups: mockGroups,
			})
			err := h.Serve(
				context.Background(),
				tt.user,
				nil, /* legacy client unused */
				realm,
				tt.userCtx,
			)

			tt.wantErr(t, err)
		})
	}
}

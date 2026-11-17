package chain

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"k8s.io/utils/ptr"

	keycloakApi "github.com/epam/edp-keycloak-operator/api/v1"
	"github.com/epam/edp-keycloak-operator/pkg/client/keycloakapi"
	"github.com/epam/edp-keycloak-operator/pkg/client/keycloakapi/mocks"
)

func user(id string) keycloakapi.UserRepresentation {
	return keycloakapi.UserRepresentation{Id: ptr.To(id), Username: ptr.To(id)}
}

func expectMembers(g *mocks.MockGroupsClient, users ...keycloakapi.UserRepresentation) {
	g.EXPECT().GetGroupMembers(context.Background(), testRealm, testGroupID, mock.Anything).
		Return(users, nil, nil).Once()
}

func TestSyncMembers_Serve(t *testing.T) {
	resolved := []ResolvedMember{
		{Member: keycloakApi.GroupMember{Username: "jane"}, UserID: "jane-id"},
		{Member: keycloakApi.GroupMember{Email: "john@example.com"}, UserID: "john-id"},
	}

	tests := []struct {
		name      string
		strategy  string
		mockSetup func(u *mocks.MockUsersClient, g *mocks.MockGroupsClient)
		wantErr   string
	}{
		{
			name: "addOnly adds missing members and keeps others",
			mockSetup: func(u *mocks.MockUsersClient, g *mocks.MockGroupsClient) {
				expectMembers(g, user("jane-id"), user("other-id"))
				u.EXPECT().AddUserToGroup(context.Background(), testRealm, "john-id", testGroupID).Return(nil, nil)
			},
		},
		{
			name: "addOnly no write when members match",
			mockSetup: func(_ *mocks.MockUsersClient, g *mocks.MockGroupsClient) {
				expectMembers(g, user("jane-id"), user("john-id"))
			},
		},
		{
			name:     "full removes unlisted members",
			strategy: keycloakApi.ReconciliationStrategyFull,
			mockSetup: func(u *mocks.MockUsersClient, g *mocks.MockGroupsClient) {
				expectMembers(g, user("jane-id"), user("other-id"))
				u.EXPECT().AddUserToGroup(context.Background(), testRealm, "john-id", testGroupID).Return(nil, nil)
				u.EXPECT().RemoveUserFromGroup(context.Background(), testRealm, "other-id", testGroupID).Return(nil, nil)
			},
		},
		{
			name: "add error",
			mockSetup: func(u *mocks.MockUsersClient, g *mocks.MockGroupsClient) {
				expectMembers(g, user("jane-id"))
				u.EXPECT().AddUserToGroup(context.Background(), testRealm, "john-id", testGroupID).
					Return(nil, errors.New("api error"))
			},
			wantErr: `unable to add user with email "john@example.com" to group: api error`,
		},
		{
			name:     "remove error",
			strategy: keycloakApi.ReconciliationStrategyFull,
			mockSetup: func(u *mocks.MockUsersClient, g *mocks.MockGroupsClient) {
				expectMembers(g, user("jane-id"), user("john-id"), user("other-id"))
				u.EXPECT().RemoveUserFromGroup(context.Background(), testRealm, "other-id", testGroupID).
					Return(nil, errors.New("api error"))
			},
			wantErr: `unable to remove user "other-id" from group: api error`,
		},
		{
			name: "list members error",
			mockSetup: func(_ *mocks.MockUsersClient, g *mocks.MockGroupsClient) {
				g.EXPECT().GetGroupMembers(context.Background(), testRealm, testGroupID, mock.Anything).
					Return(nil, nil, errors.New("api error"))
			},
			wantErr: "unable to get group members: api error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			u := mocks.NewMockUsersClient(t)
			g := mocks.NewMockGroupsClient(t)
			tt.mockSetup(u, g)

			membership := &keycloakApi.KeycloakRealmGroupMembership{
				Spec: keycloakApi.KeycloakRealmGroupMembershipSpec{Strategy: tt.strategy},
			}
			mCtx := &MembershipContext{GroupID: testGroupID, Resolved: resolved}

			err := NewSyncMembers(&keycloakapi.KeycloakClient{Users: u, Groups: g}).
				Serve(context.Background(), membership, testRealm, mCtx)

			if tt.wantErr != "" {
				require.EqualError(t, err, tt.wantErr)

				return
			}

			require.NoError(t, err)
		})
	}
}

func TestSyncMembers_Serve_Pagination(t *testing.T) {
	u := mocks.NewMockUsersClient(t)
	g := mocks.NewMockGroupsClient(t)

	firstPage := make([]keycloakapi.UserRepresentation, 0, groupMembersPageSize)
	for i := range groupMembersPageSize {
		firstPage = append(firstPage, user(fmt.Sprintf("user-%d", i)))
	}

	g.EXPECT().GetGroupMembers(context.Background(), testRealm, testGroupID,
		mock.MatchedBy(func(p *keycloakapi.GetGroupMembersParams) bool { return *p.First == 0 })).
		Return(firstPage, nil, nil)
	g.EXPECT().GetGroupMembers(context.Background(), testRealm, testGroupID,
		mock.MatchedBy(func(p *keycloakapi.GetGroupMembersParams) bool { return *p.First == groupMembersPageSize })).
		Return([]keycloakapi.UserRepresentation{user("jane-id")}, nil, nil)

	// jane-id is on the second page: no add.
	membership := &keycloakApi.KeycloakRealmGroupMembership{}
	mCtx := &MembershipContext{
		GroupID:  testGroupID,
		Resolved: []ResolvedMember{{Member: keycloakApi.GroupMember{Username: "jane"}, UserID: "jane-id"}},
	}

	err := NewSyncMembers(&keycloakapi.KeycloakClient{Users: u, Groups: g}).
		Serve(context.Background(), membership, testRealm, mCtx)
	require.NoError(t, err)
}

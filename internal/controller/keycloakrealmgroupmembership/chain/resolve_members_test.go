package chain

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"k8s.io/utils/ptr"

	keycloakApi "github.com/epam/edp-keycloak-operator/api/v1"
	"github.com/epam/edp-keycloak-operator/pkg/client/keycloakapi"
	"github.com/epam/edp-keycloak-operator/pkg/client/keycloakapi/mocks"
)

func TestResolveMembers_Serve(t *testing.T) {
	jane := keycloakApi.GroupMember{Username: "jane"}
	john := keycloakApi.GroupMember{Email: "john@example.com"}

	tests := []struct {
		name         string
		members      []keycloakApi.GroupMember
		mockSetup    func(u *mocks.MockUsersClient)
		wantResolved []ResolvedMember
		wantPending  []keycloakApi.GroupMember
		wantErr      string
	}{
		{
			name:    "username and email resolved",
			members: []keycloakApi.GroupMember{jane, john},
			mockSetup: func(u *mocks.MockUsersClient) {
				u.EXPECT().FindUserByUsername(context.Background(), testRealm, "jane").
					Return(&keycloakapi.UserRepresentation{Id: ptr.To("jane-id")}, nil, nil)
				u.EXPECT().FindUserByEmail(context.Background(), testRealm, "john@example.com").
					Return(&keycloakapi.UserRepresentation{Id: ptr.To("john-id")}, nil, nil)
			},
			wantResolved: []ResolvedMember{{Member: jane, UserID: "jane-id"}, {Member: john, UserID: "john-id"}},
		},
		{
			name:    "missing user is pending and never created",
			members: []keycloakApi.GroupMember{jane, john},
			mockSetup: func(u *mocks.MockUsersClient) {
				u.EXPECT().FindUserByUsername(context.Background(), testRealm, "jane").
					Return(&keycloakapi.UserRepresentation{Id: ptr.To("jane-id")}, nil, nil)
				u.EXPECT().FindUserByEmail(context.Background(), testRealm, "john@example.com").
					Return(nil, nil, keycloakapi.ErrNotFound)
			},
			wantResolved: []ResolvedMember{{Member: jane, UserID: "jane-id"}},
			wantPending:  []keycloakApi.GroupMember{john},
		},
		{
			name:    "duplicate spec entries looked up once",
			members: []keycloakApi.GroupMember{jane, jane},
			mockSetup: func(u *mocks.MockUsersClient) {
				u.EXPECT().FindUserByUsername(context.Background(), testRealm, "jane").
					Return(&keycloakapi.UserRepresentation{Id: ptr.To("jane-id")}, nil, nil).Once()
			},
			wantResolved: []ResolvedMember{{Member: jane, UserID: "jane-id"}},
		},
		{
			name:    "lookup error",
			members: []keycloakApi.GroupMember{john},
			mockSetup: func(u *mocks.MockUsersClient) {
				u.EXPECT().FindUserByEmail(context.Background(), testRealm, "john@example.com").
					Return(nil, nil, errors.New("2 users found"))
			},
			wantErr: "unable to find user with email",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			u := mocks.NewMockUsersClient(t)
			tt.mockSetup(u)

			membership := &keycloakApi.KeycloakRealmGroupMembership{
				Spec: keycloakApi.KeycloakRealmGroupMembershipSpec{Members: tt.members},
			}
			mCtx := &MembershipContext{}

			err := NewResolveMembers(&keycloakapi.KeycloakClient{Users: u}).
				Serve(context.Background(), membership, testRealm, mCtx)

			if tt.wantErr != "" {
				require.ErrorContains(t, err, tt.wantErr)

				return
			}

			require.NoError(t, err)
			assert.Equal(t, tt.wantResolved, mCtx.Resolved)
			assert.Equal(t, tt.wantPending, mCtx.Pending)
		})
	}
}

package chain

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"

	keycloakApi "github.com/epam/edp-keycloak-operator/api/v1"
	"github.com/epam/edp-keycloak-operator/pkg/client/keycloakapi"
	"github.com/epam/edp-keycloak-operator/pkg/client/keycloakapi/mocks"
	"github.com/epam/edp-keycloak-operator/pkg/objectmeta"
)

func TestRemoveMembers_ServeRequest(t *testing.T) {
	members := []keycloakApi.GroupMember{{Username: "jane"}, {Email: "john@example.com"}}

	tests := []struct {
		name        string
		annotations map[string]string
		mockSetup   func(u *mocks.MockUsersClient, g *mocks.MockGroupsClient)
		wantErr     string
	}{
		{
			name: "listed members removed, missing user skipped",
			mockSetup: func(u *mocks.MockUsersClient, g *mocks.MockGroupsClient) {
				g.EXPECT().GetGroupByPath(context.Background(), testRealm, testGroup).
					Return(&keycloakapi.GroupRepresentation{Id: ptr.To(testGroupID)}, nil, nil)
				u.EXPECT().FindUserByUsername(context.Background(), testRealm, "jane").
					Return(&keycloakapi.UserRepresentation{Id: ptr.To("jane-id")}, nil, nil)
				u.EXPECT().FindUserByEmail(context.Background(), testRealm, "john@example.com").
					Return(nil, nil, keycloakapi.ErrNotFound)
				u.EXPECT().RemoveUserFromGroup(context.Background(), testRealm, "jane-id", testGroupID).Return(nil, nil)
			},
		},
		{
			name:        "preserve annotation leaves keycloak alone",
			annotations: map[string]string{objectmeta.PreserveResourcesOnDeletionAnnotation: "true"},
			mockSetup:   func(_ *mocks.MockUsersClient, _ *mocks.MockGroupsClient) {},
		},
		{
			name: "group not found",
			mockSetup: func(_ *mocks.MockUsersClient, g *mocks.MockGroupsClient) {
				g.EXPECT().GetGroupByPath(context.Background(), testRealm, testGroup).
					Return(nil, nil, keycloakapi.ErrNotFound)
			},
		},
		{
			name: "remove error",
			mockSetup: func(u *mocks.MockUsersClient, g *mocks.MockGroupsClient) {
				g.EXPECT().GetGroupByPath(context.Background(), testRealm, testGroup).
					Return(&keycloakapi.GroupRepresentation{Id: ptr.To(testGroupID)}, nil, nil)
				u.EXPECT().FindUserByUsername(context.Background(), testRealm, "jane").
					Return(&keycloakapi.UserRepresentation{Id: ptr.To("jane-id")}, nil, nil)
				u.EXPECT().RemoveUserFromGroup(context.Background(), testRealm, "jane-id", testGroupID).
					Return(nil, errors.New("api error"))
			},
			wantErr: `unable to remove user with username "jane" from group: api error`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			u := mocks.NewMockUsersClient(t)
			g := mocks.NewMockGroupsClient(t)
			tt.mockSetup(u, g)

			membership := &keycloakApi.KeycloakRealmGroupMembership{
				ObjectMeta: metav1.ObjectMeta{Annotations: tt.annotations},
				Spec:       keycloakApi.KeycloakRealmGroupMembershipSpec{Group: testGroup, Members: members},
			}

			err := NewRemoveMembers(&keycloakapi.KeycloakClient{Users: u, Groups: g}).
				ServeRequest(context.Background(), membership, testRealm)

			if tt.wantErr != "" {
				require.EqualError(t, err, tt.wantErr)

				return
			}

			require.NoError(t, err)
		})
	}
}

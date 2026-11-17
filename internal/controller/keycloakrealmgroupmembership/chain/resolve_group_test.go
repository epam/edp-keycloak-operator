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

func TestResolveGroup_Serve(t *testing.T) {
	tests := []struct {
		name      string
		group     string
		mockSetup func(g *mocks.MockGroupsClient)
		wantID    string
		wantErr   string
	}{
		{
			name:  "path resolved with GetGroupByPath",
			group: testGroup,
			mockSetup: func(g *mocks.MockGroupsClient) {
				g.EXPECT().GetGroupByPath(context.Background(), testRealm, testGroup).
					Return(&keycloakapi.GroupRepresentation{Id: ptr.To(testGroupID)}, nil, nil)
			},
			wantID: testGroupID,
		},
		{
			name:  "plain name resolved with FindGroupByName",
			group: "developers",
			mockSetup: func(g *mocks.MockGroupsClient) {
				g.EXPECT().FindGroupByName(context.Background(), testRealm, "developers").
					Return(&keycloakapi.GroupRepresentation{Id: ptr.To(testGroupID)}, nil, nil)
			},
			wantID: testGroupID,
		},
		{
			name:  "group not found names the owning CRD",
			group: testGroup,
			mockSetup: func(g *mocks.MockGroupsClient) {
				g.EXPECT().GetGroupByPath(context.Background(), testRealm, testGroup).
					Return(nil, nil, keycloakapi.ErrNotFound)
			},
			wantErr: "create it with a KeycloakRealmGroup",
		},
		{
			name:  "nil group is not found",
			group: "developers",
			mockSetup: func(g *mocks.MockGroupsClient) {
				g.EXPECT().FindGroupByName(context.Background(), testRealm, "developers").
					Return(nil, nil, nil)
			},
			wantErr: "not found",
		},
		{
			name:  "api error",
			group: testGroup,
			mockSetup: func(g *mocks.MockGroupsClient) {
				g.EXPECT().GetGroupByPath(context.Background(), testRealm, testGroup).
					Return(nil, nil, errors.New("api error"))
			},
			wantErr: "unable to get group",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g := mocks.NewMockGroupsClient(t)
			tt.mockSetup(g)

			membership := &keycloakApi.KeycloakRealmGroupMembership{
				Spec: keycloakApi.KeycloakRealmGroupMembershipSpec{Group: tt.group},
			}
			mCtx := &MembershipContext{}

			err := NewResolveGroup(&keycloakapi.KeycloakClient{Groups: g}).
				Serve(context.Background(), membership, testRealm, mCtx)

			if tt.wantErr != "" {
				require.ErrorContains(t, err, tt.wantErr)

				return
			}

			require.NoError(t, err)
			assert.Equal(t, tt.wantID, mCtx.GroupID)
		})
	}
}

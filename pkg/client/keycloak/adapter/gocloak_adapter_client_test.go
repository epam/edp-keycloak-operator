package adapter

import (
	"context"
	"testing"

	"github.com/Nerzal/gocloak/v12"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestGoCloakAdapter_AddDefaultScopeToClient(t *testing.T) {
	t.Parallel()

	realm := "rl"
	clientID := "clid1"
	clientName := "cl"

	tests := map[string]struct {
		scopes                        []ClientScope
		onGetClientsResp              []*gocloak.Client
		onGetClientsErr               error
		onGetClientsDefaultScopesResp []*gocloak.ClientScope
		onGetClientsDefaultScopesErr  error
		onAddDefaultScopeToClient     error
		expectErr                     bool
	}{
		"success add scope": {
			scopes: []ClientScope{
				{
					ID:   "scid1",
					Name: "scn1",
				},
			},
			onGetClientsResp: []*gocloak.Client{
				{
					ID:       gocloak.StringP(clientID),
					ClientID: gocloak.StringP(clientName),
				},
				nil,
			},
			onGetClientsDefaultScopesResp: []*gocloak.ClientScope{
				{
					ID:   gocloak.StringP("scid2"),
					Name: gocloak.StringP("scn2"),
				},
			},
			expectErr: false,
		},
		"success scope already exists": {
			scopes: []ClientScope{
				{
					ID:   "scid1",
					Name: "scn1",
				},
			},
			onGetClientsResp: []*gocloak.Client{
				{
					ID:       gocloak.StringP(clientID),
					ClientID: gocloak.StringP(clientName),
				},
			},
			onGetClientsDefaultScopesResp: []*gocloak.ClientScope{
				{
					ID:   gocloak.StringP("scid1"),
					Name: gocloak.StringP("scn1"),
				},
			},
			expectErr: false,
		},
		"failed to get client": {
			scopes: []ClientScope{
				{
					ID:   "scid1",
					Name: "scn1",
				},
			},
			onGetClientsErr: errors.New("failed"),
			expectErr:       true,
		},
		"failed to get existing default client scope": {
			scopes: []ClientScope{
				{
					ID:   "scid1",
					Name: "scn1",
				},
			},
			onGetClientsResp: []*gocloak.Client{
				{
					ID:       gocloak.StringP(clientID),
					ClientID: gocloak.StringP(clientName),
				},
			},
			onGetClientsDefaultScopesErr: errors.New("failed"),
			expectErr:                    true,
		},
		"failed to add default client scope": {
			scopes: []ClientScope{
				{
					ID:   "scid1",
					Name: "scn1",
				},
			},
			onGetClientsResp: []*gocloak.Client{
				{
					ID:       gocloak.StringP(clientID),
					ClientID: gocloak.StringP(clientName),
				},
			},
			onGetClientsDefaultScopesResp: []*gocloak.ClientScope{
				{
					ID:   gocloak.StringP("scid2"),
					Name: gocloak.StringP("scn2"),
				},
			},
			onAddDefaultScopeToClient: errors.New("failed"),
			expectErr:                 false,
		},
	}

	for name, tc := range tests {
		tc := tc

		t.Run(name, func(t *testing.T) {
			t.Parallel()

			adapter, mockClient, _ := initAdapter(t)
			mockClient.On("GetClients", mock.Anything, "token", realm, gocloak.GetClientsParams{ClientID: gocloak.StringP("cl")}).Return(tc.onGetClientsResp, tc.onGetClientsErr)
			mockClient.On("GetClientsDefaultScopes", mock.Anything, "token", realm, clientID).Return(tc.onGetClientsDefaultScopesResp, tc.onGetClientsDefaultScopesErr).Maybe()
			mockClient.On("AddDefaultScopeToClient", mock.Anything, "token", realm, clientID, "scid1").Return(tc.onAddDefaultScopeToClient).Maybe()

			err := adapter.AddDefaultScopeToClient(context.Background(), realm, clientName, tc.scopes)
			if tc.expectErr {
				assert.Error(t, err)
				return
			}
			assert.NoError(t, err)
		})
	}
}

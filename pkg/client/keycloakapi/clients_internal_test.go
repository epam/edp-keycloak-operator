package keycloakapi

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"k8s.io/utils/ptr"

	"github.com/epam/edp-keycloak-operator/pkg/fakehttp"
)

func TestFindClientInList(t *testing.T) {
	tests := []struct {
		name     string
		clients  []ClientRepresentation
		clientID string
		wantID   string
	}{
		{
			name:     "match returns the client",
			clients:  []ClientRepresentation{{Id: ptr.To("u1"), ClientId: ptr.To("account")}},
			clientID: "account",
			wantID:   "u1",
		},
		{
			name:     "other clientId returns nil",
			clients:  []ClientRepresentation{{Id: ptr.To("u1"), ClientId: ptr.To("account")}},
			clientID: "x",
		},
		{
			name:     "empty list returns nil",
			clientID: "x",
		},
		{
			name:     "nil clientId is skipped",
			clients:  []ClientRepresentation{{Id: ptr.To("u1")}},
			clientID: "x",
		},
		{
			name:     "match without an id is not a usable client",
			clients:  []ClientRepresentation{{ClientId: ptr.To("x")}},
			clientID: "x",
		},
		{
			name: "an unusable match does not shadow a usable one",
			clients: []ClientRepresentation{
				{ClientId: ptr.To("x")},
				{Id: ptr.To("u2"), ClientId: ptr.To("x")},
			},
			clientID: "x",
			wantID:   "u2",
		},
		{
			name: "a client with another clientId does not shadow a match",
			clients: []ClientRepresentation{
				{Id: ptr.To("u1")},
				{Id: ptr.To("u2"), ClientId: ptr.To("x")},
			},
			clientID: "x",
			wantID:   "u2",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := findClientInList(tt.clients, tt.clientID)

			if tt.wantID == "" {
				assert.Nil(t, got)
				return
			}

			require.NotNil(t, got)
			assert.Equal(t, tt.wantID, ptr.Deref(got.Id, ""))
		})
	}
}

func TestClientsClient_GetClientByClientID_Match(t *testing.T) {
	t.Parallel()

	account := []ClientRepresentation{{Id: ptr.To("account-uuid"), ClientId: ptr.To("account")}}

	okServer := startClientsServer(200, account)
	t.Cleanup(okServer.Close)

	// Fake Keycloak answers 400 on /clients.
	failingServer := startClientsServer(400, map[string]string{"error": "invalid_request"})
	t.Cleanup(failingServer.Close)

	tests := []struct {
		name     string
		server   fakehttp.Server
		clientID string
		wantUUID string
		wantErr  require.ErrorAssertionFunc
	}{
		{
			name:     "exact match",
			server:   okServer,
			clientID: "account",
			wantUUID: "account-uuid",
			wantErr:  require.NoError,
		},
		{
			name:     "empty clientId is not found without a request",
			server:   failingServer,
			clientID: "",
			wantErr: func(t require.TestingT, err error, _ ...any) {
				require.ErrorIs(t, err, ErrNotFound)
			},
		},
		{
			name:     "whitespace clientId is not found without a request",
			server:   failingServer,
			clientID: " ",
			wantErr: func(t require.TestingT, err error, _ ...any) {
				require.ErrorIs(t, err, ErrNotFound)
			},
		},
		{
			name:     "case differs",
			server:   okServer,
			clientID: "Account",
			wantErr: func(t require.TestingT, err error, _ ...any) {
				require.ErrorIs(t, err, ErrNotFound)
			},
		},
		{
			name:     "list error is returned",
			server:   failingServer,
			clientID: "x",
			wantErr: func(t require.TestingT, err error, _ ...any) {
				require.Error(t, err)
				require.NotErrorIs(t, err, ErrNotFound)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			kClient, err := NewKeycloakClient(context.Background(), tt.server.GetURL(), "admin-cli", WithClientSecret("secret"))
			require.NoError(t, err)

			got, _, err := kClient.Clients.GetClientByClientID(context.Background(), "test-realm", tt.clientID)
			tt.wantErr(t, err)

			if tt.wantUUID != "" {
				require.NotNil(t, got)
				assert.Equal(t, tt.wantUUID, ptr.Deref(got.Id, ""))
			}
		})
	}
}

func startClientsServer(code int, body any) fakehttp.Server {
	return fakehttp.NewServerBuilder().
		AddKeycloakAuthResponders().
		AddJsonResponderWithCode(code, "/admin/realms/test-realm/clients", body).
		BuildAndStart()
}

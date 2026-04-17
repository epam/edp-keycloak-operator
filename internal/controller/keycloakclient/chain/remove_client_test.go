package chain

import (
	"context"
	"errors"
	"testing"

	"github.com/go-logr/logr"
	testifymock "github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"

	keycloakApi "github.com/epam/edp-keycloak-operator/api/v1"
	keycloakapi "github.com/epam/edp-keycloak-operator/pkg/client/keycloakapi"
	keycloakv2Mocks "github.com/epam/edp-keycloak-operator/pkg/client/keycloakapi/mocks"
	"github.com/epam/edp-keycloak-operator/pkg/objectmeta"
)

func TestRemoveClient_Serve(t *testing.T) {
	tests := []struct {
		name           string
		keycloakClient *keycloakApi.KeycloakClient
		kClient        func(t *testing.T) *keycloakapi.APIClient
		realmName      string
		wantErr        require.ErrorAssertionFunc
	}{
		{
			name: "preserve resources on deletion - skip",
			keycloakClient: &keycloakApi.KeycloakClient{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-client",
					Namespace: "default",
					Annotations: map[string]string{
						objectmeta.PreserveResourcesOnDeletionAnnotation: "true",
					},
				},
				Status: keycloakApi.KeycloakClientStatus{
					ClientID: "client-uuid",
				},
			},
			kClient: func(t *testing.T) *keycloakapi.APIClient {
				return &keycloakapi.APIClient{
					Clients: keycloakv2Mocks.NewMockClientsClient(t),
				}
			},
			realmName: "test-realm",
			wantErr:   require.NoError,
		},
		{
			name: "empty client ID in status - skip",
			keycloakClient: &keycloakApi.KeycloakClient{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-client",
					Namespace: "default",
				},
				Status: keycloakApi.KeycloakClientStatus{
					ClientID: "",
				},
			},
			kClient: func(t *testing.T) *keycloakapi.APIClient {
				return &keycloakapi.APIClient{
					Clients: keycloakv2Mocks.NewMockClientsClient(t),
				}
			},
			realmName: "test-realm",
			wantErr:   require.NoError,
		},
		{
			name: "delete client successfully",
			keycloakClient: &keycloakApi.KeycloakClient{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-client",
					Namespace: "default",
				},
				Status: keycloakApi.KeycloakClientStatus{
					ClientID: "client-uuid",
				},
			},
			kClient: func(t *testing.T) *keycloakapi.APIClient {
				clientsMock := keycloakv2Mocks.NewMockClientsClient(t)
				clientsMock.On("DeleteClient", testifymock.Anything, "test-realm", "client-uuid").
					Return((*keycloakapi.Response)(nil), nil)

				return &keycloakapi.APIClient{Clients: clientsMock}
			},
			realmName: "test-realm",
			wantErr:   require.NoError,
		},
		{
			name: "client not found in keycloak - skip",
			keycloakClient: &keycloakApi.KeycloakClient{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-client",
					Namespace: "default",
				},
				Status: keycloakApi.KeycloakClientStatus{
					ClientID: "client-uuid",
				},
			},
			kClient: func(t *testing.T) *keycloakapi.APIClient {
				clientsMock := keycloakv2Mocks.NewMockClientsClient(t)
				clientsMock.On("DeleteClient", testifymock.Anything, "test-realm", "client-uuid").
					Return((*keycloakapi.Response)(nil), keycloakapi.ErrNotFound)

				return &keycloakapi.APIClient{Clients: clientsMock}
			},
			realmName: "test-realm",
			wantErr:   require.NoError,
		},
		{
			name: "delete client fails",
			keycloakClient: &keycloakApi.KeycloakClient{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-client",
					Namespace: "default",
				},
				Status: keycloakApi.KeycloakClientStatus{
					ClientID: "client-uuid",
				},
			},
			kClient: func(t *testing.T) *keycloakapi.APIClient {
				clientsMock := keycloakv2Mocks.NewMockClientsClient(t)
				clientsMock.On("DeleteClient", testifymock.Anything, "test-realm", "client-uuid").
					Return((*keycloakapi.Response)(nil), errors.New("connection refused"))

				return &keycloakapi.APIClient{Clients: clientsMock}
			},
			realmName: "test-realm",
			wantErr:   require.Error,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := NewRemoveClient(tt.kClient(t))
			err := h.Serve(
				ctrl.LoggerInto(context.Background(), logr.Discard()),
				tt.keycloakClient,
				tt.realmName,
			)
			tt.wantErr(t, err)
		})
	}
}

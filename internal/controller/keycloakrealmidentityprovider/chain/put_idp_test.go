package chain

import (
	"context"
	"fmt"
	"testing"

	"github.com/go-logr/logr"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"
	ctrl "sigs.k8s.io/controller-runtime"

	keycloakApi "github.com/epam/edp-keycloak-operator/api/v1"
	keycloakv2 "github.com/epam/edp-keycloak-operator/pkg/client/keycloakv2"
	keycloakv2mocks "github.com/epam/edp-keycloak-operator/pkg/client/keycloakv2/mocks"
	secretrefmocks "github.com/epam/edp-keycloak-operator/pkg/secretref/mocks"
)

func TestPutIDP_Serve(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		idp       *keycloakApi.KeycloakRealmIdentityProvider
		idpClient func(t *testing.T) keycloakv2.IdentityProvidersClient
		secretRef func(t *testing.T) refClient
		wantErr   require.ErrorAssertionFunc
	}{
		{
			name: "create new identity provider",
			idp: &keycloakApi.KeycloakRealmIdentityProvider{
				ObjectMeta: metav1.ObjectMeta{Namespace: "default"},
				Spec: keycloakApi.KeycloakRealmIdentityProviderSpec{
					Alias:      "test-idp",
					ProviderID: "github",
					Enabled:    true,
					Config: map[string]string{
						"clientId":     "test-client",
						"clientSecret": "$secret-name:secret-key",
					},
				},
			},
			idpClient: func(t *testing.T) keycloakv2.IdentityProvidersClient {
				m := keycloakv2mocks.NewMockIdentityProvidersClient(t)
				m.On("GetIdentityProvider", mock.Anything, "realm", "test-idp").
					Return((*keycloakv2.IdentityProviderRepresentation)(nil), (*keycloakv2.Response)(nil), &keycloakv2.ApiError{Code: 404}).Once()
				m.On("CreateIdentityProvider", mock.Anything, "realm", mock.Anything).
					Return((*keycloakv2.Response)(nil), nil).Once()
				return m
			},
			secretRef: func(t *testing.T) refClient {
				m := secretrefmocks.NewMockRefClient(t)
				m.On("MapConfigSecretsRefs", mock.Anything, mock.Anything, "default").Return(nil)
				return m
			},
			wantErr: require.NoError,
		},
		{
			name: "update existing identity provider",
			idp: &keycloakApi.KeycloakRealmIdentityProvider{
				ObjectMeta: metav1.ObjectMeta{Namespace: "default"},
				Spec: keycloakApi.KeycloakRealmIdentityProviderSpec{
					Alias:      "test-idp",
					ProviderID: "github",
					Enabled:    true,
					Config:     map[string]string{"clientId": "test-client"},
				},
			},
			idpClient: func(t *testing.T) keycloakv2.IdentityProvidersClient {
				m := keycloakv2mocks.NewMockIdentityProvidersClient(t)
				m.On("GetIdentityProvider", mock.Anything, "realm", "test-idp").
					Return(&keycloakv2.IdentityProviderRepresentation{Alias: ptr.To("test-idp")}, (*keycloakv2.Response)(nil), nil).Once()
				m.On("UpdateIdentityProvider", mock.Anything, "realm", "test-idp", mock.Anything).
					Return((*keycloakv2.Response)(nil), nil).Once()
				return m
			},
			secretRef: func(t *testing.T) refClient {
				m := secretrefmocks.NewMockRefClient(t)
				m.On("MapConfigSecretsRefs", mock.Anything, mock.Anything, "default").Return(nil)
				return m
			},
			wantErr: require.NoError,
		},
		{
			name: "secret ref mapping fails",
			idp: &keycloakApi.KeycloakRealmIdentityProvider{
				ObjectMeta: metav1.ObjectMeta{Namespace: "default"},
				Spec: keycloakApi.KeycloakRealmIdentityProviderSpec{
					Alias:      "test-idp",
					ProviderID: "github",
					Config:     map[string]string{"clientSecret": "$secret:key"},
				},
			},
			idpClient: func(t *testing.T) keycloakv2.IdentityProvidersClient {
				return keycloakv2mocks.NewMockIdentityProvidersClient(t)
			},
			secretRef: func(t *testing.T) refClient {
				m := secretrefmocks.NewMockRefClient(t)
				m.On("MapConfigSecretsRefs", mock.Anything, mock.Anything, "default").
					Return(fmt.Errorf("secret not found"))
				return m
			},
			wantErr: require.Error,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			h := NewPutIDP(tt.idpClient(t), tt.secretRef(t))
			err := h.Serve(
				ctrl.LoggerInto(context.Background(), logr.Discard()),
				tt.idp,
				"realm",
			)
			tt.wantErr(t, err)
		})
	}
}

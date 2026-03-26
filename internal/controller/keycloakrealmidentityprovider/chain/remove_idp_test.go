package chain

import (
	"context"
	"fmt"
	"testing"

	"github.com/go-logr/logr"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"

	keycloakApi "github.com/epam/edp-keycloak-operator/api/v1"
	keycloakv2 "github.com/epam/edp-keycloak-operator/pkg/client/keycloakv2"
	keycloakv2mocks "github.com/epam/edp-keycloak-operator/pkg/client/keycloakv2/mocks"
	"github.com/epam/edp-keycloak-operator/pkg/objectmeta"
)

func TestRemoveIDP_Serve(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		idp     *keycloakApi.KeycloakRealmIdentityProvider
		kClient func(t *testing.T) *keycloakv2.KeycloakClient
		wantErr require.ErrorAssertionFunc
	}{
		{
			name: "successfully delete identity provider",
			idp: &keycloakApi.KeycloakRealmIdentityProvider{
				Spec: keycloakApi.KeycloakRealmIdentityProviderSpec{
					Alias: "test-idp",
				},
			},
			kClient: func(t *testing.T) *keycloakv2.KeycloakClient {
				m := keycloakv2mocks.NewMockIdentityProvidersClient(t)
				m.On("DeleteIdentityProvider", mock.Anything, "realm", "test-idp").
					Return((*keycloakv2.Response)(nil), nil).Once()
				return &keycloakv2.KeycloakClient{IdentityProviders: m}
			},
			wantErr: require.NoError,
		},
		{
			name: "identity provider not found - skip",
			idp: &keycloakApi.KeycloakRealmIdentityProvider{
				Spec: keycloakApi.KeycloakRealmIdentityProviderSpec{
					Alias: "test-idp",
				},
			},
			kClient: func(t *testing.T) *keycloakv2.KeycloakClient {
				m := keycloakv2mocks.NewMockIdentityProvidersClient(t)
				m.On("DeleteIdentityProvider", mock.Anything, "realm", "test-idp").
					Return((*keycloakv2.Response)(nil), &keycloakv2.ApiError{Code: 404}).Once()
				return &keycloakv2.KeycloakClient{IdentityProviders: m}
			},
			wantErr: require.NoError,
		},
		{
			name: "delete fails",
			idp: &keycloakApi.KeycloakRealmIdentityProvider{
				Spec: keycloakApi.KeycloakRealmIdentityProviderSpec{
					Alias: "test-idp",
				},
			},
			kClient: func(t *testing.T) *keycloakv2.KeycloakClient {
				m := keycloakv2mocks.NewMockIdentityProvidersClient(t)
				m.On("DeleteIdentityProvider", mock.Anything, "realm", "test-idp").
					Return((*keycloakv2.Response)(nil), fmt.Errorf("api error")).Once()
				return &keycloakv2.KeycloakClient{IdentityProviders: m}
			},
			wantErr: require.Error,
		},
		{
			name: "preserve resources on deletion",
			idp: &keycloakApi.KeycloakRealmIdentityProvider{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						objectmeta.PreserveResourcesOnDeletionAnnotation: "true",
					},
				},
				Spec: keycloakApi.KeycloakRealmIdentityProviderSpec{
					Alias: "test-idp",
				},
			},
			kClient: func(t *testing.T) *keycloakv2.KeycloakClient {
				return &keycloakv2.KeycloakClient{
					IdentityProviders: keycloakv2mocks.NewMockIdentityProvidersClient(t),
				}
			},
			wantErr: require.NoError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			h := NewRemoveIDP(tt.kClient(t))
			err := h.Serve(
				ctrl.LoggerInto(context.Background(), logr.Discard()),
				tt.idp,
				"realm",
			)
			tt.wantErr(t, err)
		})
	}
}

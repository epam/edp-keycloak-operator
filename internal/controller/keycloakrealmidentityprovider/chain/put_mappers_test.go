package chain

import (
	"context"
	"fmt"
	"testing"

	"github.com/go-logr/logr"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"k8s.io/utils/ptr"
	ctrl "sigs.k8s.io/controller-runtime"

	keycloakApi "github.com/epam/edp-keycloak-operator/api/v1"
	"github.com/epam/edp-keycloak-operator/pkg/client/keycloakapi"
	keycloakapimocks "github.com/epam/edp-keycloak-operator/pkg/client/keycloakapi/mocks"
)

func TestPutIDPMappers_Serve(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		idp       *keycloakApi.KeycloakRealmIdentityProvider
		idpClient func(t *testing.T) keycloakapi.IdentityProvidersClient
		wantErr   require.ErrorAssertionFunc
	}{
		{
			name: "no mappers specified",
			idp: &keycloakApi.KeycloakRealmIdentityProvider{
				Spec: keycloakApi.KeycloakRealmIdentityProviderSpec{
					Alias: "test-idp",
				},
			},
			idpClient: func(t *testing.T) keycloakapi.IdentityProvidersClient {
				return keycloakapimocks.NewMockIdentityProvidersClient(t)
			},
			wantErr: require.NoError,
		},
		{
			name: "sync mappers - delete old and create new",
			idp: &keycloakApi.KeycloakRealmIdentityProvider{
				Spec: keycloakApi.KeycloakRealmIdentityProviderSpec{
					Alias: "test-idp",
					Mappers: []keycloakApi.IdentityProviderMapper{
						{
							Name:                   "new-mapper",
							IdentityProviderMapper: "hardcoded-attribute-idp-mapper",
							Config:                 map[string]string{"attribute": "test"},
						},
					},
				},
			},
			idpClient: func(t *testing.T) keycloakapi.IdentityProvidersClient {
				m := keycloakapimocks.NewMockIdentityProvidersClient(t)
				m.On("GetIDPMappers", mock.Anything, "realm", "test-idp").
					Return([]keycloakapi.IdentityProviderMapperRepresentation{
						{Id: ptr.To("old-mapper-id"), Name: ptr.To("old-mapper")},
					}, (*keycloakapi.Response)(nil), nil).Once()
				m.On("DeleteIDPMapper", mock.Anything, "realm", "test-idp", "old-mapper-id").
					Return((*keycloakapi.Response)(nil), nil).Once()
				m.On("CreateIDPMapper", mock.Anything, "realm", "test-idp", mock.Anything).
					Return((*keycloakapi.Response)(nil), nil).Once()
				return m
			},
			wantErr: require.NoError,
		},
		{
			name: "get mappers fails",
			idp: &keycloakApi.KeycloakRealmIdentityProvider{
				Spec: keycloakApi.KeycloakRealmIdentityProviderSpec{
					Alias: "test-idp",
					Mappers: []keycloakApi.IdentityProviderMapper{
						{Name: "mapper"},
					},
				},
			},
			idpClient: func(t *testing.T) keycloakapi.IdentityProvidersClient {
				m := keycloakapimocks.NewMockIdentityProvidersClient(t)
				m.On("GetIDPMappers", mock.Anything, "realm", "test-idp").
					Return([]keycloakapi.IdentityProviderMapperRepresentation(nil), (*keycloakapi.Response)(nil), fmt.Errorf("api error")).Once()
				return m
			},
			wantErr: require.Error,
		},
		{
			name: "mapper uses idp alias when not specified",
			idp: &keycloakApi.KeycloakRealmIdentityProvider{
				Spec: keycloakApi.KeycloakRealmIdentityProviderSpec{
					Alias: "test-idp",
					Mappers: []keycloakApi.IdentityProviderMapper{
						{
							Name:                   "mapper-no-alias",
							IdentityProviderMapper: "hardcoded-attribute-idp-mapper",
						},
					},
				},
			},
			idpClient: func(t *testing.T) keycloakapi.IdentityProvidersClient {
				m := keycloakapimocks.NewMockIdentityProvidersClient(t)
				m.On("GetIDPMappers", mock.Anything, "realm", "test-idp").
					Return([]keycloakapi.IdentityProviderMapperRepresentation{}, (*keycloakapi.Response)(nil), nil).Once()
				m.On("CreateIDPMapper", mock.Anything, "realm", "test-idp", mock.MatchedBy(func(mapper keycloakapi.IdentityProviderMapperRepresentation) bool {
					return mapper.IdentityProviderAlias != nil && *mapper.IdentityProviderAlias == "test-idp"
				})).Return((*keycloakapi.Response)(nil), nil).Once()
				return m
			},
			wantErr: require.NoError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			h := NewPutIDPMappers(tt.idpClient(t))
			err := h.Serve(
				ctrl.LoggerInto(context.Background(), logr.Discard()),
				tt.idp,
				"realm",
			)
			tt.wantErr(t, err)
		})
	}
}

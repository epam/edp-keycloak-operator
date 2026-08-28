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
	"github.com/epam/edp-keycloak-operator/pkg/client/keycloakapi"
	keycloakapimocks "github.com/epam/edp-keycloak-operator/pkg/client/keycloakapi/mocks"
)

const (
	testIDPAlias   = "test-idp"
	testMapperType = "hardcoded-attribute-idp-mapper"
)

func attrMapperSpec(name, attrValue string) keycloakApi.IdentityProviderMapper {
	return keycloakApi.IdentityProviderMapper{
		Name:                   name,
		IdentityProviderMapper: testMapperType,
		Config:                 map[string]string{"attribute": attrValue},
	}
}

func attrMapperRepr(id, name, attrValue string) keycloakapi.IdentityProviderMapperRepresentation {
	return keycloakapi.IdentityProviderMapperRepresentation{
		Id:                     ptr.To(id),
		Name:                   ptr.To(name),
		IdentityProviderAlias:  ptr.To(testIDPAlias),
		IdentityProviderMapper: ptr.To(testMapperType),
		Config:                 &map[string]string{"attribute": attrValue},
	}
}

func TestPutIDPMappers_Serve(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		idp       *keycloakApi.KeycloakRealmIdentityProvider
		idpClient func(t *testing.T) keycloakapi.IdentityProvidersClient
		wantErr   require.ErrorAssertionFunc
	}{
		{
			name: "in sync - no create, update or delete",
			idp: &keycloakApi.KeycloakRealmIdentityProvider{
				ObjectMeta: metav1.ObjectMeta{Generation: 3},
				Spec: keycloakApi.KeycloakRealmIdentityProviderSpec{
					Alias:   testIDPAlias,
					Mappers: []keycloakApi.IdentityProviderMapper{attrMapperSpec("mapper1", "test")},
				},
				Status: keycloakApi.KeycloakRealmIdentityProviderStatus{ObservedGeneration: 3},
			},
			idpClient: func(t *testing.T) keycloakapi.IdentityProvidersClient {
				// Keycloak-injected key ("server.injected") must not trip the subset comparison.
				existing := attrMapperRepr("mapper1-id", "mapper1", "test")
				(*existing.Config)["server.injected"] = "default"

				m := keycloakapimocks.NewMockIdentityProvidersClient(t)
				m.On("GetIDPMappers", mock.Anything, "realm", testIDPAlias).
					Return([]keycloakapi.IdentityProviderMapperRepresentation{existing}, (*keycloakapi.Response)(nil), nil).Once()
				return m
			},
			wantErr: require.NoError,
		},
		{
			name: "config drift - update in place preserving id",
			idp: &keycloakApi.KeycloakRealmIdentityProvider{
				Spec: keycloakApi.KeycloakRealmIdentityProviderSpec{
					Alias:   testIDPAlias,
					Mappers: []keycloakApi.IdentityProviderMapper{attrMapperSpec("mapper1", "changed")},
				},
			},
			idpClient: func(t *testing.T) keycloakapi.IdentityProvidersClient {
				m := keycloakapimocks.NewMockIdentityProvidersClient(t)
				m.On("GetIDPMappers", mock.Anything, "realm", testIDPAlias).
					Return([]keycloakapi.IdentityProviderMapperRepresentation{
						attrMapperRepr("mapper1-id", "mapper1", "test"),
					}, (*keycloakapi.Response)(nil), nil).Once()
				m.On("UpdateIDPMapper", mock.Anything, "realm", testIDPAlias, "mapper1-id",
					mock.MatchedBy(func(mapper keycloakapi.IdentityProviderMapperRepresentation) bool {
						return mapper.Id != nil && *mapper.Id == "mapper1-id"
					})).
					Return((*keycloakapi.Response)(nil), nil).Once()
				return m
			},
			wantErr: require.NoError,
		},
		{
			name: "new mapper is created",
			idp: &keycloakApi.KeycloakRealmIdentityProvider{
				Spec: keycloakApi.KeycloakRealmIdentityProviderSpec{
					Alias:   testIDPAlias,
					Mappers: []keycloakApi.IdentityProviderMapper{attrMapperSpec("new-mapper", "test")},
				},
			},
			idpClient: func(t *testing.T) keycloakapi.IdentityProvidersClient {
				m := keycloakapimocks.NewMockIdentityProvidersClient(t)
				m.On("GetIDPMappers", mock.Anything, "realm", testIDPAlias).
					Return([]keycloakapi.IdentityProviderMapperRepresentation{}, (*keycloakapi.Response)(nil), nil).Once()
				m.On("CreateIDPMapper", mock.Anything, "realm", testIDPAlias, mock.Anything).
					Return((*keycloakapi.Response)(nil), nil).Once()
				return m
			},
			wantErr: require.NoError,
		},
		{
			name: "mapper removed from spec is deleted",
			idp: &keycloakApi.KeycloakRealmIdentityProvider{
				Spec: keycloakApi.KeycloakRealmIdentityProviderSpec{
					Alias:   testIDPAlias,
					Mappers: []keycloakApi.IdentityProviderMapper{attrMapperSpec("kept-mapper", "kept")},
				},
			},
			idpClient: func(t *testing.T) keycloakapi.IdentityProvidersClient {
				m := keycloakapimocks.NewMockIdentityProvidersClient(t)
				m.On("GetIDPMappers", mock.Anything, "realm", testIDPAlias).
					Return([]keycloakapi.IdentityProviderMapperRepresentation{
						{Id: ptr.To("old-mapper-id"), Name: ptr.To("old-mapper")},
						attrMapperRepr("kept-mapper-id", "kept-mapper", "kept"),
					}, (*keycloakapi.Response)(nil), nil).Once()
				m.On("DeleteIDPMapper", mock.Anything, "realm", testIDPAlias, "old-mapper-id").
					Return((*keycloakapi.Response)(nil), nil).Once()
				return m
			},
			wantErr: require.NoError,
		},
		{
			name: "explicit empty mappers list deletes all existing mappers",
			idp: &keycloakApi.KeycloakRealmIdentityProvider{
				Spec: keycloakApi.KeycloakRealmIdentityProviderSpec{
					Alias:   testIDPAlias,
					Mappers: []keycloakApi.IdentityProviderMapper{},
				},
			},
			idpClient: func(t *testing.T) keycloakapi.IdentityProvidersClient {
				m := keycloakapimocks.NewMockIdentityProvidersClient(t)
				m.On("GetIDPMappers", mock.Anything, "realm", testIDPAlias).
					Return([]keycloakapi.IdentityProviderMapperRepresentation{
						{Id: ptr.To("mapper1-id"), Name: ptr.To("mapper1")},
						{Id: ptr.To("mapper2-id"), Name: ptr.To("mapper2")},
					}, (*keycloakapi.Response)(nil), nil).Once()
				m.On("DeleteIDPMapper", mock.Anything, "realm", testIDPAlias, "mapper1-id").
					Return((*keycloakapi.Response)(nil), nil).Once()
				m.On("DeleteIDPMapper", mock.Anything, "realm", testIDPAlias, "mapper2-id").
					Return((*keycloakapi.Response)(nil), nil).Once()
				return m
			},
			wantErr: require.NoError,
		},
		{
			name: "nil mappers field leaves existing mappers untouched",
			idp: &keycloakApi.KeycloakRealmIdentityProvider{
				Spec: keycloakApi.KeycloakRealmIdentityProviderSpec{
					Alias: testIDPAlias,
				},
			},
			idpClient: func(t *testing.T) keycloakapi.IdentityProvidersClient {
				// No expectations registered: an unexpected call panics the mock.
				return keycloakapimocks.NewMockIdentityProvidersClient(t)
			},
			wantErr: require.NoError,
		},
		{
			name: "empty mapper name is rejected",
			idp: &keycloakApi.KeycloakRealmIdentityProvider{
				Spec: keycloakApi.KeycloakRealmIdentityProviderSpec{
					Alias:   testIDPAlias,
					Mappers: []keycloakApi.IdentityProviderMapper{attrMapperSpec("", "test")},
				},
			},
			idpClient: func(t *testing.T) keycloakapi.IdentityProvidersClient {
				return keycloakapimocks.NewMockIdentityProvidersClient(t)
			},
			wantErr: require.Error,
		},
		{
			name: "duplicate mapper name is rejected",
			idp: &keycloakApi.KeycloakRealmIdentityProvider{
				Spec: keycloakApi.KeycloakRealmIdentityProviderSpec{
					Alias: testIDPAlias,
					Mappers: []keycloakApi.IdentityProviderMapper{
						attrMapperSpec("dup", "test"),
						attrMapperSpec("dup", "other"),
					},
				},
			},
			idpClient: func(t *testing.T) keycloakapi.IdentityProvidersClient {
				return keycloakapimocks.NewMockIdentityProvidersClient(t)
			},
			wantErr: require.Error,
		},
		{
			name: "generation bump forces update even without drift",
			idp: &keycloakApi.KeycloakRealmIdentityProvider{
				ObjectMeta: metav1.ObjectMeta{Generation: 4},
				Spec: keycloakApi.KeycloakRealmIdentityProviderSpec{
					Alias:   testIDPAlias,
					Mappers: []keycloakApi.IdentityProviderMapper{attrMapperSpec("mapper1", "test")},
				},
				Status: keycloakApi.KeycloakRealmIdentityProviderStatus{ObservedGeneration: 3},
			},
			idpClient: func(t *testing.T) keycloakapi.IdentityProvidersClient {
				m := keycloakapimocks.NewMockIdentityProvidersClient(t)
				m.On("GetIDPMappers", mock.Anything, "realm", testIDPAlias).
					Return([]keycloakapi.IdentityProviderMapperRepresentation{
						attrMapperRepr("mapper1-id", "mapper1", "test"),
					}, (*keycloakapi.Response)(nil), nil).Once()
				m.On("UpdateIDPMapper", mock.Anything, "realm", testIDPAlias, "mapper1-id", mock.Anything).
					Return((*keycloakapi.Response)(nil), nil).Once()
				return m
			},
			wantErr: require.NoError,
		},
		{
			name: "get mappers fails",
			idp: &keycloakApi.KeycloakRealmIdentityProvider{
				Spec: keycloakApi.KeycloakRealmIdentityProviderSpec{
					Alias:   testIDPAlias,
					Mappers: []keycloakApi.IdentityProviderMapper{{Name: "mapper"}},
				},
			},
			idpClient: func(t *testing.T) keycloakapi.IdentityProvidersClient {
				m := keycloakapimocks.NewMockIdentityProvidersClient(t)
				m.On("GetIDPMappers", mock.Anything, "realm", testIDPAlias).
					Return([]keycloakapi.IdentityProviderMapperRepresentation(nil), (*keycloakapi.Response)(nil), fmt.Errorf("api error")).Once()
				return m
			},
			wantErr: require.Error,
		},
		{
			name: "mapper uses idp alias when not specified",
			idp: &keycloakApi.KeycloakRealmIdentityProvider{
				Spec: keycloakApi.KeycloakRealmIdentityProviderSpec{
					Alias: testIDPAlias,
					Mappers: []keycloakApi.IdentityProviderMapper{
						{Name: "mapper-no-alias", IdentityProviderMapper: testMapperType},
					},
				},
			},
			idpClient: func(t *testing.T) keycloakapi.IdentityProvidersClient {
				m := keycloakapimocks.NewMockIdentityProvidersClient(t)
				m.On("GetIDPMappers", mock.Anything, "realm", testIDPAlias).
					Return([]keycloakapi.IdentityProviderMapperRepresentation{}, (*keycloakapi.Response)(nil), nil).Once()
				m.On("CreateIDPMapper", mock.Anything, "realm", testIDPAlias, mock.MatchedBy(func(mapper keycloakapi.IdentityProviderMapperRepresentation) bool {
					return mapper.IdentityProviderAlias != nil && *mapper.IdentityProviderAlias == testIDPAlias
				})).Return((*keycloakapi.Response)(nil), nil).Once()
				return m
			},
			wantErr: require.NoError,
		},
		{
			name: "existing mapper with nil name and nil id are handled safely",
			idp: &keycloakApi.KeycloakRealmIdentityProvider{
				Spec: keycloakApi.KeycloakRealmIdentityProviderSpec{
					Alias:   testIDPAlias,
					Mappers: []keycloakApi.IdentityProviderMapper{},
				},
			},
			idpClient: func(t *testing.T) keycloakapi.IdentityProvidersClient {
				m := keycloakapimocks.NewMockIdentityProvidersClient(t)
				m.On("GetIDPMappers", mock.Anything, "realm", testIDPAlias).
					Return([]keycloakapi.IdentityProviderMapperRepresentation{
						{Id: nil, Name: ptr.To("no-id-mapper")},
						{Id: ptr.To("no-name-mapper-id"), Name: nil},
					}, (*keycloakapi.Response)(nil), nil).Once()
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

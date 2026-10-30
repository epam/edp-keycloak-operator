package chain

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"k8s.io/utils/ptr"

	keycloakApi "github.com/epam/edp-keycloak-operator/api/v1alpha1"
	"github.com/epam/edp-keycloak-operator/pkg/client/keycloakapi"
	keycloakapimocks "github.com/epam/edp-keycloak-operator/pkg/client/keycloakapi/mocks"
)

func TestMakeChain_Order(t *testing.T) {
	t.Parallel()

	c, ok := MakeChain(&keycloakapi.KeycloakClient{}).(*chain)
	require.True(t, ok)

	want := []Handler{
		&ValidateDomainSettings{},
		&ValidateServerVersion{},
		&CreateOrganization{},
		&ProcessIdentityProviders{},
		&ProcessDomainSettings{},
	}

	require.Equal(t, handlerTypes(want), handlerTypes(c.handlers))
}

func handlerTypes(handlers []Handler) []string {
	types := make([]string, 0, len(handlers))
	for _, h := range handlers {
		types = append(types, fmt.Sprintf("%T", h))
	}

	return types
}

func TestMakeChain_GatedFieldFailsBeforeAnyOrganizationCall(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		spec      keycloakApi.KeycloakOrganizationSpec
		wantError string
	}{
		{
			name: "domain alias",
			spec: keycloakApi.KeycloakOrganizationSpec{
				Name: "Org", Alias: "org",
				DomainsV2:         []keycloakApi.OrgDomain{{Name: "d.example", IdentityProviderAlias: ptr.To(testIdPAlias)}},
				IdentityProviders: ptr.To([]keycloakApi.OrgIdentityProvider{{Alias: testIdPAlias}}),
			},
			wantError: "identityProviderAlias requires Keycloak 26.8 or later, server version is 26.7.5",
		},
		{
			name: "link autoMembership",
			spec: keycloakApi.KeycloakOrganizationSpec{
				Name: "Org", Alias: "org",
				DomainsV2:         []keycloakApi.OrgDomain{{Name: "d.example", Verified: ptr.To(true)}},
				IdentityProviders: ptr.To([]keycloakApi.OrgIdentityProvider{{Alias: testIdPAlias, AutoMembership: ptr.To(true)}}),
			},
			wantError: "autoMembership requires Keycloak 26.8 or later, server version is 26.7.5",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			server := keycloakapimocks.NewMockServerInfoClient(t)
			server.On("GetServerInfo", mock.Anything).Return(serverInfo("26.7.5"), nil).Once()

			kc := &keycloakapi.KeycloakClient{
				Server:        server,
				Organizations: keycloakapimocks.NewMockOrganizationsClient(t),
			}

			org := &keycloakApi.KeycloakOrganization{Spec: tt.spec}

			err := MakeChain(kc).Serve(context.Background(), org, testRealmName)
			require.ErrorContains(t, err, tt.wantError)
		})
	}
}

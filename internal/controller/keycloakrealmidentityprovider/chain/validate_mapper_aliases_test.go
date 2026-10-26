package chain

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	keycloakApi "github.com/epam/edp-keycloak-operator/api/v1"
)

func TestValidateMapperAliases_Serve(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		mappers  []keycloakApi.IdentityProviderMapper
		wantErr  bool
		contains []string
		excludes []string
	}{
		{name: "mappers omitted", mappers: nil},
		{
			name:    "alias empty",
			mappers: []keycloakApi.IdentityProviderMapper{{Name: "m1"}},
		},
		{
			name: "alias equals spec.alias",
			//nolint:staticcheck // covers the deprecated identityProviderAlias
			mappers: []keycloakApi.IdentityProviderMapper{{Name: "m1", IdentityProviderAlias: testIDPAlias}},
		},
		{
			name: "alias of another identity provider",
			//nolint:staticcheck // covers the deprecated identityProviderAlias
			mappers: []keycloakApi.IdentityProviderMapper{{Name: "m1", IdentityProviderAlias: "other"}},
			wantErr: true,
			contains: []string{
				`mappers[0] "m1"`, `"other"`, `"` + testIDPAlias + `"`, "KeycloakRealmIdentityProvider",
			},
		},
		{
			name: "alias differs in case",
			//nolint:staticcheck // covers the deprecated identityProviderAlias
			mappers: []keycloakApi.IdentityProviderMapper{{Name: "m1", IdentityProviderAlias: "Test-IDP"}},
			wantErr: true,
			contains: []string{
				`mappers[0] "m1"`, `"Test-IDP"`, `"` + testIDPAlias + `"`,
			},
		},
		{
			name: "first offender only",
			mappers: []keycloakApi.IdentityProviderMapper{
				{Name: "m1"},
				//nolint:staticcheck // covers the deprecated identityProviderAlias
				{Name: "m2", IdentityProviderAlias: "x"},
				//nolint:staticcheck // covers the deprecated identityProviderAlias
				{Name: "m3", IdentityProviderAlias: testIDPAlias},
				//nolint:staticcheck // covers the deprecated identityProviderAlias
				{Name: "m4", IdentityProviderAlias: "y"},
			},
			wantErr:  true,
			contains: []string{`mappers[1] "m2"`, `"x"`, `"` + testIDPAlias + `"`, "KeycloakRealmIdentityProvider"},
			excludes: []string{`"y"`, "mappers[3]"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			idp := &keycloakApi.KeycloakRealmIdentityProvider{
				Spec: keycloakApi.KeycloakRealmIdentityProviderSpec{Alias: testIDPAlias, Mappers: tt.mappers},
			}

			err := NewValidateMapperAliases().Serve(context.Background(), idp, "realm")
			if !tt.wantErr {
				require.NoError(t, err)
				return
			}

			require.Error(t, err)

			for _, s := range tt.contains {
				require.Contains(t, err.Error(), s)
			}

			for _, s := range tt.excludes {
				require.NotContains(t, err.Error(), s)
			}
		})
	}
}

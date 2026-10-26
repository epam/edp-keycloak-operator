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
		alias    string
		mappers  []keycloakApi.IdentityProviderMapper
		wantErr  bool
		contains []string
		excludes []string
	}{
		{name: "mappers omitted", alias: testIDPAlias, mappers: nil},
		{name: "empty mappers list", alias: testIDPAlias, mappers: []keycloakApi.IdentityProviderMapper{}},
		{
			name:    "alias empty",
			alias:   testIDPAlias,
			mappers: []keycloakApi.IdentityProviderMapper{{Name: "m1"}},
		},
		{
			name:    "alias equals spec.alias",
			alias:   testIDPAlias,
			mappers: []keycloakApi.IdentityProviderMapper{{Name: "m1", IdentityProviderAlias: testIDPAlias}},
		},
		{
			name:    "alias of another identity provider",
			alias:   testIDPAlias,
			mappers: []keycloakApi.IdentityProviderMapper{{Name: "m1", IdentityProviderAlias: "other"}},
			wantErr: true,
			contains: []string{
				`mappers[0] "m1"`, `"other"`, `"` + testIDPAlias + `"`, "KeycloakRealmIdentityProvider",
			},
		},
		{
			name:    "alias differs in case",
			alias:   testIDPAlias,
			mappers: []keycloakApi.IdentityProviderMapper{{Name: "m1", IdentityProviderAlias: "Test-IDP"}},
			wantErr: true,
			contains: []string{
				`mappers[0] "m1"`, `"Test-IDP"`, `"` + testIDPAlias + `"`,
			},
		},
		{
			name:  "first offender only",
			alias: testIDPAlias,
			mappers: []keycloakApi.IdentityProviderMapper{
				{Name: "m1"},
				{Name: "m2", IdentityProviderAlias: "x"},
				{Name: "m3", IdentityProviderAlias: testIDPAlias},
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
				Spec: keycloakApi.KeycloakRealmIdentityProviderSpec{Alias: tt.alias, Mappers: tt.mappers},
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

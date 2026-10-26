package chain

import (
	"context"
	"fmt"

	keycloakApi "github.com/epam/edp-keycloak-operator/api/v1"
)

// ValidateMapperAliases rejects a mapper alias other than spec.alias before any handler writes to Keycloak.
//
// Deprecated: removed together with IdentityProviderMapper.IdentityProviderAlias.
type ValidateMapperAliases struct{}

// NewValidateMapperAliases returns a ValidateMapperAliases handler.
//
// Deprecated: removed together with IdentityProviderMapper.IdentityProviderAlias.
func NewValidateMapperAliases() *ValidateMapperAliases {
	return &ValidateMapperAliases{}
}

func (h *ValidateMapperAliases) Serve(
	_ context.Context,
	keycloakRealmIDP *keycloakApi.KeycloakRealmIdentityProvider,
	_ string,
) error {
	for i, m := range keycloakRealmIDP.Spec.Mappers {
		//nolint:staticcheck // rejects the deprecated identityProviderAlias when it names another identity provider
		if a := m.IdentityProviderAlias; a != "" && a != keycloakRealmIDP.Spec.Alias {
			return fmt.Errorf(
				"mappers[%d] %q: identityProviderAlias %q does not match spec.alias %q: a mapper can only be "+
					"attached to its own identity provider; remove identityProviderAlias, or define the mapper in the "+
					"KeycloakRealmIdentityProvider with spec.alias %q",
				i, m.Name, a, keycloakRealmIDP.Spec.Alias, a,
			)
		}
	}

	return nil
}

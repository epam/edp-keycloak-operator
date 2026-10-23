package chain

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/epam/edp-keycloak-operator/pkg/client/keycloakapi"
)

func TestMakeChain_Order(t *testing.T) {
	t.Parallel()

	c, ok := MakeChain(&keycloakapi.KeycloakClient{}).(*chain)
	require.True(t, ok)

	want := []Handler{
		&ValidateDomainSettings{},
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

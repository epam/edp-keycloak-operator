package keycloakapi_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/epam/edp-keycloak-operator/pkg/client/keycloakapi"
)

func TestOrganizationDomainName(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		input string
		want  string
	}{
		{name: "already normalized", input: "a.example.test", want: "a.example.test"},
		{name: "mixed case", input: "A.Example.TEST", want: "a.example.test"},
		{name: "surrounding spaces and tabs", input: " \ta.example.test\t ", want: "a.example.test"},
		{name: "empty", input: "", want: ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, tt.want, keycloakapi.OrganizationDomainName(tt.input))
		})
	}
}

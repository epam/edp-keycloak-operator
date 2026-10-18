package v1

import (
	"testing"

	"github.com/stretchr/testify/require"
	"k8s.io/utils/ptr"

	keycloakApi "github.com/epam/edp-keycloak-operator/api/v1"
)

func TestKeycloakClientCustomDefaulter_applyDefaults_webOrigins(t *testing.T) {
	const webUrl = "https://example.com"

	tests := []struct {
		name        string
		spec        keycloakApi.KeycloakClientSpec
		wantOrigins *[]string
		wantUpdated bool
	}{
		{
			name:        "nil with webUrl is defaulted",
			spec:        keycloakApi.KeycloakClientSpec{WebUrl: webUrl},
			wantOrigins: ptr.To([]string{webUrl}),
			wantUpdated: true,
		},
		{
			name:        "explicit empty list with webUrl is kept",
			spec:        keycloakApi.KeycloakClientSpec{WebUrl: webUrl, WebOrigins: ptr.To([]string{})},
			wantOrigins: ptr.To([]string{}),
		},
		{
			name:        "explicit list with webUrl is kept",
			spec:        keycloakApi.KeycloakClientSpec{WebUrl: webUrl, WebOrigins: ptr.To([]string{"https://other.example.com"})},
			wantOrigins: ptr.To([]string{"https://other.example.com"}),
		},
		{
			name:        "nil without webUrl stays nil",
			spec:        keycloakApi.KeycloakClientSpec{},
			wantOrigins: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// post.logout.redirect.uris is pre-set so only the webOrigins rule can flip "updated".
			tt.spec.Attributes = map[string]string{ClientAttributeLogoutRedirectUris: ClientAttributeLogoutRedirectUrisDefValue}
			kc := &keycloakApi.KeycloakClient{Spec: tt.spec}

			updated := (&KeycloakClientCustomDefaulter{}).applyDefaults(kc)

			require.Equal(t, tt.wantUpdated, updated)
			require.Equal(t, tt.wantOrigins, kc.Spec.WebOrigins)
		})
	}
}

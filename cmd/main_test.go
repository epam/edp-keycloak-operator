package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// makeDestinationGuard is the only place the operator's security policy is read from the
// environment.
func TestMakeDestinationGuard(t *testing.T) {
	tests := []struct {
		name        string
		hosts       string
		enforce     string
		wantHosts   []string
		wantEnforce bool
		wantErr     require.ErrorAssertionFunc
	}{
		{
			name:      "unset is an inactive guard",
			wantHosts: []string{},
			wantErr:   require.NoError,
		},
		{
			name:      "empty values are an inactive guard",
			hosts:     "",
			enforce:   "",
			wantHosts: []string{},
			wantErr:   require.NoError,
		},
		{
			name:      "comma separated list with padding",
			hosts:     " keycloak.example.com , smtp.example.com ",
			wantHosts: []string{"keycloak.example.com", "smtp.example.com"},
			wantErr:   require.NoError,
		},
		{
			name:        "enforcement with a list",
			hosts:       "keycloak.example.com",
			enforce:     "true",
			wantHosts:   []string{"keycloak.example.com"},
			wantEnforce: true,
			wantErr:     require.NoError,
		},
		{
			name:        "ParseBool spellings are honoured",
			hosts:       "keycloak.example.com",
			enforce:     "True",
			wantHosts:   []string{"keycloak.example.com"},
			wantEnforce: true,
			wantErr:     require.NoError,
		},
		{
			name:    "a malformed boolean is fatal rather than silently permissive",
			hosts:   "keycloak.example.com",
			enforce: "yes-please",
			wantErr: require.Error,
		},
		{
			name:    "enforcement without a list is fatal",
			enforce: "true",
			wantErr: require.Error,
		},
		{
			name:    "an entry carrying a scheme is fatal",
			hosts:   "https://keycloak.example.com",
			wantErr: require.Error,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv(allowedDestinationHostsEnv, tt.hosts)
			t.Setenv(enforceDestinationAllowlistEnv, tt.enforce)

			guard, err := makeDestinationGuard()
			tt.wantErr(t, err)

			if err != nil {
				return
			}

			assert.ElementsMatch(t, tt.wantHosts, guard.Hosts())
			assert.Equal(t, tt.wantEnforce, guard.Enforcing())
		})
	}
}

// Secrets and ConfigMaps are read without the cache, so the operator never lists or watches them cluster-wide.
func TestManagerClientOptions(t *testing.T) {
	opts := managerClientOptions()

	require.NotNil(t, opts.Cache)
	assert.ElementsMatch(t, []client.Object{&corev1.Secret{}, &corev1.ConfigMap{}}, opts.Cache.DisableFor)
}

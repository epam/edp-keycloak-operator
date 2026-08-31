package destination

import (
	"context"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNew(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		hosts     []string
		enforce   bool
		wantHosts []string
		wantErr   require.ErrorAssertionFunc
	}{
		{
			name:      "empty list without enforcement",
			hosts:     nil,
			enforce:   false,
			wantHosts: nil,
			wantErr:   require.NoError,
		},
		{
			name:    "empty list with enforcement is fatal",
			hosts:   nil,
			enforce: true,
			wantErr: require.Error,
		},
		{
			name:    "blank entries only, with enforcement, is fatal",
			hosts:   []string{"  ", ""},
			enforce: true,
			wantErr: require.Error,
		},
		{
			name:      "trims, lowercases and drops empties",
			hosts:     []string{"  Keycloak.Example.COM ", "", "smtp.example.com"},
			enforce:   true,
			wantHosts: []string{"keycloak.example.com", "smtp.example.com"},
			wantErr:   require.NoError,
		},
		{
			name:      "strips one trailing dot",
			hosts:     []string{"keycloak.example.com."},
			enforce:   true,
			wantHosts: []string{"keycloak.example.com"},
			wantErr:   require.NoError,
		},
		{
			name:      "accepts an unbracketed IPv6 literal",
			hosts:     []string{"2001:db8::1"},
			enforce:   true,
			wantHosts: []string{"2001:db8::1"},
			wantErr:   require.NoError,
		},
		{
			name:    "entry with a scheme is fatal",
			hosts:   []string{"https://keycloak.example.com"},
			enforce: true,
			wantErr: require.Error,
		},
		{
			name:    "entry with a port is fatal",
			hosts:   []string{"smtp.example.com:587"},
			enforce: true,
			wantErr: require.Error,
		},
		{
			name:    "entry with a path is fatal",
			hosts:   []string{"keycloak.example.com/auth"},
			enforce: true,
			wantErr: require.Error,
		},
		{
			name:    "entry with a control character is fatal",
			hosts:   []string{"keycloak.example.com\nevil"},
			enforce: true,
			wantErr: require.Error,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			g, err := New(tt.hosts, tt.enforce)
			tt.wantErr(t, err)

			if err != nil {
				return
			}

			for _, h := range tt.wantHosts {
				assert.NoError(t, g.RequireHost(context.Background(), "spec.url", h),
					"expected %q to be permitted", h)
			}
		})
	}
}

func TestGuard_RequireHost(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		value   string
		wantErr require.ErrorAssertionFunc
	}{
		{name: "listed host as bare name", value: "keycloak.example.com", wantErr: require.NoError},
		{name: "listed host in a url", value: "https://keycloak.example.com/auth", wantErr: require.NoError},
		{name: "listed host with a port", value: "https://keycloak.example.com:8443/auth", wantErr: require.NoError},
		{name: "listed host uppercase in a url", value: "HTTPS://Keycloak.Example.COM/", wantErr: require.NoError},
		{name: "listed host uppercase bare", value: "KEYCLOAK.EXAMPLE.COM", wantErr: require.NoError},
		{name: "listed host with a trailing dot", value: "https://keycloak.example.com./auth", wantErr: require.NoError},
		{name: "listed host with bare port", value: "keycloak.example.com:8443", wantErr: require.NoError},
		{name: "listed ipv6 bracketed in a url", value: "ldaps://[2001:db8::1]:636", wantErr: require.NoError},
		{name: "listed ipv6 bare bracketed with port", value: "[2001:db8::1]:636", wantErr: require.NoError},
		{name: "listed ipv6 bare unbracketed", value: "2001:db8::1", wantErr: require.NoError},

		{name: "unlisted host", value: "https://evil.example.com/", wantErr: require.Error},
		{
			name:  "userinfo cannot smuggle a listed host",
			value: "https://keycloak.example.com@evil.example.com/", wantErr: require.Error,
		},
		{
			name:  "userinfo with port cannot smuggle a listed host",
			value: "https://keycloak.example.com:8443@evil.example.com/", wantErr: require.Error,
		},
		{name: "suffix is not a match", value: "https://keycloak.example.com.evil.example.com/", wantErr: require.Error},
		{name: "prefix is not a match", value: "https://evilkeycloak.example.com/", wantErr: require.Error},
		{name: "empty value", value: "", wantErr: require.Error},
		{name: "opaque url with no host", value: "evil.example.com:25", wantErr: require.Error},
		{
			name:  "space separated failover list",
			value: "ldap://keycloak.example.com ldap://evil.example.com", wantErr: require.Error,
		},
		{name: "secret ref is not a destination", value: "$my-secret:url", wantErr: require.Error},
		{name: "newline injection is rejected", value: "keycloak.example.com\nX-Injected: 1", wantErr: require.Error},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			g, err := New([]string{"keycloak.example.com", "2001:db8::1"}, true)
			require.NoError(t, err)

			tt.wantErr(t, g.RequireHost(context.Background(), "spec.url", tt.value))
		})
	}
}

func TestGuard_RequireHost_WarnModeNeverDenies(t *testing.T) {
	t.Parallel()

	g, err := New([]string{"keycloak.example.com"}, false)
	require.NoError(t, err)

	for _, v := range []string{
		"https://evil.example.com/",
		"$my-secret:url",
		"evil.example.com:25",
		"",
	} {
		assert.NoError(t, g.RequireHost(context.Background(), "spec.url", v),
			"warn mode must not deny %q", v)
	}
}

func TestGuard_ScanConfig(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		config  map[string][]string
		wantErr require.ErrorAssertionFunc
	}{
		{
			name:    "nil config",
			config:  nil,
			wantErr: require.NoError,
		},
		{
			name: "plain values are not destinations",
			config: map[string][]string{
				"bindDn":      {"cn=admin,dc=example,dc=com"},
				"searchScope": {"1"},
				"vendor":      {"ad"},
			},
			wantErr: require.NoError,
		},
		{
			name: "listed destination and its credential ref",
			config: map[string][]string{
				"connectionUrl":  {"ldap://keycloak.example.com:389"},
				"bindCredential": {"$ldap-secret:password"},
			},
			wantErr: require.NoError,
		},
		{
			name: "unlisted destination in a url value",
			config: map[string][]string{
				"connectionUrl": {"ldap://evil.example.com:389"},
			},
			wantErr: require.Error,
		},
		{
			name: "secret ref in a destination key is a violation",
			config: map[string][]string{
				"tokenUrl":     {"$attacker-secret:url"},
				"clientSecret": {"$victim-secret:key"},
			},
			wantErr: require.Error,
		},
		{
			name: "secret ref in an unknown key is a violation",
			config: map[string][]string{
				"someFutureUrl": {"$attacker-secret:url"},
			},
			wantErr: require.Error,
		},
		{
			name: "keycloak vault ref passes through",
			config: map[string][]string{
				"bindCredential": {"${vault.ldap-password}"},
			},
			wantErr: require.NoError,
		},
		{
			name: "known destination key holding a non-host is a violation",
			config: map[string][]string{
				"tokenUrl": {"not a url"},
			},
			wantErr: require.Error,
		},
		{
			name: "space separated failover list is a violation",
			config: map[string][]string{
				"connectionUrl": {"ldap://keycloak.example.com ldap://evil.example.com"},
			},
			wantErr: require.Error,
		},
		{
			name: "any value carrying a scheme is checked",
			config: map[string][]string{
				"undocumentedKey": {"https://evil.example.com/collect"},
			},
			wantErr: require.Error,
		},
		{
			name: "every value in a multivalued key is checked",
			config: map[string][]string{
				"connectionUrl": {"ldap://keycloak.example.com", "ldap://evil.example.com"},
			},
			wantErr: require.Error,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			g, err := New([]string{"keycloak.example.com"}, true)
			require.NoError(t, err)

			tt.wantErr(t, g.ScanConfig(context.Background(), "spec.config", tt.config))
		})
	}
}

func TestGuard_ScanConfig_WarnModeNeverDenies(t *testing.T) {
	t.Parallel()

	g, err := New([]string{"keycloak.example.com"}, false)
	require.NoError(t, err)

	assert.NoError(t, g.ScanConfig(context.Background(), "spec.config", map[string][]string{
		"tokenUrl":     {"$attacker-secret:url"},
		"clientSecret": {"$victim-secret:key"},
	}))
}

func TestGuard_ErrorMessageIsActionableAndQuoted(t *testing.T) {
	t.Parallel()

	g, err := New([]string{"keycloak.example.com"}, true)
	require.NoError(t, err)

	err = g.RequireHost(context.Background(), "spec.url", "https://evil.example.com/")
	require.Error(t, err)

	assert.ErrorIs(t, err, ErrNotAllowed)
	assert.Contains(t, err.Error(), "allowedDestinationHosts")
	assert.Contains(t, err.Error(), "spec.url")
	assert.Contains(t, err.Error(), `"evil.example.com"`, "host must be quoted")
}

func TestGuard_ErrorMessageNeverSpansLines(t *testing.T) {
	t.Parallel()

	g, err := New([]string{"keycloak.example.com"}, true)
	require.NoError(t, err)

	err = g.RequireHost(context.Background(), "spec.smtp.connection.host",
		"evil.example.com\nlevel=info msg=\"forged\"")
	require.Error(t, err)

	assert.NotContains(t, err.Error(), "\n", "a rejected value must not inject a newline")
}

func TestGuard_ErrorMessageIsLengthCapped(t *testing.T) {
	t.Parallel()

	g, err := New([]string{"keycloak.example.com"}, true)
	require.NoError(t, err)

	err = g.RequireHost(context.Background(), "spec.url", strings.Repeat("a", 4096))
	require.Error(t, err)

	assert.Less(t, len(err.Error()), 512, "a rejected value must not blow up the message")
}

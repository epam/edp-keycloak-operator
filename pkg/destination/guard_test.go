package destination

import (
	"context"
	"strings"
	"testing"

	"github.com/prometheus/client_golang/prometheus/testutil"
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
			name:    "entry with a double colon typo is fatal",
			hosts:   []string{"smtp.example.com::587"},
			enforce: true,
			wantErr: require.Error,
		},
		{
			name:    "bracketed ipv6 entry with a port is fatal",
			hosts:   []string{"[2001:db8::1]:636"},
			enforce: true,
			wantErr: require.Error,
		},
		{
			name:      "accepts a bracketed IPv6 literal",
			hosts:     []string{"[2001:db8::1]"},
			enforce:   true,
			wantHosts: []string{"2001:db8::1"},
			wantErr:   require.NoError,
		},
		{
			name:    "colon-separated non-address is fatal",
			hosts:   []string{"a:b:c"},
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
			name: "secret ref in a key provider privateKey is a credential",
			config: map[string][]string{
				"privateKey":  {"$realm-key:private"},
				"certificate": {"$realm-key:cert"},
			},
			wantErr: require.NoError,
		},
		{
			name: "empty value in a destination key is skipped",
			config: map[string][]string{
				"baseUrl":   {""},
				"logoutUrl": {"   "},
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
			name: "url in an unknown key is not judged",
			config: map[string][]string{
				"undocumentedKey": {"https://evil.example.com/collect"},
			},
			wantErr: require.NoError,
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

// Warn mode exists so an administrator can read every host that will later be denied.
func TestGuard_ScanConfig_WarnModeReportsEveryViolation(t *testing.T) {
	t.Parallel()

	g, err := New([]string{"keycloak.example.com"}, false)
	require.NoError(t, err)

	const field = "test.reportsEveryViolation"

	before := warnViolations(field)

	require.NoError(t, g.ScanConfig(context.Background(), field, map[string][]string{
		"tokenUrl":    {"$victim:url"},
		"userInfoUrl": {"https://evil.example.com/"},
	}))

	assert.Equal(t, float64(2), warnViolations(field)-before, "both violations must be recorded")
}

func TestGuard_ScanConfig_AcceptsBareHostInADestinationKey(t *testing.T) {
	t.Parallel()

	g, err := New([]string{"keycloak.example.com"}, true)
	require.NoError(t, err)

	assert.NoError(t, g.ScanConfig(context.Background(), "spec.config", map[string][]string{
		"baseUrl": {"keycloak.example.com"},
	}))
}

func TestGuard_ScanConfig_MultiAddressFailover(t *testing.T) {
	t.Parallel()

	g, err := New([]string{"ldap1.example.com", "ldap2.example.com"}, true)
	require.NoError(t, err)

	assert.NoError(t, g.ScanConfig(context.Background(), "spec.config", map[string][]string{
		"connectionUrl": {"ldap://ldap1.example.com:389 ldap://ldap2.example.com:389"},
	}))

	assert.Error(t, g.ScanConfig(context.Background(), "spec.config", map[string][]string{
		"connectionUrl": {"ldap://ldap1.example.com:389 ldap://evil.example.com:389"},
	}), "one unlisted address in the list must still be denied")
}

func TestGuard_ScanConfig_VaultRefOutsideACredentialKeyIsDenied(t *testing.T) {
	t.Parallel()

	g, err := New([]string{"keycloak.example.com"}, true)
	require.NoError(t, err)

	assert.Error(t, g.ScanConfig(context.Background(), "spec.config", map[string][]string{
		"tokenUrl": {"${anything"},
	}))
}

// Policy-off is AllowAll; nil fails closed.
func TestGuard_NilReceiverFailsClosed(t *testing.T) {
	t.Parallel()

	var g *Guard

	assert.ErrorIs(t, g.RequireHost(context.Background(), "spec.url", "https://keycloak.example.com/"), ErrGuardRequired)
	assert.ErrorIs(t, g.ScanConfig(context.Background(), "spec.config", map[string][]string{
		"tokenUrl": {"https://keycloak.example.com/"},
	}), ErrGuardRequired)
}

func TestSanitizeForLog(t *testing.T) {
	t.Parallel()

	assert.Equal(t, "evil.example.com?forged", sanitizeForLog("evil.example.com\nforged"))
	assert.Equal(t, "keycloak.example.com", sanitizeForLog("keycloak.example.com"))
}

// warnViolations reads the counter for a field in warn mode.
func warnViolations(field string) float64 {
	return testutil.ToFloat64(violations.WithLabelValues(field, "false"))
}

// An empty allowlist leaves the guard inactive (see Guard.inactive).
func TestGuard_EmptyAllowlistIsSilent(t *testing.T) {
	t.Parallel()

	g, err := New(nil, false)
	require.NoError(t, err)

	const field = "test.emptyAllowlistIsSilent"

	before := warnViolations(field)

	require.NoError(t, g.RequireHost(context.Background(), field, "https://anything.example.com"))
	require.NoError(t, g.ScanConfig(context.Background(), field, map[string][]string{
		"tokenUrl": {"https://anything.example.com"},
	}))

	assert.Equal(t, float64(0), warnViolations(field)-before, "an unconfigured guard must record nothing")
}

// A value in an unrecognised key is never judged or reported (see ScanConfig): key semantics
// are unknown at runtime, and a guess must not deny or flood the warn log.
func TestGuard_ScanConfig_UnknownKeysAreNotJudged(t *testing.T) {
	t.Parallel()

	g, err := New([]string{"keycloak.example.com"}, false)
	require.NoError(t, err)

	const field = "test.unknownKeysNotJudged"

	before := warnViolations(field)

	require.NoError(t, g.ScanConfig(context.Background(), field, map[string][]string{
		"helpText":        {"see https://evil.example.com for details"},
		"undocumentedKey": {"evil.example.com:8443"},
	}))

	assert.Equal(t, float64(0), warnViolations(field)-before,
		"a value in an unknown key must not be reported")
}

// Enforce mode reports every violation in one error, so one reconcile surfaces the whole list.
func TestGuard_ScanConfig_EnforceModeReportsEveryViolation(t *testing.T) {
	t.Parallel()

	g, err := New([]string{"keycloak.example.com"}, true)
	require.NoError(t, err)

	err = g.ScanConfig(context.Background(), "spec.config", map[string][]string{
		"tokenUrl":    {"https://evil-one.example.com/"},
		"userInfoUrl": {"https://evil-two.example.com/"},
	})
	require.Error(t, err)

	assert.ErrorIs(t, err, ErrNotAllowed)
	assert.Contains(t, err.Error(), "evil-one.example.com")
	assert.Contains(t, err.Error(), "evil-two.example.com")
}

// The config key is author-controlled and must not forge lines in the error (see report).
func TestGuard_ScanConfig_ConfigKeyCannotForgeError(t *testing.T) {
	t.Parallel()

	g, err := New([]string{"keycloak.example.com"}, true)
	require.NoError(t, err)

	err = g.ScanConfig(context.Background(), "spec.config", map[string][]string{
		"evil\nlevel=info msg=forged": {"$attacker-secret:url"},
	})
	require.Error(t, err)

	assert.NotContains(t, err.Error(), "\n", "a config key must not inject a newline")
}

// The config key must never become a metric label (see the violations counter).
func TestGuard_ScanConfig_MetricLabelIsTheStaticField(t *testing.T) {
	t.Parallel()

	g, err := New([]string{"keycloak.example.com"}, false)
	require.NoError(t, err)

	const field = "test.metricLabelIsStatic"

	before := warnViolations(field)

	require.NoError(t, g.ScanConfig(context.Background(), field, map[string][]string{
		"attackerChosenKeyOne": {"$attacker-secret:url"},
		"attackerChosenKeyTwo": {"$attacker-secret:url"},
	}))

	assert.Equal(t, float64(2), warnViolations(field)-before, "both must land on the one static label")
	assert.Zero(t, warnViolations(field+".attackerChosenKeyOne"), "the key must not create a series")
}

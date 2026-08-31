// Package destination decides whether a Secret value resolved by the operator may be sent to a
// destination named in a custom resource.
//
// A custom resource author supplies both a Secret reference and an outbound destination in one
// spec. The operator's ServiceAccount can read Secrets the author cannot, so without this check
// the author can name any Secret and any destination and have the operator deliver one to the
// other (GHSA-wj3g-w873-xwg7).
package destination

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/url"
	"regexp"
	"strings"

	"github.com/prometheus/client_golang/prometheus"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/metrics"
)

// ErrNotAllowed wraps every denial so callers can distinguish policy from transport failure.
var ErrNotAllowed = errors.New("destination not allowed")

// maxHostLen is the DNS name limit. Values are truncated to it before they reach a log line or
// a status field.
const maxHostLen = 253

// hostPattern is the character set permitted in a host token. It excludes whitespace and
// control characters, so a rejected value cannot forge a log line.
var hostPattern = regexp.MustCompile(`^[a-z0-9.\-_:%\[\]]+$`)

// credentialKeys hold a secret value, so a secret reference is legitimate there. Every other key
// carrying a reference is a violation: the resolver substitutes references in place regardless of
// key, so a reference in a destination key would supply a destination that was never checked.
var credentialKeys = map[string]struct{}{
	"clientsecret":   {},
	"bindcredential": {},
}

// destinationKeys are Keycloak configuration keys whose value is an address. A value here that
// does not reduce to a host is rejected rather than skipped.
var destinationKeys = map[string]struct{}{
	"tokenurl":                     {},
	"tokenintrospectionurl":        {},
	"userinfourl":                  {},
	"jwksurl":                      {},
	"logouturl":                    {},
	"authorizationurl":             {},
	"metadatadescriptorurl":        {},
	"baseurl":                      {},
	"apiurl":                       {},
	"singlesignonserviceurl":       {},
	"singlelogoutserviceurl":       {},
	"artifactresolutionserviceurl": {},
	"connectionurl":                {},
}

var violations = prometheus.NewCounterVec(
	prometheus.CounterOpts{
		Name: "keycloak_operator_destination_violations_total",
		Help: "Destinations named in a custom resource that are absent from allowedDestinationHosts. " +
			"The host is deliberately not a label: it is author-controlled and would be unbounded cardinality.",
	},
	[]string{"field", "enforced"},
)

func init() {
	metrics.Registry.MustRegister(violations)
}

// Guard holds the operator's register of permitted destination hosts.
type Guard struct {
	hosts   map[string]struct{}
	enforce bool
}

// New normalises and validates the configured hosts.
//
// Entries are bare hostnames: no scheme, no port, no path. An entry that is not is a startup
// error rather than a silent mismatch, because a list that looks correct but never matches
// would deny every destination once enforcement is on.
func New(hosts []string, enforce bool) (*Guard, error) {
	set := make(map[string]struct{}, len(hosts))

	for _, raw := range hosts {
		entry := strings.TrimSpace(raw)
		if entry == "" {
			continue
		}

		host, ok := normalizeEntry(entry)
		if !ok {
			return nil, fmt.Errorf(
				"%w: allowedDestinationHosts entry %q is not a bare hostname;"+
					" write the hostname only, without scheme, port or path",
				ErrNotAllowed, truncate(entry),
			)
		}

		set[host] = struct{}{}
	}

	if enforce && len(set) == 0 {
		return nil, fmt.Errorf(
			"%w: enforceDestinationAllowlist is true but allowedDestinationHosts is empty, which would deny every destination",
			ErrNotAllowed,
		)
	}

	return &Guard{hosts: set, enforce: enforce}, nil
}

// AllowAll returns a guard that permits every destination and records nothing. It is the
// zero-configuration default, matching operator behaviour before the allowlist existed.
func AllowAll() *Guard {
	return &Guard{hosts: map[string]struct{}{}, enforce: false}
}

// Hosts returns the normalised register, for logging it once at startup.
func (g *Guard) Hosts() []string {
	out := make([]string, 0, len(g.hosts))
	for h := range g.hosts {
		out = append(out, h)
	}

	return out
}

// RequireHost checks a field whose only legitimate content is an address. A value that does not
// reduce to a host is a violation, not a skip.
func (g *Guard) RequireHost(ctx context.Context, field, value string) error {
	host, ok := hostFromValue(value)
	if !ok {
		return g.deny(ctx, field, value, "is not a usable destination host")
	}

	return g.check(ctx, field, host)
}

// ScanConfig checks a free-form Keycloak configuration map. Values are classified rather than
// filtered by key, because the map is open and Keycloak adds keys across versions.
//
// It must run before secret references are resolved. Afterwards a resolved secret would be
// indistinguishable from an author-supplied address.
func (g *Guard) ScanConfig(ctx context.Context, field string, config map[string][]string) error {
	for key, values := range config {
		lower := strings.ToLower(key)
		_, isCredential := credentialKeys[lower]
		_, isDestination := destinationKeys[lower]

		for _, value := range values {
			if isVaultRef(value) {
				continue
			}

			if isSecretRef(value) {
				if isCredential {
					continue
				}

				return g.deny(ctx, field+"."+key, value,
					"is a secret reference in a key that is not a credential, so it would supply an unchecked destination")
			}

			if !isDestination && !strings.Contains(value, "://") {
				continue
			}

			host, ok := hostFromURL(value)
			if !ok {
				return g.deny(ctx, field+"."+key, value, "is not a usable destination host")
			}

			if err := g.check(ctx, field+"."+key, host); err != nil {
				return err
			}
		}
	}

	return nil
}

func (g *Guard) check(ctx context.Context, field, host string) error {
	if _, ok := g.hosts[host]; ok {
		return nil
	}

	return g.deny(ctx, field, host, "is not in allowedDestinationHosts")
}

// deny records the violation and, when enforcing, returns it. In warn mode the reconcile
// continues so an existing installation keeps working while its administrator builds the list.
func (g *Guard) deny(ctx context.Context, field, value, reason string) error {
	safe := truncate(value)

	violations.WithLabelValues(field, fmt.Sprintf("%t", g.enforce)).Inc()

	err := fmt.Errorf(
		"%w: %q (%s) %s; add the host to allowedDestinationHosts in the operator Helm values,"+
			" or set enforceDestinationAllowlist to false",
		ErrNotAllowed, safe, field, reason,
	)

	if !g.enforce {
		ctrl.LoggerFrom(ctx).Info(
			"Destination is not in allowedDestinationHosts. It will be denied once enforceDestinationAllowlist is enabled",
			"field", field, "destination", safe, "reason", reason,
		)

		return nil
	}

	return err
}

// normalizeEntry accepts only a bare hostname, with an optional trailing dot and optional IPv6
// brackets. Anything carrying a scheme, port or path is rejected.
func normalizeEntry(entry string) (string, bool) {
	if strings.ContainsAny(entry, "/@ ") || strings.Contains(entry, "://") {
		return "", false
	}

	host := entry

	// A bracketed literal may carry a port; an unbracketed one is an IPv6 address whose colons
	// are part of the address.
	if strings.HasPrefix(host, "[") {
		h, _, err := net.SplitHostPort(host)
		if err != nil {
			h = strings.TrimSuffix(strings.TrimPrefix(host, "["), "]")
		}

		host = h
	} else if strings.Count(host, ":") == 1 {
		return "", false // host:port
	}

	return canonicalHost(host)
}

// hostFromValue accepts either an absolute URL or a bare host. Used for fields that hold nothing
// else.
func hostFromValue(value string) (string, bool) {
	if strings.Contains(value, "://") {
		return hostFromURL(value)
	}

	if isSecretRef(value) || isVaultRef(value) {
		return "", false
	}

	host := value
	if h, _, err := net.SplitHostPort(host); err == nil {
		host = h
	} else if strings.HasPrefix(host, "[") {
		host = strings.TrimSuffix(strings.TrimPrefix(host, "["), "]")
	}

	return canonicalHost(host)
}

// hostFromURL requires an absolute URL carrying a host. url.Parse rejects control characters and
// resolves userinfo, so "https://good.example.com@evil.example.com/" yields evil.example.com.
func hostFromURL(value string) (string, bool) {
	u, err := url.Parse(value)
	if err != nil {
		return "", false
	}

	return canonicalHost(u.Hostname())
}

func canonicalHost(host string) (string, bool) {
	host = strings.ToLower(strings.TrimSpace(host))
	host = strings.TrimSuffix(host, ".")

	if host == "" || len(host) > maxHostLen || !hostPattern.MatchString(host) {
		return "", false
	}

	return host, true
}

func isSecretRef(value string) bool {
	return strings.HasPrefix(value, "$") && !isVaultRef(value)
}

// isVaultRef matches Keycloak's own "${vault.x}" indirection, which the operator forwards
// untouched and never resolves.
func isVaultRef(value string) bool {
	return strings.HasPrefix(value, "${")
}

func truncate(value string) string {
	if len(value) > maxHostLen {
		return value[:maxHostLen] + "..."
	}

	return value
}

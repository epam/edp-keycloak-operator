// Package destination decides whether a Secret value resolved by the operator may be sent to a
// destination named in a custom resource.
//
// A custom resource author supplies both a Secret reference and an outbound destination in one
// spec. The operator's ServiceAccount can read Secrets the author cannot, so without this check
// the author can name any Secret and any destination and have the operator deliver one to the
// other (GHSA-wj3g-w873-xwg7).
//
// Covered spec fields are listed in deploy-templates/values.yaml (allowedDestinationHosts);
// update that list when adding a call site.
//
// The guard judges only fields this operator version knows: spec.url, spec.smtp.connection.host,
// and the destination and credential keys listed below. Values in unrecognised config keys are
// not judged. Egress control for the Keycloak pod itself belongs to a NetworkPolicy.
package destination

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/url"
	"regexp"
	"slices"
	"strconv"
	"strings"

	"github.com/prometheus/client_golang/prometheus"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/metrics"
)

// ErrNotAllowed wraps every denial, so callers can tell policy from transport failure.
var ErrNotAllowed = errors.New("destination not allowed")

// ErrInvalidConfig wraps startup configuration faults, which are not denials.
var ErrInvalidConfig = errors.New("invalid destination allowlist configuration")

// ErrGuardRequired reports an omitted destination guard. Callers must pass AllowAll explicitly
// when policy enforcement is intentionally disabled.
var ErrGuardRequired = errors.New("destination guard is required")

// RefPrefix marks a value the resolver substitutes: "$name:key" for a Kubernetes Secret,
// "${vault.x}" for Keycloak's vault. A reference is legitimate only in a credential key.
// pkg/secretref derives its secret-reference prefix from this constant.
const RefPrefix = "$"

// maxHostLen is the DNS name limit. Values are truncated to it before reaching a log or a status.
const maxHostLen = 253

// hostPattern is the character set permitted in a host token. Excludes whitespace and control
// characters.
var hostPattern = regexp.MustCompile(`^[a-z0-9.\-_:%\[\]]+$`)

// credentialKeys are the config keys in which a secret or vault reference is permitted.
// Entries are lowercase; lookup lowercases the config key.
// Mirrors Keycloak provider configs; verified against a running Keycloak by
// TestLists_MatchKeycloakModel. Re-check on KEYCLOAK_VERSION bumps (Makefile).
var credentialKeys = map[string]struct{}{
	"clientsecret":     {},
	"bindcredential":   {},
	"privatekey":       {},
	"certificate":      {},
	"keystorepassword": {},
	"keypassword":      {},
	"secret.key":       {},
	"api.key":          {},
}

// destinationKeys are Keycloak configuration keys whose value is an address. A bare hostname is
// accepted here, an empty value is skipped, a non-empty value that does not reduce to a host is
// rejected, and space-separated addresses are checked individually (LDAP connectionUrl failover).
// Mirrors Keycloak provider configs; verified against a running Keycloak by
// TestLists_MatchKeycloakModel. Re-check on KEYCLOAK_VERSION bumps (Makefile).
var destinationKeys = map[string]struct{}{
	"tokenurl":                          {},
	"tokenintrospectionurl":             {},
	"userinfourl":                       {},
	"jwksurl":                           {},
	"logouturl":                         {},
	"authorizationurl":                  {},
	"metadatadescriptorurl":             {},
	"baseurl":                           {},
	"apiurl":                            {},
	"singlesignonserviceurl":            {},
	"singlelogoutserviceurl":            {},
	"artifactresolutionserviceurl":      {},
	"connectionurl":                     {},
	"x509-cert-auth.ocsp-responder-uri": {},
	"sectoridentifieruri":               {},
	"intent-client-bind-check-endpoint": {},
}

// Labels are static: field is a spec path, enforced a boolean. Config keys and hosts are
// author-controlled and unbounded, so they appear in the log line, never in a label.
var violations = prometheus.NewCounterVec(
	prometheus.CounterOpts{
		Name: "keycloak_operator_destination_violations_total",
		Help: "Destinations named in a custom resource that are absent from allowedDestinationHosts.",
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
// Entries are bare hostnames or IP literals (IPv6 optionally bracketed): no scheme, no port,
// no path. A malformed entry is a startup error, not a skipped entry.
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
				ErrInvalidConfig, truncate(entry),
			)
		}

		set[host] = struct{}{}
	}

	if enforce && len(set) == 0 {
		return nil, fmt.Errorf(
			"%w: enforceDestinationAllowlist is true but allowedDestinationHosts is empty,"+
				" which would deny every destination",
			ErrInvalidConfig,
		)
	}

	return &Guard{hosts: set, enforce: enforce}, nil
}

// AllowAll returns an inactive guard: RequireHost and ScanConfig permit every destination and
// record nothing.
func AllowAll() *Guard {
	return &Guard{hosts: map[string]struct{}{}, enforce: false}
}

// Hosts returns the normalised register, sorted, for logging it once at startup.
func (g *Guard) Hosts() []string {
	if g == nil {
		return nil
	}

	out := make([]string, 0, len(g.hosts))
	for h := range g.hosts {
		out = append(out, h)
	}

	slices.Sort(out)

	return out
}

// Enforcing reports whether a denial fails the reconcile.
func (g *Guard) Enforcing() bool {
	return g != nil && g.enforce
}

// inactive reports that no policy is configured. With an empty register the guard neither
// checks nor reports. Populating the list turns reporting on; enforce turns denial on.
func (g *Guard) inactive() bool {
	return len(g.hosts) == 0
}

// RequireHost checks a field whose only legitimate content is one address. Unlike ScanConfig it
// does not split on whitespace: a space-separated value is rejected whole.
func (g *Guard) RequireHost(ctx context.Context, field, value string) error {
	// Nil is a wiring fault, not policy-off; policy-off is AllowAll.
	if g == nil {
		return ErrGuardRequired
	}

	if g.inactive() {
		return nil
	}

	return g.checkAddress(ctx, field, field, value)
}

// ScanConfig checks a free-form Keycloak configuration map. Every address in a destination key
// is checked; a secret or vault reference is permitted only in a credential key; values in any
// other key are not judged.
//
// It must run before references are resolved. Afterwards a resolved secret is indistinguishable
// from an author-supplied address.
func (g *Guard) ScanConfig(ctx context.Context, field string, config map[string][]string) error {
	// Nil is a wiring fault, not policy-off; policy-off is AllowAll.
	if g == nil {
		return ErrGuardRequired
	}

	if g.inactive() {
		return nil
	}

	// Every violation is reported; report returns nil in warn mode and errors.Join drops nils.
	errs := make([]error, 0, len(config))

	for key, values := range config {
		lower := strings.ToLower(key)
		_, isCredential := credentialKeys[lower]
		_, isDestination := destinationKeys[lower]
		path := field + "." + key

		for _, value := range values {
			if strings.HasPrefix(value, RefPrefix) {
				if isCredential {
					continue
				}

				errs = append(errs, g.report(ctx, field, path, value,
					"is a reference in a key that is not a credential, so it would supply an unchecked destination"))

				continue
			}

			if isDestination {
				errs = append(errs, g.scanValue(ctx, field, path, value))
			}
		}
	}

	return errors.Join(errs...)
}

// scanValue checks one destination-key value. Every whitespace-separated address is checked
// (LDAP connectionUrl failover). An empty value names no destination and is skipped.
func (g *Guard) scanValue(ctx context.Context, field, path, value string) error {
	addresses := strings.Fields(value)
	errs := make([]error, 0, len(addresses))

	for _, address := range addresses {
		errs = append(errs, g.checkAddress(ctx, field, path, address))
	}

	return errors.Join(errs...)
}

func (g *Guard) checkAddress(ctx context.Context, field, path, address string) error {
	host, ok := hostFromValue(address)
	if !ok {
		return g.report(ctx, field, path, address, "is not a usable destination host")
	}

	if _, ok := g.hosts[host]; ok {
		return nil
	}

	return g.report(ctx, field, path, host, "is not in allowedDestinationHosts")
}

// report records the violation and, when enforcing, returns it as an error; in warn mode it
// returns nil.
//
// field is the static metric label; path carries the author-controlled config key, for humans.
// path and value are author-controlled: both are sanitized and truncated before any log or error.
func (g *Guard) report(ctx context.Context, field, path, value, reason string) error {
	value = sanitizeForLog(truncate(value))
	path = sanitizeForLog(truncate(path))

	violations.WithLabelValues(field, strconv.FormatBool(g.enforce)).Inc()

	if !g.enforce {
		ctrl.LoggerFrom(ctx).Info(
			"Destination is not in allowedDestinationHosts. It will be denied once enforceDestinationAllowlist is enabled",
			"field", path, "destination", value, "reason", reason,
		)

		return nil
	}

	return fmt.Errorf(
		"%w: %q (%s) %s; add the host to allowedDestinationHosts in the operator Helm values,"+
			" or set enforceDestinationAllowlist to false",
		ErrNotAllowed, value, path, reason,
	)
}

// normalizeEntry accepts only a bare hostname with an optional trailing dot, or an IP literal
// (IPv6 optionally bracketed). Anything carrying a scheme, port or path is rejected.
func normalizeEntry(entry string) (string, bool) {
	if strings.ContainsAny(entry, "/@ ") || strings.Contains(entry, "://") {
		return "", false
	}

	host := entry

	if strings.HasPrefix(host, "[") {
		// SplitHostPort succeeding means a port follows the brackets.
		if _, _, err := net.SplitHostPort(host); err == nil {
			return "", false
		}

		host = strings.TrimSuffix(strings.TrimPrefix(host, "["), "]")
	}

	// A colon is legitimate only inside an IPv6 literal; host:port and typos are rejected.
	if strings.Contains(host, ":") && net.ParseIP(host) == nil {
		return "", false
	}

	return canonicalHost(host)
}

// hostFromValue accepts either an absolute URL or a bare host.
func hostFromValue(value string) (string, bool) {
	if strings.Contains(value, "://") {
		return hostFromURL(value)
	}

	if strings.HasPrefix(value, RefPrefix) {
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

// sanitizeForLog keeps printable ASCII only. The console log encoder does not escape control
// characters; an unescaped author-controlled value could forge log lines.
func sanitizeForLog(value string) string {
	return strings.Map(func(r rune) rune {
		if r > 0x20 && r < 0x7f {
			return r
		}

		return '?'
	}, value)
}

func truncate(value string) string {
	if len(value) > maxHostLen {
		return value[:maxHostLen] + "..."
	}

	return value
}

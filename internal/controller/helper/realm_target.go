package helper

import (
	"context"
	"fmt"
	"net"
	"net/netip"
	"net/url"
	"strings"

	"golang.org/x/net/idna"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/epam/edp-keycloak-operator/api/common"
	keycloakApi "github.com/epam/edp-keycloak-operator/api/v1"
	keycloakAlpha "github.com/epam/edp-keycloak-operator/api/v1alpha1"
)

// RealmTarget identifies the Keycloak realm an object talks to.
// URL is the canonical endpoint: lowercase scheme and host, IDNA ASCII host, no default port,
// no userinfo, query or trailing slash.
// TLSServerName is the effective SNI name: spec.tlsServerName, else the URL host for https,
// else empty. An SNI-routing front end selects the server by it.
type RealmTarget struct {
	URL           string
	TLSServerName string
	RealmName     string
}

// RealmRefKind returns the realm ref kind; an empty kind is KeycloakRealm.
func RealmRefKind(ref common.RealmRef) string {
	if ref.Kind == "" {
		return keycloakApi.KeycloakRealmKind
	}

	return ref.Kind
}

// SameRealmRef reports whether two objects reference the same realm CR.
// KeycloakRealm is namespaced: identity is (namespace, name).
// ClusterKeycloakRealm is cluster-scoped: identity is the name alone.
// A false result does not mean the objects target different realms; use ResolveRealmTarget for that.
func SameRealmRef(a, b ObjectWithRealmRef) bool {
	refA, refB := a.GetRealmRef(), b.GetRealmRef()

	if RealmRefKind(refA) != RealmRefKind(refB) || refA.Name != refB.Name {
		return false
	}

	if RealmRefKind(refA) == keycloakApi.KeycloakRealmKind {
		return a.GetNamespace() == b.GetNamespace()
	}

	return true
}

// SameRealmTarget reports whether two objects target the same Keycloak realm.
// Objects referencing the same realm CR match without API reads; otherwise both refs
// are resolved to (Keycloak URL, TLS server name, realm name) and compared.
func SameRealmTarget(ctx context.Context, k8sClient client.Client, a, b ObjectWithRealmRef) (bool, error) {
	if SameRealmRef(a, b) {
		return true, nil
	}

	targetA, err := ResolveRealmTarget(ctx, k8sClient, a)
	if err != nil {
		return false, err
	}

	targetB, err := ResolveRealmTarget(ctx, k8sClient, b)
	if err != nil {
		return false, err
	}

	return targetA == targetB, nil
}

// ResolveRealmTarget resolves the realm ref of obj to the Keycloak URL, TLS server name and realm name.
func ResolveRealmTarget(ctx context.Context, k8sClient client.Client, obj ObjectWithRealmRef) (RealmTarget, error) {
	ref := obj.GetRealmRef()

	switch RealmRefKind(ref) {
	case keycloakApi.KeycloakRealmKind:
		realm := &keycloakApi.KeycloakRealm{}
		if err := k8sClient.Get(ctx, types.NamespacedName{
			Name:      ref.Name,
			Namespace: obj.GetNamespace(),
		}, realm); err != nil {
			return RealmTarget{}, fmt.Errorf("unable to get KeycloakRealm %s/%s: %w", obj.GetNamespace(), ref.Name, err)
		}

		target, err := resolveKeycloakServer(ctx, k8sClient, realm.Namespace, realm.Spec.KeycloakRef)
		if err != nil {
			return RealmTarget{}, err
		}

		target.RealmName = realm.Spec.RealmName

		return target, nil

	case keycloakAlpha.ClusterKeycloakRealmKind:
		clusterRealm := &keycloakAlpha.ClusterKeycloakRealm{}
		if err := k8sClient.Get(ctx, types.NamespacedName{Name: ref.Name}, clusterRealm); err != nil {
			return RealmTarget{}, fmt.Errorf("unable to get ClusterKeycloakRealm %s: %w", ref.Name, err)
		}

		target, err := resolveKeycloakServer(ctx, k8sClient, "", common.KeycloakRef{
			Kind: keycloakAlpha.ClusterKeycloakKind,
			Name: clusterRealm.Spec.ClusterKeycloakRef,
		})
		if err != nil {
			return RealmTarget{}, err
		}

		target.RealmName = clusterRealm.Spec.RealmName

		return target, nil

	default:
		return RealmTarget{}, fmt.Errorf("unknown realm kind: %s", RealmRefKind(ref))
	}
}

// resolveKeycloakServer returns a RealmTarget without RealmName for the referenced Keycloak.
func resolveKeycloakServer(
	ctx context.Context,
	k8sClient client.Client,
	namespace string,
	ref common.KeycloakRef,
) (RealmTarget, error) {
	kind := ref.Kind
	if kind == "" {
		kind = keycloakApi.KeycloakKind
	}

	switch kind {
	case keycloakApi.KeycloakKind:
		kc := &keycloakApi.Keycloak{}
		if err := k8sClient.Get(ctx, types.NamespacedName{
			Name:      ref.Name,
			Namespace: namespace,
		}, kc); err != nil {
			return RealmTarget{}, fmt.Errorf("unable to get Keycloak %s/%s: %w", namespace, ref.Name, err)
		}

		return keycloakServer(kc.Spec.Url, kc.Spec.TLSServerName), nil

	case keycloakAlpha.ClusterKeycloakKind:
		kc := &keycloakAlpha.ClusterKeycloak{}
		if err := k8sClient.Get(ctx, types.NamespacedName{Name: ref.Name}, kc); err != nil {
			return RealmTarget{}, fmt.Errorf("unable to get ClusterKeycloak %s: %w", ref.Name, err)
		}

		return keycloakServer(kc.Spec.Url, kc.Spec.TLSServerName), nil

	default:
		return RealmTarget{}, fmt.Errorf("unknown keycloak kind: %s", kind)
	}
}

// keycloakServer returns the canonical endpoint and effective TLS server name.
// An unparsable url or one without a host is kept as written, without trailing slashes.
func keycloakServer(rawURL string, tlsServerName common.TLSServerName) RealmTarget {
	name := canonicalServerName(string(tlsServerName))

	u, err := url.Parse(strings.TrimSpace(rawURL))
	if err != nil || u.Host == "" {
		return RealmTarget{URL: strings.TrimRight(rawURL, "/"), TLSServerName: name}
	}

	scheme := strings.ToLower(u.Scheme)
	host := canonicalServerName(u.Hostname())

	port := u.Port()
	if port == defaultPort(scheme) {
		port = ""
	}

	if name == "" && scheme == "https" {
		name = host
	}

	return RealmTarget{
		URL:           scheme + "://" + joinHostPort(host, port) + strings.TrimRight(u.EscapedPath(), "/"),
		TLSServerName: name,
	}
}

// canonicalServerName matches the name http.Transport sends: IDNA ASCII, lowercase, no trailing dot.
// An IP address is written in its shortest form; an IPv4-mapped IPv6 address becomes IPv4.
func canonicalServerName(host string) string {
	host = strings.TrimSuffix(strings.TrimSpace(host), ".")

	if addr, err := netip.ParseAddr(host); err == nil {
		return addr.Unmap().String()
	}

	if ascii, err := idna.Lookup.ToASCII(host); err == nil {
		host = ascii
	}

	return strings.ToLower(host)
}

func defaultPort(scheme string) string {
	switch scheme {
	case "https":
		return "443"
	case "http":
		return "80"
	default:
		return ""
	}
}

func joinHostPort(host, port string) string {
	if port != "" {
		return net.JoinHostPort(host, port)
	}

	if strings.Contains(host, ":") {
		return "[" + host + "]"
	}

	return host
}

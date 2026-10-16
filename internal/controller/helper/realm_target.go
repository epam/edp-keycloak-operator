package helper

import (
	"context"
	"fmt"
	"strings"

	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/epam/edp-keycloak-operator/api/common"
	keycloakApi "github.com/epam/edp-keycloak-operator/api/v1"
	keycloakAlpha "github.com/epam/edp-keycloak-operator/api/v1alpha1"
)

// RealmTarget is the Keycloak server URL plus realm name an object talks to.
type RealmTarget struct {
	URL       string
	RealmName string
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
// are resolved to (Keycloak URL, realm name) and compared.
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

// ResolveRealmTarget resolves the realm ref of obj to the Keycloak URL and realm name.
// The URL has trailing slashes removed.
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

		url, err := resolveKeycloakURL(ctx, k8sClient, realm.Namespace, realm.Spec.KeycloakRef)
		if err != nil {
			return RealmTarget{}, err
		}

		return RealmTarget{URL: url, RealmName: realm.Spec.RealmName}, nil

	case keycloakAlpha.ClusterKeycloakRealmKind:
		clusterRealm := &keycloakAlpha.ClusterKeycloakRealm{}
		if err := k8sClient.Get(ctx, types.NamespacedName{Name: ref.Name}, clusterRealm); err != nil {
			return RealmTarget{}, fmt.Errorf("unable to get ClusterKeycloakRealm %s: %w", ref.Name, err)
		}

		url, err := resolveKeycloakURL(ctx, k8sClient, "", common.KeycloakRef{
			Kind: keycloakAlpha.ClusterKeycloakKind,
			Name: clusterRealm.Spec.ClusterKeycloakRef,
		})
		if err != nil {
			return RealmTarget{}, err
		}

		return RealmTarget{URL: url, RealmName: clusterRealm.Spec.RealmName}, nil

	default:
		return RealmTarget{}, fmt.Errorf("unknown realm kind: %s", RealmRefKind(ref))
	}
}

func resolveKeycloakURL(
	ctx context.Context,
	k8sClient client.Client,
	namespace string,
	ref common.KeycloakRef,
) (string, error) {
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
			return "", fmt.Errorf("unable to get Keycloak %s/%s: %w", namespace, ref.Name, err)
		}

		return normalizeKeycloakURL(kc.Spec.Url), nil

	case keycloakAlpha.ClusterKeycloakKind:
		kc := &keycloakAlpha.ClusterKeycloak{}
		if err := k8sClient.Get(ctx, types.NamespacedName{Name: ref.Name}, kc); err != nil {
			return "", fmt.Errorf("unable to get ClusterKeycloak %s: %w", ref.Name, err)
		}

		return normalizeKeycloakURL(kc.Spec.Url), nil

	default:
		return "", fmt.Errorf("unknown keycloak kind: %s", kind)
	}
}

func normalizeKeycloakURL(url string) string {
	return strings.TrimRight(url, "/")
}

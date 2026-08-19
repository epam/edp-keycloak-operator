package chain

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

// realmServerTarget is the Keycloak server URL plus realm name a group actually talks to.
type realmServerTarget struct {
	url       string
	realmName string
}

// findOwnerCR returns the KeycloakRealmGroup that already recorded groupID in its
// status.ID, excluding self, and only if it targets the same Keycloak realm.
// Comparing on the Keycloak group ID (rather than re-deriving parent identity from
// spec) is what lets a single check work uniformly for namespace-local ParentGroup
// refs. The ID is unique per Keycloak server, not cluster-wide: cloned instances
// reuse UUIDs, so matching refs is enough to refuse, while differing refs are
// resolved to (spec.url, realmName) from the informer cache.
func findOwnerCR(
	ctx context.Context,
	k8sClient client.Client,
	self *keycloakApi.KeycloakRealmGroup,
	groupID string,
) (*keycloakApi.KeycloakRealmGroup, error) {
	if groupID == "" {
		return nil, nil
	}

	// Resolved once per scan; a resolution error surfaces only when a same-ID
	// candidate actually needs the resolved-target comparison.
	selfTarget, selfResolveErr := resolveRealmTarget(ctx, k8sClient, self)

	var list keycloakApi.KeycloakRealmGroupList
	if err := k8sClient.List(ctx, &list); err != nil {
		return nil, fmt.Errorf("unable to list KeycloakRealmGroup resources: %w", err)
	}

	for i := range list.Items {
		candidate := &list.Items[i]

		if candidate.Namespace == self.Namespace && candidate.Name == self.Name {
			continue
		}

		if candidate.Status.ID != groupID {
			continue
		}

		if sameRealmTarget(self, candidate) {
			return candidate, nil
		}

		candidateTarget, err := resolveRealmTarget(ctx, k8sClient, candidate)
		if err != nil {
			return nil, fmt.Errorf("unable to resolve realm target for ownership check: %w", err)
		}

		if selfResolveErr != nil {
			return nil, fmt.Errorf("unable to resolve realm target for ownership check: %w", selfResolveErr)
		}

		if selfTarget == candidateTarget {
			return candidate, nil
		}
	}

	return nil, nil
}

// sameRealmTarget reports whether two KeycloakRealmGroup resources target the same
// realm instance. Mirrors sameKeycloakInstance on KeycloakRealm: the namespaced
// KeycloakRealm kind is resolved in the group's own namespace, so identity is
// (namespace, name). ClusterKeycloakRealm is cluster-scoped, so identity is the
// name alone.
func sameRealmTarget(a, b *keycloakApi.KeycloakRealmGroup) bool {
	if realmRefKind(a) != realmRefKind(b) || a.Spec.RealmRef.Name != b.Spec.RealmRef.Name {
		return false
	}

	if realmRefKind(a) == keycloakApi.KeycloakRealmKind {
		return a.Namespace == b.Namespace
	}

	return true
}

func realmRefKind(group *keycloakApi.KeycloakRealmGroup) string {
	if group.Spec.RealmRef.Kind == "" {
		return keycloakApi.KeycloakRealmKind
	}

	return group.Spec.RealmRef.Kind
}

func resolveRealmTarget(
	ctx context.Context,
	k8sClient client.Client,
	group *keycloakApi.KeycloakRealmGroup,
) (realmServerTarget, error) {
	name := group.Spec.RealmRef.Name

	switch realmRefKind(group) {
	case keycloakApi.KeycloakRealmKind:
		realm := &keycloakApi.KeycloakRealm{}
		if err := k8sClient.Get(ctx, types.NamespacedName{
			Name:      name,
			Namespace: group.Namespace,
		}, realm); err != nil {
			return realmServerTarget{}, fmt.Errorf("unable to get KeycloakRealm %s/%s: %w", group.Namespace, name, err)
		}

		url, err := resolveKeycloakURL(ctx, k8sClient, realm.Namespace, realm.Spec.KeycloakRef)
		if err != nil {
			return realmServerTarget{}, err
		}

		return realmServerTarget{url: url, realmName: realm.Spec.RealmName}, nil

	case keycloakAlpha.ClusterKeycloakRealmKind:
		clusterRealm := &keycloakAlpha.ClusterKeycloakRealm{}
		if err := k8sClient.Get(ctx, types.NamespacedName{Name: name}, clusterRealm); err != nil {
			return realmServerTarget{}, fmt.Errorf("unable to get ClusterKeycloakRealm %s: %w", name, err)
		}

		url, err := resolveKeycloakURL(ctx, k8sClient, "", common.KeycloakRef{
			Kind: keycloakAlpha.ClusterKeycloakKind,
			Name: clusterRealm.Spec.ClusterKeycloakRef,
		})
		if err != nil {
			return realmServerTarget{}, err
		}

		return realmServerTarget{url: url, realmName: clusterRealm.Spec.RealmName}, nil

	default:
		return realmServerTarget{}, fmt.Errorf("unknown realm kind: %s", realmRefKind(group))
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

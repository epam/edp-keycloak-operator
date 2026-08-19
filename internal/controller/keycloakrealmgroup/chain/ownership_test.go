package chain

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/epam/edp-keycloak-operator/api/common"
	keycloakApi "github.com/epam/edp-keycloak-operator/api/v1"
	keycloakAlpha "github.com/epam/edp-keycloak-operator/api/v1alpha1"
)

func TestSameRealmTarget(t *testing.T) {
	namespaced := func(ns, realm string) *keycloakApi.KeycloakRealmGroup {
		g := &keycloakApi.KeycloakRealmGroup{
			ObjectMeta: metav1.ObjectMeta{Name: "group", Namespace: ns},
		}
		g.Spec.RealmRef.Kind = keycloakApi.KeycloakRealmKind
		g.Spec.RealmRef.Name = realm

		return g
	}

	t.Run("same namespaced realm in same namespace", func(t *testing.T) {
		assert.True(t, sameRealmTarget(namespaced("ns-a", testRealmRefName), namespaced("ns-a", testRealmRefName)))
	})

	t.Run("same namespaced realm name in another namespace", func(t *testing.T) {
		assert.False(t, sameRealmTarget(namespaced("ns-a", testRealmRefName), namespaced("ns-b", testRealmRefName)))
	})

	t.Run("empty kind defaults to KeycloakRealm", func(t *testing.T) {
		a := namespaced("ns-a", testRealmRefName)
		b := namespaced("ns-a", testRealmRefName)
		b.Spec.RealmRef.Kind = ""

		assert.True(t, sameRealmTarget(a, b))
	})

	t.Run("cluster realm is cluster-wide", func(t *testing.T) {
		a := namespaced("ns-a", "shared")
		b := namespaced("ns-b", "shared")
		a.Spec.RealmRef.Kind = keycloakAlpha.ClusterKeycloakRealmKind
		b.Spec.RealmRef.Kind = keycloakAlpha.ClusterKeycloakRealmKind

		assert.True(t, sameRealmTarget(a, b))
	})
}

func TestNormalizeKeycloakURL(t *testing.T) {
	assert.Equal(t, "http://kc.example", normalizeKeycloakURL("http://kc.example/"))
	assert.Equal(t, "http://kc.example", normalizeKeycloakURL("http://kc.example"))
}

func TestResolveRealmTarget_ClusterKeycloakRealm(t *testing.T) {
	group := &keycloakApi.KeycloakRealmGroup{
		ObjectMeta: metav1.ObjectMeta{Name: "g", Namespace: "ns-a"},
	}
	group.Spec.RealmRef.Kind = keycloakAlpha.ClusterKeycloakRealmKind
	group.Spec.RealmRef.Name = "crealm"

	got, err := resolveRealmTarget(context.Background(), newFakeK8sClient(t,
		&keycloakAlpha.ClusterKeycloak{
			ObjectMeta: metav1.ObjectMeta{Name: "ckc"},
			Spec:       keycloakAlpha.ClusterKeycloakSpec{Url: "http://cluster-kc.example/"},
		},
		&keycloakAlpha.ClusterKeycloakRealm{
			ObjectMeta: metav1.ObjectMeta{Name: "crealm"},
			Spec: keycloakAlpha.ClusterKeycloakRealmSpec{
				ClusterKeycloakRef: "ckc",
				RealmName:          "shared",
			},
		},
	), group)
	require.NoError(t, err)
	assert.Equal(t, realmServerTarget{url: "http://cluster-kc.example", realmName: "shared"}, got)
}

func TestFindOwnerCR_SharedURLIsConflict(t *testing.T) {
	self := &keycloakApi.KeycloakRealmGroup{
		ObjectMeta: metav1.ObjectMeta{Name: testCRNameB, Namespace: "ns-a"},
	}
	self.Spec.RealmRef.Kind = keycloakApi.KeycloakRealmKind
	self.Spec.RealmRef.Name = testRealmRefName

	owner := &keycloakApi.KeycloakRealmGroup{
		ObjectMeta: metav1.ObjectMeta{Name: testCRNameA, Namespace: "ns-b"},
		Status:     keycloakApi.KeycloakRealmGroupStatus{ID: "gid"},
	}
	owner.Spec.RealmRef.Kind = keycloakApi.KeycloakRealmKind
	owner.Spec.RealmRef.Name = testRealmRefName

	objs := make([]client.Object, 0, 5)
	objs = append(objs, owner)
	objs = append(objs, namespacedKeycloakStack("ns-a", testRealmRefName, "http://shared.example")...)
	objs = append(objs, namespacedKeycloakStack("ns-b", testRealmRefName, "http://shared.example/")...)

	got, err := findOwnerCR(context.Background(), newFakeK8sClient(t, objs...), self, "gid")
	require.NoError(t, err)
	require.NotNil(t, got)
	assert.Equal(t, "ns-b", got.Namespace)
	assert.Equal(t, testCRNameA, got.Name)
}

func TestFindOwnerCR_CloneDifferentURLIsNotConflict(t *testing.T) {
	self := &keycloakApi.KeycloakRealmGroup{
		ObjectMeta: metav1.ObjectMeta{Name: testCRNameB, Namespace: "ns-a"},
	}
	self.Spec.RealmRef.Kind = keycloakApi.KeycloakRealmKind
	self.Spec.RealmRef.Name = testRealmRefName

	owner := &keycloakApi.KeycloakRealmGroup{
		ObjectMeta: metav1.ObjectMeta{Name: testCRNameA, Namespace: "ns-b"},
		Status:     keycloakApi.KeycloakRealmGroupStatus{ID: "gid"},
	}
	owner.Spec.RealmRef.Kind = keycloakApi.KeycloakRealmKind
	owner.Spec.RealmRef.Name = testRealmRefName

	objs := make([]client.Object, 0, 5)
	objs = append(objs, owner)
	objs = append(objs, namespacedKeycloakStack("ns-a", testRealmRefName, "http://kc-a.example")...)
	objs = append(objs, namespacedKeycloakStack("ns-b", testRealmRefName, "http://kc-b.example")...)

	got, err := findOwnerCR(context.Background(), newFakeK8sClient(t, objs...), self, "gid")
	require.NoError(t, err)
	assert.Nil(t, got)
}

func TestFindOwnerCR_ResolutionFailure(t *testing.T) {
	self := &keycloakApi.KeycloakRealmGroup{
		ObjectMeta: metav1.ObjectMeta{Name: testCRNameB, Namespace: "ns-a"},
	}
	self.Spec.RealmRef.Name = testRealmRefName

	owner := &keycloakApi.KeycloakRealmGroup{
		ObjectMeta: metav1.ObjectMeta{Name: testCRNameA, Namespace: "ns-b"},
		Status:     keycloakApi.KeycloakRealmGroupStatus{ID: "gid"},
	}
	owner.Spec.RealmRef.Name = testRealmRefName

	_, err := findOwnerCR(context.Background(), newFakeK8sClient(t, owner), self, "gid")
	assert.ErrorContains(t, err, "unable to resolve realm target for ownership check")
}

func TestResolveKeycloakURL_EmptyKindDefaultsToKeycloak(t *testing.T) {
	url, err := resolveKeycloakURL(
		context.Background(),
		newFakeK8sClient(t, &keycloakApi.Keycloak{
			ObjectMeta: metav1.ObjectMeta{Name: "kc", Namespace: "ns-a"},
			Spec:       keycloakApi.KeycloakSpec{Url: "http://kc.example"},
		}),
		"ns-a",
		common.KeycloakRef{Name: "kc"},
	)
	require.NoError(t, err)
	assert.Equal(t, "http://kc.example", url)
}

func TestResolveRealmTarget_UnknownKind(t *testing.T) {
	group := &keycloakApi.KeycloakRealmGroup{}
	group.Spec.RealmRef.Kind = "Nope"

	_, err := resolveRealmTarget(context.Background(), newFakeK8sClient(t), group)
	assert.ErrorContains(t, err, "unknown realm kind")
}

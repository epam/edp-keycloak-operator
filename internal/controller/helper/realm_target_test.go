package helper

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	"github.com/epam/edp-keycloak-operator/api/common"
	keycloakApi "github.com/epam/edp-keycloak-operator/api/v1"
	keycloakAlpha "github.com/epam/edp-keycloak-operator/api/v1alpha1"
)

const testRealmTargetRef = "test"

func newRealmTargetClient(t *testing.T, objs ...client.Object) client.Client {
	t.Helper()

	scheme := runtime.NewScheme()
	require.NoError(t, keycloakApi.AddToScheme(scheme))
	require.NoError(t, keycloakAlpha.AddToScheme(scheme))

	return fake.NewClientBuilder().WithScheme(scheme).WithObjects(objs...).Build()
}

func namespacedRealmStack(ns, realmCR, realmName, url string) []client.Object {
	return []client.Object{
		&keycloakApi.Keycloak{
			ObjectMeta: metav1.ObjectMeta{Name: "keycloak", Namespace: ns},
			Spec:       keycloakApi.KeycloakSpec{Url: url},
		},
		&keycloakApi.KeycloakRealm{
			ObjectMeta: metav1.ObjectMeta{Name: realmCR, Namespace: ns},
			Spec: keycloakApi.KeycloakRealmSpec{
				RealmName:   realmName,
				KeycloakRef: common.KeycloakRef{Kind: keycloakApi.KeycloakKind, Name: "keycloak"},
			},
		},
	}
}

func groupWithRealmRef(ns, kind, realm string) *keycloakApi.KeycloakRealmGroup {
	g := &keycloakApi.KeycloakRealmGroup{
		ObjectMeta: metav1.ObjectMeta{Name: "group", Namespace: ns},
	}
	g.Spec.RealmRef = common.RealmRef{Kind: kind, Name: realm}

	return g
}

func TestRealmRefKind(t *testing.T) {
	assert.Equal(t, keycloakApi.KeycloakRealmKind, RealmRefKind(common.RealmRef{}))
	assert.Equal(t, keycloakAlpha.ClusterKeycloakRealmKind, RealmRefKind(common.RealmRef{Kind: keycloakAlpha.ClusterKeycloakRealmKind}))
}

func TestSameRealmRef(t *testing.T) {
	tests := []struct {
		name string
		a, b ObjectWithRealmRef
		want bool
	}{
		{
			name: "same namespaced realm in same namespace",
			a:    groupWithRealmRef("ns-a", keycloakApi.KeycloakRealmKind, testRealmTargetRef),
			b:    groupWithRealmRef("ns-a", keycloakApi.KeycloakRealmKind, testRealmTargetRef),
			want: true,
		},
		{
			name: "same namespaced realm name in another namespace",
			a:    groupWithRealmRef("ns-a", keycloakApi.KeycloakRealmKind, testRealmTargetRef),
			b:    groupWithRealmRef("ns-b", keycloakApi.KeycloakRealmKind, testRealmTargetRef),
			want: false,
		},
		{
			name: "empty kind defaults to KeycloakRealm",
			a:    groupWithRealmRef("ns-a", keycloakApi.KeycloakRealmKind, testRealmTargetRef),
			b:    groupWithRealmRef("ns-a", "", testRealmTargetRef),
			want: true,
		},
		{
			name: "different name",
			a:    groupWithRealmRef("ns-a", keycloakApi.KeycloakRealmKind, testRealmTargetRef),
			b:    groupWithRealmRef("ns-a", keycloakApi.KeycloakRealmKind, "other"),
			want: false,
		},
		{
			name: "different kind",
			a:    groupWithRealmRef("ns-a", "", "shared"),
			b:    groupWithRealmRef("ns-a", keycloakAlpha.ClusterKeycloakRealmKind, "shared"),
			want: false,
		},
		{
			name: "cluster realm is cluster-wide",
			a:    groupWithRealmRef("ns-a", keycloakAlpha.ClusterKeycloakRealmKind, "shared"),
			b:    groupWithRealmRef("ns-b", keycloakAlpha.ClusterKeycloakRealmKind, "shared"),
			want: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, SameRealmRef(tt.a, tt.b))
			assert.Equal(t, tt.want, SameRealmRef(tt.b, tt.a))
		})
	}
}

func TestSameRealmTarget(t *testing.T) {
	clusterStack := []client.Object{
		&keycloakAlpha.ClusterKeycloak{
			ObjectMeta: metav1.ObjectMeta{Name: "ckc"},
			Spec:       keycloakAlpha.ClusterKeycloakSpec{Url: "http://kc.example/"},
		},
		&keycloakAlpha.ClusterKeycloakRealm{
			ObjectMeta: metav1.ObjectMeta{Name: "crealm"},
			Spec:       keycloakAlpha.ClusterKeycloakRealmSpec{ClusterKeycloakRef: "ckc", RealmName: "shared"},
		},
	}

	t.Run("same ref needs no reads", func(t *testing.T) {
		got, err := SameRealmTarget(context.Background(), newRealmTargetClient(t),
			groupWithRealmRef("ns-a", "", testRealmTargetRef),
			groupWithRealmRef("ns-a", keycloakApi.KeycloakRealmKind, testRealmTargetRef))
		require.NoError(t, err)
		assert.True(t, got)
	})

	t.Run("different CR kinds resolving to the same realm", func(t *testing.T) {
		objs := append(namespacedRealmStack("ns-a", testRealmTargetRef, "shared", "http://kc.example"), clusterStack...)

		got, err := SameRealmTarget(context.Background(), newRealmTargetClient(t, objs...),
			groupWithRealmRef("ns-a", keycloakApi.KeycloakRealmKind, testRealmTargetRef),
			groupWithRealmRef("ns-a", keycloakAlpha.ClusterKeycloakRealmKind, "crealm"))
		require.NoError(t, err)
		assert.True(t, got)
	})

	t.Run("different realm names", func(t *testing.T) {
		objs := append(namespacedRealmStack("ns-a", testRealmTargetRef, "other", "http://kc.example"), clusterStack...)

		got, err := SameRealmTarget(context.Background(), newRealmTargetClient(t, objs...),
			groupWithRealmRef("ns-a", keycloakApi.KeycloakRealmKind, testRealmTargetRef),
			groupWithRealmRef("ns-a", keycloakAlpha.ClusterKeycloakRealmKind, "crealm"))
		require.NoError(t, err)
		assert.False(t, got)
	})

	t.Run("resolution failure", func(t *testing.T) {
		_, err := SameRealmTarget(context.Background(), newRealmTargetClient(t),
			groupWithRealmRef("ns-a", keycloakApi.KeycloakRealmKind, testRealmTargetRef),
			groupWithRealmRef("ns-a", keycloakApi.KeycloakRealmKind, "missing"))
		assert.ErrorContains(t, err, "unable to get KeycloakRealm")
	})
}

func TestNormalizeKeycloakURL(t *testing.T) {
	assert.Equal(t, "http://kc.example", normalizeKeycloakURL("http://kc.example/"))
	assert.Equal(t, "http://kc.example", normalizeKeycloakURL("http://kc.example"))
}

func TestResolveRealmTarget_KeycloakRealm(t *testing.T) {
	got, err := ResolveRealmTarget(context.Background(),
		newRealmTargetClient(t, namespacedRealmStack("ns-a", testRealmTargetRef, "realm-a", "http://kc.example/")...),
		groupWithRealmRef("ns-a", "", testRealmTargetRef))
	require.NoError(t, err)
	assert.Equal(t, RealmTarget{URL: "http://kc.example", RealmName: "realm-a"}, got)
}

func TestResolveRealmTarget_ClusterKeycloakRealm(t *testing.T) {
	got, err := ResolveRealmTarget(context.Background(), newRealmTargetClient(t,
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
	), groupWithRealmRef("ns-a", keycloakAlpha.ClusterKeycloakRealmKind, "crealm"))
	require.NoError(t, err)
	assert.Equal(t, RealmTarget{URL: "http://cluster-kc.example", RealmName: "shared"}, got)
}

func TestResolveRealmTarget_UnknownKind(t *testing.T) {
	_, err := ResolveRealmTarget(context.Background(), newRealmTargetClient(t), groupWithRealmRef("", "Nope", ""))
	assert.ErrorContains(t, err, "unknown realm kind")
}

func TestResolveKeycloakURL_EmptyKindDefaultsToKeycloak(t *testing.T) {
	url, err := resolveKeycloakURL(
		context.Background(),
		newRealmTargetClient(t, &keycloakApi.Keycloak{
			ObjectMeta: metav1.ObjectMeta{Name: "kc", Namespace: "ns-a"},
			Spec:       keycloakApi.KeycloakSpec{Url: "http://kc.example"},
		}),
		"ns-a",
		common.KeycloakRef{Name: "kc"},
	)
	require.NoError(t, err)
	assert.Equal(t, "http://kc.example", url)
}

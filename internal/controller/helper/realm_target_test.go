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

func TestKeycloakServer(t *testing.T) {
	tests := []struct {
		name          string
		url           string
		tlsServerName string
		want          RealmTarget
	}{
		{name: "trailing slash", url: "http://kc.example/", want: RealmTarget{URL: "http://kc.example"}},
		{name: "scheme and host case", url: "HTTP://KC.Example", want: RealmTarget{URL: "http://kc.example"}},
		{name: "http default port", url: "http://kc.example:80", want: RealmTarget{URL: "http://kc.example"}},
		{name: "http on 443 keeps the port", url: "http://kc.example:443", want: RealmTarget{URL: "http://kc.example:443"}},
		{
			name: "https default port and url host as name",
			url:  "https://kc.example:443/",
			want: RealmTarget{URL: "https://kc.example", TLSServerName: "kc.example"},
		},
		{
			name: "non-default port",
			url:  "https://kc.example:8443",
			want: RealmTarget{URL: "https://kc.example:8443", TLSServerName: "kc.example"},
		},
		{
			name: "path keeps its case",
			url:  "https://kc.example/Auth/",
			want: RealmTarget{URL: "https://kc.example/Auth", TLSServerName: "kc.example"},
		},
		{
			name: "userinfo and query dropped",
			url:  "https://user@kc.example/?x=1",
			want: RealmTarget{URL: "https://kc.example", TLSServerName: "kc.example"},
		},
		{
			name: "IDNA host",
			url:  "https://bücher.example",
			want: RealmTarget{URL: "https://xn--bcher-kva.example", TLSServerName: "xn--bcher-kva.example"},
		},
		{
			name: "IPv6 host in shortest form",
			url:  "https://[0:0::1]",
			want: RealmTarget{URL: "https://[::1]", TLSServerName: "::1"},
		},
		{
			name: "IPv6 host with port",
			url:  "https://[::1]:8443",
			want: RealmTarget{URL: "https://[::1]:8443", TLSServerName: "::1"},
		},
		{
			name: "IPv4-mapped host",
			url:  "https://[::ffff:10.0.0.1]",
			want: RealmTarget{URL: "https://10.0.0.1", TLSServerName: "10.0.0.1"},
		},
		{
			name:          "explicit name canonicalized",
			url:           "https://kc-svc.ns.svc:8443",
			tlsServerName: "KC.Example.",
			want:          RealmTarget{URL: "https://kc-svc.ns.svc:8443", TLSServerName: "kc.example"},
		},
		{
			name:          "explicit IP name in shortest form",
			url:           "https://kc-svc.ns.svc",
			tlsServerName: "0:0::1",
			want:          RealmTarget{URL: "https://kc-svc.ns.svc", TLSServerName: "::1"},
		},
		{name: "url without host kept as written", url: "kc.example/", want: RealmTarget{URL: "kc.example"}},
		{name: "unparsable url kept as written", url: "http://kc.example:port/", want: RealmTarget{URL: "http://kc.example:port"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, keycloakServer(tt.url, common.TLSServerName(tt.tlsServerName)))
		})
	}
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

func TestResolveKeycloakServer_EmptyKindDefaultsToKeycloak(t *testing.T) {
	target, err := resolveKeycloakServer(
		context.Background(),
		newRealmTargetClient(t, &keycloakApi.Keycloak{
			ObjectMeta: metav1.ObjectMeta{Name: "kc", Namespace: "ns-a"},
			Spec:       keycloakApi.KeycloakSpec{Url: "http://kc.example"},
		}),
		"ns-a",
		common.KeycloakRef{Name: "kc"},
	)
	require.NoError(t, err)
	assert.Equal(t, RealmTarget{URL: "http://kc.example"}, target)
}

// Behind an SNI-routing front end, one url with two TLS server names reaches two servers.
// An unset name is the url host.
func TestSameRealmTarget_TLSServerNameSelectsTheServer(t *testing.T) {
	keycloakWithName := func(name, url, tlsServerName string) *keycloakApi.Keycloak {
		return &keycloakApi.Keycloak{
			ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: "ns-a"},
			Spec: keycloakApi.KeycloakSpec{
				Url:           url,
				TLSServerName: common.TLSServerName(tlsServerName),
			},
		}
	}
	realmOn := func(name, keycloak string) *keycloakApi.KeycloakRealm {
		return &keycloakApi.KeycloakRealm{
			ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: "ns-a"},
			Spec: keycloakApi.KeycloakRealmSpec{
				RealmName:   "shared",
				KeycloakRef: common.KeycloakRef{Kind: keycloakApi.KeycloakKind, Name: keycloak},
			},
		}
	}

	const gateway = "https://sni-gateway.example:8443"

	tests := []struct {
		name       string
		urlA       string
		nameA      string
		urlB       string
		nameB      string
		wantSameAs bool
	}{
		{name: "different names", urlA: gateway, nameA: "kc-a.example.com", urlB: gateway, nameB: "kc-b.example.com"},
		{
			name: "same name in another spelling",
			urlA: gateway, nameA: "kc-a.example.com",
			urlB: gateway, nameB: "KC-A.example.com.",
			wantSameAs: true,
		},
		{
			name: "unset name and the url host",
			urlA: "https://kc.example.com", nameA: "",
			urlB: "https://kc.example.com/", nameB: "kc.example.com",
			wantSameAs: true,
		},
		{
			name: "unset name and another name",
			urlA: "https://kc.example.com", nameA: "",
			urlB: "https://kc.example.com", nameB: "kc-b.example.com",
		},
		{
			name: "unset name and the url IP",
			urlA: "https://10.0.0.1:8443", nameA: "",
			urlB: "https://10.0.0.1:8443", nameB: "10.0.0.1",
			wantSameAs: true,
		},
		{
			name: "default port and host case",
			urlA: "https://KC.example.com:443", nameA: "",
			urlB: "https://kc.example.com", nameB: "",
			wantSameAs: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cl := newRealmTargetClient(t,
				keycloakWithName("kc-a", tt.urlA, tt.nameA), keycloakWithName("kc-b", tt.urlB, tt.nameB),
				realmOn("realm-a", "kc-a"), realmOn("realm-b", "kc-b"))

			got, err := SameRealmTarget(context.Background(), cl,
				groupWithRealmRef("ns-a", keycloakApi.KeycloakRealmKind, "realm-a"),
				groupWithRealmRef("ns-a", keycloakApi.KeycloakRealmKind, "realm-b"))
			require.NoError(t, err)
			assert.Equal(t, tt.wantSameAs, got)
		})
	}
}

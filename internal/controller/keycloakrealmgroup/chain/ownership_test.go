package chain

import (
	"testing"

	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

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
		assert.True(t, sameRealmTarget(namespaced("ns-a", "restos"), namespaced("ns-a", "restos")))
	})

	t.Run("same namespaced realm name in another namespace", func(t *testing.T) {
		assert.False(t, sameRealmTarget(namespaced("ns-a", "restos"), namespaced("ns-b", "restos")))
	})

	t.Run("empty kind defaults to KeycloakRealm", func(t *testing.T) {
		a := namespaced("ns-a", "restos")
		b := namespaced("ns-a", "restos")
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

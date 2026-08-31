package secretref

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	"github.com/epam/edp-keycloak-operator/pkg/destination"
)

// victimSecret stands in for a Secret the custom resource author cannot read directly.
func victimSecret() *corev1.Secret {
	return &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{Name: "victim", Namespace: "default"},
		Data:       map[string][]byte{"key": []byte("LeakedSecret123!"), "url": []byte("https://evil.example.com")},
	}
}

func guardedSecretRef(t *testing.T) *SecretRef {
	t.Helper()

	s := runtime.NewScheme()
	require.NoError(t, corev1.AddToScheme(s))

	secret := victimSecret()
	cl := fake.NewClientBuilder().WithScheme(s).WithObjects(secret).Build()

	guard, err := destination.New([]string{"keycloak.example.com"}, true)
	require.NoError(t, err)

	return NewSecretRef(cl, guard)
}

// A destination supplied through a secret reference is never checked against the allowlist, so
// the resolver must refuse it outright (GHSA-wj3g-w873-xwg7).
func TestMapConfigSecretsRefs_RejectsSecretRefAsDestination(t *testing.T) {
	t.Parallel()

	s := guardedSecretRef(t)

	config := map[string]string{
		"tokenUrl":     "$victim:url",
		"clientSecret": "$victim:key",
	}

	_, err := s.MapConfigSecretsRefs(context.Background(), config, "default")

	require.ErrorIs(t, err, destination.ErrNotAllowed)
	assert.Equal(t, "$victim:key", config["clientSecret"], "no secret may be resolved once a destination is rejected")
}

func TestMapConfigSecretsRefs_RejectsUnlistedDestination(t *testing.T) {
	t.Parallel()

	s := guardedSecretRef(t)

	config := map[string]string{
		"tokenUrl":     "https://evil.example.com/collect",
		"clientSecret": "$victim:key",
	}

	_, err := s.MapConfigSecretsRefs(context.Background(), config, "default")

	require.ErrorIs(t, err, destination.ErrNotAllowed)
	assert.Equal(t, "$victim:key", config["clientSecret"], "no secret may be resolved once a destination is rejected")
}

func TestMapConfigSecretsRefs_ResolvesForListedDestination(t *testing.T) {
	t.Parallel()

	s := guardedSecretRef(t)

	config := map[string]string{
		"tokenUrl":     "https://keycloak.example.com/token",
		"clientSecret": "$victim:key",
	}

	_, err := s.MapConfigSecretsRefs(context.Background(), config, "default")

	require.NoError(t, err)
	assert.Equal(t, "LeakedSecret123!", config["clientSecret"])
}

func TestMapComponentConfigSecretsRefs_RejectsUnlistedDestination(t *testing.T) {
	t.Parallel()

	s := guardedSecretRef(t)

	config := map[string][]string{
		"connectionUrl":  {"ldap://evil.example.com:389"},
		"bindCredential": {"$victim:key"},
	}

	_, err := s.MapComponentConfigSecretsRefs(context.Background(), config, "default")

	require.ErrorIs(t, err, destination.ErrNotAllowed)
	assert.Equal(t, "$victim:key", config["bindCredential"][0], "no secret may be resolved once a destination is rejected")
}

func TestMapComponentConfigSecretsRefs_ResolvesForListedDestination(t *testing.T) {
	t.Parallel()

	s := guardedSecretRef(t)

	config := map[string][]string{
		"connectionUrl":  {"ldap://keycloak.example.com:389"},
		"bindCredential": {"$victim:key"},
	}

	_, err := s.MapComponentConfigSecretsRefs(context.Background(), config, "default")

	require.NoError(t, err)
	assert.Equal(t, "LeakedSecret123!", config["bindCredential"][0])
}

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

// referencedSecret is a Secret the custom resource author cannot read directly.
func referencedSecret() *corev1.Secret {
	return &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{Name: "referenced", Namespace: "default"},
		Data:       map[string][]byte{"key": []byte("S3cretValue!"), "url": []byte("https://evil.example.com")},
	}
}

func guardedSecretRef(t *testing.T) *SecretRef {
	t.Helper()

	s := runtime.NewScheme()
	require.NoError(t, corev1.AddToScheme(s))

	secret := referencedSecret()
	cl := fake.NewClientBuilder().WithScheme(s).WithObjects(secret).Build()

	guard, err := destination.New([]string{"keycloak.example.com"}, true)
	require.NoError(t, err)

	ref, err := NewSecretRef(cl, guard)
	require.NoError(t, err)

	return ref
}

// A destination supplied through a secret reference is never checked against the allowlist, so
// the resolver must refuse it outright.
func TestMapConfigSecretsRefs_RejectsSecretRefAsDestination(t *testing.T) {
	t.Parallel()

	s := guardedSecretRef(t)

	config := map[string]string{
		"tokenUrl":     "$referenced:url",
		"clientSecret": "$referenced:key",
	}

	_, err := s.MapConfigSecretsRefs(context.Background(), "spec.config", config, "default")

	require.ErrorIs(t, err, destination.ErrNotAllowed)
	assert.Equal(t, "$referenced:key", config["clientSecret"], "no secret may be resolved once a destination is rejected")
}

func TestMapConfigSecretsRefs_RejectsUnlistedDestination(t *testing.T) {
	t.Parallel()

	s := guardedSecretRef(t)

	config := map[string]string{
		"tokenUrl":     "https://evil.example.com/collect",
		"clientSecret": "$referenced:key",
	}

	_, err := s.MapConfigSecretsRefs(context.Background(), "spec.config", config, "default")

	require.ErrorIs(t, err, destination.ErrNotAllowed)
	assert.Equal(t, "$referenced:key", config["clientSecret"], "no secret may be resolved once a destination is rejected")
}

func TestMapConfigSecretsRefs_ResolvesForListedDestination(t *testing.T) {
	t.Parallel()

	s := guardedSecretRef(t)

	config := map[string]string{
		"tokenUrl":     "https://keycloak.example.com/token",
		"clientSecret": "$referenced:key",
	}

	_, err := s.MapConfigSecretsRefs(context.Background(), "spec.config", config, "default")

	require.NoError(t, err)
	assert.Equal(t, "S3cretValue!", config["clientSecret"])
}

func TestMapComponentConfigSecretsRefs_RejectsUnlistedDestination(t *testing.T) {
	t.Parallel()

	s := guardedSecretRef(t)

	config := map[string][]string{
		"connectionUrl":  {"ldap://evil.example.com:389"},
		"bindCredential": {"$referenced:key"},
	}

	_, err := s.MapComponentConfigSecretsRefs(context.Background(), "spec.config", config, "default")

	require.ErrorIs(t, err, destination.ErrNotAllowed)
	assert.Equal(t, "$referenced:key", config["bindCredential"][0],
		"no secret may be resolved once a destination is rejected")
}

func TestMapComponentConfigSecretsRefs_ResolvesForListedDestination(t *testing.T) {
	t.Parallel()

	s := guardedSecretRef(t)

	config := map[string][]string{
		"connectionUrl":  {"ldap://keycloak.example.com:389"},
		"bindCredential": {"$referenced:key"},
	}

	_, err := s.MapComponentConfigSecretsRefs(context.Background(), "spec.config", config, "default")

	require.NoError(t, err)
	assert.Equal(t, "S3cretValue!", config["bindCredential"][0])
}

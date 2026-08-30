package secretref

import (
	"crypto/sha256"
	"encoding/hex"
	"testing"

	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestHasAnySecretRef(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		values   []string
		expected bool
	}{
		{name: "nil slice", values: nil, expected: false},
		{name: "empty slice", values: []string{}, expected: false},
		{name: "no secret ref", values: []string{"plain", "also-plain"}, expected: false},
		{name: "secret ref among plain values", values: []string{"plain", "$secret:key"}, expected: true},
		{name: "all secret refs", values: []string{"$a:b", "$c:d"}, expected: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, tt.expected, HasAnySecretRef(tt.values))
		})
	}
}

func TestValuesHash_EmptyInputIsConstant(t *testing.T) {
	t.Parallel()

	emptyDigest := hex.EncodeToString(sha256.New().Sum(nil))

	assert.Equal(t, emptyDigest, ValuesHash(nil))
	assert.Equal(t, emptyDigest, ValuesHash(map[string][]string{}))
}

func TestValuesHash_ValueOrderIsSignificant(t *testing.T) {
	t.Parallel()

	forward := ValuesHash(map[string][]string{"k": {"x", "y"}})
	reversed := ValuesHash(map[string][]string{"k": {"y", "x"}})

	assert.NotEqual(t, forward, reversed)
}

func TestValuesHash_CountPrefixDisambiguatesValueGrouping(t *testing.T) {
	t.Parallel()

	// Same concatenated characters ("ab"), grouped as one value vs two: the explicit
	// per-key value count keeps these from hashing identically.
	oneValue := ValuesHash(map[string][]string{"k": {"ab"}})
	twoValues := ValuesHash(map[string][]string{"k": {"a", "b"}})

	assert.NotEqual(t, oneValue, twoValues)
}

func TestValuesHash_Deterministic(t *testing.T) {
	t.Parallel()

	input := map[string][]string{
		"clientSecret":   {"secret:s:key@uid-1@100"},
		"bindCredential": {"secret:s:a@uid-1@100", "secret:s:b@uid-1@100"},
	}

	assert.Equal(t, ValuesHash(input), ValuesHash(input))
	assert.NotEqual(t,
		ValuesHash(map[string][]string{"clientSecret": {"secret:s:key@uid-1@100"}}),
		ValuesHash(map[string][]string{"clientSecret": {"secret:s:key@uid-1@101"}}),
	)
}

func TestValuesHashSingle_MatchesWrappedValuesHash(t *testing.T) {
	t.Parallel()

	single := map[string]string{"clientSecret": "secret:s:key@uid-1@100"}
	wrapped := map[string][]string{"clientSecret": {"secret:s:key@uid-1@100"}}

	assert.Equal(t, ValuesHash(wrapped), ValuesHashSingle(single))
}

func TestSecretKeyVersion(t *testing.T) {
	t.Parallel()

	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:            "my-secret",
			UID:             "uid-1",
			ResourceVersion: "100",
		},
		Data: map[string][]byte{"key": []byte("top-secret-value")},
	}

	token := SecretKeyVersion(secret, "key")

	assert.Equal(t, "secret:my-secret:key@uid-1@100", token)
	assert.NotContains(t, token, "top-secret-value")

	rotated := secret.DeepCopy()
	rotated.ResourceVersion = "101"
	assert.NotEqual(t, token, SecretKeyVersion(rotated, "key"))

	recreated := secret.DeepCopy()
	recreated.UID = "uid-2"
	assert.NotEqual(t, token, SecretKeyVersion(recreated, "key"))

	assert.NotEqual(t, token, SecretKeyVersion(secret, "other-key"))
}

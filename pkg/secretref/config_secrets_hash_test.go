package secretref

import (
	"crypto/sha256"
	"encoding/hex"
	"testing"

	"github.com/stretchr/testify/assert"
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

func TestConfigSecretsHash_EmptyInputIsConstant(t *testing.T) {
	t.Parallel()

	emptyDigest := hex.EncodeToString(sha256.New().Sum(nil))

	assert.Equal(t, emptyDigest, ConfigSecretsHash(nil, nil))
	assert.Equal(t, emptyDigest, ConfigSecretsHash(map[string][]string{}, map[string][]string{}))
}

func TestConfigSecretsHash_IgnoresKeysWithoutAnySecretRef(t *testing.T) {
	t.Parallel()

	rawCfg := map[string][]string{
		"plainKey":  {"literal-value"},
		"secretKey": {"$secret:key"},
	}

	base := ConfigSecretsHash(rawCfg, map[string][]string{
		"plainKey":  {"literal-value"},
		"secretKey": {"resolved-1"},
	})

	// Changing the resolved value of a key with no secret ref must not affect the hash.
	sameAfterPlainChange := ConfigSecretsHash(rawCfg, map[string][]string{
		"plainKey":  {"changed-but-irrelevant"},
		"secretKey": {"resolved-1"},
	})
	assert.Equal(t, base, sameAfterPlainChange)

	// Changing the resolved value of the secret-ref key must affect the hash.
	differsAfterSecretChange := ConfigSecretsHash(rawCfg, map[string][]string{
		"plainKey":  {"literal-value"},
		"secretKey": {"resolved-2"},
	})
	assert.NotEqual(t, base, differsAfterSecretChange)
}

func TestConfigSecretsHash_ValueOrderIsSignificant(t *testing.T) {
	t.Parallel()

	rawCfg := map[string][]string{"k": {"$s:1", "$s:2"}}

	forward := ConfigSecretsHash(rawCfg, map[string][]string{"k": {"x", "y"}})
	reversed := ConfigSecretsHash(rawCfg, map[string][]string{"k": {"y", "x"}})

	assert.NotEqual(t, forward, reversed)
}

func TestConfigSecretsHash_CountPrefixDisambiguatesValueGrouping(t *testing.T) {
	t.Parallel()

	rawCfg := map[string][]string{"k": {"$s:1", "$s:2"}}

	// Same concatenated characters ("ab"), grouped as one value vs two: the explicit
	// per-key value count keeps these from hashing identically.
	oneValue := ConfigSecretsHash(rawCfg, map[string][]string{"k": {"ab"}})
	twoValues := ConfigSecretsHash(rawCfg, map[string][]string{"k": {"a", "b"}})

	assert.NotEqual(t, oneValue, twoValues)
}

func TestConfigSecretsHash_SingleAndMultiValueLists(t *testing.T) {
	t.Parallel()

	// idp wraps a single string value as a one-element slice; component config is natively
	// multi-valued. Both shapes go through the same function and hash deterministically.
	singleValue := ConfigSecretsHash(
		map[string][]string{"clientSecret": {"$secret:key"}},
		map[string][]string{"clientSecret": {"resolved-value"}},
	)
	assert.NotEmpty(t, singleValue)
	assert.Equal(t, singleValue, ConfigSecretsHash(
		map[string][]string{"clientSecret": {"$secret:key"}},
		map[string][]string{"clientSecret": {"resolved-value"}},
	))

	multiValue := ConfigSecretsHash(
		map[string][]string{"bindCredential": {"$secret:a", "$secret:b"}},
		map[string][]string{"bindCredential": {"resolved-a", "resolved-b"}},
	)
	assert.NotEmpty(t, multiValue)
	assert.NotEqual(t, singleValue, multiValue)
}

func TestConfigSecretsHashSingle_MatchesWrappedConfigSecretsHash(t *testing.T) {
	t.Parallel()

	rawCfg := map[string]string{"clientId": "test-client", "clientSecret": "$secret:key"}
	resolvedCfg := map[string]string{"clientId": "test-client", "clientSecret": "resolved-value"}

	wrappedRaw := map[string][]string{"clientId": {"test-client"}, "clientSecret": {"$secret:key"}}
	wrappedResolved := map[string][]string{"clientId": {"test-client"}, "clientSecret": {"resolved-value"}}

	assert.Equal(t, ConfigSecretsHash(wrappedRaw, wrappedResolved), ConfigSecretsHashSingle(rawCfg, resolvedCfg))
}

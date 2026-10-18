package secretref

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/epam/edp-keycloak-operator/pkg/client/keycloakapi"
)

func TestConfigMatches(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		existing map[string]string
		desired  map[string]string
		rawSpec  map[string]string
		expected bool
	}{
		{
			name:     "clear keys equal",
			existing: map[string]string{"a": "1", "b": "2"},
			desired:  map[string]string{"a": "1"},
			rawSpec:  map[string]string{"a": "1"},
			expected: true,
		},
		{
			name:     "clear key differs",
			existing: map[string]string{"a": "1"},
			desired:  map[string]string{"a": "2"},
			rawSpec:  map[string]string{"a": "2"},
			expected: false,
		},
		{
			name:     "clear key missing from existing",
			existing: map[string]string{},
			desired:  map[string]string{"a": "1"},
			rawSpec:  map[string]string{"a": "1"},
			expected: false,
		},
		{
			name:     "masked key skipped",
			existing: map[string]string{"a": "1", "token": keycloakapi.MaskedSecretValue},
			desired:  map[string]string{"a": "1", "token": "plain"},
			rawSpec:  map[string]string{"a": "1", "token": "plain"},
			expected: true,
		},
		{
			name:     "secret ref key skipped even when fetched value is clear",
			existing: map[string]string{"a": "1", "token": "resolved-old"},
			desired:  map[string]string{"a": "1", "token": "resolved-new"},
			rawSpec:  map[string]string{"a": "1", "token": "$secret:key"},
			expected: true,
		},
		{
			name:     "secret ref key absent from existing skipped",
			existing: map[string]string{"a": "1"},
			desired:  map[string]string{"a": "1", "token": "resolved"},
			rawSpec:  map[string]string{"a": "1", "token": "$secret:key"},
			expected: true,
		},
		{
			name:     "nil rawSpec compares every unmasked key",
			existing: map[string]string{"a": "1", "token": "x"},
			desired:  map[string]string{"a": "1", "token": "y"},
			rawSpec:  nil,
			expected: false,
		},
		{
			name:     "nil existing with non-empty desired",
			existing: nil,
			desired:  map[string]string{"a": "1"},
			rawSpec:  map[string]string{"a": "1"},
			expected: false,
		},
		{
			name:     "empty desired always matches",
			existing: map[string]string{"a": "1"},
			desired:  map[string]string{},
			rawSpec:  map[string]string{},
			expected: true,
		},
		{
			name:     "desired value is the sentinel and existing is masked",
			existing: map[string]string{"apiToken": keycloakapi.MaskedSecretValue},
			desired:  map[string]string{"apiToken": keycloakapi.MaskedSecretValue},
			rawSpec:  map[string]string{"apiToken": keycloakapi.MaskedSecretValue},
			expected: true,
		},
		{
			name:     "desired value is the sentinel and existing is a different clear value",
			existing: map[string]string{"apiToken": "clear-value"},
			desired:  map[string]string{"apiToken": keycloakapi.MaskedSecretValue},
			rawSpec:  map[string]string{"apiToken": keycloakapi.MaskedSecretValue},
			expected: false,
		},
		{
			name:     "rawSpec value dollar sign alone is a ref and skipped",
			existing: map[string]string{"a": "1", "apiToken": "old"},
			desired:  map[string]string{"a": "1", "apiToken": "new"},
			rawSpec:  map[string]string{"a": "1", "apiToken": "$"},
			expected: true,
		},
		{
			name:     "rawSpec value dollar brace expression is a ref and skipped",
			existing: map[string]string{"a": "1", "apiToken": "old"},
			desired:  map[string]string{"a": "1", "apiToken": "new"},
			rawSpec:  map[string]string{"a": "1", "apiToken": "${vault.x}"},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, tt.expected, ConfigMatches(tt.existing, tt.desired, tt.rawSpec))
		})
	}
}

func TestConfigMatchesMulti(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		existing map[string][]string
		desired  map[string][]string
		rawSpec  map[string][]string
		expected bool
	}{
		{
			name:     "clear keys equal ignoring order",
			existing: map[string][]string{"a": {"2", "1"}},
			desired:  map[string][]string{"a": {"1", "2"}},
			rawSpec:  map[string][]string{"a": {"1", "2"}},
			expected: true,
		},
		{
			name:     "clear key differs",
			existing: map[string][]string{"a": {"1"}},
			desired:  map[string][]string{"a": {"2"}},
			rawSpec:  map[string][]string{"a": {"2"}},
			expected: false,
		},
		{
			name:     "masked value in key skips the key",
			existing: map[string][]string{"a": {"1"}, "token": {keycloakapi.MaskedSecretValue}},
			desired:  map[string][]string{"a": {"1"}, "token": {"plain"}},
			rawSpec:  map[string][]string{"a": {"1"}, "token": {"plain"}},
			expected: true,
		},
		{
			name:     "secret ref among values skips the key",
			existing: map[string][]string{"a": {"1"}, "token": {"old", "x"}},
			desired:  map[string][]string{"a": {"1"}, "token": {"new", "x"}},
			rawSpec:  map[string][]string{"a": {"1"}, "token": {"$secret:key", "x"}},
			expected: true,
		},
		{
			name:     "nil rawSpec compares every unmasked key",
			existing: map[string][]string{"a": {"1"}},
			desired:  map[string][]string{"a": {"2"}},
			rawSpec:  nil,
			expected: false,
		},
		{
			name:     "clear key missing from existing",
			existing: map[string][]string{"b": {"1"}},
			desired:  map[string][]string{"a": {"1"}},
			rawSpec:  map[string][]string{"a": {"1"}},
			expected: false,
		},
		{
			name:     "secret ref key absent from existing skipped",
			existing: map[string][]string{"a": {"1"}},
			desired:  map[string][]string{"a": {"1"}, "apiToken": {"resolved"}},
			rawSpec:  map[string][]string{"a": {"1"}, "apiToken": {"$secret:key"}},
			expected: true,
		},
		{
			name:     "empty desired always matches",
			existing: map[string][]string{"a": {"1"}},
			desired:  map[string][]string{},
			rawSpec:  map[string][]string{},
			expected: true,
		},
		{
			name:     "desired empty list against missing existing key",
			existing: map[string][]string{},
			desired:  map[string][]string{"a": {}},
			rawSpec:  map[string][]string{"a": {}},
			expected: false,
		},
		{
			name:     "existing list mixing sentinel and clear values with different length skips the key",
			existing: map[string][]string{"a": {"1"}, "apiToken": {"clear", keycloakapi.MaskedSecretValue}},
			desired:  map[string][]string{"a": {"1"}, "apiToken": {"plain"}},
			rawSpec:  map[string][]string{"a": {"1"}, "apiToken": {"plain"}},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, tt.expected, ConfigMatchesMulti(tt.existing, tt.desired, tt.rawSpec))
		})
	}
}

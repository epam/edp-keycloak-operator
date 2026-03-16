package maputil

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSliceToMap(t *testing.T) {
	t.Parallel()

	type item struct {
		name *string
		id   *string
	}

	strPtr := func(s string) *string { return &s }

	tests := []struct {
		name     string
		items    []item
		key      func(item) (string, bool)
		val      func(item) string
		expected map[string]string
	}{
		{
			name:     "empty slice returns empty map",
			items:    []item{},
			key:      func(i item) (string, bool) { return "", i.name != nil },
			val:      func(i item) string { return "" },
			expected: map[string]string{},
		},
		{
			name:  "single item with valid key",
			items: []item{{name: strPtr("a"), id: strPtr("1")}},
			key:   func(i item) (string, bool) { return *i.name, i.name != nil },
			val:   func(i item) string { return *i.id },
			expected: map[string]string{
				"a": "1",
			},
		},
		{
			name: "multiple items all valid",
			items: []item{
				{name: strPtr("a"), id: strPtr("1")},
				{name: strPtr("b"), id: strPtr("2")},
				{name: strPtr("c"), id: strPtr("3")},
			},
			key: func(i item) (string, bool) { return *i.name, i.name != nil && i.id != nil },
			val: func(i item) string { return *i.id },
			expected: map[string]string{
				"a": "1",
				"b": "2",
				"c": "3",
			},
		},
		{
			name: "item with nil key is skipped",
			items: []item{
				{name: strPtr("a"), id: strPtr("1")},
				{name: nil, id: strPtr("2")},
				{name: strPtr("c"), id: strPtr("3")},
			},
			key: func(i item) (string, bool) {
				if i.name == nil {
					return "", false
				}
				return *i.name, true
			},
			val: func(i item) string { return *i.id },
			expected: map[string]string{
				"a": "1",
				"c": "3",
			},
		},
		{
			// Duplicate keys: last writer wins (documented behaviour).
			name: "duplicate keys last writer wins",
			items: []item{
				{name: strPtr("dup"), id: strPtr("first")},
				{name: strPtr("dup"), id: strPtr("last")},
			},
			key: func(i item) (string, bool) { return *i.name, i.name != nil },
			val: func(i item) string { return *i.id },
			expected: map[string]string{
				"dup": "last",
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got := SliceToMap(tc.items, tc.key, tc.val)
			assert.Equal(t, tc.expected, got)
		})
	}
}

func TestSliceToMapSelf(t *testing.T) {
	t.Parallel()

	type item struct {
		name  *string
		value int
	}

	strPtr := func(s string) *string { return &s }

	t.Run("empty slice", func(t *testing.T) {
		t.Parallel()

		got := SliceToMapSelf([]item{}, func(i item) (string, bool) { return "", i.name != nil })
		assert.Equal(t, map[string]item{}, got)
	})

	t.Run("items are their own values", func(t *testing.T) {
		t.Parallel()

		a := item{name: strPtr("a"), value: 1}
		b := item{name: strPtr("b"), value: 2}
		got := SliceToMapSelf([]item{a, b}, func(i item) (string, bool) {
			return *i.name, i.name != nil
		})
		assert.Equal(t, map[string]item{"a": a, "b": b}, got)
	})

	t.Run("nil key item skipped", func(t *testing.T) {
		t.Parallel()

		a := item{name: strPtr("a"), value: 1}
		skipped := item{name: nil, value: 99}
		got := SliceToMapSelf([]item{a, skipped}, func(i item) (string, bool) {
			return "", i.name != nil
		})
		// skipped item not included; only "a" maps to ""
		assert.Len(t, got, 1)
	})
}

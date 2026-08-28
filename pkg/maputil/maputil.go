package maputil

import "slices"

// SliceToMap builds a map from a slice.
// key returns the map key and whether the item should be included (ok=false skips the item).
// val returns the map value.
func SliceToMap[T any, V any](items []T, key func(T) (string, bool), val func(T) V) map[string]V {
	m := make(map[string]V, len(items))

	for _, item := range items {
		if k, ok := key(item); ok {
			m[k] = val(item)
		}
	}

	return m
}

// SliceToMapSelf builds a map where each item is its own value.
// key returns the map key and whether the item should be included (ok=false skips the item).
func SliceToMapSelf[T any](items []T, key func(T) (string, bool)) map[string]T {
	return SliceToMap(items, key, func(item T) T { return item })
}

// ContainsSubset reports whether every key/value pair in subset is present in m.
func ContainsSubset(m, subset map[string]string) bool {
	for k, v := range subset {
		mv, ok := m[k]
		if !ok || mv != v {
			return false
		}
	}

	return true
}

// ContainsSubsetMulti reports whether every key in subset is present in m with the
// same value set; value order is ignored.
func ContainsSubsetMulti(m, subset map[string][]string) bool {
	for k, v := range subset {
		mv, ok := m[k]
		if !ok {
			return false
		}

		if len(v) != len(mv) {
			return false
		}

		if !slices.Equal(slices.Sorted(slices.Values(v)), slices.Sorted(slices.Values(mv))) {
			return false
		}
	}

	return true
}

package maputil

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

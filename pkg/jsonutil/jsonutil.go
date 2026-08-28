// Package jsonutil provides JSON-based comparison helpers.
package jsonutil

import (
	"bytes"
	"encoding/json"
)

// Unchanged reports whether v marshals to exactly before.
// Any marshal failure reports false so callers fall back to writing.
func Unchanged(before []byte, beforeErr error, v any) bool {
	if beforeErr != nil {
		return false
	}

	after, err := json.Marshal(v)

	return err == nil && bytes.Equal(before, after)
}

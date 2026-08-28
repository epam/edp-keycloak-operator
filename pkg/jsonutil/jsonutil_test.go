package jsonutil

import (
	"encoding/json"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestUnchanged(t *testing.T) {
	t.Parallel()

	type rep struct {
		Name  *string           `json:"name,omitempty"`
		Attrs map[string]string `json:"attrs,omitempty"`
	}

	name := "realm"
	v := &rep{Name: &name, Attrs: map[string]string{"a": "1"}}

	before, err := json.Marshal(v)

	assert.NoError(t, err)
	assert.True(t, Unchanged(before, nil, v))

	v.Attrs["a"] = "2"
	assert.False(t, Unchanged(before, nil, v))

	v.Attrs["a"] = "1"
	assert.True(t, Unchanged(before, nil, v))

	assert.False(t, Unchanged(before, errors.New("marshal failed"), v))
	assert.False(t, Unchanged(before, nil, func() {}))
}

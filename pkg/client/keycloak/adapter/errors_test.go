package adapter

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSkipAlreadyExistsErr(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		err     error
		wantErr assert.ErrorAssertionFunc
	}{
		{
			name:    "ReturnsNil_WhenErrorIsNil",
			err:     nil,
			wantErr: assert.NoError,
		},
		{
			name:    "ReturnsNil_WhenErrorIsConflict",
			err:     errors.New("409 Conflict"),
			wantErr: assert.NoError,
		},
		{
			name:    "ReturnsError_WhenErrorIsNotConflict",
			err:     errors.New("500 Internal Server Error"),
			wantErr: assert.Error,
		},
	}

	for _, tt := range tests {
		tt := tt

		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			tt.wantErr(t, SkipAlreadyExistsErr(tt.err))
		})
	}
}

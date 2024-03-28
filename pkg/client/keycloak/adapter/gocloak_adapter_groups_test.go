package adapter

import (
	"errors"
	"net/http"
	"testing"

	"github.com/Nerzal/gocloak/v12"
	"github.com/stretchr/testify/assert"
)

func TestIsErrNotFound(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		err  error
		want bool
	}{
		{
			name: "test error is NotFoundError",
			err:  NotFoundError(""),
			want: true,
		},
		{
			name: "test error is api error not found",
			err:  gocloak.APIError{Code: http.StatusNotFound},
			want: true,
		},
		{
			name: "test error is not api error not found",
			err:  gocloak.APIError{Code: http.StatusBadRequest},
			want: false,
		},
		{
			name: "test error is not NotFoundError",
			err:  errors.New("error"),
			want: false,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, tt.want, IsErrNotFound(tt.err))
		})
	}
}

package fakehttp

import (
	"encoding/json"
	"io"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMockServerBuilder_AddStringResponder(t *testing.T) {
	t.Parallel()

	type args struct {
		endpoint string
		response string
	}

	tests := []struct {
		name string
		args args
	}{
		{
			name: "should succeed",
			args: args{
				endpoint: "/test/address",
				response: "testdata",
			},
		},
	}

	for _, tt := range tests {
		tt := tt

		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			fb := NewServerBuilder()

			fakeServer := fb.AddStringResponder(tt.args.endpoint, tt.args.response).
				BuildAndStart()
			defer fakeServer.Close()

			resp, err := http.Get(fakeServer.GetURL() + tt.args.endpoint)
			require.NoError(t, err)

			require.Equal(t, http.StatusOK, resp.StatusCode)

			body, err := io.ReadAll(resp.Body)
			require.NoError(t, err)
			err = resp.Body.Close()
			require.NoError(t, err)

			require.Equal(t, tt.args.response, string(body))
		})
	}
}

func TestMockServerBuilder_AddStringResponderWithCode(t *testing.T) {
	t.Parallel()

	type args struct {
		code     int
		endpoint string
		response string
	}

	tests := []struct {
		name string
		args args
	}{
		{
			name: "should return 404",
			args: args{
				code:     http.StatusNotFound,
				endpoint: "/test/address",
				response: "testdata",
			},
		},
	}

	for _, tt := range tests {
		tt := tt

		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			fb := NewServerBuilder()

			fakeServer := fb.AddStringResponderWithCode(tt.args.code, tt.args.endpoint, tt.args.response).
				BuildAndStart()
			defer fakeServer.Close()

			resp, err := http.Get(fakeServer.GetURL() + tt.args.endpoint)
			require.NoError(t, err)

			require.Equal(t, tt.args.code, resp.StatusCode)

			body, err := io.ReadAll(resp.Body)
			require.NoError(t, err)
			err = resp.Body.Close()
			require.NoError(t, err)

			require.Equal(t, tt.args.response, string(body))
		})
	}
}

func TestMockServerBuilder_AddJsonResponderWithCode(t *testing.T) {
	t.Parallel()

	type args struct {
		code     int
		endpoint string
		response any
	}

	tests := []struct {
		name string
		args args
	}{
		{
			name: "should return valid JSON",
			args: args{
				code:     http.StatusOK,
				endpoint: "/test/address",
				response: map[string]string{"test": "data"},
			},
		},
	}

	for _, tt := range tests {
		tt := tt

		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			fb := NewServerBuilder()

			fakeServer := fb.AddJsonResponderWithCode(tt.args.code, tt.args.endpoint, tt.args.response).
				BuildAndStart()
			defer fakeServer.Close()

			resp, err := http.Get(fakeServer.GetURL() + tt.args.endpoint)
			require.NoError(t, err)

			require.Equal(t, tt.args.code, resp.StatusCode)

			body, err := io.ReadAll(resp.Body)
			require.NoError(t, err)
			err = resp.Body.Close()
			require.NoError(t, err)

			jsonResp, err := json.Marshal(tt.args.response)
			require.NoError(t, err)

			require.JSONEq(t, string(jsonResp), string(body))

		})
	}
}

func TestDefaultFakeServer_Close(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name             string
		createFakeServer func() *DefaultServer
		wantPanic        func(t require.TestingT, f assert.PanicTestFunc, _ ...interface{})
	}{
		{
			name: "should close",
			createFakeServer: func() *DefaultServer {
				fakeServer := NewDefaultServer()
				fakeServer.Start()

				return fakeServer
			},
			wantPanic: require.NotPanics,
		},
		{
			name:             "should fail",
			createFakeServer: NewDefaultServer,
			wantPanic:        require.Panics,
		},
	}

	for _, tt := range tests {
		tt := tt

		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			f := tt.createFakeServer()

			tt.wantPanic(t, f.Close)
		})
	}
}

func TestDefaultFakeServer_GetURL(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name             string
		createFakeServer func() *DefaultServer
		wantPanic        func(t require.TestingT, f assert.PanicTestFunc, _ ...interface{})
	}{
		{
			name: "should return URL",
			createFakeServer: func() *DefaultServer {
				fakeServer := NewDefaultServer()
				fakeServer.Start()

				return fakeServer
			},
			wantPanic: require.NotPanics,
		},
		{
			name:             "should fail",
			createFakeServer: NewDefaultServer,
			wantPanic:        require.Panics,
		},
	}

	for _, tt := range tests {
		tt := tt

		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			f := tt.createFakeServer()

			tt.wantPanic(t, func() {
				url := f.GetURL()

				require.NotEmpty(t, url)
			})
		})
	}
}

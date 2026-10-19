package fakehttp

import (
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"io"
	"net/http"
	"strings"
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
		wantPanic        func(t require.TestingT, f assert.PanicTestFunc, _ ...any)
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
		wantPanic        func(t require.TestingT, f assert.PanicTestFunc, _ ...any)
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

func TestServerBuilder_AddKeycloakAuthResponders(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		realm          string
		useCustomRealm bool
	}{
		{
			name:           "should add auth responders for master realm",
			realm:          "master",
			useCustomRealm: false,
		},
		{
			name:           "should add auth responders for custom realm",
			realm:          "custom-realm",
			useCustomRealm: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			builder := NewServerBuilder()
			if tt.useCustomRealm {
				builder.AddKeycloakAuthRespondersForRealm(tt.realm)
			} else {
				builder.AddKeycloakAuthResponders()
			}

			fakeServer := builder.BuildAndStart()
			defer fakeServer.Close()

			// Test token endpoint
			tokenEndpoint := fakeServer.GetURL() + "/realms/" + tt.realm + "/protocol/openid-connect/token"
			tokenResp, err := http.Get(tokenEndpoint)
			require.NoError(t, err)
			require.Equal(t, http.StatusOK, tokenResp.StatusCode)

			var tokenBody map[string]any
			err = json.NewDecoder(tokenResp.Body).Decode(&tokenBody)
			require.NoError(t, err)
			assert.Equal(t, "test-access-token", tokenBody["access_token"])
			assert.Equal(t, "test-refresh-token", tokenBody["refresh_token"])
			assert.Equal(t, "Bearer", tokenBody["token_type"])

			// Test admin endpoint
			adminEndpoint := fakeServer.GetURL() + "/admin/realms/" + tt.realm
			adminResp, err := http.Get(adminEndpoint)
			require.NoError(t, err)
			require.Equal(t, http.StatusOK, adminResp.StatusCode)

			var adminBody map[string]any
			err = json.NewDecoder(adminResp.Body).Decode(&adminBody)
			require.NoError(t, err)
			assert.Empty(t, adminBody)
		})
	}
}

func TestServerBuilder_BuildAndStartTLS(t *testing.T) {
	t.Parallel()

	server, certPEM, err := NewServerBuilder().
		AddStringResponder("/test", "ok").
		BuildAndStartTLS("keycloak.example.com")
	require.NoError(t, err)
	t.Cleanup(server.Close)

	require.True(t, strings.HasPrefix(server.GetURL(), "https://127.0.0.1:"))

	roots := x509.NewCertPool()
	require.True(t, roots.AppendCertsFromPEM([]byte(certPEM)))

	get := func(serverName string) (*http.Response, error) {
		client := &http.Client{Transport: &http.Transport{
			TLSClientConfig: &tls.Config{RootCAs: roots, ServerName: serverName},
		}}

		return client.Get(server.GetURL() + "/test")
	}

	_, err = get("")
	require.ErrorContains(t, err, "x509")

	resp, err := get("keycloak.example.com")
	require.NoError(t, err)

	t.Cleanup(func() { _ = resp.Body.Close() })

	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

func TestServerBuilder_BuildAndStartTLS_IPAddress(t *testing.T) {
	t.Parallel()

	server, certPEM, err := NewServerBuilder().
		AddStringResponder("/test", "ok").
		BuildAndStartTLS("127.0.0.1")
	require.NoError(t, err)
	t.Cleanup(server.Close)

	roots := x509.NewCertPool()
	require.True(t, roots.AppendCertsFromPEM([]byte(certPEM)))

	client := &http.Client{Transport: &http.Transport{TLSClientConfig: &tls.Config{RootCAs: roots}}}

	resp, err := client.Get(server.GetURL() + "/test")
	require.NoError(t, err)

	t.Cleanup(func() { _ = resp.Body.Close() })

	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

func TestCertificate_IssueClient(t *testing.T) {
	t.Parallel()

	ca, err := NewCA()
	require.NoError(t, err)

	client, err := ca.IssueClient("operator")
	require.NoError(t, err)

	_, err = tls.X509KeyPair([]byte(client.CertPEM), []byte(client.KeyPEM))
	require.NoError(t, err)

	roots := x509.NewCertPool()
	roots.AddCert(ca.Cert)

	_, err = client.Cert.Verify(x509.VerifyOptions{
		Roots:     roots,
		KeyUsages: []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth},
	})
	require.NoError(t, err)
}

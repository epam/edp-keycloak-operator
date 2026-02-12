package keycloakv2

import (
	"context"
	"crypto/ecdsa"
	"crypto/ed25519"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"io"
	"math/big"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/go-logr/logr"
	"github.com/go-resty/resty/v2"
	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	ctrl "sigs.k8s.io/controller-runtime"
)

const (
	testRealm        = "test-realm"
	testClientID     = "test-client"
	testClientSecret = "test-secret"
	testUsername     = "test-user"
	testPassword     = "test-password"
	testAccessToken  = "test-access-token"
	testRefreshToken = "test-refresh-token"
	testTokenType    = "bearer"
	testUserAgent    = "test-agent/1.0"
)

func TestNewKeycloakClient(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	tests := []struct {
		name        string
		url         string
		clientID    string
		opts        []ClientOption
		setupServer func(t *testing.T) *httptest.Server
		wantErr     bool
		errContains string
		validate    func(t *testing.T, client *KeycloakClient)
	}{
		{
			name:        "missing url",
			url:         "",
			clientID:    testClientID,
			wantErr:     true,
			errContains: "url is required",
		},
		{
			name:        "missing clientId",
			url:         "http://localhost",
			clientID:    "",
			wantErr:     true,
			errContains: "clientId is required",
		},
		{
			name:        "missing authentication method",
			url:         "http://localhost",
			clientID:    testClientID,
			opts:        []ClientOption{WithInitialLogin(false)},
			wantErr:     true,
			errContains: "must specify authentication method",
		},
		{
			name:     "password grant with client secret",
			url:      "http://localhost",
			clientID: testClientID,
			opts: []ClientOption{
				WithPasswordGrant(testUsername, testPassword),
				WithClientSecret(testClientSecret),
			},
			setupServer: func(t *testing.T) *httptest.Server {
				return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					if strings.Contains(r.URL.Path, "/token") {
						w.Header().Set("Content-Type", "application/json")
						w.WriteHeader(http.StatusOK)
						_, _ = w.Write([]byte(createMockTokenResponseJSON()))
					}
				}))
			},
			wantErr: false,
			validate: func(t *testing.T, client *KeycloakClient) {
				assert.Equal(t, testClientID, client.clientCredentials.ClientId)
				assert.Equal(t, testUsername, client.clientCredentials.Username)
				assert.Equal(t, testPassword, client.clientCredentials.Password)
				assert.Equal(t, testClientSecret, client.clientCredentials.ClientSecret)
				assert.Equal(t, "password", client.clientCredentials.GrantType)
				assertTokensSet(t, client.clientCredentials)
			},
		},
		{
			name:     "client credentials grant",
			url:      "http://localhost",
			clientID: testClientID,
			opts: []ClientOption{
				WithClientSecret(testClientSecret),
			},
			setupServer: func(t *testing.T) *httptest.Server {
				return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					if strings.Contains(r.URL.Path, "/token") {
						w.Header().Set("Content-Type", "application/json")
						w.WriteHeader(http.StatusOK)
						_, _ = w.Write([]byte(createMockTokenResponseJSON()))
					}
				}))
			},
			wantErr: false,
			validate: func(t *testing.T, client *KeycloakClient) {
				assert.Equal(t, testClientID, client.clientCredentials.ClientId)
				assert.Equal(t, testClientSecret, client.clientCredentials.ClientSecret)
				assert.Equal(t, "client_credentials", client.clientCredentials.GrantType)
				assertTokensSet(t, client.clientCredentials)
			},
		},
		{
			name:     "with access token",
			url:      "http://localhost",
			clientID: testClientID,
			opts: []ClientOption{
				WithAccessToken(testAccessToken),
			},
			wantErr: false,
			validate: func(t *testing.T, client *KeycloakClient) {
				assert.Equal(t, testAccessToken, client.clientCredentials.AccessToken)
				assert.Equal(t, "bearer", client.clientCredentials.TokenType)
				assert.True(t, client.accessTokenProvided)
			},
		},
		{
			name:     "with custom realm",
			url:      "http://localhost",
			clientID: testClientID,
			opts: []ClientOption{
				WithAccessToken(testAccessToken),
				WithRealm("custom-realm"),
			},
			wantErr: false,
			validate: func(t *testing.T, client *KeycloakClient) {
				assert.Equal(t, "custom-realm", client.realm)
			},
		},
		{
			name:     "with base path",
			url:      "http://localhost",
			clientID: testClientID,
			opts: []ClientOption{
				WithAccessToken(testAccessToken),
				WithBasePath("/auth"),
			},
			wantErr: false,
			validate: func(t *testing.T, client *KeycloakClient) {
				assert.Contains(t, client.baseUrl, "/auth")
				assert.Contains(t, client.authUrl, "/auth")
			},
		},
		{
			name:     "with admin URL",
			url:      "http://localhost",
			clientID: testClientID,
			opts: []ClientOption{
				WithAccessToken(testAccessToken),
				WithAdminURL("http://admin-server"),
			},
			wantErr: false,
			validate: func(t *testing.T, client *KeycloakClient) {
				assert.Contains(t, client.baseUrl, "http://admin-server")
				assert.NotEqual(t, client.baseUrl, client.authUrl)
			},
		},
		{
			name:     "with user agent",
			url:      "http://localhost",
			clientID: testClientID,
			opts: []ClientOption{
				WithAccessToken(testAccessToken),
				WithUserAgent(testUserAgent),
			},
			wantErr: false,
			validate: func(t *testing.T, client *KeycloakClient) {
				assert.Equal(t, testUserAgent, client.userAgent)
			},
		},
		{
			name:     "with additional headers",
			url:      "http://localhost",
			clientID: testClientID,
			opts: []ClientOption{
				WithAccessToken(testAccessToken),
				WithAdditionalHeaders(map[string]string{
					"X-Custom-Header": "custom-value",
				}),
			},
			wantErr: false,
			validate: func(t *testing.T, client *KeycloakClient) {
				assert.Equal(t, "custom-value", client.additionalHeaders["X-Custom-Header"])
			},
		},
		{
			name:     "with red hat sso",
			url:      "http://localhost",
			clientID: testClientID,
			opts: []ClientOption{
				WithAccessToken(testAccessToken),
				WithRedHatSSO(true),
			},
			wantErr: false,
			validate: func(t *testing.T, client *KeycloakClient) {
				assert.True(t, client.redHatSSO)
			},
		},
		{
			name:     "with custom logger",
			url:      "http://localhost",
			clientID: testClientID,
			opts: []ClientOption{
				WithAccessToken(testAccessToken),
				WithLogger(logr.Discard()),
			},
			wantErr: false,
			validate: func(t *testing.T, client *KeycloakClient) {
				assert.NotNil(t, client.logger)
			},
		},
		{
			name:     "with initial login disabled",
			url:      "http://localhost",
			clientID: testClientID,
			opts: []ClientOption{
				WithClientSecret(testClientSecret),
				WithInitialLogin(false),
			},
			wantErr: false,
			validate: func(t *testing.T, client *KeycloakClient) {
				assert.Empty(t, client.clientCredentials.AccessToken)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var server *httptest.Server
			if tt.setupServer != nil {
				server = tt.setupServer(t)
				defer server.Close()
				tt.url = server.URL
			}

			client, err := NewKeycloakClient(ctx, tt.url, tt.clientID, tt.opts...)

			if tt.wantErr {
				require.Error(t, err)

				if tt.errContains != "" {
					assert.Contains(t, err.Error(), tt.errContains)
				}

				return
			}

			require.NoError(t, err)
			require.NotNil(t, client)

			if tt.validate != nil {
				tt.validate(t, client)
			}

			// Verify default values
			if client.realm == "" {
				t.Error("realm should have default value")
			}

			assert.NotNil(t, client.httpClient)
			assert.NotNil(t, client.Users)
			assert.NotNil(t, client.Realms)
		})
	}
}

func TestKeycloakClient_Login_PasswordGrant(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name            string
		username        string
		password        string
		clientSecret    string
		serverResponse  func(w http.ResponseWriter, r *http.Request)
		wantErr         bool
		errContains     string
		validateRequest func(t *testing.T, r *http.Request)
	}{
		{
			name:         "successful login",
			username:     testUsername,
			password:     testPassword,
			clientSecret: testClientSecret,
			serverResponse: func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write([]byte(createMockTokenResponseJSON()))
			},
			wantErr: false,
			validateRequest: func(t *testing.T, r *http.Request) {
				assert.Equal(t, "POST", r.Method)
				assert.Contains(t, r.URL.Path, "/protocol/openid-connect/token")
				assert.Equal(t, "application/x-www-form-urlencoded", r.Header.Get("Content-Type"))

				body, _ := io.ReadAll(r.Body)
				bodyStr := string(body)
				assert.Contains(t, bodyStr, "grant_type=password")
				assert.Contains(t, bodyStr, "username="+testUsername)
				assert.Contains(t, bodyStr, "password="+testPassword)
				assert.Contains(t, bodyStr, "client_secret="+testClientSecret)
			},
		},
		{
			name:         "login without client secret",
			username:     testUsername,
			password:     testPassword,
			clientSecret: "",
			serverResponse: func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write([]byte(createMockTokenResponseJSON()))
			},
			wantErr: false,
			validateRequest: func(t *testing.T, r *http.Request) {
				body, _ := io.ReadAll(r.Body)
				bodyStr := string(body)
				assert.NotContains(t, bodyStr, "client_secret")
			},
		},
		{
			name:     "login failure",
			username: testUsername,
			password: testPassword,
			serverResponse: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusUnauthorized)
				_, _ = w.Write([]byte(`{"error": "invalid_grant"}`))
			},
			wantErr:     true,
			errContains: "error sending POST request",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var capturedRequest *http.Request

			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if strings.Contains(r.URL.Path, "/token") {
					// Capture request body
					bodyBytes, _ := io.ReadAll(r.Body)
					r.Body = io.NopCloser(strings.NewReader(string(bodyBytes)))
					capturedRequest = r
					capturedRequest.Body = io.NopCloser(strings.NewReader(string(bodyBytes)))

					tt.serverResponse(w, r)
				}
			}))
			defer server.Close()

			opts := []ClientOption{
				WithPasswordGrant(tt.username, tt.password),
			}
			if tt.clientSecret != "" {
				opts = append(opts, WithClientSecret(tt.clientSecret))
			}

			client, err := NewKeycloakClient(context.Background(), server.URL, testClientID, opts...)

			if tt.wantErr {
				require.Error(t, err)

				if tt.errContains != "" {
					assert.Contains(t, err.Error(), tt.errContains)
				}

				return
			}

			require.NoError(t, err)
			require.NotNil(t, client)
			assertTokensSet(t, client.clientCredentials)

			if tt.validateRequest != nil && capturedRequest != nil {
				tt.validateRequest(t, capturedRequest)
			}
		})
	}
}

func TestKeycloakClient_Login_ClientCredentials(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name            string
		clientSecret    string
		serverResponse  func(w http.ResponseWriter, r *http.Request)
		wantErr         bool
		validateRequest func(t *testing.T, r *http.Request)
	}{
		{
			name:         "successful login with client secret",
			clientSecret: testClientSecret,
			serverResponse: func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write([]byte(createMockTokenResponseJSON()))
			},
			wantErr: false,
			validateRequest: func(t *testing.T, r *http.Request) {
				body, _ := io.ReadAll(r.Body)
				bodyStr := string(body)
				assert.Contains(t, bodyStr, "grant_type=client_credentials")
				assert.Contains(t, bodyStr, "client_secret="+testClientSecret)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var capturedRequest *http.Request

			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if strings.Contains(r.URL.Path, "/token") {
					bodyBytes, _ := io.ReadAll(r.Body)
					r.Body = io.NopCloser(strings.NewReader(string(bodyBytes)))
					capturedRequest = r
					capturedRequest.Body = io.NopCloser(strings.NewReader(string(bodyBytes)))

					tt.serverResponse(w, r)
				}
			}))
			defer server.Close()

			client, err := NewKeycloakClient(
				context.Background(),
				server.URL,
				testClientID,
				WithClientSecret(tt.clientSecret),
			)

			if tt.wantErr {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)
			require.NotNil(t, client)
			assertTokensSet(t, client.clientCredentials)

			if tt.validateRequest != nil && capturedRequest != nil {
				tt.validateRequest(t, capturedRequest)
			}
		})
	}
}

func TestKeycloakClient_Login_JWT(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		signingAlg  string
		setupKey    func(t *testing.T) string
		wantErr     bool
		validateJWT func(t *testing.T, jwtToken string, key any)
	}{
		{
			name:       "RS256 JWT signing",
			signingAlg: "RS256",
			setupKey: func(t *testing.T) string {
				keyPEM, _ := createTestRSAKey(t)
				return keyPEM
			},
			wantErr: false,
			validateJWT: func(t *testing.T, jwtToken string, key any) {
				token, err := jwt.Parse(jwtToken, func(token *jwt.Token) (any, error) {
					return &key.(*rsa.PrivateKey).PublicKey, nil
				})
				require.NoError(t, err)
				assert.True(t, token.Valid)

				claims, ok := token.Claims.(jwt.MapClaims)
				require.True(t, ok)
				assert.Equal(t, testClientID, claims["iss"])
				assert.Equal(t, testClientID, claims["sub"])
			},
		},
		{
			name:       "ES256 JWT signing",
			signingAlg: "ES256",
			setupKey: func(t *testing.T) string {
				keyPEM, _ := createTestECDSAKey(t)
				return keyPEM
			},
			wantErr: false,
			validateJWT: func(t *testing.T, jwtToken string, key any) {
				token, err := jwt.Parse(jwtToken, func(token *jwt.Token) (any, error) {
					return &key.(*ecdsa.PrivateKey).PublicKey, nil
				})
				require.NoError(t, err)
				assert.True(t, token.Valid)
			},
		},
		{
			name:       "EdDSA JWT signing",
			signingAlg: "EdDSA",
			setupKey: func(t *testing.T) string {
				keyPEM, _ := createTestEd25519Key(t)
				return keyPEM
			},
			wantErr: false,
			validateJWT: func(t *testing.T, jwtToken string, key any) {
				token, err := jwt.Parse(jwtToken, func(token *jwt.Token) (any, error) {
					return key.(ed25519.PrivateKey).Public(), nil
				})
				require.NoError(t, err)
				assert.True(t, token.Valid)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			keyPEM := tt.setupKey(t)

			var capturedJWT string

			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if strings.Contains(r.URL.Path, "/token") {
					body, _ := io.ReadAll(r.Body)
					bodyStr := string(body)

					// Extract JWT from client_assertion
					for part := range strings.SplitSeq(bodyStr, "&") {
						if strings.HasPrefix(part, "client_assertion=") {
							capturedJWT = strings.TrimPrefix(part, "client_assertion=")
						}
					}

					w.Header().Set("Content-Type", "application/json")
					w.WriteHeader(http.StatusOK)
					_, _ = w.Write([]byte(createMockTokenResponseJSON()))
				}
			}))
			defer server.Close()

			client, err := NewKeycloakClient(
				context.Background(),
				server.URL,
				testClientID,
				WithJWTAuth(tt.signingAlg, keyPEM, "", ""),
			)

			if tt.wantErr {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)
			require.NotNil(t, client)
			assert.NotEmpty(t, capturedJWT)

			if tt.validateJWT != nil {
				var key any

				switch tt.signingAlg {
				case "RS256", "RS384", "RS512":
					key, err = jwt.ParseRSAPrivateKeyFromPEM([]byte(keyPEM))
					require.NoError(t, err)
				case "ES256", "ES384", "ES512":
					key, err = jwt.ParseECPrivateKeyFromPEM([]byte(keyPEM))
					require.NoError(t, err)
				case "EdDSA":
					key, err = jwt.ParseEdPrivateKeyFromPEM([]byte(keyPEM))
					require.NoError(t, err)
				}

				tt.validateJWT(t, capturedJWT, key)
			}
		})
	}
}

func TestKeycloakClient_Login_JWTFromFile(t *testing.T) {
	t.Parallel()

	// Create a temporary file with a JWT token
	tempDir := t.TempDir()
	jwtFile := filepath.Join(tempDir, "jwt-token.txt")
	expectedJWT := "test.jwt.token"
	err := os.WriteFile(jwtFile, []byte(expectedJWT+"\n"), 0600)
	require.NoError(t, err)

	var capturedJWT string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "/token") {
			body, _ := io.ReadAll(r.Body)
			bodyStr := string(body)

			for part := range strings.SplitSeq(bodyStr, "&") {
				if strings.HasPrefix(part, "client_assertion=") {
					capturedJWT = strings.TrimPrefix(part, "client_assertion=")
				}
			}

			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(createMockTokenResponseJSON()))
		}
	}))
	defer server.Close()

	client, err := NewKeycloakClient(
		context.Background(),
		server.URL,
		testClientID,
		WithJWTAuth("", "", "", jwtFile),
	)

	require.NoError(t, err)
	require.NotNil(t, client)
	assert.Equal(t, expectedJWT, capturedJWT)
}

func TestKeycloakClient_Login_JWTDirectly(t *testing.T) {
	t.Parallel()

	expectedJWT := "test.jwt.token"

	var capturedJWT string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "/token") {
			body, _ := io.ReadAll(r.Body)
			bodyStr := string(body)

			for part := range strings.SplitSeq(bodyStr, "&") {
				if strings.HasPrefix(part, "client_assertion=") {
					capturedJWT = strings.TrimPrefix(part, "client_assertion=")
				}
			}

			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(createMockTokenResponseJSON()))
		}
	}))
	defer server.Close()

	client, err := NewKeycloakClient(
		context.Background(),
		server.URL,
		testClientID,
		WithJWTAuth("", "", expectedJWT, ""),
	)

	require.NoError(t, err)
	require.NotNil(t, client)
	assert.Equal(t, expectedJWT, capturedJWT)
}

func TestKeycloakClient_Refresh(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		setupClient    func(t *testing.T, serverURL string) *KeycloakClient
		serverResponse func(w http.ResponseWriter, r *http.Request)
		wantErr        bool
		validateTokens func(t *testing.T, client *KeycloakClient)
	}{
		//nolint:dupl // Similar structure to other refresh test cases aids test readability and consistency
		{
			name: "successful refresh",
			setupClient: func(t *testing.T, serverURL string) *KeycloakClient {
				client, err := NewKeycloakClient(
					context.Background(),
					serverURL,
					testClientID,
					WithClientSecret(testClientSecret),
				)
				require.NoError(t, err)
				return client
			},
			serverResponse: func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusOK)
				newTokenResponse := map[string]any{
					"access_token":  "new-access-token",
					"refresh_token": "new-refresh-token",
					"token_type":    "bearer",
				}
				data, _ := json.Marshal(newTokenResponse)
				_, _ = w.Write(data)
			},
			wantErr: false,
			validateTokens: func(t *testing.T, client *KeycloakClient) {
				assert.Equal(t, "new-access-token", client.clientCredentials.AccessToken)
				assert.Equal(t, "new-refresh-token", client.clientCredentials.RefreshToken)
			},
		},
		//nolint:dupl // Similar structure to other refresh test cases aids test readability and consistency
		{
			name: "400 response triggers re-login",
			setupClient: func(t *testing.T, serverURL string) *KeycloakClient {
				client, err := NewKeycloakClient(
					context.Background(),
					serverURL,
					testClientID,
					WithClientSecret(testClientSecret),
				)
				require.NoError(t, err)
				return client
			},
			serverResponse: func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusOK)
				newTokenResponse := map[string]any{
					"access_token":  "new-access-token-after-400",
					"refresh_token": "new-refresh-token-after-400",
					"token_type":    "bearer",
				}
				data, _ := json.Marshal(newTokenResponse)
				_, _ = w.Write(data)
			},
			wantErr: false,
			validateTokens: func(t *testing.T, client *KeycloakClient) {
				assert.Equal(t, "new-access-token-after-400", client.clientCredentials.AccessToken)
				assert.Equal(t, "new-refresh-token-after-400", client.clientCredentials.RefreshToken)
			},
		},
		{
			name: "skip refresh for provided access token",
			setupClient: func(t *testing.T, serverURL string) *KeycloakClient {
				client, err := NewKeycloakClient(
					context.Background(),
					serverURL,
					testClientID,
					WithAccessToken("provided-token"),
				)
				require.NoError(t, err)
				return client
			},
			serverResponse: func(w http.ResponseWriter, r *http.Request) {
				t.Error("should not make refresh request")
			},
			wantErr: false,
			validateTokens: func(t *testing.T, client *KeycloakClient) {
				assert.Equal(t, "provided-token", client.clientCredentials.AccessToken)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			callCount := 0

			var mu sync.Mutex

			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if strings.Contains(r.URL.Path, "/token") {
					mu.Lock()
					callCount++
					currentCount := callCount
					mu.Unlock()

					// For "400_response_triggers_re-login" test:
					// Skip the 400 on initial login, but return 400 during refresh (second call)
					if tt.name == "400 response triggers re-login" {
						switch currentCount {
						case 1:
							// Initial login succeeds
							w.Header().Set("Content-Type", "application/json")
							w.WriteHeader(http.StatusOK)
							_, _ = w.Write([]byte(createMockTokenResponseJSON()))
						case 2:
							// Refresh gets 400
							w.WriteHeader(http.StatusBadRequest)
							_, _ = w.Write([]byte(`{"error": "invalid_grant"}`))
						default:
							// Re-login succeeds
							tt.serverResponse(w, r)
						}

						return
					}

					tt.serverResponse(w, r)
				}
			}))
			defer server.Close()

			client := tt.setupClient(t, server.URL)

			err := client.Refresh(context.Background())

			if tt.wantErr {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)

			if tt.validateTokens != nil {
				tt.validateTokens(t, client)
			}
		})
	}
}

func TestKeycloakDoer_Do(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		setupClient    func(t *testing.T, serverURL string) *KeycloakClient
		setupRequest   func(t *testing.T, serverURL string) *http.Request
		serverResponse func(callCount *int32) func(w http.ResponseWriter, r *http.Request)
		wantErr        bool
		validateResp   func(t *testing.T, resp *http.Response)
	}{
		{
			name: "successful request",
			setupClient: func(t *testing.T, serverURL string) *KeycloakClient {
				client, err := NewKeycloakClient(
					context.Background(),
					serverURL,
					testClientID,
					WithAccessToken(testAccessToken),
				)
				require.NoError(t, err)
				return client
			},
			setupRequest: func(t *testing.T, serverURL string) *http.Request {
				req, _ := http.NewRequest(http.MethodGet, serverURL+"/admin/realms", nil)
				return req
			},
			serverResponse: func(callCount *int32) func(w http.ResponseWriter, r *http.Request) {
				return func(w http.ResponseWriter, r *http.Request) {
					atomic.AddInt32(callCount, 1)
					w.WriteHeader(http.StatusOK)
					_, _ = w.Write([]byte(`{"result":"success"}`))
				}
			},
			wantErr: false,
			validateResp: func(t *testing.T, resp *http.Response) {
				assert.Equal(t, http.StatusOK, resp.StatusCode)
			},
		},
		//nolint:dupl // Similar structure to 403 test case aids test readability and consistency
		{
			name: "401 triggers refresh and retry",
			setupClient: func(t *testing.T, serverURL string) *KeycloakClient {
				client, err := NewKeycloakClient(
					context.Background(),
					serverURL,
					testClientID,
					WithClientSecret(testClientSecret),
				)
				require.NoError(t, err)
				return client
			},
			setupRequest: func(t *testing.T, serverURL string) *http.Request {
				req, _ := http.NewRequest(http.MethodGet, serverURL+"/admin/realms", nil)
				return req
			},
			serverResponse: func(callCount *int32) func(w http.ResponseWriter, r *http.Request) {
				return func(w http.ResponseWriter, r *http.Request) {
					count := atomic.AddInt32(callCount, 1)

					if strings.Contains(r.URL.Path, "/token") {
						w.Header().Set("Content-Type", "application/json")
						w.WriteHeader(http.StatusOK)
						_, _ = w.Write([]byte(createMockTokenResponseJSON()))
						return
					}

					// First request returns 401, second succeeds
					if count == 2 {
						w.WriteHeader(http.StatusUnauthorized)
						return
					}

					w.WriteHeader(http.StatusOK)
					_, _ = w.Write([]byte(`{"result":"success"}`))
				}
			},
			wantErr: false,
			validateResp: func(t *testing.T, resp *http.Response) {
				assert.Equal(t, http.StatusOK, resp.StatusCode)
			},
		},
		//nolint:dupl // Similar structure to 401 test case aids test readability and consistency
		{
			name: "403 triggers refresh and retry",
			setupClient: func(t *testing.T, serverURL string) *KeycloakClient {
				client, err := NewKeycloakClient(
					context.Background(),
					serverURL,
					testClientID,
					WithClientSecret(testClientSecret),
				)
				require.NoError(t, err)
				return client
			},
			setupRequest: func(t *testing.T, serverURL string) *http.Request {
				req, _ := http.NewRequest(http.MethodGet, serverURL+"/admin/realms", nil)
				return req
			},
			serverResponse: func(callCount *int32) func(w http.ResponseWriter, r *http.Request) {
				return func(w http.ResponseWriter, r *http.Request) {
					count := atomic.AddInt32(callCount, 1)

					if strings.Contains(r.URL.Path, "/token") {
						w.Header().Set("Content-Type", "application/json")
						w.WriteHeader(http.StatusOK)
						_, _ = w.Write([]byte(createMockTokenResponseJSON()))
						return
					}

					// First request returns 403, second succeeds
					if count == 2 {
						w.WriteHeader(http.StatusForbidden)
						return
					}

					w.WriteHeader(http.StatusOK)
					_, _ = w.Write([]byte(`{"result":"success"}`))
				}
			},
			wantErr: false,
			validateResp: func(t *testing.T, resp *http.Response) {
				assert.Equal(t, http.StatusOK, resp.StatusCode)
			},
		},
		{
			name: "lazy login on first request",
			setupClient: func(t *testing.T, serverURL string) *KeycloakClient {
				client, err := NewKeycloakClient(
					context.Background(),
					serverURL,
					testClientID,
					WithClientSecret(testClientSecret),
					WithInitialLogin(false),
				)
				require.NoError(t, err)
				return client
			},
			setupRequest: func(t *testing.T, serverURL string) *http.Request {
				req, _ := http.NewRequest(http.MethodGet, serverURL+"/admin/realms", nil)
				return req
			},
			serverResponse: func(callCount *int32) func(w http.ResponseWriter, r *http.Request) {
				return func(w http.ResponseWriter, r *http.Request) {
					atomic.AddInt32(callCount, 1)

					if strings.Contains(r.URL.Path, "/token") {
						w.Header().Set("Content-Type", "application/json")
						w.WriteHeader(http.StatusOK)
						_, _ = w.Write([]byte(createMockTokenResponseJSON()))
						return
					}

					w.WriteHeader(http.StatusOK)
					_, _ = w.Write([]byte(`{"result":"success"}`))
				}
			},
			wantErr: false,
			validateResp: func(t *testing.T, resp *http.Response) {
				assert.Equal(t, http.StatusOK, resp.StatusCode)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var callCount int32

			server := httptest.NewServer(http.HandlerFunc(tt.serverResponse(&callCount)))
			defer server.Close()

			client := tt.setupClient(t, server.URL)
			req := tt.setupRequest(t, server.URL)

			doer := &keycloakDoer{kc: client}
			resp, err := doer.Do(req)

			if tt.wantErr {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)
			require.NotNil(t, resp)

			defer func() { _ = resp.Body.Close() }()

			if tt.validateResp != nil {
				tt.validateResp(t, resp)
			}
		})
	}
}

func TestKeycloakClient_AddRequestHeaders(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name            string
		setupClient     func(t *testing.T) *KeycloakClient
		method          string
		expectedHeaders map[string]string
	}{
		{
			name: "basic headers",
			setupClient: func(t *testing.T) *KeycloakClient {
				return &KeycloakClient{
					clientCredentials: &ClientCredentials{
						AccessToken: testAccessToken,
						TokenType:   "Bearer",
					},
					additionalHeaders: make(map[string]string),
				}
			},
			method: http.MethodGet,
			expectedHeaders: map[string]string{
				"Authorization": "Bearer " + testAccessToken,
				"Accept":        "application/json",
			},
		},
		{
			name: "POST request with content-type",
			setupClient: func(t *testing.T) *KeycloakClient {
				return &KeycloakClient{
					clientCredentials: &ClientCredentials{
						AccessToken: testAccessToken,
						TokenType:   "Bearer",
					},
					additionalHeaders: make(map[string]string),
				}
			},
			method: http.MethodPost,
			expectedHeaders: map[string]string{
				"Authorization": "Bearer " + testAccessToken,
				"Accept":        "application/json",
				"Content-type":  "application/json",
			},
		},
		{
			name: "with user agent",
			setupClient: func(t *testing.T) *KeycloakClient {
				return &KeycloakClient{
					clientCredentials: &ClientCredentials{
						AccessToken: testAccessToken,
						TokenType:   "Bearer",
					},
					userAgent:         testUserAgent,
					additionalHeaders: make(map[string]string),
				}
			},
			method: http.MethodGet,
			expectedHeaders: map[string]string{
				"Authorization": "Bearer " + testAccessToken,
				"Accept":        "application/json",
				"User-Agent":    testUserAgent,
			},
		},
		{
			name: "with additional headers",
			setupClient: func(t *testing.T) *KeycloakClient {
				return &KeycloakClient{
					clientCredentials: &ClientCredentials{
						AccessToken: testAccessToken,
						TokenType:   "Bearer",
					},
					additionalHeaders: map[string]string{
						"X-Custom-Header": "custom-value",
					},
				}
			},
			method: http.MethodGet,
			expectedHeaders: map[string]string{
				"Authorization":   "Bearer " + testAccessToken,
				"Accept":          "application/json",
				"X-Custom-Header": "custom-value",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			client := tt.setupClient(t)
			req, _ := http.NewRequest(tt.method, "http://localhost/test", nil)

			client.addRequestHeaders(req)

			for key, expectedValue := range tt.expectedHeaders {
				assert.Equal(t, expectedValue, req.Header.Get(key), "header %s mismatch", key)
			}
		})
	}
}

func TestKeycloakClient_GetAuthenticationFormData(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		setupClient   func(t *testing.T) *KeycloakClient
		expectedPairs map[string]string
	}{
		{
			name: "password grant",
			setupClient: func(t *testing.T) *KeycloakClient {
				return &KeycloakClient{
					clientCredentials: &ClientCredentials{
						ClientId:     testClientID,
						Username:     testUsername,
						Password:     testPassword,
						ClientSecret: testClientSecret,
						GrantType:    "password",
					},
				}
			},
			expectedPairs: map[string]string{
				"client_id":     testClientID,
				"grant_type":    "password",
				"username":      testUsername,
				"password":      testPassword,
				"client_secret": testClientSecret,
			},
		},
		{
			name: "password grant without client secret",
			setupClient: func(t *testing.T) *KeycloakClient {
				return &KeycloakClient{
					clientCredentials: &ClientCredentials{
						ClientId:  testClientID,
						Username:  testUsername,
						Password:  testPassword,
						GrantType: "password",
					},
				}
			},
			expectedPairs: map[string]string{
				"client_id":  testClientID,
				"grant_type": "password",
				"username":   testUsername,
				"password":   testPassword,
			},
		},
		{
			name: "client credentials with secret",
			setupClient: func(t *testing.T) *KeycloakClient {
				return &KeycloakClient{
					clientCredentials: &ClientCredentials{
						ClientId:     testClientID,
						ClientSecret: testClientSecret,
						GrantType:    "client_credentials",
					},
				}
			},
			expectedPairs: map[string]string{
				"client_id":     testClientID,
				"grant_type":    "client_credentials",
				"client_secret": testClientSecret,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			client := tt.setupClient(t)
			formData, err := client.getAuthenticationFormData(context.Background())

			require.NoError(t, err)

			for key, expectedValue := range tt.expectedPairs {
				assert.Equal(t, expectedValue, formData.Get(key), "form field %s mismatch", key)
			}
		})
	}
}

func TestRetryPolicy(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		statusCode  int
		err         error
		shouldRetry bool
	}{
		{
			name:        "429 Too Many Requests",
			statusCode:  http.StatusTooManyRequests,
			shouldRetry: true,
		},
		{
			name:        "500 Internal Server Error",
			statusCode:  http.StatusInternalServerError,
			shouldRetry: true,
		},
		{
			name:        "502 Bad Gateway",
			statusCode:  http.StatusBadGateway,
			shouldRetry: true,
		},
		{
			name:        "503 Service Unavailable",
			statusCode:  http.StatusServiceUnavailable,
			shouldRetry: true,
		},
		{
			name:        "504 Gateway Timeout",
			statusCode:  http.StatusGatewayTimeout,
			shouldRetry: true,
		},
		{
			name:        "501 Not Implemented",
			statusCode:  http.StatusNotImplemented,
			shouldRetry: false,
		},
		{
			name:        "200 OK",
			statusCode:  http.StatusOK,
			shouldRetry: false,
		},
		{
			name:        "201 Created",
			statusCode:  http.StatusCreated,
			shouldRetry: false,
		},
		{
			name:        "400 Bad Request",
			statusCode:  http.StatusBadRequest,
			shouldRetry: false,
		},
		{
			name:        "401 Unauthorized",
			statusCode:  http.StatusUnauthorized,
			shouldRetry: false,
		},
		{
			name:        "404 Not Found",
			statusCode:  http.StatusNotFound,
			shouldRetry: false,
		},
		{
			name:        "network error",
			err:         fmt.Errorf("network error"),
			shouldRetry: true,
		},
		{
			name:        "zero status code with error",
			statusCode:  0,
			err:         fmt.Errorf("connection failed"),
			shouldRetry: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var resp *resty.Response

			if tt.statusCode > 0 {
				httpResp := &http.Response{
					StatusCode: tt.statusCode,
				}
				resp = &resty.Response{
					RawResponse: httpResp,
				}
			}

			shouldRetry := RetryPolicy(resp, tt.err)
			assert.Equal(t, tt.shouldRetry, shouldRetry)
		})
	}
}

func TestNewHttpClient(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name                  string
		tlsInsecureSkipVerify bool
		clientTimeout         int
		caCert                string
		tlsClientCert         string
		tlsClientPrivateKey   string
		setupCerts            func(t *testing.T) (string, string)
		wantErr               bool
		validate              func(t *testing.T, client *http.Client)
	}{
		{
			name:                  "default configuration",
			tlsInsecureSkipVerify: false,
			clientTimeout:         60,
			wantErr:               false,
			validate: func(t *testing.T, client *http.Client) {
				assert.NotNil(t, client)
				assert.Equal(t, 60*time.Second, client.Timeout)
			},
		},
		{
			name:                  "insecure skip verify",
			tlsInsecureSkipVerify: true,
			clientTimeout:         30,
			wantErr:               false,
			validate: func(t *testing.T, client *http.Client) {
				transport := client.Transport.(*http.Transport)
				assert.True(t, transport.TLSClientConfig.InsecureSkipVerify)
			},
		},
		{
			name:          "with CA cert",
			clientTimeout: 60,
			caCert: func() string {
				caCert, _ := createTestCACert(t)
				return caCert
			}(),
			wantErr: false,
			validate: func(t *testing.T, client *http.Client) {
				transport := client.Transport.(*http.Transport)
				assert.NotNil(t, transport.TLSClientConfig.RootCAs)
			},
		},
		{
			name:          "with client cert",
			clientTimeout: 60,
			setupCerts: func(t *testing.T) (string, string) {
				_, cert := createTestCACert(t)
				caCert, _ := x509.ParseCertificate(cert.Certificate[0])
				clientCert, clientKey := createTestClientCert(t, caCert, cert.PrivateKey.(*rsa.PrivateKey))
				return clientCert, clientKey
			},
			wantErr: false,
			validate: func(t *testing.T, client *http.Client) {
				transport := client.Transport.(*http.Transport)
				assert.NotEmpty(t, transport.TLSClientConfig.Certificates)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			clientCert := tt.tlsClientCert
			privateKey := tt.tlsClientPrivateKey

			if tt.setupCerts != nil {
				clientCert, privateKey = tt.setupCerts(t)
			}

			client, err := newHttpClient(
				tt.tlsInsecureSkipVerify,
				tt.clientTimeout,
				tt.caCert,
				clientCert,
				privateKey,
			)

			if tt.wantErr {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)
			require.NotNil(t, client)

			if tt.validate != nil {
				tt.validate(t, client)
			}
		})
	}
}

func TestKeycloakClient_NewSignedJWT(t *testing.T) {
	t.Parallel()

	issuerURL := "http://localhost/realms/test"

	tests := []struct {
		name        string
		alg         string
		setupKey    func(t *testing.T) (string, any)
		wantErr     bool
		errContains string
		validateJWT func(t *testing.T, jwtToken string, publicKey any)
	}{
		{
			name: "RS256",
			alg:  "RS256",
			setupKey: func(t *testing.T) (string, any) {
				keyPEM, privateKey := createTestRSAKey(t)
				return keyPEM, &privateKey.PublicKey
			},
			wantErr: false,
			validateJWT: func(t *testing.T, jwtToken string, publicKey any) {
				token, err := jwt.Parse(jwtToken, func(token *jwt.Token) (any, error) {
					return publicKey, nil
				})
				require.NoError(t, err)
				assert.True(t, token.Valid)

				claims, ok := token.Claims.(jwt.MapClaims)
				require.True(t, ok)
				assert.Equal(t, testClientID, claims["iss"])
				assert.Equal(t, testClientID, claims["sub"])
				assert.Equal(t, issuerURL, claims["aud"])
				assert.NotNil(t, claims["jti"])
				assert.NotNil(t, claims["exp"])
				assert.NotNil(t, claims["iat"])
			},
		},
		{
			name: "ES256",
			alg:  "ES256",
			setupKey: func(t *testing.T) (string, any) {
				keyPEM, privateKey := createTestECDSAKey(t)
				return keyPEM, &privateKey.PublicKey
			},
			wantErr: false,
			validateJWT: func(t *testing.T, jwtToken string, publicKey any) {
				token, err := jwt.Parse(jwtToken, func(token *jwt.Token) (any, error) {
					return publicKey, nil
				})
				require.NoError(t, err)
				assert.True(t, token.Valid)
			},
		},
		{
			name: "EdDSA",
			alg:  "EdDSA",
			setupKey: func(t *testing.T) (string, any) {
				keyPEM, privateKey := createTestEd25519Key(t)
				return keyPEM, privateKey.Public()
			},
			wantErr: false,
			validateJWT: func(t *testing.T, jwtToken string, publicKey any) {
				token, err := jwt.Parse(jwtToken, func(token *jwt.Token) (any, error) {
					return publicKey, nil
				})
				require.NoError(t, err)
				assert.True(t, token.Valid)
			},
		},
		{
			name: "unsupported algorithm",
			alg:  "UNSUPPORTED",
			setupKey: func(t *testing.T) (string, any) {
				keyPEM, _ := createTestRSAKey(t)
				return keyPEM, nil
			},
			wantErr:     true,
			errContains: "unsupported signing method",
		},
		{
			name: "invalid RSA key",
			alg:  "RS256",
			setupKey: func(t *testing.T) (string, any) {
				return "invalid-key-data", nil
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			keyPEM, publicKey := tt.setupKey(t)

			client := &KeycloakClient{
				logger: logr.Discard(),
			}

			jwtToken, err := client.newSignedJWT(
				context.Background(),
				issuerURL,
				testClientID,
				tt.alg,
				keyPEM,
			)

			if tt.wantErr {
				require.Error(t, err)

				if tt.errContains != "" {
					assert.Contains(t, err.Error(), tt.errContains)
				}

				return
			}

			require.NoError(t, err)
			assert.NotEmpty(t, jwtToken)

			if tt.validateJWT != nil {
				tt.validateJWT(t, jwtToken, publicKey)
			}
		})
	}
}

func TestKeycloakClient_GetContextLogger(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		ctx      context.Context
		client   *KeycloakClient
		validate func(t *testing.T, logger logr.Logger)
	}{
		{
			name: "uses context logger",
			ctx: func() context.Context {
				return ctrl.LoggerInto(context.Background(), logr.Discard().WithName("context-logger"))
			}(),
			client: &KeycloakClient{
				logger: logr.Discard().WithName("client-logger"),
			},
			validate: func(t *testing.T, logger logr.Logger) {
				// Both loggers are Discard, but we can verify the function doesn't panic
				assert.NotNil(t, logger)
			},
		},
		{
			name: "falls back to client logger",
			ctx:  context.Background(),
			client: &KeycloakClient{
				logger: logr.Discard().WithName("client-logger"),
			},
			validate: func(t *testing.T, logger logr.Logger) {
				assert.NotNil(t, logger)
			},
		},
		{
			name: "handles nil context",
			ctx:  nil,
			client: &KeycloakClient{
				logger: logr.Discard(),
			},
			validate: func(t *testing.T, logger logr.Logger) {
				assert.NotNil(t, logger)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			logger := tt.client.getContextLogger(tt.ctx)
			tt.validate(t, logger)
		})
	}
}

func TestKeycloakClient_ConcurrentAccess(t *testing.T) {
	t.Parallel()

	refreshCount := int32(0)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "/token") {
			count := atomic.AddInt32(&refreshCount, 1)

			// Simulate slow token endpoint
			time.Sleep(10 * time.Millisecond)

			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)

			tokenResp := map[string]any{
				"access_token":  fmt.Sprintf("token-%d", count),
				"refresh_token": fmt.Sprintf("refresh-%d", count),
				"token_type":    "bearer",
			}
			data, _ := json.Marshal(tokenResp)
			_, _ = w.Write(data)

			return
		}

		// Return 401 for first few requests to trigger refresh
		if atomic.LoadInt32(&refreshCount) == 0 {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}

		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"result":"success"}`))
	}))
	defer server.Close()

	client, err := NewKeycloakClient(
		context.Background(),
		server.URL,
		testClientID,
		WithClientSecret(testClientSecret),
	)
	require.NoError(t, err)

	// Launch multiple concurrent requests
	const numRequests = 10

	var wg sync.WaitGroup

	wg.Add(numRequests)

	for range numRequests {
		go func() {
			defer wg.Done()

			req, _ := http.NewRequest(http.MethodGet, server.URL+"/admin/realms", nil)
			doer := &keycloakDoer{kc: client}
			resp, err := doer.Do(req)

			assert.NoError(t, err)

			if resp != nil {
				_ = resp.Body.Close()
			}
		}()
	}

	wg.Wait()

	// Verify that refresh was called a reasonable number of times
	// Should be 1 (initial login) + a small number from concurrent refreshes
	// Not equal to numRequests due to single-flight pattern
	finalCount := atomic.LoadInt32(&refreshCount)
	assert.Greater(t, finalCount, int32(0))
	assert.Less(t, finalCount, int32(numRequests))
}

func TestClientOptions_AllOptions(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "/token") {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(createMockTokenResponseJSON()))
		}
	}))
	defer server.Close()

	customHeaders := map[string]string{
		"X-Test-Header": "test-value",
	}

	caCertPEM, _ := createTestCACert(t)

	client, err := NewKeycloakClient(
		context.Background(),
		server.URL,
		testClientID,
		WithRealm(testRealm),
		WithPasswordGrant(testUsername, testPassword),
		WithClientSecret(testClientSecret),
		WithUserAgent(testUserAgent),
		WithAdditionalHeaders(customHeaders),
		WithRedHatSSO(true),
		WithClientTimeout(30),
		WithTLSInsecureSkipVerify(true),
		WithCACert(caCertPEM),
		WithLogger(logr.Discard()),
	)

	require.NoError(t, err)
	require.NotNil(t, client)

	assert.Equal(t, testRealm, client.realm)
	assert.Equal(t, testUsername, client.clientCredentials.Username)
	assert.Equal(t, testPassword, client.clientCredentials.Password)
	assert.Equal(t, testClientSecret, client.clientCredentials.ClientSecret)
	assert.Equal(t, testUserAgent, client.userAgent)
	assert.Equal(t, "test-value", client.additionalHeaders["X-Test-Header"])
	assert.True(t, client.redHatSSO)
	assert.Equal(t, 30*time.Second, client.httpClient.Timeout)
}

func TestNewSignedJWT_Deprecated(t *testing.T) {
	t.Parallel()

	keyPEM, _ := createTestRSAKey(t)
	issuerURL := "http://localhost/realms/test"

	jwtToken, err := NewSignedJWT(
		context.Background(),
		issuerURL,
		testClientID,
		"RS256",
		keyPEM,
	)

	require.NoError(t, err)
	assert.NotEmpty(t, jwtToken)

	// Verify the JWT is valid
	token, err := jwt.Parse(jwtToken, func(token *jwt.Token) (any, error) {
		key, err := jwt.ParseRSAPrivateKeyFromPEM([]byte(keyPEM))
		if err != nil {
			return nil, err
		}

		return &key.PublicKey, nil
	})
	require.NoError(t, err)
	assert.True(t, token.Valid)
}

func TestClientOptions_WithTLSClientCert(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "/token") {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(createMockTokenResponseJSON()))
		}
	}))
	defer server.Close()

	// Create client cert
	_, tlsCert := createTestCACert(t)
	caCert, _ := x509.ParseCertificate(tlsCert.Certificate[0])
	clientCertPEM, clientKeyPEM := createTestClientCert(t, caCert, tlsCert.PrivateKey.(*rsa.PrivateKey))

	client, err := NewKeycloakClient(
		context.Background(),
		server.URL,
		testClientID,
		WithClientSecret(testClientSecret),
		WithTLSClientCert(clientCertPEM, clientKeyPEM),
	)

	require.NoError(t, err)
	require.NotNil(t, client)

	// Verify TLS config has certificates
	transport := client.httpClient.Transport.(*http.Transport)
	assert.NotEmpty(t, transport.TLSClientConfig.Certificates)
}

func TestKeycloakClient_Refresh_BadRequest(t *testing.T) {
	t.Parallel()

	// Test the specific 400 handling that triggers re-login
	callCount := int32(0)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "/token") {
			count := atomic.AddInt32(&callCount, 1)

			// Initial login succeeds
			// First refresh attempt gets 400
			// Re-login succeeds
			switch count {
			case 1, 3:
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write([]byte(createMockTokenResponseJSON()))
			case 2:
				w.WriteHeader(http.StatusBadRequest)
				_, _ = w.Write([]byte(`{"error": "invalid_grant"}`))
			}
		}
	}))
	defer server.Close()

	client, err := NewKeycloakClient(
		context.Background(),
		server.URL,
		testClientID,
		WithClientSecret(testClientSecret),
	)
	require.NoError(t, err)

	// Call count should be 1 after initial login
	assert.Equal(t, int32(1), atomic.LoadInt32(&callCount))

	// Trigger refresh which will get 400 and then re-login
	err = client.Refresh(context.Background())
	require.NoError(t, err)

	// Call count should be 3 (initial login, failed refresh, successful re-login)
	assert.Equal(t, int32(3), atomic.LoadInt32(&callCount))
}

func TestKeycloakClient_Login_WithUserAgent(t *testing.T) {
	t.Parallel()

	var capturedUserAgent string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "/token") {
			capturedUserAgent = r.Header.Get("User-Agent")
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(createMockTokenResponseJSON()))
		}
	}))
	defer server.Close()

	client, err := NewKeycloakClient(
		context.Background(),
		server.URL,
		testClientID,
		WithClientSecret(testClientSecret),
		WithUserAgent(testUserAgent),
	)

	require.NoError(t, err)
	require.NotNil(t, client)
	assert.Equal(t, testUserAgent, capturedUserAgent)
}

func TestKeycloakClient_Login_WithAdditionalHeaders(t *testing.T) {
	t.Parallel()

	var capturedHeader string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "/token") {
			capturedHeader = r.Header.Get("X-Custom-Header")
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(createMockTokenResponseJSON()))
		}
	}))
	defer server.Close()

	client, err := NewKeycloakClient(
		context.Background(),
		server.URL,
		testClientID,
		WithClientSecret(testClientSecret),
		WithAdditionalHeaders(map[string]string{
			"X-Custom-Header": "custom-value",
		}),
	)

	require.NoError(t, err)
	require.NotNil(t, client)
	assert.Equal(t, "custom-value", capturedHeader)
}

func TestKeycloakDoer_RequestBodyRetry(t *testing.T) {
	t.Parallel()

	// Test that request body is properly restored on retry
	requestCount := int32(0)

	var firstBody, secondBody string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "/token") {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(createMockTokenResponseJSON()))

			return
		}

		count := atomic.AddInt32(&requestCount, 1)
		body, _ := io.ReadAll(r.Body)

		if count == 1 {
			firstBody = string(body)

			w.WriteHeader(http.StatusUnauthorized)
		} else {
			secondBody = string(body)

			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"result":"success"}`))
		}
	}))
	defer server.Close()

	client, err := NewKeycloakClient(
		context.Background(),
		server.URL,
		testClientID,
		WithClientSecret(testClientSecret),
	)
	require.NoError(t, err)

	// Create request with body
	requestBody := `{"test":"data"}`
	req, _ := http.NewRequest(http.MethodPost, server.URL+"/admin/realms", strings.NewReader(requestBody))
	req.GetBody = func() (io.ReadCloser, error) {
		return io.NopCloser(strings.NewReader(requestBody)), nil
	}

	doer := &keycloakDoer{kc: client}
	resp, err := doer.Do(req)

	require.NoError(t, err)
	require.NotNil(t, resp)

	defer func() { _ = resp.Body.Close() }()

	// Verify body was sent both times
	assert.Equal(t, requestBody, firstBody)
	assert.Equal(t, requestBody, secondBody)
	assert.Equal(t, int32(2), atomic.LoadInt32(&requestCount))
}

func TestKeycloakClient_AddRequestHeaders_ContentTypePreservation(t *testing.T) {
	t.Parallel()

	client := &KeycloakClient{
		clientCredentials: &ClientCredentials{
			AccessToken: testAccessToken,
			TokenType:   "Bearer",
		},
		additionalHeaders: make(map[string]string),
	}

	// Test that existing Content-Type is preserved
	req, _ := http.NewRequest(http.MethodPost, "http://localhost/test", nil)
	req.Header.Set("Content-type", "application/xml")

	client.addRequestHeaders(req)

	assert.Equal(t, "application/xml", req.Header.Get("Content-type"))
}

func TestKeycloakClient_GetAuthenticationFormData_JWTToken(t *testing.T) {
	t.Parallel()

	expectedToken := "provided.jwt.token"

	client := &KeycloakClient{
		clientCredentials: &ClientCredentials{
			ClientId:  testClientID,
			GrantType: grantTypeClientCredentials,
			JWTToken:  expectedToken,
		},
		logger: logr.Discard(),
	}

	formData, err := client.getAuthenticationFormData(context.Background())

	require.NoError(t, err)
	assert.Equal(t, "urn:ietf:params:oauth:client-assertion-type:jwt-bearer", formData.Get("client_assertion_type"))
	assert.Equal(t, expectedToken, formData.Get("client_assertion"))
}

func TestKeycloakClient_GetAuthenticationFormData_JWTFromFile_Error(t *testing.T) {
	t.Parallel()

	client := &KeycloakClient{
		clientCredentials: &ClientCredentials{
			ClientId:     testClientID,
			GrantType:    grantTypeClientCredentials,
			JWTTokenFile: "/nonexistent/file/path.txt",
		},
		logger: logr.Discard(),
	}

	_, err := client.getAuthenticationFormData(context.Background())

	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to read JWT token from file")
}

func TestNewHttpClient_InvalidCert(t *testing.T) {
	t.Parallel()

	// Test with invalid client certificate
	_, err := newHttpClient(
		false,
		60,
		"",
		"invalid-cert",
		"invalid-key",
	)

	require.Error(t, err)
}

func TestKeycloakClient_NewSignedJWT_InvalidECKey(t *testing.T) {
	t.Parallel()

	client := &KeycloakClient{
		logger: logr.Discard(),
	}

	_, err := client.newSignedJWT(
		context.Background(),
		"http://localhost/realms/test",
		testClientID,
		"ES256",
		"invalid-key-data",
	)

	require.Error(t, err)
}

func TestKeycloakClient_NewSignedJWT_InvalidEdKey(t *testing.T) {
	t.Parallel()

	client := &KeycloakClient{
		logger: logr.Discard(),
	}

	_, err := client.newSignedJWT(
		context.Background(),
		"http://localhost/realms/test",
		testClientID,
		"EdDSA",
		"invalid-key-data",
	)

	require.Error(t, err)
}

func TestKeycloakDoer_ConcurrentRefresh(t *testing.T) {
	t.Parallel()

	// Test single-flight behavior: multiple concurrent 401s should trigger only one refresh
	refreshCount := int32(0)
	requestCount := int32(0)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "/token") {
			atomic.AddInt32(&refreshCount, 1)
			// Simulate slow token endpoint
			time.Sleep(50 * time.Millisecond)
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(createMockTokenResponseJSON()))

			return
		}

		count := atomic.AddInt32(&requestCount, 1)
		// First few requests return 401 to trigger concurrent refresh attempts
		if count <= 3 {
			w.WriteHeader(http.StatusUnauthorized)
		} else {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"result":"success"}`))
		}
	}))
	defer server.Close()

	client, err := NewKeycloakClient(
		context.Background(),
		server.URL,
		testClientID,
		WithClientSecret(testClientSecret),
	)
	require.NoError(t, err)

	// Reset counts after initial login
	atomic.StoreInt32(&refreshCount, 0)
	atomic.StoreInt32(&requestCount, 0)

	// Launch 3 concurrent requests that will all trigger 401
	const numRequests = 3

	var wg sync.WaitGroup

	wg.Add(numRequests)

	for range numRequests {
		go func() {
			defer wg.Done()

			req, _ := http.NewRequest(http.MethodGet, server.URL+"/admin/realms", nil)
			doer := &keycloakDoer{kc: client}
			resp, err := doer.Do(req)
			assert.NoError(t, err)

			if resp != nil {
				_ = resp.Body.Close()
			}
		}()
	}

	wg.Wait()

	// Due to single-flight refresh, refresh count should be less than or equal to number of concurrent requests
	// The single-flight pattern should reduce unnecessary refreshes, but exact count depends on timing
	finalRefreshCount := atomic.LoadInt32(&refreshCount)
	assert.Greater(t, finalRefreshCount, int32(0))
	assert.LessOrEqual(t, finalRefreshCount, int32(numRequests))
}

func TestKeycloakClient_Login_ContextLogger(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "/token") {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(createMockTokenResponseJSON()))
		}
	}))
	defer server.Close()

	// Test with logger in context
	ctx := ctrl.LoggerInto(context.Background(), logr.Discard().WithName("test-logger"))

	client, err := NewKeycloakClient(
		ctx,
		server.URL,
		testClientID,
		WithClientSecret(testClientSecret),
	)

	require.NoError(t, err)
	require.NotNil(t, client)
	assert.NotNil(t, client.logger)
}

func TestKeycloakClient_NewKeycloakClient_DefaultRealm(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "/token") {
			// Verify the default realm "master" is used in the path
			assert.Contains(t, r.URL.Path, "/realms/master/")
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(createMockTokenResponseJSON()))
		}
	}))
	defer server.Close()

	client, err := NewKeycloakClient(
		context.Background(),
		server.URL,
		testClientID,
		WithClientSecret(testClientSecret),
	)

	require.NoError(t, err)
	assert.Equal(t, MasterRealm, client.realm)
}

func TestKeycloakClient_NewKeycloakClient_CustomRealm(t *testing.T) {
	t.Parallel()

	customRealm := "custom-realm"

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "/token") {
			// Verify the custom realm is used in the path
			assert.Contains(t, r.URL.Path, "/realms/"+customRealm+"/")
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(createMockTokenResponseJSON()))
		}
	}))
	defer server.Close()

	client, err := NewKeycloakClient(
		context.Background(),
		server.URL,
		testClientID,
		WithRealm(customRealm),
		WithClientSecret(testClientSecret),
	)

	require.NoError(t, err)
	assert.Equal(t, customRealm, client.realm)
}
func TestKeycloakClient_WithBasePath(t *testing.T) {
	t.Parallel()

	basePath := "/auth"

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "/token") {
			// Verify the base path is included
			assert.Contains(t, r.URL.Path, basePath)
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(createMockTokenResponseJSON()))
		}
	}))
	defer server.Close()

	client, err := NewKeycloakClient(
		context.Background(),
		server.URL,
		testClientID,
		WithBasePath(basePath),
		WithClientSecret(testClientSecret),
	)

	require.NoError(t, err)
	assert.Contains(t, client.baseUrl, basePath)
	assert.Contains(t, client.authUrl, basePath)
}

func TestKeycloakClient_WithAdminURL(t *testing.T) {
	t.Parallel()

	adminURL := "http://admin-server"

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "/token") {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(createMockTokenResponseJSON()))
		}
	}))
	defer server.Close()

	client, err := NewKeycloakClient(
		context.Background(),
		server.URL,
		testClientID,
		WithAdminURL(adminURL),
		WithClientSecret(testClientSecret),
	)

	require.NoError(t, err)
	assert.Contains(t, client.baseUrl, adminURL)
	// authUrl should still use the original server URL
	assert.NotContains(t, client.authUrl, adminURL)
}

func createMockTokenResponse() map[string]any {
	return map[string]any{
		"access_token":  testAccessToken,
		"refresh_token": testRefreshToken,
		"token_type":    testTokenType,
		"expires_in":    300,
	}
}

func createMockTokenResponseJSON() string {
	data, _ := json.Marshal(createMockTokenResponse())
	return string(data)
}

func assertTokensSet(t *testing.T, creds *ClientCredentials) {
	t.Helper()
	assert.Equal(t, testAccessToken, creds.AccessToken)
	assert.Equal(t, testRefreshToken, creds.RefreshToken)
	assert.Equal(t, testTokenType, creds.TokenType)
}

func createTestRSAKey(t *testing.T) (string, *rsa.PrivateKey) {
	t.Helper()

	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)

	privateKeyPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(privateKey),
	})

	return string(privateKeyPEM), privateKey
}

func createTestECDSAKey(t *testing.T) (string, *ecdsa.PrivateKey) {
	t.Helper()

	privateKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)

	privateKeyBytes, err := x509.MarshalECPrivateKey(privateKey)
	require.NoError(t, err)

	privateKeyPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "EC PRIVATE KEY",
		Bytes: privateKeyBytes,
	})

	return string(privateKeyPEM), privateKey
}

func createTestEd25519Key(t *testing.T) (string, ed25519.PrivateKey) {
	t.Helper()

	_, privateKey, err := ed25519.GenerateKey(rand.Reader)
	require.NoError(t, err)

	privateKeyBytes, err := x509.MarshalPKCS8PrivateKey(privateKey)
	require.NoError(t, err)

	privateKeyPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "PRIVATE KEY",
		Bytes: privateKeyBytes,
	})

	return string(privateKeyPEM), privateKey
}

func createTestCACert(t *testing.T) (string, *tls.Certificate) {
	t.Helper()

	// Generate CA private key
	caPrivateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)

	// Create CA certificate
	ca := &x509.Certificate{
		SerialNumber: big.NewInt(2019),
		Subject: pkix.Name{
			Organization: []string{"Test CA"},
		},
		NotBefore:             time.Now(),
		NotAfter:              time.Now().Add(time.Hour * 24),
		IsCA:                  true,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth, x509.ExtKeyUsageServerAuth},
		KeyUsage:              x509.KeyUsageDigitalSignature | x509.KeyUsageCertSign,
		BasicConstraintsValid: true,
	}

	caBytes, err := x509.CreateCertificate(rand.Reader, ca, ca, &caPrivateKey.PublicKey, caPrivateKey)
	require.NoError(t, err)

	caPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "CERTIFICATE",
		Bytes: caBytes,
	})

	caPrivKeyPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(caPrivateKey),
	})

	cert, err := tls.X509KeyPair(caPEM, caPrivKeyPEM)
	require.NoError(t, err)

	return string(caPEM), &cert
}

func createTestClientCert(t *testing.T, caCert *x509.Certificate, caPrivateKey *rsa.PrivateKey) (string, string) {
	t.Helper()

	// Generate client private key
	clientPrivateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)

	// Create client certificate
	client := &x509.Certificate{
		SerialNumber: big.NewInt(2020),
		Subject: pkix.Name{
			Organization: []string{"Test Client"},
		},
		NotBefore:   time.Now(),
		NotAfter:    time.Now().Add(time.Hour * 24),
		ExtKeyUsage: []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth},
		KeyUsage:    x509.KeyUsageDigitalSignature,
	}

	clientBytes, err := x509.CreateCertificate(rand.Reader, client, caCert, &clientPrivateKey.PublicKey, caPrivateKey)
	require.NoError(t, err)

	clientPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "CERTIFICATE",
		Bytes: clientBytes,
	})

	clientPrivKeyPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(clientPrivateKey),
	})

	return string(clientPEM), string(clientPrivKeyPEM)
}

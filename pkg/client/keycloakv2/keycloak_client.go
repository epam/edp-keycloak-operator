package keycloakv2

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/go-logr/logr"
	"github.com/google/uuid"
	ctrl "sigs.k8s.io/controller-runtime"

	"github.com/golang-jwt/jwt/v5"

	"golang.org/x/net/publicsuffix"

	"github.com/go-resty/resty/v2"

	"github.com/epam/edp-keycloak-operator/pkg/client/keycloakv2/generated"
)

type KeycloakClient struct {
	mu                  sync.Mutex
	baseUrl             string
	authUrl             string
	realm               string
	clientCredentials   *ClientCredentials
	httpClient          *http.Client
	initialLogin        bool
	userAgent           string
	additionalHeaders   map[string]string
	redHatSSO           bool
	accessTokenProvided bool
	logger              logr.Logger
	Users               UsersClient
	Realms              RealmClient
}

type ClientCredentials struct {
	ClientId      string
	ClientSecret  string
	JWTSigningKey string
	JWTSigningAlg string
	JWTToken      string
	JWTTokenFile  string
	Username      string
	Password      string
	GrantType     string
	AccessToken   string `json:"access_token"`
	RefreshToken  string `json:"refresh_token"`
	TokenType     string `json:"token_type"`
}

const (
	issuerUrl = "%s/realms/%s"
	tokenUrl  = "%s/realms/%s/protocol/openid-connect/token"

	// DefaultRealm is the default Keycloak realm if none is specified
	DefaultRealm = "master"

	// AdminCLIClientID is the default client ID for Keycloak admin operations
	AdminCLIClientID = "admin-cli"

	grantTypeClientCredentials = "client_credentials"

	// debugVerbosityLevel is the verbosity level for debug-level logs
	debugVerbosityLevel = 1
)

// clientConfig holds configuration parameters that need to be captured before creating the client
type clientConfig struct {
	basePath              string
	adminUrl              string
	clientTimeout         int
	caCert                string
	tlsInsecureSkipVerify bool
	tlsClientCert         string
	tlsClientPrivateKey   string
	logger                logr.Logger
}

// ClientOption is a function that configures a KeycloakClient
type ClientOption func(*KeycloakClient, *clientConfig)

// WithRealm sets a custom realm (default: "master")
func WithRealm(realm string) ClientOption {
	return func(c *KeycloakClient, cfg *clientConfig) {
		c.realm = realm
	}
}

// WithBasePath sets the base path for the Keycloak API
func WithBasePath(basePath string) ClientOption {
	return func(c *KeycloakClient, cfg *clientConfig) {
		cfg.basePath = basePath
	}
}

// WithAdminURL sets a separate admin URL for the Keycloak API
func WithAdminURL(adminUrl string) ClientOption {
	return func(c *KeycloakClient, cfg *clientConfig) {
		cfg.adminUrl = adminUrl
	}
}

// WithClientSecret sets the client secret (can be used with both password and client_credentials grants)
func WithClientSecret(clientSecret string) ClientOption {
	return func(c *KeycloakClient, cfg *clientConfig) {
		c.clientCredentials.ClientSecret = clientSecret
	}
}

// WithPasswordGrant configures password grant authentication.
// Note: For password grant, typically use "admin-cli" as the clientId parameter.
// Use WithClientSecret separately if the client also requires a client secret.
func WithPasswordGrant(username, password string) ClientOption {
	return func(c *KeycloakClient, cfg *clientConfig) {
		c.clientCredentials.Username = username
		c.clientCredentials.Password = password
		c.clientCredentials.GrantType = "password"
	}
}

// WithAccessToken sets a pre-existing access token
func WithAccessToken(accessToken string) ClientOption {
	return func(c *KeycloakClient, cfg *clientConfig) {
		c.clientCredentials.AccessToken = accessToken
		c.clientCredentials.TokenType = "bearer"
		c.accessTokenProvided = true
	}
}

// WithJWTAuth configures JWT-based authentication for client_credentials grant
func WithJWTAuth(jwtSigningAlg, jwtSigningKey, jwtToken, jwtTokenFile string) ClientOption {
	return func(c *KeycloakClient, cfg *clientConfig) {
		c.clientCredentials.JWTSigningAlg = jwtSigningAlg
		c.clientCredentials.JWTSigningKey = jwtSigningKey
		c.clientCredentials.JWTToken = jwtToken
		c.clientCredentials.JWTTokenFile = jwtTokenFile
		c.clientCredentials.GrantType = grantTypeClientCredentials
	}
}

// WithInitialLogin controls whether to perform login during client creation (default: true)
func WithInitialLogin(initialLogin bool) ClientOption {
	return func(c *KeycloakClient, cfg *clientConfig) {
		c.initialLogin = initialLogin
	}
}

// WithClientTimeout sets the HTTP client timeout in seconds (default: 60)
func WithClientTimeout(timeout int) ClientOption {
	return func(c *KeycloakClient, cfg *clientConfig) {
		cfg.clientTimeout = timeout
	}
}

// WithCACert sets a custom CA certificate for TLS
func WithCACert(caCert string) ClientOption {
	return func(c *KeycloakClient, cfg *clientConfig) {
		cfg.caCert = caCert
	}
}

// WithTLSInsecureSkipVerify disables TLS certificate verification
func WithTLSInsecureSkipVerify(skip bool) ClientOption {
	return func(c *KeycloakClient, cfg *clientConfig) {
		cfg.tlsInsecureSkipVerify = skip
	}
}

// WithTLSClientCert sets client certificate and private key for mutual TLS
func WithTLSClientCert(cert, privateKey string) ClientOption {
	return func(c *KeycloakClient, cfg *clientConfig) {
		cfg.tlsClientCert = cert
		cfg.tlsClientPrivateKey = privateKey
	}
}

// WithUserAgent sets a custom User-Agent header
func WithUserAgent(userAgent string) ClientOption {
	return func(c *KeycloakClient, cfg *clientConfig) {
		c.userAgent = userAgent
	}
}

// WithRedHatSSO enables Red Hat SSO mode
func WithRedHatSSO(enabled bool) ClientOption {
	return func(c *KeycloakClient, cfg *clientConfig) {
		c.redHatSSO = enabled
	}
}

// WithAdditionalHeaders sets additional headers to include in requests
func WithAdditionalHeaders(headers map[string]string) ClientOption {
	return func(c *KeycloakClient, cfg *clientConfig) {
		c.additionalHeaders = headers
	}
}

// WithLogger sets a custom logger for the KeycloakClient.
// If not provided, the client will attempt to use the logger from context,
// falling back to a no-op logger (logr.Discard).
func WithLogger(log logr.Logger) ClientOption {
	return func(c *KeycloakClient, cfg *clientConfig) {
		cfg.logger = log
	}
}

// NewKeycloakClient creates a new KeycloakClient with the provided options.
//
// Required parameters:
//   - url: The base URL of the Keycloak server
//   - clientId: The client ID (use "admin-cli" for password grant)
//
// Optional parameters (can be set using ClientOption functions):
//   - WithRealm: Set the Keycloak realm (default: "master")
//   - WithPasswordGrant: Enable password grant authentication (use "admin-cli" as clientId)
//   - WithClientSecret: Set client secret (used with password or client_credentials grant)
//   - WithAccessToken: Use a pre-existing access token
//   - WithJWTAuth: Enable JWT-based authentication
//   - WithInitialLogin: Control whether to login during client creation (default: true)
//   - WithBasePath/WithAdminURL: Customize API paths
//   - WithClientTimeout: Set HTTP client timeout (default: 60 seconds)
//   - WithTLSInsecureSkipVerify: Disable TLS certificate verification
//   - WithCACert: Set custom CA certificate
//   - WithTLSClientCert: Set client certificate for mutual TLS
//   - WithUserAgent: Set custom User-Agent
//   - WithRedHatSSO: Enable Red Hat SSO mode
//   - WithAdditionalHeaders: Add custom headers
//   - WithLogger: Set a custom logger (default: uses context logger or discard)
func NewKeycloakClient(ctx context.Context, baseURL, clientId string, opts ...ClientOption) (*KeycloakClient, error) {
	if baseURL == "" {
		return nil, fmt.Errorf("url is required")
	}

	if clientId == "" {
		return nil, fmt.Errorf("clientId is required")
	}

	// Initialize configuration with defaults
	config := &clientConfig{
		clientTimeout:         60, // default timeout in seconds
		tlsInsecureSkipVerify: false,
	}

	// Initialize client with defaults
	keycloakClient := &KeycloakClient{
		authUrl: baseURL,
		baseUrl: baseURL,
		realm:   DefaultRealm, // default to "master"
		clientCredentials: &ClientCredentials{
			ClientId: clientId,
		},
		initialLogin:      true,
		additionalHeaders: make(map[string]string),
	}

	// Apply options
	for _, opt := range opts {
		opt(keycloakClient, config)
	}

	// Initialize logger with fallback chain:
	// 1. Use explicitly provided logger via WithLogger option
	// 2. Try to get logger from context (includes trace ID if set)
	// 3. Fall back to discard logger (silent)
	var zeroLogger logr.Logger
	if config.logger != zeroLogger {
		keycloakClient.logger = config.logger
	} else if ctxLogger := ctrl.LoggerFrom(ctx); ctxLogger != zeroLogger {
		keycloakClient.logger = ctxLogger
	} else {
		keycloakClient.logger = logr.Discard()
	}

	// Apply basePath and adminUrl from config
	keycloakClient.authUrl = baseURL + config.basePath
	keycloakClient.baseUrl = keycloakClient.authUrl

	if config.adminUrl != "" {
		keycloakClient.baseUrl = config.adminUrl + config.basePath
	}

	// Determine grant type if not already set by WithPasswordGrant or WithJWTAuth.
	// If only WithClientSecret was used, default to client_credentials grant.
	if keycloakClient.clientCredentials.GrantType == "" {
		if keycloakClient.clientCredentials.ClientSecret != "" {
			keycloakClient.clientCredentials.GrantType = "client_credentials"
		}
	}

	// Validate authentication configuration
	if keycloakClient.clientCredentials.AccessToken == "" &&
		keycloakClient.clientCredentials.GrantType == "" {
		return nil, fmt.Errorf(
			"must specify authentication method: use WithPasswordGrant, WithClientSecret, WithJWTAuth, or WithAccessToken",
		)
	}

	httpClient, err := newHttpClient(
		config.tlsInsecureSkipVerify,
		config.clientTimeout,
		config.caCert,
		config.tlsClientCert,
		config.tlsClientPrivateKey,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create http client: %v", err)
	}

	keycloakClient.httpClient = httpClient

	if keycloakClient.clientCredentials.AccessToken == "" && keycloakClient.initialLogin {
		err = keycloakClient.login(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to perform initial login to Keycloak: %v", err)
		}
	}

	// Initialize service clients using the same generated client
	generatedClient, err := generated.NewClientWithResponses(
		keycloakClient.baseUrl,
		generated.WithHTTPClient(&keycloakDoer{kc: keycloakClient}),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create generated keycloak client: %v", err)
	}

	keycloakClient.Users = &usersClient{client: generatedClient}
	keycloakClient.Realms = &realmClient{client: generatedClient}

	return keycloakClient, nil
}

// getContextLogger returns a logger bound to the context (with trace ID) if available,
// otherwise falls back to the stored logger. This ensures all logs carry trace context.
func (kc *KeycloakClient) getContextLogger(ctx context.Context) logr.Logger {
	var zeroLogger logr.Logger
	if ctx != nil {
		// Try to get logger from context first (this includes trace ID if set)
		if ctxLogger := ctrl.LoggerFrom(ctx); ctxLogger != zeroLogger {
			return ctxLogger
		}
	}
	// Fall back to stored logger
	return kc.logger
}

func (keycloakClient *KeycloakClient) login(ctx context.Context) error {
	logger := keycloakClient.getContextLogger(ctx)

	if !keycloakClient.accessTokenProvided {
		accessTokenUrl := fmt.Sprintf(tokenUrl, keycloakClient.authUrl, keycloakClient.realm)

		accessTokenData, err := keycloakClient.getAuthenticationFormData(ctx)
		if err != nil {
			return err
		}

		logger.V(debugVerbosityLevel).Info("Login request", "request", accessTokenData.Encode())

		accessTokenRequest, err := http.NewRequestWithContext(
			ctx,
			http.MethodPost,
			accessTokenUrl,
			strings.NewReader(accessTokenData.Encode()),
		)
		if err != nil {
			return err
		}

		for header, value := range keycloakClient.additionalHeaders {
			accessTokenRequest.Header.Set(header, value)
		}

		accessTokenRequest.Header.Set("Content-Type", "application/x-www-form-urlencoded")

		if keycloakClient.userAgent != "" {
			accessTokenRequest.Header.Set("User-Agent", keycloakClient.userAgent)
		}

		accessTokenResponse, err := keycloakClient.httpClient.Do(accessTokenRequest)
		if err != nil {
			return err
		}

		if accessTokenResponse.StatusCode != http.StatusOK {
			return fmt.Errorf("error sending POST request to %s: %s", accessTokenUrl, accessTokenResponse.Status)
		}

		defer func() {
			_ = accessTokenResponse.Body.Close()
		}()

		body, _ := io.ReadAll(accessTokenResponse.Body)

		logger.V(debugVerbosityLevel).Info("Login response", "response", string(body))

		var clientCredentials ClientCredentials

		err = json.Unmarshal(body, &clientCredentials)
		if err != nil {
			return err
		}

		keycloakClient.mu.Lock()
		keycloakClient.clientCredentials.AccessToken = clientCredentials.AccessToken
		keycloakClient.clientCredentials.RefreshToken = clientCredentials.RefreshToken
		keycloakClient.clientCredentials.TokenType = clientCredentials.TokenType
		keycloakClient.mu.Unlock()
	} else {
		logger.V(debugVerbosityLevel).Info(
			"Using provided access_token",
			"access_token",
			keycloakClient.clientCredentials.AccessToken,
		)
	}

	return nil
}

func (keycloakClient *KeycloakClient) Refresh(ctx context.Context) error {
	logger := keycloakClient.getContextLogger(ctx)

	if keycloakClient.accessTokenProvided {
		// If an access_token was provided, we skip refresh
		return nil
	}

	refreshTokenUrl := fmt.Sprintf(tokenUrl, keycloakClient.authUrl, keycloakClient.realm)

	refreshTokenData, err := keycloakClient.getAuthenticationFormData(ctx)
	if err != nil {
		return err
	}

	logger.V(debugVerbosityLevel).Info("Refresh request", "request", refreshTokenData.Encode())

	refreshTokenRequest, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		refreshTokenUrl,
		strings.NewReader(refreshTokenData.Encode()),
	)
	if err != nil {
		return err
	}

	for header, value := range keycloakClient.additionalHeaders {
		refreshTokenRequest.Header.Set(header, value)
	}

	refreshTokenRequest.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	if keycloakClient.userAgent != "" {
		refreshTokenRequest.Header.Set("User-Agent", keycloakClient.userAgent)
	}

	refreshTokenResponse, err := keycloakClient.httpClient.Do(refreshTokenRequest)
	if err != nil {
		return err
	}

	defer func() {
		_ = refreshTokenResponse.Body.Close()
	}()

	body, _ := io.ReadAll(refreshTokenResponse.Body)

	logger.V(debugVerbosityLevel).Info("Refresh response", "response", string(body))

	// Handle 401 "User or client no longer has role permissions for client key"
	// until I better understand why that happens in the first place
	if refreshTokenResponse.StatusCode == http.StatusBadRequest {
		logger.V(debugVerbosityLevel).Info("Unexpected 400, attempting to log in again")

		return keycloakClient.login(ctx)
	}

	var clientCredentials ClientCredentials

	err = json.Unmarshal(body, &clientCredentials)
	if err != nil {
		return err
	}

	keycloakClient.mu.Lock()
	defer keycloakClient.mu.Unlock()

	keycloakClient.clientCredentials.AccessToken = clientCredentials.AccessToken
	keycloakClient.clientCredentials.RefreshToken = clientCredentials.RefreshToken
	keycloakClient.clientCredentials.TokenType = clientCredentials.TokenType

	return nil
}

func (keycloakClient *KeycloakClient) getAuthenticationFormData(
	ctx context.Context,
) (url.Values, error) {
	authenticationFormData := url.Values{}
	authenticationFormData.Set("client_id", keycloakClient.clientCredentials.ClientId)
	authenticationFormData.Set("grant_type", keycloakClient.clientCredentials.GrantType)

	switch keycloakClient.clientCredentials.GrantType {
	case "password":
		authenticationFormData.Set("username", keycloakClient.clientCredentials.Username)
		authenticationFormData.Set("password", keycloakClient.clientCredentials.Password)

		if keycloakClient.clientCredentials.ClientSecret != "" {
			authenticationFormData.Set("client_secret", keycloakClient.clientCredentials.ClientSecret)
		}

	case grantTypeClientCredentials:
		if len(keycloakClient.clientCredentials.JWTToken) > 0 ||
			len(keycloakClient.clientCredentials.JWTTokenFile) > 0 ||
			len(keycloakClient.clientCredentials.JWTSigningKey) > 0 {
			var signedJWT string

			var err error

			signedJWT = keycloakClient.clientCredentials.JWTToken
			if len(signedJWT) == 0 && len(keycloakClient.clientCredentials.JWTTokenFile) > 0 {
				var content []byte

				content, err = os.ReadFile(keycloakClient.clientCredentials.JWTTokenFile)
				if err != nil {
					return nil, fmt.Errorf("failed to read JWT token from file: %v", err)
				}

				signedJWT = strings.TrimSpace(string(content))
			}

			if len(signedJWT) == 0 {
				signedJWT, err = keycloakClient.newSignedJWT(
					ctx,
					fmt.Sprintf(issuerUrl, keycloakClient.baseUrl, keycloakClient.realm),
					keycloakClient.clientCredentials.ClientId,
					keycloakClient.clientCredentials.JWTSigningAlg,
					keycloakClient.clientCredentials.JWTSigningKey,
				)
				if err != nil {
					return nil, fmt.Errorf("failed to create signed JWT: %v", err)
				}
			}

			authenticationFormData.Set("client_assertion_type", "urn:ietf:params:oauth:client-assertion-type:jwt-bearer")
			authenticationFormData.Set("client_assertion", signedJWT)
		} else {
			authenticationFormData.Set("client_secret", keycloakClient.clientCredentials.ClientSecret)
		}
	}

	return authenticationFormData, nil
}

func (keycloakClient *KeycloakClient) addRequestHeaders(request *http.Request) {
	keycloakClient.mu.Lock()
	tokenType := keycloakClient.clientCredentials.TokenType
	accessToken := keycloakClient.clientCredentials.AccessToken
	keycloakClient.mu.Unlock()

	for header, value := range keycloakClient.additionalHeaders {
		request.Header.Set(header, value)
	}

	request.Header.Set("Authorization", fmt.Sprintf("%s %s", tokenType, accessToken))
	request.Header.Set("Accept", "application/json")

	if keycloakClient.userAgent != "" {
		request.Header.Set("User-Agent", keycloakClient.userAgent)
	}

	if request.Header.Get("Content-type") == "" &&
		(request.Method == http.MethodPost ||
			request.Method == http.MethodPut ||
			request.Method == http.MethodDelete) {
		request.Header.Set("Content-type", "application/json")
	}
}

// keycloakDoer implements generated.HttpRequestDoer so the oapi-generated
// client reuses KeycloakClient's configured httpClient (TLS, retries, timeouts)
// and adds lazy-login + 401/403 token-refresh on top.
type keycloakDoer struct {
	kc *KeycloakClient
}

func (d *keycloakDoer) Do(req *http.Request) (*http.Response, error) {
	ctx := req.Context()
	logger := d.kc.getContextLogger(ctx)

	// Lazy-login: runs at most once, guarded by double-checked lock.
	d.kc.mu.Lock()
	if !d.kc.initialLogin {
		d.kc.initialLogin = true
		d.kc.mu.Unlock()

		if err := d.kc.login(ctx); err != nil {
			return nil, fmt.Errorf("error logging in: %s", err)
		}
	} else {
		d.kc.mu.Unlock()
	}

	logger.V(debugVerbosityLevel).Info("Sending request",
		"method", req.Method,
		"path", req.URL.Path,
	)

	// Snapshot the access token before the first request. Used below to
	// detect whether another goroutine already refreshed after a 401.
	d.kc.mu.Lock()
	tokenBefore := d.kc.clientCredentials.AccessToken
	d.kc.mu.Unlock()

	d.kc.addRequestHeaders(req)

	resp, err := d.kc.httpClient.Do(req)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden {
		logger.V(debugVerbosityLevel).Info("Got unexpected response, attempting refresh",
			"status", resp.Status,
		)

		_ = resp.Body.Close()

		// Single-flight: if another goroutine already refreshed the token
		// since we took our snapshot, skip the Refresh call and just retry
		// with the new token.
		d.kc.mu.Lock()
		needsRefresh := d.kc.clientCredentials.AccessToken == tokenBefore
		d.kc.mu.Unlock()

		if needsRefresh {
			if err := d.kc.Refresh(req.Context()); err != nil {
				return nil, fmt.Errorf("error refreshing credentials: %s", err)
			}
		}

		// Restore the request body for the retry — the first Do consumed it.
		if req.GetBody != nil {
			req.Body, err = req.GetBody()
			if err != nil {
				return nil, fmt.Errorf("error restoring request body for retry: %s", err)
			}
		}

		d.kc.addRequestHeaders(req)

		resp, err = d.kc.httpClient.Do(req)
		if err != nil {
			return nil, err
		}
	}

	return resp, nil
}

func RetryPolicy(resp *resty.Response, err error) bool {
	if err != nil {
		return true
	}

	statusCode := resp.RawResponse.StatusCode

	// 429 Too Many Requests is recoverable. Sometimes the server puts
	// a Retry-After response header to indicate when the server is
	// available to start processing request from client.
	if statusCode == http.StatusTooManyRequests {
		return true
	}

	// Check the response code. We retry on 500-range responses to allow
	// the server time to recover, as 500's are typically not permanent
	// errors and may relate to outages on the server side. This will catch
	// invalid response codes as well, like 0 and 999.
	if statusCode == 0 || (statusCode >= 500 && statusCode != http.StatusNotImplemented) {
		return true
	}

	return false
}

func newHttpClient(
	tlsInsecureSkipVerify bool,
	clientTimeout int,
	caCert string,
	tlsClientCert string,
	tlsClientPrivateKey string,
) (*http.Client, error) {
	cookieJar, err := cookiejar.New(&cookiejar.Options{
		PublicSuffixList: publicsuffix.List,
	})
	if err != nil {
		return nil, err
	}

	transport := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: tlsInsecureSkipVerify},
		Proxy:           http.ProxyFromEnvironment,
	}
	transport.MaxIdleConnsPerHost = 100

	if caCert != "" {
		caCertPool := x509.NewCertPool()
		caCertPool.AppendCertsFromPEM([]byte(caCert))
		transport.TLSClientConfig.RootCAs = caCertPool
	}

	if tlsClientCert != "" && tlsClientPrivateKey != "" {
		clientKeyPairCert, err := tls.X509KeyPair([]byte(tlsClientCert), []byte(tlsClientPrivateKey))
		if err != nil {
			return nil, err
		}

		transport.TLSClientConfig.Certificates = []tls.Certificate{clientKeyPairCert}
	}

	restyClient := resty.New().
		SetTransport(transport).
		SetTimeout(time.Second * time.Duration(clientTimeout)).
		SetCookieJar(cookieJar).
		SetRetryCount(5).
		SetRetryWaitTime(time.Second).
		SetRetryMaxWaitTime(time.Second * 60).
		AddRetryCondition(RetryPolicy)

	httpClient := restyClient.GetClient()

	return httpClient, nil
}

func (keycloakClient *KeycloakClient) newSignedJWT(
	ctx context.Context,
	issuerURL, clientId, alg, jwtSigningKey string,
) (string, error) {
	logger := keycloakClient.getContextLogger(ctx)

	// Create the Claims
	jti, err := uuid.NewUUID()
	if err != nil {
		return "", fmt.Errorf("failed to generate JWT ID: %v", err)
	}

	claims := jwt.MapClaims{
		"jti": jti,
		"iss": clientId,
		"sub": clientId,
		"aud": issuerURL,
		"exp": jwt.NewNumericDate(time.Now().Add(time.Second * 60)),
		"iat": jwt.NewNumericDate(time.Now()),
	}

	signingMethod := jwt.GetSigningMethod(alg)
	if signingMethod == nil {
		return "", fmt.Errorf("unsupported signing method: %s", alg)
	}

	// Create the token
	token := jwt.NewWithClaims(jwt.GetSigningMethod(alg), claims)

	var key any
	if _, isRsa := signingMethod.(*jwt.SigningMethodRSA); isRsa {
		key, err = jwt.ParseRSAPrivateKeyFromPEM([]byte(jwtSigningKey))
	} else if _, isEcdsa := signingMethod.(*jwt.SigningMethodECDSA); isEcdsa {
		key, err = jwt.ParseECPrivateKeyFromPEM([]byte(jwtSigningKey))
	} else if _, isEd25519 := signingMethod.(*jwt.SigningMethodEd25519); isEd25519 {
		key, err = jwt.ParseEdPrivateKeyFromPEM([]byte(jwtSigningKey))
	} else {
		err = fmt.Errorf("unsupported signing method: %s", signingMethod)
	}

	if err != nil {
		return "", err
	}

	tokenString, err := token.SignedString(key)
	if err != nil {
		return "", err
	}

	logger.V(debugVerbosityLevel).Info("Generated client_assertion", "jti", jti)

	return tokenString, nil
}

// NewSignedJWT creates a signed JWT token for authentication.
// Deprecated: This function is deprecated and kept for backward compatibility.
// Use KeycloakClient.newSignedJWT instead.
func NewSignedJWT(
	ctx context.Context,
	issuerURL, clientId, alg, jwtSigningKey string,
) (string, error) {
	// Create a discarded client to use the method (for backward compatibility only)
	tempClient := &KeycloakClient{logger: logr.Discard()}
	return tempClient.newSignedJWT(ctx, issuerURL, clientId, alg, jwtSigningKey)
}

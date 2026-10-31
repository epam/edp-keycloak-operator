package keycloakapi

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"os"
	"reflect"
	"strings"
	"sync"
	"time"

	"github.com/go-logr/logr"
	"github.com/google/uuid"
	ctrl "sigs.k8s.io/controller-runtime"

	"github.com/golang-jwt/jwt/v5"

	"golang.org/x/net/idna"
	"golang.org/x/net/publicsuffix"

	"github.com/go-resty/resty/v2"

	"github.com/epam/edp-keycloak-operator/pkg/client/keycloakapi/generated"
)

type KeycloakClient struct {
	mu                  sync.Mutex
	baseUrl             string
	authUrl             string
	realm               string
	clientCredentials   *ClientCredentials
	restyClient         *resty.Client
	initialLogin        bool
	userAgent           string
	additionalHeaders   map[string]string
	redHatSSO           bool
	accessTokenProvided bool
	logger              logr.Logger
	Users               UsersClient
	Realms              RealmClient
	Groups              GroupsClient
	Roles               RolesClient
	Clients             ClientsClient
	Organizations       OrganizationsClient
	IdentityProviders   IdentityProvidersClient
	Authorization       AuthorizationClient
	ClientScopes        ClientScopesClient
	RealmComponents     RealmComponentsClient
	AuthFlows           AuthFlowsClient
	Server              ServerInfoClient
	Events              EventsClient
	Sessions            SessionsClient
	ClientPolicies      ClientPoliciesClient
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
	ExpiresIn     int    `json:"expires_in"`
}

const (
	issuerUrl = "%s/realms/%s"
	tokenUrl  = "%s/realms/%s/protocol/openid-connect/token"

	grantTypeClientCredentials = "client_credentials"

	// debugVerbosityLevel is the verbosity level for debug-level logs
	debugVerbosityLevel = 1

	// Values of http.DefaultTransport.
	dialTimeout         = 30 * time.Second
	dialKeepAlive       = 30 * time.Second
	tlsHandshakeTimeout = 10 * time.Second
	idleConnTimeout     = 90 * time.Second
)

// clientConfig holds configuration parameters that need to be captured before creating the client
type clientConfig struct {
	baseURL               string
	basePath              string
	adminUrl              string
	clientTimeout         int
	caCert                string
	tlsInsecureSkipVerify bool
	tlsServerName         string
	tlsClientCert         string
	tlsClientPrivateKey   string
	logger                logr.Logger
	retryWaitTime         time.Duration
	retryMaxWaitTime      time.Duration
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

// WithRetryWaitTime sets the initial wait time between retries (default: 1s).
// Resty uses exponential back-off; the actual delay is capped by WithRetryMaxWaitTime.
func WithRetryWaitTime(d time.Duration) ClientOption {
	return func(c *KeycloakClient, cfg *clientConfig) {
		cfg.retryWaitTime = d
	}
}

// WithRetryMaxWaitTime sets the maximum wait time between retries (default: 10s).
func WithRetryMaxWaitTime(d time.Duration) ClientOption {
	return func(c *KeycloakClient, cfg *clientConfig) {
		cfg.retryMaxWaitTime = d
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

// WithTLSServerName sets the certificate verification name for the host and port of the base url.
// A host name is also sent as SNI; an IP address is not (RFC 6066). Other addresses, including an
// https:// proxy, use the dialed host name; through a proxy the name applies to every target.
// Empty uses the host of each request.
func WithTLSServerName(serverName string) ClientOption {
	return func(c *KeycloakClient, cfg *clientConfig) {
		cfg.tlsServerName = serverName
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
//   - WithTLSServerName: Verify the server certificate against a name other than the url host
//   - WithTLSClientCert: Set client certificate for mutual TLS
//   - WithUserAgent: Set custom User-Agent
//   - WithRedHatSSO: Enable Red Hat SSO mode
//   - WithAdditionalHeaders: Add custom headers
//   - WithLogger: Set a custom logger (default: uses context logger or discard)
//
// Trailing slashes in baseURL, the admin URL and the base path are ignored.
// The client follows no HTTP redirect: a 3xx response fails the request with ErrRedirectRefused.
func NewKeycloakClient(ctx context.Context, baseURL, clientId string, opts ...ClientOption) (*KeycloakClient, error) {
	baseURL = strings.TrimRight(baseURL, "/")
	if baseURL == "" {
		return nil, fmt.Errorf("url is required")
	}

	if clientId == "" {
		return nil, fmt.Errorf("clientId is required")
	}

	// Initialize configuration with defaults
	config := &clientConfig{
		baseURL:               baseURL,
		clientTimeout:         60, // default timeout in seconds
		tlsInsecureSkipVerify: false,
		retryWaitTime:         time.Second,
		retryMaxWaitTime:      10 * time.Second,
	}

	// Initialize client with defaults
	keycloakClient := &KeycloakClient{
		authUrl: baseURL,
		baseUrl: baseURL,
		realm:   MasterRealm, // default to "master"
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

	config.basePath = strings.TrimRight(config.basePath, "/")
	config.adminUrl = strings.TrimRight(config.adminUrl, "/")

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

	restyClient, err := newRestyClient(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create http client: %w", err)
	}

	keycloakClient.restyClient = restyClient

	if keycloakClient.clientCredentials.AccessToken == "" && keycloakClient.initialLogin {
		err = keycloakClient.login(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to perform initial login to Keycloak: %w", err)
		}
	}

	// Initialize service clients using the same generated client
	generatedClient, err := generated.NewClientWithResponses(
		keycloakClient.baseUrl,
		generated.WithHTTPClient(&keycloakDoer{kc: keycloakClient}),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create generated keycloak client: %w", err)
	}

	keycloakClient.Users = &usersClient{client: generatedClient}
	keycloakClient.Realms = &realmClient{client: generatedClient}
	keycloakClient.Groups = &groupsClient{client: generatedClient}
	keycloakClient.Roles = &rolesClient{client: generatedClient}
	keycloakClient.Clients = &clientsClient{client: generatedClient}
	keycloakClient.Organizations = &organizationsClient{client: generatedClient}
	keycloakClient.IdentityProviders = &identityProvidersClient{client: generatedClient}
	keycloakClient.Authorization = &authorizationClient{client: generatedClient, kc: keycloakClient}
	keycloakClient.ClientScopes = &clientScopesClient{client: generatedClient}
	keycloakClient.RealmComponents = &realmComponentsClient{client: generatedClient}
	keycloakClient.AuthFlows = &authFlowsClient{client: generatedClient}
	keycloakClient.Server = keycloakClient // KeycloakClient implements ServerInfoClient
	keycloakClient.Events = &eventsClient{client: generatedClient}
	keycloakClient.Sessions = &sessionsClient{client: generatedClient}
	keycloakClient.ClientPolicies = &clientPoliciesClient{client: generatedClient}

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

		logger.V(debugVerbosityLevel).Info("Login request", "content_length", len(accessTokenData.Encode()))

		var clientCredentials ClientCredentials

		req := keycloakClient.restyClient.R().
			SetContext(ctx).
			SetFormDataFromValues(accessTokenData).
			SetHeaders(keycloakClient.additionalHeaders).
			SetHeader("Content-Type", "application/x-www-form-urlencoded").
			SetResult(&clientCredentials)

		if keycloakClient.userAgent != "" {
			req.SetHeader("User-Agent", keycloakClient.userAgent)
		}

		resp, err := req.Post(accessTokenUrl)
		if err != nil {
			return err
		}

		if err = refuseRedirect(resp.RawResponse); err != nil {
			return err
		}

		if resp.IsError() {
			return fmt.Errorf("error sending POST request to %s: %s", accessTokenUrl, resp.Status())
		}

		logger.V(debugVerbosityLevel).Info("Login response", "expires_in", clientCredentials.ExpiresIn)

		keycloakClient.mu.Lock()
		keycloakClient.clientCredentials.AccessToken = clientCredentials.AccessToken
		keycloakClient.clientCredentials.RefreshToken = clientCredentials.RefreshToken
		keycloakClient.clientCredentials.TokenType = clientCredentials.TokenType
		keycloakClient.mu.Unlock()
	} else {
		logger.V(debugVerbosityLevel).Info("Using provided access_token")
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

	logger.V(debugVerbosityLevel).Info("Refresh request", "content_length", len(refreshTokenData.Encode()))

	req := keycloakClient.restyClient.R().
		SetContext(ctx).
		SetFormDataFromValues(refreshTokenData).
		SetHeaders(keycloakClient.additionalHeaders).
		SetHeader("Content-Type", "application/x-www-form-urlencoded")

	if keycloakClient.userAgent != "" {
		req.SetHeader("User-Agent", keycloakClient.userAgent)
	}

	resp, err := req.Post(refreshTokenUrl)
	if err != nil {
		return err
	}

	if err = refuseRedirect(resp.RawResponse); err != nil {
		return err
	}

	// Handle 401 "User or client no longer has role permissions for client key"
	// until I better understand why that happens in the first place
	if resp.StatusCode() == http.StatusBadRequest {
		logger.V(debugVerbosityLevel).Info("Unexpected 400, attempting to log in again")

		return keycloakClient.login(ctx)
	}

	var clientCredentials ClientCredentials

	err = json.Unmarshal(resp.Body(), &clientCredentials)
	if err != nil {
		return err
	}

	logger.V(debugVerbosityLevel).Info("Refresh response", "expires_in", clientCredentials.ExpiresIn)

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
			return nil, fmt.Errorf("error logging in: %w", err)
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

	resp, err := d.executeViaResty(req)
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
				return nil, fmt.Errorf("error refreshing credentials: %w", err)
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

		resp, err = d.executeViaResty(req)
		if err != nil {
			return nil, err
		}
	}

	return resp, nil
}

// executeViaResty forwards an *http.Request through the configured resty.Client
// so that resty's retry middleware (SetRetryCount + RetryPolicy) is actually
// invoked. Calling restyClient.GetClient().Do bypasses retries entirely.
func (d *keycloakDoer) executeViaResty(req *http.Request) (*http.Response, error) {
	var bodyBytes []byte

	if req.Body != nil {
		b, err := io.ReadAll(req.Body)
		if err != nil {
			return nil, fmt.Errorf("reading request body: %w", err)
		}

		_ = req.Body.Close()
		bodyBytes = b
	}

	rr := d.kc.restyClient.R().
		SetContext(req.Context()).
		SetDoNotParseResponse(true)

	for k, vs := range req.Header {
		for _, v := range vs {
			rr.SetHeader(k, v)
		}
	}

	if bodyBytes != nil {
		rr.SetBody(bodyBytes)
	}

	resp, err := rr.Execute(req.Method, req.URL.String())
	if err != nil {
		if resp != nil && resp.RawResponse != nil {
			return resp.RawResponse, err
		}

		return nil, err
	}

	if err = refuseRedirect(resp.RawResponse); err != nil {
		_ = resp.RawResponse.Body.Close()

		return nil, err
	}

	return resp.RawResponse, nil
}

// ErrRedirectRefused is wrapped by the error for every 3xx response. The client follows no redirect.
var ErrRedirectRefused = errors.New("redirect refused")

// refuseRedirect returns an ErrRedirectRefused error for a 3xx response.
// Call it on the response of every send site: login, Refresh and executeViaResty.
// The redirect policy of the resty client only stops at the first 3xx.
func refuseRedirect(resp *http.Response) error {
	if resp.StatusCode < 300 || resp.StatusCode > 399 {
		return nil
	}

	u := *resp.Request.URL
	u.User, u.RawQuery, u.ForceQuery, u.Fragment, u.RawFragment = nil, "", false, "", ""

	return fmt.Errorf("%w: %s %s answered %d; set spec.url to the address that serves Keycloak without a redirect",
		ErrRedirectRefused, resp.Request.Method, u.String(), resp.StatusCode)
}

func RetryPolicy(resp *resty.Response, err error) bool {
	if err != nil {
		return !isPermanentTLSError(err)
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

// permanentTLSAlerts are the alert codes (RFC 8446, section 6) a peer sends when it rejects the
// handshake on configuration: certificates, protocol versions, cipher suites or the server name.
var permanentTLSAlerts = map[uint8]struct{}{
	40:  {}, // handshake_failure
	42:  {}, // bad_certificate
	43:  {}, // unsupported_certificate
	44:  {}, // certificate_revoked
	45:  {}, // certificate_expired
	46:  {}, // certificate_unknown
	48:  {}, // unknown_ca
	49:  {}, // access_denied
	70:  {}, // protocol_version
	71:  {}, // insufficient_security
	112: {}, // unrecognized_name
	116: {}, // certificate_required
}

// isPermanentTLSError reports a TLS failure that repeats with the same configuration: failed certificate
// verification, a peer that does not speak TLS, or a permanentTLSAlerts alert.
func isPermanentTLSError(err error) bool {
	var (
		certErr   *tls.CertificateVerificationError
		recordErr tls.RecordHeaderError
	)

	if errors.As(err, &certErr) || errors.As(err, &recordErr) || errors.Is(err, http.ErrSchemeMismatch) {
		return true
	}

	code, ok := remoteTLSAlert(err)
	if !ok {
		return false
	}

	_, permanent := permanentTLSAlerts[code]

	return permanent
}

// remoteTLSAlert returns the code of a TLS alert received from the peer anywhere in the chain of err.
// crypto/tls reports it over TCP as *net.OpError{Op: "remote error"} wrapping an unexported uint8 type;
// tls.AlertError is used for QUIC only.
func remoteTLSAlert(err error) (uint8, bool) {
	for err != nil {
		if opErr, ok := err.(*net.OpError); ok && opErr.Op == "remote error" {
			if v := reflect.ValueOf(opErr.Err); v.IsValid() && v.Kind() == reflect.Uint8 {
				return uint8(v.Uint()), true
			}
		}

		if multi, ok := err.(interface{ Unwrap() []error }); ok {
			for _, inner := range multi.Unwrap() {
				if code, ok := remoteTLSAlert(inner); ok {
					return code, true
				}
			}

			return 0, false
		}

		err = errors.Unwrap(err)
	}

	return 0, false
}

// newRestyClient builds the HTTP client of KeycloakClient.
// The redirect policy stops at the first 3xx; every send site (login, Refresh, executeViaResty)
// calls refuseRedirect on the response.
func newRestyClient(config *clientConfig) (*resty.Client, error) {
	cookieJar, err := cookiejar.New(&cookiejar.Options{
		PublicSuffixList: publicsuffix.List,
	})
	if err != nil {
		return nil, err
	}

	transport, err := newHTTPTransport(config)
	if err != nil {
		return nil, err
	}

	restyClient := resty.New().
		SetTransport(transport).
		SetTimeout(time.Second * time.Duration(config.clientTimeout)).
		SetCookieJar(cookieJar).
		SetRedirectPolicy(resty.RedirectPolicyFunc(func(*http.Request, []*http.Request) error {
			return http.ErrUseLastResponse
		})).
		SetRetryCount(5).
		SetRetryWaitTime(config.retryWaitTime).
		SetRetryMaxWaitTime(config.retryMaxWaitTime).
		AddRetryCondition(RetryPolicy)

	return restyClient, nil
}

func newHTTPTransport(config *clientConfig) (*http.Transport, error) {
	dialer := &net.Dialer{Timeout: dialTimeout, KeepAlive: dialKeepAlive}

	transport := &http.Transport{
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: config.tlsInsecureSkipVerify,
			// Applies to the TLS session with the target inside a proxy tunnel.
			ServerName: config.tlsServerName,
		},
		Proxy:               http.ProxyFromEnvironment,
		DialContext:         dialer.DialContext,
		TLSHandshakeTimeout: tlsHandshakeTimeout,
		MaxIdleConnsPerHost: 100,
		IdleConnTimeout:     idleConnTimeout,
	}

	if config.caCert != "" {
		caCertPool := x509.NewCertPool()
		if !caCertPool.AppendCertsFromPEM([]byte(config.caCert)) && config.verifiesServerCertificate() {
			return nil, errors.New("CA certificate contains no PEM-encoded certificate")
		}

		transport.TLSClientConfig.RootCAs = caCertPool
	}

	if (config.tlsClientCert == "") != (config.tlsClientPrivateKey == "") {
		return nil, errors.New("TLS client certificate and private key must be set together")
	}

	if config.tlsClientCert != "" {
		clientKeyPairCert, err := tls.X509KeyPair([]byte(config.tlsClientCert), []byte(config.tlsClientPrivateKey))
		if err != nil {
			return nil, err
		}

		transport.TLSClientConfig.Certificates = []tls.Certificate{clientKeyPairCert}
	}

	if config.tlsServerName != "" {
		originAddr, err := hostPortFromURL(config.baseURL)
		if err != nil {
			return nil, err
		}

		transport.DialTLSContext = serverNameTLSDialer(
			dialer, transport.TLSClientConfig, config.tlsServerName, originAddr, tlsHandshakeTimeout,
		)
	}

	return transport, nil
}

// verifiesServerCertificate reports whether verification is on and the base url or the admin url uses https.
func (c *clientConfig) verifiesServerCertificate() bool {
	if c.tlsInsecureSkipVerify {
		return false
	}

	for _, raw := range []string{c.baseURL, c.adminUrl} {
		if u, err := url.Parse(raw); err == nil && strings.EqualFold(u.Scheme, "https") {
			return true
		}
	}

	return false
}

// hostPortFromURL returns the address http.Transport dials: the IDNA ASCII host and the scheme default port.
func hostPortFromURL(rawURL string) (string, error) {
	u, err := url.Parse(rawURL)
	if err != nil {
		return "", fmt.Errorf("unable to parse url: %w", err)
	}

	host := u.Hostname()
	if ascii, err := idna.Lookup.ToASCII(host); err == nil {
		host = ascii
	}

	port := u.Port()
	if port == "" {
		port = "443"
		if strings.EqualFold(u.Scheme, "http") {
			port = "80"
		}
	}

	return net.JoinHostPort(host, port), nil
}

// serverNameTLSDialer uses serverName for originAddr and the dialed host for any other address.
// It bounds the handshake itself: http.Transport skips TLSHandshakeTimeout for DialTLSContext.
func serverNameTLSDialer(
	dialer *net.Dialer,
	base *tls.Config,
	serverName, originAddr string,
	handshakeTimeout time.Duration,
) func(ctx context.Context, network, addr string) (net.Conn, error) {
	return func(ctx context.Context, network, addr string) (net.Conn, error) {
		cfg := base.Clone()
		cfg.ServerName = serverName

		if !strings.EqualFold(addr, originAddr) {
			host, _, err := net.SplitHostPort(addr)
			if err != nil {
				return nil, err
			}

			cfg.ServerName = host
		}

		conn, err := dialer.DialContext(ctx, network, addr)
		if err != nil {
			return nil, err
		}

		handshakeCtx, cancel := context.WithTimeout(ctx, handshakeTimeout)
		defer cancel()

		tlsConn := tls.Client(conn, cfg)
		if err := tlsConn.HandshakeContext(handshakeCtx); err != nil {
			_ = conn.Close()

			return nil, err
		}

		return tlsConn, nil
	}
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

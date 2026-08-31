package destination

import (
	"context"
	"encoding/json"
	"net/http"
	"net/url"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// configProperty is the slice of ConfigPropertyRepresentation this test reads. secret is
// Keycloak's own declaration that the property holds a secret value.
type configProperty struct {
	Name   string `json:"name"`
	Secret bool   `json:"secret"`
}

type componentType struct {
	ID         string           `json:"id"`
	Properties []configProperty `json:"properties"`
}

type serverInfoRepresentation struct {
	ComponentTypes map[string][]componentType `json:"componentTypes"`
}

// TestLists_MatchKeycloakModel verifies the hand-pinned key lists against the provider
// metadata of a running Keycloak (GET /admin/serverinfo, componentTypes):
//
//   - a property Keycloak marks secret:true must be in credentialKeys, or a legitimate
//     secret reference in it is denied;
//   - a property with a destination-shaped name must be in destinationKeys, or an address
//     in it is never checked.
//
// The destination-shape suffix match is a heuristic; it informs this test only and never
// runs at reconcile time. Identity provider config keys (tokenUrl, ...) are exposed by no
// admin endpoint and stay human-curated.
//
// Runs only against the pinned Keycloak (TEST_KEYCLOAK_URL, make start-keycloak). A
// KEYCLOAK_TEST_VERSION bump that introduces new keys fails here until they are classified.
func TestLists_MatchKeycloakModel(t *testing.T) {
	baseURL := os.Getenv("TEST_KEYCLOAK_URL")
	if baseURL == "" {
		t.Skip("TEST_KEYCLOAK_URL is not set")
	}

	info := fetchServerInfo(t, baseURL, adminToken(t, baseURL))

	require.NotEmpty(t, info.ComponentTypes, "serverinfo returned no component types")

	var missingCredentials, missingDestinations []string

	for spi, types := range info.ComponentTypes {
		for _, ct := range types {
			for _, prop := range ct.Properties {
				key := strings.ToLower(prop.Name)
				id := spi + "/" + ct.ID + "." + prop.Name

				if prop.Secret {
					if _, ok := credentialKeys[key]; !ok {
						missingCredentials = append(missingCredentials, id)
					}

					continue
				}

				if _, exempt := nonDestinationExceptions[key]; exempt {
					continue
				}

				if isDestinationShaped(key) {
					if _, ok := destinationKeys[key]; !ok {
						missingDestinations = append(missingDestinations, id)
					}
				}
			}
		}
	}

	assert.Empty(t, missingCredentials,
		"Keycloak marks these properties secret: true; add them to credentialKeys or a valid secret ref in them is denied")
	assert.Empty(t, missingDestinations,
		"these properties carry destination-shaped names; classify them in destinationKeys")
}

// nonDestinationExceptions are destination-shaped property names reviewed to hold no
// dial-out address. Each entry records why it is exempt.
var nonDestinationExceptions = map[string]string{
	"requirevalidurl":             "boolean flag of the uri validator",
	"allow-ipv4-loopback-address": "boolean flag of secure-redirect-uris-enforcer",
	"allow-ipv6-loopback-address": "boolean flag of secure-redirect-uris-enforcer",
	"trusted-hosts":               "inbound ACL for client registration; nothing is dialed",
}

// isDestinationShaped reports whether a lowercased property name looks like it holds an
// address. Test-time heuristic only.
func isDestinationShaped(key string) bool {
	for _, suffix := range []string{"url", "uri", "host", "hosts", "address", "endpoint"} {
		if strings.HasSuffix(key, suffix) {
			return true
		}
	}

	return false
}

func httpClient() *http.Client {
	return &http.Client{Timeout: 30 * time.Second}
}

// adminToken logs in with the credentials of make start-keycloak.
func adminToken(t *testing.T, baseURL string) string {
	t.Helper()

	form := url.Values{
		"grant_type": {"password"},
		"client_id":  {"admin-cli"},
		"username":   {"admin"},
		"password":   {"admin"},
	}

	req, err := http.NewRequestWithContext(context.Background(), http.MethodPost,
		baseURL+"/realms/master/protocol/openid-connect/token", strings.NewReader(form.Encode()))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := httpClient().Do(req)
	require.NoError(t, err)

	defer func() {
		require.NoError(t, resp.Body.Close())
	}()

	require.Equal(t, http.StatusOK, resp.StatusCode)

	var body struct {
		AccessToken string `json:"access_token"`
	}

	require.NoError(t, json.NewDecoder(resp.Body).Decode(&body))
	require.NotEmpty(t, body.AccessToken)

	return body.AccessToken
}

// fetchServerInfo reads /admin/serverinfo, which is absent from the published OpenAPI spec
// and therefore from the generated client.
func fetchServerInfo(t *testing.T, baseURL, token string) serverInfoRepresentation {
	t.Helper()

	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet,
		baseURL+"/admin/serverinfo", nil)
	require.NoError(t, err)
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := httpClient().Do(req)
	require.NoError(t, err)

	defer func() {
		require.NoError(t, resp.Body.Close())
	}()

	require.Equal(t, http.StatusOK, resp.StatusCode)

	var info serverInfoRepresentation

	require.NoError(t, json.NewDecoder(resp.Body).Decode(&info))

	return info
}

package keycloakapi

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

type SystemInfo struct {
	ServerVersion string `json:"version"`
}

type ComponentType struct {
	Id string `json:"id"`
}

type ProviderType struct {
	Internal  bool                `json:"internal"`
	Providers map[string]Provider `json:"providers"`
}

type Provider struct {
}

type Theme struct {
	Name    string   `json:"name"`
	Locales []string `json:"locales,omitempty"`
}

type ServerFeature struct {
	Name    string `json:"name"`
	Enabled bool   `json:"enabled"`
}

type ServerInfo struct {
	SystemInfo     SystemInfo                 `json:"systemInfo"`
	ComponentTypes map[string][]ComponentType `json:"componentTypes"`
	ProviderTypes  map[string]ProviderType    `json:"providers"`
	Themes         map[string][]Theme         `json:"themes"`
	Features       []ServerFeature            `json:"features"`
}

func (keycloakClient *KeycloakClient) GetServerInfo(ctx context.Context) (*ServerInfo, error) {
	serverInfoUrl := keycloakClient.baseUrl + "/admin/serverinfo"

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, serverInfoUrl, nil)
	if err != nil {
		return nil, err
	}

	resp, err := (&keycloakDoer{kc: keycloakClient}).Do(req)
	if err != nil {
		return nil, err
	}

	defer func() {
		_ = resp.Body.Close()
	}()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		apiErr := parseKeycloakError(resp.StatusCode, body)
		apiErr.HTTPResponse = resp

		return nil, apiErr
	}

	var serverInfo ServerInfo
	if err := json.Unmarshal(body, &serverInfo); err != nil {
		return nil, err
	}

	return &serverInfo, nil
}

// FeatureFlagEnabled checks if a specific feature flag is enabled on the Keycloak server.
func (keycloakClient *KeycloakClient) FeatureFlagEnabled(ctx context.Context, featureFlag string) (bool, error) {
	serverInfo, err := keycloakClient.GetServerInfo(ctx)
	if err != nil {
		return false, fmt.Errorf("failed to get server info: %w", err)
	}

	for _, feature := range serverInfo.Features {
		if feature.Name == featureFlag {
			return feature.Enabled, nil
		}
	}

	return false, nil
}

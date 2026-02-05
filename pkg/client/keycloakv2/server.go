package keycloakv2

import (
	"context"
	"encoding/json"
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

type ServerInfo struct {
	SystemInfo     SystemInfo                 `json:"systemInfo"`
	ComponentTypes map[string][]ComponentType `json:"componentTypes"`
	ProviderTypes  map[string]ProviderType    `json:"providers"`
	Themes         map[string][]Theme         `json:"themes"`
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

	body, _ := io.ReadAll(resp.Body)

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

package testutils

import (
	"context"
	"sync"
	"time"

	"github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"

	"github.com/Nerzal/gocloak/v12"
)

// KeycloakClientManager manages a Keycloak API client with token refresh.
type KeycloakClientManager struct {
	Client     *gocloak.GoCloak
	token      string
	tokenMutex sync.Mutex
}

// NewKeycloakClientManager creates a new Keycloak client manager.
func NewKeycloakClientManager(keycloakURL string) *KeycloakClientManager {
	return &KeycloakClientManager{
		Client: gocloak.NewClient(keycloakURL),
	}
}

// Initialize sets up the initial token and starts the token refresh goroutine.
func (m *KeycloakClientManager) Initialize(ctx context.Context) {
	m.setToken(ctx)

	// To prevent token expiration, we need to refresh it every 30 seconds.
	go func() {
		ticker := time.NewTicker(30 * time.Second)
		defer ticker.Stop()
		defer ginkgo.GinkgoRecover()

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				m.setToken(ctx)
			}
		}
	}()
}

// GetToken returns the current token in a thread-safe manner.
func (m *KeycloakClientManager) GetToken() string {
	m.tokenMutex.Lock()
	defer m.tokenMutex.Unlock()

	return m.token
}

// setToken refreshes the Keycloak token.
func (m *KeycloakClientManager) setToken(ctx context.Context) {
	token, err := m.Client.LoginAdmin(ctx, "admin", "admin", "master")
	gomega.Expect(err).ShouldNot(gomega.HaveOccurred(), "failed to login to keycloak")

	m.tokenMutex.Lock()
	m.token = token.AccessToken
	m.tokenMutex.Unlock()
}

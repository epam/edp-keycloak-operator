package testutils

import (
	"context"
	"encoding/json"
	"sync"
	"time"

	"github.com/Nerzal/gocloak/v12"
	"github.com/go-logr/logr"
	"github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"

	"github.com/epam/edp-keycloak-operator/pkg/client/keycloak/adapter"
)

// KeycloakAdapterManager manages a Keycloak adapter with token refresh.
type KeycloakAdapterManager struct {
	adapter      *adapter.GoCloakAdapter
	config       adapter.GoCloakConfig
	log          logr.Logger
	adapterMutex sync.Mutex
}

// NewKeycloakAdapterManager creates a new Keycloak adapter manager.
func NewKeycloakAdapterManager(keycloakURL string, log logr.Logger) *KeycloakAdapterManager {
	return &KeycloakAdapterManager{
		config: adapter.GoCloakConfig{
			Url:      keycloakURL,
			User:     "admin",
			Password: "admin",
		},
		log: log,
	}
}

// Initialize sets up the initial adapter and starts the token refresh goroutine.
func (m *KeycloakAdapterManager) Initialize(ctx context.Context) {
	m.refreshAdapter(ctx)

	// Refresh adapter every 30 seconds to prevent token expiration
	go func() {
		ticker := time.NewTicker(30 * time.Second)
		defer ticker.Stop()
		defer ginkgo.GinkgoRecover()

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				m.refreshAdapter(ctx)
			}
		}
	}()
}

// GetAdapter returns the current adapter in a thread-safe manner.
func (m *KeycloakAdapterManager) GetAdapter() *adapter.GoCloakAdapter {
	m.adapterMutex.Lock()
	defer m.adapterMutex.Unlock()

	return m.adapter
}

// refreshAdapter creates a new adapter with a fresh token.
func (m *KeycloakAdapterManager) refreshAdapter(ctx context.Context) {
	client := gocloak.NewClient(m.config.Url)
	token, err := client.LoginAdmin(ctx, m.config.User, m.config.Password, "master")
	gomega.Expect(err).ShouldNot(gomega.HaveOccurred(), "failed to login to keycloak")

	tokenData, err := json.Marshal(token)
	gomega.Expect(err).ShouldNot(gomega.HaveOccurred(), "failed to marshal token")

	newAdapter, err := adapter.MakeFromToken(m.config, tokenData, m.log)
	gomega.Expect(err).ShouldNot(gomega.HaveOccurred(), "failed to create adapter")

	m.adapterMutex.Lock()
	m.adapter = newAdapter
	m.adapterMutex.Unlock()
}

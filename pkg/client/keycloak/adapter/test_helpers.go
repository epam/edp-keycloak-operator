package adapter

import (
	"testing"

	"github.com/go-resty/resty/v2"

	"github.com/epam/edp-keycloak-operator/pkg/client/keycloak/adapter/mocks"
)

// newMockClientWithResty creates a mock GoCloak client with resty client configured.
func newMockClientWithResty(t *testing.T, serverUrl string) *mocks.MockGoCloak {
	mockClient := mocks.NewMockGoCloak(t)
	restyClient := resty.New()
	restyClient.SetBaseURL(serverUrl)
	mockClient.On("RestyClient").Return(restyClient).Maybe()

	return mockClient
}

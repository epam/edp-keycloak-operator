package adapter

import (
	"errors"
	"github.com/epam/keycloak-operator/pkg/client/keycloak/mock"
	"testing"

	"github.com/Nerzal/gocloak/v8"
	"github.com/epam/keycloak-operator/pkg/client/keycloak/dto"
	"github.com/stretchr/testify/assert"
)

func TestNewAdapterValidCredentials(t *testing.T) {
	// prepare
	mockClient := new(MockGoCloakClient)
	mockClient.On("LoginAdmin", "user", "password", "master").
		Return(&gocloak.JWT{
			AccessToken: "test",
		}, nil)
	goCloakClientSupplier = func(url string) GoCloak {
		return mockClient
	}
	key := dto.Keycloak{
		Url:  "url",
		User: "user",
		Pwd:  "password",
	}
	factory := GoCloakAdapterFactory{
		Log: &mock.Logger{},
	}

	//test
	client, err := factory.New(key)

	//verify
	assert.NoError(t, err)
	assert.NotNil(t, client)
}

func TestNewAdapterInValidCredentials(t *testing.T) {
	// prepare
	mockClient := new(MockGoCloakClient)
	mockClient.On("LoginAdmin", "user", "invalid", "master").
		Return(&gocloak.JWT{}, errors.New("error in login"))
	goCloakClientSupplier = func(url string) GoCloak {
		return mockClient
	}
	key := dto.Keycloak{
		Url:  "url",
		User: "user",
		Pwd:  "invalid",
	}

	factory := GoCloakAdapterFactory{
		Log: &mock.Logger{},
	}

	//test
	ad, err := factory.New(key)

	//verify
	assert.Error(t, err)
	assert.Nil(t, ad)
}

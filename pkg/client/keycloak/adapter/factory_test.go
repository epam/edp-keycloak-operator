package adapter

import (
	"errors"
	"github.com/stretchr/testify/assert"
	"gopkg.in/nerzal/gocloak.v2"
	"keycloak-operator/pkg/apis/v1/v1alpha1"
	"testing"
)

func TestNewAdapterValidCredentials(t *testing.T) {
	// prepare
	mockClient := new(MockGoCloakClient)
	mockClient.On("LoginAdmin", "user", "password", "master").
		Return(&gocloak.JWT{
			AccessToken: "test",
		}, nil)
	goCloakClientSupplier = func(url string) gocloak.GoCloak {
		return mockClient
	}
	spec := v1alpha1.KeycloakSpec{
		User: "user",
		Pwd:  "password",
		Url:  "url",
	}
	factory := new(GoCloakAdapterFactory)

	//test
	client, err := factory.New(spec)

	//verify
	assert.NoError(t, err)
	assert.NotNil(t, client)
}

func TestNewAdapterInValidCredentials(t *testing.T) {
	// prepare
	mockClient := new(MockGoCloakClient)
	mockClient.On("LoginAdmin", "user", "invalid", "master").
		Return(&gocloak.JWT{}, errors.New("error in login"))
	goCloakClientSupplier = func(url string) gocloak.GoCloak {
		return mockClient
	}
	spec := v1alpha1.KeycloakSpec{
		User: "user",
		Pwd:  "invalid",
		Url:  "url",
	}
	factory := new(GoCloakAdapterFactory)

	//test
	ad, err := factory.New(spec)

	//verify
	assert.Error(t, err)
	assert.Nil(t, ad)
}

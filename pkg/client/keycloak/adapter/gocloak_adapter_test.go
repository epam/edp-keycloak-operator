package adapter

import (
	"errors"
	"github.com/Nerzal/gocloak/v3"
	"github.com/stretchr/testify/assert"
	"keycloak-operator/pkg/client/keycloak/dto"
	"testing"
)

func TestGoCloakAdapter_ExistRealmPositive(t *testing.T) {
	//prepare
	mockClient := new(MockGoCloakClient)
	mockClient.On("GetRealm", "token", "realmName").
		Return(&gocloak.RealmRepresentation{Realm: "realm"}, nil)
	adapter := GoCloakAdapter{
		client: mockClient,
		token:  gocloak.JWT{AccessToken: "token"},
	}
	realm := dto.Realm{
		Name: "realmName",
	}

	//test
	res, err := adapter.ExistRealm(realm)

	//verify
	assert.NoError(t, err)
	assert.True(t, *res)
}

func TestGoCloakAdapter_ExistRealm404(t *testing.T) {
	//prepare
	mockClient := new(MockGoCloakClient)
	mockClient.On("GetRealm", "token", "realmName").
		Return(nil, errors.New("404"))
	adapter := GoCloakAdapter{
		client: mockClient,
		token:  gocloak.JWT{AccessToken: "token"},
	}
	realm := dto.Realm{
		Name: "realmName",
	}

	//test
	res, err := adapter.ExistRealm(realm)

	//verify
	assert.NoError(t, err)
	assert.False(t, *res)
}

func TestGoCloakAdapter_ExistRealmError(t *testing.T) {
	//prepare
	mockClient := new(MockGoCloakClient)
	mockClient.On("GetRealm", "token", "realmName").
		Return(nil, errors.New("error in get realm"))
	adapter := GoCloakAdapter{
		client: mockClient,
		token:  gocloak.JWT{AccessToken: "token"},
	}
	realm := dto.Realm{
		Name: "realmName",
	}

	//test
	res, err := adapter.ExistRealm(realm)

	//verify
	assert.Error(t, err)
	assert.Nil(t, res)
}

func TestGoCloakAdapter_CreateRealm(t *testing.T) {
	//prepare
	mockClient := new(MockGoCloakClient)
	defRealm := getDefaultRealm("realmName")
	mockClient.On("CreateRealm", "token", defRealm).
		Return(nil)
	adapter := GoCloakAdapter{
		client: mockClient,
		token:  gocloak.JWT{AccessToken: "token"},
	}
	realm := dto.Realm{
		Name: "realmName",
	}

	//test
	err := adapter.CreateRealmWithDefaultConfig(realm)

	//verify
	assert.NoError(t, err)
}

func TestGoCloakAdapter_CreateRealmError(t *testing.T) {
	//prepare
	mockClient := new(MockGoCloakClient)
	defRealm := getDefaultRealm("realmName")
	mockClient.On("CreateRealm", "token", defRealm).
		Return(errors.New("error in create realm"))
	adapter := GoCloakAdapter{
		client: mockClient,
		token:  gocloak.JWT{AccessToken: "token"},
	}
	realm := dto.Realm{
		Name: "realmName",
	}

	//test
	err := adapter.CreateRealmWithDefaultConfig(realm)

	//verify
	assert.Error(t, err)
}

package keycloak

import (
	"errors"
	"github.com/stretchr/testify/assert"
	"gopkg.in/nerzal/gocloak.v2"
	"keycloak-operator/pkg/apis/v1/v1alpha1"
	"testing"
)

func TestGoCloakAdapter_GetConnectionHappyPath(t *testing.T) {
	// prepare
	client := new(MockGoCloakClient)
	client.On("LoginAdmin", "user", "pass", "master").
		Return(&gocloak.JWT{
			AccessToken: "test",
		}, nil)
	adapter := GoCloakAdapter{
		ClientSup: func(url string) gocloak.GoCloak {
			return client
		},
	}
	cr := v1alpha1.Keycloak{
		Spec: v1alpha1.KeycloakSpec{
			User: "user",
			Pwd:  "pass",
		},
		Status: v1alpha1.KeycloakStatus{
			Connected: true,
		},
	}

	// test
	res, err := adapter.GetConnection(cr)

	// verify
	assert.Nil(t, err)
	assert.NotNil(t, res)
	assert.Equal(t, "test", res.Token.AccessToken)
}

func TestGoCloakAdapter_GetConnectionInvalidPass(t *testing.T) {
	// prepare
	client := new(MockGoCloakClient)
	client.On("LoginAdmin", "user", "pass", "master").
		Return(&gocloak.JWT{}, errors.New("some test error"))
	adapter := GoCloakAdapter{
		ClientSup: func(url string) gocloak.GoCloak {
			return client
		},
	}
	cr := v1alpha1.Keycloak{
		Spec: v1alpha1.KeycloakSpec{
			User: "user",
			Pwd:  "pass",
		},
		Status: v1alpha1.KeycloakStatus{
			Connected: true,
		},
	}

	// test
	res, err := adapter.GetConnection(cr)

	// verify
	assert.Nil(t, res)
	assert.NotNil(t, err)
}

func TestGoCloakAdapter_GetConnectionInvalidAdapter(t *testing.T) {
	// prepare
	adapter := GoCloakAdapter{}
	cr := v1alpha1.Keycloak{
		Spec: v1alpha1.KeycloakSpec{
			User: "user",
			Pwd:  "pass",
		},
		Status: v1alpha1.KeycloakStatus{
			Connected: true,
		},
	}

	// test
	res, err := adapter.GetConnection(cr)

	// verify
	assert.Nil(t, res)
	assert.NotNil(t, err)
}

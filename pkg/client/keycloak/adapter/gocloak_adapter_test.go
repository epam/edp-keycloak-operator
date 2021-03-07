package adapter

import (
	"fmt"
	"net/url"
	"testing"

	"github.com/Nerzal/gocloak/v8"
	"github.com/epmd-edp/keycloak-operator/pkg/client/keycloak/dto"
	"github.com/go-resty/resty/v2"
	"github.com/jarcoal/httpmock"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
)

func TestGoCloakAdapter_ExistRealmPositive(t *testing.T) {
	//prepare
	mockClient := new(MockGoCloakClient)
	mockClient.On("GetRealm", "token", "realmName").
		Return(&gocloak.RealmRepresentation{Realm: gocloak.StringP("realm")}, nil)
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

func TestGoCloakAdapter_GetClientProtocolMappers_Failure2(t *testing.T) {
	client := dto.Client{
		RealmName: "test",
		ClientId:  "test",
	}
	clientID := "321"
	mockClient := new(MockGoCloakClient)
	restyClient := resty.New()
	httpmock.ActivateNonDefault(restyClient.GetClient())
	mockClient.On("RestyClient").Return(restyClient)
	messageBody := "not found"
	responder := httpmock.NewStringResponder(404, messageBody)
	httpmock.RegisterResponder(
		"GET",
		fmt.Sprintf("/auth/admin/realms/%s/clients/%s/protocol-mappers/models", client.RealmName, clientID),
		responder)

	adapter := GoCloakAdapter{
		client:   mockClient,
		token:    gocloak.JWT{AccessToken: "token"},
		basePath: "",
	}

	_, err := adapter.GetClientProtocolMappers(client, clientID)
	if err == nil {
		t.Fatal(err)
	}

	if err.Error() != messageBody {
		t.Fatal("wrong error returned")
	}
}

func TestGoCloakAdapter_GetClientProtocolMappers_Failure(t *testing.T) {
	client := dto.Client{
		RealmName: "test",
		ClientId:  "test",
	}
	clientID := "321"
	mockClient := new(MockGoCloakClient)
	restyClient := resty.New()
	httpmock.ActivateNonDefault(restyClient.GetClient())
	mockClient.On("RestyClient").Return(restyClient)

	mockErr := errors.New("fatal")

	responder := httpmock.NewErrorResponder(mockErr)
	httpmock.RegisterResponder(
		"GET",
		fmt.Sprintf("/auth/admin/realms/%s/clients/%s/protocol-mappers/models", client.RealmName, clientID),
		responder)

	adapter := GoCloakAdapter{
		client:   mockClient,
		token:    gocloak.JWT{AccessToken: "token"},
		basePath: "",
	}

	_, err := adapter.GetClientProtocolMappers(client, clientID)
	if err == nil {
		t.Fatal(err)
	}

	switch errors.Cause(err).(type) {
	case *url.Error:
		if errors.Cause(err).(*url.Error).Err != mockErr {
			t.Fatal("wrong error returned")
		}
	default:
		t.Fatal("wrong error returned")
	}
}

func TestGoCloakAdapter_SyncClientProtocolMapper_Success(t *testing.T) {
	client := dto.Client{
		RealmName: "test",
		ClientId:  "test",
	}
	clientID := "321"

	mockClient := new(MockGoCloakClient)
	restyClient := resty.New()
	httpmock.ActivateNonDefault(restyClient.GetClient())
	mockClient.On("RestyClient").Return(restyClient)
	mockClient.On("GetClients", client.RealmName, gocloak.GetClientsParams{
		ClientID: &client.ClientId,
	}).Return([]*gocloak.Client{
		{
			ClientID: &client.ClientId,
			ID:       &clientID,
		},
	}, nil)

	kcMappers := []gocloak.ProtocolMapperRepresentation{
		{
			ID:             gocloak.StringP("8863fce4-dcd1-48af-afbc-499cc07c31bd"),
			Name:           gocloak.StringP("test123"),
			Protocol:       gocloak.StringP("openid-connect"),
			ProtocolMapper: gocloak.StringP("oidc-claims-param-token-mapper"),
			Config:         &map[string]string{},
		},
		{
			ID:             gocloak.StringP("8863fce4-dcd1-48af-afbc-499cc07c31bd4"),
			Name:           gocloak.StringP("test1234"),
			Protocol:       gocloak.StringP("openid-connect"),
			ProtocolMapper: gocloak.StringP("oidc-claims-param-token-mapper"),
			Config:         &map[string]string{},
		},
	}

	crMappers := []gocloak.ProtocolMapperRepresentation{
		{
			ID:             gocloak.StringP("8863fce4-dcd1-48af-afbc-499cc07c31bd4"),
			Name:           gocloak.StringP("test1234"),
			Protocol:       gocloak.StringP("openid-connect"),
			ProtocolMapper: gocloak.StringP("oidc-claims-param-token-mapper"),
			Config: &map[string]string{
				"foo": "bar",
			},
		},
		{
			Name:           gocloak.StringP("test12341125"),
			Protocol:       gocloak.StringP("openid-connect"),
			ProtocolMapper: gocloak.StringP("oidc-claims-param-token-mapper"),
			Config: &map[string]string{
				"bar": "foo",
			},
		},
		{
			Name:           gocloak.StringP("test1234112554684"),
			Protocol:       gocloak.StringP("openid-connect"),
			ProtocolMapper: gocloak.StringP("oidc-claims-param-token-mapper"),
		},
	}

	responder, err := httpmock.NewJsonResponder(200, kcMappers)
	if err != nil {
		t.Fatal(err)
	}

	mockClient.On("DeleteClientProtocolMapper", client.RealmName, clientID, *kcMappers[0].ID).
		Return(nil)

	mockClient.On("UpdateClientProtocolMapper", client.RealmName, clientID, *crMappers[0].ID, crMappers[0]).
		Return(nil)

	mockClient.On("CreateClientProtocolMapper", client.RealmName, clientID, crMappers[1]).
		Return("", nil)

	mockClient.On("CreateClientProtocolMapper", client.RealmName, clientID,
		gocloak.ProtocolMapperRepresentation{
			Name:           gocloak.StringP("test1234112554684"),
			Protocol:       gocloak.StringP("openid-connect"),
			ProtocolMapper: gocloak.StringP("oidc-claims-param-token-mapper"),
			Config:         &map[string]string{},
		}).
		Return("", nil)

	httpmock.RegisterResponder(
		"GET",
		fmt.Sprintf("/auth/admin/realms/%s/clients/%s/protocol-mappers/models", client.RealmName, clientID),
		responder)

	adapter := GoCloakAdapter{
		client:   mockClient,
		token:    gocloak.JWT{AccessToken: "token"},
		basePath: "",
	}

	if err := adapter.SyncClientProtocolMapper(client, crMappers); err != nil {
		t.Fatal(err)
	}
}

func TestGoCloakAdapter_SyncClientProtocolMapper_ClientIDFailure(t *testing.T) {
	client := dto.Client{
		RealmName: "test",
		ClientId:  "test",
	}
	clientID := "123"
	mockErr := errors.New("fatal")

	mockClient := new(MockGoCloakClient)
	mockClient.On("GetClients", client.RealmName, gocloak.GetClientsParams{
		ClientID: &client.ClientId,
	}).Return([]*gocloak.Client{
		{
			ClientID: &client.ClientId,
			ID:       &clientID,
		},
	}, mockErr)

	adapter := GoCloakAdapter{
		client:   mockClient,
		token:    gocloak.JWT{AccessToken: "token"},
		basePath: "",
	}

	err := adapter.SyncClientProtocolMapper(client, []gocloak.ProtocolMapperRepresentation{})
	if err == nil {
		t.Fatal("no error on get clients fatal")
	}

	if errors.Cause(err) != mockErr {
		t.Fatal("wrong error returned")
	}
}

func TestGoCloakAdapter_SyncRealmRole(t *testing.T) {
	mockClient := MockGoCloakClient{}

	currentRole := gocloak.Role{Name: gocloak.StringP("test"), ID: gocloak.StringP("321"),
		Composite: gocloak.BoolP(true), Description: gocloak.StringP(""),
		Attributes: &map[string][]string{
			"foo": []string{"foo", "bar"},
		}}
	mockClient.On("GetRealmRole", "test", "test").Return(&currentRole, nil)

	composite1 := gocloak.Role{Name: gocloak.StringP("c1")}
	mockClient.On("GetCompositeRealmRolesByRoleID", "test", "321").Return([]*gocloak.Role{
		&composite1,
	}, nil)
	compositeFoo := gocloak.Role{Name: gocloak.StringP("foo")}
	mockClient.On("GetRealmRole", "test", "foo").Return(&compositeFoo, nil)
	compositeBar := gocloak.Role{Name: gocloak.StringP("bar")}
	mockClient.On("GetRealmRole", "test", "bar").Return(&compositeBar, nil)
	mockClient.On("AddRealmRoleComposite", "test", "test",
		[]gocloak.Role{compositeFoo, compositeBar}).
		Return(nil)
	mockClient.On("DeleteRealmRoleComposite", "test", "test", []gocloak.Role{
		composite1,
	}).Return(nil)
	mockClient.On("UpdateRealmRole", "test", "test", currentRole).Return(nil)

	adapter := GoCloakAdapter{
		client:   &mockClient,
		token:    gocloak.JWT{AccessToken: "token"},
		basePath: "",
	}

	realm := dto.Realm{Name: "test"}
	role := dto.RealmRole{Name: "test", Composites: []string{"foo", "bar"}, IsComposite: true,
		Attributes: map[string][]string{
			"foo": []string{"foo", "bar"},
		}}

	if err := adapter.SyncRealmRole(&realm, &role); err != nil {
		t.Fatal(err)
	}
}

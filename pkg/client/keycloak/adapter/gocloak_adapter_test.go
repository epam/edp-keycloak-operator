package adapter

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/Nerzal/gocloak/v12"
	"github.com/go-logr/logr"
	"github.com/go-resty/resty/v2"
	"github.com/jarcoal/httpmock"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	keycloakApi "github.com/epam/edp-keycloak-operator/api/v1"
	"github.com/epam/edp-keycloak-operator/pkg/client/keycloak/api"
	"github.com/epam/edp-keycloak-operator/pkg/client/keycloak/dto"
	"github.com/epam/edp-keycloak-operator/pkg/client/keycloak/mock"
	"github.com/epam/edp-keycloak-operator/pkg/fakehttp"
)

type AdapterTestSuite struct {
	suite.Suite
	restyClient       *resty.Client
	goCloakMockClient *MockGoCloakClient
	adapter           *GoCloakAdapter
	realmName         string
}

func (e *AdapterTestSuite) SetupTest() {
	e.restyClient = resty.New()

	httpmock.Reset()
	httpmock.ActivateNonDefault(e.restyClient.GetClient())

	e.goCloakMockClient = new(MockGoCloakClient)
	e.goCloakMockClient.On("RestyClient").Return(e.restyClient)

	e.adapter = &GoCloakAdapter{
		client: e.goCloakMockClient,
		token:  &gocloak.JWT{AccessToken: "token"},
		log:    mock.NewLogr(),
	}

	e.realmName = "realm123"
}

func TestAdapterTestSuite(t *testing.T) {
	suite.Run(t, new(AdapterTestSuite))
}

func (e *AdapterTestSuite) TestMakeFromServiceAccount() {
	t := e.T()

	t.Parallel()

	realmsEndpoint := "/realms/master/protocol/openid-connect/token"

	tests := []struct {
		name       string
		mockServer fakehttp.Server
		wantErr    require.ErrorAssertionFunc
	}{
		{
			name: "should succeed",
			mockServer: fakehttp.NewServerBuilder().
				AddStringResponder(realmsEndpoint, "{}").
				BuildAndStart(),
			wantErr: require.NoError,
		},
		{
			name: "should succeed with legacy endpoint",
			mockServer: fakehttp.NewServerBuilder().
				AddStringResponder("/auth"+realmsEndpoint, "{}").
				BuildAndStart(),
			wantErr: require.NoError,
		},
		{
			name: "should fail on status bad request",
			mockServer: fakehttp.NewServerBuilder().
				AddStringResponderWithCode(http.StatusBadRequest, "/auth"+realmsEndpoint, "{}").
				BuildAndStart(),
			wantErr: func(t require.TestingT, err error, _ ...interface{}) {
				require.Error(t, err)
				require.EqualError(t, err, "failed to login with client creds on both current and legacy clients - clientID: k-cl-id, realm: master: 400 Bad Request")
			},
		},
	}

	for _, tt := range tests {
		tt := tt

		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			defer tt.mockServer.Close()

			_, err := MakeFromServiceAccount(context.Background(), tt.mockServer.GetURL(),
				"k-cl-id", "k-secret", "master", mock.NewLogr(), resty.New())
			tt.wantErr(t, err)
		})
	}
}

func (e *AdapterTestSuite) TestMake() {
	t := e.T()

	t.Parallel()

	realmsEndpoint := "/realms/master/protocol/openid-connect/token"

	tests := []struct {
		name       string
		mockServer fakehttp.Server
		wantErr    require.ErrorAssertionFunc
	}{
		{
			name: "should succeed",
			mockServer: fakehttp.NewServerBuilder().
				AddStringResponder(realmsEndpoint, "{}").
				BuildAndStart(),
			wantErr: require.NoError,
		},
		{
			name: "should succeed with legacy endpoint",
			mockServer: fakehttp.NewServerBuilder().
				AddStringResponder("/auth"+realmsEndpoint, "{}").
				BuildAndStart(),
			wantErr: require.NoError,
		},
		{
			name:       "should fail on unsupported protocol scheme",
			mockServer: nil,
			wantErr: func(t require.TestingT, err error, i ...interface{}) {
				require.Error(t, err)
				require.Contains(t, err.Error(), "unsupported protocol scheme")
			},
		},
		{
			name: "should fail with status 400",
			mockServer: fakehttp.NewServerBuilder().
				AddStringResponderWithCode(http.StatusBadRequest, "/auth"+realmsEndpoint, "{}").
				BuildAndStart(),
			wantErr: func(t require.TestingT, err error, i ...interface{}) {
				require.Error(t, err)
				require.Contains(t, err.Error(), "400")
			},
		},
	}

	for _, tt := range tests {
		tt := tt

		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			url := "test_url"
			if tt.mockServer != nil {
				url = tt.mockServer.GetURL()

				defer tt.mockServer.Close()
			}

			_, err := Make(context.Background(), url, "bar", "baz", mock.NewLogr(), resty.New())
			tt.wantErr(t, err)
		})
	}
}

func (e *AdapterTestSuite) TestGoCloakAdapter_ExistRealmPositive() {
	e.goCloakMockClient.On("GetRealm", "token", "realmName").
		Return(&gocloak.RealmRepresentation{Realm: gocloak.StringP("realm")}, nil)

	realm := dto.Realm{
		Name: "realmName",
	}

	res, err := e.adapter.ExistRealm(realm.Name)

	// verify
	assert.NoError(e.T(), err)
	assert.True(e.T(), res)
}

func TestGetDefaultRealm(t *testing.T) {
	id := "test"
	r := getDefaultRealm(&dto.Realm{
		ID: &id,
	})

	if *r.ID != id {
		t.Fatal("wrong realm id")
	}
}

func TestGoCloakAdapter_ExistRealm404(t *testing.T) {
	// prepare
	mockClient := new(MockGoCloakClient)
	mockClient.On("GetRealm", "token", "realmName").
		Return(nil, errors.New("404"))

	adapter := GoCloakAdapter{
		client: mockClient,
		token:  &gocloak.JWT{AccessToken: "token"},
		log:    mock.NewLogr(),
	}
	realm := dto.Realm{
		Name: "realmName",
	}

	// test
	res, err := adapter.ExistRealm(realm.Name)

	// verify
	assert.NoError(t, err)
	assert.False(t, res)
}

func TestGoCloakAdapter_ExistRealmError(t *testing.T) {
	// prepare
	mockClient := new(MockGoCloakClient)
	mockClient.On("GetRealm", "token", "realmName").
		Return(nil, errors.New("error in get realm"))

	adapter := GoCloakAdapter{
		client: mockClient,
		token:  &gocloak.JWT{AccessToken: "token"},
		log:    mock.NewLogr(),
	}
	realm := dto.Realm{
		Name: "realmName",
	}

	// test
	res, err := adapter.ExistRealm(realm.Name)

	// verify
	assert.Error(t, err)
	assert.False(t, res)
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
		fmt.Sprintf("/admin/realms/%s/clients/%s/protocol-mappers/models", client.RealmName, clientID),
		responder)

	adapter := GoCloakAdapter{
		client:   mockClient,
		token:    &gocloak.JWT{AccessToken: "token"},
		basePath: "",
	}

	_, err := adapter.GetClientProtocolMappers(&client, clientID)
	require.Error(t, err)

	assert.Equal(t, messageBody, err.Error())
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
		fmt.Sprintf("/admin/realms/%s/clients/%s/protocol-mappers/models", client.RealmName, clientID),
		responder)

	adapter := GoCloakAdapter{
		client:   mockClient,
		token:    &gocloak.JWT{AccessToken: "token"},
		basePath: "",
	}

	_, err := adapter.GetClientProtocolMappers(&client, clientID)
	require.Error(t, err)

	assert.ErrorIs(t, err, mockErr)
}

func TestGoCloakAdapter_CreateClient(t *testing.T) {
	mockClient := new(MockGoCloakClient)
	logger := mock.NewLogr()

	cl := dto.Client{
		RedirectUris: []string{"https://test.com"},
	}
	a := GoCloakAdapter{
		client: mockClient,
		token:  &gocloak.JWT{AccessToken: "token"},
		log:    logger,
	}

	mockClient.On("CreateClient", "", getGclCln(&cl)).Return("id", nil).Once()

	err := a.CreateClient(context.Background(), &cl)
	assert.NoError(t, err)

	createErr := errors.New("create-err")
	mockClient.On("CreateClient", "", getGclCln(&cl)).Return("", createErr).Once()
	err = a.CreateClient(context.Background(), &cl)

	assert.ErrorIs(t, err, createErr)
}

func TestGoCloakAdapter_UpdateClient(t *testing.T) {
	mockClient := new(MockGoCloakClient)
	logger := mock.NewLogr()

	cl := dto.Client{}
	a := GoCloakAdapter{
		client: mockClient,
		token:  &gocloak.JWT{AccessToken: "token"},
		log:    logger,
	}

	mockClient.On("UpdateClient", a.token.AccessToken, cl.RealmName,
		getGclCln(&cl)).Return(nil).Once()

	err := a.UpdateClient(context.Background(), &cl)
	assert.NoError(t, err)

	updErr := errors.New("update-error")

	mockClient.On("UpdateClient", a.token.AccessToken, cl.RealmName,
		getGclCln(&cl)).Return(updErr).Once()

	err = a.UpdateClient(context.Background(), &cl)
	assert.True(t, errors.Is(err, updErr))

	mockClient.AssertExpectations(t)
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
	require.NoError(t, err)

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
		fmt.Sprintf("/admin/realms/%s/clients/%s/protocol-mappers/models", client.RealmName, clientID),
		responder)

	adapter := GoCloakAdapter{
		client:   mockClient,
		token:    &gocloak.JWT{AccessToken: "token"},
		basePath: "",
		log:      mock.NewLogr(),
	}

	err = adapter.SyncClientProtocolMapper(&client, crMappers, false)
	require.NoError(t, err)
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
		token:    &gocloak.JWT{AccessToken: "token"},
		basePath: "",
		log:      mock.NewLogr(),
	}

	err := adapter.SyncClientProtocolMapper(&client, []gocloak.ProtocolMapperRepresentation{}, false)
	if err == nil {
		t.Fatal("no error on get clients fatal")
	}

	assert.ErrorIs(t, err, mockErr)
}

func TestGoCloakAdapter_SyncRealmRole_Duplicated(t *testing.T) {
	mockClient := MockGoCloakClient{}
	currentRole := gocloak.Role{Name: gocloak.StringP("role1")}

	mockClient.On("GetRealmRole", "realm1", "role1").Return(&currentRole, nil)

	adapter := GoCloakAdapter{
		client: &mockClient,
		token:  &gocloak.JWT{AccessToken: "token"},
		log:    mock.NewLogr(),
	}

	role := dto.PrimaryRealmRole{Name: "role1"}

	err := adapter.SyncRealmRole("realm1", &role)
	if err == nil {
		t.Fatal("no error returned on duplicated role")
	}

	if !IsErrDuplicated(err) {
		t.Log(err)
		t.Fatal("wrong error returned")
	}
}

func TestGoCloakAdapter_SyncRealmRole(t *testing.T) {
	mockClient := MockGoCloakClient{}
	realmName, roleName, roleID := "realm1", "role1", "id321"
	currentRole := gocloak.Role{Name: &roleName, ID: &roleID,
		Composite: gocloak.BoolP(true), Description: gocloak.StringP(""),
		Attributes: &map[string][]string{
			"foo": []string{"foo", "bar"},
		}}
	mockClient.On("GetRealmRole", realmName, roleName).Return(&currentRole, nil)

	composite1 := gocloak.Role{Name: gocloak.StringP("c1")}
	mockClient.On("GetCompositeRealmRolesByRoleID", realmName, roleID).Return([]*gocloak.Role{
		&composite1,
	}, nil)

	compositeFoo := gocloak.Role{Name: gocloak.StringP("foo")}
	mockClient.On("GetRealmRole", realmName, *compositeFoo.Name).Return(&compositeFoo, nil)

	compositeBar := gocloak.Role{Name: gocloak.StringP("bar")}
	mockClient.On("GetRealmRole", realmName, *compositeBar.Name).Return(&compositeBar, nil)
	mockClient.On("AddRealmRoleComposite", realmName, roleName,
		[]gocloak.Role{compositeFoo, compositeBar}).
		Return(nil)
	mockClient.On("DeleteRealmRoleComposite", realmName, roleName, []gocloak.Role{
		composite1,
	}).Return(nil)
	mockClient.On("UpdateRealmRole", realmName, roleName, currentRole).Return(nil)

	realm := gocloak.RealmRepresentation{}
	updatedRealm := gocloak.RealmRepresentation{DefaultRoles: &[]string{roleName}}

	mockClient.On("GetRealm", "token", realmName).Return(&realm, nil)
	mockClient.On("UpdateRealm", updatedRealm).Return(nil)

	adapter := GoCloakAdapter{
		client:   &mockClient,
		token:    &gocloak.JWT{AccessToken: "token"},
		basePath: "",
		log:      mock.NewLogr(),
	}

	role := dto.PrimaryRealmRole{Name: roleName, Composites: []string{"foo", "bar"}, IsComposite: true,
		Attributes: map[string][]string{
			"foo": []string{"foo", "bar"},
		}, IsDefault: true, ID: gocloak.StringP("id321")}

	if err := adapter.SyncRealmRole(realmName, &role); err != nil {
		require.NoError(t, err)
	}
}

func TestGoCloakAdapter_SyncServiceAccountRoles_AddOnly(t *testing.T) {
	mockClient := MockGoCloakClient{}
	adapter := GoCloakAdapter{
		client:   &mockClient,
		token:    &gocloak.JWT{AccessToken: "token"},
		basePath: "",
		log:      mock.NewLogr(),
	}

	mockClient.On("GetClientServiceAccount", "realm", "client").Return(&gocloak.User{
		ID: gocloak.StringP("id"),
	}, nil)

	mockClient.On("GetRoleMappingByUserID", "realm", "id").
		Return(&gocloak.MappingsRepresentation{}, nil)
	mockClient.On("GetRealmRole", "realm", "foo").
		Return(&gocloak.Role{}, nil)
	mockClient.On("AddRealmRoleToUser", "realm", "id", []gocloak.Role{{}}).
		Return(nil)
	mockClient.On("GetClients", "realm",
		gocloak.GetClientsParams{ClientID: gocloak.StringP("bar")}).Return(nil,
		errors.New("get clients fatal"))

	err := adapter.SyncServiceAccountRoles("realm", "client", []string{"foo"},
		map[string][]string{
			"bar": {"john"},
		}, true)
	require.Error(t, err)

	if !strings.Contains(err.Error(),
		"unable to sync service account client roles: error during syncOneEntityClientRole: unable to get client id, realm: realm, clientID bar: unable to get realm clients: get clients fatal") {
		t.Fatalf("wrong error returned: %s", err.Error())
	}
}

func TestGoCloakAdapter_SyncServiceAccountRoles(t *testing.T) {
	mockClient := MockGoCloakClient{}
	adapter := GoCloakAdapter{
		client:   &mockClient,
		token:    &gocloak.JWT{AccessToken: "token"},
		basePath: "",
		log:      mock.NewLogr(),
	}

	mockClient.On("GetClientServiceAccount", "realm", "client").Return(&gocloak.User{
		ID: gocloak.StringP("id"),
	}, nil)
	mockClient.On("GetRoleMappingByUserID", "realm", "id").
		Return(&gocloak.MappingsRepresentation{RealmMappings: &[]gocloak.Role{
			{Name: gocloak.StringP("exist_realm_role1")},
			{Name: gocloak.StringP("exist_realm_role2")},
		}, ClientMappings: map[string]*gocloak.ClientMappingsRepresentation{
			"zabrod": {Client: gocloak.StringP("zabrod"), ID: gocloak.StringP("iiss123"),
				Mappings: &[]gocloak.Role{
					{Name: gocloak.StringP("exist_client_role1")},
					{Name: gocloak.StringP("exist_client_role2")},
				}},
			"foo": {Client: gocloak.StringP("foo"), ID: gocloak.StringP("foo321"),
				Mappings: &[]gocloak.Role{
					{Name: gocloak.StringP("baz")},
					{Name: gocloak.StringP("zaz")},
				}},
		}}, nil)
	mockClient.On("GetRealmRole", "realm", "foo").
		Return(&gocloak.Role{}, nil)
	mockClient.On("GetRealmRole", "realm", "bar").
		Return(&gocloak.Role{}, nil)
	mockClient.On("AddRealmRoleToUser", "realm", "id", []gocloak.Role{{}, {}}).
		Return(nil)
	mockClient.On("GetClients", "realm",
		gocloak.GetClientsParams{ClientID: gocloak.StringP("foo")}).Return([]*gocloak.Client{
		{ClientID: gocloak.StringP("foo"), ID: gocloak.StringP("foo321")},
	}, nil)
	mockClient.On("GetClients", "realm",
		gocloak.GetClientsParams{ClientID: gocloak.StringP("bar")}).Return([]*gocloak.Client{
		{ClientID: gocloak.StringP("bar"), ID: gocloak.StringP("bar321")},
	}, nil)
	mockClient.On("GetClientRole", "realm", "foo321", "foo").Return(&gocloak.Role{}, nil)
	mockClient.On("GetClientRole", "realm", "foo321", "bar").Return(&gocloak.Role{}, nil)
	mockClient.On("GetClientRole", "realm", "bar321", "john").Return(&gocloak.Role{}, nil)
	mockClient.On("AddClientRoleToUser", "realm", "foo321", "id", []gocloak.Role{{}, {}}).
		Return(nil)
	mockClient.On("AddClientRoleToUser", "realm", "bar321", "id", []gocloak.Role{{}}).
		Return(nil)
	mockClient.On("DeleteRealmRoleFromUser", "realm", "id", []gocloak.Role{
		{Name: gocloak.StringP("exist_realm_role1")},
		{Name: gocloak.StringP("exist_realm_role2")},
	}).Return(nil)
	mockClient.On("DeleteClientRoleFromUser", "realm", "foo321", "id",
		[]gocloak.Role{
			{Name: gocloak.StringP("baz")},
			{Name: gocloak.StringP("zaz")},
		}).Return(nil)
	mockClient.On("DeleteClientRoleFromUser", "realm", "iiss123", "id",
		[]gocloak.Role{
			{Name: gocloak.StringP("exist_client_role1")},
			{Name: gocloak.StringP("exist_client_role2")},
		}).Return(nil)

	if err := adapter.SyncServiceAccountRoles("realm", "client", []string{"foo", "bar"},
		map[string][]string{
			"foo": {"foo", "bar"},
			"bar": {"john"},
		}, false); err != nil {
		require.NoError(t, err)
	}
}

func TestGoCloakAdapter_SyncRealmGroup_FailureGetGroupsFatal(t *testing.T) {
	clMock := MockGoCloakClient{}

	adapter := GoCloakAdapter{
		client: &clMock,
		token:  &gocloak.JWT{AccessToken: "token"},
	}

	group := keycloakApi.KeycloakRealmGroupSpec{
		Name: "group1",
	}

	clMock.On("GetGroups", "realm1", gocloak.GetGroupsParams{
		Search: &group.Name,
	}).Return(nil, errors.New("fatal mock"))

	_, err := adapter.SyncRealmGroup("realm1", &group)

	if err == nil {
		t.Fatal("error is not returned")
	}

	if errors.Cause(err).Error() != "fatal mock" {
		t.Fatalf("wrong error returned: %s", errors.Cause(err).Error())
	}
}

func TestGoCloakAdapter_SyncRealmGroup(t *testing.T) {
	mockClient := MockGoCloakClient{}
	adapter := GoCloakAdapter{
		client:   &mockClient,
		token:    &gocloak.JWT{AccessToken: "token"},
		basePath: "",
		log:      mock.NewLogr(),
	}

	oldChildGroup := gocloak.Group{Name: gocloak.StringP("old-group")}

	mockClient.On("GetGroups", "realm1",
		gocloak.GetGroupsParams{Search: gocloak.StringP("group1")}).
		Return([]*gocloak.Group{{Name: gocloak.StringP("group1"), ID: gocloak.StringP("1"),
			SubGroups: &[]gocloak.Group{oldChildGroup}}}, nil)
	mockClient.On("UpdateGroup", "realm1", gocloak.Group{Name: gocloak.StringP("group1"),
		Attributes: &map[string][]string{"foo": {"foo", "bar"}},
		Path:       gocloak.StringP(""),
		Access:     &map[string]bool{}, ID: gocloak.StringP("1"),
		SubGroups: &[]gocloak.Group{{Name: gocloak.StringP("old-group")}}}).Return(nil)

	oldRole1, oldRole2 := gocloak.Role{Name: gocloak.StringP("old-role-1")},
		gocloak.Role{Name: gocloak.StringP("old-role-2")}
	newRole1, newRole2 := gocloak.Role{Name: gocloak.StringP("realm-role1")},
		gocloak.Role{Name: gocloak.StringP("realm-role2")}
	oldClientRole1, oldClientRole2, oldClientRole3 := gocloak.Role{Name: gocloak.StringP("oclient-role-1")},
		gocloak.Role{Name: gocloak.StringP("oclient-role-2")},
		gocloak.Role{Name: gocloak.StringP("oclient-role-3")}
	newClientRole1, newClientRole2, newClientRole4 := gocloak.Role{Name: gocloak.StringP("client-role1")},
		gocloak.Role{Name: gocloak.StringP("client-role2")}, gocloak.Role{Name: gocloak.StringP("client-role4")}

	mockClient.On("GetRoleMappingByGroupID", "realm1", "1").
		Return(&gocloak.MappingsRepresentation{
			RealmMappings: &[]gocloak.Role{oldRole1, oldRole2},
			ClientMappings: map[string]*gocloak.ClientMappingsRepresentation{
				"old-cl-1": {Client: gocloak.StringP("old-cl-1"), ID: gocloak.StringP("321"),
					Mappings: &[]gocloak.Role{oldClientRole1, oldClientRole2}},
				"old-cl-3": {Client: gocloak.StringP("old-cl-3"), ID: gocloak.StringP("3214"),
					Mappings: &[]gocloak.Role{oldClientRole3}},
			},
		}, nil)

	subGroup1, subGroup2 := gocloak.Group{Name: gocloak.StringP("subgroup1"), ID: gocloak.StringP("2")},
		gocloak.Group{Name: gocloak.StringP("subgroup2"), ID: gocloak.StringP("3")}

	mockClient.On("CreateChildGroup", "realm1", "1", &gocloak.Group{}).Return(nil)

	mockClient.On("GetGroups", "realm1",
		gocloak.GetGroupsParams{Search: subGroup1.Name}).
		Return([]*gocloak.Group{&subGroup1}, nil)
	mockClient.On("GetGroups", "realm1",
		gocloak.GetGroupsParams{Search: subGroup2.Name}).
		Return([]*gocloak.Group{&subGroup2}, nil)
	mockClient.On("CreateChildGroup", "realm1", "1", subGroup1).Return("", nil)
	mockClient.On("CreateChildGroup", "realm1", "1", subGroup2).Return("", nil)
	mockClient.On("CreateGroup", "realm1", oldChildGroup).Return("", nil)
	mockClient.On("GetRealmRole", "realm1", "realm-role1").Return(&newRole1, nil)
	mockClient.On("GetRealmRole", "realm1", "realm-role2").Return(&newRole2, nil)
	mockClient.On("AddRealmRoleToGroup", "realm1", "1", []gocloak.Role{newRole1, newRole2}).Return(nil)
	mockClient.On("DeleteRealmRoleFromGroup", "realm1", "1", []gocloak.Role{oldRole1, oldRole2}).Return(nil)
	mockClient.On("GetClients", "realm1",
		gocloak.GetClientsParams{ClientID: gocloak.StringP("client1")}).
		Return([]*gocloak.Client{{ID: gocloak.StringP("clid1"), ClientID: gocloak.StringP("client1")}}, nil)
	mockClient.On("GetClients", "realm1",
		gocloak.GetClientsParams{ClientID: gocloak.StringP("old-cl-3")}).
		Return([]*gocloak.Client{{ID: gocloak.StringP("3214"), ClientID: gocloak.StringP("old-cl-3")}}, nil)
	mockClient.On("GetClientRole", "realm1", "clid1", *newClientRole1.Name).Return(&newClientRole1, nil)
	mockClient.On("GetClientRole", "realm1", "clid1", *newClientRole2.Name).Return(&newClientRole2, nil)
	mockClient.On("GetClientRole", "realm1", "3214", *newClientRole4.Name).Return(&newClientRole4, nil)
	mockClient.On("AddClientRoleToGroup", "realm1", "clid1", "1",
		[]gocloak.Role{newClientRole1, newClientRole2}).Return(nil)
	mockClient.On("AddClientRoleToGroup", "realm1", "3214", "1",
		[]gocloak.Role{newClientRole4}).Return(nil)

	mockClient.On("DeleteClientRoleFromGroup", "realm1", "321", "1",
		[]gocloak.Role{oldClientRole1, oldClientRole2}).Return(nil)
	mockClient.On("DeleteClientRoleFromGroup", "realm1", "3214", "1",
		[]gocloak.Role{oldClientRole3}).Return(nil)

	groupID, err := adapter.SyncRealmGroup("realm1", &keycloakApi.KeycloakRealmGroupSpec{
		Name:       "group1",
		Attributes: map[string][]string{"foo": {"foo", "bar"}},
		Access:     map[string]bool{},
		SubGroups:  []string{"subgroup1", "subgroup2"},
		RealmRoles: []string{"realm-role1", "realm-role2"},
		ClientRoles: []keycloakApi.ClientRole{
			{ClientID: "client1", Roles: []string{"client-role1", "client-role2"}},
			{ClientID: "old-cl-3", Roles: []string{"client-role4"}},
		},
	})
	if err != nil {
		t.Fatalf("%+v", err)
	}

	if groupID == "" {
		t.Fatal("group id is empty")
	}
}

func TestGoCloakAdapter_DeleteGroup(t *testing.T) {
	mockClient := MockGoCloakClient{}
	adapter := GoCloakAdapter{
		client:   &mockClient,
		token:    &gocloak.JWT{AccessToken: "token"},
		basePath: "",
		log:      mock.NewLogr(),
	}

	mockClient.On("GetGroups", "realm1", gocloak.GetGroupsParams{Search: gocloak.StringP("group1")}).
		Return([]*gocloak.Group{{Name: gocloak.StringP("group1"), ID: gocloak.StringP("1")}}, nil)
	mockClient.On("DeleteGroup", "realm1", "1").Return(nil)

	if err := adapter.DeleteGroup(context.Background(), "realm1", "group1"); err != nil {
		t.Fatalf("%+v", err)
	}
}

func TestGoCloakAdapter_GetGoCloak(t *testing.T) {
	gcl := GoCloakAdapter{}
	if gcl.GetGoCloak() != nil {
		t.Fatal("go cloak must be nil")
	}
}

func TestMakeFromToken(t *testing.T) {
	t.Parallel()

	expiredToken := `eyJhbGciOiJIUzI1NiJ9.eyJSb2xlIjoiQWRtaW4iLCJJc3N1ZXIiOiJJc3N1ZXIiLCJVc2VybmFtZSI6IkphdmFJblVzZSIsImV4cCI6MTYzNDAzOTA2OCwiaWF0IjoxNjM0MDM5MDY4fQ.OZJDXUqfmajSh0vpqL8VnoQGqUXH25CAVkKnoyJX3AI`

	tokenParts := strings.Split(expiredToken, ".")
	rawTokenPayload, _ := base64.RawURLEncoding.DecodeString(tokenParts[1])

	var decodedTokenPayload JWTPayload
	_ = json.Unmarshal(rawTokenPayload, &decodedTokenPayload)
	decodedTokenPayload.Exp = time.Now().Unix() + 1000
	rawTokenPayload, err := json.Marshal(decodedTokenPayload)
	require.NoError(t, err)

	tokenParts[1] = base64.RawURLEncoding.EncodeToString(rawTokenPayload)
	workingToken := strings.Join(tokenParts, ".")

	tests := []struct {
		name       string
		token      string
		mockServer fakehttp.Server
		wantErr    func(require.TestingT, error, ...interface{})
	}{
		{
			name:  "should succeed",
			token: workingToken,
			mockServer: fakehttp.NewServerBuilder().
				AddStringResponder("/admin/realms/", "{}").
				BuildAndStart(),
			wantErr: func(t require.TestingT, err error, i ...interface{}) {
				require.NoError(t, err)

				cl, ok := i[0].(*GoCloakAdapter)
				require.True(t, ok)

				clientToken, _ := cl.ExportToken()

				jwtToken := gocloak.JWT{AccessToken: workingToken}
				token, err := json.Marshal(jwtToken)
				require.NoError(t, err)

				require.Equal(t, token, clientToken)
			},
		},
		{
			name:  "should succeed with legacy endpoint",
			token: workingToken,
			mockServer: fakehttp.NewServerBuilder().
				AddStringResponder("/auth/admin/realms/", "{}").
				BuildAndStart(),
			wantErr: func(t require.TestingT, err error, i ...interface{}) {
				require.NoError(t, err)

				cl, ok := i[0].(*GoCloakAdapter)
				require.True(t, ok)

				clientToken, _ := cl.ExportToken()

				jwtToken := gocloak.JWT{AccessToken: workingToken}
				token, err := json.Marshal(jwtToken)
				require.NoError(t, err)

				require.Equal(t, token, clientToken)
			},
		},
		{
			name:       "should fail on expired token",
			token:      expiredToken,
			mockServer: nil,
			wantErr: func(t require.TestingT, err error, i ...interface{}) {
				require.Error(t, err)
				require.True(t, IsErrTokenExpired(err) || err.Error() == "token is expired")
			},
		},
		{
			name:       "should fail on wrong token structure",
			token:      "foo.bar",
			mockServer: nil,
			wantErr: func(t require.TestingT, err error, i ...interface{}) {
				require.Error(t, err)
				require.Contains(t, err.Error(), "wrong JWT token structure")
			},
		},
		{
			name:       "should fail on wrong token encoding",
			token:      "foo.bar .baz",
			mockServer: nil,
			wantErr: func(t require.TestingT, err error, i ...interface{}) {
				require.Error(t, err)
				require.Contains(t, err.Error(), "wrong JWT token base64 encoding")
			},
		},
		{
			name:       "should fail on decoding json payload",
			token:      "foo.bar.baz",
			mockServer: nil,
			wantErr: func(t require.TestingT, err error, i ...interface{}) {
				require.Error(t, err)
				require.Contains(t, err.Error(), "unable to decode JWT payload json")
			},
		},
	}

	for _, tt := range tests {
		tt := tt

		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			jwtToken := gocloak.JWT{AccessToken: tt.token}
			token, err := json.Marshal(jwtToken)
			require.NoError(t, err)

			url := "test_url"
			if tt.mockServer != nil {
				url = tt.mockServer.GetURL()

				defer tt.mockServer.Close()
			}

			cl, err := MakeFromToken(url, token, mock.NewLogr())
			tt.wantErr(t, err, cl)
		})
	}
}

func TestMakeFromToken_invalidJSON(t *testing.T) {
	t.Parallel()

	_, err := MakeFromToken("test_url", []byte("qwdqwdwq"), mock.NewLogr())
	require.Error(t, err)
	require.Contains(t, err.Error(), "invalid character")
}

func TestGoCloakAdapter_CreateCentralIdentityProvider(t *testing.T) {
	mockClient := MockGoCloakClient{}
	restyClient := resty.New()
	httpmock.ActivateNonDefault(restyClient.GetClient())
	mockClient.On("RestyClient").Return(restyClient)

	a := GoCloakAdapter{
		log:    mock.NewLogr(),
		client: &mockClient,
		token:  &gocloak.JWT{AccessToken: "token"},
	}
	realm := dto.Realm{Name: "name1", SsoRealmName: "sso-realm1"}

	httpmock.RegisterResponder("POST",
		fmt.Sprintf("/admin/realms/%s/identity-provider/instances", realm.Name),
		httpmock.NewStringResponder(201, ""))

	httpmock.RegisterResponder("POST",
		fmt.Sprintf("/admin/realms/%s/identity-provider/instances/%s/mappers", realm.Name, realm.SsoRealmName),
		httpmock.NewStringResponder(201, ""))

	err := a.CreateCentralIdentityProvider(&realm, &dto.Client{})
	assert.NoError(t, err)

	httpmock.RegisterResponder("POST",
		fmt.Sprintf("/admin/realms/%s/identity-provider/instances/%s/mappers", realm.Name, realm.SsoRealmName),
		httpmock.NewStringResponder(500, "fatal"))

	err = a.CreateCentralIdentityProvider(&realm, &dto.Client{})
	assert.Error(t, err)
	assert.EqualError(t, err,
		"unable to create central idp mappers: unable to create central idp mapper: error in creation idP mapper by name administrator")
}

func (e *AdapterTestSuite) TestGoCloakAdapter_DeleteRealmUser() {
	username := "username"
	httpmock.RegisterResponder("DELETE",
		fmt.Sprintf("/admin/realms/%s/users/%s", e.realmName, username),
		httpmock.NewStringResponder(200, ""))
	e.goCloakMockClient.On("GetUsers", e.realmName, gocloak.GetUsersParams{Username: &username}).
		Return([]*gocloak.User{
			{Username: &username, ID: &username},
		}, nil).Once()

	err := e.adapter.DeleteRealmUser(context.Background(), e.realmName, username)
	assert.NoError(e.T(), err)

	e.goCloakMockClient.On("GetUsers", e.realmName, gocloak.GetUsersParams{Username: &username}).
		Return([]*gocloak.User{
			{Username: &username, ID: &username},
		}, nil).Once()
	httpmock.RegisterResponder("DELETE",
		fmt.Sprintf("/admin/realms/%s/users/%s", e.realmName, username),
		httpmock.NewStringResponder(404, ""))

	err = e.adapter.DeleteRealmUser(context.Background(), e.realmName, username)
	assert.Error(e.T(), err)
	assert.EqualError(e.T(), err, "unable to delete user: status: 404, body: ")

	e.goCloakMockClient.On("GetUsers", e.realmName, gocloak.GetUsersParams{Username: &username}).
		Return([]*gocloak.User{
			{},
		}, nil).Once()

	err = e.adapter.DeleteRealmUser(context.Background(), e.realmName, username)
	assert.Error(e.T(), err)
	assert.EqualError(e.T(), err, "user not found")

	e.goCloakMockClient.On("GetUsers", e.realmName, gocloak.GetUsersParams{Username: &username}).
		Return(nil, errors.New("fatal get users")).Once()

	err = e.adapter.DeleteRealmUser(context.Background(), e.realmName, username)
	assert.Error(e.T(), err)
	assert.EqualError(e.T(), err, "unable to get users: fatal get users")
}

func TestGoCloakAdapter_PutDefaultIdp(t *testing.T) {
	mockClient := MockGoCloakClient{}
	adapter := GoCloakAdapter{
		client:   &mockClient,
		token:    &gocloak.JWT{AccessToken: "token"},
		basePath: "",
		log:      mock.NewLogr(),
	}

	realm := dto.Realm{Name: "realm1", SsoAutoRedirectEnabled: false}

	restyClient := resty.New()
	httpmock.ActivateNonDefault(restyClient.GetClient())
	mockClient.On("RestyClient").Return(restyClient)

	authExecs := []api.SimpleAuthExecution{{
		ProviderId: "identity-provider-redirector",
		Id:         "id1",
	}, {}}
	authExecsRsp, err := httpmock.NewJsonResponder(200, authExecs)
	require.NoError(t, err)

	httpmock.RegisterResponder(
		"GET",
		fmt.Sprintf("/admin/realms/%s/authentication/flows/browser/executions", realm.Name),
		authExecsRsp)

	httpmock.RegisterResponder("POST",
		fmt.Sprintf("/admin/realms/%s/authentication/executions/%s/config", realm.Name, authExecs[0].Id),
		httpmock.NewStringResponder(201, "ok"))

	httpmock.RegisterResponder("PUT",
		fmt.Sprintf("/admin/realms/%s/authentication/flows/browser/executions", realm.Name),
		httpmock.NewStringResponder(202, "ok"))

	if err := adapter.PutDefaultIdp(&realm); err != nil {
		t.Fatalf("%+v", err)
	}
}

func TestGoCloakAdapter_PutDefaultIdp1(t *testing.T) {
	t.Parallel()

	const (
		realm       = "realm1"
		executionID = "executionID"
		configID    = "configID1"
	)

	authExecutionsEndpoint := strings.ReplaceAll(authExecutions, "{realm}", realm)
	authExecutionsConfigEndpoint := strings.ReplaceAll(
		strings.ReplaceAll(authExecutionConfig, "{realm}", realm),
		"{id}",
		executionID,
	)
	authFlowConfigEndpoint := strings.ReplaceAll(
		strings.ReplaceAll(authFlowConfig, "{realm}", "realm1"),
		"{id}",
		configID,
	)

	tests := []struct {
		name       string
		realm      *dto.Realm
		mockServer fakehttp.Server
		wantErr    require.ErrorAssertionFunc
	}{
		{
			name: "update default identity provider config",
			realm: &dto.Realm{
				Name: realm,
			},
			mockServer: fakehttp.NewServerBuilder().
				AddJsonResponderWithCode(
					http.StatusOK,
					authExecutionsEndpoint,
					[]api.SimpleAuthExecution{
						{
							Id:                   executionID,
							ProviderId:           "identity-provider-redirector",
							AuthenticationConfig: configID,
						},
					},
				).
				AddStringResponder(authFlowConfigEndpoint, "").
				BuildAndStart(),
			wantErr: require.NoError,
		},
		{
			name: "create default identity provider config",
			realm: &dto.Realm{
				Name:                   realm,
				SsoAutoRedirectEnabled: true,
			},
			mockServer: fakehttp.NewServerBuilder().
				AddJsonResponderWithCode(
					http.StatusOK,
					authExecutionsEndpoint,
					[]api.SimpleAuthExecution{
						{
							Id:         executionID,
							ProviderId: "identity-provider-redirector",
						},
					},
				).
				AddStringResponderWithCode(http.StatusCreated, authExecutionsConfigEndpoint, "").
				BuildAndStart(),
			wantErr: require.NoError,
		},
		{
			name: "failed to update default identity provider config",
			realm: &dto.Realm{
				Name: realm,
			},
			mockServer: fakehttp.NewServerBuilder().
				AddJsonResponderWithCode(
					http.StatusOK,
					authExecutionsEndpoint,
					[]api.SimpleAuthExecution{
						{
							Id:                   executionID,
							ProviderId:           "identity-provider-redirector",
							AuthenticationConfig: configID,
						},
					},
				).
				AddStringResponderWithCode(http.StatusBadRequest, authFlowConfigEndpoint, "").
				BuildAndStart(),
			wantErr: func(t require.TestingT, err error, i ...interface{}) {
				require.Error(t, err)
				require.Contains(t, err.Error(), "failed to update redirect config")
			},
		},
	}

	for _, tt := range tests {
		tt := tt

		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			defer tt.mockServer.Close()

			a := GoCloakAdapter{
				client: gocloak.NewClient(tt.mockServer.GetURL()),
				log:    logr.Discard(),
				token: &gocloak.JWT{
					AccessToken: "token",
				},
				basePath: tt.mockServer.GetURL(),
			}

			err := a.PutDefaultIdp(tt.realm)

			tt.wantErr(t, err)
		})
	}
}

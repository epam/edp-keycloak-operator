package adapter

import (
	"fmt"
	"net/url"
	"testing"

	"github.com/Nerzal/gocloak/v8"
	"github.com/epam/edp-keycloak-operator/pkg/apis/v1/v1alpha1"
	"github.com/epam/edp-keycloak-operator/pkg/client/keycloak/api"
	"github.com/epam/edp-keycloak-operator/pkg/client/keycloak/dto"
	"github.com/epam/edp-keycloak-operator/pkg/client/keycloak/mock"
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
		token:  &gocloak.JWT{AccessToken: "token"},
		log:    &mock.Logger{},
	}
	realm := dto.Realm{
		Name: "realmName",
	}

	//test
	res, err := adapter.ExistRealm(realm.Name)

	//verify
	assert.NoError(t, err)
	assert.True(t, res)
}

func TestGoCloakAdapter_ExistRealm404(t *testing.T) {
	//prepare
	mockClient := new(MockGoCloakClient)
	mockClient.On("GetRealm", "token", "realmName").
		Return(nil, errors.New("404"))
	adapter := GoCloakAdapter{
		client: mockClient,
		token:  &gocloak.JWT{AccessToken: "token"},
		log:    &mock.Logger{},
	}
	realm := dto.Realm{
		Name: "realmName",
	}

	//test
	res, err := adapter.ExistRealm(realm.Name)

	//verify
	assert.NoError(t, err)
	assert.False(t, res)
}

func TestGoCloakAdapter_ExistRealmError(t *testing.T) {
	//prepare
	mockClient := new(MockGoCloakClient)
	mockClient.On("GetRealm", "token", "realmName").
		Return(nil, errors.New("error in get realm"))
	adapter := GoCloakAdapter{
		client: mockClient,
		token:  &gocloak.JWT{AccessToken: "token"},
		log:    &mock.Logger{},
	}
	realm := dto.Realm{
		Name: "realmName",
	}

	//test
	res, err := adapter.ExistRealm(realm.Name)

	//verify
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
		fmt.Sprintf("/auth/admin/realms/%s/clients/%s/protocol-mappers/models", client.RealmName, clientID),
		responder)

	adapter := GoCloakAdapter{
		client:   mockClient,
		token:    &gocloak.JWT{AccessToken: "token"},
		basePath: "",
	}

	_, err := adapter.GetClientProtocolMappers(&client, clientID)
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
		token:    &gocloak.JWT{AccessToken: "token"},
		basePath: "",
	}

	_, err := adapter.GetClientProtocolMappers(&client, clientID)
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
		token:    &gocloak.JWT{AccessToken: "token"},
		basePath: "",
		log:      &mock.Logger{},
	}

	if err := adapter.SyncClientProtocolMapper(&client, crMappers); err != nil {
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
		token:    &gocloak.JWT{AccessToken: "token"},
		basePath: "",
		log:      &mock.Logger{},
	}

	err := adapter.SyncClientProtocolMapper(&client, []gocloak.ProtocolMapperRepresentation{})
	if err == nil {
		t.Fatal("no error on get clients fatal")
	}

	if errors.Cause(err) != mockErr {
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
		log:      &mock.Logger{},
	}

	role := dto.PrimaryRealmRole{Name: roleName, Composites: []string{"foo", "bar"}, IsComposite: true,
		Attributes: map[string][]string{
			"foo": []string{"foo", "bar"},
		}, IsDefault: true}

	if err := adapter.SyncRealmRole(realmName, &role); err != nil {
		t.Fatal(err)
	}
}

func TestGoCloakAdapter_SyncServiceAccountRoles(t *testing.T) {
	mockClient := MockGoCloakClient{}
	adapter := GoCloakAdapter{
		client:   &mockClient,
		token:    &gocloak.JWT{AccessToken: "token"},
		basePath: "",
		log:      &mock.Logger{},
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
		}); err != nil {
		t.Fatal(err)
	}
}

func TestGoCloakAdapter_SyncRealmGroup(t *testing.T) {
	mockClient := MockGoCloakClient{}
	adapter := GoCloakAdapter{
		client:   &mockClient,
		token:    &gocloak.JWT{AccessToken: "token"},
		basePath: "",
		log:      &mock.Logger{},
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

	groupID, err := adapter.SyncRealmGroup("realm1", &v1alpha1.KeycloakRealmGroupSpec{
		Name:       "group1",
		Attributes: map[string][]string{"foo": {"foo", "bar"}},
		Access:     map[string]bool{},
		SubGroups:  []string{"subgroup1", "subgroup2"},
		RealmRoles: []string{"realm-role1", "realm-role2"},
		ClientRoles: []v1alpha1.ClientRole{
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
		log:      &mock.Logger{},
	}
	mockClient.On("GetGroups", "realm1", gocloak.GetGroupsParams{Search: gocloak.StringP("group1")}).
		Return([]*gocloak.Group{{Name: gocloak.StringP("group1"), ID: gocloak.StringP("1")}}, nil)
	mockClient.On("DeleteGroup", "realm1", "1").Return(nil)

	if err := adapter.DeleteGroup("realm1", "group1"); err != nil {
		t.Fatalf("%+v", err)
	}
}

func TestGoCloakAdapter_PutDefaultIdp(t *testing.T) {
	mockClient := MockGoCloakClient{}
	adapter := GoCloakAdapter{
		client:   &mockClient,
		token:    &gocloak.JWT{AccessToken: "token"},
		basePath: "",
		log:      &mock.Logger{},
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
	if err != nil {
		t.Fatal(err)
	}

	httpmock.RegisterResponder(
		"GET",
		fmt.Sprintf("/auth/admin/realms/%s/authentication/flows/browser/executions", realm.Name),
		authExecsRsp)

	httpmock.RegisterResponder("POST",
		fmt.Sprintf("/auth/admin/realms/%s/authentication/executions/%s/config", realm.Name, authExecs[0].Id),
		httpmock.NewStringResponder(201, "ok"))

	httpmock.RegisterResponder("PUT",
		fmt.Sprintf("/auth/admin/realms/%s/authentication/flows/browser/executions", realm.Name),
		httpmock.NewStringResponder(202, "ok"))

	if err := adapter.PutDefaultIdp(&realm); err != nil {
		t.Fatalf("%+v", err)
	}
}

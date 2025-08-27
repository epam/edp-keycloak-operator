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
	testifymock "github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	ctrl "sigs.k8s.io/controller-runtime"

	"github.com/epam/edp-keycloak-operator/pkg/client/keycloak/adapter/mocks"
	"github.com/epam/edp-keycloak-operator/pkg/client/keycloak/dto"
	"github.com/epam/edp-keycloak-operator/pkg/client/keycloak/mock"
	"github.com/epam/edp-keycloak-operator/pkg/fakehttp"
)

type AdapterTestSuite struct {
	suite.Suite
	restyClient       *resty.Client
	goCloakMockClient *mocks.MockGoCloak
	adapter           *GoCloakAdapter
	realmName         string
}

func (e *AdapterTestSuite) SetupTest() {
	e.restyClient = resty.New()

	httpmock.Reset()
	httpmock.ActivateNonDefault(e.restyClient.GetClient())

	e.goCloakMockClient = mocks.NewMockGoCloak(e.T())
	e.goCloakMockClient.On("RestyClient").Return(e.restyClient).Maybe()

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
				AddStringResponder(authPath+realmsEndpoint, "{}").
				BuildAndStart(),
			wantErr: require.NoError,
		},
		{
			name: "should fail on status bad request",
			mockServer: fakehttp.NewServerBuilder().
				AddStringResponderWithCode(http.StatusBadRequest, authPath+realmsEndpoint, "{}").
				BuildAndStart(),
			wantErr: func(t require.TestingT, err error, _ ...interface{}) {
				require.Error(t, err)
				require.EqualError(
					t,
					err,
					"failed to login with client creds on both current and legacy clients - "+
						"clientID: k-cl-id, realm: master: 400 Bad Request",
				)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			defer tt.mockServer.Close()

			_, err := MakeFromServiceAccount(
				context.Background(),
				GoCloakConfig{
					Url:      tt.mockServer.GetURL(),
					User:     "k-cl-id",
					Password: "k-secret",
				},

				"master",
				mock.NewLogr(),
				resty.New(),
			)
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
				AddStringResponder(authPath+realmsEndpoint, "{}").
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
				AddStringResponderWithCode(http.StatusBadRequest, authPath+realmsEndpoint, "{}").
				BuildAndStart(),
			wantErr: func(t require.TestingT, err error, i ...interface{}) {
				require.Error(t, err)
				require.Contains(t, err.Error(), "400")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			url := "test_url"
			if tt.mockServer != nil {
				url = tt.mockServer.GetURL()

				defer tt.mockServer.Close()
			}

			_, err := Make(
				context.Background(),
				GoCloakConfig{
					Url:      url,
					User:     "bar",
					Password: "baz",
				},
				mock.NewLogr(),
				resty.New(),
			)
			tt.wantErr(t, err)
		})
	}
}

func (e *AdapterTestSuite) TestGoCloakAdapter_ExistRealmPositive() {
	e.goCloakMockClient.On("GetRealm", testifymock.Anything, "token", "realmName").
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
	mockClient := mocks.NewMockGoCloak(t)
	mockClient.On("GetRealm", testifymock.Anything, "token", "realmName").
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
	mockClient := mocks.NewMockGoCloak(t)
	mockClient.On("GetRealm", testifymock.Anything, "token", "realmName").
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
	mockClient := mocks.NewMockGoCloak(t)
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
	mockClient := mocks.NewMockGoCloak(t)
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
	mockClient := mocks.NewMockGoCloak(t)
	logger := mock.NewLogr()

	cl := dto.Client{
		RedirectUris: []string{"https://test.com"},
	}
	a := GoCloakAdapter{
		client: mockClient,
		token:  &gocloak.JWT{AccessToken: "token"},
		log:    logger,
	}

	mockClient.On("CreateClient", testifymock.Anything, "token", "", getGclCln(&cl)).Return("id", nil).Once()

	err := a.CreateClient(context.Background(), &cl)
	assert.NoError(t, err)

	createErr := errors.New("create-err")
	mockClient.On("CreateClient", testifymock.Anything, "token", "", getGclCln(&cl)).Return("", createErr).Once()
	err = a.CreateClient(context.Background(), &cl)

	assert.ErrorIs(t, err, createErr)
}

func TestGoCloakAdapter_UpdateClient(t *testing.T) {
	mockClient := mocks.NewMockGoCloak(t)
	logger := mock.NewLogr()

	cl := dto.Client{}
	a := GoCloakAdapter{
		client: mockClient,
		token:  &gocloak.JWT{AccessToken: "token"},
		log:    logger,
	}

	mockClient.On("UpdateClient", testifymock.Anything, a.token.AccessToken, cl.RealmName,
		getGclCln(&cl)).Return(nil).Once()

	err := a.UpdateClient(context.Background(), &cl)
	assert.NoError(t, err)

	updErr := errors.New("update-error")

	mockClient.On("UpdateClient", testifymock.Anything, a.token.AccessToken, cl.RealmName,
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

	mockClient := mocks.NewMockGoCloak(t)
	restyClient := resty.New()
	httpmock.ActivateNonDefault(restyClient.GetClient())
	mockClient.On("RestyClient").Return(restyClient)
	mockClient.On("GetClients", testifymock.Anything, "token", client.RealmName, gocloak.GetClientsParams{
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

	mockClient.On(
		"DeleteClientProtocolMapper",
		testifymock.Anything,
		"token",
		client.RealmName,
		clientID,
		*kcMappers[0].ID,
	).
		Return(nil)

	mockClient.On(
		"UpdateClientProtocolMapper",
		testifymock.Anything,
		"token",
		client.RealmName,
		clientID,
		*crMappers[0].ID,
		crMappers[0],
	).
		Return(nil)

	mockClient.On("CreateClientProtocolMapper", testifymock.Anything, "token", client.RealmName, clientID, crMappers[1]).
		Return("", nil)

	mockClient.On("CreateClientProtocolMapper", testifymock.Anything, "token", client.RealmName, clientID,
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

	mockClient := mocks.NewMockGoCloak(t)
	mockClient.On("GetClients", testifymock.Anything, "token", client.RealmName, gocloak.GetClientsParams{
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

func TestGoCloakAdapter_DeleteGroup(t *testing.T) {
	mockClient := mocks.NewMockGoCloak(t)
	adapter := GoCloakAdapter{
		client:   mockClient,
		token:    &gocloak.JWT{AccessToken: "token"},
		basePath: "",
		log:      mock.NewLogr(),
	}

	mockClient.On(
		"GetGroups",
		testifymock.Anything,
		"token",
		"realm1",
		gocloak.GetGroupsParams{Search: gocloak.StringP("group1")},
	).
		Return([]*gocloak.Group{{Name: gocloak.StringP("group1"), ID: gocloak.StringP("1")}}, nil)
	mockClient.On("DeleteGroup", testifymock.Anything, "token", "realm1", "1").Return(nil)

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

	expiredToken := `eyJhbGciOiJIUzI1NiJ9.` +
		`eyJSb2xlIjoiQWRtaW4iLCJJc3N1ZXIiOiJJc3N1ZXIiLCJVc2VybmFtZSI6IkphdmFJblV` +
		`zZSIsImV4cCI6MTYzNDAzOTA2OCwiaWF0IjoxNjM0MDM5MDY4fQ.` +
		`OZJDXUqfmajSh0vpqL8VnoQGqUXH25CAVkKnoyJX3AI`

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

			cl, err := MakeFromToken(GoCloakConfig{Url: url}, token, mock.NewLogr())
			tt.wantErr(t, err, cl)
		})
	}
}

func TestMakeFromToken_invalidJSON(t *testing.T) {
	t.Parallel()

	_, err := MakeFromToken(GoCloakConfig{Url: "test_url"}, []byte("qwdqwdwq"), mock.NewLogr())
	require.Error(t, err)
	require.Contains(t, err.Error(), "invalid character")
}

func (e *AdapterTestSuite) TestGoCloakAdapter_DeleteRealmUser() {
	username := "username"
	httpmock.RegisterResponder("DELETE",
		fmt.Sprintf("/admin/realms/%s/users/%s", e.realmName, username),
		httpmock.NewStringResponder(200, ""))
	e.goCloakMockClient.On(
		"GetUsers",
		testifymock.Anything,
		"token",
		e.realmName,
		gocloak.GetUsersParams{Username: &username},
	).
		Return([]*gocloak.User{
			{Username: &username, ID: &username},
		}, nil).Once()

	err := e.adapter.DeleteRealmUser(context.Background(), e.realmName, username)
	assert.NoError(e.T(), err)

	e.goCloakMockClient.On(
		"GetUsers",
		testifymock.Anything,
		"token",
		e.realmName,
		gocloak.GetUsersParams{Username: &username},
	).
		Return([]*gocloak.User{
			{Username: &username, ID: &username},
		}, nil).Once()
	httpmock.RegisterResponder("DELETE",
		fmt.Sprintf("/admin/realms/%s/users/%s", e.realmName, username),
		httpmock.NewStringResponder(404, ""))

	err = e.adapter.DeleteRealmUser(context.Background(), e.realmName, username)
	assert.Error(e.T(), err)
	assert.EqualError(e.T(), err, "unable to delete user: status: 404, body: ")

	e.goCloakMockClient.On(
		"GetUsers",
		testifymock.Anything,
		"token",
		e.realmName,
		gocloak.GetUsersParams{Username: &username},
	).
		Return([]*gocloak.User{
			{},
		}, nil).Once()

	err = e.adapter.DeleteRealmUser(context.Background(), e.realmName, username)
	assert.Error(e.T(), err)
	assert.EqualError(e.T(), err, "user not found")

	e.goCloakMockClient.On(
		"GetUsers",
		testifymock.Anything,
		"token",
		e.realmName,
		gocloak.GetUsersParams{Username: &username},
	).
		Return(nil, errors.New("fatal get users")).Once()

	err = e.adapter.DeleteRealmUser(context.Background(), e.realmName, username)
	assert.Error(e.T(), err)
	assert.EqualError(e.T(), err, "unable to get users: fatal get users")
}

func TestGoCloakAdapter_GetUsersByNames(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		client  func(t *testing.T) GoCloak
		names   []string
		wantErr require.ErrorAssertionFunc
		want    map[string]gocloak.User
	}{
		{
			name: "should return users",
			client: func(t *testing.T) GoCloak {
				mockClient := mocks.NewMockGoCloak(t)
				param := gocloak.GetUsersParams{
					BriefRepresentation: gocloak.BoolP(true),
					Max:                 gocloak.IntP(100),
					Username:            gocloak.StringP("user1"),
				}
				mockClient.On(
					"GetUsers", testifymock.Anything, "token", "master", param).
					Return([]*gocloak.User{
						{Username: gocloak.StringP("user1")},
					}, nil)
				param2 := gocloak.GetUsersParams{
					BriefRepresentation: gocloak.BoolP(true),
					Max:                 gocloak.IntP(100),
					Username:            gocloak.StringP("user2"),
				}
				param.Username = gocloak.StringP("user2")
				mockClient.On("GetUsers", testifymock.Anything, "token", "master", param2).
					Return([]*gocloak.User{
						{Username: gocloak.StringP("user2")},
					}, nil)
				param3 := gocloak.GetUsersParams{
					BriefRepresentation: gocloak.BoolP(true),
					Max:                 gocloak.IntP(100),
					Username:            gocloak.StringP("user3"),
				}
				param3.Username = gocloak.StringP("user3")
				mockClient.On("GetUsers", testifymock.Anything, "token", "master", param3).
					Return(nil, nil)

				return mockClient
			},
			names:   []string{"user1", "user2", "user3"},
			wantErr: require.NoError,
			want: map[string]gocloak.User{
				"user1": {Username: gocloak.StringP("user1")},
				"user2": {Username: gocloak.StringP("user2")},
			},
		},
		{
			name: "keycloak api error",
			client: func(t *testing.T) GoCloak {
				mockClient := mocks.NewMockGoCloak(t)
				param := gocloak.GetUsersParams{
					BriefRepresentation: gocloak.BoolP(true),
					Max:                 gocloak.IntP(100),
					Username:            gocloak.StringP("user1"),
				}
				mockClient.On(
					"GetUsers", testifymock.Anything, "token", "master", param).
					Return([]*gocloak.User{
						{Username: gocloak.StringP("user1")},
					}, nil)
				param2 := gocloak.GetUsersParams{
					BriefRepresentation: gocloak.BoolP(true),
					Max:                 gocloak.IntP(100),
					Username:            gocloak.StringP("user2"),
				}
				param.Username = gocloak.StringP("user2")
				mockClient.On("GetUsers", testifymock.Anything, "token", "master", param2).
					Return(nil, errors.New("fatal"))

				return mockClient
			},
			names: []string{"user1", "user2"},
			wantErr: func(t require.TestingT, err error, i ...interface{}) {
				require.Error(t, err)
				require.Contains(t, err.Error(), "fatal")
			},
			want: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			a := GoCloakAdapter{
				client: tt.client(t),
				token:  &gocloak.JWT{AccessToken: "token"},
				log:    logr.Discard(),
			}

			got, err := a.GetUsersByNames(context.Background(), "master", tt.names)

			tt.wantErr(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestGoCloakAdapter_CreatePrimaryRealmRole(t *testing.T) {
	t.Parallel()

	var (
		token = "token"
		realm = "realm"
	)

	tests := []struct {
		name    string
		role    *dto.PrimaryRealmRole
		client  func(t *testing.T) *mocks.MockGoCloak
		want    string
		wantErr require.ErrorAssertionFunc
	}{
		{
			name: "should create role successfully",
			role: &dto.PrimaryRealmRole{
				Name:        "role1",
				Description: "Role description",
			},
			client: func(t *testing.T) *mocks.MockGoCloak {
				m := mocks.NewMockGoCloak(t)

				m.On(
					"CreateRealmRole",
					testifymock.Anything,
					token,
					realm,
					testifymock.MatchedBy(func(role gocloak.Role) bool {
						return assert.Equal(t, "role1", *role.Name) &&
							assert.Equal(t, "Role description", *role.Description)
					})).
					Return("", nil)

				m.On("GetRealmRole", testifymock.Anything, token, realm, testifymock.Anything).
					Return(&gocloak.Role{Name: gocloak.StringP("role1"), ID: gocloak.StringP("role1-id")}, nil)

				return m
			},
			want:    "role1-id",
			wantErr: require.NoError,
		},
		{
			name: "should fail to get role",
			role: &dto.PrimaryRealmRole{
				Name:        "role1",
				Description: "Role description",
			},
			client: func(t *testing.T) *mocks.MockGoCloak {
				m := mocks.NewMockGoCloak(t)

				m.On(
					"CreateRealmRole",
					testifymock.Anything,
					token,
					realm,
					testifymock.MatchedBy(func(role gocloak.Role) bool {
						return assert.Equal(t, "role1", *role.Name) &&
							assert.Equal(t, "Role description", *role.Description)
					})).
					Return("", nil)

				m.On("GetRealmRole", testifymock.Anything, token, realm, testifymock.Anything).
					Return(nil, errors.New("failed to get role"))

				return m
			},
			want: "",
			wantErr: func(t require.TestingT, err error, i ...interface{}) {
				require.Error(t, err)
				require.Contains(t, err.Error(), "failed to get role")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			a := GoCloakAdapter{
				client: tt.client(t),
				token:  &gocloak.JWT{AccessToken: token},
				log:    logr.Discard(),
			}

			got, err := a.CreatePrimaryRealmRole(ctrl.LoggerInto(context.Background(), logr.Discard()), realm, tt.role)
			tt.wantErr(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

// setupBasicClientRoleMocks sets up the basic mock calls for getting client and roles
func setupBasicClientRoleMocks(m *mocks.MockGoCloak, token, realmName, clientID string) {
	// Mock GetClientID
	m.On("GetClients", testifymock.Anything, token, realmName, testifymock.Anything).
		Return([]*gocloak.Client{
			{
				ID:       gocloak.StringP(clientID),
				ClientID: gocloak.StringP("test-client"),
			},
		}, nil).Once()

	// Mock getExistingClientRolesMap
	m.On("GetClientRoles", testifymock.Anything, token, realmName, clientID, testifymock.Anything).
		Return([]*gocloak.Role{
			{
				ID:          gocloak.StringP("role1-id"),
				Name:        gocloak.StringP("role1"),
				Description: gocloak.StringP("Role 1 description"),
			},
		}, nil).Once()

	// Mock syncClientRoleComposites - get existing roles again
	m.On("GetClientRoles", testifymock.Anything, token, realmName, clientID, testifymock.Anything).
		Return([]*gocloak.Role{
			{
				ID:          gocloak.StringP("role1-id"),
				Name:        gocloak.StringP("role1"),
				Description: gocloak.StringP("Role 1 description"),
			},
		}, nil).Once()
}

// setupCompositeRoleRemovalMocks sets up mocks for composite role removal scenarios
func setupCompositeRoleRemovalMocks(m *mocks.MockGoCloak, token, realmName string) {
	// Mock GetCompositeRolesByRoleID - return existing composite roles
	m.On("GetCompositeRolesByRoleID", testifymock.Anything, token, realmName, "role1-id").
		Return([]*gocloak.Role{
			{
				ID:   gocloak.StringP("composite-role1-id"),
				Name: gocloak.StringP("composite-role1"),
			},
			{
				ID:   gocloak.StringP("composite-role2-id"),
				Name: gocloak.StringP("composite-role2"),
			},
		}, nil).Once()

	// Mock DeleteClientRoleComposite for removing composite roles
	m.On("DeleteClientRoleComposite", testifymock.Anything, token, realmName, "role1-id", testifymock.Anything).
		Return(nil).Once()
}

func TestGoCloakAdapter_SyncClientRoles(t *testing.T) {
	t.Parallel()

	var (
		token     = "test-token"
		realmName = "test-realm"
		clientID  = "test-client-id"
	)

	tests := []struct {
		name          string
		client        *dto.Client
		setupMocks    func(*mocks.MockGoCloak)
		expectedError string
		expectedCalls map[string]int
	}{
		{
			name: "should successfully sync client roles - create new roles",
			client: &dto.Client{
				ClientId: "test-client",
				Roles: []dto.ClientRole{
					{
						Name:        "role1",
						Description: "Role 1 description",
					},
					{
						Name:        "role2",
						Description: "Role 2 description",
					},
				},
			},
			setupMocks: func(m *mocks.MockGoCloak) {
				// Mock GetClientID
				m.On("GetClients", testifymock.Anything, token, realmName, testifymock.Anything).
					Return([]*gocloak.Client{
						{
							ID:       gocloak.StringP(clientID),
							ClientID: gocloak.StringP("test-client"),
						},
					}, nil).Once()

				// Mock getExistingClientRolesMap - no existing roles
				m.On("GetClientRoles", testifymock.Anything, token, realmName, clientID, testifymock.Anything).
					Return([]*gocloak.Role{}, nil).Once()

				// Mock CreateClientRole for both roles
				m.On("CreateClientRole", testifymock.Anything, token, realmName, clientID, testifymock.Anything).
					Return("role1-id", nil).Once()
				m.On("CreateClientRole", testifymock.Anything, token, realmName, clientID, testifymock.Anything).
					Return("role2-id", nil).Once()

				// Mock syncClientRoleComposites - get existing roles again
				m.On("GetClientRoles", testifymock.Anything, token, realmName, clientID, testifymock.Anything).
					Return([]*gocloak.Role{
						{
							ID:   gocloak.StringP("role1-id"),
							Name: gocloak.StringP("role1"),
						},
						{
							ID:   gocloak.StringP("role2-id"),
							Name: gocloak.StringP("role2"),
						},
					}, nil).Once()

				// Mock GetCompositeRolesByRoleID for both roles
				m.On("GetCompositeRolesByRoleID", testifymock.Anything, token, realmName, "role1-id").
					Return([]*gocloak.Role{}, nil).Once()
				m.On("GetCompositeRolesByRoleID", testifymock.Anything, token, realmName, "role2-id").
					Return([]*gocloak.Role{}, nil).Once()
			},
			expectedError: "",
			expectedCalls: map[string]int{
				"GetClients":                1,
				"GetClientRoles":            2,
				"CreateClientRole":          2,
				"GetCompositeRolesByRoleID": 2,
			},
		},
		{
			name: "should successfully sync client roles - update existing roles",
			client: &dto.Client{
				ClientId: "test-client",
				Roles: []dto.ClientRole{
					{
						Name:        "role1",
						Description: "Updated Role 1 description",
					},
				},
			},
			setupMocks: func(m *mocks.MockGoCloak) {
				// Mock GetClientID
				m.On("GetClients", testifymock.Anything, token, realmName, testifymock.Anything).
					Return([]*gocloak.Client{
						{
							ID:       gocloak.StringP(clientID),
							ClientID: gocloak.StringP("test-client"),
						},
					}, nil).Once()

				// Mock getExistingClientRolesMap - existing role with different description
				m.On("GetClientRoles", testifymock.Anything, token, realmName, clientID, testifymock.Anything).
					Return([]*gocloak.Role{
						{
							ID:          gocloak.StringP("role1-id"),
							Name:        gocloak.StringP("role1"),
							Description: gocloak.StringP("Old description"),
						},
					}, nil).Once()

				// Mock UpdateRole for existing role
				m.On("UpdateRole", testifymock.Anything, token, realmName, clientID, testifymock.Anything).
					Return(nil).Once()

				// Mock syncClientRoleComposites - get existing roles again
				m.On("GetClientRoles", testifymock.Anything, token, realmName, clientID, testifymock.Anything).
					Return([]*gocloak.Role{
						{
							ID:   gocloak.StringP("role1-id"),
							Name: gocloak.StringP("role1"),
						},
					}, nil).Once()

				// Mock GetCompositeRolesByRoleID
				m.On("GetCompositeRolesByRoleID", testifymock.Anything, token, realmName, "role1-id").
					Return([]*gocloak.Role{}, nil).Once()
			},
			expectedError: "",
			expectedCalls: map[string]int{
				"GetClients":                1,
				"GetClientRoles":            2,
				"UpdateRole":                1,
				"GetCompositeRolesByRoleID": 1,
			},
		},
		{
			name: "should successfully sync client roles - delete removed roles",
			client: &dto.Client{
				ClientId: "test-client",
				Roles: []dto.ClientRole{
					{
						Name:        "role1",
						Description: "Role 1 description",
					},
				},
			},
			setupMocks: func(m *mocks.MockGoCloak) {
				// Mock GetClientID
				m.On("GetClients", testifymock.Anything, token, realmName, testifymock.Anything).
					Return([]*gocloak.Client{
						{
							ID:       gocloak.StringP(clientID),
							ClientID: gocloak.StringP("test-client"),
						},
					}, nil).Once()

				// Mock getExistingClientRolesMap - existing roles including one to be deleted
				m.On("GetClientRoles", testifymock.Anything, token, realmName, clientID, testifymock.Anything).
					Return([]*gocloak.Role{
						{
							ID:          gocloak.StringP("role1-id"),
							Name:        gocloak.StringP("role1"),
							Description: gocloak.StringP("Role 1 description"), // Same description, no update needed
						},
						{
							ID:   gocloak.StringP("role2-id"),
							Name: gocloak.StringP("role2"), // This role should be deleted
						},
					}, nil).Once()

				// Mock DeleteClientRole for removed role
				m.On("DeleteClientRole", testifymock.Anything, token, realmName, clientID, "role2").
					Return(nil).Once()

				// Mock syncClientRoleComposites - get existing roles again
				m.On("GetClientRoles", testifymock.Anything, token, realmName, clientID, testifymock.Anything).
					Return([]*gocloak.Role{
						{
							ID:   gocloak.StringP("role1-id"),
							Name: gocloak.StringP("role1"),
						},
					}, nil).Once()

				// Mock GetCompositeRolesByRoleID
				m.On("GetCompositeRolesByRoleID", testifymock.Anything, token, realmName, "role1-id").
					Return([]*gocloak.Role{}, nil).Once()
			},
			expectedError: "",
			expectedCalls: map[string]int{
				"GetClients":                1,
				"GetClientRoles":            2,
				"DeleteClientRole":          1,
				"GetCompositeRolesByRoleID": 1,
			},
		},
		{
			name: "should successfully sync client roles with composite roles",
			client: &dto.Client{
				ClientId: "test-client",
				Roles: []dto.ClientRole{
					{
						Name:                  "role1",
						Description:           "Role 1 description",
						AssociatedClientRoles: []string{"composite-role1", "composite-role2"},
					},
				},
			},
			setupMocks: func(m *mocks.MockGoCloak) {
				// Mock GetClientID
				m.On("GetClients", testifymock.Anything, token, realmName, testifymock.Anything).
					Return([]*gocloak.Client{
						{
							ID:       gocloak.StringP(clientID),
							ClientID: gocloak.StringP("test-client"),
						},
					}, nil).Once()

				// Mock getExistingClientRolesMap
				m.On("GetClientRoles", testifymock.Anything, token, realmName, clientID, testifymock.Anything).
					Return([]*gocloak.Role{
						{
							ID:          gocloak.StringP("role1-id"),
							Name:        gocloak.StringP("role1"),
							Description: gocloak.StringP("Role 1 description"), // Same description, no update needed
						},
					}, nil).Once()

				// Mock syncClientRoleComposites - get existing roles again
				m.On("GetClientRoles", testifymock.Anything, token, realmName, clientID, testifymock.Anything).
					Return([]*gocloak.Role{
						{
							ID:          gocloak.StringP("role1-id"),
							Name:        gocloak.StringP("role1"),
							Description: gocloak.StringP("Role 1 description"), // Same description, no update needed
						},
					}, nil).Once()

				// Mock GetCompositeRolesByRoleID
				m.On("GetCompositeRolesByRoleID", testifymock.Anything, token, realmName, "role1-id").
					Return([]*gocloak.Role{}, nil).Once()

				// Mock GetClientRole for composite roles - these are called with the same clientID
				m.On("GetClientRole", testifymock.Anything, token, realmName, clientID, "composite-role1").
					Return(&gocloak.Role{
						ID:   gocloak.StringP("composite-role1-id"),
						Name: gocloak.StringP("composite-role1"),
					}, nil).Once()
				m.On("GetClientRole", testifymock.Anything, token, realmName, clientID, "composite-role2").
					Return(&gocloak.Role{
						ID:   gocloak.StringP("composite-role2-id"),
						Name: gocloak.StringP("composite-role2"),
					}, nil).Once()

				// Mock AddClientRoleComposite
				m.On("AddClientRoleComposite", testifymock.Anything, token, realmName, "role1-id", testifymock.Anything).
					Return(nil).Once()
			},
			expectedError: "",
			expectedCalls: map[string]int{
				"GetClients":                1,
				"GetClientRoles":            2,
				"GetCompositeRolesByRoleID": 1,
				"GetClientRole":             2,
				"AddClientRoleComposite":    1,
			},
		},
		{
			name: "should successfully sync client roles with composite roles - remove existing composites",
			client: &dto.Client{
				ClientId: "test-client",
				Roles: []dto.ClientRole{
					{
						Name:                  "role1",
						Description:           "Role 1 description",
						AssociatedClientRoles: []string{"composite-role1"}, // Only one role now
					},
				},
			},
			setupMocks: func(m *mocks.MockGoCloak) {
				setupBasicClientRoleMocks(m, token, realmName, clientID)
				setupCompositeRoleRemovalMocks(m, token, realmName)
			},
			expectedError: "",
			expectedCalls: map[string]int{
				"GetClients":                1,
				"GetClientRoles":            2,
				"GetCompositeRolesByRoleID": 1,
				"DeleteClientRoleComposite": 1,
			},
		},
		{
			name: "should fail when GetClientID returns error",
			client: &dto.Client{
				ClientId: "test-client",
				Roles:    []dto.ClientRole{},
			},
			setupMocks: func(m *mocks.MockGoCloak) {
				// Mock GetClientID to fail
				m.On("GetClients", testifymock.Anything, token, realmName, testifymock.Anything).
					Return(nil, errors.New("client not found")).Once()
			},
			expectedError: "failed to get client ID: unable to get realm clients: client not found",
			expectedCalls: map[string]int{
				"GetClients": 1,
			},
		},
		{
			name: "should fail when getExistingClientRolesMap returns error",
			client: &dto.Client{
				ClientId: "test-client",
				Roles:    []dto.ClientRole{},
			},
			setupMocks: func(m *mocks.MockGoCloak) {
				// Mock GetClientID
				m.On("GetClients", testifymock.Anything, token, realmName, testifymock.Anything).
					Return([]*gocloak.Client{
						{
							ID:       gocloak.StringP(clientID),
							ClientID: gocloak.StringP("test-client"),
						},
					}, nil).Once()

				// Mock getExistingClientRolesMap to fail
				m.On("GetClientRoles", testifymock.Anything, token, realmName, clientID, testifymock.Anything).
					Return(nil, errors.New("failed to get roles")).Once()
			},
			expectedError: "failed to get client roles: failed to get client roles: failed to get roles",
			expectedCalls: map[string]int{
				"GetClients":     1,
				"GetClientRoles": 1,
			},
		},
		{
			name: "should fail when createClientRole returns error",
			client: &dto.Client{
				ClientId: "test-client",
				Roles: []dto.ClientRole{
					{
						Name:        "role1",
						Description: "Role 1 description",
					},
				},
			},
			setupMocks: func(m *mocks.MockGoCloak) {
				// Mock GetClientID
				m.On("GetClients", testifymock.Anything, token, realmName, testifymock.Anything).
					Return([]*gocloak.Client{
						{
							ID:       gocloak.StringP(clientID),
							ClientID: gocloak.StringP("test-client"),
						},
					}, nil).Once()

				// Mock getExistingClientRolesMap - no existing roles
				m.On("GetClientRoles", testifymock.Anything, token, realmName, clientID, testifymock.Anything).
					Return([]*gocloak.Role{}, nil).Once()

				// Mock CreateClientRole to fail
				m.On("CreateClientRole", testifymock.Anything, token, realmName, clientID, testifymock.Anything).
					Return("", errors.New("failed to create role")).Once()
			},
			expectedError: "failed to create client role role1: failed to create role",
			expectedCalls: map[string]int{
				"GetClients":       1,
				"GetClientRoles":   1,
				"CreateClientRole": 1,
			},
		},
		{
			name: "should fail when updateClientRole returns error",
			client: &dto.Client{
				ClientId: "test-client",
				Roles: []dto.ClientRole{
					{
						Name:        "role1",
						Description: "Updated description",
					},
				},
			},
			setupMocks: func(m *mocks.MockGoCloak) {
				// Mock GetClientID
				m.On("GetClients", testifymock.Anything, token, realmName, testifymock.Anything).
					Return([]*gocloak.Client{
						{
							ID:       gocloak.StringP(clientID),
							ClientID: gocloak.StringP("test-client"),
						},
					}, nil).Once()

				// Mock getExistingClientRolesMap - existing role with different description
				m.On("GetClientRoles", testifymock.Anything, token, realmName, clientID, testifymock.Anything).
					Return([]*gocloak.Role{
						{
							ID:          gocloak.StringP("role1-id"),
							Name:        gocloak.StringP("role1"),
							Description: gocloak.StringP("Old description"),
						},
					}, nil).Once()

				// Mock UpdateRole to fail
				m.On("UpdateRole", testifymock.Anything, token, realmName, clientID, testifymock.Anything).
					Return(errors.New("failed to update role")).Once()
			},
			expectedError: "failed to update client role role1: failed to update role",
			expectedCalls: map[string]int{
				"GetClients":     1,
				"GetClientRoles": 1,
				"UpdateRole":     1,
			},
		},
		{
			name: "should fail when deleteRemovedRoles returns error",
			client: &dto.Client{
				ClientId: "test-client",
				Roles:    []dto.ClientRole{},
			},
			setupMocks: func(m *mocks.MockGoCloak) {
				// Mock GetClientID
				m.On("GetClients", testifymock.Anything, token, realmName, testifymock.Anything).
					Return([]*gocloak.Client{
						{
							ID:       gocloak.StringP(clientID),
							ClientID: gocloak.StringP("test-client"),
						},
					}, nil).Once()

				// Mock getExistingClientRolesMap - existing role to be deleted
				m.On("GetClientRoles", testifymock.Anything, token, realmName, clientID, testifymock.Anything).
					Return([]*gocloak.Role{
						{
							ID:   gocloak.StringP("role1-id"),
							Name: gocloak.StringP("role1"),
						},
					}, nil).Once()

				// Mock DeleteClientRole to fail
				m.On("DeleteClientRole", testifymock.Anything, token, realmName, clientID, "role1").
					Return(errors.New("failed to delete role")).Once()
			},
			expectedError: "failed to delete client role role1: failed to delete role",
			expectedCalls: map[string]int{
				"GetClients":       1,
				"GetClientRoles":   1,
				"DeleteClientRole": 1,
			},
		},
		{
			name: "should fail when syncClientRoleComposites returns error",
			client: &dto.Client{
				ClientId: "test-client",
				Roles: []dto.ClientRole{
					{
						Name:        "role1",
						Description: "Role 1 description",
					},
				},
			},
			setupMocks: func(m *mocks.MockGoCloak) {
				// Mock GetClientID
				m.On("GetClients", testifymock.Anything, token, realmName, testifymock.Anything).
					Return([]*gocloak.Client{
						{
							ID:       gocloak.StringP(clientID),
							ClientID: gocloak.StringP("test-client"),
						},
					}, nil).Once()

				// Mock getExistingClientRolesMap
				m.On("GetClientRoles", testifymock.Anything, token, realmName, clientID, testifymock.Anything).
					Return([]*gocloak.Role{
						{
							ID:          gocloak.StringP("role1-id"),
							Name:        gocloak.StringP("role1"),
							Description: gocloak.StringP("Role 1 description"), // Same description, no update needed
						},
					}, nil).Once()

				// Mock syncClientRoleComposites - get existing roles again
				m.On("GetClientRoles", testifymock.Anything, token, realmName, clientID, testifymock.Anything).
					Return([]*gocloak.Role{
						{
							ID:          gocloak.StringP("role1-id"),
							Name:        gocloak.StringP("role1"),
							Description: gocloak.StringP("Role 1 description"), // Same description, no update needed
						},
					}, nil).Once()

				// Mock GetCompositeRolesByRoleID to fail
				m.On("GetCompositeRolesByRoleID", testifymock.Anything, token, realmName, "role1-id").
					Return(nil, errors.New("failed to get composite roles")).Once()
			},
			expectedError: "failed to sync client role composites: failed to get composite roles for role role1: " +
				"failed to get composite roles",
			expectedCalls: map[string]int{
				"GetClients":                1,
				"GetClientRoles":            2,
				"GetCompositeRolesByRoleID": 1,
			},
		},
		{
			name: "should fail when DeleteClientRoleComposite returns error",
			client: &dto.Client{
				ClientId: "test-client",
				Roles: []dto.ClientRole{
					{
						Name:                  "role1",
						Description:           "Role 1 description",
						AssociatedClientRoles: []string{"composite-role1"}, // Only one role now
					},
				},
			},
			setupMocks: func(m *mocks.MockGoCloak) {
				// Mock GetClientID
				m.On("GetClients", testifymock.Anything, token, realmName, testifymock.Anything).
					Return([]*gocloak.Client{
						{
							ID:       gocloak.StringP(clientID),
							ClientID: gocloak.StringP("test-client"),
						},
					}, nil).Once()

				// Mock getExistingClientRolesMap
				m.On("GetClientRoles", testifymock.Anything, token, realmName, clientID, testifymock.Anything).
					Return([]*gocloak.Role{
						{
							ID:          gocloak.StringP("role1-id"),
							Name:        gocloak.StringP("role1"),
							Description: gocloak.StringP("Role 1 description"), // Same description, no update needed
						},
					}, nil).Once()

				// Mock syncClientRoleComposites - get existing roles again
				m.On("GetClientRoles", testifymock.Anything, token, realmName, clientID, testifymock.Anything).
					Return([]*gocloak.Role{
						{
							ID:          gocloak.StringP("role1-id"),
							Name:        gocloak.StringP("role1"),
							Description: gocloak.StringP("Role 1 description"), // Same description, no update needed
						},
					}, nil).Once()

				// Mock GetCompositeRolesByRoleID - return existing composite roles that need to be removed
				m.On("GetCompositeRolesByRoleID", testifymock.Anything, token, realmName, "role1-id").
					Return([]*gocloak.Role{
						{
							ID:   gocloak.StringP("composite-role1-id"),
							Name: gocloak.StringP("composite-role1"),
						},
						{
							ID:   gocloak.StringP("composite-role2-id"),
							Name: gocloak.StringP("composite-role2"), // This role should be removed
						},
					}, nil).Once()

				// Mock DeleteClientRoleComposite to fail
				m.On("DeleteClientRoleComposite", testifymock.Anything, token, realmName, "role1-id", testifymock.Anything).
					Return(errors.New("failed to delete composite role")).Once()
			},
			expectedError: "failed to sync client role composites: failed to remove composite roles from role1: " +
				"failed to delete composite role",
			expectedCalls: map[string]int{
				"GetClients":                1,
				"GetClientRoles":            2,
				"GetCompositeRolesByRoleID": 1,
				"DeleteClientRoleComposite": 1,
			},
		},
		{
			name: "should handle empty roles list successfully",
			client: &dto.Client{
				ClientId: "test-client",
				Roles:    []dto.ClientRole{},
			},
			setupMocks: func(m *mocks.MockGoCloak) {
				// Mock GetClientID
				m.On("GetClients", testifymock.Anything, token, realmName, testifymock.Anything).
					Return([]*gocloak.Client{
						{
							ID:       gocloak.StringP(clientID),
							ClientID: gocloak.StringP("test-client"),
						},
					}, nil).Once()

				// Mock getExistingClientRolesMap - no existing roles
				m.On("GetClientRoles", testifymock.Anything, token, realmName, clientID, testifymock.Anything).
					Return([]*gocloak.Role{}, nil).Once()

				// Mock syncClientRoleComposites - get existing roles again
				m.On("GetClientRoles", testifymock.Anything, token, realmName, clientID, testifymock.Anything).
					Return([]*gocloak.Role{}, nil).Once()
			},
			expectedError: "",
			expectedCalls: map[string]int{
				"GetClients":     1,
				"GetClientRoles": 2,
			},
		},
		{
			name: "should handle empty composite roles - remove all existing composites",
			client: &dto.Client{
				ClientId: "test-client",
				Roles: []dto.ClientRole{
					{
						Name:                  "role1",
						Description:           "Role 1 description",
						AssociatedClientRoles: []string{}, // Empty - should remove all existing composites
					},
				},
			},
			setupMocks: func(m *mocks.MockGoCloak) {
				setupBasicClientRoleMocks(m, token, realmName, clientID)
				setupCompositeRoleRemovalMocks(m, token, realmName)
			},
			expectedError: "",
			expectedCalls: map[string]int{
				"GetClients":                1,
				"GetClientRoles":            2,
				"GetCompositeRolesByRoleID": 1,
				"DeleteClientRoleComposite": 1,
			},
		},
		{
			name: "should fail when handleEmptyComposites returns error",
			client: &dto.Client{
				ClientId: "test-client",
				Roles: []dto.ClientRole{
					{
						Name:                  "role1",
						Description:           "Role 1 description",
						AssociatedClientRoles: []string{}, // Empty - should trigger handleEmptyComposites
					},
				},
			},
			setupMocks: func(m *mocks.MockGoCloak) {
				// Mock GetClientID
				m.On("GetClients", testifymock.Anything, token, realmName, testifymock.Anything).
					Return([]*gocloak.Client{
						{
							ID:       gocloak.StringP(clientID),
							ClientID: gocloak.StringP("test-client"),
						},
					}, nil).Once()

				// Mock getExistingClientRolesMap
				m.On("GetClientRoles", testifymock.Anything, token, realmName, clientID, testifymock.Anything).
					Return([]*gocloak.Role{
						{
							ID:          gocloak.StringP("role1-id"),
							Name:        gocloak.StringP("role1"),
							Description: gocloak.StringP("Role 1 description"), // Same description, no update needed
						},
					}, nil).Once()

				// Mock syncClientRoleComposites - get existing roles again
				m.On("GetClientRoles", testifymock.Anything, token, realmName, clientID, testifymock.Anything).
					Return([]*gocloak.Role{
						{
							ID:          gocloak.StringP("role1-id"),
							Name:        gocloak.StringP("role1"),
							Description: gocloak.StringP("Role 1 description"), // Same description, no update needed
						},
					}, nil).Once()

				// Mock GetCompositeRolesByRoleID - return existing composite roles that should all be removed
				m.On("GetCompositeRolesByRoleID", testifymock.Anything, token, realmName, "role1-id").
					Return([]*gocloak.Role{
						{
							ID:   gocloak.StringP("composite-role1-id"),
							Name: gocloak.StringP("composite-role1"),
						},
					}, nil).Once()

				// Mock DeleteClientRoleComposite to fail
				m.On("DeleteClientRoleComposite", testifymock.Anything, token, realmName, "role1-id", testifymock.Anything).
					Return(errors.New("failed to delete composite role")).Once()
			},
			expectedError: "failed to sync client role composites: failed to remove composite roles from role1: " +
				"failed to delete composite role",
			expectedCalls: map[string]int{
				"GetClients":                1,
				"GetClientRoles":            2,
				"GetCompositeRolesByRoleID": 1,
				"DeleteClientRoleComposite": 1,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			// Create mock client
			mockClient := mocks.NewMockGoCloak(t)

			// Setup mocks
			if tt.setupMocks != nil {
				tt.setupMocks(mockClient)
			}

			// Create adapter
			adapter := &GoCloakAdapter{
				client: mockClient,
				token:  &gocloak.JWT{AccessToken: token},
				log:    logr.Discard(),
			}

			// Call the method
			err := adapter.SyncClientRoles(context.Background(), realmName, tt.client)

			// Assert results
			if tt.expectedError != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedError)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

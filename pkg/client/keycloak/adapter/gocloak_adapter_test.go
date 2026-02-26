package adapter

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/Nerzal/gocloak/v12"
	"github.com/go-logr/logr"
	"github.com/go-resty/resty/v2"
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
	server            *httptest.Server
}

func (e *AdapterTestSuite) SetupTest() {
	e.server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	e.restyClient = resty.New()
	e.restyClient.SetBaseURL(e.server.URL)

	e.goCloakMockClient = mocks.NewMockGoCloak(e.T())
	e.goCloakMockClient.On("RestyClient").Return(e.restyClient).Maybe()

	e.adapter = &GoCloakAdapter{
		client: e.goCloakMockClient,
		token:  &gocloak.JWT{AccessToken: "token"},
		log:    mock.NewLogr(),
	}

	e.realmName = "realm123"
}

func (e *AdapterTestSuite) TearDownTest() {
	if e.server != nil {
		e.server.Close()
	}
}

func TestAdapterTestSuite(t *testing.T) {
	suite.Run(t, new(AdapterTestSuite))
}

func (e *AdapterTestSuite) TestMakeFromServiceAccount() {
	t := e.T()

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
			wantErr: func(t require.TestingT, err error, _ ...any) {
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
			wantErr: func(t require.TestingT, err error, i ...any) {
				require.Error(t, err)
				require.Contains(t, err.Error(), "unsupported protocol scheme")
			},
		},
		{
			name: "should fail with status 400",
			mockServer: fakehttp.NewServerBuilder().
				AddStringResponderWithCode(http.StatusBadRequest, authPath+realmsEndpoint, "{}").
				BuildAndStart(),
			wantErr: func(t require.TestingT, err error, i ...any) {
				require.Error(t, err)
				require.Contains(t, err.Error(), "400")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
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
	messageBody := "not found"

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		expectedPath := strings.Replace(getClientProtocolMappers, "{realm}", client.RealmName, 1)
		expectedPath = strings.Replace(expectedPath, "{id}", clientID, 1)

		if r.Method == http.MethodGet && r.URL.Path == expectedPath {
			w.WriteHeader(http.StatusNotFound)
			_, _ = w.Write([]byte(messageBody))

			return
		}

		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	mockClient := mocks.NewMockGoCloak(t)
	restyClient := resty.New()
	restyClient.SetBaseURL(server.URL)
	mockClient.On("RestyClient").Return(restyClient)

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

	// Create a server that will close the connection to simulate a network error
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hj, ok := w.(http.Hijacker)
		if !ok {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		conn, _, _ := hj.Hijack()
		_ = conn.Close() // Close connection to simulate network error
	}))
	defer server.Close()

	mockClient := mocks.NewMockGoCloak(t)
	restyClient := resty.New()
	restyClient.SetBaseURL(server.URL)
	mockClient.On("RestyClient").Return(restyClient)

	adapter := GoCloakAdapter{
		client:   mockClient,
		token:    &gocloak.JWT{AccessToken: "token"},
		basePath: "",
	}

	_, err := adapter.GetClientProtocolMappers(&client, clientID)
	require.Error(t, err)

	// Check that we get a network-related error (EOF or connection-related)
	assert.True(t, strings.Contains(err.Error(), "connection") || strings.Contains(err.Error(), "EOF"))
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
}

func TestGoCloakAdapter_SyncClientProtocolMapper_Success(t *testing.T) {
	client := dto.Client{
		RealmName: "test",
		ClientId:  "test",
	}
	clientID := "321"

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

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		expectedPath := strings.Replace(getClientProtocolMappers, "{realm}", client.RealmName, 1)
		expectedPath = strings.Replace(expectedPath, "{id}", clientID, 1)

		if r.Method == http.MethodGet && r.URL.Path == expectedPath {
			setJSONContentType(w)
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(kcMappers)

			return
		}

		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	mockClient := mocks.NewMockGoCloak(t)
	restyClient := resty.New()
	restyClient.SetBaseURL(server.URL)
	mockClient.On("RestyClient").Return(restyClient)
	mockClient.On("GetClients", testifymock.Anything, "token", client.RealmName, gocloak.GetClientsParams{
		ClientID: &client.ClientId,
	}).Return([]*gocloak.Client{
		{
			ClientID: &client.ClientId,
			ID:       &clientID,
		},
	}, nil)

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

	adapter := GoCloakAdapter{
		client:   mockClient,
		token:    &gocloak.JWT{AccessToken: "token"},
		basePath: "",
		log:      mock.NewLogr(),
	}

	err := adapter.SyncClientProtocolMapper(&client, crMappers, false)
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

func TestGoCloakAdapter_GetGoCloak(t *testing.T) {
	gcl := GoCloakAdapter{}
	if gcl.GetGoCloak() != nil {
		t.Fatal("go cloak must be nil")
	}
}

func TestMakeFromToken(t *testing.T) {
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
		wantErr    func(require.TestingT, error, ...any)
	}{
		{
			name:  "should succeed",
			token: workingToken,
			mockServer: fakehttp.NewServerBuilder().
				AddStringResponder("/admin/realms/", "{}").
				BuildAndStart(),
			wantErr: func(t require.TestingT, err error, i ...any) {
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
			wantErr: func(t require.TestingT, err error, i ...any) {
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
			wantErr: func(t require.TestingT, err error, i ...any) {
				require.Error(t, err)
				require.True(t, IsErrTokenExpired(err) || err.Error() == "token is expired")
			},
		},
		{
			name:       "should fail on wrong token structure",
			token:      "foo.bar",
			mockServer: nil,
			wantErr: func(t require.TestingT, err error, i ...any) {
				require.Error(t, err)
				require.Contains(t, err.Error(), "wrong JWT token structure")
			},
		},
		{
			name:       "should fail on wrong token encoding",
			token:      "foo.bar .baz",
			mockServer: nil,
			wantErr: func(t require.TestingT, err error, i ...any) {
				require.Error(t, err)
				require.Contains(t, err.Error(), "wrong JWT token base64 encoding")
			},
		},
		{
			name:       "should fail on decoding json payload",
			token:      "foo.bar.baz",
			mockServer: nil,
			wantErr: func(t require.TestingT, err error, i ...any) {
				require.Error(t, err)
				require.Contains(t, err.Error(), "unable to decode JWT payload json")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
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
	_, err := MakeFromToken(GoCloakConfig{Url: "test_url"}, []byte("qwdqwdwq"), mock.NewLogr())
	require.Error(t, err)
	require.Contains(t, err.Error(), "invalid character")
}

func (e *AdapterTestSuite) TestGoCloakAdapter_DeleteRealmUser() {
	tests := []struct {
		name           string
		username       string
		getUsersResult []*gocloak.User
		getUsersError  error
		deleteStatus   int
		expectedError  string
	}{
		{
			name:           "success",
			username:       "username",
			getUsersResult: []*gocloak.User{{Username: gocloak.StringP("username"), ID: gocloak.StringP("username")}},
			deleteStatus:   200,
		},
		{
			name:           "delete error",
			username:       "username",
			getUsersResult: []*gocloak.User{{Username: gocloak.StringP("username"), ID: gocloak.StringP("username")}},
			deleteStatus:   404,
			expectedError:  "unable to delete user: status: 404 Not Found, body: ",
		},
		{
			name:           "user not found",
			username:       "username",
			getUsersResult: []*gocloak.User{{}},
			expectedError:  "user not found",
		},
		{
			name:          "get users error",
			username:      "username",
			getUsersError: errors.New("fatal get users"),
			expectedError: "unable to get users: fatal get users",
		},
	}

	for _, tt := range tests {
		e.T().Run(tt.name, func(t *testing.T) {
			// Setup server for this specific test
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				expectedPath := strings.Replace(deleteRealmUser, "{realm}", e.realmName, 1)
				expectedPath = strings.Replace(expectedPath, "{id}", tt.username, 1)

				if r.Method == http.MethodDelete && r.URL.Path == expectedPath {
					w.WriteHeader(tt.deleteStatus)
					return
				}

				w.WriteHeader(http.StatusNotFound)
			}))
			defer server.Close()

			// Create adapter with test-specific server
			testClient := resty.New()
			testClient.SetBaseURL(server.URL)

			mockClient := mocks.NewMockGoCloak(t)
			mockClient.On("RestyClient").Return(testClient).Maybe()

			mockClient.On(
				"GetUsers",
				testifymock.Anything,
				"token",
				e.realmName,
				gocloak.GetUsersParams{Username: &tt.username},
			).Return(tt.getUsersResult, tt.getUsersError).Once()

			adapter := &GoCloakAdapter{
				client: mockClient,
				token:  &gocloak.JWT{AccessToken: "token"},
				log:    mock.NewLogr(),
			}

			err := adapter.DeleteRealmUser(context.Background(), e.realmName, tt.username)
			if tt.expectedError != "" {
				assert.Error(t, err)
				assert.EqualError(t, err, tt.expectedError)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestGoCloakAdapter_GetUsersByNames(t *testing.T) {
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
			wantErr: func(t require.TestingT, err error, i ...any) {
				require.Error(t, err)
				require.Contains(t, err.Error(), "fatal")
			},
			want: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
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
			wantErr: func(t require.TestingT, err error, i ...any) {
				require.Error(t, err)
				require.Contains(t, err.Error(), "failed to get role")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
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

func TestGoCloakAdapter_ExportToken(t *testing.T) {
	tests := []struct {
		name        string
		token       *gocloak.JWT
		expectError bool
		errorMsg    string
	}{
		{
			name: "should export token successfully",
			token: &gocloak.JWT{
				AccessToken:      "valid-access-token",
				RefreshToken:     "valid-refresh-token",
				ExpiresIn:        3600,
				RefreshExpiresIn: 7200,
				TokenType:        "Bearer",
			},
			expectError: false,
		},
		{
			name: "should export token with minimal fields",
			token: &gocloak.JWT{
				AccessToken: "minimal-token",
			},
			expectError: false,
		},
		{
			name:        "should handle nil token",
			token:       nil,
			expectError: false, // json.Marshal(nil) returns "null" which is valid JSON
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			adapter := &GoCloakAdapter{
				token: tt.token,
				log:   mock.NewLogr(),
			}

			tokenData, err := adapter.ExportToken()

			if tt.expectError {
				require.Error(t, err)

				if tt.errorMsg != "" {
					require.Contains(t, err.Error(), tt.errorMsg)
				}

				require.Nil(t, tokenData)
			} else {
				require.NoError(t, err)
				require.NotNil(t, tokenData)

				// Verify the exported token can be unmarshaled back
				var jwt gocloak.JWT
				unmarshalErr := json.Unmarshal(tokenData, &jwt)
				require.NoError(t, unmarshalErr)

				if tt.token != nil {
					require.Equal(t, tt.token.AccessToken, jwt.AccessToken)
					require.Equal(t, tt.token.RefreshToken, jwt.RefreshToken)
					require.Equal(t, tt.token.ExpiresIn, jwt.ExpiresIn)
					require.Equal(t, tt.token.RefreshExpiresIn, jwt.RefreshExpiresIn)
					require.Equal(t, tt.token.TokenType, jwt.TokenType)
				}
			}
		})
	}
}

func TestGoCloakAdapter_DeleteClient(t *testing.T) {
	tests := []struct {
		name          string
		kcClientID    string
		realmName     string
		setupMocks    func(*mocks.MockGoCloak)
		expectedError string
	}{
		{
			name:       "should delete client successfully",
			kcClientID: "test-client-id",
			realmName:  "test-realm",
			setupMocks: func(m *mocks.MockGoCloak) {
				m.On("DeleteClient", testifymock.Anything, "token", "test-realm", "test-client-id").
					Return(nil).Once()
			},
			expectedError: "",
		},
		{
			name:       "should return error when delete client fails",
			kcClientID: "test-client-id",
			realmName:  "test-realm",
			setupMocks: func(m *mocks.MockGoCloak) {
				m.On("DeleteClient", testifymock.Anything, "token", "test-realm", "test-client-id").
					Return(errors.New("keycloak delete error")).Once()
			},
			expectedError: "unable to delete client: keycloak delete error",
		},
		{
			name:       "should return error when client not found",
			kcClientID: "non-existent-client",
			realmName:  "test-realm",
			setupMocks: func(m *mocks.MockGoCloak) {
				m.On("DeleteClient", testifymock.Anything, "token", "test-realm", "non-existent-client").
					Return(&gocloak.APIError{Code: 404, Message: "Client not found"}).Once()
			},
			expectedError: "unable to delete client: Client not found",
		},
		{
			name:       "should return error when unauthorized",
			kcClientID: "test-client-id",
			realmName:  "test-realm",
			setupMocks: func(m *mocks.MockGoCloak) {
				m.On("DeleteClient", testifymock.Anything, "token", "test-realm", "test-client-id").
					Return(&gocloak.APIError{Code: 403, Message: "Insufficient permissions"}).Once()
			},
			expectedError: "unable to delete client: Insufficient permissions",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create mock client
			mockClient := mocks.NewMockGoCloak(t)

			// Setup mocks
			if tt.setupMocks != nil {
				tt.setupMocks(mockClient)
			}

			// Create adapter
			adapter := &GoCloakAdapter{
				client: mockClient,
				token:  &gocloak.JWT{AccessToken: "token"},
				log:    mock.NewLogr(),
			}

			// Call the method
			err := adapter.DeleteClient(context.Background(), tt.kcClientID, tt.realmName)

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

func TestGoCloakAdapter_HasUserRealmRole(t *testing.T) {
	tests := []struct {
		name           string
		realmName      string
		user           *dto.User
		role           string
		setupMocks     func(*mocks.MockGoCloak)
		expectedError  string
		expectedResult bool
	}{
		{
			name:      "should return true when user has role",
			realmName: "test-realm",
			user:      &dto.User{Username: "test-user"},
			role:      "test-role",
			setupMocks: func(m *mocks.MockGoCloak) {
				m.On("GetUsers", testifymock.Anything, "token", "test-realm", gocloak.GetUsersParams{
					Username: gocloak.StringP("test-user"),
				}).Return([]*gocloak.User{
					{
						ID:       gocloak.StringP("user-id"),
						Username: gocloak.StringP("test-user"),
					},
				}, nil).Once()

				m.On("GetRoleMappingByUserID", testifymock.Anything, "token", "test-realm", "user-id").
					Return(&gocloak.MappingsRepresentation{
						RealmMappings: &[]gocloak.Role{
							{Name: gocloak.StringP("test-role")},
							{Name: gocloak.StringP("other-role")},
						},
					}, nil).Once()
			},
			expectedError:  "",
			expectedResult: true,
		},
		{
			name:      "should return false when user doesn't have role",
			realmName: "test-realm",
			user:      &dto.User{Username: "test-user"},
			role:      "missing-role",
			setupMocks: func(m *mocks.MockGoCloak) {
				m.On("GetUsers", testifymock.Anything, "token", "test-realm", gocloak.GetUsersParams{
					Username: gocloak.StringP("test-user"),
				}).Return([]*gocloak.User{
					{
						ID:       gocloak.StringP("user-id"),
						Username: gocloak.StringP("test-user"),
					},
				}, nil).Once()

				m.On("GetRoleMappingByUserID", testifymock.Anything, "token", "test-realm", "user-id").
					Return(&gocloak.MappingsRepresentation{
						RealmMappings: &[]gocloak.Role{
							{Name: gocloak.StringP("test-role")},
							{Name: gocloak.StringP("other-role")},
						},
					}, nil).Once()
			},
			expectedError:  "",
			expectedResult: false,
		},
		{
			name:      "should return error when GetUsers fails",
			realmName: "test-realm",
			user:      &dto.User{Username: "test-user"},
			role:      "test-role",
			setupMocks: func(m *mocks.MockGoCloak) {
				m.On("GetUsers", testifymock.Anything, "token", "test-realm", gocloak.GetUsersParams{
					Username: gocloak.StringP("test-user"),
				}).Return(nil, errors.New("keycloak connection error")).Once()
			},
			expectedError:  "unable to get users from keycloak: keycloak connection error",
			expectedResult: false,
		},
		{
			name:      "should return error when user not found",
			realmName: "test-realm",
			user:      &dto.User{Username: "non-existent-user"},
			role:      "test-role",
			setupMocks: func(m *mocks.MockGoCloak) {
				m.On("GetUsers", testifymock.Anything, "token", "test-realm", gocloak.GetUsersParams{
					Username: gocloak.StringP("non-existent-user"),
				}).Return([]*gocloak.User{}, nil).Once()
			},
			expectedError:  "no such user non-existent-user has been found",
			expectedResult: false,
		},
		{
			name:      "should return error when GetRoleMappingByUserID fails",
			realmName: "test-realm",
			user:      &dto.User{Username: "test-user"},
			role:      "test-role",
			setupMocks: func(m *mocks.MockGoCloak) {
				m.On("GetUsers", testifymock.Anything, "token", "test-realm", gocloak.GetUsersParams{
					Username: gocloak.StringP("test-user"),
				}).Return([]*gocloak.User{
					{
						ID:       gocloak.StringP("user-id"),
						Username: gocloak.StringP("test-user"),
					},
				}, nil).Once()

				m.On("GetRoleMappingByUserID", testifymock.Anything, "token", "test-realm", "user-id").
					Return(nil, errors.New("role mapping retrieval failed")).Once()
			},
			expectedError:  "unable to GetRoleMappingByUserID: role mapping retrieval failed",
			expectedResult: false,
		},
		{
			name:      "should handle API error from GetUsers",
			realmName: "test-realm",
			user:      &dto.User{Username: "test-user"},
			role:      "test-role",
			setupMocks: func(m *mocks.MockGoCloak) {
				m.On("GetUsers", testifymock.Anything, "token", "test-realm", gocloak.GetUsersParams{
					Username: gocloak.StringP("test-user"),
				}).Return(nil, &gocloak.APIError{Code: 500, Message: "Internal server error"}).Once()
			},
			expectedError:  "unable to get users from keycloak: Internal server error",
			expectedResult: false,
		},
		{
			name:      "should handle API error from GetRoleMappingByUserID",
			realmName: "test-realm",
			user:      &dto.User{Username: "test-user"},
			role:      "test-role",
			setupMocks: func(m *mocks.MockGoCloak) {
				m.On("GetUsers", testifymock.Anything, "token", "test-realm", gocloak.GetUsersParams{
					Username: gocloak.StringP("test-user"),
				}).Return([]*gocloak.User{
					{
						ID:       gocloak.StringP("user-id"),
						Username: gocloak.StringP("test-user"),
					},
				}, nil).Once()

				m.On("GetRoleMappingByUserID", testifymock.Anything, "token", "test-realm", "user-id").
					Return(nil, &gocloak.APIError{Code: 403, Message: "Forbidden"}).Once()
			},
			expectedError:  "unable to GetRoleMappingByUserID: Forbidden",
			expectedResult: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create mock client
			mockClient := mocks.NewMockGoCloak(t)

			// Setup mocks
			if tt.setupMocks != nil {
				tt.setupMocks(mockClient)
			}

			// Create adapter
			adapter := &GoCloakAdapter{
				client: mockClient,
				token:  &gocloak.JWT{AccessToken: "token"},
				log:    mock.NewLogr(),
			}

			// Call the method
			result, err := adapter.HasUserRealmRole(tt.realmName, tt.user, tt.role)

			// Assert results
			if tt.expectedError != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedError)
				assert.Equal(t, tt.expectedResult, result)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.expectedResult, result)
			}

		})
	}
}

func TestGoCloakAdapter_AddRealmRoleToUser(t *testing.T) {
	tests := []struct {
		name          string
		realmName     string
		username      string
		roleName      string
		setupMocks    func(*mocks.MockGoCloak)
		expectedError string
	}{
		{
			name:      "should add realm role to user successfully",
			realmName: "test-realm",
			username:  "test-user",
			roleName:  "test-role",
			setupMocks: func(m *mocks.MockGoCloak) {
				// Mock GetUsers - success
				m.On("GetUsers", testifymock.Anything, "token", "test-realm", gocloak.GetUsersParams{
					Username: gocloak.StringP("test-user"),
				}).Return([]*gocloak.User{
					{
						ID:       gocloak.StringP("user-id"),
						Username: gocloak.StringP("test-user"),
					},
				}, nil).Once()

				// Mock GetRealmRole - success
				m.On("GetRealmRole", testifymock.Anything, "token", "test-realm", "test-role").
					Return(&gocloak.Role{
						ID:   gocloak.StringP("role-id"),
						Name: gocloak.StringP("test-role"),
					}, nil).Once()

				// Mock AddRealmRoleToUser - success
				m.On(
					"AddRealmRoleToUser",
					testifymock.Anything,
					"token",
					"test-realm",
					"user-id",
					testifymock.MatchedBy(func(roles []gocloak.Role) bool {
						return len(roles) == 1 &&
							roles[0].ID != nil && *roles[0].ID == "role-id" &&
							roles[0].Name != nil && *roles[0].Name == "test-role"
					}),
				).Return(nil).Once()
			},
			expectedError: "",
		},
		{
			name:      "should return error when GetUsers fails",
			realmName: "test-realm",
			username:  "test-user",
			roleName:  "test-role",
			setupMocks: func(m *mocks.MockGoCloak) {
				// Mock GetUsers - failure
				m.On("GetUsers", testifymock.Anything, "token", "test-realm", gocloak.GetUsersParams{
					Username: gocloak.StringP("test-user"),
				}).Return(nil, errors.New("keycloak connection error")).Once()
			},
			expectedError: "error during get kc users: keycloak connection error",
		},
		{
			name:      "should return error when GetUsers returns API error",
			realmName: "test-realm",
			username:  "test-user",
			roleName:  "test-role",
			setupMocks: func(m *mocks.MockGoCloak) {
				// Mock GetUsers - API error
				m.On("GetUsers", testifymock.Anything, "token", "test-realm", gocloak.GetUsersParams{
					Username: gocloak.StringP("test-user"),
				}).Return(nil, &gocloak.APIError{Code: 500, Message: "Internal server error"}).Once()
			},
			expectedError: "error during get kc users: Internal server error",
		},
		{
			name:      "should return error when no users found",
			realmName: "test-realm",
			username:  "non-existent-user",
			roleName:  "test-role",
			setupMocks: func(m *mocks.MockGoCloak) {
				// Mock GetUsers - empty result
				m.On("GetUsers", testifymock.Anything, "token", "test-realm", gocloak.GetUsersParams{
					Username: gocloak.StringP("non-existent-user"),
				}).Return([]*gocloak.User{}, nil).Once()
			},
			expectedError: "no users with username non-existent-user found",
		},
		{
			name:      "should return error when GetUsers returns nil slice",
			realmName: "test-realm",
			username:  "test-user",
			roleName:  "test-role",
			setupMocks: func(m *mocks.MockGoCloak) {
				// Mock GetUsers - nil result
				m.On("GetUsers", testifymock.Anything, "token", "test-realm", gocloak.GetUsersParams{
					Username: gocloak.StringP("test-user"),
				}).Return(nil, nil).Once()
			},
			expectedError: "no users with username test-user found",
		},
		{
			name:      "should return error when GetRealmRole fails",
			realmName: "test-realm",
			username:  "test-user",
			roleName:  "test-role",
			setupMocks: func(m *mocks.MockGoCloak) {
				// Mock GetUsers - success
				m.On("GetUsers", testifymock.Anything, "token", "test-realm", gocloak.GetUsersParams{
					Username: gocloak.StringP("test-user"),
				}).Return([]*gocloak.User{
					{
						ID:       gocloak.StringP("user-id"),
						Username: gocloak.StringP("test-user"),
					},
				}, nil).Once()

				// Mock GetRealmRole - failure
				m.On("GetRealmRole", testifymock.Anything, "token", "test-realm", "test-role").
					Return(nil, errors.New("role not found")).Once()
			},
			expectedError: "unable to get realm role from keycloak: role not found",
		},
		{
			name:      "should return error when GetRealmRole returns API error",
			realmName: "test-realm",
			username:  "test-user",
			roleName:  "non-existent-role",
			setupMocks: func(m *mocks.MockGoCloak) {
				// Mock GetUsers - success
				m.On("GetUsers", testifymock.Anything, "token", "test-realm", gocloak.GetUsersParams{
					Username: gocloak.StringP("test-user"),
				}).Return([]*gocloak.User{
					{
						ID:       gocloak.StringP("user-id"),
						Username: gocloak.StringP("test-user"),
					},
				}, nil).Once()

				// Mock GetRealmRole - API error
				m.On("GetRealmRole", testifymock.Anything, "token", "test-realm", "non-existent-role").
					Return(nil, &gocloak.APIError{Code: 404, Message: "Role not found"}).Once()
			},
			expectedError: "unable to get realm role from keycloak: Role not found",
		},
		{
			name:      "should return error when AddRealmRoleToUser fails",
			realmName: "test-realm",
			username:  "test-user",
			roleName:  "test-role",
			setupMocks: func(m *mocks.MockGoCloak) {
				// Mock GetUsers - success
				m.On("GetUsers", testifymock.Anything, "token", "test-realm", gocloak.GetUsersParams{
					Username: gocloak.StringP("test-user"),
				}).Return([]*gocloak.User{
					{
						ID:       gocloak.StringP("user-id"),
						Username: gocloak.StringP("test-user"),
					},
				}, nil).Once()

				// Mock GetRealmRole - success
				m.On("GetRealmRole", testifymock.Anything, "token", "test-realm", "test-role").
					Return(&gocloak.Role{
						ID:   gocloak.StringP("role-id"),
						Name: gocloak.StringP("test-role"),
					}, nil).Once()

				// Mock AddRealmRoleToUser - failure
				m.On("AddRealmRoleToUser", testifymock.Anything, "token", "test-realm", "user-id", testifymock.Anything).
					Return(errors.New("insufficient permissions")).Once()
			},
			expectedError: "unable to add realm role to user: insufficient permissions",
		},
		{
			name:      "should return error when AddRealmRoleToUser returns API error",
			realmName: "test-realm",
			username:  "test-user",
			roleName:  "test-role",
			setupMocks: func(m *mocks.MockGoCloak) {
				// Mock GetUsers - success
				m.On("GetUsers", testifymock.Anything, "token", "test-realm", gocloak.GetUsersParams{
					Username: gocloak.StringP("test-user"),
				}).Return([]*gocloak.User{
					{
						ID:       gocloak.StringP("user-id"),
						Username: gocloak.StringP("test-user"),
					},
				}, nil).Once()

				// Mock GetRealmRole - success
				m.On("GetRealmRole", testifymock.Anything, "token", "test-realm", "test-role").
					Return(&gocloak.Role{
						ID:   gocloak.StringP("role-id"),
						Name: gocloak.StringP("test-role"),
					}, nil).Once()

				// Mock AddRealmRoleToUser - API error
				m.On("AddRealmRoleToUser", testifymock.Anything, "token", "test-realm", "user-id", testifymock.Anything).
					Return(&gocloak.APIError{Code: 403, Message: "Forbidden"}).Once()
			},
			expectedError: "unable to add realm role to user: Forbidden",
		},
		{
			name:      "should handle multiple users with same username",
			realmName: "test-realm",
			username:  "test-user",
			roleName:  "test-role",
			setupMocks: func(m *mocks.MockGoCloak) {
				// Mock GetUsers - multiple users (should use first one)
				m.On("GetUsers", testifymock.Anything, "token", "test-realm", gocloak.GetUsersParams{
					Username: gocloak.StringP("test-user"),
				}).Return([]*gocloak.User{
					{
						ID:       gocloak.StringP("user-id-1"),
						Username: gocloak.StringP("test-user"),
					},
					{
						ID:       gocloak.StringP("user-id-2"),
						Username: gocloak.StringP("test-user"),
					},
				}, nil).Once()

				// Mock GetRealmRole - success
				m.On("GetRealmRole", testifymock.Anything, "token", "test-realm", "test-role").
					Return(&gocloak.Role{
						ID:   gocloak.StringP("role-id"),
						Name: gocloak.StringP("test-role"),
					}, nil).Once()

				// Mock AddRealmRoleToUser - should use first user ID
				m.On("AddRealmRoleToUser", testifymock.Anything, "token", "test-realm", "user-id-1", testifymock.Anything).
					Return(nil).Once()
			},
			expectedError: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create mock client
			mockClient := mocks.NewMockGoCloak(t)

			// Setup mocks
			if tt.setupMocks != nil {
				tt.setupMocks(mockClient)
			}

			// Create adapter
			adapter := &GoCloakAdapter{
				client: mockClient,
				token:  &gocloak.JWT{AccessToken: "token"},
				log:    mock.NewLogr(),
			}

			// Call the method
			err := adapter.AddRealmRoleToUser(context.Background(), tt.realmName, tt.username, tt.roleName)

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

func TestGoCloakAdapter_GetClientProtocolMappers_Success(t *testing.T) {
	client := &dto.Client{
		RealmName: "test-realm",
		ClientId:  "test-client",
	}
	clientID := "client-uuid7"

	expectedMappers := []gocloak.ProtocolMapperRepresentation{
		{
			ID:             gocloak.StringP("mapper1-id"),
			Name:           gocloak.StringP("username-mapper"),
			Protocol:       gocloak.StringP("openid-connect"),
			ProtocolMapper: gocloak.StringP("oidc-usermodel-property-mapper"),
			Config: &map[string]string{
				"user.attribute": "username",
				"claim.name":     "preferred_username",
			},
		},
		{
			ID:             gocloak.StringP("mapper2-id"),
			Name:           gocloak.StringP("role-mapper"),
			Protocol:       gocloak.StringP("openid-connect"),
			ProtocolMapper: gocloak.StringP("oidc-usermodel-realm-role-mapper"),
			Config: &map[string]string{
				"claim.name": "roles",
			},
		},
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		expectedPath := strings.Replace(getClientProtocolMappers, "{realm}", client.RealmName, 1)
		expectedPath = strings.Replace(expectedPath, "{id}", clientID, 1)

		if r.Method == http.MethodGet && r.URL.Path == expectedPath {
			setJSONContentType(w)
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(expectedMappers)

			return
		}

		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	mockClient := mocks.NewMockGoCloak(t)
	restyClient := resty.New()
	restyClient.SetBaseURL(server.URL)
	mockClient.On("RestyClient").Return(restyClient)

	adapter := GoCloakAdapter{
		client:   mockClient,
		token:    &gocloak.JWT{AccessToken: "token"},
		basePath: "",
		log:      mock.NewLogr(),
	}

	mappers, err := adapter.GetClientProtocolMappers(client, clientID)
	require.NoError(t, err)
	require.Len(t, mappers, 2)
	assert.Equal(t, "username-mapper", *mappers[0].Name)
	assert.Equal(t, "role-mapper", *mappers[1].Name)
}

func TestGoCloakAdapter_GetClientProtocolMappers_NetworkError(t *testing.T) {
	client := &dto.Client{
		RealmName: "test-realm",
		ClientId:  "test-client",
	}
	clientID := "client-uuid7"

	// Create a server that will close the connection to simulate a network error
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hj, ok := w.(http.Hijacker)
		if !ok {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		conn, _, _ := hj.Hijack()
		_ = conn.Close() // Close connection to simulate network error
	}))
	defer server.Close()

	mockClient := mocks.NewMockGoCloak(t)
	restyClient := resty.New()
	restyClient.SetBaseURL(server.URL)
	mockClient.On("RestyClient").Return(restyClient)

	adapter := GoCloakAdapter{
		client:   mockClient,
		token:    &gocloak.JWT{AccessToken: "token"},
		basePath: "",
		log:      mock.NewLogr(),
	}

	_, err := adapter.GetClientProtocolMappers(client, clientID)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to get client protocol mappers:")
}

func TestGoCloakAdapter_SyncClientProtocolMapper_CreateError(t *testing.T) {
	client := dto.Client{
		RealmName: "test-realm",
		ClientId:  "test-client",
	}
	clientID := "client-uuid2"

	// Mappers that need to be created (don't exist in Keycloak)
	claimedMappers := []gocloak.ProtocolMapperRepresentation{
		{
			Name:           gocloak.StringP("new-mapper"),
			Protocol:       gocloak.StringP("openid-connect"),
			ProtocolMapper: gocloak.StringP("oidc-usermodel-property-mapper"),
			Config: &map[string]string{
				"user.attribute": "username",
				"claim.name":     "preferred_username",
			},
		},
	}

	// Current mappers in Keycloak (empty - so new mapper needs to be created)
	currentMappers := []gocloak.ProtocolMapperRepresentation{}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		expectedPath := strings.Replace(getClientProtocolMappers, "{realm}", client.RealmName, 1)
		expectedPath = strings.Replace(expectedPath, "{id}", clientID, 1)

		if r.Method == http.MethodGet && r.URL.Path == expectedPath {
			setJSONContentType(w)
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(currentMappers)

			return
		}

		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	mockClient := mocks.NewMockGoCloak(t)
	restyClient := resty.New()
	restyClient.SetBaseURL(server.URL)
	mockClient.On("RestyClient").Return(restyClient)

	// Mock GetClientID
	mockClient.On("GetClients", testifymock.Anything, "token", client.RealmName, gocloak.GetClientsParams{
		ClientID: &client.ClientId,
	}).Return([]*gocloak.Client{
		{
			ClientID: &client.ClientId,
			ID:       &clientID,
		},
	}, nil)

	// Mock CreateClientProtocolMapper to fail
	mockClient.On(
		"CreateClientProtocolMapper",
		testifymock.Anything,
		"token",
		client.RealmName,
		clientID,
		testifymock.Anything,
	).
		Return("", errors.New("insufficient permissions to create mapper")).Once()

	adapter := GoCloakAdapter{
		client:   mockClient,
		token:    &gocloak.JWT{AccessToken: "token"},
		basePath: "",
		log:      mock.NewLogr(),
	}

	err := adapter.SyncClientProtocolMapper(&client, claimedMappers, false)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unable to client create protocol mapper: insufficient permissions to create mapper")
}

func TestGoCloakAdapter_SyncClientProtocolMapper_UpdateError(t *testing.T) {
	client := dto.Client{
		RealmName: "test-realm",
		ClientId:  "test-client",
	}
	clientID := "client-uuid3"

	// Mappers that need to be updated
	claimedMappers := []gocloak.ProtocolMapperRepresentation{
		{
			Name:           gocloak.StringP("existing-mapper"),
			Protocol:       gocloak.StringP("openid-connect"),
			ProtocolMapper: gocloak.StringP("oidc-usermodel-property-mapper"),
			Config: &map[string]string{
				"user.attribute": "email", // Different from current
				"claim.name":     "email",
			},
		},
	}

	// Current mappers in Keycloak (different config - needs update)
	currentMappers := []gocloak.ProtocolMapperRepresentation{
		{
			ID:             gocloak.StringP("mapper-id"),
			Name:           gocloak.StringP("existing-mapper"),
			Protocol:       gocloak.StringP("openid-connect"),
			ProtocolMapper: gocloak.StringP("oidc-usermodel-property-mapper"),
			Config: &map[string]string{
				"user.attribute": "username", // Different from claimed
				"claim.name":     "preferred_username",
			},
		},
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		expectedPath := strings.Replace(getClientProtocolMappers, "{realm}", client.RealmName, 1)
		expectedPath = strings.Replace(expectedPath, "{id}", clientID, 1)

		if r.Method == http.MethodGet && r.URL.Path == expectedPath {
			setJSONContentType(w)
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(currentMappers)

			return
		}

		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	mockClient := mocks.NewMockGoCloak(t)
	restyClient := resty.New()
	restyClient.SetBaseURL(server.URL)
	mockClient.On("RestyClient").Return(restyClient)

	// Mock GetClientID
	mockClient.On("GetClients", testifymock.Anything, "token", client.RealmName, gocloak.GetClientsParams{
		ClientID: &client.ClientId,
	}).Return([]*gocloak.Client{
		{
			ClientID: &client.ClientId,
			ID:       &clientID,
		},
	}, nil)

	// Mock UpdateClientProtocolMapper to fail
	mockClient.On(
		"UpdateClientProtocolMapper",
		testifymock.Anything,
		"token",
		client.RealmName,
		clientID,
		"mapper-id",
		testifymock.Anything,
	).
		Return(errors.New("validation error during update")).Once()

	adapter := GoCloakAdapter{
		client:   mockClient,
		token:    &gocloak.JWT{AccessToken: "token"},
		basePath: "",
		log:      mock.NewLogr(),
	}

	err := adapter.SyncClientProtocolMapper(&client, claimedMappers, false)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unable to update client protocol mapper: validation error during update")
}

func TestGoCloakAdapter_SyncClientProtocolMapper_PrepareMapError(t *testing.T) {
	client := dto.Client{
		RealmName: "test-realm",
		ClientId:  "test-client",
	}
	clientID := "client-uuid4"

	claimedMappers := []gocloak.ProtocolMapperRepresentation{
		{
			Name:           gocloak.StringP("test-mapper"),
			Protocol:       gocloak.StringP("openid-connect"),
			ProtocolMapper: gocloak.StringP("oidc-usermodel-property-mapper"),
			Config:         &map[string]string{},
		},
	}

	mockClient := mocks.NewMockGoCloak(t)

	// Mock GetClientID to succeed
	mockClient.On("GetClients", testifymock.Anything, "token", client.RealmName, gocloak.GetClientsParams{
		ClientID: &client.ClientId,
	}).Return([]*gocloak.Client{
		{
			ClientID: &client.ClientId,
			ID:       &clientID,
		},
	}, nil)

	// Create a resty client that will fail when trying to get protocol mappers
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Close connection immediately to cause network error
		hj, ok := w.(http.Hijacker)
		if !ok {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		conn, _, _ := hj.Hijack()
		_ = conn.Close()
	}))
	defer server.Close()

	restyClient := resty.New()
	restyClient.SetBaseURL(server.URL)
	mockClient.On("RestyClient").Return(restyClient)

	adapter := GoCloakAdapter{
		client:   mockClient,
		token:    &gocloak.JWT{AccessToken: "token"},
		basePath: "",
		log:      mock.NewLogr(),
	}

	err := adapter.SyncClientProtocolMapper(&client, claimedMappers, false)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unable to prepare protocol mapper maps:")
}

func TestGoCloakAdapter_SyncClientProtocolMapper_CreateMapperError(t *testing.T) {
	client := dto.Client{
		RealmName: "test-realm",
		ClientId:  "test-client",
	}
	clientID := "client-uuid5"

	// Mapper that needs to be created (doesn't exist in current mappers)
	claimedMappers := []gocloak.ProtocolMapperRepresentation{
		{
			Name:           gocloak.StringP("new-mapper"),
			Protocol:       gocloak.StringP("openid-connect"),
			ProtocolMapper: gocloak.StringP("oidc-usermodel-property-mapper"),
			Config: &map[string]string{
				"user.attribute": "username",
			},
		},
	}

	// Empty current mappers - so the claimed mapper will need to be created
	currentMappers := []gocloak.ProtocolMapperRepresentation{}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		expectedPath := strings.Replace(getClientProtocolMappers, "{realm}", client.RealmName, 1)
		expectedPath = strings.Replace(expectedPath, "{id}", clientID, 1)

		if r.Method == http.MethodGet && r.URL.Path == expectedPath {
			setJSONContentType(w)
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(currentMappers)

			return
		}

		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	mockClient := mocks.NewMockGoCloak(t)
	restyClient := resty.New()
	restyClient.SetBaseURL(server.URL)
	mockClient.On("RestyClient").Return(restyClient)

	// Mock GetClientID
	mockClient.On("GetClients", testifymock.Anything, "token", client.RealmName, gocloak.GetClientsParams{
		ClientID: &client.ClientId,
	}).Return([]*gocloak.Client{
		{
			ClientID: &client.ClientId,
			ID:       &clientID,
		},
	}, nil)

	// Mock CreateClientProtocolMapper to fail
	mockClient.On(
		"CreateClientProtocolMapper",
		testifymock.Anything,
		"token",
		client.RealmName,
		clientID,
		testifymock.Anything,
	).
		Return("", errors.New("creation failed")).Once()

	adapter := GoCloakAdapter{
		client:   mockClient,
		token:    &gocloak.JWT{AccessToken: "token"},
		basePath: "",
		log:      mock.NewLogr(),
	}

	err := adapter.SyncClientProtocolMapper(&client, claimedMappers, false)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "error during mapperNeedsToBeCreated:")
}

func TestGoCloakAdapter_SyncClientProtocolMapper_DeleteError(t *testing.T) {
	client := dto.Client{
		RealmName: "test-realm",
		ClientId:  "test-client",
	}
	clientID := "client-uuid6"

	// No claimed mappers - so existing mappers should be deleted
	claimedMappers := []gocloak.ProtocolMapperRepresentation{}

	// Current mappers in Keycloak that need to be deleted
	currentMappers := []gocloak.ProtocolMapperRepresentation{
		{
			ID:             gocloak.StringP("mapper-to-delete-id"),
			Name:           gocloak.StringP("old-mapper"),
			Protocol:       gocloak.StringP("openid-connect"),
			ProtocolMapper: gocloak.StringP("oidc-usermodel-property-mapper"),
			Config:         &map[string]string{},
		},
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		expectedPath := strings.Replace(getClientProtocolMappers, "{realm}", client.RealmName, 1)
		expectedPath = strings.Replace(expectedPath, "{id}", clientID, 1)

		if r.Method == http.MethodGet && r.URL.Path == expectedPath {
			setJSONContentType(w)
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(currentMappers)

			return
		}

		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	mockClient := mocks.NewMockGoCloak(t)
	restyClient := resty.New()
	restyClient.SetBaseURL(server.URL)
	mockClient.On("RestyClient").Return(restyClient)

	// Mock GetClientID
	mockClient.On("GetClients", testifymock.Anything, "token", client.RealmName, gocloak.GetClientsParams{
		ClientID: &client.ClientId,
	}).Return([]*gocloak.Client{
		{
			ClientID: &client.ClientId,
			ID:       &clientID,
		},
	}, nil)

	// Mock DeleteClientProtocolMapper to fail
	mockClient.On(
		"DeleteClientProtocolMapper",
		testifymock.Anything,
		"token",
		client.RealmName,
		clientID,
		"mapper-to-delete-id",
	).
		Return(errors.New("deletion failed")).Once()

	adapter := GoCloakAdapter{
		client:   mockClient,
		token:    &gocloak.JWT{AccessToken: "token"},
		basePath: "",
		log:      mock.NewLogr(),
	}

	// Set addOnly to false so deletion will be attempted
	err := adapter.SyncClientProtocolMapper(&client, claimedMappers, false)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unable to delete client protocol mapper:")
}

func TestGoCloakAdapter_SyncClientRoles(t *testing.T) {
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
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
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

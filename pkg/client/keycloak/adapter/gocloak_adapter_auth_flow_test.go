package adapter

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Nerzal/gocloak/v12"
	"github.com/go-resty/resty/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	"github.com/epam/edp-keycloak-operator/pkg/client/keycloak/adapter/mocks"
)

type ExecFlowTestSuite struct {
	suite.Suite
	restyClient       *resty.Client
	goCloakMockClient *mocks.MockGoCloak
	adapter           *GoCloakAdapter
	realmName         string
	server            *httptest.Server
}

func (e *ExecFlowTestSuite) SetupTest() {
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
	}
	e.realmName = "realm123"
}

func (e *ExecFlowTestSuite) TearDownTest() {
	if e.server != nil {
		e.server.Close()
	}
}

// Helper function to build auth flow execution path with realm and alias substitution
func buildAuthFlowExecutionPath(realm, alias string) string {
	return strings.Replace(
		strings.Replace(authFlowExecutionGetUpdate, "{realm}", realm, 1), "{alias}", alias, 1)
}

func (e *ExecFlowTestSuite) TestCreateAuthFlowParent() {
	var (
		parentName = "parent-name"
		newFlowID  = "new-flow-id"
	)

	// Replace the default server with a test-specific one
	e.server.Close()
	e.server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		expectedPath := strings.Replace(
			strings.Replace(realmAuthFlowParentExecutions, "{realm}", e.realmName, 1),
			"{parentName}", parentName, 1)

		if r.Method == http.MethodPost && r.URL.Path == expectedPath {
			w.Header().Set("Location", fmt.Sprintf("id/%s", newFlowID))
			w.WriteHeader(http.StatusOK)

			return
		}

		w.WriteHeader(http.StatusNotFound)
	}))
	e.restyClient.SetBaseURL(e.server.URL)

	flowID, err := e.adapter.createAuthFlow(e.realmName, &KeycloakAuthFlow{
		ParentName: parentName,
	})

	assert.NoError(e.T(), err)
	assert.Equal(e.T(), flowID, newFlowID)
}

func (e *ExecFlowTestSuite) TestSyncAuthFlow() {
	flow := KeycloakAuthFlow{
		Alias:       "alias1",
		Description: "test description",
		TopLevel:    false,
		BuiltIn:     false,
		ProviderID:  "generic",
		AuthenticationExecutions: []AuthenticationExecution{
			{
				Authenticator:    "basic-auth",
				Priority:         3,
				AutheticatorFlow: false,
				Requirement:      "DISABLED",
			},
			{
				Authenticator:    "cookie",
				Priority:         2,
				AutheticatorFlow: false,
				Requirement:      "DISABLED",
				AuthenticatorConfig: &AuthenticatorConfig{
					Alias: "config-12",
					Config: map[string]string{
						"bar": "3",
					},
				},
			},
		},
	}

	existFlowID := "flow-id-1"
	newExecID := "new-exec-id"
	childExecutionID := "child-exec-id1"

	// Replace the default server with a test-specific one
	e.server.Close()
	e.server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		flowsPath := strings.Replace(authFlows, "{realm}", e.realmName, 1)
		flowDeletePath := strings.Replace(authFlow, "{realm}", e.realmName, 1)
		flowDeletePath = strings.Replace(flowDeletePath, "{id}", existFlowID, 1)
		executionsPath := strings.Replace(authFlowExecutionCreate, "{realm}", e.realmName, 1)
		execConfigPath := strings.Replace(authFlowExecutionConfig, "{realm}", e.realmName, 1)
		execConfigPath = strings.Replace(execConfigPath, "{id}", newExecID, 1)
		flowExecutionsPath := strings.Replace(authFlowExecutionGetUpdate, "{realm}", e.realmName, 1)
		flowExecutionsPath = strings.Replace(flowExecutionsPath, "{alias}", flow.Alias, 1)
		execDeletePath := strings.Replace(authFlowExecutionDelete, "{realm}", e.realmName, 1)
		execDeletePath = strings.Replace(execDeletePath, "{id}", childExecutionID, 1)
		configDeletePath := strings.Replace(authFlowConfig, "{realm}", e.realmName, 1)
		configDeletePath = strings.Replace(configDeletePath, "{id}", "authConf1", 1)

		switch {
		case r.Method == http.MethodGet && r.URL.Path == flowsPath:
			setJSONContentType(w)
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode([]KeycloakAuthFlow{{Alias: flow.Alias, ID: existFlowID},
				{Alias: "some-another-flow", ID: "321"}})
		case r.Method == http.MethodDelete && r.URL.Path == flowDeletePath:
			w.WriteHeader(http.StatusOK)
		case r.Method == http.MethodPost && r.URL.Path == flowsPath:
			w.Header().Set("Location", "id/new-flow-id")
			w.WriteHeader(http.StatusOK)
		case r.Method == http.MethodPost && r.URL.Path == executionsPath:
			w.Header().Set("Location", fmt.Sprintf("id/%s", newExecID))
			w.WriteHeader(http.StatusOK)
		case r.Method == http.MethodPost && r.URL.Path == execConfigPath:
			w.WriteHeader(http.StatusOK)
		case r.Method == http.MethodGet && r.URL.Path == flowExecutionsPath:
			setJSONContentType(w)
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode([]FlowExecution{{ID: childExecutionID, AuthenticationConfig: "authConf1"}})
		case r.Method == http.MethodDelete && r.URL.Path == execDeletePath:
			w.WriteHeader(http.StatusOK)
		case r.Method == http.MethodDelete && r.URL.Path == configDeletePath:
			w.WriteHeader(http.StatusOK)
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	e.restyClient.SetBaseURL(e.server.URL)

	err := e.adapter.SyncAuthFlow(e.realmName, &flow)
	assert.NoError(e.T(), err)
}

func (e *ExecFlowTestSuite) setupDeleteAuthFlowWithParentServer(flow KeycloakAuthFlow, deleteStatus int) {
	e.server.Close()
	e.server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && r.URL.Path == buildAuthFlowExecutionPath("realm123", "par"):
			setJSONContentType(w)
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode([]FlowExecution{{DisplayName: flow.Alias, ID: "id12"}})
		case r.Method == http.MethodDelete && r.URL.Path == strings.Replace(
			strings.Replace(authFlowExecutionDelete, "{realm}", "realm123", 1), "{id}", "id12", 1):
			w.WriteHeader(deleteStatus)
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	e.restyClient.SetBaseURL(e.server.URL)
}

func (e *ExecFlowTestSuite) TestDeleteAuthFlowWithParent() {
	flow := KeycloakAuthFlow{Alias: "al", ParentName: "par"}

	e.setupDeleteAuthFlowWithParentServer(flow, http.StatusOK)

	err := e.adapter.DeleteAuthFlow(e.realmName, &flow)
	assert.NoError(e.T(), err)
}

func (e *ExecFlowTestSuite) TestDeleteAuthFlowWithParentUnableGetFlow() {
	flow := KeycloakAuthFlow{Alias: "al", ParentName: "par"}

	// Replace the default server with a test-specific one
	e.server.Close()
	e.server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet && r.URL.Path == buildAuthFlowExecutionPath("realm123", "par") {
			w.WriteHeader(http.StatusNotFound)
			return
		}

		w.WriteHeader(http.StatusNotFound)
	}))
	e.restyClient.SetBaseURL(e.server.URL)

	err := e.adapter.DeleteAuthFlow(e.realmName, &flow)
	assert.Error(e.T(), err)
}

func (e *ExecFlowTestSuite) TestDeleteAuthFlowWithParentUnableDelete() {
	flow := KeycloakAuthFlow{Alias: "al", ParentName: "par"}

	e.setupDeleteAuthFlowWithParentServer(flow, http.StatusBadRequest)

	err := e.adapter.DeleteAuthFlow(e.realmName, &flow)
	assert.Error(e.T(), err)
}

func (e *ExecFlowTestSuite) TestDeleteAuthFlow() {
	var (
		flowAlias           = "flow-alias"
		existFlowID         = "id321"
		newBrowserFlowAlias = "alias-br-1"
	)

	// Replace the default server with a test-specific one
	e.server.Close()
	e.server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && r.URL.Path == strings.Replace(authFlows, "{realm}", e.realmName, 1):
			setJSONContentType(w)
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode([]KeycloakAuthFlow{
				{Alias: flowAlias, ID: existFlowID},
				{
					Alias: newBrowserFlowAlias,
				},
			})
		case r.Method == http.MethodDelete && r.URL.Path == strings.Replace(
			strings.Replace(authFlow, "{realm}", e.realmName, 1), "{id}", existFlowID, 1):
			w.WriteHeader(http.StatusOK)
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	e.restyClient.SetBaseURL(e.server.URL)

	e.goCloakMockClient.On("GetRealm", mock.Anything, "token", e.realmName).
		Return(&gocloak.RealmRepresentation{
			BrowserFlow: gocloak.StringP(flowAlias),
		}, nil)
	e.goCloakMockClient.On("UpdateRealm", mock.Anything, "token", gocloak.RealmRepresentation{
		BrowserFlow: gocloak.StringP(newBrowserFlowAlias),
	}).Return(nil)

	err := e.adapter.DeleteAuthFlow(e.realmName, &KeycloakAuthFlow{Alias: flowAlias})
	assert.NoError(e.T(), err)
}

func (e *ExecFlowTestSuite) TestGetAuthFlowID() {
	var (
		flow   = KeycloakAuthFlow{ParentName: "kowabunga", Alias: "alias-1"}
		flowID = "id-122"
	)

	// Replace the default server with a test-specific one
	e.server.Close()
	e.server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		expectedPath := buildAuthFlowExecutionPath(e.realmName, flow.ParentName)
		if r.Method == http.MethodGet && r.URL.Path == expectedPath {
			setJSONContentType(w)
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode([]FlowExecution{
				{
					DisplayName: flow.Alias,
					FlowID:      flowID,
				},
			})

			return
		}

		w.WriteHeader(http.StatusNotFound)
	}))
	e.restyClient.SetBaseURL(e.server.URL)

	id, err := e.adapter.getAuthFlowID(e.realmName, &flow)

	assert.NoError(e.T(), err)
	assert.Equal(e.T(), flowID, id)
}

func (e *ExecFlowTestSuite) TestSetRealmBrowserFlow() {
	realm := gocloak.RealmRepresentation{
		BrowserFlow: gocloak.StringP("flow1"),
	}

	e.goCloakMockClient.On("GetRealm", mock.Anything, "token", "realm1").Return(&realm, nil)
	e.goCloakMockClient.On("UpdateRealm", mock.Anything, "token", realm).Return(nil)

	err := e.adapter.SetRealmBrowserFlow(context.Background(), "realm1", "flow1")
	assert.NoError(e.T(), err)
}

func (e *ExecFlowTestSuite) TestSetRealmBrowserFlow_FailureGetRealm() {
	mockErr := errors.New("mock err")
	e.goCloakMockClient.On("GetRealm", mock.Anything, "token", "realm1").Return(nil, mockErr)

	err := e.adapter.SetRealmBrowserFlow(context.Background(), "realm1", "flow1")
	assert.Error(e.T(), err)
	assert.ErrorIs(e.T(), err, mockErr)
}

func (e *ExecFlowTestSuite) TestSetRealmBrowserFlow_FailureUpdateRealm() {
	mockErr := errors.New("mock err")

	realm := gocloak.RealmRepresentation{
		BrowserFlow: gocloak.StringP("flow1"),
	}

	e.goCloakMockClient.On("GetRealm", mock.Anything, "token", "realm1").Return(&realm, nil)
	e.goCloakMockClient.On("UpdateRealm", mock.Anything, "token", realm).Return(mockErr)

	err := e.adapter.SetRealmBrowserFlow(context.Background(), "realm1", "flow1")
	assert.Error(e.T(), err)
	assert.ErrorIs(e.T(), err, mockErr)
}

func (e *ExecFlowTestSuite) TestSyncBaseAuthFlow() {
	var (
		flow = KeycloakAuthFlow{
			Alias: "flow2",
			AuthenticationExecutions: []AuthenticationExecution{
				{
					AutheticatorFlow: true,
					Alias:            "child-flow",
				},
			},
		}
		flowID = "flow-id-2"
	)

	// Replace the default server with a test-specific one
	e.server.Close()
	e.server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && r.URL.Path == strings.Replace(authFlows, "{realm}", e.realmName, 1):
			setJSONContentType(w)
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode([]KeycloakAuthFlow{})
		case r.Method == http.MethodPost && r.URL.Path == strings.Replace(authFlows, "{realm}", e.realmName, 1):
			w.Header().Set("Location", fmt.Sprintf("id/%s", flowID))
			w.WriteHeader(http.StatusOK)
		case r.Method == http.MethodGet && r.URL.Path == strings.Replace(
			strings.Replace(authFlowExecutionGetUpdate, "{realm}", e.realmName, 1), "{alias}", flow.Alias, 1):
			setJSONContentType(w)
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode([]FlowExecution{{}})
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	e.restyClient.SetBaseURL(e.server.URL)

	_, err := e.adapter.syncBaseAuthFlow(e.realmName, &flow)

	assert.Error(e.T(), err)
	assert.EqualError(e.T(), err, "child flows validation failed: not all child flows created")
}

func (e *ExecFlowTestSuite) TestSyncBaseAuthFlowShouldUpdateChildFlowRequirement() {
	flow := KeycloakAuthFlow{
		Alias:            "flow3",
		ParentName:       "parent",
		ChildRequirement: "REQUIRED",
	}
	flowID := "flow-id-3"

	// Replace the default server with a test-specific one
	e.server.Close()
	e.server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		parentPath := buildAuthFlowExecutionPath(e.realmName, flow.ParentName)
		aliasPath := buildAuthFlowExecutionPath(e.realmName, flow.Alias)

		switch {
		case r.Method == http.MethodGet && r.URL.Path == parentPath:
			setJSONContentType(w)
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode([]FlowExecution{{
				DisplayName: flow.Alias,
				FlowID:      flowID,
			}})
		case r.Method == http.MethodGet && r.URL.Path == aliasPath:
			setJSONContentType(w)
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode([]FlowExecution{})
		case r.Method == http.MethodPut && r.URL.Path == parentPath:
			setJSONContentType(w)
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(map[string]string{})
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	e.restyClient.SetBaseURL(e.server.URL)

	_, err := e.adapter.syncBaseAuthFlow(e.realmName, &flow)

	assert.NoError(e.T(), err)
}

func (e *ExecFlowTestSuite) TestSyncBaseAuthFlowFailedUpdateChildFlowRequirement() {
	flow := KeycloakAuthFlow{
		Alias:            "flow4",
		ParentName:       "parent",
		ChildRequirement: "REQUIRED",
	}
	flowID := "flow-id-4"

	// Replace the default server with a test-specific one
	e.server.Close()
	e.server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		parentPath := buildAuthFlowExecutionPath(e.realmName, flow.ParentName)
		aliasPath := buildAuthFlowExecutionPath(e.realmName, flow.Alias)

		switch {
		case r.Method == http.MethodGet && r.URL.Path == parentPath:
			setJSONContentType(w)
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode([]FlowExecution{{
				DisplayName: flow.Alias,
				FlowID:      flowID,
			}})
		case r.Method == http.MethodGet && r.URL.Path == aliasPath:
			setJSONContentType(w)
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode([]FlowExecution{})
		case r.Method == http.MethodPut && r.URL.Path == parentPath:
			setJSONContentType(w)
			w.WriteHeader(http.StatusBadRequest)
			_ = json.NewEncoder(w).Encode(map[string]string{})
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	e.restyClient.SetBaseURL(e.server.URL)

	_, err := e.adapter.syncBaseAuthFlow(e.realmName, &flow)

	require.Error(e.T(), err)
	assert.Contains(e.T(), err.Error(), "unable to update flow execution requirement")
}

func (e *ExecFlowTestSuite) TestSyncBaseAuthFlowFailedToGetFlowExecution() {
	flow := KeycloakAuthFlow{
		Alias:            "flow1",
		ParentName:       "parent",
		ChildRequirement: "REQUIRED",
	}

	// Replace the default server with a test-specific one
	e.server.Close()
	e.server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		expectedPath := buildAuthFlowExecutionPath(e.realmName, flow.ParentName)
		if r.Method == http.MethodGet && r.URL.Path == expectedPath {
			setJSONContentType(w)
			w.WriteHeader(http.StatusInternalServerError)
			_ = json.NewEncoder(w).Encode([]FlowExecution{})

			return
		}

		w.WriteHeader(http.StatusNotFound)
	}))
	e.restyClient.SetBaseURL(e.server.URL)

	_, err := e.adapter.syncBaseAuthFlow(e.realmName, &flow)

	require.Error(e.T(), err)
	assert.Contains(e.T(), err.Error(), "unable to get auth flow")
}

func (e *ExecFlowTestSuite) TestSyncBaseAuthFlowFailedToCreateChildFlow() {
	flow := KeycloakAuthFlow{
		Alias:            "flow1",
		ParentName:       "parent",
		ChildRequirement: "REQUIRED",
	}

	// Replace the default server with a test-specific one
	e.server.Close()
	e.server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		parentPath := buildAuthFlowExecutionPath(e.realmName, flow.ParentName)

		switch {
		case r.Method == http.MethodGet && r.URL.Path == parentPath:
			setJSONContentType(w)
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode([]FlowExecution{})
		case r.Method == http.MethodPost && r.URL.Path == strings.Replace(
			strings.Replace(realmAuthFlowParentExecutions, "{realm}", e.realmName, 1), "{parentName}", flow.ParentName, 1):
			setJSONContentType(w)
			w.WriteHeader(http.StatusInternalServerError)
			_ = json.NewEncoder(w).Encode(map[string]string{})
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	e.restyClient.SetBaseURL(e.server.URL)

	_, err := e.adapter.syncBaseAuthFlow(e.realmName, &flow)

	require.Error(e.T(), err)
	assert.Contains(e.T(), err.Error(), "unable to create child auth flow in realm")
}

func (e *ExecFlowTestSuite) TestGetFlowExecutionID() {
	flow := KeycloakAuthFlow{ParentName: "parent", Alias: "fff"}

	// First test: server returns error - should get error message
	e.server.Close()
	e.server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	e.restyClient.SetBaseURL(e.server.URL)

	_, err := e.adapter.getFlowExecutionID(e.realmName, &flow)
	assert.Error(e.T(), err)
	assert.Contains(e.T(), err.Error(), "unable to get auth flow executions")

	// Second test: empty executions list - should get "auth flow not found" error
	e.server.Close()
	e.server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		expectedPath := buildAuthFlowExecutionPath("realm123", "parent")
		if r.Method == http.MethodGet && r.URL.Path == expectedPath {
			setJSONContentType(w)
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode([]FlowExecution{})

			return
		}

		w.WriteHeader(http.StatusNotFound)
	}))
	e.restyClient.SetBaseURL(e.server.URL)

	_, err = e.adapter.getFlowExecutionID(e.realmName, &flow)
	assert.Error(e.T(), err)
	assert.EqualError(e.T(), err, "auth flow not found")

	// Third test: execution found - should succeed
	e.server.Close()
	e.server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		expectedPath := buildAuthFlowExecutionPath("realm123", "parent")
		if r.Method == http.MethodGet && r.URL.Path == expectedPath {
			setJSONContentType(w)
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode([]FlowExecution{
				{
					DisplayName: flow.Alias,
					ID:          "as12",
				},
			})

			return
		}

		w.WriteHeader(http.StatusNotFound)
	}))
	e.restyClient.SetBaseURL(e.server.URL)

	_, err = e.adapter.getFlowExecutionID(e.realmName, &flow)
	assert.NoError(e.T(), err)
}

func (e *ExecFlowTestSuite) TestAdjustChildFlowsPriority() {
	var (
		flow = KeycloakAuthFlow{
			Alias: "flow-1",
			AuthenticationExecutions: []AuthenticationExecution{
				{
					AutheticatorFlow: true,
					Alias:            "child-flow-1",
					Priority:         1,
					Requirement:      "REQUIRED",
				},
				{
					Priority: 0,
				},
			},
		}
		flowExecutionID = "flow-exec-id-1"
	)

	// Replace the default server with a test-specific one
	e.server.Close()
	e.server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		aliasPath := strings.Replace(
			strings.Replace(authFlowExecutionGetUpdate, "{realm}", e.realmName, 1), "{alias}", flow.Alias, 1)

		switch {
		case r.Method == http.MethodGet && r.URL.Path == aliasPath:
			setJSONContentType(w)
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode([]FlowExecution{
				{
					AuthenticationFlow: true,
					DisplayName:        flow.AuthenticationExecutions[0].Alias,
					Index:              0,
					ID:                 flowExecutionID,
					Requirement:        "ALTERNATIVE",
				},
			})
		case r.Method == http.MethodPost && r.URL.Path == strings.Replace(
			strings.Replace(lowerExecutionPriority, "{realm}", e.realmName, 1), "{id}", flowExecutionID, 1):
			w.WriteHeader(http.StatusOK)
		case r.Method == http.MethodPut && r.URL.Path == aliasPath:
			w.WriteHeader(http.StatusOK)
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	e.restyClient.SetBaseURL(e.server.URL)

	err := e.adapter.adjustChildFlowsPriority(e.realmName, &flow)
	assert.NoError(e.T(), err)
}

func TestExecFlowTestSuite(t *testing.T) {
	suite.Run(t, new(ExecFlowTestSuite))
}

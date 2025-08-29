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

type ServerHandler func(w http.ResponseWriter, r *http.Request)
type ServerSetupOption func(*ServerConfig)

type ServerConfig struct {
	Handlers      map[string]map[string]ServerHandler // method -> path -> handler
	DefaultStatus int
}

type PathBuilder struct {
	realm string
}

func NewPathBuilder(realm string) *PathBuilder {
	return &PathBuilder{realm: realm}
}

func (pb *PathBuilder) AuthFlows() string {
	return strings.Replace(authFlows, "{realm}", pb.realm, 1)
}

func (pb *PathBuilder) AuthFlow(id string) string {
	return strings.Replace(strings.Replace(authFlow, "{realm}", pb.realm, 1), "{id}", id, 1)
}

func (pb *PathBuilder) AuthFlowExecution(alias string) string {
	return strings.Replace(strings.Replace(authFlowExecutionGetUpdate, "{realm}", pb.realm, 1), "{alias}", alias, 1)
}

func (pb *PathBuilder) AuthFlowExecutionCreate() string {
	return strings.Replace(authFlowExecutionCreate, "{realm}", pb.realm, 1)
}

func (pb *PathBuilder) AuthFlowExecutionDelete(id string) string {
	return strings.Replace(strings.Replace(authFlowExecutionDelete, "{realm}", pb.realm, 1), "{id}", id, 1)
}

func (pb *PathBuilder) AuthFlowExecutionConfig(id string) string {
	return strings.Replace(strings.Replace(authFlowExecutionConfig, "{realm}", pb.realm, 1), "{id}", id, 1)
}

func (pb *PathBuilder) AuthFlowConfig(id string) string {
	return strings.Replace(strings.Replace(authFlowConfig, "{realm}", pb.realm, 1), "{id}", id, 1)
}

func (pb *PathBuilder) RealmAuthFlowParentExecutions(parentName string) string {
	return strings.Replace(
		strings.Replace(realmAuthFlowParentExecutions, "{realm}", pb.realm, 1),
		"{parentName}",
		parentName,
		1,
	)
}

func (pb *PathBuilder) LowerExecutionPriority(id string) string {
	return strings.Replace(strings.Replace(lowerExecutionPriority, "{realm}", pb.realm, 1), "{id}", id, 1)
}

type ExecFlowTestSuite struct {
	suite.Suite
	restyClient       *resty.Client
	goCloakMockClient *mocks.MockGoCloak
	adapter           *GoCloakAdapter
	realmName         string
	server            *httptest.Server
	pathBuilder       *PathBuilder
}

func (e *ExecFlowTestSuite) SetupTest() {
	e.realmName = "realm123"
	e.pathBuilder = NewPathBuilder(e.realmName)
	e.setupDefaultServer()

	e.goCloakMockClient = mocks.NewMockGoCloak(e.T())
	e.goCloakMockClient.On("RestyClient").Return(e.restyClient).Maybe()

	e.adapter = &GoCloakAdapter{
		client: e.goCloakMockClient,
		token:  &gocloak.JWT{AccessToken: "token"},
	}
}

func (e *ExecFlowTestSuite) setupDefaultServer() {
	e.server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	e.restyClient = resty.New()
	e.restyClient.SetBaseURL(e.server.URL)
}

func (e *ExecFlowTestSuite) setupServerWithConfig(config *ServerConfig) {
	if e.server != nil {
		e.server.Close()
	}

	defaultStatus := http.StatusNotFound
	if config.DefaultStatus != 0 {
		defaultStatus = config.DefaultStatus
	}

	e.server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if config.Handlers != nil {
			if methodHandlers, exists := config.Handlers[r.Method]; exists {
				if handler, exists := methodHandlers[r.URL.Path]; exists {
					handler(w, r)
					return
				}
			}
		}

		w.WriteHeader(defaultStatus)
	}))
	e.restyClient.SetBaseURL(e.server.URL)
}

func (e *ExecFlowTestSuite) TearDownTest() {
	if e.server != nil {
		e.server.Close()
	}
}

func (e *ExecFlowTestSuite) TestCreateAuthFlowParent() {
	var (
		parentName = "parent-name"
		newFlowID  = "new-flow-id"
	)

	e.setupServerWithConfig(&ServerConfig{
		Handlers: map[string]map[string]ServerHandler{
			http.MethodPost: {
				e.pathBuilder.RealmAuthFlowParentExecutions(parentName): func(w http.ResponseWriter, r *http.Request) {
					w.Header().Set("Location", fmt.Sprintf("id/%s", newFlowID))
					w.WriteHeader(http.StatusOK)
				},
			},
		},
	})

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

	e.setupServerWithConfig(&ServerConfig{
		Handlers: map[string]map[string]ServerHandler{
			http.MethodGet: {
				e.pathBuilder.AuthFlows(): func(w http.ResponseWriter, r *http.Request) {
					setJSONContentType(w)
					w.WriteHeader(http.StatusOK)
					_ = json.NewEncoder(w).Encode([]KeycloakAuthFlow{{Alias: flow.Alias, ID: existFlowID},
						{Alias: "some-another-flow", ID: "321"}})
				},
				e.pathBuilder.AuthFlowExecution(flow.Alias): func(w http.ResponseWriter, r *http.Request) {
					setJSONContentType(w)
					w.WriteHeader(http.StatusOK)
					_ = json.NewEncoder(w).Encode([]FlowExecution{{ID: childExecutionID, AuthenticationConfig: "authConf1"}})
				},
			},
			http.MethodPost: {
				e.pathBuilder.AuthFlows(): func(w http.ResponseWriter, r *http.Request) {
					w.Header().Set("Location", "id/new-flow-id")
					w.WriteHeader(http.StatusOK)
				},
				e.pathBuilder.AuthFlowExecutionCreate(): func(w http.ResponseWriter, r *http.Request) {
					w.Header().Set("Location", fmt.Sprintf("id/%s", newExecID))
					w.WriteHeader(http.StatusOK)
				},
				e.pathBuilder.AuthFlowExecutionConfig(newExecID): func(w http.ResponseWriter, r *http.Request) {
					w.WriteHeader(http.StatusOK)
				},
			},
			http.MethodDelete: {
				e.pathBuilder.AuthFlow(existFlowID): func(w http.ResponseWriter, r *http.Request) {
					w.WriteHeader(http.StatusOK)
				},
				e.pathBuilder.AuthFlowExecutionDelete(childExecutionID): func(w http.ResponseWriter, r *http.Request) {
					w.WriteHeader(http.StatusOK)
				},
				e.pathBuilder.AuthFlowConfig("authConf1"): func(w http.ResponseWriter, r *http.Request) {
					w.WriteHeader(http.StatusOK)
				},
			},
		},
	})

	err := e.adapter.SyncAuthFlow(e.realmName, &flow)
	assert.NoError(e.T(), err)
}

func (e *ExecFlowTestSuite) setupDeleteAuthFlowWithParentServer(flow KeycloakAuthFlow, deleteStatus int) {
	e.setupServerWithConfig(&ServerConfig{
		Handlers: map[string]map[string]ServerHandler{
			http.MethodGet: {
				e.pathBuilder.AuthFlowExecution("par"): func(w http.ResponseWriter, r *http.Request) {
					setJSONContentType(w)
					w.WriteHeader(http.StatusOK)
					_ = json.NewEncoder(w).Encode([]FlowExecution{{DisplayName: flow.Alias, ID: "id12"}})
				},
			},
			http.MethodDelete: {
				e.pathBuilder.AuthFlowExecutionDelete("id12"): func(w http.ResponseWriter, r *http.Request) {
					w.WriteHeader(deleteStatus)
				},
			},
		},
	})
}

func (e *ExecFlowTestSuite) TestDeleteAuthFlowWithParent() {
	flow := KeycloakAuthFlow{Alias: "al", ParentName: "par"}

	e.setupDeleteAuthFlowWithParentServer(flow, http.StatusOK)

	err := e.adapter.DeleteAuthFlow(e.realmName, &flow)
	assert.NoError(e.T(), err)
}

func (e *ExecFlowTestSuite) TestDeleteAuthFlowWithParentUnableGetFlow() {
	flow := KeycloakAuthFlow{Alias: "al", ParentName: "par"}

	e.setupServerWithConfig(&ServerConfig{
		Handlers: map[string]map[string]ServerHandler{
			http.MethodGet: {
				e.pathBuilder.AuthFlowExecution("par"): func(w http.ResponseWriter, r *http.Request) {
					w.WriteHeader(http.StatusNotFound)
				},
			},
		},
	})

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

	e.setupServerWithConfig(&ServerConfig{
		Handlers: map[string]map[string]ServerHandler{
			http.MethodGet: {
				e.pathBuilder.AuthFlows(): func(w http.ResponseWriter, r *http.Request) {
					setJSONContentType(w)
					w.WriteHeader(http.StatusOK)
					_ = json.NewEncoder(w).Encode([]KeycloakAuthFlow{
						{Alias: flowAlias, ID: existFlowID},
						{
							Alias: newBrowserFlowAlias,
						},
					})
				},
			},
			http.MethodDelete: {
				e.pathBuilder.AuthFlow(existFlowID): func(w http.ResponseWriter, r *http.Request) {
					w.WriteHeader(http.StatusOK)
				},
			},
		},
	})

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

	e.setupServerWithConfig(&ServerConfig{
		Handlers: map[string]map[string]ServerHandler{
			http.MethodGet: {
				e.pathBuilder.AuthFlowExecution(flow.ParentName): func(w http.ResponseWriter, r *http.Request) {
					setJSONContentType(w)
					w.WriteHeader(http.StatusOK)
					_ = json.NewEncoder(w).Encode([]FlowExecution{
						{
							DisplayName: flow.Alias,
							FlowID:      flowID,
						},
					})
				},
			},
		},
	})

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

	e.setupServerWithConfig(&ServerConfig{
		Handlers: map[string]map[string]ServerHandler{
			http.MethodGet: {
				e.pathBuilder.AuthFlows(): func(w http.ResponseWriter, r *http.Request) {
					setJSONContentType(w)
					w.WriteHeader(http.StatusOK)
					_ = json.NewEncoder(w).Encode([]KeycloakAuthFlow{})
				},
				e.pathBuilder.AuthFlowExecution(flow.Alias): func(w http.ResponseWriter, r *http.Request) {
					setJSONContentType(w)
					w.WriteHeader(http.StatusOK)
					_ = json.NewEncoder(w).Encode([]FlowExecution{{}})
				},
			},
			http.MethodPost: {
				e.pathBuilder.AuthFlows(): func(w http.ResponseWriter, r *http.Request) {
					w.Header().Set("Location", fmt.Sprintf("id/%s", flowID))
					w.WriteHeader(http.StatusOK)
				},
			},
		},
	})

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

	parentPath := e.pathBuilder.AuthFlowExecution(flow.ParentName)
	aliasPath := e.pathBuilder.AuthFlowExecution(flow.Alias)

	e.setupServerWithConfig(&ServerConfig{
		Handlers: map[string]map[string]ServerHandler{
			http.MethodGet: {
				parentPath: func(w http.ResponseWriter, r *http.Request) {
					setJSONContentType(w)
					w.WriteHeader(http.StatusOK)
					_ = json.NewEncoder(w).Encode([]FlowExecution{{
						DisplayName: flow.Alias,
						FlowID:      flowID,
					}})
				},
				aliasPath: func(w http.ResponseWriter, r *http.Request) {
					setJSONContentType(w)
					w.WriteHeader(http.StatusOK)
					_ = json.NewEncoder(w).Encode([]FlowExecution{})
				},
			},
			http.MethodPut: {
				parentPath: func(w http.ResponseWriter, r *http.Request) {
					setJSONContentType(w)
					w.WriteHeader(http.StatusOK)
					_ = json.NewEncoder(w).Encode(map[string]string{})
				},
			},
		},
	})

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

	parentPath := e.pathBuilder.AuthFlowExecution(flow.ParentName)
	aliasPath := e.pathBuilder.AuthFlowExecution(flow.Alias)

	e.setupServerWithConfig(&ServerConfig{
		Handlers: map[string]map[string]ServerHandler{
			http.MethodGet: {
				parentPath: func(w http.ResponseWriter, r *http.Request) {
					setJSONContentType(w)
					w.WriteHeader(http.StatusOK)
					_ = json.NewEncoder(w).Encode([]FlowExecution{{
						DisplayName: flow.Alias,
						FlowID:      flowID,
					}})
				},
				aliasPath: func(w http.ResponseWriter, r *http.Request) {
					setJSONContentType(w)
					w.WriteHeader(http.StatusOK)
					_ = json.NewEncoder(w).Encode([]FlowExecution{})
				},
			},
			http.MethodPut: {
				parentPath: func(w http.ResponseWriter, r *http.Request) {
					setJSONContentType(w)
					w.WriteHeader(http.StatusBadRequest)
					_ = json.NewEncoder(w).Encode(map[string]string{})
				},
			},
		},
	})

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

	e.setupServerWithConfig(&ServerConfig{
		Handlers: map[string]map[string]ServerHandler{
			http.MethodGet: {
				e.pathBuilder.AuthFlowExecution(flow.ParentName): func(w http.ResponseWriter, r *http.Request) {
					setJSONContentType(w)
					w.WriteHeader(http.StatusInternalServerError)
					_ = json.NewEncoder(w).Encode([]FlowExecution{})
				},
			},
		},
	})

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

	e.setupServerWithConfig(&ServerConfig{
		Handlers: map[string]map[string]ServerHandler{
			http.MethodGet: {
				e.pathBuilder.AuthFlowExecution(flow.ParentName): func(w http.ResponseWriter, r *http.Request) {
					setJSONContentType(w)
					w.WriteHeader(http.StatusOK)
					_ = json.NewEncoder(w).Encode([]FlowExecution{})
				},
			},
			http.MethodPost: {
				e.pathBuilder.RealmAuthFlowParentExecutions(flow.ParentName): func(w http.ResponseWriter, r *http.Request) {
					setJSONContentType(w)
					w.WriteHeader(http.StatusInternalServerError)
					_ = json.NewEncoder(w).Encode(map[string]string{})
				},
			},
		},
	})

	_, err := e.adapter.syncBaseAuthFlow(e.realmName, &flow)

	require.Error(e.T(), err)
	assert.Contains(e.T(), err.Error(), "unable to create child auth flow in realm")
}

func (e *ExecFlowTestSuite) TestGetFlowExecutionID() {
	flow := KeycloakAuthFlow{ParentName: "parent", Alias: "fff"}

	// First test: server returns error - should get error message
	e.setupDefaultServer()

	_, err := e.adapter.getFlowExecutionID(e.realmName, &flow)
	assert.Error(e.T(), err)
	assert.Contains(e.T(), err.Error(), "unable to get auth flow executions")
}

func (e *ExecFlowTestSuite) TestGetFlowExecutionID_EmptyList() {
	flow := KeycloakAuthFlow{ParentName: "parent", Alias: "fff"}

	// Empty executions list - should get "auth flow not found" error
	e.setupServerWithConfig(&ServerConfig{
		Handlers: map[string]map[string]ServerHandler{
			http.MethodGet: {
				e.pathBuilder.AuthFlowExecution("parent"): func(w http.ResponseWriter, r *http.Request) {
					setJSONContentType(w)
					w.WriteHeader(http.StatusOK)
					_ = json.NewEncoder(w).Encode([]FlowExecution{})
				},
			},
		},
	})

	_, err := e.adapter.getFlowExecutionID(e.realmName, &flow)
	assert.Error(e.T(), err)
	assert.EqualError(e.T(), err, "auth flow not found")
}

func (e *ExecFlowTestSuite) TestGetFlowExecutionID_Success() {
	flow := KeycloakAuthFlow{ParentName: "parent", Alias: "fff"}

	// Execution found - should succeed
	e.setupServerWithConfig(&ServerConfig{
		Handlers: map[string]map[string]ServerHandler{
			http.MethodGet: {
				e.pathBuilder.AuthFlowExecution("parent"): func(w http.ResponseWriter, r *http.Request) {
					setJSONContentType(w)
					w.WriteHeader(http.StatusOK)
					_ = json.NewEncoder(w).Encode([]FlowExecution{
						{
							DisplayName: flow.Alias,
							ID:          "as12",
						},
					})
				},
			},
		},
	})

	_, err := e.adapter.getFlowExecutionID(e.realmName, &flow)
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

	aliasPath := e.pathBuilder.AuthFlowExecution(flow.Alias)

	e.setupServerWithConfig(&ServerConfig{
		Handlers: map[string]map[string]ServerHandler{
			http.MethodGet: {
				aliasPath: func(w http.ResponseWriter, r *http.Request) {
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
				},
			},
			http.MethodPost: {
				e.pathBuilder.LowerExecutionPriority(flowExecutionID): func(w http.ResponseWriter, r *http.Request) {
					w.WriteHeader(http.StatusOK)
				},
			},
			http.MethodPut: {
				aliasPath: func(w http.ResponseWriter, r *http.Request) {
					w.WriteHeader(http.StatusOK)
				},
			},
		},
	})

	err := e.adapter.adjustChildFlowsPriority(e.realmName, &flow)
	assert.NoError(e.T(), err)
}

// Error scenario tests for DeleteAuthFlow
func (e *ExecFlowTestSuite) TestDeleteAuthFlow_ErrorGettingAuthFlowID() {
	flowAlias := "test-flow"

	e.setupServerWithConfig(&ServerConfig{
		Handlers: map[string]map[string]ServerHandler{
			http.MethodGet: {
				e.pathBuilder.AuthFlows(): func(w http.ResponseWriter, r *http.Request) {
					w.WriteHeader(http.StatusInternalServerError)
				},
			},
		},
	})

	err := e.adapter.DeleteAuthFlow(e.realmName, &KeycloakAuthFlow{Alias: flowAlias})
	assert.Error(e.T(), err)
	assert.Contains(e.T(), err.Error(), "unable to get auth flow")
}

func (e *ExecFlowTestSuite) TestDeleteAuthFlow_ErrorUnsettingBrowserFlow() {
	var (
		flowAlias   = "test-flow"
		existFlowID = "flow-id-123"
	)

	e.setupServerWithConfig(&ServerConfig{
		Handlers: map[string]map[string]ServerHandler{
			http.MethodGet: {
				e.pathBuilder.AuthFlows(): func(w http.ResponseWriter, r *http.Request) {
					setJSONContentType(w)
					w.WriteHeader(http.StatusOK)
					_ = json.NewEncoder(w).Encode([]KeycloakAuthFlow{
						{Alias: flowAlias, ID: existFlowID},
					})
				},
			},
		},
	})

	e.goCloakMockClient.On("GetRealm", mock.Anything, "token", e.realmName).
		Return(nil, errors.New("realm error"))

	err := e.adapter.DeleteAuthFlow(e.realmName, &KeycloakAuthFlow{Alias: flowAlias})
	assert.Error(e.T(), err)
	assert.Contains(e.T(), err.Error(), "unable to unset browser flow for realm")
}

func (e *ExecFlowTestSuite) TestDeleteAuthFlow_ErrorDeletingAuthFlow() {
	var (
		flowAlias           = "test-flow"
		existFlowID         = "flow-id-123"
		newBrowserFlowAlias = "replacement-flow"
	)

	e.setupServerWithConfig(&ServerConfig{
		Handlers: map[string]map[string]ServerHandler{
			http.MethodGet: {
				e.pathBuilder.AuthFlows(): func(w http.ResponseWriter, r *http.Request) {
					setJSONContentType(w)
					w.WriteHeader(http.StatusOK)
					_ = json.NewEncoder(w).Encode([]KeycloakAuthFlow{
						{Alias: flowAlias, ID: existFlowID},
						{Alias: newBrowserFlowAlias, ID: "replacement-id"},
					})
				},
			},
			http.MethodDelete: {
				e.pathBuilder.AuthFlow(existFlowID): func(w http.ResponseWriter, r *http.Request) {
					w.WriteHeader(http.StatusInternalServerError)
				},
			},
		},
	})

	e.goCloakMockClient.On("GetRealm", mock.Anything, "token", e.realmName).
		Return(&gocloak.RealmRepresentation{
			BrowserFlow: gocloak.StringP(flowAlias),
		}, nil)
	e.goCloakMockClient.On("UpdateRealm", mock.Anything, "token", mock.Anything).Return(nil)

	err := e.adapter.DeleteAuthFlow(e.realmName, &KeycloakAuthFlow{Alias: flowAlias})
	assert.Error(e.T(), err)
	assert.Contains(e.T(), err.Error(), "unable to delete auth flow")
}

func (e *ExecFlowTestSuite) TestDeleteAuthFlow_WithParent_ErrorGettingFlowExecID() {
	flow := KeycloakAuthFlow{Alias: "child-flow", ParentName: "parent"}

	e.setupServerWithConfig(&ServerConfig{
		Handlers: map[string]map[string]ServerHandler{
			http.MethodGet: {
				e.pathBuilder.AuthFlowExecution("parent"): func(w http.ResponseWriter, r *http.Request) {
					w.WriteHeader(http.StatusInternalServerError)
				},
			},
		},
	})

	err := e.adapter.DeleteAuthFlow(e.realmName, &flow)
	assert.Error(e.T(), err)
	assert.Contains(e.T(), err.Error(), "unable to get flow exec id")
}

func (e *ExecFlowTestSuite) TestDeleteAuthFlow_WithParent_ErrorDeletingExecution() {
	flow := KeycloakAuthFlow{Alias: "child-flow", ParentName: "parent"}

	e.setupServerWithConfig(&ServerConfig{
		Handlers: map[string]map[string]ServerHandler{
			http.MethodGet: {
				e.pathBuilder.AuthFlowExecution("parent"): func(w http.ResponseWriter, r *http.Request) {
					setJSONContentType(w)
					w.WriteHeader(http.StatusOK)
					_ = json.NewEncoder(w).Encode([]FlowExecution{{DisplayName: flow.Alias, ID: "exec-id"}})
				},
			},
			http.MethodDelete: {
				e.pathBuilder.AuthFlowExecutionDelete("exec-id"): func(w http.ResponseWriter, r *http.Request) {
					w.WriteHeader(http.StatusInternalServerError)
				},
			},
		},
	})

	err := e.adapter.DeleteAuthFlow(e.realmName, &flow)
	assert.Error(e.T(), err)
	assert.Contains(e.T(), err.Error(), "unable to delete execution")
}

// Error scenario tests for SyncAuthFlow
func (e *ExecFlowTestSuite) TestSyncAuthFlow_ErrorInSyncBaseAuthFlow() {
	flow := KeycloakAuthFlow{
		Alias: "test-flow",
	}

	e.setupServerWithConfig(&ServerConfig{
		Handlers: map[string]map[string]ServerHandler{
			http.MethodGet: {
				e.pathBuilder.AuthFlows(): func(w http.ResponseWriter, r *http.Request) {
					w.WriteHeader(http.StatusInternalServerError)
				},
			},
		},
	})

	err := e.adapter.SyncAuthFlow(e.realmName, &flow)
	assert.Error(e.T(), err)
	assert.Contains(e.T(), err.Error(), "unable to sync base auth flow")
}

func (e *ExecFlowTestSuite) TestSyncAuthFlow_ErrorAddingAuthFlowExecution() {
	flow := KeycloakAuthFlow{
		Alias: "test-flow",
		AuthenticationExecutions: []AuthenticationExecution{
			{
				Authenticator:    "basic-auth",
				Priority:         1,
				AutheticatorFlow: false,
				Requirement:      "REQUIRED",
			},
		},
	}

	e.setupServerWithConfig(&ServerConfig{
		Handlers: map[string]map[string]ServerHandler{
			http.MethodGet: {
				e.pathBuilder.AuthFlows(): func(w http.ResponseWriter, r *http.Request) {
					setJSONContentType(w)
					w.WriteHeader(http.StatusOK)
					_ = json.NewEncoder(w).Encode([]KeycloakAuthFlow{})
				},
				e.pathBuilder.AuthFlowExecution(flow.Alias): func(w http.ResponseWriter, r *http.Request) {
					setJSONContentType(w)
					w.WriteHeader(http.StatusOK)
					_ = json.NewEncoder(w).Encode([]FlowExecution{})
				},
			},
			http.MethodPost: {
				e.pathBuilder.AuthFlows(): func(w http.ResponseWriter, r *http.Request) {
					w.Header().Set("Location", "id/new-flow-id")
					w.WriteHeader(http.StatusOK)
				},
				e.pathBuilder.AuthFlowExecutionCreate(): func(w http.ResponseWriter, r *http.Request) {
					w.WriteHeader(http.StatusInternalServerError)
				},
			},
		},
	})

	err := e.adapter.SyncAuthFlow(e.realmName, &flow)
	assert.Error(e.T(), err)
	assert.Contains(e.T(), err.Error(), "unable to add auth execution")
}

func (e *ExecFlowTestSuite) TestSyncAuthFlow_ErrorAdjustingChildFlowsPriority() {
	flow := KeycloakAuthFlow{
		Alias: "test-flow",
		AuthenticationExecutions: []AuthenticationExecution{
			{
				AutheticatorFlow: true,
				Alias:            "child-flow",
				Priority:         1,
				Requirement:      "REQUIRED",
			},
		},
	}

	// Counter to handle multiple calls to the same endpoint
	callCount := 0

	e.setupServerWithConfig(&ServerConfig{
		Handlers: map[string]map[string]ServerHandler{
			http.MethodGet: {
				e.pathBuilder.AuthFlows(): func(w http.ResponseWriter, r *http.Request) {
					setJSONContentType(w)
					w.WriteHeader(http.StatusOK)
					_ = json.NewEncoder(w).Encode([]KeycloakAuthFlow{})
				},
				e.pathBuilder.AuthFlowExecution(flow.Alias): func(w http.ResponseWriter, r *http.Request) {
					callCount++
					if callCount == 1 {
						// First call for validateChildFlowsCreated - return success with child flow
						setJSONContentType(w)
						w.WriteHeader(http.StatusOK)
						_ = json.NewEncoder(w).Encode([]FlowExecution{
							{
								AuthenticationFlow: true,
								DisplayName:        "child-flow",
								Level:              0,
							},
						})
					} else {
						// Second call for adjustChildFlowsPriority - return error
						w.WriteHeader(http.StatusInternalServerError)
					}
				},
			},
			http.MethodPost: {
				e.pathBuilder.AuthFlows(): func(w http.ResponseWriter, r *http.Request) {
					w.Header().Set("Location", "id/new-flow-id")
					w.WriteHeader(http.StatusOK)
				},
			},
		},
	})

	err := e.adapter.SyncAuthFlow(e.realmName, &flow)
	assert.Error(e.T(), err)
	assert.Contains(e.T(), err.Error(), "unable to adjust child flow priority")
}

// Error scenario tests for adjustChildFlowsPriority
func (e *ExecFlowTestSuite) TestAdjustChildFlowsPriority_ErrorGettingFlowExecutions() {
	flow := KeycloakAuthFlow{
		Alias: "test-flow",
		AuthenticationExecutions: []AuthenticationExecution{
			{
				AutheticatorFlow: true,
				Alias:            "child-flow",
				Priority:         1,
				Requirement:      "REQUIRED",
			},
		},
	}

	e.setupServerWithConfig(&ServerConfig{
		Handlers: map[string]map[string]ServerHandler{
			http.MethodGet: {
				e.pathBuilder.AuthFlowExecution(flow.Alias): func(w http.ResponseWriter, r *http.Request) {
					w.WriteHeader(http.StatusInternalServerError)
				},
			},
		},
	})

	err := e.adapter.adjustChildFlowsPriority(e.realmName, &flow)
	assert.Error(e.T(), err)
	assert.Contains(e.T(), err.Error(), "unable to get flow executions")
}

// Helper function to reduce code duplication for child flow priority tests
func (e *ExecFlowTestSuite) setupChildFlowPriorityTest(
	childAlias string, childPriority int, serverDisplayName string, serverRequirement string,
) KeycloakAuthFlow {
	flow := KeycloakAuthFlow{
		Alias: "test-flow",
		AuthenticationExecutions: []AuthenticationExecution{
			{
				AutheticatorFlow: true,
				Alias:            childAlias,
				Priority:         childPriority,
				Requirement:      "REQUIRED",
			},
		},
	}

	e.setupServerWithConfig(&ServerConfig{
		Handlers: map[string]map[string]ServerHandler{
			http.MethodGet: {
				e.pathBuilder.AuthFlowExecution(flow.Alias): func(w http.ResponseWriter, r *http.Request) {
					setJSONContentType(w)
					w.WriteHeader(http.StatusOK)
					_ = json.NewEncoder(w).Encode([]FlowExecution{
						{
							AuthenticationFlow: true,
							DisplayName:        serverDisplayName,
							Index:              0,
							ID:                 "flow-exec-id",
							Requirement:        serverRequirement,
							Level:              0,
						},
					})
				},
			},
		},
	})

	return flow
}

func (e *ExecFlowTestSuite) TestAdjustChildFlowsPriority_ErrorChildFlowNotFound() {
	flow := e.setupChildFlowPriorityTest("child-flow-1", 1, "different-child-flow", "ALTERNATIVE")

	err := e.adapter.adjustChildFlowsPriority(e.realmName, &flow)
	assert.Error(e.T(), err)
	assert.Contains(e.T(), err.Error(), "unable to find child flow with name")
}

func (e *ExecFlowTestSuite) TestAdjustChildFlowsPriority_ErrorWrongFlowPriority() {
	// Invalid priority, matching names and requirements
	flow := e.setupChildFlowPriorityTest("child-flow", 10, "child-flow", "REQUIRED")

	err := e.adapter.adjustChildFlowsPriority(e.realmName, &flow)
	assert.Error(e.T(), err)
	assert.Contains(e.T(), err.Error(), "wrong flow priority")
}

func (e *ExecFlowTestSuite) TestAdjustChildFlowsPriority_ErrorUpdatingFlowExecution() {
	flow := KeycloakAuthFlow{
		Alias: "test-flow",
		AuthenticationExecutions: []AuthenticationExecution{
			{
				AutheticatorFlow: true,
				Alias:            "child-flow",
				Priority:         0,
				Requirement:      "REQUIRED", // Different from what server returns
			},
		},
	}

	aliasPath := e.pathBuilder.AuthFlowExecution(flow.Alias)

	e.setupServerWithConfig(&ServerConfig{
		Handlers: map[string]map[string]ServerHandler{
			http.MethodGet: {
				aliasPath: func(w http.ResponseWriter, r *http.Request) {
					setJSONContentType(w)
					w.WriteHeader(http.StatusOK)
					_ = json.NewEncoder(w).Encode([]FlowExecution{
						{
							AuthenticationFlow: true,
							DisplayName:        "child-flow",
							Index:              0,
							ID:                 "flow-exec-id",
							Requirement:        "ALTERNATIVE", // Different from flow requirement
							Level:              0,
						},
					})
				},
			},
			http.MethodPut: {
				aliasPath: func(w http.ResponseWriter, r *http.Request) {
					w.WriteHeader(http.StatusInternalServerError)
				},
			},
		},
	})

	err := e.adapter.adjustChildFlowsPriority(e.realmName, &flow)
	assert.Error(e.T(), err)
	assert.Contains(e.T(), err.Error(), "unable to update flow execution")
}

func (e *ExecFlowTestSuite) TestAdjustChildFlowsPriority_ErrorAdjustingExecutionPriority() {
	flow := KeycloakAuthFlow{
		Alias: "test-flow",
		AuthenticationExecutions: []AuthenticationExecution{
			{
				AutheticatorFlow: true,
				Alias:            "child-flow",
				Priority:         1, // Different from Index to trigger priority adjustment
				Requirement:      "REQUIRED",
			},
		},
	}

	flowExecID := "flow-exec-id"

	e.setupServerWithConfig(&ServerConfig{
		Handlers: map[string]map[string]ServerHandler{
			http.MethodGet: {
				e.pathBuilder.AuthFlowExecution(flow.Alias): func(w http.ResponseWriter, r *http.Request) {
					setJSONContentType(w)
					w.WriteHeader(http.StatusOK)
					_ = json.NewEncoder(w).Encode([]FlowExecution{
						{
							AuthenticationFlow: true,
							DisplayName:        "child-flow",
							Index:              0, // Different from Priority to trigger adjustment
							ID:                 flowExecID,
							Requirement:        "REQUIRED",
							Level:              0,
						},
					})
				},
			},
			http.MethodPost: {
				e.pathBuilder.LowerExecutionPriority(flowExecID): func(w http.ResponseWriter, r *http.Request) {
					w.WriteHeader(http.StatusInternalServerError)
				},
			},
		},
	})

	err := e.adapter.adjustChildFlowsPriority(e.realmName, &flow)
	assert.Error(e.T(), err)
	assert.Contains(e.T(), err.Error(), "unable to adjust flow priority")
}

// Error scenario tests for clearFlowExecutions
func (e *ExecFlowTestSuite) TestClearFlowExecutions_ErrorGettingFlowExecutions() {
	e.setupServerWithConfig(&ServerConfig{
		Handlers: map[string]map[string]ServerHandler{
			http.MethodGet: {
				e.pathBuilder.AuthFlowExecution("test-flow"): func(w http.ResponseWriter, r *http.Request) {
					w.WriteHeader(http.StatusInternalServerError)
				},
			},
		},
	})

	err := e.adapter.clearFlowExecutions(e.realmName, "test-flow")
	assert.Error(e.T(), err)
	assert.Contains(e.T(), err.Error(), "unable to get flow executions")
}

func (e *ExecFlowTestSuite) TestClearFlowExecutions_ErrorDeletingFlowExecution() {
	execID := "exec-id-1"

	e.setupServerWithConfig(&ServerConfig{
		Handlers: map[string]map[string]ServerHandler{
			http.MethodGet: {
				e.pathBuilder.AuthFlowExecution("test-flow"): func(w http.ResponseWriter, r *http.Request) {
					setJSONContentType(w)
					w.WriteHeader(http.StatusOK)
					_ = json.NewEncoder(w).Encode([]FlowExecution{
						{
							AuthenticationFlow:   false,
							ID:                   execID,
							Level:                0,
							AuthenticationConfig: "", // No config to delete
						},
					})
				},
			},
			http.MethodDelete: {
				e.pathBuilder.AuthFlowExecutionDelete(execID): func(w http.ResponseWriter, r *http.Request) {
					w.WriteHeader(http.StatusInternalServerError)
				},
			},
		},
	})

	err := e.adapter.clearFlowExecutions(e.realmName, "test-flow")
	assert.Error(e.T(), err)
	assert.Contains(e.T(), err.Error(), "unable to delete flow execution")
}

func (e *ExecFlowTestSuite) TestClearFlowExecutions_ErrorDeletingFlowExecutionConfig() {
	execID := "exec-id-1"
	configID := "config-id-1"

	e.setupServerWithConfig(&ServerConfig{
		Handlers: map[string]map[string]ServerHandler{
			http.MethodGet: {
				e.pathBuilder.AuthFlowExecution("test-flow"): func(w http.ResponseWriter, r *http.Request) {
					setJSONContentType(w)
					w.WriteHeader(http.StatusOK)
					_ = json.NewEncoder(w).Encode([]FlowExecution{
						{
							AuthenticationFlow:   false,
							ID:                   execID,
							Level:                0,
							AuthenticationConfig: configID, // Has config to delete
						},
					})
				},
			},
			http.MethodDelete: {
				e.pathBuilder.AuthFlowExecutionDelete(execID): func(w http.ResponseWriter, r *http.Request) {
					w.WriteHeader(http.StatusOK)
				},
				e.pathBuilder.AuthFlowConfig(configID): func(w http.ResponseWriter, r *http.Request) {
					w.WriteHeader(http.StatusInternalServerError)
				},
			},
		},
	})

	err := e.adapter.clearFlowExecutions(e.realmName, "test-flow")
	assert.Error(e.T(), err)
	assert.Contains(e.T(), err.Error(), "unable to delete flow execution config")
}

// Error scenario tests for validateChildFlowsCreated
func (e *ExecFlowTestSuite) TestValidateChildFlowsCreated_ErrorGettingFlowExecutions() {
	flow := &KeycloakAuthFlow{
		Alias: "test-flow",
		AuthenticationExecutions: []AuthenticationExecution{
			{
				AutheticatorFlow: true,
				Alias:            "child-flow",
			},
		},
	}

	e.setupServerWithConfig(&ServerConfig{
		Handlers: map[string]map[string]ServerHandler{
			http.MethodGet: {
				e.pathBuilder.AuthFlowExecution(flow.Alias): func(w http.ResponseWriter, r *http.Request) {
					w.WriteHeader(http.StatusInternalServerError)
				},
			},
		},
	})

	err := e.adapter.validateChildFlowsCreated(e.realmName, flow)
	assert.Error(e.T(), err)
	assert.Contains(e.T(), err.Error(), "unable to get flow executions")
}

// Error scenario tests for syncBaseAuthFlow
func (e *ExecFlowTestSuite) TestSyncBaseAuthFlow_ErrorGettingAuthFlowID() {
	flow := &KeycloakAuthFlow{
		Alias: "test-flow",
	}

	e.setupServerWithConfig(&ServerConfig{
		Handlers: map[string]map[string]ServerHandler{
			http.MethodGet: {
				e.pathBuilder.AuthFlows(): func(w http.ResponseWriter, r *http.Request) {
					w.WriteHeader(http.StatusInternalServerError)
				},
			},
		},
	})

	_, err := e.adapter.syncBaseAuthFlow(e.realmName, flow)
	assert.Error(e.T(), err)
	assert.Contains(e.T(), err.Error(), "unable to get auth flow")
}

func (e *ExecFlowTestSuite) TestSyncBaseAuthFlow_ErrorCreatingAuthFlow() {
	flow := &KeycloakAuthFlow{
		Alias: "test-flow",
	}

	e.setupServerWithConfig(&ServerConfig{
		Handlers: map[string]map[string]ServerHandler{
			http.MethodGet: {
				e.pathBuilder.AuthFlows(): func(w http.ResponseWriter, r *http.Request) {
					setJSONContentType(w)
					w.WriteHeader(http.StatusOK)
					_ = json.NewEncoder(w).Encode([]KeycloakAuthFlow{}) // Empty list = not found
				},
			},
			http.MethodPost: {
				e.pathBuilder.AuthFlows(): func(w http.ResponseWriter, r *http.Request) {
					w.WriteHeader(http.StatusInternalServerError)
				},
			},
		},
	})

	_, err := e.adapter.syncBaseAuthFlow(e.realmName, flow)
	assert.Error(e.T(), err)
	assert.Contains(e.T(), err.Error(), "unable to create auth flow")
}

func (e *ExecFlowTestSuite) TestSyncBaseAuthFlow_ErrorClearingFlowExecutions() {
	flow := &KeycloakAuthFlow{
		Alias: "test-flow",
		ID:    "existing-flow-id",
	}

	e.setupServerWithConfig(&ServerConfig{
		Handlers: map[string]map[string]ServerHandler{
			http.MethodGet: {
				e.pathBuilder.AuthFlows(): func(w http.ResponseWriter, r *http.Request) {
					setJSONContentType(w)
					w.WriteHeader(http.StatusOK)
					_ = json.NewEncoder(w).Encode([]KeycloakAuthFlow{
						{Alias: flow.Alias, ID: flow.ID}, // Flow exists
					})
				},
				e.pathBuilder.AuthFlowExecution(flow.Alias): func(w http.ResponseWriter, r *http.Request) {
					w.WriteHeader(http.StatusInternalServerError) // This will cause clearFlowExecutions to fail
				},
			},
		},
	})

	_, err := e.adapter.syncBaseAuthFlow(e.realmName, flow)
	assert.Error(e.T(), err)
	assert.Contains(e.T(), err.Error(), "unable to clear flow executions")
}

func (e *ExecFlowTestSuite) TestSyncBaseAuthFlow_ErrorGettingFlowExecution() {
	flow := &KeycloakAuthFlow{
		Alias:            "test-flow",
		ParentName:       "parent",
		ChildRequirement: "REQUIRED",
		ID:               "existing-flow-id",
	}

	e.setupServerWithConfig(&ServerConfig{
		Handlers: map[string]map[string]ServerHandler{
			http.MethodGet: {
				e.pathBuilder.AuthFlows(): func(w http.ResponseWriter, r *http.Request) {
					setJSONContentType(w)
					w.WriteHeader(http.StatusOK)
					_ = json.NewEncoder(w).Encode([]KeycloakAuthFlow{
						{Alias: flow.Alias, ID: flow.ID}, // Flow exists
					})
				},
				e.pathBuilder.AuthFlowExecution(flow.Alias): func(w http.ResponseWriter, r *http.Request) {
					setJSONContentType(w)
					w.WriteHeader(http.StatusOK)
					_ = json.NewEncoder(w).Encode([]FlowExecution{}) // Empty for clearFlowExecutions
				},
				e.pathBuilder.AuthFlowExecution(flow.ParentName): func(w http.ResponseWriter, r *http.Request) {
					w.WriteHeader(http.StatusInternalServerError) // This will cause getFlowExecution to fail
				},
			},
		},
	})

	_, err := e.adapter.syncBaseAuthFlow(e.realmName, flow)
	assert.Error(e.T(), err)
	assert.Contains(e.T(), err.Error(), "unable to get auth flow executions")
}

func (e *ExecFlowTestSuite) TestSyncBaseAuthFlow_ErrorUpdatingFlowExecution() {
	flow := &KeycloakAuthFlow{
		Alias:            "test-flow",
		ParentName:       "parent",
		ChildRequirement: "REQUIRED",
		ID:               "existing-flow-id",
	}

	parentPath := e.pathBuilder.AuthFlowExecution(flow.ParentName)

	e.setupServerWithConfig(&ServerConfig{
		Handlers: map[string]map[string]ServerHandler{
			http.MethodGet: {
				e.pathBuilder.AuthFlows(): func(w http.ResponseWriter, r *http.Request) {
					setJSONContentType(w)
					w.WriteHeader(http.StatusOK)
					_ = json.NewEncoder(w).Encode([]KeycloakAuthFlow{
						{Alias: flow.Alias, ID: flow.ID}, // Flow exists
					})
				},
				e.pathBuilder.AuthFlowExecution(flow.Alias): func(w http.ResponseWriter, r *http.Request) {
					setJSONContentType(w)
					w.WriteHeader(http.StatusOK)
					_ = json.NewEncoder(w).Encode([]FlowExecution{}) // Empty for clearFlowExecutions and validateChildFlowsCreated
				},
				parentPath: func(w http.ResponseWriter, r *http.Request) {
					setJSONContentType(w)
					w.WriteHeader(http.StatusOK)
					_ = json.NewEncoder(w).Encode([]FlowExecution{
						{DisplayName: flow.Alias, ID: "exec-id"},
					})
				},
			},
			http.MethodPut: {
				parentPath: func(w http.ResponseWriter, r *http.Request) {
					w.WriteHeader(http.StatusInternalServerError)
				},
			},
		},
	})

	_, err := e.adapter.syncBaseAuthFlow(e.realmName, flow)
	assert.Error(e.T(), err)
	assert.Contains(e.T(), err.Error(), "unable to update flow execution requirement")
}

func (e *ExecFlowTestSuite) TestSyncBaseAuthFlow_ErrorValidatingChildFlowsCreated() {
	flow := &KeycloakAuthFlow{
		Alias: "test-flow",
		AuthenticationExecutions: []AuthenticationExecution{
			{
				AutheticatorFlow: true,
				Alias:            "child-flow",
			},
		},
		ID: "existing-flow-id",
	}

	e.setupServerWithConfig(&ServerConfig{
		Handlers: map[string]map[string]ServerHandler{
			http.MethodGet: {
				e.pathBuilder.AuthFlows(): func(w http.ResponseWriter, r *http.Request) {
					setJSONContentType(w)
					w.WriteHeader(http.StatusOK)
					_ = json.NewEncoder(w).Encode([]KeycloakAuthFlow{
						{Alias: flow.Alias, ID: flow.ID}, // Flow exists
					})
				},
				e.pathBuilder.AuthFlowExecution(flow.Alias): func(w http.ResponseWriter, r *http.Request) {
					setJSONContentType(w)
					w.WriteHeader(http.StatusOK)
					_ = json.NewEncoder(w).Encode([]FlowExecution{}) // Empty executions for both clear and validate
				},
			},
		},
	})

	_, err := e.adapter.syncBaseAuthFlow(e.realmName, flow)
	assert.Error(e.T(), err)
	assert.Contains(e.T(), err.Error(), "child flows validation failed")
}

// Error scenario tests for additional methods not covered above
func (e *ExecFlowTestSuite) TestGetRealmAuthFlows_Error() {
	e.setupServerWithConfig(&ServerConfig{
		Handlers: map[string]map[string]ServerHandler{
			http.MethodGet: {
				e.pathBuilder.AuthFlows(): func(w http.ResponseWriter, r *http.Request) {
					w.WriteHeader(http.StatusInternalServerError)
				},
			},
		},
	})

	_, err := e.adapter.GetRealmAuthFlows(e.realmName)
	assert.Error(e.T(), err)
	assert.Contains(e.T(), err.Error(), "unable to list auth flow by realm")
}

func (e *ExecFlowTestSuite) TestCreateAuthFlow_ErrorCreatingParentFlow() {
	flow := &KeycloakAuthFlow{
		Alias: "test-flow",
	}

	e.setupServerWithConfig(&ServerConfig{
		Handlers: map[string]map[string]ServerHandler{
			http.MethodPost: {
				e.pathBuilder.AuthFlows(): func(w http.ResponseWriter, r *http.Request) {
					w.WriteHeader(http.StatusInternalServerError)
				},
			},
		},
	})

	_, err := e.adapter.createAuthFlow(e.realmName, flow)
	assert.Error(e.T(), err)
	assert.Contains(e.T(), err.Error(), "unable to create auth flow in realm")
}

func (e *ExecFlowTestSuite) TestCreateAuthFlow_ErrorCreatingChildFlow() {
	flow := &KeycloakAuthFlow{
		Alias:      "child-flow",
		ParentName: "parent-flow",
	}

	e.setupServerWithConfig(&ServerConfig{
		Handlers: map[string]map[string]ServerHandler{
			http.MethodPost: {
				e.pathBuilder.RealmAuthFlowParentExecutions("parent-flow"): func(w http.ResponseWriter, r *http.Request) {
					w.WriteHeader(http.StatusInternalServerError)
				},
			},
		},
	})

	_, err := e.adapter.createAuthFlow(e.realmName, flow)
	assert.Error(e.T(), err)
	assert.Contains(e.T(), err.Error(), "unable to create child auth flow in realm")
}

func (e *ExecFlowTestSuite) TestCreateAuthFlow_ErrorGettingIDFromLocation() {
	flow := &KeycloakAuthFlow{
		Alias: "test-flow",
	}

	e.setupServerWithConfig(&ServerConfig{
		Handlers: map[string]map[string]ServerHandler{
			http.MethodPost: {
				e.pathBuilder.AuthFlows(): func(w http.ResponseWriter, r *http.Request) {
					// Don't set Location header
					w.WriteHeader(http.StatusOK)
				},
			},
		},
	})

	_, err := e.adapter.createAuthFlow(e.realmName, flow)
	assert.Error(e.T(), err)
	assert.Contains(e.T(), err.Error(), "unable to get flow id")
}

func (e *ExecFlowTestSuite) TestUpdateFlowExecution_Error() {
	flow := &FlowExecution{
		ID: "exec-id",
	}

	parentPath := e.pathBuilder.AuthFlowExecution("parent-flow")

	e.setupServerWithConfig(&ServerConfig{
		Handlers: map[string]map[string]ServerHandler{
			http.MethodPut: {
				parentPath: func(w http.ResponseWriter, r *http.Request) {
					w.WriteHeader(http.StatusInternalServerError)
				},
			},
		},
	})

	err := e.adapter.updateFlowExecution(e.realmName, "parent-flow", flow)
	assert.Error(e.T(), err)
	assert.Contains(e.T(), err.Error(), "unable to update flow execution")
}

func (e *ExecFlowTestSuite) TestAdjustExecutionPriority_Error() {
	execID := "exec-id"

	e.setupServerWithConfig(&ServerConfig{
		Handlers: map[string]map[string]ServerHandler{
			http.MethodPost: {
				e.pathBuilder.LowerExecutionPriority(execID): func(w http.ResponseWriter, r *http.Request) {
					w.WriteHeader(http.StatusInternalServerError)
				},
			},
		},
	})

	err := e.adapter.adjustExecutionPriority(e.realmName, execID, -1)
	assert.Error(e.T(), err)
	assert.Contains(e.T(), err.Error(), "unable to adjust execution priority")
}

func (e *ExecFlowTestSuite) TestGetFlowExecutions_Error() {
	e.setupServerWithConfig(&ServerConfig{
		Handlers: map[string]map[string]ServerHandler{
			http.MethodGet: {
				e.pathBuilder.AuthFlowExecution("test-flow"): func(w http.ResponseWriter, r *http.Request) {
					w.WriteHeader(http.StatusInternalServerError)
				},
			},
		},
	})

	_, err := e.adapter.getFlowExecutions(e.realmName, "test-flow")
	assert.Error(e.T(), err)
	assert.Contains(e.T(), err.Error(), "unable get flow executions")
}

func (e *ExecFlowTestSuite) TestDeleteAuthFlowInternal_Error() {
	flowID := "flow-id"

	e.setupServerWithConfig(&ServerConfig{
		Handlers: map[string]map[string]ServerHandler{
			http.MethodDelete: {
				e.pathBuilder.AuthFlow(flowID): func(w http.ResponseWriter, r *http.Request) {
					w.WriteHeader(http.StatusInternalServerError)
				},
			},
		},
	})

	err := e.adapter.deleteAuthFlow(e.realmName, flowID)
	assert.Error(e.T(), err)
	assert.Contains(e.T(), err.Error(), "unable to delete auth flow")
}

func (e *ExecFlowTestSuite) TestAddAuthFlowExecution_Error() {
	flowExec := &AuthenticationExecution{
		Authenticator: "basic-auth",
		ParentFlow:    "parent-flow",
	}

	e.setupServerWithConfig(&ServerConfig{
		Handlers: map[string]map[string]ServerHandler{
			http.MethodPost: {
				e.pathBuilder.AuthFlowExecutionCreate(): func(w http.ResponseWriter, r *http.Request) {
					w.WriteHeader(http.StatusInternalServerError)
				},
			},
		},
	})

	err := e.adapter.addAuthFlowExecution(e.realmName, flowExec)
	assert.Error(e.T(), err)
	assert.Contains(e.T(), err.Error(), "unable to add auth flow execution")
}

func (e *ExecFlowTestSuite) TestAddAuthFlowExecution_ErrorGettingIDFromLocation() {
	flowExec := &AuthenticationExecution{
		Authenticator: "basic-auth",
		ParentFlow:    "parent-flow",
	}

	e.setupServerWithConfig(&ServerConfig{
		Handlers: map[string]map[string]ServerHandler{
			http.MethodPost: {
				e.pathBuilder.AuthFlowExecutionCreate(): func(w http.ResponseWriter, r *http.Request) {
					// Don't set Location header
					w.WriteHeader(http.StatusOK)
				},
			},
		},
	})

	err := e.adapter.addAuthFlowExecution(e.realmName, flowExec)
	assert.Error(e.T(), err)
	assert.Contains(e.T(), err.Error(), "unable to get auth exec id")
}

func (e *ExecFlowTestSuite) TestAddAuthFlowExecution_ErrorCreatingConfig() {
	flowExec := &AuthenticationExecution{
		Authenticator: "basic-auth",
		ParentFlow:    "parent-flow",
		AuthenticatorConfig: &AuthenticatorConfig{
			Alias:  "test-config",
			Config: map[string]string{"key": "value"},
		},
	}

	execID := "new-exec-id"

	e.setupServerWithConfig(&ServerConfig{
		Handlers: map[string]map[string]ServerHandler{
			http.MethodPost: {
				e.pathBuilder.AuthFlowExecutionCreate(): func(w http.ResponseWriter, r *http.Request) {
					w.Header().Set("Location", fmt.Sprintf("id/%s", execID))
					w.WriteHeader(http.StatusOK)
				},
				e.pathBuilder.AuthFlowExecutionConfig(execID): func(w http.ResponseWriter, r *http.Request) {
					w.WriteHeader(http.StatusInternalServerError)
				},
			},
		},
	})

	err := e.adapter.addAuthFlowExecution(e.realmName, flowExec)
	assert.Error(e.T(), err)
	assert.Contains(e.T(), err.Error(), "unable to create auth flow execution config")
}

func (e *ExecFlowTestSuite) TestCreateAuthFlowExecutionConfig_Error() {
	flowExec := &AuthenticationExecution{
		ID: "exec-id",
		AuthenticatorConfig: &AuthenticatorConfig{
			Alias:  "test-config",
			Config: map[string]string{"key": "value"},
		},
	}

	e.setupServerWithConfig(&ServerConfig{
		Handlers: map[string]map[string]ServerHandler{
			http.MethodPost: {
				e.pathBuilder.AuthFlowExecutionConfig(flowExec.ID): func(w http.ResponseWriter, r *http.Request) {
					w.WriteHeader(http.StatusInternalServerError)
				},
			},
		},
	})

	err := e.adapter.createAuthFlowExecutionConfig(e.realmName, flowExec)
	assert.Error(e.T(), err)
	assert.Contains(e.T(), err.Error(), "unable to add auth flow execution")
}

func (e *ExecFlowTestSuite) TestUnsetBrowserFlow_ErrorGettingRealm() {
	e.goCloakMockClient.On("GetRealm", mock.Anything, "token", e.realmName).
		Return(nil, errors.New("realm error"))

	err := e.adapter.unsetBrowserFlow(e.realmName, "test-flow")
	assert.Error(e.T(), err)
	assert.Contains(e.T(), err.Error(), "unable to get realm")
}

func (e *ExecFlowTestSuite) TestUnsetBrowserFlow_ErrorGettingAuthFlows() {
	e.goCloakMockClient.On("GetRealm", mock.Anything, "token", e.realmName).
		Return(&gocloak.RealmRepresentation{
			BrowserFlow: gocloak.StringP("test-flow"),
		}, nil)

	e.setupServerWithConfig(&ServerConfig{
		Handlers: map[string]map[string]ServerHandler{
			http.MethodGet: {
				e.pathBuilder.AuthFlows(): func(w http.ResponseWriter, r *http.Request) {
					w.WriteHeader(http.StatusInternalServerError)
				},
			},
		},
	})

	err := e.adapter.unsetBrowserFlow(e.realmName, "test-flow")
	assert.Error(e.T(), err)
	assert.Contains(e.T(), err.Error(), "unable to get auth flows for realm")
}

func (e *ExecFlowTestSuite) TestUnsetBrowserFlow_ErrorNoReplacementFlow() {
	e.goCloakMockClient.On("GetRealm", mock.Anything, "token", e.realmName).
		Return(&gocloak.RealmRepresentation{
			BrowserFlow: gocloak.StringP("test-flow"),
		}, nil)

	e.setupServerWithConfig(&ServerConfig{
		Handlers: map[string]map[string]ServerHandler{
			http.MethodGet: {
				e.pathBuilder.AuthFlows(): func(w http.ResponseWriter, r *http.Request) {
					setJSONContentType(w)
					w.WriteHeader(http.StatusOK)
					// Only return the current flow, no replacement available
					_ = json.NewEncoder(w).Encode([]KeycloakAuthFlow{
						{Alias: "test-flow", ID: "flow-id"},
					})
				},
			},
		},
	})

	err := e.adapter.unsetBrowserFlow(e.realmName, "test-flow")
	assert.Error(e.T(), err)
	assert.Contains(e.T(), err.Error(), "no replacement for browser flow found")
}

func (e *ExecFlowTestSuite) TestUnsetBrowserFlow_ErrorUpdatingRealm() {
	e.goCloakMockClient.On("GetRealm", mock.Anything, "token", e.realmName).
		Return(&gocloak.RealmRepresentation{
			BrowserFlow: gocloak.StringP("test-flow"),
		}, nil)

	e.setupServerWithConfig(&ServerConfig{
		Handlers: map[string]map[string]ServerHandler{
			http.MethodGet: {
				e.pathBuilder.AuthFlows(): func(w http.ResponseWriter, r *http.Request) {
					setJSONContentType(w)
					w.WriteHeader(http.StatusOK)
					_ = json.NewEncoder(w).Encode([]KeycloakAuthFlow{
						{Alias: "test-flow", ID: "flow-id"},
						{Alias: "replacement-flow", ID: "replacement-id"},
					})
				},
			},
		},
	})

	e.goCloakMockClient.On("UpdateRealm", mock.Anything, "token", mock.Anything).
		Return(errors.New("update error"))

	err := e.adapter.unsetBrowserFlow(e.realmName, "test-flow")
	assert.Error(e.T(), err)
	assert.Contains(e.T(), err.Error(), "unable to update realm")
}

func (e *ExecFlowTestSuite) TestDeleteAuthFlowConfig_Error() {
	configID := "config-id"

	e.setupServerWithConfig(&ServerConfig{
		Handlers: map[string]map[string]ServerHandler{
			http.MethodDelete: {
				e.pathBuilder.AuthFlowConfig(configID): func(w http.ResponseWriter, r *http.Request) {
					w.WriteHeader(http.StatusInternalServerError)
				},
			},
		},
	})

	err := e.adapter.deleteAuthFlowConfig(e.realmName, configID)
	assert.Error(e.T(), err)
	assert.Contains(e.T(), err.Error(), "unable to delete auth flow config")
}

func TestExecFlowTestSuite(t *testing.T) {
	suite.Run(t, new(ExecFlowTestSuite))
}

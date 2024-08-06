package adapter

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"path"
	"strings"
	"testing"

	"github.com/Nerzal/gocloak/v12"
	"github.com/go-resty/resty/v2"
	"github.com/jarcoal/httpmock"
	"github.com/pkg/errors"
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
}

func (e *ExecFlowTestSuite) SetupTest() {
	e.restyClient = resty.New()
	httpmock.ActivateNonDefault(e.restyClient.GetClient())

	e.goCloakMockClient = mocks.NewMockGoCloak(e.T())
	e.goCloakMockClient.On("RestyClient").Return(e.restyClient).Maybe()

	e.adapter = &GoCloakAdapter{
		client: e.goCloakMockClient,
		token:  &gocloak.JWT{AccessToken: "token"},
	}
	e.realmName = "realm123"
}

func (e *ExecFlowTestSuite) TestCreateAuthFlowParent() {
	var (
		parentName = "parent-name"
		newFlowID  = "new-flow-id"
	)

	createFlowResponse := httpmock.NewStringResponse(200, "")
	defer closeWithFailOnError(e.T(), createFlowResponse.Body)
	createFlowResponse.Header.Set("Location", fmt.Sprintf("id/%s", newFlowID))

	httpmock.RegisterResponder("POST",
		strings.ReplaceAll(path.Join(authFlows, parentName, "executions/flow"), "{realm}", e.realmName),
		httpmock.ResponderFromResponse(createFlowResponse))

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

	httpmock.RegisterResponder("GET", strings.ReplaceAll(authFlows, "{realm}", e.realmName),
		httpmock.NewJsonResponderOrPanic(200, []KeycloakAuthFlow{{Alias: flow.Alias, ID: existFlowID},
			{Alias: "some-another-flow", ID: "321"}}))

	deleteURL := strings.ReplaceAll(authFlow, "{realm}", e.realmName)
	deleteURL = strings.ReplaceAll(deleteURL, "{id}", existFlowID)

	httpmock.RegisterResponder("DELETE", deleteURL, httpmock.NewStringResponder(200, ""))

	createFlowResponse := httpmock.NewStringResponse(200, "")
	defer closeWithFailOnError(e.T(), createFlowResponse.Body)
	createFlowResponse.Header.Set("Location", "id/new-flow-id")

	httpmock.RegisterResponder("POST", strings.ReplaceAll(authFlows, "{realm}", e.realmName),
		httpmock.ResponderFromResponse(createFlowResponse))

	createExecResponse := httpmock.NewStringResponse(200, "")

	defer closeWithFailOnError(e.T(), createFlowResponse.Body)

	newExecID := "new-exec-id"
	createExecResponse.Header.Set("Location", fmt.Sprintf("id/%s", newExecID))
	httpmock.RegisterResponder("POST", strings.ReplaceAll(authFlowExecutionCreate, "{realm}", e.realmName),
		httpmock.ResponderFromResponse(createExecResponse))

	createConfigURL := strings.ReplaceAll(authFlowExecutionConfig, "{realm}", e.realmName)
	createConfigURL = strings.ReplaceAll(createConfigURL, "{id}", newExecID)

	httpmock.RegisterResponder("POST", createConfigURL, httpmock.NewStringResponder(200, ""))

	childExecutionID := "child-exec-id1"

	httpmock.RegisterResponder("GET",
		fmt.Sprintf("/admin/realms/%s/authentication/flows/%s/executions", e.realmName, flow.Alias),
		httpmock.NewJsonResponderOrPanic(200, []FlowExecution{{ID: childExecutionID, AuthenticationConfig: "authConf1"}}))
	httpmock.RegisterResponder("DELETE",
		fmt.Sprintf("/admin/realms/%s/authentication/executions/%s", e.realmName, childExecutionID),
		httpmock.NewStringResponder(200, ""))
	httpmock.RegisterResponder("DELETE",
		strings.ReplaceAll(
			strings.ReplaceAll(authFlowConfig, "{realm}", e.realmName),
			"{id}",
			"authConf1"),
		httpmock.NewStringResponder(200, ""),
	)

	err := e.adapter.SyncAuthFlow(e.realmName, &flow)
	assert.NoError(e.T(), err)
}

func (e *ExecFlowTestSuite) TestDeleteAuthFlowWithParent() {
	flow := KeycloakAuthFlow{Alias: "al", ParentName: "par"}

	httpmock.RegisterResponder(http.MethodGet, "/admin/realms/realm123/authentication/flows/par/executions",
		httpmock.NewJsonResponderOrPanic(http.StatusOK, []FlowExecution{
			{DisplayName: flow.Alias, ID: "id12"},
		}))
	httpmock.RegisterResponder(http.MethodDelete, "/admin/realms/realm123/authentication/executions/id12",
		httpmock.NewStringResponder(http.StatusOK, ""))

	err := e.adapter.DeleteAuthFlow(e.realmName, &flow)
	assert.NoError(e.T(), err)
}

func (e *ExecFlowTestSuite) TestDeleteAuthFlowWithParentUnableGetFlow() {
	flow := KeycloakAuthFlow{Alias: "al", ParentName: "par"}

	httpmock.RegisterResponder(http.MethodGet, "/admin/realms/realm123/authentication/flows/par/executions",
		httpmock.NewJsonResponderOrPanic(http.StatusNotFound, nil))

	err := e.adapter.DeleteAuthFlow(e.realmName, &flow)
	assert.Error(e.T(), err)
}

func (e *ExecFlowTestSuite) TestDeleteAuthFlowWithParentUnableDelete() {
	flow := KeycloakAuthFlow{Alias: "al", ParentName: "par"}

	httpmock.RegisterResponder(http.MethodGet, "/admin/realms/realm123/authentication/flows/par/executions",
		httpmock.NewJsonResponderOrPanic(http.StatusOK, []FlowExecution{
			{DisplayName: flow.Alias, ID: "id12"},
		}))
	httpmock.RegisterResponder(http.MethodDelete, "/admin/realms/realm123/authentication/executions/id12",
		httpmock.NewStringResponder(http.StatusBadRequest, ""))

	err := e.adapter.DeleteAuthFlow(e.realmName, &flow)
	assert.Error(e.T(), err)
}

func (e *ExecFlowTestSuite) TestDeleteAuthFlow() {
	var (
		flowAlias           = "flow-alias"
		existFlowID         = "id321"
		newBrowserFlowAlias = "alias-br-1"
	)

	httpmock.RegisterResponder("GET", strings.ReplaceAll(authFlows, "{realm}", e.realmName),
		httpmock.NewJsonResponderOrPanic(200, []KeycloakAuthFlow{
			{Alias: flowAlias, ID: existFlowID},
			{
				Alias: newBrowserFlowAlias,
			},
		}))

	deleteURL := strings.ReplaceAll(authFlow, "{realm}", e.realmName)
	deleteURL = strings.ReplaceAll(deleteURL, "{id}", existFlowID)
	httpmock.RegisterResponder("DELETE", deleteURL, httpmock.NewStringResponder(200, ""))

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

	httpmock.RegisterResponder("GET",
		fmt.Sprintf("/admin/realms/%s/authentication/flows/%s/executions", e.realmName, flow.ParentName),
		httpmock.NewJsonResponderOrPanic(200, []FlowExecution{
			{
				DisplayName: flow.Alias,
				FlowID:      flowID,
			},
		}))

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
	assert.ErrorIs(e.T(), errors.Cause(err), mockErr)
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
	assert.ErrorIs(e.T(), errors.Cause(err), mockErr)
}

func (e *ExecFlowTestSuite) TestSyncBaseAuthFlow() {
	var (
		flow = KeycloakAuthFlow{
			Alias: "flow1",
			AuthenticationExecutions: []AuthenticationExecution{
				{
					AutheticatorFlow: true,
					Alias:            "child-flow",
				},
			},
		}
		flowID = "flow-id-1"
	)

	httpmock.RegisterResponder("GET",
		fmt.Sprintf("/admin/realms/%s/authentication/flows", e.realmName),
		httpmock.NewJsonResponderOrPanic(200, []KeycloakAuthFlow{}))

	createFlowResponse := httpmock.NewStringResponse(200, "")
	defer closeWithFailOnError(e.T(), createFlowResponse.Body)
	createFlowResponse.Header.Set("Location", fmt.Sprintf("id/%s", flowID))

	httpmock.RegisterResponder("POST",
		fmt.Sprintf("/admin/realms/%s/authentication/flows", e.realmName),
		httpmock.ResponderFromResponse(createFlowResponse))

	httpmock.RegisterResponder("GET",
		fmt.Sprintf("/admin/realms/%s/authentication/flows/%s/executions", e.realmName,
			flow.Alias),
		httpmock.NewJsonResponderOrPanic(200, []FlowExecution{{}}))

	_, err := e.adapter.syncBaseAuthFlow(e.realmName, &flow)

	assert.Error(e.T(), err)
	assert.EqualError(e.T(), err, "child flows validation failed: not all child flows created")
}

func (e *ExecFlowTestSuite) TestSyncBaseAuthFlowShouldUpdateChildFlowRequirement() {
	flow := KeycloakAuthFlow{
		Alias:            "flow1",
		ParentName:       "parent",
		ChildRequirement: "REQUIRED",
	}
	flowID := "flow-id-1"

	httpmock.RegisterResponder(
		http.MethodGet,
		fmt.Sprintf("/admin/realms/%s/authentication/flows/%s/executions", e.realmName, flow.ParentName),
		httpmock.NewJsonResponderOrPanic(
			http.StatusOK,
			[]FlowExecution{{
				DisplayName: flow.Alias,
				FlowID:      flowID,
			}},
		),
	)

	httpmock.RegisterResponder(
		http.MethodGet,
		fmt.Sprintf("/admin/realms/%s/authentication/flows/%s/executions", e.realmName, flow.Alias),
		httpmock.NewJsonResponderOrPanic(http.StatusOK, []FlowExecution{}),
	)

	httpmock.RegisterResponder(
		http.MethodPut,
		fmt.Sprintf("/admin/realms/%s/authentication/flows/%s/executions", e.realmName, flow.ParentName),
		httpmock.NewJsonResponderOrPanic(http.StatusOK, map[string]string{}),
	)

	_, err := e.adapter.syncBaseAuthFlow(e.realmName, &flow)

	assert.NoError(e.T(), err)
}

func (e *ExecFlowTestSuite) TestSyncBaseAuthFlowFailedUpdateChildFlowRequirement() {
	flow := KeycloakAuthFlow{
		Alias:            "flow1",
		ParentName:       "parent",
		ChildRequirement: "REQUIRED",
	}
	flowID := "flow-id-1"

	httpmock.RegisterResponder(
		http.MethodGet,
		fmt.Sprintf("/admin/realms/%s/authentication/flows/%s/executions", e.realmName, flow.ParentName),
		httpmock.NewJsonResponderOrPanic(
			http.StatusOK,
			[]FlowExecution{{
				DisplayName: flow.Alias,
				FlowID:      flowID,
			}},
		),
	)

	httpmock.RegisterResponder(
		http.MethodGet,
		fmt.Sprintf("/admin/realms/%s/authentication/flows/%s/executions", e.realmName, flow.Alias),
		httpmock.NewJsonResponderOrPanic(http.StatusOK, []FlowExecution{}),
	)

	httpmock.RegisterResponder(
		http.MethodPut,
		fmt.Sprintf("/admin/realms/%s/authentication/flows/%s/executions", e.realmName, flow.ParentName),
		httpmock.NewJsonResponderOrPanic(http.StatusBadRequest, map[string]string{}),
	)

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

	httpmock.RegisterResponder(
		http.MethodGet,
		fmt.Sprintf("/admin/realms/%s/authentication/flows/%s/executions", e.realmName, flow.ParentName),
		httpmock.NewJsonResponderOrPanic(
			http.StatusInternalServerError,
			[]FlowExecution{},
		),
	)

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

	httpmock.RegisterResponder(
		http.MethodGet,
		fmt.Sprintf("/admin/realms/%s/authentication/flows/%s/executions", e.realmName, flow.ParentName),
		httpmock.NewJsonResponderOrPanic(
			http.StatusOK,
			[]FlowExecution{},
		),
	)

	httpmock.RegisterResponder(
		http.MethodPost,
		fmt.Sprintf("/admin/realms/%s/authentication/flows/%s/executions/flow", e.realmName, flow.ParentName),
		httpmock.NewJsonResponderOrPanic(
			http.StatusInternalServerError,
			map[string]string{},
		),
	)

	_, err := e.adapter.syncBaseAuthFlow(e.realmName, &flow)

	require.Error(e.T(), err)
	assert.Contains(e.T(), err.Error(), "unable to create child auth flow in realm")
}

func (e *ExecFlowTestSuite) TestGetFlowExecutionID() {
	flow := KeycloakAuthFlow{ParentName: "parent", Alias: "fff"}
	_, err := e.adapter.getFlowExecutionID(e.realmName, &flow)

	assert.Error(e.T(), err)
	assert.Contains(e.T(), err.Error(), "no responder found")

	httpmock.RegisterResponder("GET", "/admin/realms/realm123/authentication/flows/parent/executions",
		httpmock.NewJsonResponderOrPanic(200, []FlowExecution{}))

	_, err = e.adapter.getFlowExecutionID(e.realmName, &flow)
	assert.Error(e.T(), err)
	assert.EqualError(e.T(), err, "auth flow not found")

	httpmock.RegisterResponder("GET", "/admin/realms/realm123/authentication/flows/parent/executions",
		httpmock.NewJsonResponderOrPanic(200, []FlowExecution{
			{
				DisplayName: flow.Alias,
				ID:          "as12",
			},
		}))

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

	httpmock.RegisterResponder("GET",
		fmt.Sprintf("/admin/realms/%s/authentication/flows/%s/executions", e.realmName, flow.Alias),
		httpmock.NewJsonResponderOrPanic(200, []FlowExecution{
			{
				AuthenticationFlow: true,
				DisplayName:        flow.AuthenticationExecutions[0].Alias,
				Index:              0,
				ID:                 flowExecutionID,
				Requirement:        "ALTERNATIVE",
			},
		}))

	httpmock.RegisterResponder("POST",
		fmt.Sprintf("/admin/realms/%s/authentication/executions/%s/lower-priority", e.realmName, flowExecutionID),
		httpmock.NewStringResponder(200, ""))

	httpmock.RegisterResponder("PUT",
		fmt.Sprintf("/admin/realms/%s/authentication/flows/%s/executions", e.realmName, flow.Alias),
		httpmock.NewStringResponder(200, ""))

	err := e.adapter.adjustChildFlowsPriority(e.realmName, &flow)
	assert.NoError(e.T(), err)
}

func TestExecFlowTestSuite(t *testing.T) {
	suite.Run(t, new(ExecFlowTestSuite))
}

func closeWithFailOnError(t *testing.T, closer io.Closer) {
	err := closer.Close()
	require.NoError(t, err)
}

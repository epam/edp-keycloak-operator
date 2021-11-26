package adapter

import (
	"fmt"
	"strings"
	"testing"

	"github.com/Nerzal/gocloak/v10"
	"github.com/go-resty/resty/v2"
	"github.com/jarcoal/httpmock"
	"github.com/pkg/errors"
)

func TestGoCloakAdapter_SyncAuthFlow(t *testing.T) {
	mockClient := new(MockGoCloakClient)
	restyClient := resty.New()
	httpmock.ActivateNonDefault(restyClient.GetClient())
	mockClient.On("RestyClient").Return(restyClient)

	adapter := GoCloakAdapter{
		client:   mockClient,
		basePath: "",
		token:    &gocloak.JWT{AccessToken: "token"},
	}

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

	realmName := "realm-name1"
	existFlowID := "flow-id-1"
	httpmock.RegisterResponder("GET", strings.ReplaceAll(authFlows, "{realm}", realmName),
		httpmock.NewJsonResponderOrPanic(200, []KeycloakAuthFlow{{Alias: flow.Alias, ID: existFlowID},
			{Alias: "some-another-flow", ID: "321"}}))
	deleteURL := strings.ReplaceAll(authFlow, "{realm}", realmName)
	deleteURL = strings.ReplaceAll(deleteURL, "{id}", existFlowID)
	httpmock.RegisterResponder("DELETE", deleteURL, httpmock.NewStringResponder(200, ""))

	mockClient.On("GetRealm", "token", realmName).Return(&gocloak.RealmRepresentation{
		BrowserFlow: gocloak.StringP("alias1"),
	}, nil)
	mockClient.On("UpdateRealm",
		gocloak.RealmRepresentation{BrowserFlow: gocloak.StringP("some-another-flow")}).
		Return(nil).Once()
	mockClient.On("UpdateRealm",
		gocloak.RealmRepresentation{BrowserFlow: gocloak.StringP("alias1")}).
		Return(nil).Once()

	createFlowResponse := httpmock.NewStringResponse(200, "")
	createFlowResponse.Header.Set("Location", "id/new-flow-id")

	httpmock.RegisterResponder("POST", strings.ReplaceAll(authFlows, "{realm}", realmName),
		httpmock.ResponderFromResponse(createFlowResponse))

	createExecResponse := httpmock.NewStringResponse(200, "")
	newExecID := "new-exec-id"
	createExecResponse.Header.Set("Location", fmt.Sprintf("id/%s", newExecID))
	httpmock.RegisterResponder("POST", strings.ReplaceAll(authFlowExecutionCreate, "{realm}", realmName),
		httpmock.ResponderFromResponse(createExecResponse))

	createConfigURL := strings.ReplaceAll(authFlowExecutionConfig, "{realm}", realmName)
	createConfigURL = strings.ReplaceAll(createConfigURL, "{id}", newExecID)

	httpmock.RegisterResponder("POST", createConfigURL, httpmock.NewStringResponder(200, ""))

	if err := adapter.SyncAuthFlow(realmName, &flow); err != nil {
		t.Fatalf("%+v", err)
	}
}

func TestGoCloakAdapter_DeleteAuthFlow(t *testing.T) {
	mockClient := new(MockGoCloakClient)
	restyClient := resty.New()
	httpmock.ActivateNonDefault(restyClient.GetClient())
	mockClient.On("RestyClient").Return(restyClient)

	adapter := GoCloakAdapter{
		client:   mockClient,
		basePath: "",
		token:    &gocloak.JWT{AccessToken: "token"},
	}

	realmName, flowAlias, existFlowID := "realm-name", "flow-alias", "id321"

	httpmock.RegisterResponder("GET", strings.ReplaceAll(authFlows, "{realm}", realmName),
		httpmock.NewJsonResponderOrPanic(200, []KeycloakAuthFlow{{Alias: flowAlias, ID: existFlowID}}))
	deleteURL := strings.ReplaceAll(authFlow, "{realm}", realmName)
	deleteURL = strings.ReplaceAll(deleteURL, "{id}", existFlowID)
	httpmock.RegisterResponder("DELETE", deleteURL, httpmock.NewStringResponder(200, ""))

	mockClient.On("GetRealm", "token", "realm-name").Return(&gocloak.RealmRepresentation{}, nil)

	if err := adapter.DeleteAuthFlow(realmName, flowAlias); err != nil {
		t.Fatalf("%+v", err)
	}
}

func TestGoCloakAdapter_SetRealmBrowserFlow(t *testing.T) {
	mockClient := new(MockGoCloakClient)
	adapter := GoCloakAdapter{
		client:   mockClient,
		basePath: "",
		token:    &gocloak.JWT{AccessToken: "token"},
	}
	realm := gocloak.RealmRepresentation{
		BrowserFlow: gocloak.StringP("flow1"),
	}

	mockClient.On("GetRealm", "token", "realm1").Return(&realm, nil)
	mockClient.On("UpdateRealm", realm).Return(nil)

	if err := adapter.SetRealmBrowserFlow("realm1", "flow1"); err != nil {
		t.Fatal(err)
	}
}

func TestGoCloakAdapter_SetRealmBrowserFlow_FailureGetRealm(t *testing.T) {
	mockClient := new(MockGoCloakClient)
	adapter := GoCloakAdapter{
		client:   mockClient,
		basePath: "",
		token:    &gocloak.JWT{AccessToken: "token"},
	}

	mockErr := errors.New("mock err")

	mockClient.On("GetRealm", "token", "realm1").Return(nil, mockErr)

	err := adapter.SetRealmBrowserFlow("realm1", "flow1")
	if err == nil {
		t.Fatal("no error on mock fatal")
	}
	if errors.Cause(err) != mockErr {
		t.Log(err)
		t.Fatal("wrong error returned")
	}
}

func TestGoCloakAdapter_SetRealmBrowserFlow_FailureUpdateRealm(t *testing.T) {
	mockClient := new(MockGoCloakClient)
	adapter := GoCloakAdapter{
		client:   mockClient,
		basePath: "",
		token:    &gocloak.JWT{AccessToken: "token"},
	}

	mockErr := errors.New("mock err")

	realm := gocloak.RealmRepresentation{
		BrowserFlow: gocloak.StringP("flow1"),
	}

	mockClient.On("GetRealm", "token", "realm1").Return(&realm, nil)
	mockClient.On("UpdateRealm", realm).Return(mockErr)

	err := adapter.SetRealmBrowserFlow("realm1", "flow1")
	if err == nil {
		t.Fatal("no error on mock fatal")
	}
	if errors.Cause(err) != mockErr {
		t.Log(err)
		t.Fatal("wrong error returned")
	}
}

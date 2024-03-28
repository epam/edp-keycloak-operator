package adapter

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"testing"

	"github.com/Nerzal/gocloak/v12"
	"github.com/go-resty/resty/v2"
	"github.com/jarcoal/httpmock"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/epam/edp-keycloak-operator/pkg/client/keycloak/adapter/mocks"
	logmock "github.com/epam/edp-keycloak-operator/pkg/client/keycloak/mock"
)

func TestGoCloakAdapter_CreateClientScope(t *testing.T) {
	mockClient := mocks.NewMockGoCloak(t)
	adapter := GoCloakAdapter{
		client:   mockClient,
		token:    &gocloak.JWT{AccessToken: "token"},
		basePath: "",
		log:      logmock.NewLogr(),
	}

	restyClient := resty.New()
	httpmock.ActivateNonDefault(restyClient.GetClient())
	mockClient.On("RestyClient").Return(restyClient)

	rsp := httpmock.NewStringResponse(200, "")
	defer closeWithFailOnError(t, rsp.Body)
	rsp.Header.Set("Location", "id/new-scope-id")

	httpmock.RegisterResponder("POST", strings.Replace(postClientScope, "{realm}", "realm1", 1),
		httpmock.ResponderFromResponse(rsp))

	defaultClientScopePath := strings.ReplaceAll(putDefaultClientScope, "{realm}", "realm1")
	defaultClientScopePath = strings.ReplaceAll(defaultClientScopePath, "{clientScopeID}", "new-scope-id")
	httpmock.RegisterResponder("PUT", defaultClientScopePath, httpmock.NewStringResponder(200, ""))

	id, err := adapter.CreateClientScope(context.Background(), "realm1",
		&ClientScope{Name: "demo", Default: true})
	require.NoError(t, err)

	if id == "" {
		t.Fatal("scope id is empty")
	}
}

func TestGoCloakAdapter_CreateClientScope_FailureSetDefault(t *testing.T) {
	mockClient := mocks.NewMockGoCloak(t)
	adapter := GoCloakAdapter{
		client:   mockClient,
		token:    &gocloak.JWT{AccessToken: "token"},
		basePath: "",
		log:      logmock.NewLogr(),
	}

	restyClient := resty.New()
	httpmock.ActivateNonDefault(restyClient.GetClient())
	mockClient.On("RestyClient").Return(restyClient)

	rsp := httpmock.NewStringResponse(200, "")
	defer closeWithFailOnError(t, rsp.Body)
	rsp.Header.Set("Location", "id/new-scope-id")
	httpmock.Reset()
	httpmock.RegisterResponder("POST", strings.Replace(postClientScope, "{realm}", "realm1", 1),
		httpmock.ResponderFromResponse(rsp))

	_, err := adapter.CreateClientScope(context.Background(), "realm1",
		&ClientScope{Name: "demo", Default: true})
	require.Error(t, err)

	if !strings.Contains(err.Error(), "unable to set default client scope for realm") {
		t.Fatalf("wrong error returned: %s", err.Error())
	}
}

func TestGoCloakAdapter_CreateClientScope_FailureCreate(t *testing.T) {
	mockClient := mocks.NewMockGoCloak(t)
	adapter := GoCloakAdapter{
		client: mockClient,
		token:  &gocloak.JWT{AccessToken: "token"},
	}

	restyClient := resty.New()

	httpmock.Reset()
	httpmock.ActivateNonDefault(restyClient.GetClient())
	mockClient.On("RestyClient").Return(restyClient)

	_, err := adapter.CreateClientScope(context.Background(), "realm1",
		&ClientScope{Name: "demo", Default: true})

	require.Error(t, err)

	if !strings.Contains(err.Error(), "unable to create client scope") {
		t.Fatalf("wrong error returned: %s", err.Error())
	}
}

func TestGoCloakAdapter_CreateClientScope_FailureGetID(t *testing.T) {
	mockClient := mocks.NewMockGoCloak(t)
	adapter := GoCloakAdapter{
		client: mockClient,
		token:  &gocloak.JWT{AccessToken: "token"},
	}

	restyClient := resty.New()

	httpmock.Reset()
	httpmock.ActivateNonDefault(restyClient.GetClient())
	mockClient.On("RestyClient").Return(restyClient)

	rsp := httpmock.NewStringResponse(200, "")
	defer closeWithFailOnError(t, rsp.Body)
	httpmock.RegisterResponder("POST", strings.Replace(postClientScope, "{realm}", "realm1", 1),
		httpmock.ResponderFromResponse(rsp))

	_, err := adapter.CreateClientScope(context.Background(), "realm1",
		&ClientScope{Name: "demo", Default: true})

	err = errors.Cause(err)
	require.Error(t, err)

	if !strings.Contains(err.Error(), "location header is not set or empty") {
		t.Fatalf("wrong error returned: %s", err.Error())
	}
}

func TestGoCloakAdapter_UpdateClientScope(t *testing.T) {
	mockClient := mocks.NewMockGoCloak(t)
	adapter := GoCloakAdapter{
		client:   mockClient,
		token:    &gocloak.JWT{AccessToken: "token"},
		basePath: "",
		log:      logmock.NewLogr(),
	}

	var (
		realmName = "realm1"
		scopeID   = "scope1"
	)

	restyClient := resty.New()
	httpmock.ActivateNonDefault(restyClient.GetClient())
	mockClient.On("RestyClient").Return(restyClient)
	mockClient.On("GetClientScope", mock.Anything, "token", realmName, scopeID).Return(&gocloak.ClientScope{
		ID: gocloak.StringP("scope1"),
		ProtocolMappers: &[]gocloak.ProtocolMappers{
			{
				Name: gocloak.StringP("mp1"),
				ID:   gocloak.StringP("mp_id1"),
			},
		},
	}, nil)

	putClientScope := strings.ReplaceAll(putClientScope, "{realm}", realmName)
	putClientScope = strings.ReplaceAll(putClientScope, "{id}", scopeID)
	httpmock.RegisterResponder("PUT", putClientScope, httpmock.NewStringResponder(200, ""))

	deleteDefaultClientScope := strings.ReplaceAll(deleteDefaultClientScope, "{realm}", realmName)
	deleteDefaultClientScope = strings.ReplaceAll(deleteDefaultClientScope, "{clientScopeID}", scopeID)
	httpmock.RegisterResponder("DELETE", deleteDefaultClientScope, httpmock.NewStringResponder(200, ""))

	deleteClientScopeProtocolMapper := strings.ReplaceAll(deleteClientScopeProtocolMapper, "{realm}", realmName)
	deleteClientScopeProtocolMapper = strings.ReplaceAll(deleteClientScopeProtocolMapper, "{clientScopeID}", scopeID)
	deleteClientScopeProtocolMapper = strings.ReplaceAll(deleteClientScopeProtocolMapper, "{protocolMapperID}", "mp_id1")
	httpmock.RegisterResponder("DELETE", deleteClientScopeProtocolMapper, httpmock.NewStringResponder(200, ""))

	createClientScopeProtocolMapper := strings.ReplaceAll(createClientScopeProtocolMapper, "{realm}", realmName)
	createClientScopeProtocolMapper = strings.ReplaceAll(createClientScopeProtocolMapper, "{clientScopeID}", scopeID)
	httpmock.RegisterResponder("POST", createClientScopeProtocolMapper, httpmock.NewStringResponder(200, ""))

	putDefaultClientScope := strings.ReplaceAll(putDefaultClientScope, "{realm}", realmName)
	putDefaultClientScope = strings.ReplaceAll(putDefaultClientScope, "{clientScopeID}", scopeID)
	httpmock.RegisterResponder("PUT", putDefaultClientScope, httpmock.NewStringResponder(200, ""))
	httpmock.RegisterResponder("GET", "/admin/realms/realm1/default-default-client-scopes",
		httpmock.NewJsonResponderOrPanic(200, []ClientScope{}))

	if err := adapter.UpdateClientScope(context.Background(), realmName, scopeID, &ClientScope{
		Name: "scope1",
		ProtocolMappers: []ProtocolMapper{
			{
				Name: "mp2",
			},
		},
	}); err != nil {
		t.Fatalf("%+v", err)
	}

	if err := adapter.UpdateClientScope(context.Background(), realmName, scopeID, &ClientScope{
		Name: "scope1",
		ProtocolMappers: []ProtocolMapper{
			{
				Name: "mp2",
			},
		},
		Default: true,
	}); err != nil {
		t.Fatalf("%+v", err)
	}

	httpmock.RegisterResponder("GET", "/admin/realms/realm1/default-default-client-scopes",
		httpmock.NewJsonResponderOrPanic(200, []ClientScope{{Name: "scope1"}}))

	if err := adapter.UpdateClientScope(context.Background(), realmName, scopeID, &ClientScope{
		Name: "scope1",
		ProtocolMappers: []ProtocolMapper{
			{
				Name: "mp2",
			},
		},
		Default: false,
	}); err != nil {
		t.Fatalf("%+v", err)
	}
}

func TestGoCloakAdapter_GetClientScope(t *testing.T) {
	mockClient := mocks.NewMockGoCloak(t)
	adapter := GoCloakAdapter{
		client:   mockClient,
		token:    &gocloak.JWT{AccessToken: "token"},
		basePath: "",
		log:      logmock.NewLogr(),
	}

	restyClient := resty.New()
	httpmock.ActivateNonDefault(restyClient.GetClient())
	mockClient.On("RestyClient").Return(restyClient)

	result := []ClientScope{{Name: "name1"}}

	getOneClientScope := strings.ReplaceAll(getRealmClientScopes, "{realm}", "realm1")
	httpmock.RegisterResponder("GET", getOneClientScope,
		httpmock.NewJsonResponderOrPanic(200, &result))

	_, err := adapter.GetClientScope("name1", "realm1")
	require.NoError(t, err)
}

func TestGoCloakAdapter_DeleteClientScope(t *testing.T) {
	mockClient := mocks.NewMockGoCloak(t)
	adapter := GoCloakAdapter{
		client:   mockClient,
		token:    &gocloak.JWT{AccessToken: "token"},
		basePath: "",
		log:      logmock.NewLogr(),
	}

	restyClient := resty.New()
	httpmock.ActivateNonDefault(restyClient.GetClient())
	mockClient.On("RestyClient").Return(restyClient)

	deleteDefaultClientScope := strings.ReplaceAll(deleteDefaultClientScope, "{realm}", "realm1")
	deleteDefaultClientScope = strings.ReplaceAll(deleteDefaultClientScope, "{clientScopeID}", "scope1")

	httpmock.RegisterResponder("DELETE", deleteDefaultClientScope, httpmock.NewStringResponder(200, ""))
	mockClient.On("DeleteClientScope", mock.Anything, "token", "realm1", "scope1").Return(nil)

	err := adapter.DeleteClientScope(context.Background(), "realm1", "scope1")
	require.NoError(t, err)
}

func TestGetClientScope(t *testing.T) {
	_, err := getClientScope("scope1", []ClientScope{})
	if !IsErrNotFound(err) {
		t.Fatalf("wrong error returned: %s", err.Error())
	}
}

func TestGoCloakAdapter_DeleteClientScope_Failure(t *testing.T) {
	mockClient := mocks.NewMockGoCloak(t)
	adapter := GoCloakAdapter{
		client:   mockClient,
		token:    &gocloak.JWT{AccessToken: "token"},
		basePath: "",
		log:      logmock.NewLogr(),
	}

	restyClient := resty.New()
	httpmock.ActivateNonDefault(restyClient.GetClient())
	mockClient.On("RestyClient").Return(restyClient)
	httpmock.Reset()

	err := adapter.DeleteClientScope(context.Background(), "realm1", "scope1")
	require.Error(t, err)

	if !strings.Contains(err.Error(), "unable to unset default client scope for realm") {
		t.Fatalf("wrong error returned: %s", err.Error())
	}

	deleteDefaultClientScope := strings.ReplaceAll(deleteDefaultClientScope, "{realm}", "realm1")
	deleteDefaultClientScope = strings.ReplaceAll(deleteDefaultClientScope, "{clientScopeID}", "scope1")

	httpmock.RegisterResponder("DELETE", deleteDefaultClientScope, httpmock.NewStringResponder(200, ""))
	mockClient.On("DeleteClientScope", mock.Anything, "token", "realm1", "scope1").Return(errors.New("logmock fatal"))

	err = adapter.DeleteClientScope(context.Background(), "realm1", "scope1")
	require.Error(t, err)

	if !strings.Contains(err.Error(), "unable to delete client scope") {
		t.Fatalf("wrong error returned: %s", err.Error())
	}
}

func TestGoCloakAdapter_GetClientScopeMappers(t *testing.T) {
	kcClient, _, _ := initAdapter(t)

	httpmock.Reset()
	httpmock.RegisterResponder("GET",
		"/admin/realms/realm1/client-scopes/scope1/protocol-mappers/models",
		httpmock.NewStringResponder(200, ""))

	_, err := kcClient.GetClientScopeMappers(context.Background(), "realm1", "scope1")
	require.NoError(t, err)

	httpmock.RegisterResponder("GET",
		"/admin/realms/realm1/client-scopes/scope2/protocol-mappers/models",
		httpmock.NewStringResponder(422, "forbidden"))

	_, err = kcClient.GetClientScopeMappers(context.Background(), "realm1", "scope2")
	require.Error(t, err)

	if err.Error() != "unable to get client scope mappers: status: 422, body: forbidden" {
		t.Fatalf("wrong error returned: '%s'", err.Error())
	}
}

func TestGoCloakAdapter_PutClientScopeMapper(t *testing.T) {
	kcClient, _, _ := initAdapter(t)

	httpmock.RegisterResponder("POST",
		"/admin/realms/realm1/client-scopes/scope1/protocol-mappers/models",
		httpmock.NewStringResponder(200, ""))

	err := kcClient.PutClientScopeMapper("realm1", "scope1", &ProtocolMapper{})
	require.NoError(t, err)

	httpmock.RegisterResponder("POST",
		"/admin/realms/realm1/client-scopes/scope2/protocol-mappers/models",
		httpmock.NewStringResponder(422, "forbidden"))

	err = kcClient.PutClientScopeMapper("realm1", "scope2", &ProtocolMapper{})
	require.Error(t, err)

	if err.Error() != "unable to put client scope mapper: status: 422, body: forbidden" {
		t.Fatalf("wrong error returned: '%s'", err.Error())
	}
}

func TestGoCloakAdapter_GetClientScopesByNames(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		realm      string
		response   httpmock.Responder
		scopeNames []string
		want       []ClientScope
		wantErr    require.ErrorAssertionFunc
	}{
		{
			name:  "should get client scope",
			realm: "realm1",
			response: httpmock.NewJsonResponderOrPanic(http.StatusOK, []ClientScope{
				{
					ID:   "testScope",
					Name: "scope1",
				},
			}),
			scopeNames: []string{"scope1"},
			want: []ClientScope{
				{
					ID:   "testScope",
					Name: "scope1",
				},
			},
			wantErr: require.NoError,
		},
		{
			name:  "should not find the client scope",
			realm: "realm2",
			response: httpmock.NewJsonResponderOrPanic(http.StatusOK, []ClientScope{
				{
					ID:   "testScope",
					Name: "scope2",
				},
			}),
			scopeNames: []string{"scope1, scope"},
			want:       nil,
			wantErr:    require.Error,
		},
		{
			name:       "should fail to get scopes",
			realm:      "realm3",
			response:   httpmock.NewStringResponder(http.StatusBadRequest, ""),
			scopeNames: []string{"scope1"},
			want:       nil,
			wantErr:    require.Error,
		},
	}

	for _, tt := range tests {
		tt := tt

		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			adapter, _, _ := initAdapter(t)

			httpmock.RegisterResponder(http.MethodGet, fmt.Sprintf("/admin/realms/%s/client-scopes", tt.realm), tt.response)

			got, err := adapter.GetClientScopesByNames(context.Background(), tt.realm, tt.scopeNames)
			tt.wantErr(t, err)

			require.Equal(t, tt.want, got)
		})
	}
}

package chain

import (
	"context"
	"errors"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"k8s.io/utils/ptr"

	keycloakApi "github.com/epam/edp-keycloak-operator/api/v1"
	keycloakv2 "github.com/epam/edp-keycloak-operator/pkg/client/keycloakv2"
	"github.com/epam/edp-keycloak-operator/pkg/client/keycloakv2/mocks"
)

const testExecID = "exec-id-456"

func locationResponse(id string) *keycloakv2.Response {
	return &keycloakv2.Response{
		HTTPResponse: &http.Response{
			Header: http.Header{
				"Location": []string{"http://localhost/admin/realms/test-realm/authentication/executions/" + id},
			},
		},
	}
}

func TestSyncAuthFlowExecutions_Serve_NoExistingNoSpec(t *testing.T) {
	mockFlows := mocks.NewMockAuthFlowsClient(t)
	kc := &keycloakv2.KeycloakClient{AuthFlows: mockFlows}

	flow := &keycloakApi.KeycloakAuthFlow{}
	flow.Spec.Alias = testFlowAlias

	// clearNonFlowExecutions: nothing to delete
	mockFlows.EXPECT().GetFlowExecutions(context.Background(), testRealmName, testFlowAlias).
		Return([]keycloakv2.AuthenticationExecutionInfoRepresentation{}, nil, nil)

	// adjustChildFlowsPriority: no child flow specs → returns immediately

	h := NewSyncAuthFlowExecutions(kc)
	err := h.Serve(context.Background(), flow, testRealmName)
	require.NoError(t, err)
}

func TestSyncAuthFlowExecutions_Serve_ClearExistingNonFlowExec(t *testing.T) {
	mockFlows := mocks.NewMockAuthFlowsClient(t)
	kc := &keycloakv2.KeycloakClient{AuthFlows: mockFlows}

	flow := &keycloakApi.KeycloakAuthFlow{}
	flow.Spec.Alias = testFlowAlias

	// clearNonFlowExecutions: one non-flow top-level exec to delete
	mockFlows.EXPECT().GetFlowExecutions(context.Background(), testRealmName, testFlowAlias).
		Return([]keycloakv2.AuthenticationExecutionInfoRepresentation{
			{Id: ptr.To("old-exec"), AuthenticationFlow: ptr.To(false), Level: ptr.To(int32(0))},
		}, nil, nil)

	mockFlows.EXPECT().DeleteExecution(context.Background(), testRealmName, "old-exec").
		Return(nil, nil)

	h := NewSyncAuthFlowExecutions(kc)
	err := h.Serve(context.Background(), flow, testRealmName)
	require.NoError(t, err)
}

func TestSyncAuthFlowExecutions_Serve_SkipFlowTypeExec(t *testing.T) {
	mockFlows := mocks.NewMockAuthFlowsClient(t)
	kc := &keycloakv2.KeycloakClient{AuthFlows: mockFlows}

	flow := &keycloakApi.KeycloakAuthFlow{}
	flow.Spec.Alias = testFlowAlias

	// clearNonFlowExecutions: one flow-type exec — must NOT be deleted
	mockFlows.EXPECT().GetFlowExecutions(context.Background(), testRealmName, testFlowAlias).
		Return([]keycloakv2.AuthenticationExecutionInfoRepresentation{
			{Id: ptr.To("flow-exec"), AuthenticationFlow: ptr.To(true), Level: ptr.To(int32(0))},
		}, nil, nil)

	// DeleteExecution must NOT be called

	h := NewSyncAuthFlowExecutions(kc)
	err := h.Serve(context.Background(), flow, testRealmName)
	require.NoError(t, err)
}

func TestSyncAuthFlowExecutions_Serve_AddExecution(t *testing.T) {
	mockFlows := mocks.NewMockAuthFlowsClient(t)
	kc := &keycloakv2.KeycloakClient{AuthFlows: mockFlows}

	flow := &keycloakApi.KeycloakAuthFlow{}
	flow.Spec.Alias = testFlowAlias
	flow.Status.ID = testFlowID
	flow.Spec.AuthenticationExecutions = []keycloakApi.AuthenticationExecution{
		{Authenticator: "basic-auth", AuthenticatorFlow: false},
	}

	// clearNonFlowExecutions: empty
	mockFlows.EXPECT().GetFlowExecutions(context.Background(), testRealmName, testFlowAlias).
		Return([]keycloakv2.AuthenticationExecutionInfoRepresentation{}, nil, nil)

	// addExecution
	mockFlows.EXPECT().AddExecutionToFlow(
		context.Background(), testRealmName,
		keycloakv2.AuthenticationExecutionRepresentation{
			Authenticator: ptr.To("basic-auth"),
			ParentFlow:    ptr.To(testFlowID),
			Requirement:   ptr.To(""),
		},
	).Return(locationResponse(testExecID), nil)

	// adjustChildFlowsPriority: no AuthenticatorFlow=true entries → returns immediately

	h := NewSyncAuthFlowExecutions(kc)
	err := h.Serve(context.Background(), flow, testRealmName)
	require.NoError(t, err)
}

func TestSyncAuthFlowExecutions_Serve_AddExecutionWithConfig(t *testing.T) {
	mockFlows := mocks.NewMockAuthFlowsClient(t)
	kc := &keycloakv2.KeycloakClient{AuthFlows: mockFlows}

	flow := &keycloakApi.KeycloakAuthFlow{}
	flow.Spec.Alias = testFlowAlias
	flow.Status.ID = testFlowID
	flow.Spec.AuthenticationExecutions = []keycloakApi.AuthenticationExecution{
		{
			Authenticator:     "identity-provider-redirector",
			AuthenticatorFlow: false,
			AuthenticatorConfig: &keycloakApi.AuthenticatorConfig{
				Alias:  "idp-config",
				Config: map[string]string{"defaultProvider": "github"},
			},
		},
	}

	mockFlows.EXPECT().GetFlowExecutions(context.Background(), testRealmName, testFlowAlias).
		Return([]keycloakv2.AuthenticationExecutionInfoRepresentation{}, nil, nil)

	mockFlows.EXPECT().AddExecutionToFlow(
		context.Background(), testRealmName,
		keycloakv2.AuthenticationExecutionRepresentation{
			Authenticator: ptr.To("identity-provider-redirector"),
			ParentFlow:    ptr.To(testFlowID),
			Requirement:   ptr.To(""),
		},
	).Return(locationResponse(testExecID), nil)

	cfg := map[string]string{"defaultProvider": "github"}
	mockFlows.EXPECT().CreateExecutionConfig(
		context.Background(), testRealmName, testExecID,
		keycloakv2.AuthenticatorConfigRepresentation{
			Alias:  ptr.To("idp-config"),
			Config: &cfg,
		},
	).Return(nil, nil)

	h := NewSyncAuthFlowExecutions(kc)
	err := h.Serve(context.Background(), flow, testRealmName)
	require.NoError(t, err)
}

func TestSyncAuthFlowExecutions_Serve_AdjustChildFlowPriority(t *testing.T) {
	mockFlows := mocks.NewMockAuthFlowsClient(t)
	kc := &keycloakv2.KeycloakClient{AuthFlows: mockFlows}

	flow := &keycloakApi.KeycloakAuthFlow{}
	flow.Spec.Alias = testFlowAlias
	flow.Spec.AuthenticationExecutions = []keycloakApi.AuthenticationExecution{
		{Alias: "sub-flow", AuthenticatorFlow: true, Priority: 10, Requirement: "REQUIRED"},
	}

	// clearNonFlowExecutions: one flow exec (not deleted)
	mockFlows.EXPECT().GetFlowExecutions(context.Background(), testRealmName, testFlowAlias).
		Return([]keycloakv2.AuthenticationExecutionInfoRepresentation{
			{
				DisplayName:        ptr.To("sub-flow"),
				AuthenticationFlow: ptr.To(true),
				Level:              ptr.To(int32(0)),
				Priority:           ptr.To(int32(20)), // differs from spec (10)
				Requirement:        ptr.To("DISABLED"),
			},
		}, nil, nil).Once()

	// adjustChildFlowsPriority re-fetches executions
	updatedExec := keycloakv2.AuthenticationExecutionInfoRepresentation{
		DisplayName:        ptr.To("sub-flow"),
		AuthenticationFlow: ptr.To(true),
		Level:              ptr.To(int32(0)),
		Priority:           ptr.To(int32(10)),
		Requirement:        ptr.To("REQUIRED"),
	}

	mockFlows.EXPECT().GetFlowExecutions(context.Background(), testRealmName, testFlowAlias).
		Return([]keycloakv2.AuthenticationExecutionInfoRepresentation{
			{
				DisplayName:        ptr.To("sub-flow"),
				AuthenticationFlow: ptr.To(true),
				Level:              ptr.To(int32(0)),
				Priority:           ptr.To(int32(20)),
				Requirement:        ptr.To("DISABLED"),
			},
		}, nil, nil).Once()

	mockFlows.EXPECT().UpdateFlowExecution(context.Background(), testRealmName, testFlowAlias, updatedExec).
		Return(nil, nil)

	h := NewSyncAuthFlowExecutions(kc)
	err := h.Serve(context.Background(), flow, testRealmName)
	require.NoError(t, err)
}

func TestSyncAuthFlowExecutions_Serve_GetFlowExecutionsError(t *testing.T) {
	mockFlows := mocks.NewMockAuthFlowsClient(t)
	kc := &keycloakv2.KeycloakClient{AuthFlows: mockFlows}

	flow := &keycloakApi.KeycloakAuthFlow{}
	flow.Spec.Alias = testFlowAlias

	mockFlows.EXPECT().GetFlowExecutions(context.Background(), testRealmName, testFlowAlias).
		Return(nil, nil, errors.New("api error"))

	h := NewSyncAuthFlowExecutions(kc)
	err := h.Serve(context.Background(), flow, testRealmName)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to clear flow executions")
}

func TestSyncAuthFlowExecutions_Serve_DeleteExecutionError(t *testing.T) {
	mockFlows := mocks.NewMockAuthFlowsClient(t)
	kc := &keycloakv2.KeycloakClient{AuthFlows: mockFlows}

	flow := &keycloakApi.KeycloakAuthFlow{}
	flow.Spec.Alias = testFlowAlias

	mockFlows.EXPECT().GetFlowExecutions(context.Background(), testRealmName, testFlowAlias).
		Return([]keycloakv2.AuthenticationExecutionInfoRepresentation{
			{Id: ptr.To("bad-exec"), AuthenticationFlow: ptr.To(false), Level: ptr.To(int32(0))},
		}, nil, nil)

	mockFlows.EXPECT().DeleteExecution(context.Background(), testRealmName, "bad-exec").
		Return(nil, errors.New("delete failed"))

	h := NewSyncAuthFlowExecutions(kc)
	err := h.Serve(context.Background(), flow, testRealmName)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to delete execution")
}

func TestSyncAuthFlowExecutions_Serve_AddExecutionError(t *testing.T) {
	mockFlows := mocks.NewMockAuthFlowsClient(t)
	kc := &keycloakv2.KeycloakClient{AuthFlows: mockFlows}

	flow := &keycloakApi.KeycloakAuthFlow{}
	flow.Spec.Alias = testFlowAlias
	flow.Status.ID = testFlowID
	flow.Spec.AuthenticationExecutions = []keycloakApi.AuthenticationExecution{
		{Authenticator: "basic-auth"},
	}

	mockFlows.EXPECT().GetFlowExecutions(context.Background(), testRealmName, testFlowAlias).
		Return([]keycloakv2.AuthenticationExecutionInfoRepresentation{}, nil, nil)

	mockFlows.EXPECT().AddExecutionToFlow(
		context.Background(), testRealmName,
		keycloakv2.AuthenticationExecutionRepresentation{
			Authenticator: ptr.To("basic-auth"),
			ParentFlow:    ptr.To(testFlowID),
			Requirement:   ptr.To(""),
		},
	).Return(nil, errors.New("add failed"))

	h := NewSyncAuthFlowExecutions(kc)
	err := h.Serve(context.Background(), flow, testRealmName)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to add execution")
}

func TestSyncAuthFlowExecutions_Serve_EmptyFlowID(t *testing.T) {
	mockFlows := mocks.NewMockAuthFlowsClient(t)
	kc := &keycloakv2.KeycloakClient{AuthFlows: mockFlows}

	flow := &keycloakApi.KeycloakAuthFlow{}
	flow.Spec.Alias = testFlowAlias
	// Status.ID intentionally left empty
	flow.Spec.AuthenticationExecutions = []keycloakApi.AuthenticationExecution{
		{Authenticator: "basic-auth"},
	}

	mockFlows.EXPECT().GetFlowExecutions(context.Background(), testRealmName, testFlowAlias).
		Return([]keycloakv2.AuthenticationExecutionInfoRepresentation{}, nil, nil)

	h := NewSyncAuthFlowExecutions(kc)
	err := h.Serve(context.Background(), flow, testRealmName)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "flow ID is empty")
}

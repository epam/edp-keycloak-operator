package chain

import (
	"context"
	"errors"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"k8s.io/utils/ptr"

	keycloakApi "github.com/epam/edp-keycloak-operator/api/v1"
	"github.com/epam/edp-keycloak-operator/pkg/client/keycloakapi"
	"github.com/epam/edp-keycloak-operator/pkg/client/keycloakapi/mocks"
)

const testExecID = "exec-id-456"

func locationResponse(id string) *keycloakapi.Response {
	return &keycloakapi.Response{
		HTTPResponse: &http.Response{
			Header: http.Header{
				"Location": []string{"http://localhost/admin/realms/test-realm/authentication/executions/" + id},
			},
		},
	}
}

// setIdpRedirectorSpec fills the spec with one configured identity-provider-redirector execution.
func setIdpRedirectorSpec(flow *keycloakApi.KeycloakAuthFlow) {
	flow.Spec.Alias = testFlowAlias
	flow.Spec.AuthenticationExecutions = []keycloakApi.AuthenticationExecution{
		{
			Authenticator: "identity-provider-redirector",
			Requirement:   "ALTERNATIVE",
			Priority:      10,
			AuthenticatorConfig: &keycloakApi.AuthenticatorConfig{
				Alias:  "idp-config",
				Config: map[string]string{"defaultProvider": "github"},
			},
		},
	}
}

// nonFlowExec builds a top-level, non-flow execution as returned by GetFlowExecutions.
func nonFlowExec(id, providerID, requirement string, priority int32, configID *string) keycloakapi.AuthenticationExecutionInfoRepresentation {
	return keycloakapi.AuthenticationExecutionInfoRepresentation{
		Id:                   ptr.To(id),
		ProviderId:           ptr.To(providerID),
		AuthenticationFlow:   ptr.To(false),
		Level:                ptr.To(int32(0)),
		Requirement:          ptr.To(requirement),
		Priority:             ptr.To(priority),
		AuthenticationConfig: configID,
	}
}

func TestSyncAuthFlowExecutions_Serve_NoExistingNoSpec(t *testing.T) {
	mockFlows := mocks.NewMockAuthFlowsClient(t)
	kc := &keycloakapi.KeycloakClient{AuthFlows: mockFlows}

	flow := &keycloakApi.KeycloakAuthFlow{}
	flow.Spec.Alias = testFlowAlias

	mockFlows.EXPECT().GetFlowExecutions(context.Background(), testRealmName, testFlowAlias).
		Return([]keycloakapi.AuthenticationExecutionInfoRepresentation{}, nil, nil)

	h := NewSyncAuthFlowExecutions(kc)
	err := h.Serve(context.Background(), flow, testRealmName)
	require.NoError(t, err)
}

func TestSyncAuthFlowExecutions_Serve_SkipFlowTypeExec(t *testing.T) {
	mockFlows := mocks.NewMockAuthFlowsClient(t)
	kc := &keycloakapi.KeycloakClient{AuthFlows: mockFlows}

	flow := &keycloakApi.KeycloakAuthFlow{}
	flow.Spec.Alias = testFlowAlias
	flow.Spec.AuthenticationExecutions = []keycloakApi.AuthenticationExecution{
		{Alias: "sub-flow", AuthenticatorFlow: true},
	}

	// Flow-type exec is filtered from the non-flow sync; spec entry already matches → no update
	mockFlows.EXPECT().GetFlowExecutions(context.Background(), testRealmName, testFlowAlias).
		Return([]keycloakapi.AuthenticationExecutionInfoRepresentation{
			{Id: ptr.To("flow-exec"), DisplayName: ptr.To("sub-flow"), AuthenticationFlow: ptr.To(true), Level: ptr.To(int32(0)), Priority: ptr.To(int32(0))},
		}, nil, nil).Once()

	h := NewSyncAuthFlowExecutions(kc)
	err := h.Serve(context.Background(), flow, testRealmName)
	require.NoError(t, err)
}

func TestSyncAuthFlowExecutions_Serve_InSync(t *testing.T) {
	mockFlows := mocks.NewMockAuthFlowsClient(t)
	kc := &keycloakapi.KeycloakClient{AuthFlows: mockFlows}

	flow := &keycloakApi.KeycloakAuthFlow{}
	flow.Spec.Alias = testFlowAlias
	flow.Spec.AuthenticationExecutions = []keycloakApi.AuthenticationExecution{
		{Authenticator: "basic-auth", Requirement: "REQUIRED", Priority: 10},
	}

	mockFlows.EXPECT().GetFlowExecutions(context.Background(), testRealmName, testFlowAlias).
		Return([]keycloakapi.AuthenticationExecutionInfoRepresentation{
			nonFlowExec("exec-1", "basic-auth", "REQUIRED", 10, nil),
		}, nil, nil)

	h := NewSyncAuthFlowExecutions(kc)
	err := h.Serve(context.Background(), flow, testRealmName)
	require.NoError(t, err)
}

func TestSyncAuthFlowExecutions_Serve_RequirementDrift(t *testing.T) {
	mockFlows := mocks.NewMockAuthFlowsClient(t)
	kc := &keycloakapi.KeycloakClient{AuthFlows: mockFlows}

	flow := &keycloakApi.KeycloakAuthFlow{}
	flow.Spec.Alias = testFlowAlias
	flow.Spec.AuthenticationExecutions = []keycloakApi.AuthenticationExecution{
		{Authenticator: "basic-auth", Requirement: "REQUIRED", Priority: 10},
	}

	existing := nonFlowExec("exec-1", "basic-auth", "DISABLED", 10, nil)

	mockFlows.EXPECT().GetFlowExecutions(context.Background(), testRealmName, testFlowAlias).
		Return([]keycloakapi.AuthenticationExecutionInfoRepresentation{existing}, nil, nil)

	updated := existing
	updated.Requirement = ptr.To("REQUIRED")
	updated.Priority = ptr.To(int32(10))

	mockFlows.EXPECT().UpdateFlowExecution(context.Background(), testRealmName, testFlowAlias, updated).
		Return(nil, nil)

	h := NewSyncAuthFlowExecutions(kc)
	err := h.Serve(context.Background(), flow, testRealmName)
	require.NoError(t, err)
}

func TestSyncAuthFlowExecutions_Serve_PriorityDrift(t *testing.T) {
	mockFlows := mocks.NewMockAuthFlowsClient(t)
	kc := &keycloakapi.KeycloakClient{AuthFlows: mockFlows}

	flow := &keycloakApi.KeycloakAuthFlow{}
	flow.Spec.Alias = testFlowAlias
	flow.Spec.AuthenticationExecutions = []keycloakApi.AuthenticationExecution{
		{Authenticator: "basic-auth", Requirement: "REQUIRED", Priority: 30},
	}

	existing := nonFlowExec("exec-1", "basic-auth", "REQUIRED", 10, nil)

	mockFlows.EXPECT().GetFlowExecutions(context.Background(), testRealmName, testFlowAlias).
		Return([]keycloakapi.AuthenticationExecutionInfoRepresentation{existing}, nil, nil)

	updated := existing
	updated.Priority = ptr.To(int32(30))

	mockFlows.EXPECT().UpdateFlowExecution(context.Background(), testRealmName, testFlowAlias, updated).
		Return(nil, nil)

	h := NewSyncAuthFlowExecutions(kc)
	err := h.Serve(context.Background(), flow, testRealmName)
	require.NoError(t, err)
}

func TestSyncAuthFlowExecutions_Serve_ConfigDrift(t *testing.T) {
	mockFlows := mocks.NewMockAuthFlowsClient(t)
	kc := &keycloakapi.KeycloakClient{AuthFlows: mockFlows}

	const configID = "cfg-1"

	flow := &keycloakApi.KeycloakAuthFlow{}
	setIdpRedirectorSpec(flow)

	existing := nonFlowExec("exec-1", "identity-provider-redirector", "ALTERNATIVE", 10, ptr.To(configID))

	mockFlows.EXPECT().GetFlowExecutions(context.Background(), testRealmName, testFlowAlias).
		Return([]keycloakapi.AuthenticationExecutionInfoRepresentation{existing}, nil, nil)

	existingConfig := map[string]string{"defaultProvider": "gitlab"}
	mockFlows.EXPECT().GetAuthenticatorConfig(context.Background(), testRealmName, configID).
		Return(&keycloakapi.AuthenticatorConfigRepresentation{
			Id:     ptr.To(configID),
			Alias:  ptr.To("idp-config"),
			Config: &existingConfig,
		}, nil, nil)

	desiredConfig := map[string]string{"defaultProvider": "github"}
	mockFlows.EXPECT().UpdateAuthenticatorConfig(context.Background(), testRealmName, configID,
		keycloakapi.AuthenticatorConfigRepresentation{
			Id:     ptr.To(configID),
			Alias:  ptr.To("idp-config"),
			Config: &desiredConfig,
		},
	).Return(nil, nil)

	h := NewSyncAuthFlowExecutions(kc)
	err := h.Serve(context.Background(), flow, testRealmName)
	require.NoError(t, err)
}

func TestSyncAuthFlowExecutions_Serve_ConfigRemovedFromSpec(t *testing.T) {
	mockFlows := mocks.NewMockAuthFlowsClient(t)
	kc := &keycloakapi.KeycloakClient{AuthFlows: mockFlows}

	const configID = "cfg-2"

	flow := &keycloakApi.KeycloakAuthFlow{}
	flow.Spec.Alias = testFlowAlias
	flow.Spec.AuthenticationExecutions = []keycloakApi.AuthenticationExecution{
		{Authenticator: "basic-auth", Requirement: "REQUIRED", Priority: 10},
	}

	existing := nonFlowExec("exec-1", "basic-auth", "REQUIRED", 10, ptr.To(configID))

	mockFlows.EXPECT().GetFlowExecutions(context.Background(), testRealmName, testFlowAlias).
		Return([]keycloakapi.AuthenticationExecutionInfoRepresentation{existing}, nil, nil)

	mockFlows.EXPECT().DeleteAuthenticatorConfig(context.Background(), testRealmName, configID).
		Return(nil, nil)

	h := NewSyncAuthFlowExecutions(kc)
	err := h.Serve(context.Background(), flow, testRealmName)
	require.NoError(t, err)
}

func TestSyncAuthFlowExecutions_Serve_ExecutionRemovedFromSpec(t *testing.T) {
	mockFlows := mocks.NewMockAuthFlowsClient(t)
	kc := &keycloakapi.KeycloakClient{AuthFlows: mockFlows}

	const configID = "cfg-3"

	flow := &keycloakApi.KeycloakAuthFlow{}
	flow.Spec.Alias = testFlowAlias

	existing := nonFlowExec("exec-removed", "old-auth", "REQUIRED", 10, ptr.To(configID))

	mockFlows.EXPECT().GetFlowExecutions(context.Background(), testRealmName, testFlowAlias).
		Return([]keycloakapi.AuthenticationExecutionInfoRepresentation{existing}, nil, nil)

	mockFlows.EXPECT().DeleteAuthenticatorConfig(context.Background(), testRealmName, configID).
		Return(nil, nil)

	mockFlows.EXPECT().DeleteExecution(context.Background(), testRealmName, "exec-removed").
		Return(nil, nil)

	h := NewSyncAuthFlowExecutions(kc)
	err := h.Serve(context.Background(), flow, testRealmName)
	require.NoError(t, err)
}

func TestSyncAuthFlowExecutions_Serve_ExecutionRemovedFromSpec_ConfigDeleteNotFound(t *testing.T) {
	mockFlows := mocks.NewMockAuthFlowsClient(t)
	kc := &keycloakapi.KeycloakClient{AuthFlows: mockFlows}

	const configID = "cfg-4"

	flow := &keycloakApi.KeycloakAuthFlow{}
	flow.Spec.Alias = testFlowAlias

	existing := nonFlowExec("exec-removed", "old-auth", "REQUIRED", 10, ptr.To(configID))

	mockFlows.EXPECT().GetFlowExecutions(context.Background(), testRealmName, testFlowAlias).
		Return([]keycloakapi.AuthenticationExecutionInfoRepresentation{existing}, nil, nil)

	mockFlows.EXPECT().DeleteAuthenticatorConfig(context.Background(), testRealmName, configID).
		Return(nil, keycloakapi.ErrNotFound)

	mockFlows.EXPECT().DeleteExecution(context.Background(), testRealmName, "exec-removed").
		Return(nil, nil)

	h := NewSyncAuthFlowExecutions(kc)
	err := h.Serve(context.Background(), flow, testRealmName)
	require.NoError(t, err)
}

func TestSyncAuthFlowExecutions_Serve_AddExecution(t *testing.T) {
	mockFlows := mocks.NewMockAuthFlowsClient(t)
	kc := &keycloakapi.KeycloakClient{AuthFlows: mockFlows}

	flow := &keycloakApi.KeycloakAuthFlow{}
	flow.Spec.Alias = testFlowAlias
	flow.Status.ID = testFlowID
	flow.Spec.AuthenticationExecutions = []keycloakApi.AuthenticationExecution{
		{Authenticator: "basic-auth", AuthenticatorFlow: false},
	}

	mockFlows.EXPECT().GetFlowExecutions(context.Background(), testRealmName, testFlowAlias).
		Return([]keycloakapi.AuthenticationExecutionInfoRepresentation{}, nil, nil)

	mockFlows.EXPECT().AddExecutionToFlow(
		context.Background(), testRealmName,
		keycloakapi.AuthenticationExecutionRepresentation{
			Authenticator: ptr.To("basic-auth"),
			ParentFlow:    ptr.To(testFlowID),
			Requirement:   ptr.To(""),
			Priority:      ptr.To(int32(0)),
		},
	).Return(locationResponse(testExecID), nil)

	h := NewSyncAuthFlowExecutions(kc)
	err := h.Serve(context.Background(), flow, testRealmName)
	require.NoError(t, err)
}

func TestSyncAuthFlowExecutions_Serve_AddExecutionWithConfig(t *testing.T) {
	mockFlows := mocks.NewMockAuthFlowsClient(t)
	kc := &keycloakapi.KeycloakClient{AuthFlows: mockFlows}

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
		Return([]keycloakapi.AuthenticationExecutionInfoRepresentation{}, nil, nil)

	mockFlows.EXPECT().AddExecutionToFlow(
		context.Background(), testRealmName,
		keycloakapi.AuthenticationExecutionRepresentation{
			Authenticator: ptr.To("identity-provider-redirector"),
			ParentFlow:    ptr.To(testFlowID),
			Requirement:   ptr.To(""),
			Priority:      ptr.To(int32(0)),
		},
	).Return(locationResponse(testExecID), nil)

	cfg := map[string]string{"defaultProvider": "github"}
	mockFlows.EXPECT().CreateExecutionConfig(
		context.Background(), testRealmName, testExecID,
		keycloakapi.AuthenticatorConfigRepresentation{
			Alias:  ptr.To("idp-config"),
			Config: &cfg,
		},
	).Return(nil, nil)

	h := NewSyncAuthFlowExecutions(kc)
	err := h.Serve(context.Background(), flow, testRealmName)
	require.NoError(t, err)
}

func TestSyncAuthFlowExecutions_Serve_AddExecutionPropagatesPriority(t *testing.T) {
	mockFlows := mocks.NewMockAuthFlowsClient(t)
	kc := &keycloakapi.KeycloakClient{AuthFlows: mockFlows}

	flow := &keycloakApi.KeycloakAuthFlow{}
	flow.Spec.Alias = testFlowAlias
	flow.Status.ID = testFlowID
	flow.Spec.AuthenticationExecutions = []keycloakApi.AuthenticationExecution{
		{Authenticator: "basic-auth", AuthenticatorFlow: false, Priority: 5, Requirement: "REQUIRED"},
	}

	mockFlows.EXPECT().GetFlowExecutions(context.Background(), testRealmName, testFlowAlias).
		Return([]keycloakapi.AuthenticationExecutionInfoRepresentation{}, nil, nil)

	mockFlows.EXPECT().AddExecutionToFlow(
		context.Background(), testRealmName,
		keycloakapi.AuthenticationExecutionRepresentation{
			Authenticator: ptr.To("basic-auth"),
			ParentFlow:    ptr.To(testFlowID),
			Requirement:   ptr.To("REQUIRED"),
			Priority:      ptr.To(int32(5)),
		},
	).Return(locationResponse(testExecID), nil)

	h := NewSyncAuthFlowExecutions(kc)
	err := h.Serve(context.Background(), flow, testRealmName)
	require.NoError(t, err)
}

func TestSyncAuthFlowExecutions_Serve_ForceUpdateOnGenerationChange(t *testing.T) {
	mockFlows := mocks.NewMockAuthFlowsClient(t)
	kc := &keycloakapi.KeycloakClient{AuthFlows: mockFlows}

	const configID = "cfg-5"

	flow := &keycloakApi.KeycloakAuthFlow{}
	flow.Generation = 2
	flow.Status.ObservedGeneration = 1
	setIdpRedirectorSpec(flow)

	existing := nonFlowExec("exec-1", "identity-provider-redirector", "ALTERNATIVE", 10, ptr.To(configID))

	mockFlows.EXPECT().GetFlowExecutions(context.Background(), testRealmName, testFlowAlias).
		Return([]keycloakapi.AuthenticationExecutionInfoRepresentation{existing}, nil, nil)

	// Requirement and priority match spec: exact scalar diffs need no forced write.
	desiredConfig := map[string]string{"defaultProvider": "github"}

	// Forced by generation drift: config updated without a prior read.
	mockFlows.EXPECT().UpdateAuthenticatorConfig(context.Background(), testRealmName, configID,
		keycloakapi.AuthenticatorConfigRepresentation{
			Id:     ptr.To(configID),
			Alias:  ptr.To("idp-config"),
			Config: &desiredConfig,
		},
	).Return(nil, nil)

	h := NewSyncAuthFlowExecutions(kc)
	err := h.Serve(context.Background(), flow, testRealmName)
	require.NoError(t, err)
}

func TestSyncAuthFlowExecutions_Serve_DanglingConfigRecreated(t *testing.T) {
	mockFlows := mocks.NewMockAuthFlowsClient(t)
	kc := &keycloakapi.KeycloakClient{AuthFlows: mockFlows}

	const configID = "cfg-gone"

	flow := &keycloakApi.KeycloakAuthFlow{}
	setIdpRedirectorSpec(flow)

	existing := nonFlowExec("exec-1", "identity-provider-redirector", "ALTERNATIVE", 10, ptr.To(configID))

	mockFlows.EXPECT().GetFlowExecutions(context.Background(), testRealmName, testFlowAlias).
		Return([]keycloakapi.AuthenticationExecutionInfoRepresentation{existing}, nil, nil)

	// Execution references a config that no longer exists: read 404s, update 404s, config is recreated.
	mockFlows.EXPECT().GetAuthenticatorConfig(context.Background(), testRealmName, configID).
		Return(nil, nil, keycloakapi.ErrNotFound)

	mockFlows.EXPECT().UpdateAuthenticatorConfig(context.Background(), testRealmName, configID, mock.Anything).
		Return(nil, keycloakapi.ErrNotFound)

	desiredConfig := map[string]string{"defaultProvider": "github"}

	mockFlows.EXPECT().CreateExecutionConfig(context.Background(), testRealmName, "exec-1",
		keycloakapi.AuthenticatorConfigRepresentation{
			Alias:  ptr.To("idp-config"),
			Config: &desiredConfig,
		},
	).Return(nil, nil)

	h := NewSyncAuthFlowExecutions(kc)
	err := h.Serve(context.Background(), flow, testRealmName)
	require.NoError(t, err)
}

func TestSyncAuthFlowExecutions_Serve_DuplicateAuthenticatorInSync(t *testing.T) {
	mockFlows := mocks.NewMockAuthFlowsClient(t)
	kc := &keycloakapi.KeycloakClient{AuthFlows: mockFlows}

	flow := &keycloakApi.KeycloakAuthFlow{}
	flow.Spec.Alias = testFlowAlias
	flow.Spec.AuthenticationExecutions = []keycloakApi.AuthenticationExecution{
		{Authenticator: "otp-form", Requirement: "REQUIRED", Priority: 10},
		{Authenticator: "otp-form", Requirement: "ALTERNATIVE", Priority: 20},
	}

	existing1 := nonFlowExec("exec-1", "otp-form", "REQUIRED", 10, nil)
	existing2 := nonFlowExec("exec-2", "otp-form", "ALTERNATIVE", 20, nil)

	mockFlows.EXPECT().GetFlowExecutions(context.Background(), testRealmName, testFlowAlias).
		Return([]keycloakapi.AuthenticationExecutionInfoRepresentation{existing1, existing2}, nil, nil)

	h := NewSyncAuthFlowExecutions(kc)
	err := h.Serve(context.Background(), flow, testRealmName)
	require.NoError(t, err)
}

func TestSyncAuthFlowExecutions_Serve_AdjustChildFlowPriority(t *testing.T) {
	mockFlows := mocks.NewMockAuthFlowsClient(t)
	kc := &keycloakapi.KeycloakClient{AuthFlows: mockFlows}

	flow := &keycloakApi.KeycloakAuthFlow{}
	flow.Spec.Alias = testFlowAlias
	flow.Spec.AuthenticationExecutions = []keycloakApi.AuthenticationExecution{
		{Alias: "sub-flow", AuthenticatorFlow: true, Priority: 10, Requirement: "REQUIRED"},
	}

	// Single fetch serves both the non-flow sync and the child-flow priority adjustment.
	mockFlows.EXPECT().GetFlowExecutions(context.Background(), testRealmName, testFlowAlias).
		Return([]keycloakapi.AuthenticationExecutionInfoRepresentation{
			{
				DisplayName:        ptr.To("sub-flow"),
				AuthenticationFlow: ptr.To(true),
				Level:              ptr.To(int32(0)),
				Priority:           ptr.To(int32(20)), // differs from spec (10)
				Requirement:        ptr.To("DISABLED"),
			},
		}, nil, nil).Once()

	updatedExec := keycloakapi.AuthenticationExecutionInfoRepresentation{
		DisplayName:        ptr.To("sub-flow"),
		AuthenticationFlow: ptr.To(true),
		Level:              ptr.To(int32(0)),
		Priority:           ptr.To(int32(10)),
		Requirement:        ptr.To("REQUIRED"),
	}

	mockFlows.EXPECT().UpdateFlowExecution(context.Background(), testRealmName, testFlowAlias, updatedExec).
		Return(nil, nil)

	h := NewSyncAuthFlowExecutions(kc)
	err := h.Serve(context.Background(), flow, testRealmName)
	require.NoError(t, err)
}

func TestSyncAuthFlowExecutions_Serve_GetFlowExecutionsError(t *testing.T) {
	mockFlows := mocks.NewMockAuthFlowsClient(t)
	kc := &keycloakapi.KeycloakClient{AuthFlows: mockFlows}

	flow := &keycloakApi.KeycloakAuthFlow{}
	flow.Spec.Alias = testFlowAlias

	mockFlows.EXPECT().GetFlowExecutions(context.Background(), testRealmName, testFlowAlias).
		Return(nil, nil, errors.New("api error"))

	h := NewSyncAuthFlowExecutions(kc)
	err := h.Serve(context.Background(), flow, testRealmName)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to get flow executions")
}

func TestSyncAuthFlowExecutions_Serve_DeleteExecutionError(t *testing.T) {
	mockFlows := mocks.NewMockAuthFlowsClient(t)
	kc := &keycloakapi.KeycloakClient{AuthFlows: mockFlows}

	flow := &keycloakApi.KeycloakAuthFlow{}
	flow.Spec.Alias = testFlowAlias

	mockFlows.EXPECT().GetFlowExecutions(context.Background(), testRealmName, testFlowAlias).
		Return([]keycloakapi.AuthenticationExecutionInfoRepresentation{
			nonFlowExec("bad-exec", "basic-auth", "REQUIRED", 0, nil),
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
	kc := &keycloakapi.KeycloakClient{AuthFlows: mockFlows}

	flow := &keycloakApi.KeycloakAuthFlow{}
	flow.Spec.Alias = testFlowAlias
	flow.Status.ID = testFlowID
	flow.Spec.AuthenticationExecutions = []keycloakApi.AuthenticationExecution{
		{Authenticator: "basic-auth"},
	}

	mockFlows.EXPECT().GetFlowExecutions(context.Background(), testRealmName, testFlowAlias).
		Return([]keycloakapi.AuthenticationExecutionInfoRepresentation{}, nil, nil)

	mockFlows.EXPECT().AddExecutionToFlow(
		context.Background(), testRealmName,
		keycloakapi.AuthenticationExecutionRepresentation{
			Authenticator: ptr.To("basic-auth"),
			ParentFlow:    ptr.To(testFlowID),
			Requirement:   ptr.To(""),
			Priority:      ptr.To(int32(0)),
		},
	).Return(nil, errors.New("add failed"))

	h := NewSyncAuthFlowExecutions(kc)
	err := h.Serve(context.Background(), flow, testRealmName)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to add execution")
}

func TestSyncAuthFlowExecutions_Serve_EmptyFlowID(t *testing.T) {
	mockFlows := mocks.NewMockAuthFlowsClient(t)
	kc := &keycloakapi.KeycloakClient{AuthFlows: mockFlows}

	flow := &keycloakApi.KeycloakAuthFlow{}
	flow.Spec.Alias = testFlowAlias
	// Status.ID intentionally left empty
	flow.Spec.AuthenticationExecutions = []keycloakApi.AuthenticationExecution{
		{Authenticator: "basic-auth"},
	}

	mockFlows.EXPECT().GetFlowExecutions(context.Background(), testRealmName, testFlowAlias).
		Return([]keycloakapi.AuthenticationExecutionInfoRepresentation{}, nil, nil)

	h := NewSyncAuthFlowExecutions(kc)
	err := h.Serve(context.Background(), flow, testRealmName)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "flow ID is empty")
}

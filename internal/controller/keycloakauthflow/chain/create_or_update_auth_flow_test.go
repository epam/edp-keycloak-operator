package chain

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"k8s.io/utils/ptr"

	keycloakApi "github.com/epam/edp-keycloak-operator/api/v1"
	"github.com/epam/edp-keycloak-operator/pkg/client/keycloakapi"
	"github.com/epam/edp-keycloak-operator/pkg/client/keycloakapi/mocks"
)

const (
	testFlowAlias  = "test-flow"
	testFlowID     = "flow-id-123"
	testParentFlow = "parent-flow"
	testChildAlias = "child-flow"
	testProviderID = "basic-flow"
)

func TestCreateOrUpdateAuthFlow_TopLevel_Create(t *testing.T) {
	mockFlows := mocks.NewMockAuthFlowsClient(t)
	kc := &keycloakapi.KeycloakClient{AuthFlows: mockFlows}

	flow := &keycloakApi.KeycloakAuthFlow{}
	flow.Spec.Alias = testFlowAlias
	flow.Spec.Description = "desc"
	flow.Spec.ProviderID = testProviderID
	flow.Spec.TopLevel = true

	// GetAuthFlows returns empty — flow not found
	mockFlows.EXPECT().GetAuthFlows(context.Background(), testRealmName).
		Return([]keycloakapi.AuthFlowRepresentation{}, nil, nil)

	// CreateAuthFlow called
	mockFlows.EXPECT().CreateAuthFlow(
		context.Background(), testRealmName,
		keycloakapi.AuthFlowRepresentation{
			Alias:       ptr.To(testFlowAlias),
			Description: ptr.To("desc"),
			ProviderId:  ptr.To(testProviderID),
			BuiltIn:     ptr.To(false),
			TopLevel:    ptr.To(true),
		},
	).Return(locationResponse(testFlowID), nil)

	// No child flow-type executions in spec → no validateChildFlows call needed
	h := NewCreateOrUpdateAuthFlow(kc)
	err := h.Serve(context.Background(), flow, testRealmName)
	require.NoError(t, err)
	assert.Equal(t, testFlowID, flow.Status.ID)
}

func TestCreateOrUpdateAuthFlow_TopLevel_AlreadyExists(t *testing.T) {
	mockFlows := mocks.NewMockAuthFlowsClient(t)
	kc := &keycloakapi.KeycloakClient{AuthFlows: mockFlows}

	flow := &keycloakApi.KeycloakAuthFlow{}
	flow.Spec.Alias = testFlowAlias
	flow.Spec.Description = "updated-desc"
	flow.Spec.ProviderID = testProviderID
	flow.Spec.TopLevel = true

	// Flow already exists
	mockFlows.EXPECT().GetAuthFlows(context.Background(), testRealmName).
		Return([]keycloakapi.AuthFlowRepresentation{
			{Alias: ptr.To(testFlowAlias), Id: ptr.To(testFlowID)},
		}, nil, nil)

	// UpdateAuthFlow must be called with updated fields; CreateAuthFlow must NOT be called
	mockFlows.EXPECT().UpdateAuthFlow(
		context.Background(), testRealmName, testFlowID,
		keycloakapi.AuthFlowRepresentation{
			Alias:       ptr.To(testFlowAlias),
			Description: ptr.To("updated-desc"),
			ProviderId:  ptr.To(testProviderID),
			BuiltIn:     ptr.To(false),
			TopLevel:    ptr.To(true),
		},
	).Return(nil, nil)

	h := NewCreateOrUpdateAuthFlow(kc)
	err := h.Serve(context.Background(), flow, testRealmName)
	require.NoError(t, err)
}

func TestCreateOrUpdateAuthFlow_ChildFlow_Create(t *testing.T) {
	mockFlows := mocks.NewMockAuthFlowsClient(t)
	kc := &keycloakapi.KeycloakClient{AuthFlows: mockFlows}

	flow := &keycloakApi.KeycloakAuthFlow{}
	flow.Spec.Alias = testChildAlias
	flow.Spec.ParentName = testParentFlow
	flow.Spec.Description = "child desc"
	flow.Spec.ProviderID = testProviderID
	flow.Spec.ChildType = testProviderID

	// First GetFlowExecutions — child not found
	mockFlows.EXPECT().GetFlowExecutions(context.Background(), testRealmName, testParentFlow).
		Return([]keycloakapi.AuthenticationExecutionInfoRepresentation{}, nil, nil).Once()

	// AddChildFlowToFlow called
	mockFlows.EXPECT().AddChildFlowToFlow(
		context.Background(), testRealmName, testParentFlow,
		map[string]any{
			"alias":       testChildAlias,
			"description": "child desc",
			"provider":    testProviderID,
			"type":        testProviderID,
		},
	).Return(nil, nil)

	// Re-fetch after creation
	mockFlows.EXPECT().GetFlowExecutions(context.Background(), testRealmName, testParentFlow).
		Return([]keycloakapi.AuthenticationExecutionInfoRepresentation{
			{DisplayName: ptr.To(testChildAlias), Id: ptr.To("exec-id")},
		}, nil, nil).Once()

	h := NewCreateOrUpdateAuthFlow(kc)
	err := h.Serve(context.Background(), flow, testRealmName)
	require.NoError(t, err)
}

func TestCreateOrUpdateAuthFlow_ChildFlow_UpdateRequirement(t *testing.T) {
	mockFlows := mocks.NewMockAuthFlowsClient(t)
	kc := &keycloakapi.KeycloakClient{AuthFlows: mockFlows}

	flow := &keycloakApi.KeycloakAuthFlow{}
	flow.Spec.Alias = testChildAlias
	flow.Spec.ParentName = testParentFlow
	flow.Spec.ChildRequirement = "REQUIRED"

	execEntry := keycloakapi.AuthenticationExecutionInfoRepresentation{
		DisplayName: ptr.To(testChildAlias),
		Id:          ptr.To("exec-id"),
		Requirement: ptr.To("DISABLED"),
	}

	// Child already exists
	mockFlows.EXPECT().GetFlowExecutions(context.Background(), testRealmName, testParentFlow).
		Return([]keycloakapi.AuthenticationExecutionInfoRepresentation{execEntry}, nil, nil)

	// Requirement differs — UpdateFlowExecution called
	execEntry.Requirement = ptr.To("REQUIRED")
	mockFlows.EXPECT().UpdateFlowExecution(context.Background(), testRealmName, testParentFlow, execEntry).
		Return(nil, nil)

	h := NewCreateOrUpdateAuthFlow(kc)
	err := h.Serve(context.Background(), flow, testRealmName)
	require.NoError(t, err)
}

func TestCreateOrUpdateAuthFlow_ValidateChildFlows_NotAllCreated(t *testing.T) {
	mockFlows := mocks.NewMockAuthFlowsClient(t)
	kc := &keycloakapi.KeycloakClient{AuthFlows: mockFlows}

	flow := &keycloakApi.KeycloakAuthFlow{}
	flow.Spec.Alias = testFlowAlias
	// Spec has one flow-type execution
	flow.Spec.AuthenticationExecutions = []keycloakApi.AuthenticationExecution{
		{AuthenticatorFlow: true, Alias: "sub-flow"},
	}

	// Flow exists (with ID so UpdateAuthFlow is called)
	mockFlows.EXPECT().GetAuthFlows(context.Background(), testRealmName).
		Return([]keycloakapi.AuthFlowRepresentation{{Alias: ptr.To(testFlowAlias), Id: ptr.To(testFlowID)}}, nil, nil)

	mockFlows.EXPECT().UpdateAuthFlow(
		context.Background(), testRealmName, testFlowID,
		keycloakapi.AuthFlowRepresentation{
			Alias:       ptr.To(testFlowAlias),
			Description: ptr.To(""),
			ProviderId:  ptr.To(""),
			BuiltIn:     ptr.To(false),
			TopLevel:    ptr.To(false),
		},
	).Return(nil, nil)

	// validateChildFlows: GetFlowExecutions returns non-flow execution only → 0 child flows
	mockFlows.EXPECT().GetFlowExecutions(context.Background(), testRealmName, testFlowAlias).
		Return([]keycloakapi.AuthenticationExecutionInfoRepresentation{
			{AuthenticationFlow: ptr.To(false), Level: ptr.To(int32(0))},
		}, nil, nil)

	h := NewCreateOrUpdateAuthFlow(kc)
	err := h.Serve(context.Background(), flow, testRealmName)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not all child flows have been created yet")
}

func TestCreateOrUpdateAuthFlow_ValidateChildFlows_AllCreated(t *testing.T) {
	mockFlows := mocks.NewMockAuthFlowsClient(t)
	kc := &keycloakapi.KeycloakClient{AuthFlows: mockFlows}

	flow := &keycloakApi.KeycloakAuthFlow{}
	flow.Spec.Alias = testFlowAlias
	flow.Spec.AuthenticationExecutions = []keycloakApi.AuthenticationExecution{
		{AuthenticatorFlow: true, Alias: "sub-flow"},
	}

	mockFlows.EXPECT().GetAuthFlows(context.Background(), testRealmName).
		Return([]keycloakapi.AuthFlowRepresentation{{Alias: ptr.To(testFlowAlias), Id: ptr.To(testFlowID)}}, nil, nil)

	mockFlows.EXPECT().UpdateAuthFlow(
		context.Background(), testRealmName, testFlowID,
		keycloakapi.AuthFlowRepresentation{
			Alias:       ptr.To(testFlowAlias),
			Description: ptr.To(""),
			ProviderId:  ptr.To(""),
			BuiltIn:     ptr.To(false),
			TopLevel:    ptr.To(false),
		},
	).Return(nil, nil)

	// validateChildFlows: one AuthenticationFlow=true, Level=0 execution → 1 child
	mockFlows.EXPECT().GetFlowExecutions(context.Background(), testRealmName, testFlowAlias).
		Return([]keycloakapi.AuthenticationExecutionInfoRepresentation{
			{AuthenticationFlow: ptr.To(true), Level: ptr.To(int32(0))},
		}, nil, nil)

	h := NewCreateOrUpdateAuthFlow(kc)
	err := h.Serve(context.Background(), flow, testRealmName)
	require.NoError(t, err)
}

func TestCreateOrUpdateAuthFlow_GetAuthFlowsError(t *testing.T) {
	mockFlows := mocks.NewMockAuthFlowsClient(t)
	kc := &keycloakapi.KeycloakClient{AuthFlows: mockFlows}

	flow := &keycloakApi.KeycloakAuthFlow{}
	flow.Spec.Alias = testFlowAlias

	mockFlows.EXPECT().GetAuthFlows(context.Background(), testRealmName).
		Return(nil, nil, errors.New("api error"))

	h := NewCreateOrUpdateAuthFlow(kc)
	err := h.Serve(context.Background(), flow, testRealmName)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to get auth flows")
}

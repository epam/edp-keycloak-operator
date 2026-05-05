package keycloakapi_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/epam/edp-keycloak-operator/pkg/client/keycloakapi"
	"github.com/epam/edp-keycloak-operator/pkg/testutils"
)

const basicFlowProviderID = "basic-flow"
const conditionalOTPFormAuthenticator = "auth-conditional-otp-form"

func TestAuthFlowsClient_CRUD(t *testing.T) {
	keycloakURL := testutils.GetKeycloakURLOrSkip(t)
	t.Parallel()

	c, err := keycloakapi.NewKeycloakClient(
		context.Background(),
		keycloakURL,
		keycloakapi.DefaultAdminClientID,
		keycloakapi.WithPasswordGrant(keycloakapi.DefaultAdminUsername, keycloakapi.DefaultAdminPassword),
	)
	require.NoError(t, err)

	ctx := context.Background()
	realmName := fmt.Sprintf("test-realm-af-crud-%d", time.Now().UnixNano())
	testutils.CreateRealmWithRetry(t, c, realmName)

	flowAlias := fmt.Sprintf("test-flow-%d", time.Now().UnixNano())
	topLevel := true
	builtIn := false
	providerID := basicFlowProviderID
	description := "test flow description"

	// 1. Create auth flow
	resp, err := c.AuthFlows.CreateAuthFlow(ctx, realmName, keycloakapi.AuthFlowRepresentation{
		Alias:       &flowAlias,
		TopLevel:    &topLevel,
		BuiltIn:     &builtIn,
		ProviderId:  &providerID,
		Description: &description,
	})
	require.NoError(t, err)
	require.NotNil(t, resp)

	flowID := keycloakapi.GetResourceIDFromResponse(resp)
	require.NotEmpty(t, flowID, "flow ID should be extracted from Location header")

	// 2. Get all flows and verify ours is present
	flows, _, err := c.AuthFlows.GetAuthFlows(ctx, realmName)
	require.NoError(t, err)

	found := false

	for _, f := range flows {
		if f.Id != nil && *f.Id == flowID {
			found = true

			require.Equal(t, flowAlias, *f.Alias)
			require.Equal(t, description, *f.Description)

			break
		}
	}

	require.True(t, found, "created flow should appear in GetAuthFlows")

	// 3. Update auth flow
	updatedDesc := "updated flow description"
	_, err = c.AuthFlows.UpdateAuthFlow(ctx, realmName, flowID, keycloakapi.AuthFlowRepresentation{
		Alias:       &flowAlias,
		TopLevel:    &topLevel,
		BuiltIn:     &builtIn,
		ProviderId:  &providerID,
		Description: &updatedDesc,
	})
	require.NoError(t, err)

	// Verify description was updated
	flows, _, err = c.AuthFlows.GetAuthFlows(ctx, realmName)
	require.NoError(t, err)

	for _, f := range flows {
		if f.Id != nil && *f.Id == flowID {
			require.Equal(t, updatedDesc, *f.Description)

			break
		}
	}

	// 4. Delete auth flow
	_, err = c.AuthFlows.DeleteAuthFlow(ctx, realmName, flowID)
	require.NoError(t, err)

	// 5. Verify deletion
	flows, _, err = c.AuthFlows.GetAuthFlows(ctx, realmName)
	require.NoError(t, err)

	for _, f := range flows {
		if f.Id != nil && *f.Id == flowID {
			t.Fatal("deleted flow should not appear in GetAuthFlows")
		}
	}
}

func TestAuthFlowsClient_Executions(t *testing.T) {
	keycloakURL := testutils.GetKeycloakURLOrSkip(t)
	t.Parallel()

	c, err := keycloakapi.NewKeycloakClient(
		context.Background(),
		keycloakURL,
		keycloakapi.DefaultAdminClientID,
		keycloakapi.WithPasswordGrant(keycloakapi.DefaultAdminUsername, keycloakapi.DefaultAdminPassword),
	)
	require.NoError(t, err)

	ctx := context.Background()
	realmName := fmt.Sprintf("test-realm-af-exec-%d", time.Now().UnixNano())
	testutils.CreateRealmWithRetry(t, c, realmName)

	// Create a custom flow to add executions to
	flowAlias := fmt.Sprintf("test-flow-exec-%d", time.Now().UnixNano())
	topLevel := true
	builtIn := false
	providerID := basicFlowProviderID

	resp, err := c.AuthFlows.CreateAuthFlow(ctx, realmName, keycloakapi.AuthFlowRepresentation{
		Alias:      &flowAlias,
		TopLevel:   &topLevel,
		BuiltIn:    &builtIn,
		ProviderId: &providerID,
	})
	require.NoError(t, err)

	flowID := keycloakapi.GetResourceIDFromResponse(resp)
	require.NotEmpty(t, flowID)

	// 1. Add execution to flow
	authenticator := "deny-access-authenticator"
	requirement := keycloakapi.AuthExecutionRequirementDisabled
	resp, err = c.AuthFlows.AddExecutionToFlow(ctx, realmName, keycloakapi.AuthenticationExecutionRepresentation{
		ParentFlow:    &flowID,
		Authenticator: &authenticator,
		Requirement:   &requirement,
	})
	require.NoError(t, err)
	require.NotNil(t, resp)

	executionID := keycloakapi.GetResourceIDFromResponse(resp)
	require.NotEmpty(t, executionID, "execution ID should be extracted from Location header")

	// 2. Get flow executions and verify ours is present
	executions, _, err := c.AuthFlows.GetFlowExecutions(ctx, realmName, flowAlias)
	require.NoError(t, err)
	require.NotEmpty(t, executions)

	var targetExec keycloakapi.AuthenticationExecutionInfoRepresentation

	found := false

	for _, exec := range executions {
		if exec.Id != nil && *exec.Id == executionID {
			targetExec = exec
			found = true

			break
		}
	}

	require.True(t, found, "added execution should appear in GetFlowExecutions")

	// 3. Update execution requirement
	requirement = keycloakapi.AuthExecutionRequirementRequired
	targetExec.Requirement = &requirement

	_, err = c.AuthFlows.UpdateFlowExecution(ctx, realmName, flowAlias, targetExec)
	require.NoError(t, err)

	// Verify requirement was updated
	executions, _, err = c.AuthFlows.GetFlowExecutions(ctx, realmName, flowAlias)
	require.NoError(t, err)

	for _, exec := range executions {
		if exec.Id != nil && *exec.Id == executionID {
			require.Equal(t, requirement, *exec.Requirement)

			break
		}
	}

	// 4. Delete execution
	_, err = c.AuthFlows.DeleteExecution(ctx, realmName, executionID)
	require.NoError(t, err)

	// 5. Verify execution was deleted
	executions, _, err = c.AuthFlows.GetFlowExecutions(ctx, realmName, flowAlias)
	require.NoError(t, err)

	for _, exec := range executions {
		if exec.Id != nil && *exec.Id == executionID {
			t.Fatal("deleted execution should not appear in GetFlowExecutions")
		}
	}
}

func TestAuthFlowsClient_ChildFlow(t *testing.T) {
	keycloakURL := testutils.GetKeycloakURLOrSkip(t)
	t.Parallel()

	c, err := keycloakapi.NewKeycloakClient(
		context.Background(),
		keycloakURL,
		keycloakapi.DefaultAdminClientID,
		keycloakapi.WithPasswordGrant(keycloakapi.DefaultAdminUsername, keycloakapi.DefaultAdminPassword),
	)
	require.NoError(t, err)

	ctx := context.Background()
	realmName := fmt.Sprintf("test-realm-af-child-%d", time.Now().UnixNano())
	testutils.CreateRealmWithRetry(t, c, realmName)

	// Create parent flow
	flowAlias := fmt.Sprintf("test-parent-flow-%d", time.Now().UnixNano())
	topLevel := true
	builtIn := false
	providerID := basicFlowProviderID

	resp, err := c.AuthFlows.CreateAuthFlow(ctx, realmName, keycloakapi.AuthFlowRepresentation{
		Alias:      &flowAlias,
		TopLevel:   &topLevel,
		BuiltIn:    &builtIn,
		ProviderId: &providerID,
	})
	require.NoError(t, err)
	require.NotNil(t, resp)

	childAlias := "child-sub-flow"

	// Add child sub-flow to the parent flow
	resp, err = c.AuthFlows.AddChildFlowToFlow(ctx, realmName, flowAlias, map[string]any{
		"alias":       childAlias,
		"type":        basicFlowProviderID,
		"description": "child sub-flow for testing",
		"provider":    "registration-page-form",
	})
	require.NoError(t, err)
	require.NotNil(t, resp)

	// Verify child flow appears in parent's executions.
	// Keycloak sets the child flow name as DisplayName (not Alias) on execution info.
	executions, _, err := c.AuthFlows.GetFlowExecutions(ctx, realmName, flowAlias)
	require.NoError(t, err)
	require.NotEmpty(t, executions)

	found := false

	for _, exec := range executions {
		if exec.DisplayName != nil && *exec.DisplayName == childAlias {
			found = true

			break
		}
	}

	require.True(t, found, "child sub-flow should appear in parent flow executions")
}

func TestAuthFlowsClient_ExecutionConfig(t *testing.T) {
	keycloakURL := testutils.GetKeycloakURLOrSkip(t)
	t.Parallel()

	c, err := keycloakapi.NewKeycloakClient(
		context.Background(),
		keycloakURL,
		keycloakapi.DefaultAdminClientID,
		keycloakapi.WithPasswordGrant(keycloakapi.DefaultAdminUsername, keycloakapi.DefaultAdminPassword),
	)
	require.NoError(t, err)

	ctx := context.Background()
	realmName := fmt.Sprintf("test-realm-af-cfg-%d", time.Now().UnixNano())
	testutils.CreateRealmWithRetry(t, c, realmName)

	// Create a flow
	flowAlias := fmt.Sprintf("test-flow-cfg-%d", time.Now().UnixNano())
	topLevel := true
	builtIn := false
	providerID := basicFlowProviderID

	resp, err := c.AuthFlows.CreateAuthFlow(ctx, realmName, keycloakapi.AuthFlowRepresentation{
		Alias:      &flowAlias,
		TopLevel:   &topLevel,
		BuiltIn:    &builtIn,
		ProviderId: &providerID,
	})
	require.NoError(t, err)

	flowID := keycloakapi.GetResourceIDFromResponse(resp)
	require.NotEmpty(t, flowID)

	// Add a configurable execution (conditionalOTPFormAuthenticator is configurable)
	authenticator := conditionalOTPFormAuthenticator
	requirement := keycloakapi.AuthExecutionRequirementDisabled
	resp, err = c.AuthFlows.AddExecutionToFlow(ctx, realmName, keycloakapi.AuthenticationExecutionRepresentation{
		ParentFlow:    &flowID,
		Authenticator: &authenticator,
		Requirement:   &requirement,
	})
	require.NoError(t, err)

	executionID := keycloakapi.GetResourceIDFromResponse(resp)
	require.NotEmpty(t, executionID)

	// Create authenticator config for the execution
	configAlias := "test-otp-config"
	configValues := map[string]string{
		"defaultOtpOutcome":   "skip",
		"otpControlAttribute": "skip_otp",
		"noDeviceAction":      "force",
	}

	config := keycloakapi.AuthenticatorConfigRepresentation{
		Alias:  &configAlias,
		Config: &configValues,
	}
	resp, err = c.AuthFlows.CreateExecutionConfig(ctx, realmName, executionID, config)
	require.NoError(t, err)
	require.NotNil(t, resp)

	configID := keycloakapi.GetResourceIDFromResponse(resp)
	require.NotEmpty(t, configID, "config ID should be extracted from Location header")

	// Verify config was associated with the execution
	executions, _, err := c.AuthFlows.GetFlowExecutions(ctx, realmName, flowAlias)
	require.NoError(t, err)

	for _, exec := range executions {
		if exec.Id != nil && *exec.Id == executionID {
			require.NotNil(t, exec.AuthenticationConfig, "execution should have authentication config set")

			break
		}
	}
}

func TestAuthFlowsClient_GetAuthenticatorConfig(t *testing.T) {
	keycloakURL := testutils.GetKeycloakURLOrSkip(t)
	t.Parallel()

	c, err := keycloakapi.NewKeycloakClient(
		context.Background(),
		keycloakURL,
		keycloakapi.DefaultAdminClientID,
		keycloakapi.WithPasswordGrant(keycloakapi.DefaultAdminUsername, keycloakapi.DefaultAdminPassword),
	)
	require.NoError(t, err)

	ctx := context.Background()
	realmName := fmt.Sprintf("test-realm-af-getcfg-%d", time.Now().UnixNano())
	testutils.CreateRealmWithRetry(t, c, realmName)

	// Create a flow and add a configurable execution.
	flowAlias := fmt.Sprintf("test-flow-getcfg-%d", time.Now().UnixNano())
	topLevel := true
	builtIn := false
	providerID := basicFlowProviderID

	resp, err := c.AuthFlows.CreateAuthFlow(ctx, realmName, keycloakapi.AuthFlowRepresentation{
		Alias:      &flowAlias,
		TopLevel:   &topLevel,
		BuiltIn:    &builtIn,
		ProviderId: &providerID,
	})
	require.NoError(t, err)

	flowID := keycloakapi.GetResourceIDFromResponse(resp)
	require.NotEmpty(t, flowID)

	authenticator := conditionalOTPFormAuthenticator
	requirement := keycloakapi.AuthExecutionRequirementDisabled

	resp, err = c.AuthFlows.AddExecutionToFlow(ctx, realmName, keycloakapi.AuthenticationExecutionRepresentation{
		ParentFlow:    &flowID,
		Authenticator: &authenticator,
		Requirement:   &requirement,
	})
	require.NoError(t, err)

	executionID := keycloakapi.GetResourceIDFromResponse(resp)
	require.NotEmpty(t, executionID)

	// Create an authenticator config for the execution.
	configAlias := "test-get-cfg"
	configValues := map[string]string{
		"defaultOtpOutcome":   "skip",
		"otpControlAttribute": "skip_otp",
	}

	resp, err = c.AuthFlows.CreateExecutionConfig(
		ctx,
		realmName,
		executionID,
		keycloakapi.AuthenticatorConfigRepresentation{
			Alias:  &configAlias,
			Config: &configValues,
		},
	)
	require.NoError(t, err)

	configID := keycloakapi.GetResourceIDFromResponse(resp)
	require.NotEmpty(t, configID, "config ID should be extracted from Location header")

	// GetAuthenticatorConfig — happy path.
	gotConfig, gotResp, err := c.AuthFlows.GetAuthenticatorConfig(ctx, realmName, configID)
	require.NoError(t, err)
	require.NotNil(t, gotResp)
	require.NotNil(t, gotConfig)
	require.NotNil(t, gotConfig.Alias)
	require.Equal(t, configAlias, *gotConfig.Alias)
	require.NotNil(t, gotConfig.Config)
	require.Equal(t, configValues, *gotConfig.Config)

	// GetAuthenticatorConfig — not-found path.
	_, _, err = c.AuthFlows.GetAuthenticatorConfig(ctx, realmName, "non-existent-config-id")
	require.Error(t, err)
}

func TestAuthFlowsClient_GetAuthFlow(t *testing.T) {
	keycloakURL := testutils.GetKeycloakURLOrSkip(t)
	t.Parallel()

	c, err := keycloakapi.NewKeycloakClient(
		context.Background(),
		keycloakURL,
		keycloakapi.DefaultAdminClientID,
		keycloakapi.WithPasswordGrant(keycloakapi.DefaultAdminUsername, keycloakapi.DefaultAdminPassword),
	)
	require.NoError(t, err)

	ctx := context.Background()
	realmName := fmt.Sprintf("test-realm-af-getflow-%d", time.Now().UnixNano())
	testutils.CreateRealmWithRetry(t, c, realmName)

	// Create a flow and then fetch it by ID.
	flowAlias := fmt.Sprintf("test-get-flow-%d", time.Now().UnixNano())
	topLevel := true
	builtIn := false
	providerID := basicFlowProviderID

	resp, err := c.AuthFlows.CreateAuthFlow(ctx, realmName, keycloakapi.AuthFlowRepresentation{
		Alias:      &flowAlias,
		TopLevel:   &topLevel,
		BuiltIn:    &builtIn,
		ProviderId: &providerID,
	})
	require.NoError(t, err)

	flowID := keycloakapi.GetResourceIDFromResponse(resp)
	require.NotEmpty(t, flowID)

	flow, resp, err := c.AuthFlows.GetAuthFlow(ctx, realmName, flowID)
	require.NoError(t, err)
	require.NotNil(t, resp)
	require.NotNil(t, flow)
	require.Equal(t, flowAlias, *flow.Alias)

	// Not found.
	_, _, err = c.AuthFlows.GetAuthFlow(ctx, realmName, "non-existent-flow-id")
	require.Error(t, err)
	require.True(t, keycloakapi.IsNotFound(err))
}

func TestAuthFlowsClient_CopyAuthFlow(t *testing.T) {
	keycloakURL := testutils.GetKeycloakURLOrSkip(t)
	t.Parallel()

	c, err := keycloakapi.NewKeycloakClient(
		context.Background(),
		keycloakURL,
		keycloakapi.DefaultAdminClientID,
		keycloakapi.WithPasswordGrant(keycloakapi.DefaultAdminUsername, keycloakapi.DefaultAdminPassword),
	)
	require.NoError(t, err)

	ctx := context.Background()
	realmName := fmt.Sprintf("test-realm-af-copy-%d", time.Now().UnixNano())
	testutils.CreateRealmWithRetry(t, c, realmName)

	newFlowAlias := fmt.Sprintf("copy-of-browser-%d", time.Now().UnixNano())

	resp, err := c.AuthFlows.CopyAuthFlow(ctx, realmName, "browser", newFlowAlias)
	require.NoError(t, err)
	require.NotNil(t, resp)

	// Verify the copy exists by listing flows.
	flows, _, err := c.AuthFlows.GetAuthFlows(ctx, realmName)
	require.NoError(t, err)

	found := false

	for _, f := range flows {
		if f.Alias != nil && *f.Alias == newFlowAlias {
			found = true

			break
		}
	}

	require.True(t, found, "copied flow should appear in the list")
}

func TestAuthFlowsClient_UpdateAndDeleteAuthenticatorConfig(t *testing.T) {
	keycloakURL := testutils.GetKeycloakURLOrSkip(t)
	t.Parallel()

	c, err := keycloakapi.NewKeycloakClient(
		context.Background(),
		keycloakURL,
		keycloakapi.DefaultAdminClientID,
		keycloakapi.WithPasswordGrant(keycloakapi.DefaultAdminUsername, keycloakapi.DefaultAdminPassword),
	)
	require.NoError(t, err)

	ctx := context.Background()
	realmName := fmt.Sprintf("test-realm-af-updcfg-%d", time.Now().UnixNano())
	testutils.CreateRealmWithRetry(t, c, realmName)

	// Create a flow with an execution that supports config (conditional-otp).
	flowAlias := fmt.Sprintf("cfg-flow-%d", time.Now().UnixNano())
	topLevel := true
	builtIn := false
	providerID := basicFlowProviderID

	_, err = c.AuthFlows.CreateAuthFlow(ctx, realmName, keycloakapi.AuthFlowRepresentation{
		Alias:      &flowAlias,
		TopLevel:   &topLevel,
		BuiltIn:    &builtIn,
		ProviderId: &providerID,
	})
	require.NoError(t, err)

	// Get the flow ID by listing flows.
	flows, _, err := c.AuthFlows.GetAuthFlows(ctx, realmName)
	require.NoError(t, err)

	var flowID string

	for _, f := range flows {
		if f.Alias != nil && *f.Alias == flowAlias {
			flowID = *f.Id

			break
		}
	}

	require.NotEmpty(t, flowID)

	execProvider := conditionalOTPFormAuthenticator
	requirement := keycloakapi.AuthExecutionRequirementRequired

	execResp, err := c.AuthFlows.AddExecutionToFlow(ctx, realmName, keycloakapi.AuthenticationExecutionRepresentation{
		ParentFlow:    &flowID,
		Authenticator: &execProvider,
		Requirement:   &requirement,
	})
	require.NoError(t, err)

	executionID := keycloakapi.GetResourceIDFromResponse(execResp)

	configAlias := "test-upd-cfg"
	configValues := map[string]string{
		"defaultOtpOutcome":   "skip",
		"otpControlAttribute": "skip_otp",
	}

	cfgResp, err := c.AuthFlows.CreateExecutionConfig(ctx, realmName, executionID,
		keycloakapi.AuthenticatorConfigRepresentation{
			Alias:  &configAlias,
			Config: &configValues,
		},
	)
	require.NoError(t, err)

	configID := keycloakapi.GetResourceIDFromResponse(cfgResp)
	require.NotEmpty(t, configID)

	// Update the config.
	updatedValues := map[string]string{
		"defaultOtpOutcome":   "force",
		"otpControlAttribute": "force_otp",
	}

	resp, err := c.AuthFlows.UpdateAuthenticatorConfig(ctx, realmName, configID,
		keycloakapi.AuthenticatorConfigRepresentation{
			Id:     &configID,
			Alias:  &configAlias,
			Config: &updatedValues,
		},
	)
	require.NoError(t, err)
	require.NotNil(t, resp)

	// Verify update.
	gotCfg, _, err := c.AuthFlows.GetAuthenticatorConfig(ctx, realmName, configID)
	require.NoError(t, err)
	require.NotNil(t, gotCfg.Config)
	require.Equal(t, "force", (*gotCfg.Config)["defaultOtpOutcome"])

	// Delete the config.
	resp, err = c.AuthFlows.DeleteAuthenticatorConfig(ctx, realmName, configID)
	require.NoError(t, err)
	require.NotNil(t, resp)

	// Verify deletion.
	_, _, err = c.AuthFlows.GetAuthenticatorConfig(ctx, realmName, configID)
	require.Error(t, err)
}

func TestAuthFlowsClient_RequiredActions(t *testing.T) {
	keycloakURL := testutils.GetKeycloakURLOrSkip(t)
	t.Parallel()

	c, err := keycloakapi.NewKeycloakClient(
		context.Background(),
		keycloakURL,
		keycloakapi.DefaultAdminClientID,
		keycloakapi.WithPasswordGrant(keycloakapi.DefaultAdminUsername, keycloakapi.DefaultAdminPassword),
	)
	require.NoError(t, err)

	ctx := context.Background()
	realmName := fmt.Sprintf("test-realm-af-reqaction-%d", time.Now().UnixNano())
	testutils.CreateRealmWithRetry(t, c, realmName)

	// Get required actions — every realm has built-in required actions.
	actions, resp, err := c.AuthFlows.GetRequiredActions(ctx, realmName)
	require.NoError(t, err)
	require.NotNil(t, resp)
	require.Greater(t, len(actions), 0, "realm should have at least one required action")

	// Find VERIFY_EMAIL action (always present).
	var verifyEmail *keycloakapi.RequiredActionProviderRepresentation

	for i := range actions {
		if actions[i].Alias != nil && *actions[i].Alias == "VERIFY_EMAIL" {
			verifyEmail = &actions[i]

			break
		}
	}

	require.NotNil(t, verifyEmail, "VERIFY_EMAIL required action should exist")

	// Update the required action (toggle enabled state).
	originalEnabled := *verifyEmail.Enabled
	verifyEmail.Enabled = &originalEnabled // no-op update to verify API works

	resp, err = c.AuthFlows.UpdateRequiredAction(ctx, realmName, "VERIFY_EMAIL", *verifyEmail)
	require.NoError(t, err)
	require.NotNil(t, resp)
}

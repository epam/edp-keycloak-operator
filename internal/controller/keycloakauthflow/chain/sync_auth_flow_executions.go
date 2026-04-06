package chain

import (
	"context"
	"fmt"
	"sort"

	ctrl "sigs.k8s.io/controller-runtime"

	keycloakApi "github.com/epam/edp-keycloak-operator/api/v1"
	keycloakv2 "github.com/epam/edp-keycloak-operator/pkg/client/keycloakv2"
)

// SyncAuthFlowExecutions syncs authentication executions for an auth flow.
// It clears all existing non-flow executions and recreates them from the spec.
// This ports the execution-sync logic from the legacy gocloak adapter.
type SyncAuthFlowExecutions struct {
	kClientV2 *keycloakv2.KeycloakClient
}

func NewSyncAuthFlowExecutions(kClientV2 *keycloakv2.KeycloakClient) *SyncAuthFlowExecutions {
	return &SyncAuthFlowExecutions{kClientV2: kClientV2}
}

func (h *SyncAuthFlowExecutions) Serve(ctx context.Context, flow *keycloakApi.KeycloakAuthFlow, realmName string) error {
	log := ctrl.LoggerFrom(ctx)
	log.Info("Syncing auth flow executions", "alias", flow.Spec.Alias)

	if err := h.clearNonFlowExecutions(ctx, flow.Spec.Alias, realmName); err != nil {
		return fmt.Errorf("failed to clear flow executions: %w", err)
	}

	// Collect only non-flow executions from spec (flow-type executions are managed
	// by separate child KeycloakAuthFlow resources).
	var execsToAdd []keycloakApi.AuthenticationExecution

	for _, e := range flow.Spec.AuthenticationExecutions {
		if !e.AuthenticatorFlow {
			execsToAdd = append(execsToAdd, e)
		}
	}

	if len(execsToAdd) == 0 {
		log.Info("No non-flow executions to sync")

		return h.adjustChildFlowsPriority(ctx, flow, realmName)
	}

	// The flow's internal Keycloak ID is set by the preceding CreateOrUpdateAuthFlow chain step.
	flowID := flow.Status.ID
	if flowID == "" {
		return fmt.Errorf("flow ID is empty for alias %q; ensure CreateOrUpdateAuthFlow ran first", flow.Spec.Alias)
	}

	// Sort by priority before adding (mirrors legacy adapter behaviour).
	sort.Slice(execsToAdd, func(i, j int) bool {
		return execsToAdd[i].Priority < execsToAdd[j].Priority
	})

	for i := range execsToAdd {
		e := &execsToAdd[i]

		execID, err := h.addExecution(ctx, realmName, flowID, e.Authenticator, e.Requirement)
		if err != nil {
			return fmt.Errorf("failed to add execution %q: %w", e.Authenticator, err)
		}

		if e.AuthenticatorConfig != nil {
			if err := h.createExecutionConfig(ctx, realmName, execID, e.AuthenticatorConfig); err != nil {
				return fmt.Errorf("failed to create config for execution %q: %w", e.Authenticator, err)
			}
		}
	}

	return h.adjustChildFlowsPriority(ctx, flow, realmName)
}

// clearNonFlowExecutions deletes all top-level non-flow executions from the flow.
func (h *SyncAuthFlowExecutions) clearNonFlowExecutions(ctx context.Context, flowAlias, realmName string) error {
	execs, _, err := h.kClientV2.AuthFlows.GetFlowExecutions(ctx, realmName, flowAlias)
	if err != nil {
		return fmt.Errorf("failed to get flow executions: %w", err)
	}

	for i := range execs {
		e := &execs[i]

		isFlowExec := e.AuthenticationFlow != nil && *e.AuthenticationFlow
		isTopLevel := e.Level != nil && *e.Level == 0

		if isFlowExec || !isTopLevel {
			continue
		}

		execID := ""
		if e.Id != nil {
			execID = *e.Id
		}

		if execID == "" {
			continue
		}

		if _, err := h.kClientV2.AuthFlows.DeleteExecution(ctx, realmName, execID); err != nil {
			return fmt.Errorf("failed to delete execution %q: %w", execID, err)
		}
	}

	return nil
}

// addExecution posts a new execution to the flow and returns the new execution ID from the Location header.
func (h *SyncAuthFlowExecutions) addExecution(ctx context.Context, realmName, flowID, authenticator, requirement string) (string, error) {
	resp, err := h.kClientV2.AuthFlows.AddExecutionToFlow(ctx, realmName, keycloakv2.AuthenticationExecutionRepresentation{
		Authenticator: &authenticator,
		ParentFlow:    &flowID,
		Requirement:   &requirement,
	})
	if err != nil {
		return "", fmt.Errorf("failed to post execution: %w", err)
	}

	execID := keycloakv2.GetResourceIDFromResponse(resp)
	if execID == "" {
		return "", fmt.Errorf("execution Location header missing or empty for authenticator %q", authenticator)
	}

	return execID, nil
}

func (h *SyncAuthFlowExecutions) createExecutionConfig(
	ctx context.Context,
	realmName, execID string,
	cfg *keycloakApi.AuthenticatorConfig,
) error {
	_, err := h.kClientV2.AuthFlows.CreateExecutionConfig(ctx, realmName, execID, keycloakv2.AuthenticatorConfigRepresentation{
		Alias:  &cfg.Alias,
		Config: &cfg.Config,
	})
	if err != nil {
		return fmt.Errorf("failed to create execution config: %w", err)
	}

	return nil
}

// adjustChildFlowsPriority updates priority (and requirement) of flow-type executions
// to match the spec. Ports adjustChildFlowsPriority from the legacy adapter.
func (h *SyncAuthFlowExecutions) adjustChildFlowsPriority(ctx context.Context, flow *keycloakApi.KeycloakAuthFlow, realmName string) error {
	// Build a map of alias -> spec execution for flow-type entries.
	childFlowSpecs := make(map[string]keycloakApi.AuthenticationExecution)

	for _, e := range flow.Spec.AuthenticationExecutions {
		if e.AuthenticatorFlow {
			childFlowSpecs[e.Alias] = e
		}
	}

	if len(childFlowSpecs) == 0 {
		return nil
	}

	execs, _, err := h.kClientV2.AuthFlows.GetFlowExecutions(ctx, realmName, flow.Spec.Alias)
	if err != nil {
		return fmt.Errorf("failed to get flow executions for priority adjustment: %w", err)
	}

	for i := range execs {
		e := &execs[i]

		isFlowExec := e.AuthenticationFlow != nil && *e.AuthenticationFlow
		isTopLevel := e.Level != nil && *e.Level == 0

		if !isFlowExec || !isTopLevel {
			continue
		}

		if e.DisplayName == nil {
			continue
		}

		specEntry, ok := childFlowSpecs[*e.DisplayName]
		if !ok {
			continue
		}

		needsUpdate := false

		if specEntry.Requirement != "" && (e.Requirement == nil || *e.Requirement != specEntry.Requirement) {
			e.Requirement = &specEntry.Requirement
			needsUpdate = true
		}

		expectedPriority := int32(specEntry.Priority)
		if e.Priority == nil || *e.Priority != expectedPriority {
			e.Priority = &expectedPriority
			needsUpdate = true
		}

		if needsUpdate {
			if _, err := h.kClientV2.AuthFlows.UpdateFlowExecution(ctx, realmName, flow.Spec.Alias, *e); err != nil {
				return fmt.Errorf("failed to update priority for child flow %q: %w", *e.DisplayName, err)
			}
		}
	}

	return nil
}

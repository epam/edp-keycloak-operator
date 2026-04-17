package chain

import (
	"context"
	"errors"
	"fmt"

	"k8s.io/utils/ptr"
	ctrl "sigs.k8s.io/controller-runtime"

	keycloakApi "github.com/epam/edp-keycloak-operator/api/v1"
	keycloakapi "github.com/epam/edp-keycloak-operator/pkg/client/keycloakapi"
)

// CreateOrUpdateAuthFlow creates or ensures the base auth flow exists in Keycloak.
// It ports the logic of syncBaseAuthFlow from the legacy gocloak adapter.
type CreateOrUpdateAuthFlow struct {
	kClientV2 *keycloakapi.APIClient
}

func NewCreateOrUpdateAuthFlow(kClientV2 *keycloakapi.APIClient) *CreateOrUpdateAuthFlow {
	return &CreateOrUpdateAuthFlow{kClientV2: kClientV2}
}

func (h *CreateOrUpdateAuthFlow) Serve(ctx context.Context, flow *keycloakApi.KeycloakAuthFlow, realmName string) error {
	log := ctrl.LoggerFrom(ctx)
	log.Info("Creating or updating auth flow")

	spec := flow.Spec

	if spec.ParentName != "" {
		return h.serveChildFlow(ctx, flow, realmName)
	}

	return h.serveTopLevelFlow(ctx, flow, realmName)
}

func (h *CreateOrUpdateAuthFlow) serveTopLevelFlow(ctx context.Context, flow *keycloakApi.KeycloakAuthFlow, realmName string) error {
	log := ctrl.LoggerFrom(ctx)

	flows, _, err := h.kClientV2.AuthFlows.GetAuthFlows(ctx, realmName)
	if err != nil {
		return fmt.Errorf("failed to get auth flows: %w", err)
	}

	for i := range flows {
		if flows[i].Alias != nil && *flows[i].Alias == flow.Spec.Alias {
			if flows[i].Id == nil {
				return fmt.Errorf("auth flow %q has no ID", flow.Spec.Alias)
			}

			log.Info("Top-level auth flow already exists, updating", "alias", flow.Spec.Alias)

			flow.Status.ID = *flows[i].Id

			if _, err = h.kClientV2.AuthFlows.UpdateAuthFlow(ctx, realmName, *flows[i].Id, authFlowRepFromSpec(flow.Spec)); err != nil {
				return fmt.Errorf("failed to update auth flow: %w", err)
			}

			return h.validateChildFlows(ctx, flow, realmName)
		}
	}

	log.Info("Creating top-level auth flow", "alias", flow.Spec.Alias)

	resp, err := h.kClientV2.AuthFlows.CreateAuthFlow(ctx, realmName, authFlowRepFromSpec(flow.Spec))
	if err != nil {
		return fmt.Errorf("failed to create auth flow: %w", err)
	}

	flow.Status.ID = keycloakapi.GetResourceIDFromResponse(resp)
	if flow.Status.ID == "" {
		return fmt.Errorf("auth flow Location header missing or empty for alias %q", flow.Spec.Alias)
	}

	log.Info("Top-level auth flow created")

	return h.validateChildFlows(ctx, flow, realmName)
}

func (h *CreateOrUpdateAuthFlow) serveChildFlow(ctx context.Context, flow *keycloakApi.KeycloakAuthFlow, realmName string) error {
	log := ctrl.LoggerFrom(ctx)

	execs, _, err := h.kClientV2.AuthFlows.GetFlowExecutions(ctx, realmName, flow.Spec.ParentName)
	if err != nil {
		return fmt.Errorf("failed to get parent flow executions: %w", err)
	}

	existing := findExecByDisplayName(execs, flow.Spec.Alias)

	if existing == nil {
		log.Info("Creating child auth flow", "alias", flow.Spec.Alias, "parent", flow.Spec.ParentName)

		_, err = h.kClientV2.AuthFlows.AddChildFlowToFlow(ctx, realmName, flow.Spec.ParentName, map[string]any{
			"alias":       flow.Spec.Alias,
			"description": flow.Spec.Description,
			"provider":    flow.Spec.ProviderID,
			"type":        flow.Spec.ChildType,
		})
		if err != nil {
			return fmt.Errorf("failed to create child auth flow: %w", err)
		}

		// Re-fetch executions to get the newly created child flow entry
		execs, _, err = h.kClientV2.AuthFlows.GetFlowExecutions(ctx, realmName, flow.Spec.ParentName)
		if err != nil {
			return fmt.Errorf("failed to get parent flow executions after child creation: %w", err)
		}

		existing = findExecByDisplayName(execs, flow.Spec.Alias)

		log.Info("Child auth flow created")
	}

	if existing != nil && existing.FlowId != nil {
		flow.Status.ID = *existing.FlowId
	}

	if flow.Spec.ChildRequirement != "" && existing != nil {
		currentReq := ""
		if existing.Requirement != nil {
			currentReq = *existing.Requirement
		}

		if currentReq != flow.Spec.ChildRequirement {
			log.Info("Updating child flow requirement", "alias", flow.Spec.Alias, "requirement", flow.Spec.ChildRequirement)

			existing.Requirement = ptr.To(flow.Spec.ChildRequirement)

			if _, err := h.kClientV2.AuthFlows.UpdateFlowExecution(ctx, realmName, flow.Spec.ParentName, *existing); err != nil {
				return fmt.Errorf("failed to update child flow requirement: %w", err)
			}
		}
	}

	return h.validateChildFlows(ctx, flow, realmName)
}

// validateChildFlows checks that all flow-type executions specified in the spec have been
// created in Keycloak. Returns an error if any are missing (causing a requeue).
func (h *CreateOrUpdateAuthFlow) validateChildFlows(ctx context.Context, flow *keycloakApi.KeycloakAuthFlow, realmName string) error {
	expectedChildFlows := 0

	for _, e := range flow.Spec.AuthenticationExecutions {
		if e.AuthenticatorFlow {
			expectedChildFlows++
		}
	}

	if expectedChildFlows == 0 {
		return nil
	}

	execs, _, err := h.kClientV2.AuthFlows.GetFlowExecutions(ctx, realmName, flow.Spec.Alias)
	if err != nil {
		return fmt.Errorf("failed to get flow executions for validation: %w", err)
	}

	createdChildFlows := 0

	for i := range execs {
		if execs[i].AuthenticationFlow != nil && *execs[i].AuthenticationFlow &&
			execs[i].Level != nil && *execs[i].Level == 0 {
			createdChildFlows++
		}
	}

	if createdChildFlows < expectedChildFlows {
		return errors.New("not all child flows have been created yet")
	}

	return nil
}

func authFlowRepFromSpec(spec keycloakApi.KeycloakAuthFlowSpec) keycloakapi.AuthFlowRepresentation {
	builtIn := spec.BuiltIn
	topLevel := spec.TopLevel

	return keycloakapi.AuthFlowRepresentation{
		Alias:       &spec.Alias,
		Description: &spec.Description,
		ProviderId:  &spec.ProviderID,
		BuiltIn:     &builtIn,
		TopLevel:    &topLevel,
	}
}

func findExecByDisplayName(
	execs []keycloakapi.AuthenticationExecutionInfoRepresentation,
	displayName string,
) *keycloakapi.AuthenticationExecutionInfoRepresentation {
	for i := range execs {
		if execs[i].DisplayName != nil && *execs[i].DisplayName == displayName {
			return &execs[i]
		}
	}

	return nil
}

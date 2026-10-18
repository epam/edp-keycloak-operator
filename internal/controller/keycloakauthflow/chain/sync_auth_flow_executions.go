package chain

import (
	"cmp"
	"context"
	"fmt"
	"slices"

	"k8s.io/utils/ptr"
	ctrl "sigs.k8s.io/controller-runtime"

	keycloakApi "github.com/epam/edp-keycloak-operator/api/v1"
	"github.com/epam/edp-keycloak-operator/internal/controller/helper"
	"github.com/epam/edp-keycloak-operator/pkg/client/keycloakapi"
	"github.com/epam/edp-keycloak-operator/pkg/secretref"
)

// configSecretsField is the spec path of the authenticator config in destination guard
// reports. It is a metric label and must stay static; the execution index goes into the error.
const configSecretsField = "spec.authenticationExecutions[].authenticatorConfig.config"

type refClient interface {
	MapConfigSecretsRefs(ctx context.Context, field string, config map[string]string, namespace string) (map[string]string, error)
}

// SyncAuthFlowExecutions syncs authentication executions for an auth flow.
// Non-flow executions have no name in Keycloak; they are paired with spec entries by
// authenticator (duplicates paired by priority order within the group) and only diffs are written.
// This ports the execution-sync logic from the legacy gocloak adapter.
type SyncAuthFlowExecutions struct {
	kClient   *keycloakapi.KeycloakClient
	secretRef refClient
}

func NewSyncAuthFlowExecutions(kClient *keycloakapi.KeycloakClient, secretRef refClient) *SyncAuthFlowExecutions {
	return &SyncAuthFlowExecutions{kClient: kClient, secretRef: secretRef}
}

// executionPair matches an existing Keycloak execution to its spec counterpart.
type executionPair struct {
	existing keycloakapi.AuthenticationExecutionInfoRepresentation
	desired  keycloakApi.AuthenticationExecution
}

func (h *SyncAuthFlowExecutions) Serve(ctx context.Context, flow *keycloakApi.KeycloakAuthFlow, realmName string) error {
	log := ctrl.LoggerFrom(ctx)
	log.Info("Syncing auth flow executions", "alias", flow.Spec.Alias)

	// References and destinations are checked before any execution is read or written; the
	// flow itself was already written by the previous handler.
	desired, newHash, err := h.resolveExecutions(ctx, flow)
	if err != nil {
		return err
	}

	execs, _, err := h.kClient.AuthFlows.GetFlowExecutions(ctx, realmName, flow.Spec.Alias)
	if err != nil {
		return fmt.Errorf("failed to get flow executions: %w", err)
	}

	forceUpdate := helper.SpecChanged(flow.Generation, flow.Status.ObservedGeneration) ||
		newHash != flow.Status.ConfigSecretsHash

	if err := h.syncNonFlowExecutions(ctx, flow, realmName, execs, desired, forceUpdate); err != nil {
		return err
	}

	// Stamp only after every config write succeeded. The controller stamps observedGeneration.
	flow.Status.ConfigSecretsHash = newHash

	// Non-flow writes above never touch flow-type siblings, so the fetched list stays valid.
	return h.adjustChildFlowsPriority(ctx, flow, realmName, execs)
}

// resolveExecutions returns deep copies of the non-flow spec entries with the secret
// references in their authenticator config resolved, and the hash of the referenced secret
// versions. The spec itself is never mutated.
func (h *SyncAuthFlowExecutions) resolveExecutions(
	ctx context.Context, flow *keycloakApi.KeycloakAuthFlow,
) ([]keycloakApi.AuthenticationExecution, string, error) {
	desired := make([]keycloakApi.AuthenticationExecution, 0, len(flow.Spec.AuthenticationExecutions))
	versions := make(map[string]string)

	for i := range flow.Spec.AuthenticationExecutions {
		if flow.Spec.AuthenticationExecutions[i].AuthenticatorFlow {
			continue
		}

		e := flow.Spec.AuthenticationExecutions[i].DeepCopy()

		if e.AuthenticatorConfig != nil {
			tokens, err := h.secretRef.MapConfigSecretsRefs(ctx, configSecretsField, e.AuthenticatorConfig.Config, flow.Namespace)
			if err != nil {
				return nil, "", fmt.Errorf("execution %d (%q): unable to map config secrets: %w", i, e.Authenticator, err)
			}

			// Key by spec index: the same config key may appear in several executions.
			for k, v := range tokens {
				versions[fmt.Sprintf("%d/%s", i, k)] = v
			}
		}

		desired = append(desired, *e)
	}

	// Hash is empty when the flow has no references.
	if len(versions) == 0 {
		return desired, "", nil
	}

	return desired, secretref.ValuesHashSingle(versions), nil
}

func (h *SyncAuthFlowExecutions) syncNonFlowExecutions(
	ctx context.Context,
	flow *keycloakApi.KeycloakAuthFlow,
	realmName string,
	execs []keycloakapi.AuthenticationExecutionInfoRepresentation,
	desired []keycloakApi.AuthenticationExecution,
	forceUpdate bool,
) error {
	existing := make([]keycloakapi.AuthenticationExecutionInfoRepresentation, 0, len(execs))

	for i := range execs {
		e := &execs[i]

		isFlowExec := e.AuthenticationFlow != nil && *e.AuthenticationFlow
		isTopLevel := e.Level != nil && *e.Level == 0

		if isFlowExec || !isTopLevel {
			continue
		}

		existing = append(existing, *e)
	}

	pairs, unmatchedExisting, unmatchedDesired := pairExecutionsByAuthenticator(existing, desired)

	for _, p := range pairs {
		if err := h.syncPairedExecution(ctx, flow.Spec.Alias, realmName, p, forceUpdate); err != nil {
			return err
		}
	}

	// Deletes go first so an added execution never transiently shares a priority slot.
	for i := range unmatchedExisting {
		if err := h.deleteExecution(ctx, realmName, &unmatchedExisting[i]); err != nil {
			return err
		}
	}

	return h.addExecutions(ctx, flow, realmName, unmatchedDesired)
}

// pairExecutionsByAuthenticator matches existing Keycloak executions to spec entries by
// authenticator. The spec may list the same authenticator more than once; each side's group
// is sorted by priority and paired index-wise.
func pairExecutionsByAuthenticator(
	existing []keycloakapi.AuthenticationExecutionInfoRepresentation,
	desired []keycloakApi.AuthenticationExecution,
) (pairs []executionPair, unmatchedExisting []keycloakapi.AuthenticationExecutionInfoRepresentation, unmatchedDesired []keycloakApi.AuthenticationExecution) {
	existingByAuth := make(map[string][]keycloakapi.AuthenticationExecutionInfoRepresentation)

	for _, e := range existing {
		key := ptr.Deref(e.ProviderId, "")
		existingByAuth[key] = append(existingByAuth[key], e)
	}

	desiredByAuth := make(map[string][]keycloakApi.AuthenticationExecution)
	for _, e := range desired {
		desiredByAuth[e.Authenticator] = append(desiredByAuth[e.Authenticator], e)
	}

	for auth, existGroup := range existingByAuth {
		// Stable sorts keep pairing deterministic when priorities tie.
		slices.SortStableFunc(existGroup, func(a, b keycloakapi.AuthenticationExecutionInfoRepresentation) int {
			return cmp.Compare(ptr.Deref(a.Priority, 0), ptr.Deref(b.Priority, 0))
		})

		desireGroup := desiredByAuth[auth]
		slices.SortStableFunc(desireGroup, func(a, b keycloakApi.AuthenticationExecution) int {
			return cmp.Compare(a.Priority, b.Priority)
		})

		paired := min(len(existGroup), len(desireGroup))

		for i := 0; i < paired; i++ {
			pairs = append(pairs, executionPair{existing: existGroup[i], desired: desireGroup[i]})
		}

		unmatchedExisting = append(unmatchedExisting, existGroup[paired:]...)
		unmatchedDesired = append(unmatchedDesired, desireGroup[paired:]...)

		delete(desiredByAuth, auth)
	}

	for _, desireGroup := range desiredByAuth {
		unmatchedDesired = append(unmatchedDesired, desireGroup...)
	}

	return pairs, unmatchedExisting, unmatchedDesired
}

// syncPairedExecution updates an existing execution in place when its requirement or priority
// drifts from spec, then reconciles its authenticator config. Executions are never
// deleted and recreated to keep their Keycloak ID (and position) stable.
func (h *SyncAuthFlowExecutions) syncPairedExecution(
	ctx context.Context, flowAlias, realmName string, p executionPair, forceUpdate bool,
) error {
	existing := p.existing
	desired := p.desired

	// Requirement and priority are exact scalar diffs; no force needed to converge.
	requirementDiffers := ptr.Deref(existing.Requirement, "") != desired.Requirement
	priorityDiffers := ptr.Deref(existing.Priority, 0) != int32(desired.Priority)

	if requirementDiffers || priorityDiffers {
		existing.Requirement = ptr.To(desired.Requirement)
		existing.Priority = ptr.To(int32(desired.Priority))

		if _, err := h.kClient.AuthFlows.UpdateFlowExecution(ctx, realmName, flowAlias, existing); err != nil {
			return fmt.Errorf("failed to update execution %q: %w", desired.Authenticator, err)
		}
	}

	return h.syncExecutionConfig(ctx, realmName, existing, desired, forceUpdate)
}

// syncExecutionConfig reconciles the authenticator config of a paired execution: creates it
// when only the spec has one, deletes it when only Keycloak has one, and updates it in place
// when both have one but the alias or values drifted.
func (h *SyncAuthFlowExecutions) syncExecutionConfig(
	ctx context.Context,
	realmName string,
	existing keycloakapi.AuthenticationExecutionInfoRepresentation,
	desired keycloakApi.AuthenticationExecution,
	forceUpdate bool,
) error {
	existingConfigID := existing.AuthenticationConfig
	desiredConfig := desired.AuthenticatorConfig

	switch {
	case desiredConfig == nil && existingConfigID == nil:
		return nil
	case desiredConfig != nil && existingConfigID == nil:
		execID := ptr.Deref(existing.Id, "")
		if execID == "" {
			return fmt.Errorf("execution %q has no ID; cannot create config", desired.Authenticator)
		}

		return h.createExecutionConfig(ctx, realmName, execID, desiredConfig)
	case desiredConfig == nil && existingConfigID != nil:
		if _, err := h.kClient.AuthFlows.DeleteAuthenticatorConfig(ctx, realmName, *existingConfigID); err != nil && !keycloakapi.IsNotFound(err) {
			return fmt.Errorf("failed to delete authenticator config %q: %w", *existingConfigID, err)
		}

		return nil
	default:
		return h.updateExecutionConfigIfNeeded(ctx, realmName, ptr.Deref(existing.Id, ""), *existingConfigID, desiredConfig, forceUpdate)
	}
}

func (h *SyncAuthFlowExecutions) updateExecutionConfigIfNeeded(
	ctx context.Context,
	realmName, execID, configID string,
	desiredConfig *keycloakApi.AuthenticatorConfig,
	forceUpdate bool,
) error {
	if !forceUpdate {
		existingCfg, _, err := h.kClient.AuthFlows.GetAuthenticatorConfig(ctx, realmName, configID)
		if err != nil && !keycloakapi.IsNotFound(err) {
			return fmt.Errorf("failed to get authenticator config %q: %w", configID, err)
		}

		if existingCfg != nil && configMatchesSpec(existingCfg, desiredConfig) {
			return nil
		}
	}

	_, err := h.kClient.AuthFlows.UpdateAuthenticatorConfig(ctx, realmName, configID, keycloakapi.AuthenticatorConfigRepresentation{
		Id:     &configID,
		Alias:  &desiredConfig.Alias,
		Config: &desiredConfig.Config,
	})
	if err == nil {
		return nil
	}

	if !keycloakapi.IsNotFound(err) {
		return fmt.Errorf("failed to update authenticator config %q: %w", configID, err)
	}

	// Dangling config reference: the execution points at a config that no longer exists.
	return h.createExecutionConfig(ctx, realmName, execID, desiredConfig)
}

// configMatchesSpec reports whether the fetched authenticator config already matches the
// resolved spec config. Keycloak masks every config key the provider does not declare as a
// non-secret property; masked keys are excluded and repaired only by a forced update
// (generation change or secret rotation).
func configMatchesSpec(existing *keycloakapi.AuthenticatorConfigRepresentation, desired *keycloakApi.AuthenticatorConfig) bool {
	return ptr.Deref(existing.Alias, "") == desired.Alias &&
		secretref.ConfigMatches(ptr.Deref(existing.Config, nil), desired.Config, nil)
}

// addExecutions posts spec entries that had no matching existing execution.
func (h *SyncAuthFlowExecutions) addExecutions(
	ctx context.Context, flow *keycloakApi.KeycloakAuthFlow, realmName string, execsToAdd []keycloakApi.AuthenticationExecution,
) error {
	if len(execsToAdd) == 0 {
		return nil
	}

	// The flow's internal Keycloak ID is set by the preceding CreateOrUpdateAuthFlow chain step.
	flowID := flow.Status.ID
	if flowID == "" {
		return fmt.Errorf("flow ID is empty for alias %q; ensure CreateOrUpdateAuthFlow ran first", flow.Spec.Alias)
	}

	// Sort by priority before adding (mirrors legacy adapter behaviour).
	slices.SortStableFunc(execsToAdd, func(a, b keycloakApi.AuthenticationExecution) int {
		return cmp.Compare(a.Priority, b.Priority)
	})

	for i := range execsToAdd {
		e := &execsToAdd[i]

		execID, err := h.addExecution(ctx, realmName, flowID, e.Authenticator, e.Requirement, e.Priority)
		if err != nil {
			return fmt.Errorf("failed to add execution %q: %w", e.Authenticator, err)
		}

		if e.AuthenticatorConfig != nil {
			if err := h.createExecutionConfig(ctx, realmName, execID, e.AuthenticatorConfig); err != nil {
				return fmt.Errorf("failed to create config for execution %q: %w", e.Authenticator, err)
			}
		}
	}

	return nil
}

// deleteExecution removes an execution that left the spec, along with its authenticator config.
func (h *SyncAuthFlowExecutions) deleteExecution(
	ctx context.Context, realmName string, e *keycloakapi.AuthenticationExecutionInfoRepresentation,
) error {
	if e.AuthenticationConfig != nil {
		if _, err := h.kClient.AuthFlows.DeleteAuthenticatorConfig(ctx, realmName, *e.AuthenticationConfig); err != nil && !keycloakapi.IsNotFound(err) {
			return fmt.Errorf("failed to delete authenticator config %q: %w", *e.AuthenticationConfig, err)
		}
	}

	execID := ptr.Deref(e.Id, "")
	if execID == "" {
		return nil
	}

	if _, err := h.kClient.AuthFlows.DeleteExecution(ctx, realmName, execID); err != nil {
		return fmt.Errorf("failed to delete execution %q: %w", execID, err)
	}

	return nil
}

// addExecution posts a new execution to the flow and returns the new execution ID from the Location header.
// Priority is always serialized (even when zero) so that Keycloak does not auto-assign it sequentially —
// auto-assigned priorities collide with later adjustChildFlowsPriority updates when a parent flow mixes
// non-flow executions with a child sub-flow.
func (h *SyncAuthFlowExecutions) addExecution(
	ctx context.Context, realmName, flowID, authenticator, requirement string, priority int,
) (string, error) {
	p := int32(priority)

	resp, err := h.kClient.AuthFlows.AddExecutionToFlow(ctx, realmName, keycloakapi.AuthenticationExecutionRepresentation{
		Authenticator: &authenticator,
		ParentFlow:    &flowID,
		Requirement:   &requirement,
		Priority:      &p,
	})
	if err != nil {
		return "", fmt.Errorf("failed to post execution: %w", err)
	}

	execID := keycloakapi.GetResourceIDFromResponse(resp)
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
	_, err := h.kClient.AuthFlows.CreateExecutionConfig(ctx, realmName, execID, keycloakapi.AuthenticatorConfigRepresentation{
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
func (h *SyncAuthFlowExecutions) adjustChildFlowsPriority(
	ctx context.Context,
	flow *keycloakApi.KeycloakAuthFlow,
	realmName string,
	execs []keycloakapi.AuthenticationExecutionInfoRepresentation,
) error {
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
			if _, err := h.kClient.AuthFlows.UpdateFlowExecution(ctx, realmName, flow.Spec.Alias, *e); err != nil {
				return fmt.Errorf("failed to update priority for child flow %q: %w", *e.DisplayName, err)
			}
		}
	}

	return nil
}

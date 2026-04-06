package chain

import (
	"context"
	"fmt"

	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	keycloakApi "github.com/epam/edp-keycloak-operator/api/v1"
	keycloakv2 "github.com/epam/edp-keycloak-operator/pkg/client/keycloakv2"
	"github.com/epam/edp-keycloak-operator/pkg/objectmeta"
)

// RemoveAuthFlow handles deletion of a KeycloakAuthFlow from Keycloak.
// It ports the terminator and legacy DeleteAuthFlow + unsetBrowserFlow logic.
type RemoveAuthFlow struct {
	kClientV2 *keycloakv2.KeycloakClient
	k8sClient client.Client
}

func NewRemoveAuthFlow(kClientV2 *keycloakv2.KeycloakClient, k8sClient client.Client) *RemoveAuthFlow {
	return &RemoveAuthFlow{kClientV2: kClientV2, k8sClient: k8sClient}
}

func (h *RemoveAuthFlow) Serve(ctx context.Context, flow *keycloakApi.KeycloakAuthFlow, realmName string) error {
	log := ctrl.LoggerFrom(ctx).WithValues("realm", realmName, "alias", flow.Spec.Alias)

	if objectmeta.PreserveResourcesOnDeletion(flow) {
		log.Info("PreserveResourcesOnDeletion is enabled, skipping deletion")

		return nil
	}

	if err := h.checkNoChildFlows(ctx, flow); err != nil {
		return err
	}

	if flow.Spec.ParentName != "" {
		return h.deleteChildFlow(ctx, flow, realmName)
	}

	return h.deleteTopLevelFlow(ctx, flow, realmName)
}

// checkNoChildFlows blocks deletion if any K8s KeycloakAuthFlow references this flow as parent.
func (h *RemoveAuthFlow) checkNoChildFlows(ctx context.Context, flow *keycloakApi.KeycloakAuthFlow) error {
	var list keycloakApi.KeycloakAuthFlowList
	if err := h.k8sClient.List(ctx, &list); err != nil {
		return fmt.Errorf("failed to list KeycloakAuthFlow resources: %w", err)
	}

	for i := range list.Items {
		item := &list.Items[i]

		if item.Spec.RealmRef.Name == flow.Spec.RealmRef.Name &&
			item.Spec.RealmRef.Kind == flow.Spec.RealmRef.Kind &&
			item.Spec.ParentName == flow.Spec.Alias {
			return fmt.Errorf("cannot delete flow %q: child flow %q still exists", flow.Spec.Alias, item.Spec.Alias)
		}
	}

	return nil
}

// deleteChildFlow finds the execution representing the child flow in its parent and deletes it.
func (h *RemoveAuthFlow) deleteChildFlow(ctx context.Context, flow *keycloakApi.KeycloakAuthFlow, realmName string) error {
	log := ctrl.LoggerFrom(ctx)

	execs, _, err := h.kClientV2.AuthFlows.GetFlowExecutions(ctx, realmName, flow.Spec.ParentName)
	if err != nil {
		if keycloakv2.IsNotFound(err) {
			log.Info("Parent flow not found, skipping child deletion")

			return nil
		}

		return fmt.Errorf("failed to get parent flow executions: %w", err)
	}

	for i := range execs {
		if execs[i].DisplayName == nil || *execs[i].DisplayName != flow.Spec.Alias {
			continue
		}

		if execs[i].Id == nil {
			return fmt.Errorf("child flow execution %q has no ID", flow.Spec.Alias)
		}

		log.Info("Deleting child flow execution", "alias", flow.Spec.Alias)

		if _, err := h.kClientV2.AuthFlows.DeleteExecution(ctx, realmName, *execs[i].Id); err != nil {
			if keycloakv2.IsNotFound(err) {
				log.Info("Child flow execution not found, skipping")

				return nil
			}

			return fmt.Errorf("failed to delete child flow execution: %w", err)
		}

		return nil
	}

	log.Info("Child flow not found in parent executions, skipping")

	return nil
}

// deleteTopLevelFlow unsets the realm browser flow if needed, then deletes the flow.
func (h *RemoveAuthFlow) deleteTopLevelFlow(ctx context.Context, flow *keycloakApi.KeycloakAuthFlow, realmName string) error {
	log := ctrl.LoggerFrom(ctx)

	flowID := flow.Status.ID
	if flowID == "" {
		log.Info("Auth flow ID not set in status, skipping deletion")

		return nil
	}

	if err := h.unsetBrowserFlow(ctx, realmName, flow.Spec.Alias); err != nil {
		return fmt.Errorf("failed to unset browser flow: %w", err)
	}

	log.Info("Deleting top-level auth flow", "id", flowID)

	if _, err := h.kClientV2.AuthFlows.DeleteAuthFlow(ctx, realmName, flowID); err != nil {
		if keycloakv2.IsNotFound(err) {
			log.Info("Auth flow not found, skipping deletion")

			return nil
		}

		return fmt.Errorf("failed to delete auth flow: %w", err)
	}

	return nil
}

// unsetBrowserFlow replaces the realm browser flow with another flow if it currently
// points to the flow being deleted. Ports legacy unsetBrowserFlow from the gocloak adapter.
func (h *RemoveAuthFlow) unsetBrowserFlow(ctx context.Context, realmName, flowAlias string) error {
	realm, _, err := h.kClientV2.Realms.GetRealm(ctx, realmName)
	if err != nil {
		return fmt.Errorf("failed to get realm: %w", err)
	}

	if realm.BrowserFlow == nil || *realm.BrowserFlow != flowAlias {
		return nil
	}

	flows, _, err := h.kClientV2.AuthFlows.GetAuthFlows(ctx, realmName)
	if err != nil {
		return fmt.Errorf("failed to get auth flows: %w", err)
	}

	var replaceAlias string

	for i := range flows {
		if flows[i].Alias != nil && *flows[i].Alias != flowAlias {
			replaceAlias = *flows[i].Alias

			break
		}
	}

	if replaceAlias == "" {
		return fmt.Errorf("no replacement flow found for browser flow %q", flowAlias)
	}

	realm.BrowserFlow = &replaceAlias

	if _, err := h.kClientV2.Realms.UpdateRealm(ctx, realmName, *realm); err != nil {
		return fmt.Errorf("failed to update realm browser flow: %w", err)
	}

	return nil
}

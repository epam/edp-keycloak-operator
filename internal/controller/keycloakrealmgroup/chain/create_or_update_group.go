package chain

import (
	"context"
	"fmt"

	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	keycloakApi "github.com/epam/edp-keycloak-operator/api/v1"
	"github.com/epam/edp-keycloak-operator/pkg/client/keycloakapi"
)

type CreateOrUpdateGroup struct {
	k8sClient client.Client
}

func NewCreateOrUpdateGroup(k8sClient client.Client) *CreateOrUpdateGroup {
	return &CreateOrUpdateGroup{k8sClient: k8sClient}
}

func (h *CreateOrUpdateGroup) Serve(
	ctx context.Context,
	group *keycloakApi.KeycloakRealmGroup,
	kClient *keycloakapi.KeycloakClient,
	groupCtx *GroupContext,
) error {
	log := ctrl.LoggerFrom(ctx)
	log.Info("Creating or updating group in Keycloak")

	spec := &group.Spec
	realm := groupCtx.RealmName

	var (
		existingGroup *keycloakapi.GroupRepresentation
		err           error
	)

	// If we already have an ID from a previous reconciliation, fetch by ID first.
	// This handles renames: spec.Name may have changed but the ID stays the same.
	if groupCtx.GroupID != "" {
		existingGroup, _, err = kClient.Groups.GetGroup(ctx, realm, groupCtx.GroupID)
		if err != nil && !keycloakapi.IsNotFound(err) {
			return fmt.Errorf("unable to get group by ID %q: %w", groupCtx.GroupID, err)
		}

		if keycloakapi.IsNotFound(err) {
			log.Info("Group not found by ID, will search by name", "groupID", groupCtx.GroupID)

			existingGroup = nil
		}
	}

	// If we didn't find the group by ID, search by name.
	foundByName := false

	if existingGroup == nil {
		if groupCtx.ParentGroupID != "" {
			existingGroup, _, err = kClient.Groups.FindChildGroupByName(ctx, realm, groupCtx.ParentGroupID, spec.Name)
		} else {
			existingGroup, _, err = kClient.Groups.FindGroupByName(ctx, realm, spec.Name)
		}

		if err != nil && !keycloakapi.IsNotFound(err) {
			return fmt.Errorf("unable to search for group %q: %w", spec.Name, err)
		}

		foundByName = existingGroup != nil
	}

	// A group found by name search (as opposed to by our own status.ID) may already be
	// managed by a different KeycloakRealmGroup resource, e.g. another CR using the same
	// spec.name/parentGroup combination. Adopting it would make two CRs share one Keycloak
	// group ID and fight over its children on every reconcile. Refuse instead of adopting.
	if foundByName && existingGroup.Id != nil {
		owner, ownerErr := findOwnerCR(ctx, h.k8sClient, group, *existingGroup.Id)
		if ownerErr != nil {
			return fmt.Errorf("unable to check group ownership for %q: %w", spec.Name, ownerErr)
		}

		if owner != nil {
			return fmt.Errorf(
				"group %q (id %s) is already managed by KeycloakRealmGroup %s/%s; "+
					"refusing to adopt it - check for a duplicate spec.name/parentGroup combination",
				spec.Name, *existingGroup.Id, owner.Namespace, owner.Name,
			)
		}
	}

	if existingGroup == nil {
		groupRep := keycloakapi.GroupRepresentation{
			Name:        &spec.Name,
			Description: &spec.Description,
			Path:        &spec.Path,
			Attributes:  &spec.Attributes,
		}

		var resp *keycloakapi.Response

		if groupCtx.ParentGroupID != "" {
			resp, err = kClient.Groups.CreateChildGroup(ctx, realm, groupCtx.ParentGroupID, groupRep)
		} else {
			resp, err = kClient.Groups.CreateGroup(ctx, realm, groupRep)
		}

		if err != nil {
			return fmt.Errorf("unable to create group %q: %w", spec.Name, err)
		}

		groupCtx.GroupID = keycloakapi.GetResourceIDFromResponse(resp)
		log.Info("Group created", "groupID", groupCtx.GroupID)
	} else {
		groupCtx.GroupID = *existingGroup.Id
		existingGroup.Name = &spec.Name
		existingGroup.Description = &spec.Description
		existingGroup.Path = &spec.Path
		existingGroup.Attributes = &spec.Attributes

		if _, err := kClient.Groups.UpdateGroup(ctx, realm, groupCtx.GroupID, *existingGroup); err != nil {
			return fmt.Errorf("unable to update group %q: %w", spec.Name, err)
		}

		log.Info("Group updated", "groupID", groupCtx.GroupID)
	}

	return nil
}

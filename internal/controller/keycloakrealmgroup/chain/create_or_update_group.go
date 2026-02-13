package chain

import (
	"context"
	"fmt"

	ctrl "sigs.k8s.io/controller-runtime"

	keycloakApi "github.com/epam/edp-keycloak-operator/api/v1"
	keycloakv2 "github.com/epam/edp-keycloak-operator/pkg/client/keycloakv2"
)

type CreateOrUpdateGroup struct{}

func NewCreateOrUpdateGroup() *CreateOrUpdateGroup {
	return &CreateOrUpdateGroup{}
}

func (h *CreateOrUpdateGroup) Serve(
	ctx context.Context,
	group *keycloakApi.KeycloakRealmGroup,
	kClient *keycloakv2.KeycloakClient,
	groupCtx *GroupContext,
) error {
	log := ctrl.LoggerFrom(ctx)
	log.Info("Creating or updating group in Keycloak")

	spec := &group.Spec
	realm := groupCtx.RealmName

	var (
		existingGroup *keycloakv2.GroupRepresentation
		err           error
	)

	if groupCtx.ParentGroupID != "" {
		existingGroup, _, err = kClient.Groups.FindChildGroupByName(ctx, realm, groupCtx.ParentGroupID, spec.Name)
	} else {
		existingGroup, _, err = kClient.Groups.FindGroupByName(ctx, realm, spec.Name)
	}

	if err != nil {
		return fmt.Errorf("unable to search for group %q: %w", spec.Name, err)
	}

	if existingGroup == nil {
		groupRep := keycloakv2.GroupRepresentation{
			Name:       &spec.Name,
			Path:       &spec.Path,
			Attributes: &spec.Attributes,
		}

		var resp *keycloakv2.Response

		if groupCtx.ParentGroupID != "" {
			resp, err = kClient.Groups.CreateChildGroup(ctx, realm, groupCtx.ParentGroupID, groupRep)
		} else {
			resp, err = kClient.Groups.CreateGroup(ctx, realm, groupRep)
		}

		if err != nil {
			return fmt.Errorf("unable to create group %q: %w", spec.Name, err)
		}

		groupCtx.GroupID = keycloakv2.GetResourceIDFromResponse(resp)
		log.Info("Group created", "groupID", groupCtx.GroupID)
	} else {
		groupCtx.GroupID = *existingGroup.Id
		existingGroup.Path = &spec.Path
		existingGroup.Attributes = &spec.Attributes

		if _, err := kClient.Groups.UpdateGroup(ctx, realm, groupCtx.GroupID, *existingGroup); err != nil {
			return fmt.Errorf("unable to update group %q: %w", spec.Name, err)
		}

		log.Info("Group updated", "groupID", groupCtx.GroupID)
	}

	return nil
}

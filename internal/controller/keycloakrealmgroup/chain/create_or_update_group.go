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

	existingGroup, err := findGroupByName(ctx, kClient, realm, spec.Name)
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

// findGroupByName searches for a group by exact name match.
// Returns nil if not found.
func findGroupByName(
	ctx context.Context,
	kClient *keycloakv2.KeycloakClient,
	realm, groupName string,
) (*keycloakv2.GroupRepresentation, error) {
	search := groupName

	groups, _, err := kClient.Groups.GetGroups(ctx, realm, &keycloakv2.GetGroupsParams{
		Search: &search,
	})
	if err != nil {
		return nil, err
	}

	for i := range groups {
		if found := findGroupByNameRecursive(groups[i], groupName); found != nil {
			return found, nil
		}
	}

	return nil, nil
}

// findGroupByNameRecursive does a recursive exact-name search through group and its subgroups.
func findGroupByNameRecursive(
	group keycloakv2.GroupRepresentation,
	name string,
) *keycloakv2.GroupRepresentation {
	if group.Name != nil && *group.Name == name {
		return &group
	}

	if group.SubGroups != nil {
		for i := range *group.SubGroups {
			if found := findGroupByNameRecursive((*group.SubGroups)[i], name); found != nil {
				return found
			}
		}
	}

	return nil
}

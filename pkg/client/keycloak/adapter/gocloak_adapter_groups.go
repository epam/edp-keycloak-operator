package adapter

import (
	"context"

	"github.com/Nerzal/gocloak/v10"
	"github.com/pkg/errors"

	keycloakApi "github.com/epam/edp-keycloak-operator/pkg/apis/v1/v1"
)

type ErrNotFound string

func (e ErrNotFound) Error() string {
	return string(e)
}

func IsErrNotFound(err error) bool {
	switch errors.Cause(err).(type) {
	case ErrNotFound:
		return true
	default:
		return false
	}
}

func (a GoCloakAdapter) getGroup(realm, groupName string) (*gocloak.Group, error) {
	groups, err := a.client.GetGroups(context.Background(), a.token.AccessToken, realm, gocloak.GetGroupsParams{
		Search: gocloak.StringP(groupName),
	})
	if err != nil {
		return nil, errors.Wrap(err, "unable to search groups")
	}

	for _, g := range groups {
		if *g.Name == groupName {
			return g, nil
		}
	}

	return nil, ErrNotFound("group not found")
}

func (a GoCloakAdapter) syncGroupRoles(realmName, groupID string, spec *keycloakApi.KeycloakRealmGroupSpec) error {
	roleMap, err := a.client.GetRoleMappingByGroupID(context.Background(), a.token.AccessToken, realmName, groupID)
	if err != nil {
		return errors.Wrapf(err, "unable to get role mappings for group spec %+v", spec)
	}

	if err := a.syncEntityRealmRoles(groupID, realmName, spec.RealmRoles, roleMap.RealmMappings,
		a.client.AddRealmRoleToGroup, a.client.DeleteRealmRoleFromGroup); err != nil {
		return errors.Wrapf(err, "unable to sync group realm roles, groupID: %s with spec %+v", groupID, spec)
	}

	claimedClientRoles := make(map[string][]string)
	for _, cr := range spec.ClientRoles {
		claimedClientRoles[cr.ClientID] = cr.Roles
	}

	if err := a.syncEntityClientRoles(realmName, groupID, claimedClientRoles, roleMap.ClientMappings,
		a.client.AddClientRoleToGroup, a.client.DeleteClientRoleFromGroup); err != nil {
		return errors.Wrapf(err, "unable to sync client roles for group: %+v", spec)
	}

	return nil
}

func (a GoCloakAdapter) syncSubGroups(realm string, group *gocloak.Group, subGroups []string) error {
	currentGroups, claimedGroups := make(map[string]gocloak.Group), make(map[string]struct{})

	if group.SubGroups != nil {
		for i, cg := range *group.SubGroups {
			currentGroups[*cg.Name] = (*group.SubGroups)[i]
		}
	}

	for _, g := range subGroups {
		claimedGroups[g] = struct{}{}
	}

	for _, claimed := range subGroups {
		if _, ok := currentGroups[claimed]; !ok {
			gr, err := a.getGroup(realm, claimed)
			if err != nil {
				return errors.Wrapf(err, "unable to get group, realm: %s, group: %s", realm, claimed)
			}

			if _, err := a.client.CreateChildGroup(context.Background(), a.token.AccessToken, realm, *group.ID,
				*gr); err != nil {
				return errors.Wrapf(err, "unable to create child group, realm: %s, group: %s", realm, claimed)
			}
		}
	}

	for name, current := range currentGroups {
		if _, ok := claimedGroups[name]; !ok {
			//this is strange but if we call create group on subgroup it will be detached from parent group %)
			if _, err := a.client.CreateGroup(context.Background(), a.token.AccessToken, realm, current); err != nil {
				return errors.Wrapf(err, "unable to detach subgroup from group, realm: %s, subgroup: %s, group: %+v",
					realm, name, group)
			}
		}
	}

	return nil
}

func (a GoCloakAdapter) SyncRealmGroup(realmName string, spec *keycloakApi.KeycloakRealmGroupSpec) (string, error) {
	group, err := a.getGroup(realmName, spec.Name)
	if err != nil {
		if !IsErrNotFound(err) {
			return "", errors.Wrapf(err, "unable to get group with spec %+v", spec)
		}

		group = &gocloak.Group{Name: &spec.Name, Path: &spec.Path, Attributes: &spec.Attributes,
			Access: &spec.Access}
		groupID, err := a.client.CreateGroup(context.Background(), a.token.AccessToken, realmName, *group)
		if err != nil {
			return "", errors.Wrapf(err, "unable to create group with spec %+v", spec)
		}
		group.ID = &groupID
	} else {
		group.Path, group.Access, group.Attributes = &spec.Path, &spec.Access, &spec.Attributes
		if err := a.client.UpdateGroup(context.Background(), a.token.AccessToken, realmName, *group); err != nil {
			return "", errors.Wrapf(err, "unable to update group, realm: %s, group spec: %+v", realmName, spec)
		}
	}

	if err := a.syncGroupRoles(realmName, *group.ID, spec); err != nil {
		return "", errors.Wrapf(err, "unable to sync group realm roles, group: %+v with spec %+v", group, spec)
	}

	if err := a.syncSubGroups(realmName, group, spec.SubGroups); err != nil {
		return "", errors.Wrapf(err, "unable to sync subgroups, group: %+v with spec: %+v", group, spec)
	}

	return *group.ID, nil
}

func (a GoCloakAdapter) DeleteGroup(ctx context.Context, realm, groupName string) error {
	group, err := a.getGroup(realm, groupName)
	if err != nil {
		return errors.Wrapf(err, "unable to get group, realm: %s, group: %s", realm, groupName)
	}

	if err := a.client.DeleteGroup(ctx, a.token.AccessToken, realm, *group.ID); err != nil {
		return errors.Wrapf(err, "unable to delete group, realm: %s, group: %s", realm, groupName)
	}

	return nil
}

package adapter

import (
	"context"
	"fmt"
	"strings"
	"sync"

	"github.com/Nerzal/gocloak/v12"
	"github.com/pkg/errors"
	"golang.org/x/sync/errgroup"

	keycloakApi "github.com/epam/edp-keycloak-operator/api/v1"
)

type NotFoundError string

func (e NotFoundError) Error() string {
	return string(e)
}

// GetGroups return top-level groups for a realm.
func (a GoCloakAdapter) GetGroups(ctx context.Context, realm string) (map[string]*gocloak.Group, error) {
	groups, err := a.client.GetGroups(
		ctx,
		a.token.AccessToken,
		realm,
		gocloak.GetGroupsParams{
			Max: gocloak.IntP(100),
		},
	)
	if err != nil {
		return nil, fmt.Errorf("failed to get groups: %w", err)
	}

	groupMap := make(map[string]*gocloak.Group, len(groups))

	for _, g := range groups {
		if g != nil && g.Name != nil {
			groupMap[*g.Name] = g
		}
	}

	return groupMap, nil
}

func (a GoCloakAdapter) getGroup(ctx context.Context, realm, groupName string) (*gocloak.Group, error) {
	groups, err := a.client.GetGroups(ctx, a.token.AccessToken, realm, gocloak.GetGroupsParams{
		Search: gocloak.StringP(groupName),
	})
	if err != nil {
		return nil, errors.Wrap(err, "unable to search groups")
	}

	gr := make([]gocloak.Group, len(groups))

	for i, g := range groups {
		if g != nil {
			gr[i] = *g
		}
	}

	group := getGroupByName(gr, groupName)
	if group != nil {
		return group, nil
	}

	return nil, NotFoundError("group not found")
}

func (a GoCloakAdapter) getGroupsByNames(ctx context.Context, realm string, groupNames []string) (map[string]gocloak.Group, error) {
	groups := make(map[string]gocloak.Group, len(groupNames))
	eg := errgroup.Group{}
	m := sync.Mutex{}

	for _, groupName := range groupNames {
		eg.Go(func() error {
			group, err := a.getGroup(ctx, realm, groupName)
			if err != nil {
				return err
			}

			m.Lock()
			defer m.Unlock()

			groups[groupName] = *group

			return nil
		})
	}

	if err := eg.Wait(); err != nil {
		return nil, fmt.Errorf("failed to get groups by names: %w", err)
	}

	return groups, nil
}

func getGroupByName(groups []gocloak.Group, groupName string) *gocloak.Group {
	for _, g := range groups {
		if *g.Name == groupName {
			return &g
		}

		if g.SubGroups != nil {
			return getGroupByName(*g.SubGroups, groupName)
		}
	}

	return nil
}

func (a GoCloakAdapter) getChildGroups(ctx context.Context, realm string, parentGroup *gocloak.Group) ([]gocloak.Group, error) {
	var result []gocloak.Group

	resp, err := a.client.RestyClient().R().
		SetContext(ctx).
		SetAuthToken(a.token.AccessToken).
		SetHeader(contentTypeHeader, contentTypeJson).
		SetPathParams(map[string]string{
			keycloakApiParamRealm: realm,
			"groupID":             *parentGroup.ID,
		}).
		SetResult(&result).
		Get(a.buildPath(getChildGroups))

	if err = a.checkError(err, resp); err != nil {
		// Use workaround for Keycloak versions < 23.0.0 for backward compatibility.
		if strings.Contains(err.Error(), "No resource method found for GET, return 405 with Allow header") {
			r, err := a.getChildGroupsKCVersionUnder23(ctx, realm, parentGroup)
			if err != nil {
				return nil, fmt.Errorf("unable to get child groups: %w", err)
			}

			return r, nil
		}

		return nil, fmt.Errorf("unable to get child groups, rsp: %s", resp.String())
	}

	return result, nil
}

// getChildGroupsKCVersionUnder23 is a workaround for Keycloak versions < 23.0.0.
// Group model in Keycloak < 23.0.0 contains subgroups.
// In Keycloak >= 23.0.0 to get subgroups we need to use dedicated endpoint.
func (a GoCloakAdapter) getChildGroupsKCVersionUnder23(ctx context.Context, realm string, parentGroup *gocloak.Group) ([]gocloak.Group, error) {
	result := &gocloak.Group{}

	resp, err := a.client.RestyClient().R().
		SetContext(ctx).
		SetAuthToken(a.token.AccessToken).
		SetHeader(contentTypeHeader, contentTypeJson).
		SetPathParams(map[string]string{
			keycloakApiParamRealm: realm,
			"groupID":             *parentGroup.ID,
		}).
		SetResult(result).
		Get(a.buildPath(getGroup))

	if err = a.checkError(err, resp); err != nil {
		return nil, fmt.Errorf("unable to get group: %s", resp.String())
	}

	if result.SubGroups == nil {
		return nil, nil
	}

	return *result.SubGroups, nil
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

func (a GoCloakAdapter) syncSubGroups(ctx context.Context, realm string, group *gocloak.Group, subGroups []string) error {
	currentGroups, err := a.makeCurrentGroups(ctx, realm, group)
	if err != nil {
		return err
	}

	claimedGroups := make(map[string]struct{}, len(subGroups))

	for _, g := range subGroups {
		claimedGroups[g] = struct{}{}
	}

	for _, claimed := range subGroups {
		if _, ok := currentGroups[claimed]; !ok {
			gr, err := a.getGroup(ctx, realm, claimed)
			if err != nil {
				return errors.Wrapf(err, "unable to get group, realm: %s, group: %s", realm, claimed)
			}

			if _, err := a.client.CreateChildGroup(ctx, a.token.AccessToken, realm, *group.ID, *gr); err != nil {
				return errors.Wrapf(err, "unable to create child group, realm: %s, group: %s", realm, claimed)
			}
		}
	}

	for name, current := range currentGroups {
		if _, ok := claimedGroups[name]; !ok {
			// this is strange but if we call create group on subgroup it will be detached from parent group %)
			if _, err := a.client.CreateGroup(ctx, a.token.AccessToken, realm, current); err != nil {
				return errors.Wrapf(err, "unable to detach subgroup from group, realm: %s, subgroup: %s, group: %+v",
					realm, name, group)
			}
		}
	}

	return nil
}

func (a GoCloakAdapter) SyncRealmGroup(ctx context.Context, realmName string, spec *keycloakApi.KeycloakRealmGroupSpec) (string, error) {
	group, err := a.getGroup(ctx, realmName, spec.Name)
	if err != nil {
		if !IsErrNotFound(err) {
			return "", errors.Wrapf(err, "unable to get group with spec %+v", spec)
		}

		group = &gocloak.Group{Name: &spec.Name, Path: &spec.Path, Attributes: &spec.Attributes, Access: &spec.Access}

		groupID, err := a.client.CreateGroup(ctx, a.token.AccessToken, realmName, *group)
		if err != nil {
			return "", errors.Wrapf(err, "unable to create group with spec %+v", spec)
		}

		group.ID = &groupID
	} else {
		group.Path, group.Access, group.Attributes = &spec.Path, &spec.Access, &spec.Attributes
		if err := a.client.UpdateGroup(ctx, a.token.AccessToken, realmName, *group); err != nil {
			return "", errors.Wrapf(err, "unable to update group, realm: %s, group spec: %+v", realmName, spec)
		}
	}

	if err := a.syncGroupRoles(realmName, *group.ID, spec); err != nil {
		return "", errors.Wrapf(err, "unable to sync group realm roles, group: %+v with spec %+v", group, spec)
	}

	if err := a.syncSubGroups(ctx, realmName, group, spec.SubGroups); err != nil {
		return "", errors.Wrapf(err, "unable to sync subgroups, group: %+v with spec: %+v", group, spec)
	}

	return *group.ID, nil
}

func (a GoCloakAdapter) DeleteGroup(ctx context.Context, realm, groupName string) error {
	group, err := a.getGroup(ctx, realm, groupName)
	if err != nil {
		return errors.Wrapf(err, "unable to get group, realm: %s, group: %s", realm, groupName)
	}

	if err := a.client.DeleteGroup(ctx, a.token.AccessToken, realm, *group.ID); err != nil {
		return errors.Wrapf(err, "unable to delete group, realm: %s, group: %s", realm, groupName)
	}

	return nil
}

func (a GoCloakAdapter) makeCurrentGroups(ctx context.Context, realm string, group *gocloak.Group) (map[string]gocloak.Group, error) {
	child, err := a.getChildGroups(ctx, realm, group)
	if err != nil {
		return nil, err
	}

	currentGroups := make(map[string]gocloak.Group, len(child))

	for _, c := range child {
		currentGroups[*c.Name] = c
	}

	return currentGroups, nil
}

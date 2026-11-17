package chain

import (
	"context"
	"fmt"
	"strings"

	"k8s.io/utils/ptr"

	keycloakApi "github.com/epam/edp-keycloak-operator/api/v1"
	"github.com/epam/edp-keycloak-operator/pkg/client/keycloakapi"
)

// groupMembersPageSize is the page size used to list group members.
const groupMembersPageSize = 100

// findGroupID resolves a group name or slash-separated path to the Keycloak group ID,
// with the same rules as KeycloakRealmUser.spec.groups.
// It returns an error wrapping keycloakapi.ErrNotFound when the group does not exist.
func findGroupID(ctx context.Context, kClient *keycloakapi.KeycloakClient, realmName, group string) (string, error) {
	var (
		grp *keycloakapi.GroupRepresentation
		err error
	)

	if strings.HasPrefix(group, "/") {
		grp, _, err = kClient.Groups.GetGroupByPath(ctx, realmName, group)
	} else {
		grp, _, err = kClient.Groups.FindGroupByName(ctx, realmName, group)
	}

	if err != nil && !keycloakapi.IsNotFound(err) {
		return "", fmt.Errorf("unable to get group %q: %w", group, err)
	}

	if err != nil || grp == nil || grp.Id == nil {
		return "", fmt.Errorf("group %q not found, create it with a KeycloakRealmGroup: %w", group, keycloakapi.ErrNotFound)
	}

	return *grp.Id, nil
}

// findUser looks up the user of a member by exact username or exact email.
// It returns an error wrapping keycloakapi.ErrNotFound when the user does not exist.
func findUser(
	ctx context.Context,
	kClient *keycloakapi.KeycloakClient,
	realmName string,
	member keycloakApi.GroupMember,
) (string, error) {
	var (
		user *keycloakapi.UserRepresentation
		err  error
	)

	if member.Username != "" {
		user, _, err = kClient.Users.FindUserByUsername(ctx, realmName, member.Username)
	} else {
		user, _, err = kClient.Users.FindUserByEmail(ctx, realmName, member.Email)
	}

	if err != nil {
		if keycloakapi.IsNotFound(err) {
			return "", fmt.Errorf("user %s not found: %w", memberLabel(member), keycloakapi.ErrNotFound)
		}

		return "", fmt.Errorf("unable to find user %s: %w", memberLabel(member), err)
	}

	if user == nil || user.Id == nil {
		return "", fmt.Errorf("user %s not found: %w", memberLabel(member), keycloakapi.ErrNotFound)
	}

	return *user.Id, nil
}

// listGroupMembers returns the direct members of a group as a map of user ID to username.
func listGroupMembers(
	ctx context.Context,
	kClient *keycloakapi.KeycloakClient,
	realmName, groupID string,
) (map[string]string, error) {
	members := make(map[string]string)

	for first := int32(0); ; first += groupMembersPageSize {
		page, _, err := kClient.Groups.GetGroupMembers(ctx, realmName, groupID, &keycloakapi.GetGroupMembersParams{
			BriefRepresentation: ptr.To(true),
			First:               ptr.To(first),
			Max:                 ptr.To(int32(groupMembersPageSize)),
		})
		if err != nil {
			return nil, fmt.Errorf("unable to get group members: %w", err)
		}

		for _, u := range page {
			if u.Id == nil {
				continue
			}

			username := *u.Id
			if u.Username != nil {
				username = *u.Username
			}

			members[*u.Id] = username
		}

		if len(page) < groupMembersPageSize {
			return members, nil
		}
	}
}

// memberLabel returns a human-readable identifier for a member.
func memberLabel(member keycloakApi.GroupMember) string {
	if member.Username != "" {
		return fmt.Sprintf("with username %q", member.Username)
	}

	return fmt.Sprintf("with email %q", member.Email)
}

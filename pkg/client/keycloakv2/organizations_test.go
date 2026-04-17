package keycloakv2_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"k8s.io/utils/ptr"

	"github.com/epam/edp-keycloak-operator/pkg/client/keycloakv2"
	"github.com/epam/edp-keycloak-operator/pkg/testutils"
)

// newOrganizationsTestRealm creates a Keycloak client and a fresh realm with Organizations enabled.
// The realm is automatically deleted in t.Cleanup.
func newOrganizationsTestRealm(t *testing.T) (*keycloakv2.KeycloakClient, string) {
	t.Helper()

	keycloakURL := testutils.GetKeycloakURLOrSkip(t)

	c, err := keycloakv2.NewKeycloakClient(
		context.Background(),
		keycloakURL,
		keycloakv2.DefaultAdminClientID,
		keycloakv2.WithPasswordGrant(keycloakv2.DefaultAdminUsername, keycloakv2.DefaultAdminPassword),
	)
	require.NoError(t, err)

	realmName := fmt.Sprintf("test-realm-org-%d", time.Now().UnixNano())
	enabled := true

	t.Cleanup(func() {
		_, _ = c.Realms.DeleteRealm(context.Background(), realmName)
	})

	_, err = c.Realms.CreateRealm(context.Background(), keycloakv2.RealmRepresentation{
		Realm:                &realmName,
		Enabled:              &enabled,
		OrganizationsEnabled: ptr.To(true),
	})
	require.NoError(t, err)

	return c, realmName
}

func TestOrganizationsClient_CRUD(t *testing.T) {
	t.Parallel()

	c, realmName := newOrganizationsTestRealm(t)
	ctx := context.Background()

	alias := fmt.Sprintf("test-org-%d", time.Now().UnixNano())
	orgName := "Test Organization"
	orgDescription := "Test organization description"
	redirectURL := "https://example.com/redirect"

	// 1. Create organization
	resp, err := c.Organizations.CreateOrganization(ctx, realmName, keycloakv2.OrganizationRepresentation{
		Name:        &orgName,
		Alias:       &alias,
		Description: &orgDescription,
		RedirectUrl: &redirectURL,
		Domains: &[]keycloakv2.OrganizationDomainRepresentation{
			{Name: ptr.To("example.com")},
		},
	})
	require.NoError(t, err)
	require.NotNil(t, resp)
	require.NotNil(t, resp.HTTPResponse)

	// 2. List all organizations and verify ours is present
	orgs, resp, err := c.Organizations.GetOrganizations(ctx, realmName, nil)
	require.NoError(t, err)
	require.NotNil(t, resp)

	found := false

	for _, o := range orgs {
		if o.Alias != nil && *o.Alias == alias {
			found = true

			break
		}
	}

	require.True(t, found, "created organization should appear in list")

	// 3. Get organization by alias
	org, resp, err := c.Organizations.GetOrganizationByAlias(ctx, realmName, alias)
	require.NoError(t, err)
	require.NotNil(t, resp)
	require.NotNil(t, org)
	require.NotNil(t, org.Id)
	require.Equal(t, alias, ptr.Deref(org.Alias, ""))
	require.Equal(t, orgName, ptr.Deref(org.Name, ""))

	orgID := ptr.Deref(org.Id, "")
	require.NotEmpty(t, orgID)

	// 4. Update organization
	updatedName := "Updated Organization"
	updatedDescription := "Updated description"

	_, err = c.Organizations.UpdateOrganization(ctx, realmName, orgID, keycloakv2.OrganizationRepresentation{
		Id:          &orgID,
		Name:        &updatedName,
		Alias:       &alias,
		Description: &updatedDescription,
		Domains: &[]keycloakv2.OrganizationDomainRepresentation{
			{Name: ptr.To("example.com")},
		},
	})
	require.NoError(t, err)

	// Verify update
	updatedOrg, _, err := c.Organizations.GetOrganizationByAlias(ctx, realmName, alias)
	require.NoError(t, err)
	require.Equal(t, updatedName, ptr.Deref(updatedOrg.Name, ""))
	require.Equal(t, updatedDescription, ptr.Deref(updatedOrg.Description, ""))

	// 5. Delete organization
	_, err = c.Organizations.DeleteOrganization(ctx, realmName, orgID)
	require.NoError(t, err)

	// 6. Verify deletion
	_, _, err = c.Organizations.GetOrganizationByAlias(ctx, realmName, alias)
	require.Error(t, err)
	require.True(t, keycloakv2.IsNotFound(err), "expected 404 after deletion")
}

func TestOrganizationsClient_GetOrganizationByAlias_NotFound(t *testing.T) {
	t.Parallel()

	c, realmName := newOrganizationsTestRealm(t)
	ctx := context.Background()

	_, _, err := c.Organizations.GetOrganizationByAlias(ctx, realmName, "nonexistent-org-alias-12345")
	require.Error(t, err)
	require.True(t, keycloakv2.IsNotFound(err), "expected 404 for non-existent alias")
}

func TestOrganizationsClient_GetOrganizationByAlias(t *testing.T) {
	t.Parallel()

	c, realmName := newOrganizationsTestRealm(t)
	ctx := context.Background()

	alias := fmt.Sprintf("alias-org-%d", time.Now().UnixNano())
	orgName := "Alias Test Organization"

	// Create organization
	_, err := c.Organizations.CreateOrganization(ctx, realmName, keycloakv2.OrganizationRepresentation{
		Name:  &orgName,
		Alias: &alias,
		Domains: &[]keycloakv2.OrganizationDomainRepresentation{
			{Name: ptr.To("alias-test.com")},
		},
	})
	require.NoError(t, err)

	// Get by alias — verify id, name, alias
	org, _, err := c.Organizations.GetOrganizationByAlias(ctx, realmName, alias)
	require.NoError(t, err)
	require.NotNil(t, org)
	require.Equal(t, alias, ptr.Deref(org.Alias, ""))
	require.Equal(t, orgName, ptr.Deref(org.Name, ""))

	orgID := ptr.Deref(org.Id, "")
	require.NotEmpty(t, orgID)

	// Rename the organization (name changes, alias stays the same)
	updatedName := "Alias Test Organization Renamed"
	_, err = c.Organizations.UpdateOrganization(ctx, realmName, orgID, keycloakv2.OrganizationRepresentation{
		Id:    &orgID,
		Name:  &updatedName,
		Alias: &alias,
		Domains: &[]keycloakv2.OrganizationDomainRepresentation{
			{Name: ptr.To("alias-test.com")},
		},
	})
	require.NoError(t, err)

	// GetOrganizationByAlias still finds the org by the original alias despite name change
	renamedOrg, _, err := c.Organizations.GetOrganizationByAlias(ctx, realmName, alias)
	require.NoError(t, err)
	require.Equal(t, alias, ptr.Deref(renamedOrg.Alias, ""))
	require.Equal(t, updatedName, ptr.Deref(renamedOrg.Name, ""))

	// Delete
	_, err = c.Organizations.DeleteOrganization(ctx, realmName, orgID)
	require.NoError(t, err)

	// Verify 404 after deletion
	_, _, err = c.Organizations.GetOrganizationByAlias(ctx, realmName, alias)
	require.Error(t, err)
	require.True(t, keycloakv2.IsNotFound(err), "expected 404 after deletion")
}

func TestOrganizationsClient_CreateOrganization_Conflict(t *testing.T) {
	t.Parallel()

	c, realmName := newOrganizationsTestRealm(t)
	ctx := context.Background()

	alias := fmt.Sprintf("conflict-org-%d", time.Now().UnixNano())
	orgName := "Conflict Organization"

	org := keycloakv2.OrganizationRepresentation{
		Name:  &orgName,
		Alias: &alias,
		Domains: &[]keycloakv2.OrganizationDomainRepresentation{
			{Name: ptr.To("conflict.com")},
		},
	}

	// First create should succeed
	_, err := c.Organizations.CreateOrganization(ctx, realmName, org)
	require.NoError(t, err)

	// Second create with same alias should conflict
	_, err = c.Organizations.CreateOrganization(ctx, realmName, org)
	require.Error(t, err)
	require.True(t, keycloakv2.IsConflict(err), "expected 409 conflict for duplicate alias")
}

func TestOrganizationsClient_DeleteOrganization_NotFound(t *testing.T) {
	t.Parallel()

	c, realmName := newOrganizationsTestRealm(t)
	ctx := context.Background()

	_, err := c.Organizations.DeleteOrganization(ctx, realmName, "nonexistent-org-id-12345")
	require.Error(t, err)
	require.True(t, keycloakv2.IsNotFound(err), "expected 404 for non-existent org ID")
}

func TestOrganizationsClient_IdentityProviders(t *testing.T) {
	t.Parallel()

	c, realmName := newOrganizationsTestRealm(t)
	ctx := context.Background()

	// Create an identity provider in the realm
	idpAlias := fmt.Sprintf("test-idp-%d", time.Now().UnixNano())
	displayName := "Test IdP"
	providerID := testGithubProviderID
	enabled := true

	_, err := c.IdentityProviders.CreateIdentityProvider(ctx, realmName, keycloakv2.IdentityProviderRepresentation{
		Alias:       &idpAlias,
		DisplayName: &displayName,
		ProviderId:  &providerID,
		Enabled:     &enabled,
		Config: &map[string]string{
			"clientId":     "test-client-id",
			"clientSecret": "test-client-secret",
		},
	})
	require.NoError(t, err)

	// Create organization
	alias := fmt.Sprintf("idp-org-%d", time.Now().UnixNano())
	orgName := "IdP Organization"

	_, err = c.Organizations.CreateOrganization(ctx, realmName, keycloakv2.OrganizationRepresentation{
		Name:  &orgName,
		Alias: &alias,
		Domains: &[]keycloakv2.OrganizationDomainRepresentation{
			{Name: ptr.To("idp-org.com")},
		},
	})
	require.NoError(t, err)

	org, _, err := c.Organizations.GetOrganizationByAlias(ctx, realmName, alias)
	require.NoError(t, err)

	orgID := ptr.Deref(org.Id, "")
	require.NotEmpty(t, orgID)

	// Initially no identity providers linked
	idps, _, err := c.Organizations.GetOrganizationIdentityProviders(ctx, realmName, orgID)
	require.NoError(t, err)
	require.Empty(t, idps)

	// Link identity provider
	_, err = c.Organizations.LinkIdentityProviderToOrganization(ctx, realmName, orgID, idpAlias)
	require.NoError(t, err)

	// Verify link
	idps, _, err = c.Organizations.GetOrganizationIdentityProviders(ctx, realmName, orgID)
	require.NoError(t, err)
	require.Len(t, idps, 1)
	require.Equal(t, idpAlias, ptr.Deref(idps[0].Alias, ""))

	// Unlink identity provider
	_, err = c.Organizations.UnlinkIdentityProviderFromOrganization(ctx, realmName, orgID, idpAlias)
	require.NoError(t, err)

	// Verify unlink
	idps, _, err = c.Organizations.GetOrganizationIdentityProviders(ctx, realmName, orgID)
	require.NoError(t, err)
	require.Empty(t, idps)
}

func TestOrganizationsClient_Members(t *testing.T) {
	t.Parallel()

	c, realmName := newOrganizationsTestRealm(t)
	ctx := context.Background()

	// Create a user in the realm.
	username := fmt.Sprintf("org-member-%d", time.Now().UnixNano())

	userResp, err := c.Users.CreateUser(ctx, realmName, keycloakv2.UserRepresentation{
		Username: &username,
		Enabled:  ptr.To(true),
		Email:    ptr.To(fmt.Sprintf("%s@member-test.com", username)),
	})
	require.NoError(t, err)

	userID := keycloakv2.GetResourceIDFromResponse(userResp)
	require.NotEmpty(t, userID)

	// Create an organization.
	alias := fmt.Sprintf("member-org-%d", time.Now().UnixNano())

	_, err = c.Organizations.CreateOrganization(ctx, realmName, keycloakv2.OrganizationRepresentation{
		Name:  ptr.To("Member Test Org"),
		Alias: &alias,
		Domains: &[]keycloakv2.OrganizationDomainRepresentation{
			{Name: ptr.To("member-test.com")},
		},
	})
	require.NoError(t, err)

	org, _, err := c.Organizations.GetOrganizationByAlias(ctx, realmName, alias)
	require.NoError(t, err)

	orgID := ptr.Deref(org.Id, "")
	require.NotEmpty(t, orgID)

	// Initially no members.
	members, resp, err := c.Organizations.GetOrganizationMembers(ctx, realmName, orgID, nil)
	require.NoError(t, err)
	require.NotNil(t, resp)
	require.Empty(t, members)

	// Add member.
	resp, err = c.Organizations.AddOrganizationMember(ctx, realmName, orgID, userID)
	require.NoError(t, err)
	require.NotNil(t, resp)

	// List members — should find the user.
	members, _, err = c.Organizations.GetOrganizationMembers(ctx, realmName, orgID, nil)
	require.NoError(t, err)
	require.Len(t, members, 1)

	// Remove member.
	resp, err = c.Organizations.RemoveOrganizationMember(ctx, realmName, orgID, userID)
	require.NoError(t, err)
	require.NotNil(t, resp)

	// Verify removal.
	members, _, err = c.Organizations.GetOrganizationMembers(ctx, realmName, orgID, nil)
	require.NoError(t, err)
	require.Empty(t, members)
}

func TestOrganizationsClient_InviteExistingMember(t *testing.T) {
	t.Parallel()

	c, realmName := newOrganizationsTestRealm(t)
	ctx := context.Background()

	// Create a user.
	username := fmt.Sprintf("invite-existing-%d", time.Now().UnixNano())

	userResp, err := c.Users.CreateUser(ctx, realmName, keycloakv2.UserRepresentation{
		Username: &username,
		Enabled:  ptr.To(true),
		Email:    ptr.To(fmt.Sprintf("%s@invite-existing.com", username)),
	})
	require.NoError(t, err)

	userID := keycloakv2.GetResourceIDFromResponse(userResp)
	require.NotEmpty(t, userID)

	// Create an organization.
	alias := fmt.Sprintf("invite-org-%d", time.Now().UnixNano())

	_, err = c.Organizations.CreateOrganization(ctx, realmName, keycloakv2.OrganizationRepresentation{
		Name:  ptr.To("Invite Test Org"),
		Alias: &alias,
		Domains: &[]keycloakv2.OrganizationDomainRepresentation{
			{Name: ptr.To("invite-existing.com")},
		},
	})
	require.NoError(t, err)

	org, _, err := c.Organizations.GetOrganizationByAlias(ctx, realmName, alias)
	require.NoError(t, err)

	orgID := ptr.Deref(org.Id, "")

	// Invite existing user — requires SMTP. Without SMTP config, Keycloak returns a server error.
	_, err = c.Organizations.InviteExistingOrganizationMember(ctx, realmName, orgID, userID)
	if err != nil {
		require.True(t, keycloakv2.IsServerError(err), "expected server error when SMTP is not configured, got: %v", err)
	}
}

func TestOrganizationsClient_InviteNewMember(t *testing.T) {
	t.Parallel()

	c, realmName := newOrganizationsTestRealm(t)
	ctx := context.Background()

	// Create an organization.
	alias := fmt.Sprintf("invite-new-org-%d", time.Now().UnixNano())

	_, err := c.Organizations.CreateOrganization(ctx, realmName, keycloakv2.OrganizationRepresentation{
		Name:  ptr.To("Invite New Org"),
		Alias: &alias,
		Domains: &[]keycloakv2.OrganizationDomainRepresentation{
			{Name: ptr.To("invite-new.com")},
		},
	})
	require.NoError(t, err)

	org, _, err := c.Organizations.GetOrganizationByAlias(ctx, realmName, alias)
	require.NoError(t, err)

	orgID := ptr.Deref(org.Id, "")

	// Invite new member — requires SMTP. Without SMTP config, Keycloak returns a server error.
	_, err = c.Organizations.InviteNewOrganizationMember(
		ctx, realmName, orgID, "new-member@invite-new.com", "New", "Member",
	)
	if err != nil {
		require.True(t, keycloakv2.IsServerError(err), "expected server error when SMTP is not configured, got: %v", err)
	}
}

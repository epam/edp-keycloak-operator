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

func TestRealmComponentsClient_CRUD(t *testing.T) {
	keycloakURL := testutils.GetKeycloakURLOrSkip(t)
	t.Parallel()

	c, err := keycloakv2.NewKeycloakClient(
		context.Background(),
		keycloakURL,
		keycloakv2.DefaultAdminClientID,
		keycloakv2.WithPasswordGrant(keycloakv2.DefaultAdminUsername, keycloakv2.DefaultAdminPassword),
	)
	require.NoError(t, err)

	ctx := context.Background()

	realmName := fmt.Sprintf("test-realm-rc-crud-%d", time.Now().UnixNano())
	enabled := true

	t.Cleanup(func() {
		_, _ = c.Realms.DeleteRealm(context.Background(), realmName)
	})

	_, err = c.Realms.CreateRealm(ctx, keycloakv2.RealmRepresentation{
		Realm:   &realmName,
		Enabled: &enabled,
	})
	require.NoError(t, err)

	componentName := "test-client-reg-policy"
	providerID := "scope"
	providerType := "org.keycloak.services.clientregistration.policy.ClientRegistrationPolicy"
	config := keycloakv2.MultivaluedHashMapStringString{
		"priority": {"0"},
	}

	// 1. Create component
	resp, err := c.RealmComponents.CreateComponent(ctx, realmName, keycloakv2.ComponentRepresentation{
		Name:         &componentName,
		ProviderId:   &providerID,
		ProviderType: &providerType,
		Config:       &config,
	})
	require.NoError(t, err)
	require.NotNil(t, resp)

	componentID := keycloakv2.GetResourceIDFromResponse(resp)
	require.NotEmpty(t, componentID, "component ID should be extracted from Location header")

	// 2. GetComponent by ID — assert fields
	component, _, err := c.RealmComponents.GetComponent(ctx, realmName, componentID)
	require.NoError(t, err)
	require.NotNil(t, component)
	require.Equal(t, componentName, *component.Name)
	require.Equal(t, providerID, *component.ProviderId)
	require.Equal(t, providerType, *component.ProviderType)

	// 3. FindComponentByName — assert same component
	found, err := c.RealmComponents.FindComponentByName(ctx, realmName, componentName)
	require.NoError(t, err)
	require.NotNil(t, found)
	require.Equal(t, componentID, *found.Id)
	require.Equal(t, componentName, *found.Name)

	// 4. FindComponentByName for non-existent name — returns nil
	notFound, err := c.RealmComponents.FindComponentByName(ctx, realmName, "does-not-exist")
	require.NoError(t, err)
	require.Nil(t, notFound)

	// 5. UpdateComponent — change priority
	updatedConfig := keycloakv2.MultivaluedHashMapStringString{
		"priority": {"10"},
	}
	_, err = c.RealmComponents.UpdateComponent(ctx, realmName, componentID, keycloakv2.ComponentRepresentation{
		Id:           ptr.To(componentID),
		Name:         &componentName,
		ProviderId:   &providerID,
		ProviderType: &providerType,
		Config:       &updatedConfig,
	})
	require.NoError(t, err)

	updated, _, err := c.RealmComponents.GetComponent(ctx, realmName, componentID)
	require.NoError(t, err)
	require.NotNil(t, updated)
	require.Equal(t, componentName, *updated.Name)

	// 6. GetComponents — list with name filter
	components, _, err := c.RealmComponents.GetComponents(ctx, realmName, &keycloakv2.GetComponentsParams{
		Name: &componentName,
	})
	require.NoError(t, err)
	require.NotEmpty(t, components)

	foundInList := false

	for _, comp := range components {
		if comp.Name != nil && *comp.Name == componentName {
			foundInList = true
			break
		}
	}

	require.True(t, foundInList, "component should be in the filtered list")

	// 7. DeleteComponent
	_, err = c.RealmComponents.DeleteComponent(ctx, realmName, componentID)
	require.NoError(t, err)

	// 8. GetComponent after deletion — assert IsNotFound
	_, _, err = c.RealmComponents.GetComponent(ctx, realmName, componentID)
	require.Error(t, err)
	require.True(t, keycloakv2.IsNotFound(err), "expected 404 after deletion")
}

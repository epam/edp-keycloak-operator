package adapter

import (
	"context"
	"testing"

	"github.com/jarcoal/httpmock"
	"github.com/stretchr/testify/require"
)

func TestGoCloakAdapter_GetIdentityProvider(t *testing.T) {
	kc, _, _ := initAdapter(t)

	httpmock.RegisterResponder("GET", "/admin/realms/realm1/identity-provider/instances/alias1",
		httpmock.NewStringResponder(200, ""))

	_, err := kc.GetIdentityProvider(context.Background(), "realm1", "alias1")
	require.NoError(t, err)

	httpmock.RegisterResponder("GET", "/admin/realms/realm1/identity-provider/instances/alias2",
		httpmock.NewStringResponder(404, ""))

	_, err = kc.GetIdentityProvider(context.Background(), "realm1", "alias2")
	if !IsErrNotFound(err) {
		require.NoError(t, err)
	}

	httpmock.RegisterResponder("GET", "/admin/realms/realm1/identity-provider/instances/alias3",
		httpmock.NewStringResponder(500, "fatal"))

	_, err = kc.GetIdentityProvider(context.Background(), "realm1", "alias3")
	require.Error(t, err)

	if err.Error() != "unable to get idp: status: 500, body: fatal" {
		t.Fatalf("wrong error returned: %s", err.Error())
	}
}

func TestGoCloakAdapter_CreateIdentityProvider(t *testing.T) {
	kc, _, _ := initAdapter(t)

	httpmock.RegisterResponder("POST", "/admin/realms/realm1/identity-provider/instances",
		httpmock.NewStringResponder(200, ""))

	err := kc.CreateIdentityProvider(context.Background(), "realm1", &IdentityProvider{})
	require.NoError(t, err)

	httpmock.RegisterResponder("POST", "/admin/realms/realm2/identity-provider/instances",
		httpmock.NewStringResponder(500, "fatal"))

	err = kc.CreateIdentityProvider(context.Background(), "realm2", &IdentityProvider{})
	require.Error(t, err)

	if err.Error() != "unable to create idp: status: 500, body: fatal" {
		t.Fatalf("wrong error returned: %s", err.Error())
	}
}

func TestGoCloakAdapter_UpdateIdentityProvider(t *testing.T) {
	kc, _, _ := initAdapter(t)

	httpmock.RegisterResponder("PUT", "/admin/realms/realm1/identity-provider/instances/alias1",
		httpmock.NewStringResponder(200, ""))

	err := kc.UpdateIdentityProvider(context.Background(), "realm1", &IdentityProvider{Alias: "alias1"})
	require.NoError(t, err)

	httpmock.RegisterResponder("PUT", "/admin/realms/realm1/identity-provider/instances/alias2",
		httpmock.NewStringResponder(500, "fatal"))

	err = kc.UpdateIdentityProvider(context.Background(), "realm1", &IdentityProvider{Alias: "alias2"})
	require.Error(t, err)

	if err.Error() != "unable to update idp: status: 500, body: fatal" {
		t.Fatalf("wrong error returned: %s", err.Error())
	}
}

func TestGoCloakAdapter_DeleteIdentityProvider(t *testing.T) {
	kc, _, _ := initAdapter(t)

	httpmock.RegisterResponder("DELETE", "/admin/realms/realm1/identity-provider/instances/alias1",
		httpmock.NewStringResponder(200, ""))

	err := kc.DeleteIdentityProvider(context.Background(), "realm1", "alias1")
	require.NoError(t, err)

	httpmock.RegisterResponder("DELETE", "/admin/realms/realm1/identity-provider/instances/alias2",
		httpmock.NewStringResponder(500, "fatal"))

	err = kc.DeleteIdentityProvider(context.Background(), "realm1", "alias2")
	require.Error(t, err)

	if err.Error() != "unable to delete idp: status: 500, body: fatal" {
		t.Fatalf("wrong error returned: %s", err.Error())
	}
}

func TestGoCloakAdapter_CreateIDPMapper(t *testing.T) {
	kc, _, _ := initAdapter(t)

	rsp := httpmock.NewStringResponse(200, "")
	defer closeWithFailOnError(t, rsp.Body)
	rsp.Header.Set("Location", "id/new-id")

	httpmock.RegisterResponder("POST",
		"/admin/realms/realm1/identity-provider/instances/alias1/mappers",
		httpmock.ResponderFromResponse(rsp))

	_, err := kc.CreateIDPMapper(context.Background(), "realm1", "alias1", &IdentityProviderMapper{})
	require.NoError(t, err)

	httpmock.RegisterResponder("POST",
		"/admin/realms/realm1/identity-provider/instances/alias2/mappers",
		httpmock.NewStringResponder(500, "fatal"))

	_, err = kc.CreateIDPMapper(context.Background(), "realm1", "alias2",
		&IdentityProviderMapper{})

	require.Error(t, err)

	if err.Error() != "unable to create idp mapper: status: 500, body: fatal" {
		t.Fatalf("wrong error returned: %s", err.Error())
	}
}

func TestGoCloakAdapter_UpdateIDPMapper(t *testing.T) {
	kc, _, _ := initAdapter(t)

	httpmock.RegisterResponder("PUT",
		"/admin/realms/realm1/identity-provider/instances/alias1/mappers/id11",
		httpmock.NewStringResponder(200, ""))

	err := kc.UpdateIDPMapper(context.Background(), "realm1", "alias1",
		&IdentityProviderMapper{ID: "id11"})
	require.NoError(t, err)

	httpmock.RegisterResponder("PUT",
		"/admin/realms/realm1/identity-provider/instances/alias2/mappers/id11",
		httpmock.NewStringResponder(500, "fatal"))

	err = kc.UpdateIDPMapper(context.Background(), "realm1", "alias2",
		&IdentityProviderMapper{ID: "id11"})
	require.Error(t, err)

	if err.Error() != "unable to update idp mapper: status: 500, body: fatal" {
		t.Fatalf("wrong error returned: %s", err.Error())
	}
}

func TestGoCloakAdapter_DeleteIDPMapper(t *testing.T) {
	kc, _, _ := initAdapter(t)

	httpmock.RegisterResponder("DELETE",
		"/admin/realms/realm1/identity-provider/instances/alias1/mappers/mapper1",
		httpmock.NewStringResponder(200, ""))

	err := kc.DeleteIDPMapper(context.Background(), "realm1", "alias1", "mapper1")
	require.NoError(t, err)

	httpmock.RegisterResponder("DELETE",
		"/admin/realms/realm1/identity-provider/instances/alias1/mappers/mapper2",
		httpmock.NewStringResponder(500, "fatal"))

	err = kc.DeleteIDPMapper(context.Background(), "realm1", "alias1", "mapper2")
	require.Error(t, err)

	if err.Error() != "unable to delete idp mapper: status: 500, body: fatal" {
		t.Fatalf("wrong error returned: %s", err.Error())
	}
}

func TestGoCloakAdapter_GetIDPMappers(t *testing.T) {
	kc, _, _ := initAdapter(t)

	httpmock.RegisterResponder("GET",
		"/admin/realms/realm1/identity-provider/instances/alias1/mappers",
		httpmock.NewStringResponder(200, ""))

	_, err := kc.GetIDPMappers(context.Background(), "realm1", "alias1")
	require.NoError(t, err)

	httpmock.RegisterResponder("GET",
		"/admin/realms/realm1/identity-provider/instances/alias2/mappers",
		httpmock.NewStringResponder(500, "fatal"))

	_, err = kc.GetIDPMappers(context.Background(), "realm1", "alias2")
	require.Error(t, err)

	if err.Error() != "unable to get idp mappers: status: 500, body: fatal" {
		t.Fatalf("wrong error returned: %s", err.Error())
	}
}

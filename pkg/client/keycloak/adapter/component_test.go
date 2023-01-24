package adapter

import (
	"context"
	"testing"

	"github.com/Nerzal/gocloak/v12"
	"github.com/go-resty/resty/v2"
	"github.com/jarcoal/httpmock"
	"github.com/stretchr/testify/require"

	"github.com/epam/edp-keycloak-operator/pkg/client/keycloak/mock"
)

func initAdapter() (*GoCloakAdapter, *MockGoCloakClient, *resty.Client) {
	mockClient := new(MockGoCloakClient)
	restyClient := resty.New()
	httpmock.ActivateNonDefault(restyClient.GetClient())
	mockClient.On("RestyClient").Return(restyClient)

	logger := mock.NewLogr()

	return &GoCloakAdapter{
		client:   mockClient,
		basePath: "",
		token:    &gocloak.JWT{AccessToken: "token"},
		log:      logger,
	}, mockClient, restyClient
}

func testComponent() *Component {
	return &Component{
		Name:         "test-name",
		ProviderType: "test-provider-type",
		Config: map[string][]string{
			"foo": {"bar", "vaz"},
		},
	}
}

func TestGoCloakAdapter_CreateComponent(t *testing.T) {
	kcAdapter, _, _ := initAdapter()

	httpmock.RegisterResponder("POST", "/admin/realms/realm-name/components",
		httpmock.NewStringResponder(200, ""))

	err := kcAdapter.CreateComponent(context.Background(), "realm-name", testComponent())
	require.NoError(t, err)

	httpmock.RegisterResponder("POST", "/admin/realms/realm-name-error/components",
		httpmock.NewStringResponder(500, "fatal"))

	err = kcAdapter.CreateComponent(context.Background(), "realm-name-error",
		testComponent())
	require.Error(t, err)

	if err.Error() != "error during request: status: 500, body: fatal" {
		t.Fatalf("wrong error returned: %s", err.Error())
	}
}

func TestMock_UpdateComponent(t *testing.T) {
	kcAdapter, _, _ := initAdapter()
	testCmp := testComponent()
	testCmp.ID = "test-id"

	httpmock.RegisterResponder("GET", "/admin/realms/realm-name/components",
		httpmock.NewJsonResponderOrPanic(200, []Component{*testCmp}))
	httpmock.RegisterResponder("PUT", "/admin/realms/realm-name/components/test-id",
		httpmock.NewStringResponder(200, ""))

	err := kcAdapter.UpdateComponent(context.Background(), "realm-name", testComponent())
	require.NoError(t, err)

	httpmock.RegisterResponder("GET", "/admin/realms/realm-name-no-components/components",
		httpmock.NewJsonResponderOrPanic(200, []Component{}))

	err = kcAdapter.UpdateComponent(context.Background(), "realm-name-no-components", testComponent())
	require.Error(t, err)

	if err.Error() != "unable to get component id: component not found" {
		t.Fatalf("wrong error returned: %s", err.Error())
	}

	httpmock.RegisterResponder("GET", "/admin/realms/realm-name-update-failure/components",
		httpmock.NewJsonResponderOrPanic(200, []Component{*testCmp}))
	httpmock.RegisterResponder("PUT", "/admin/realms/realm-name-update-failure/components/test-id",
		httpmock.NewStringResponder(404, "not found"))

	err = kcAdapter.UpdateComponent(context.Background(), "realm-name-update-failure", testComponent())
	require.Error(t, err)

	if err.Error() != "error during update component request: status: 404, body: not found" {
		t.Fatalf("wrong error returned: %s", err.Error())
	}
}

func TestGoCloakAdapter_DeleteComponent(t *testing.T) {
	kcAdapter, _, _ := initAdapter()
	testCmp := testComponent()
	testCmp.ID = "test-id"

	httpmock.RegisterResponder("GET", "/admin/realms/realm-name/components",
		httpmock.NewJsonResponderOrPanic(200, []Component{*testCmp}))
	httpmock.RegisterResponder("DELETE", "/admin/realms/realm-name/components/test-id",
		httpmock.NewStringResponder(200, ""))

	err := kcAdapter.DeleteComponent(context.Background(), "realm-name", testCmp.Name)
	require.NoError(t, err)

	httpmock.RegisterResponder("GET", "/admin/realms/realm-name-no-components/components",
		httpmock.NewJsonResponderOrPanic(200, []Component{}))

	err = kcAdapter.DeleteComponent(context.Background(), "realm-name-no-components", testCmp.Name)
	require.Error(t, err)

	if err.Error() != "unable to get component id: component not found" {
		t.Fatalf("wrong error returned: %s", err.Error())
	}

	httpmock.RegisterResponder("GET", "/admin/realms/realm-name-delete-failure/components",
		httpmock.NewJsonResponderOrPanic(200, []Component{*testCmp}))
	httpmock.RegisterResponder("DELETE", "/admin/realms/realm-name-delete-failure/components/test-id",
		httpmock.NewStringResponder(404, "delete not found"))

	err = kcAdapter.DeleteComponent(context.Background(), "realm-name-delete-failure", testCmp.Name)
	require.Error(t, err)

	if err.Error() != "error during delete component request: status: 404, body: delete not found" {
		t.Fatalf("wrong error returned: %s", err.Error())
	}
}

func TestGoCloakAdapter_GetComponent_Failure(t *testing.T) {
	kcAdapter, _, _ := initAdapter()

	httpmock.RegisterResponder("GET", "/admin/realms/realm-name/components",
		httpmock.NewStringResponder(422, "forbidden"))

	_, err := kcAdapter.GetComponent(context.Background(), "realm-name", "test-name")
	require.Error(t, err)

	if err.Error() != "error during get component request: status: 422, body: forbidden" {
		t.Fatalf("wrong error returned: %s", err.Error())
	}
}

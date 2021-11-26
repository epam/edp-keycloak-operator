package adapter

import (
	"strings"
	"testing"

	"github.com/Nerzal/gocloak/v10"
	"github.com/go-resty/resty/v2"
	"github.com/jarcoal/httpmock"
)

func TestGoCloakAdapter_SetRealmEventConfig(t *testing.T) {
	mockClient := new(MockGoCloakClient)
	restyClient := resty.New()
	httpmock.ActivateNonDefault(restyClient.GetClient())
	mockClient.On("RestyClient").Return(restyClient)

	adapter := GoCloakAdapter{
		client:   mockClient,
		basePath: "",
		token:    &gocloak.JWT{AccessToken: "token"},
	}

	err := adapter.SetRealmEventConfig("realm1", &RealmEventConfig{})
	if err == nil {
		t.Fatal("no error returned")
	}

	if !strings.Contains(err.Error(), "error during set realm event config request") {
		t.Fatalf("wrong error returned: %s", err.Error())
	}

	httpmock.RegisterResponder("PUT", "/auth/admin/realms/r1/events/config",
		httpmock.NewStringResponder(200, ""))

	if err := adapter.SetRealmEventConfig("r1",
		&RealmEventConfig{EventsListeners: []string{"foo", "bar"}}); err != nil {
		t.Fatal(err)
	}
}
